package getwayServer

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/tcpassembly"
	"github.com/wangshiben/gogetway/Types"
	"github.com/wangshiben/gogetway/proto"
)

type TrafficDirection string

const (
	TrafficDirectionUnknown      TrafficDirection = "unknown"
	TrafficDirectionToObserved   TrafficDirection = "to_observed"
	TrafficDirectionFromObserved TrafficDirection = "from_observed"
)

type ObservedTrafficChunk struct {
	Data      []byte
	From      string
	To        string
	Direction TrafficDirection
	Type      Types.ClientType
	SeenAt    time.Time
}

type TrafficMirrorHook interface {
	Handle(ctx context.Context, chunk *ObservedTrafficChunk) (data []byte, keep bool, err error)
}

type TrafficMirrorHookFunc func(ctx context.Context, chunk *ObservedTrafficChunk) (data []byte, keep bool, err error)

func (f TrafficMirrorHookFunc) Handle(ctx context.Context, chunk *ObservedTrafficChunk) ([]byte, bool, error) {
	return f(ctx, chunk)
}

type DefaultTrafficMirrorHook struct{}

type DefaultForwardTrafficMirrorHook struct{}

type DefaultRecordTrafficMirrorHook struct{}

func (DefaultTrafficMirrorHook) Handle(_ context.Context, chunk *ObservedTrafficChunk) ([]byte, bool, error) {
	data := make([]byte, len(chunk.Data))
	copy(data, chunk.Data)
	return data, true, nil
}

func (DefaultForwardTrafficMirrorHook) Handle(_ context.Context, chunk *ObservedTrafficChunk) ([]byte, bool, error) {
	if chunk.Direction != TrafficDirectionToObserved {
		return nil, false, nil
	}
	data := make([]byte, len(chunk.Data))
	copy(data, chunk.Data)
	return data, true, nil
}

func (DefaultRecordTrafficMirrorHook) Handle(_ context.Context, chunk *ObservedTrafficChunk) ([]byte, bool, error) {
	if chunk.Direction != TrafficDirectionToObserved {
		return nil, false, nil
	}
	data := make([]byte, len(chunk.Data))
	copy(data, chunk.Data)
	return data, true, nil
}

type ObservedPacketSource interface {
	Packets() <-chan gopacket.Packet
	Close() error
}

type ChannelObservedPacketSource struct {
	packets <-chan gopacket.Packet
}

func NewChannelObservedPacketSource(packets <-chan gopacket.Packet) *ChannelObservedPacketSource {
	return &ChannelObservedPacketSource{packets: packets}
}

func (s *ChannelObservedPacketSource) Packets() <-chan gopacket.Packet {
	return s.packets
}

func (s *ChannelObservedPacketSource) Close() error {
	return nil
}

type GopacketMirrorConfig struct {
	ObservedPort uint16
	TargetAddr   string
	Writer       io.Writer
	WriteFunc    WriteFunc

	ForwardHook TrafficMirrorHook
	RecordHook  TrafficMirrorHook
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

type GopacketTrafficMirror struct {
	config GopacketMirrorConfig
}

func NewGopacketTrafficMirror(config GopacketMirrorConfig) *GopacketTrafficMirror {
	if config.ForwardHook == nil {
		config.ForwardHook = DefaultForwardTrafficMirrorHook{}
	}
	if config.RecordHook == nil {
		config.RecordHook = DefaultRecordTrafficMirrorHook{}
	}
	if config.DialContext == nil {
		dialer := &net.Dialer{Timeout: 5 * time.Second}
		config.DialContext = dialer.DialContext
	}
	return &GopacketTrafficMirror{config: config}
}

func (m *GopacketTrafficMirror) Start(ctx context.Context, source ObservedPacketSource) error {
	if source == nil {
		return errors.New("packet source is nil")
	}
	defer source.Close()

	session := &gopacketMirrorSession{mirror: m, ctx: ctx}
	pool := tcpassembly.NewStreamPool(&gopacketMirrorStreamFactory{
		session:      session,
		observedPort: m.config.ObservedPort,
	})
	assembler := tcpassembly.NewAssembler(pool)

	for {
		select {
		case <-ctx.Done():
			assembler.FlushAll()
			return ctx.Err()
		case packet, ok := <-source.Packets():
			if !ok {
				assembler.FlushAll()
				return session.err()
			}
			if packet == nil {
				continue
			}
			if err := m.assemblePacket(packet, assembler); err != nil {
				assembler.FlushAll()
				return err
			}
			if err := session.err(); err != nil {
				assembler.FlushAll()
				return err
			}
		}
	}
}

func (m *GopacketTrafficMirror) assemblePacket(packet gopacket.Packet, assembler *tcpassembly.Assembler) error {
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return nil
	}
	networkLayer := packet.NetworkLayer()
	if networkLayer == nil {
		return nil
	}
	tcp, ok := tcpLayer.(*layers.TCP)
	if !ok {
		return nil
	}
	seenAt := packet.Metadata().Timestamp
	if seenAt.IsZero() {
		seenAt = time.Now()
	}
	assembler.AssembleWithTimestamp(networkLayer.NetworkFlow(), tcp, seenAt)
	return nil
}

type gopacketMirrorSession struct {
	mirror   *GopacketTrafficMirror
	ctx      context.Context
	recordMu sync.Mutex
	errMu    sync.Mutex
	lastErr  error
}

