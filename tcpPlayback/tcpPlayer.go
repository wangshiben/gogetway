package tcpPlayback

import (
	"errors"
	"gogetway/proto"
	"net"
)

type TcpPlayer struct {
	Target     string
	Client     string
	clientConn net.Conn
	targetConn net.Conn
}

func (t *TcpPlayer) SendPacket(packet proto.Packet) error {
	if packet.From == t.Client {
		_, err := t.clientConn.Write(packet.Data)
		if err != nil {
			return err
		}
	} else if packet.From == t.Target {
		_, err := t.targetConn.Write(packet.Data)
		if err != nil {
			return err
		}
	} else {
		return errors.New("packet from error")
	}
	return nil
}
