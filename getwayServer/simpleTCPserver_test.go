package getwayServer

import (
	"testing"
)

func TestNewSimpleTCPServer(t *testing.T) {
	server := NewSimpleTCPServer("127.0.0.1:8000", ":8081", TCPType)
	server.StartListen()
}