func (s *gopacketMirrorSession) setErr(err error) {
	if err == nil {
		return
	}
	s.errMu.Lock()
	defer s.errMu.Unlock()
	if s.lastErr == nil {
		s.lastErr = err
	}
}

func (s *gopacketMirrorSession) err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.lastErr
}

func (s *gopacketMirrorSession) writeRecord(chunk *ObservedTrafficChunk, recordData []byte) error {
	if len(recordData) == 0 {
		return nil
	}
	if s.mirror.config.Writer == nil && s.mirror.config.WriteFunc == nil {
		return nil
	}

	writeCtx := context.WithValue(s.ctx, ListenType, chunk.Type)
	writeCtx = context.WithValue(writeCtx, FromIP, chunk.From)
	writeCtx = context.WithValue(writeCtx, ToIP, chunk.To)
	writeCtx = context.WithValue(writeCtx, FromTo, chunk.From+"..."+chunk.To)
	protoBytes := proto.WriteProto(recordData, chunk.Type, chunk.From+"..."+chunk.To)

	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	if s.mirror.config.WriteFunc != nil {
		_, err := s.mirror.config.WriteFunc(protoBytes, writeCtx)
		return err
	}
	_, err := s.mirror.config.Writer.Write(protoBytes)
	return err
}

type gopacketMirrorStreamFactory struct {
	session      *gopacketMirrorSession
	observedPort uint16
}

func (f *gopacketMirrorStreamFactory) New(networkFlow, transportFlow gopacket.Flow) tcpassembly.Stream {
	netSrc, netDst := networkFlow.Endpoints()
	portSrc, portDst := transportFlow.Endpoints()

	from := net.JoinHostPort(netSrc.String(), portSrc.String())
	to := net.JoinHostPort(netDst.String(), portDst.String())
	direction, ignore := detectTrafficDirection(portSrc.String(), portDst.String(), f.observedPort)

	return &gopacketMirrorStream{
		session:   f.session,
		from:      from,
		to:        to,
		direction: direction,
		ignore:    ignore,
	}
}

type gopacketMirrorStream struct {
	session   *gopacketMirrorSession
	from      string
	to        string
	direction TrafficDirection
	ignore    bool

	forwardMu   sync.Mutex
	forwardConn net.Conn
}

func (s *gopacketMirrorStream) Reassembled(reassemblies []tcpassembly.Reassembly) {
	if s.ignore {
		return
	}
	for _, reassembly := range reassemblies {
		if len(reassembly.Bytes) == 0 {
			continue
		}
		data := make([]byte, len(reassembly.Bytes))
		copy(data, reassembly.Bytes)
		seenAt := reassembly.Seen
		if seenAt.IsZero() {
			seenAt = time.Now()
		}
		chunk := &ObservedTrafficChunk{
			Data:      data,
			From:      s.from,
			To:        s.to,
			Direction: s.direction,
			Type:      TCPType,
			SeenAt:    seenAt,
		}
		if err := s.forwardChunk(chunk); err != nil {
			s.session.setErr(err)
			return
		}
		if err := s.recordChunk(chunk); err != nil {
			s.session.setErr(err)
			return
		}
	}
}

func (s *gopacketMirrorStream) forwardChunk(chunk *ObservedTrafficChunk) error {
	if s.session.mirror.config.TargetAddr == "" {
		return nil
	}
	forwardData, keepForward, err := s.session.mirror.config.ForwardHook.Handle(s.session.ctx, chunk)
	if err != nil {
		return err
	}
	if !keepForward || len(forwardData) == 0 {
		return nil
	}
	conn, err := s.ensureForwardConn()
	if err != nil {
		return err
	}
	s.forwardMu.Lock()
	defer s.forwardMu.Unlock()
	_, err = conn.Write(forwardData)
	return err
}

func (s *gopacketMirrorStream) recordChunk(chunk *ObservedTrafficChunk) error {
	recordData, keepRecord, err := s.session.mirror.config.RecordHook.Handle(s.session.ctx, chunk)
	if err != nil {
		return err
	}
	if !keepRecord || len(recordData) == 0 {
		return nil
	}
	return s.session.writeRecord(chunk, recordData)
}

func (s *gopacketMirrorStream) ensureForwardConn() (net.Conn, error) {
	s.forwardMu.Lock()
	defer s.forwardMu.Unlock()
	if s.forwardConn != nil {
		return s.forwardConn, nil
	}
	conn, err := s.session.mirror.config.DialContext(s.session.ctx, "tcp", s.session.mirror.config.TargetAddr)
	if err != nil {
		return nil, err
	}
	s.forwardConn = conn
	return conn, nil
}

func (s *gopacketMirrorStream) ReassemblyComplete() {
	s.forwardMu.Lock()
	defer s.forwardMu.Unlock()
	if s.forwardConn != nil {
		s.forwardConn.Close()
		s.forwardConn = nil
	}
}

func detectTrafficDirection(srcPort, dstPort string, observedPort uint16) (TrafficDirection, bool) {
	if observedPort == 0 {
		return TrafficDirectionUnknown, false
	}
	src, err := strconv.ParseUint(srcPort, 10, 16)
	if err != nil {
		return TrafficDirectionUnknown, true
	}
	dst, err := strconv.ParseUint(dstPort, 10, 16)
	if err != nil {
		return TrafficDirectionUnknown, true
	}
	if uint16(dst) == observedPort {
		return TrafficDirectionToObserved, false
	}
	if uint16(src) == observedPort {
		return TrafficDirectionFromObserved, false
	}
	return TrafficDirectionUnknown, true
}
