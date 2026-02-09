package httpParser

import (
	"fmt"
	"net"
	"testing"
)

func TestNewHttpParser(t *testing.T) {
	port := "8888"

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Errorf("failed to listen on port %s: %w", port, err)
	}
	defer listener.Close()

	fmt.Printf("Listening on :%s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue
		}
		peekReader := NewPeekReader(conn)
		line := peekReader.FirstLine()
		// 将连接交给 NewHttpParser 创建的 reader
		reader := NewHttpParser(conn)
		header, err := reader.ParseHeader(line)
		if err != nil {
			return
		}
		fmt.Printf("header: %v\n", header)
		body, err := reader.ReadBody()
		if err != nil {
			return
		}
		fmt.Printf("body: %v\n", string(body))
	}
}
