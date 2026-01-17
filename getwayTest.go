package main

import (
	"fmt"
	"gogetway/getwayServer"
	"io"
	"log"
	"net"
)

func forward(src, dst net.Conn) {
	defer src.Close()
	defer dst.Close()
	// 单向拷贝：从 src 到 dst
	fmt.Printf("Forwarding from %s to %s\n", src.RemoteAddr(), dst.RemoteAddr())
	io.Copy(dst, src)

}

func handleConnection(clientConn net.Conn) {
	// 连接到目标服务器 (localhost:8090)
	targetAddr := "127.0.0.1:8090"
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", targetAddr, err)
		clientConn.Close()
		return
	}

	// 启动两个 goroutine 实现双向转发
	go forward(clientConn, targetConn) // client → target
	go forward(targetConn, clientConn) // target → client
}

func main() {
	server := getwayServer.NewSimpleTCPServer("127.0.0.1:8000", ":8081", 1)
	server.StartListen()
}
