package test_server

import (
	"fmt"
	"net"
	"time"
)

func SimpleTCPReceiver() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(fmt.Sprintf("Failed to start server: %v", err))
	}
	defer listener.Close()

	fmt.Println("TCP server listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue
		}

		// 每个连接启动一个 goroutine 处理
		go handleConnection(conn)
	}
}
func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 打印客户端地址
	fmt.Printf("New connection from %s\n", conn.RemoteAddr())

	// 使用 bufio.Reader 提升读取效率（可选，也可直接用 conn.Read）
	//reader := bufio.NewReader(conn)

	buffer := make([]byte, 1024)
	for {
		// 方法1：按行读（适合文本协议）
		// line, err := reader.ReadString('\n')

		// 方法2：按块读（适合任意二进制数据）← 推荐通用方式
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Client %s disconnected: %v\n", conn.RemoteAddr(), err)
			return
		}

		// 打印收到的原始字节（以十六进制 + 可打印字符形式）
		fmt.Printf("[%s] Received %d bytes: %q\n",
			time.Now().Format("15:04:05"),
			n,
			string(buffer[:n]))
		time.Sleep(1 * time.Second)
		conn.Write([]byte("hello\n"))
		// 如果你只想打印可读文本（丢弃不可打印字符），可用：
		// fmt.Printf("[%s] Data: %s", time.Now().Format("15:04:05"), string(buffer[:n]))
	}
}
