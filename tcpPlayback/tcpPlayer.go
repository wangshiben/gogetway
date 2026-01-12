package tcpPlayback

import (
	"errors"
	"gogetway/lockMap"
	"gogetway/proto"
	"io"
	"log"
	"net"
	"time"
)

type TcpPlayer struct {
	Target           string
	Client           string
	clientConn       net.Conn
	targetConn       net.Conn
	replayTime       bool         // 是否按照请求时间进行重放
	clientTimeRecord lockMap.Lock // 上次向client发送的包的时间信息
	targetTimeRecord lockMap.Lock // 上次向target发送的包时间信息
}

const (
	PackageTime = "packageTime"
	LastCalled  = "lastCalled"
)

// DataParser : parse data to proto.Packet,maybe you can change some data if you use your own proto(with time identify)
// DataParser : 解析数据为proto.Packet，你可以在这里修改数据，比如你使用了带有时间戳校验的信息
type DataParser func(data *proto.Packet) (*proto.Packet, error)

// SendSinglePacket : replay single forward packet
// SendSinglePacket : 回放单向的所有流量包
// SendSinglePacket:
// - reader : data reader
// - to : replay traffic to where address
// - parser : parse data to proto.Packet
func (t *TcpPlayer) SendSinglePacket(reader io.Reader, to string, parser DataParser) {
	fromReader, err := proto.ReadProtoFromReader(reader)
	if err != nil {
		panic(err)
		return
	}
	for fromReader.Scan() {
		bytes := fromReader.Bytes()
		packet, err := proto.UnMarshal(bytes)
		if err != nil {
			log.Default().Printf("unmarshal error: %s \n", err.Error())
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
}

// SendPacket : replay all packet in once tcp/http traffic
// SendPacket : 重放一个tcp/http的所有流量
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

		if isClient { // 如果是client，读取上次发送的包的时间
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
		// nil or other
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
		// packetCallTime-ptime-(milli-ltime)
		sleepTime := packetCallTime - ptime - milli + ltime
		time.Sleep(time.Duration(sleepTime))
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
