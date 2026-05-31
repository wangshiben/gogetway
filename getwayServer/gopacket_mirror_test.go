package getwayServer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/wangshiben/gogetway/proto"
	"github.com/wangshiben/gogetway/test_server"
)

var simpleGetServerOnce sync.Once

func ensureSimpleGetServer(t *testing.T) {
	t.Helper()
	simpleGetServerOnce.Do(func() {
		go test_server.SimpleGetServer()
	})
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:8090", 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("simple get server did not start on 127.0.0.1:8090")
}

func exerciseObservedServer(t *testing.T, rawRequest []byte) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8090", time.Second)
	if err != nil {
		t.Fatalf("dial observed server: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write(rawRequest); err != nil {
		t.Fatalf("write observed request: %v", err)
	}
	resp, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read observed response: %v", err)
	}
	if !bytes.Contains(resp, []byte("success")) {
		t.Fatalf("unexpected observed response: %q", string(resp))
	}
}

func startMirrorTargetServer(t *testing.T) (string, <-chan *http.Request) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen target server: %v", err)
	}
	requests := make(chan *http.Request, 1)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r.Clone(context.Background())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
	}
	go server.Serve(ln)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		_ = ln.Close()
	})
	return ln.Addr().String(), requests
}

func buildObservedPacket(t *testing.T, srcIP string, srcPort uint16, dstIP string, dstPort uint16, payload []byte, seq uint32) gopacket.Packet {
	t.Helper()
	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    net.ParseIP(srcIP).To4(),
		DstIP:    net.ParseIP(dstIP).To4(),
	}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     seq,
		ACK:     true,
		PSH:     true,
		Window:  14600,
	}
	tcp.SetNetworkLayerForChecksum(ip)
	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}, ip, tcp, gopacket.Payload(payload)); err != nil {
		t.Fatalf("serialize packet: %v", err)
	}
	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeIPv4, gopacket.Default)
}

func parseSingleRecordedPacket(t *testing.T, data []byte) *proto.Packet {
	t.Helper()
	scanner, err := proto.ReadProtoFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("read proto from writer: %v", err)
	}
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			t.Fatalf("scan proto packet: %v", err)
		}
		t.Fatal("expected one recorded proto packet")
	}
	packet, err := proto.UnMarshal(scanner.Bytes())
	if err != nil {
		t.Fatalf("unmarshal recorded packet: %v", err)
	}
	return packet
}

