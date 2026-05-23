package tcpPlayback

import (
	"errors"
	"github.com/wangshiben/gogetway/lockMap"
	"github.com/wangshiben/gogetway/proto"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type TcpPlayer struct {
	Target           string
	Client           string
	clientConn       net.Conn
	targetConn       net.Conn
	replayTime       bool
	clientTimeRecord lockMap.Lock
	targetTimeRecord lockMap.Lock
}

const (
	PackageTime = "packageTime"
	LastCalled  = "lastCalled"
)

type DataParser func(data *proto.Packet) (*proto.Packet, error)

func (t *TcpPlayer) SendSinglePacket(reader io.Reader, to string, parser DataParser) {
	fromReader, err := proto.ReadProtoFromReader(reader)
	if err != nil {
		panic(err)
	}
	for fromReader.Scan() {
		bytes := fromReader.Bytes()
		packet, err := proto.UnMarshal(bytes)
		if err != nil {
			log.Default().Printf("unmarshal error: %s \n", err.Error())
			continue
		}
		if parser != nil {
			packet, err = parser(packet)
			if err != nil {
				log.Default().Printf("parse error: %s \n", err.Error())
				continue
			}
		}
		if packet != nil && packet.To == to {
			t.SendPacket(packet)
		}
	}
	if err := fromReader.Err(); err != nil {
		log.Default().Printf("read proto error: %s \n", err.Error())
	}
}

func (t *TcpPlayer) ReplayToTarget(reader io.Reader, recordedTo string, parser DataParser) (int, error) {
	if t.targetConn == nil {
		return 0, errors.New("target connection is nil")
	}
	if recordedTo == "" {
		return 0, errors.New("recorded target is empty")
	}
	fromReader, err := proto.ReadProtoFromReader(reader)
	if err != nil {
		return 0, err
	}
	count := 0
	for fromReader.Scan() {
		packet, err := proto.UnMarshal(fromReader.Bytes())
		if err != nil {
			log.Default().Printf("unmarshal error: %s \n", err.Error())
			continue
		}
		if parser != nil {
			packet, err = parser(packet)
			if err != nil {
				log.Default().Printf("parse error: %s \n", err.Error())
				continue
			}
		}
		if packet == nil || packet.To != recordedTo {
			continue
		}
		if t.replayTime {
			t.wait(packet.Timestamp(), t.targetTimeRecord)
		}
		_, err = t.targetConn.Write(packet.Data)
		if err != nil {
			return count, err
		}
		count++
	}
	if err := fromReader.Err(); err != nil {
		return count, err
	}
	if count == 0 {
		return 0, errors.New("no packet replayed")
	}
	return count, nil
}

func detectRecordedTarget(reader io.Reader) (string, error) {
	fromReader, err := proto.ReadProtoFromReader(reader)
	if err != nil {
		return "", err
	}
	counts := make(map[string]int)
	for fromReader.Scan() {
		packet, err := proto.UnMarshal(fromReader.Bytes())
		if err != nil {
			log.Default().Printf("unmarshal error: %s \n", err.Error())
			continue
		}
		if packet != nil {
			counts[packet.To]++
		}
	}
	if err := fromReader.Err(); err != nil {
		return "", err
	}
	recordedTo := ""
	maxCount := 0
	for to, count := range counts {
		if count > maxCount {
			recordedTo = to
			maxCount = count
		}
	}
	if recordedTo == "" {
		return "", errors.New("no valid packet found")
	}
	return recordedTo, nil
}

func DetectRecordedTargetFromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return detectRecordedTarget(file)
}

func ReplayFileToTarget(filePath, target string, replayTime bool, parser DataParser) (int, error) {
	recordedTo, err := DetectRecordedTargetFromFile(filePath)
	if err != nil {
		return 0, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	conn, err := net.Dial("tcp", target)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	player := NewTCPPlayer(target, conn, "", nil, replayTime)
	return player.ReplayToTarget(file, recordedTo, parser)
}

func (t *TcpPlayer) SendPacket(packet *proto.Packet) error {
	if packet.From == t.Client {
		t.waitingAndSend(true, packet.Timestamp())
		_, err := t.clientConn.Write(packet.Data)
		if err != nil {
			return err
		}
	} else if packet.From == t.Target {
		t.waitingAndSend(false, packet.Timestamp())
		_, err := t.targetConn.Write(packet.Data)
		if err != nil {
			return err
		}
	} else {
		return errors.New("packet from error")
	}
	return nil
}

func (t *TcpPlayer) waitingAndSend(isClient bool, packetCallTime int64) {
	if t.replayTime {
		if isClient {
			t.wait(packetCallTime, t.clientTimeRecord)
		} else {
			t.wait(packetCallTime, t.targetTimeRecord)
		}
	}
}

func (t *TcpPlayer) wait(packetCallTime int64, timeLock lockMap.Lock) {
	milli := time.Now().UnixMilli()
	other, ok := timeLock.Other().(map[string]int64)
	if !ok {
		if other != nil {
			panic("something wrong ,maybe you shouldn't change lock other")
		}
		other = make(map[string]int64)
		timeLock.UpdateOther(other)
	}
	ptime := other[PackageTime]
	ltime := other[LastCalled]
	if packetCallTime-ptime <= milli-ltime {
	} else {
		sleepTime := packetCallTime - ptime - milli + ltime
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		milli += sleepTime
	}
	other[LastCalled] = milli
	other[PackageTime] = packetCallTime
}

func NewTCPPlayer(target string, targetConnect net.Conn, Client string, ClientConn net.Conn, replayTime bool) *TcpPlayer {
	return &TcpPlayer{
		Target:           target,
		Client:           Client,
		clientConn:       ClientConn,
		targetConn:       targetConnect,
		replayTime:       replayTime,
		clientTimeRecord: lockMap.LockDefaultWithOther(make(map[string]int64), 0),
		targetTimeRecord: lockMap.LockDefaultWithOther(make(map[string]int64), 0),
	}
}
