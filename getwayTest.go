package main

import (
	"fmt"
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
	listenAddr := ":8888"
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
	}
	defer listener.Close()

	log.Printf("TCP proxy listening on %s, forwarding to localhost:8090", listenAddr)

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(clientConn)
	}
}