func TestGopacketTrafficMirror_ForwardsObservedTrafficToTarget(t *testing.T) {
	ensureSimpleGetServer(t)
	rawRequest := []byte("GET / HTTP/1.1\r\nHost: 127.0.0.1:8090\r\nConnection: close\r\n\r\n")
	exerciseObservedServer(t, rawRequest)

	targetAddr, requests := startMirrorTargetServer(t)
	forwardHook := TrafficMirrorHookFunc(func(_ context.Context, chunk *ObservedTrafficChunk) ([]byte, bool, error) {
		if chunk.Direction != TrafficDirectionToObserved {
			return nil, false, nil
		}
		modified := bytes.Replace(chunk.Data, []byte("\r\n\r\n"), []byte("\r\nX-Mirror-Test: forward\r\n\r\n"), 1)
		return modified, true, nil
	})
	mirror := NewGopacketTrafficMirror(GopacketMirrorConfig{
		ObservedPort: 8090,
		TargetAddr:   targetAddr,
		ForwardHook:  forwardHook,
	})

	packets := make(chan gopacket.Packet, 1)
	packets <- buildObservedPacket(t, "127.0.0.1", 42001, "127.0.0.1", 8090, rawRequest, 1000)
	close(packets)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := mirror.Start(ctx, NewChannelObservedPacketSource(packets)); err != nil {
		t.Fatalf("start traffic mirror: %v", err)
	}

	select {
	case req := <-requests:
		if req.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if req.Header.Get("X-Mirror-Test") != "forward" {
			t.Fatalf("missing forwarded header: %#v", req.Header)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive forwarded request on target server")
	}
}

func TestGopacketTrafficMirror_RecordsObservedTrafficWithWriteFunc(t *testing.T) {
	ensureSimpleGetServer(t)
	rawRequest := []byte("GET / HTTP/1.1\r\nHost: 127.0.0.1:8090\r\nConnection: close\r\n\r\n")
	exerciseObservedServer(t, rawRequest)

	var mu sync.Mutex
	var recorded bytes.Buffer
	recordHook := TrafficMirrorHookFunc(func(_ context.Context, chunk *ObservedTrafficChunk) ([]byte, bool, error) {
		if chunk.Direction != TrafficDirectionToObserved {
			return nil, false, nil
		}
		modified := bytes.Replace(chunk.Data, []byte("\r\n\r\n"), []byte("\r\nX-Record-Test: enabled\r\n\r\n"), 1)
		return modified, true, nil
	})
	writeFunc := func(data []byte, _ context.Context) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return recorded.Write(data)
	}
	mirror := NewGopacketTrafficMirror(GopacketMirrorConfig{
		ObservedPort: 8090,
		WriteFunc:    writeFunc,
		RecordHook:   recordHook,
	})

	packets := make(chan gopacket.Packet, 2)
	packets <- buildObservedPacket(t, "127.0.0.1", 42002, "127.0.0.1", 8090, rawRequest, 2000)
	packets <- buildObservedPacket(t, "127.0.0.1", 8090, "127.0.0.1", 42002, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"), 3000)
	close(packets)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := mirror.Start(ctx, NewChannelObservedPacketSource(packets)); err != nil {
		t.Fatalf("start traffic mirror: %v", err)
	}

	mu.Lock()
	data := append([]byte(nil), recorded.Bytes()...)
	mu.Unlock()
	if len(data) == 0 {
		t.Fatal("expected recorded proto data")
	}
	packet := parseSingleRecordedPacket(t, data)
	if packet.Type != TCPType {
		t.Fatalf("unexpected packet type: %d", packet.Type)
	}
	if !strings.Contains(packet.From, "127.0.0.1:42002") {
		t.Fatalf("unexpected packet from: %s", packet.From)
	}
	if !strings.Contains(packet.To, "127.0.0.1:8090") {
		t.Fatalf("unexpected packet to: %s", packet.To)
	}
	if !bytes.Contains(packet.Data, []byte("X-Record-Test: enabled")) {
		t.Fatalf("record hook did not modify payload: %q", string(packet.Data))
	}
	if bytes.Contains(packet.Data, []byte("HTTP/1.1 200 OK")) {
		t.Fatalf("response traffic should not be recorded by default hook: %q", string(packet.Data))
	}
}

func TestDetectTrafficDirection(t *testing.T) {
	direction, ignore := detectTrafficDirection("42003", "8090", 8090)
	if direction != TrafficDirectionToObserved || ignore {
		t.Fatalf("unexpected request direction: %s ignore=%v", direction, ignore)
	}
	direction, ignore = detectTrafficDirection("8090", "42003", 8090)
	if direction != TrafficDirectionFromObserved || ignore {
		t.Fatalf("unexpected response direction: %s ignore=%v", direction, ignore)
	}
	direction, ignore = detectTrafficDirection("42003", "8080", 8090)
	if direction != TrafficDirectionUnknown || !ignore {
		t.Fatalf("unexpected unrelated direction: %s ignore=%v", direction, ignore)
	}
}

func TestObservedPacketSourceNil(t *testing.T) {
	mirror := NewGopacketTrafficMirror(GopacketMirrorConfig{ObservedPort: 8090})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := mirror.Start(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "packet source is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func ExampleNewGopacketTrafficMirror() {
	mirror := NewGopacketTrafficMirror(GopacketMirrorConfig{
		ObservedPort: 8090,
		TargetAddr:   "127.0.0.1:18090",
		ForwardHook: TrafficMirrorHookFunc(func(_ context.Context, chunk *ObservedTrafficChunk) ([]byte, bool, error) {
			if chunk.Direction != TrafficDirectionToObserved {
				return nil, false, nil
			}
			return chunk.Data, true, nil
		}),
		RecordHook: DefaultRecordTrafficMirrorHook{},
	})
	fmt.Println(mirror.config.ObservedPort)
	fmt.Println(mirror.config.TargetAddr)
	fmt.Println(mirror.config.ForwardHook != nil)
	fmt.Println(mirror.config.RecordHook != nil)
	// Output:
	// 8090
	// 127.0.0.1:18090
	// true
	// true
}
