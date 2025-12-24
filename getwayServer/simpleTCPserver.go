package getwayServer

import (
	"bytes"
	"context"
	"fmt"
	"gogetway/Types"
	"gogetway/UsefullStructs"
	"gogetway/proto"
	"io"
	"log"
	"net"
	"os"
)

// SimpleTCPServer : Simple TCP Server 简单TCP转发服务，实现监听，转发，写入功能
type SimpleTCPServer struct {
	// Forward : Forward IP 转发地址
	Forward string
	// Port : Listen Port 本地监听端口
	Port string
	// ListenType : Listen Type 监听类型: TCP / HTTP
	ListenType Types.ClientType
	// WriteType : Write Type 写入类型: File / Other TODO: // File类型和Other类型待实现，自定义写入方式写入/插件化接入
	WriteType string
	// ClientRespParse : Client Response Parse 客户端响应解析函数 TODO: 当startAnalyze时候才启用
	ClientRespParse ClientRespParse
	// ForwardRespParse : Forward Response Parse 转发响应解析函数 TODO: 当startAnalyze时候才启用
	ForwardRespParse ClientRespParse
	// default Writer in your disk as default writer
	Writer *os.File

	startAnalyze *UsefullStructs.LockValue[bool]

	listener net.Listener

	bufferPool  *UsefullStructs.BufferPool
	contextPool *UsefullStructs.ContextPool

	writeFunc    WriteFunc                         // 写入函数，可以写入 文件/数据库
	currentIndex *UsefullStructs.LockValue[uint64] // startFrom uint 1
	writeQueue   *WriteQueue
}

type ClientRespParse func(DataPaket *proto.Packet) (isContinue bool)
type WriteFunc func(data []byte, ctx context.Context) (offset int, err error)

func (s *SimpleTCPServer) StartListen() {
	listener, err := net.Listen("tcp", s.Port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", s.Port, err)
	}
	s.listener = listener
	defer listener.Close()
	log.Printf("TCP proxy listening on %s, forwarding to %s", s.Port, s.Forward)
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go func(clientConn net.Conn) {
			targetAddr := s.Forward
			targetConn, err := net.Dial("tcp", targetAddr)
			if err != nil {
				log.Printf("Failed to connect to target %s: %v", targetAddr, err)
				clientConn.Close()
				return
			}

			// 启动两个 goroutine 实现双向转发
			go s.PackageToForward(targetConn, clientConn) // client → target
			go s.PackageToClient(clientConn, targetConn)  // target → client
		}(clientConn)
	}
}

// PackageToForward : Client Package to Forward IP 将Client端的TCP包转发到指定IP端口
// Forward : Forward IP 转发地址
// Client : Client Conn 客户端连接
func (s *SimpleTCPServer) PackageToForward(Forward, Client net.Conn) {
	defer Forward.Close()
	defer Client.Close()
	buffer := s.bufferPool.Get()
	defer func() {
		buffer.Reset()
		s.bufferPool.Put(buffer)
	}()

	var midReader io.Reader
	ctx := s.contextPool.GetContext().(*UsefullStructs.Contexts)

	switch s.ListenType {
	case TCPType:
		buffer, midReader = ReadTcpType(Client, buffer)
	case HTTPType:
		buffer, midReader = ReadTcpType(Client, buffer)
	}
	ctx.Put(ListenType, s.ListenType)
	From := Client.RemoteAddr().String()
	To := Forward.RemoteAddr().String()
	ctx.Put(FromTo, fmt.Sprintf("%s...%s", From, To))

	// Single forward copy  from client to forwardIP 单向拷贝： client -> forward
	if s.startAnalyze.Get() {
		packet := proto.NewPacket(buffer.Bytes(), From, To, s.ListenType)
		s.ClientRespParse(packet)
	}
	io.Copy(Forward, midReader)
	currentIndex := s.currentIndex.Get()
	s.currentIndex.Set(currentIndex + 1)
	bytesWrite := buffer.Bytes()
	go func() {
		//defer s.contextPool.Put(ctx)
		s.writeQueue.AddItem(ctx, bytesWrite, currentIndex, s.writeDataGenerator)
	}()

}

// PackageToClient :  Forward IP Package to Client 将指定IP端口的TCP包转发到Client
// Forward : Forward IP 转发地址
// Client : Client Conn 客户端连接
func (s *SimpleTCPServer) PackageToClient(Client, Forward net.Conn) {
	defer Forward.Close()
	defer Client.Close()
	buffer := s.bufferPool.Get()
	defer func() {
		buffer.Reset()
		s.bufferPool.Put(buffer)
	}()

	var midReader io.Reader
	ctx := s.contextPool.GetContext().(*UsefullStructs.Contexts)

	switch s.ListenType {
	case TCPType:
		buffer, midReader = ReadTcpType(Forward, buffer)
	case HTTPType:
		buffer, midReader = ReadTcpType(Forward, buffer)
	}
	ctx.Put(ListenType, s.ListenType)
	From := Client.RemoteAddr().String()
	To := Forward.RemoteAddr().String()
	ctx.Put(FromTo, fmt.Sprintf("%s...%s", From, To))
	// Single forward copy  from client to forwardIP 单向拷贝：从 client 到 forward
	if s.startAnalyze.Get() {
		packet := proto.NewPacket(buffer.Bytes(), From, To, s.ListenType)
		s.ForwardRespParse(packet)
	}
	io.Copy(Client, midReader)
	currentIndex := s.currentIndex.Get()
	s.currentIndex.Set(currentIndex + 1)
	bytesWrite := buffer.Bytes()
	go func() {
		s.writeQueue.AddItem(ctx, bytesWrite, currentIndex, s.writeDataGenerator)
		//defer s.contextPool.Put(ctx)
	}()
}
func (s *SimpleTCPServer) writeDataGenerator(data []byte, ctx context.Context) (offset int, err error) {
	ctxs, ok := ctx.(*UsefullStructs.Contexts)
	if ok {
		clientType := ctxs.Value(ListenType).(Types.ClientType)
		fromTo := ctxs.Value(FromTo).(string)
		writeProtoData := proto.WriteProto(data, clientType, fromTo)
		if s.writeFunc != nil {
			return s.writeFunc(writeProtoData, ctx)
		} else {
			n, err := s.Writer.Write(writeProtoData)
			s.Writer.Sync()
			return n, err
		}
	}
	return 0, err
}

func ReadTcpType(conn net.Conn, buffer *bytes.Buffer) (buffers *bytes.Buffer, midReader io.Reader) {
	reader := io.TeeReader(conn, buffer)
	return buffer, reader
}

func NewSimpleTCPServer(ForwardAdd, LocalAdd string, ListenType Types.ClientType) *SimpleTCPServer {
	file, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil
	}
	return &SimpleTCPServer{
		Forward:          ForwardAdd,
		Port:             LocalAdd,
		ListenType:       ListenType,
		WriteType:        "",
		ClientRespParse:  nil,
		ForwardRespParse: nil,
		Writer:           file,
		startAnalyze:     UsefullStructs.NewLockValue(false),
		listener:         nil,
		bufferPool:       UsefullStructs.NewBufferPool(10),
		contextPool:      UsefullStructs.NewContextPool(),
		writeFunc:        nil,
		currentIndex:     UsefullStructs.NewLockValue(uint64(1)),
		writeQueue:       NewWriteQueue(context.Background()),
	}
}

// TODO 待实现
//func ReadHttpType(connect net.Conn, buffer *bytes.Buffer) (buffers *bytes.Buffer, midReader io.Reader) {
//
//}
