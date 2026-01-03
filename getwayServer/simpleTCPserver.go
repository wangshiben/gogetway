package getwayServer

import (
	"bytes"
	"context"
	"fmt"
	"gogetway/Types"
	"gogetway/UsefullStructs"
	"gogetway/lockMap"
	"gogetway/logger"
	"gogetway/proto"
	"gogetway/reader"
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
	Writer *os.File //TODO: multi writer

	startAnalyze *UsefullStructs.LockValue[bool] // analyze

	listener      net.Listener
	resourceGroup ResourceGroup
	bufferPool    *UsefullStructs.BufferPool
	contextPool   *UsefullStructs.ContextPool

	writeFunc    WriteFunc                         // 写入函数，可以写入 文件/数据库
	currentIndex *UsefullStructs.LockValue[uint64] // startFrom uint 1
	writeQueue   *WriteQueue
}

type ClientRespParse func(ctx context.Context, DataPaket *proto.Packet) (isContinue bool, err error)
type WriteFunc func(data []byte, ctx context.Context) (offset int, err error)

func (s *SimpleTCPServer) StartListen() {
	listener, err := net.Listen("tcp", s.Port)
	logger.InitLogger()
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
			ctx := s.contextPool.GetContext()
			// TODO feature: you can init ctx with connection init
			resource, err := s.resourceGroup.GetResource(ctx, clientConn)
			if err != nil {
				return
			}
			// 启动两个 goroutine 实现双向转发
			go s.PackageToForward(targetConn, clientConn, resource) // client → target
			go s.PackageToClient(clientConn, targetConn, resource)  // target → client
		}(clientConn)
	}
}

// PackageToForward : Client Package to Forward IP 将Client端的TCP包转 发到指定IP端口
// Forward : Forward IP 转发地址
// Client : Client Conn 客户端连接
func (s *SimpleTCPServer) PackageToForward(Forward, Client net.Conn, Resource ConnectResource) {
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
	ctx.Put(FromIP, From)
	ctx.Put(ToIP, To)
	CountReader := reader.NewNetTeeReader(midReader, Resource.GetLock(), s.startRecording(buffer, ctx, Resource))

	// Single forward copy  from client to forwardIP 单向拷贝： client -> forward
	if s.startAnalyze.Get() {
		packet := proto.NewPacket(buffer.Bytes(), From, To, s.ListenType)
		s.ClientRespParse(ctx, packet)
	}
	io.Copy(Forward, CountReader)
	//currentIndex, err := CountReader.GetIndexed()
	//if err != nil {
	//	fmt.Printf("error: %s\n", err.Error())
	//}
	//io.Copy(Forward, midReader)
	//currentIndex := s.currentIndex.Get()
	//s.currentIndex.Set(currentIndex + 1)
	//bytesWrite := buffer.Bytes()
	//go func() {
	//	//defer s.contextPool.Put(ctx)
	//	s.writeQueue.AddItem(ctx, bytesWrite, currentIndex, s.writeDataGenerator)
	//}()

}

// PackageToClient :  Forward IP Package to Client 将指定IP端口的TCP包转发到Client
// Forward : Forward IP 转发地址
// Client : Client Conn 客户端连接
func (s *SimpleTCPServer) PackageToClient(Client, Forward net.Conn, Resource ConnectResource) {
	defer Forward.Close()
	defer Client.Close()
	buffer := s.bufferPool.Get()
	defer func() {
		buffer.Reset()
		s.bufferPool.Put(buffer)
	}()

	var midReader io.Reader
	ctx := s.contextPool.GetContext().(*UsefullStructs.Contexts)
	//CountReader := UsefullStructs.NewNetTeeReader(Forward, s.currentIndex, s.startRecording(buffer, ctx))
	switch s.ListenType {
	case TCPType:
		buffer, midReader = ReadTcpType(Forward, buffer)
	case HTTPType:
		buffer, midReader = ReadTcpType(Forward, buffer)
	}
	ctx.Put(ListenType, s.ListenType)
	From := Forward.RemoteAddr().String()
	To := Client.RemoteAddr().String()
	ctx.Put(FromTo, fmt.Sprintf("%s...%s", From, To))
	ctx.Put(FromIP, From)
	ctx.Put(ToIP, To)

	CountReader := reader.NewNetTeeReader(midReader, Resource.GetLock(), s.startRecording(buffer, ctx, Resource))

	// Single forward copy  from client to forwardIP 单向拷贝：从 client 到 forward

	// Data is Steam reading block
	io.Copy(Client, CountReader)
	//currentIndex, err := CountReader.GetIndexed()
	//if err != nil {
	//	fmt.Printf("error: %s\n", err.Error())
	//}
	//io.Copy(Client, midReader)
	//currentIndex := s.currentIndex.Get()
	//s.currentIndex.Set(currentIndex + 1)

}
func (s *SimpleTCPServer) startRecording(buffer *bytes.Buffer, ctx context.Context, resource ConnectResource) reader.ReadHook {
	return func(index uint64) (isContinue bool, err error) {
		bytesWrite := buffer.Bytes()
		defer buffer.Reset()
		if s.startAnalyze.Get() {
			packet := proto.NewPacket(buffer.Bytes(), ctx.Value(FromIP).(string), ctx.Value(ToIP).(string), s.ListenType)
			parse, err := s.ForwardRespParse(ctx, packet)
			if err != nil {
				return false, err
			}
			if !parse {
				return parse, nil
			}
		}
		go func() {
			resource.WriteQueue().AddItem(ctx, bytesWrite, index, writeDataGenerator(resource.Writer(), resource.WriteFunc()))
			//defer s.contextPool.Put(ctx)
		}()
		return true, nil
	}
}

func writeDataGenerator(writer io.Writer, writeFunc WriteFunc) WriteFunc {
	return func(data []byte, ctx context.Context) (offset int, err error) {
		ctxs, ok := ctx.(*UsefullStructs.Contexts)
		if ok {
			clientType := ctxs.Value(ListenType).(Types.ClientType)
			fromTo := ctxs.Value(FromTo).(string)
			writeProtoData := proto.WriteProto(data, clientType, fromTo)
			if writeFunc != nil {
				return writeFunc(writeProtoData, ctx)
			} else {
				n, err := writer.Write(writeProtoData)
				fileWriter, ok := writer.(*os.File)
				if ok {
					// if io.Writer is a file, sync it now
					err = fileWriter.Sync()
					if err != nil {
						fmt.Printf("error: %s\n", err.Error())
						return 0, err
					}
				}
				return n, err
			}
		}
		return 0, err
	}
}

func ReadTcpType(conn io.Reader, buffer *bytes.Buffer) (buffers *bytes.Buffer, midReader io.Reader) {
	reader := io.TeeReader(conn, buffer)
	return buffer, reader
}

func NewSimpleTCPServer(ForwardAdd, LocalAdd string, ListenType Types.ClientType) *SimpleTCPServer {
	file, err := os.OpenFile("log1.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
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
		//Writer:           file,
		startAnalyze: UsefullStructs.NewLockValue(false),
		listener:     nil,
		bufferPool:   UsefullStructs.NewBufferPool(10),
		contextPool:  UsefullStructs.NewContextPool(),
		writeFunc:    nil,
		//currentIndex:     UsefullStructs.NewLockValue(uint64(1)),
		writeQueue:    NewWriteQueue(context.Background()),
		resourceGroup: NewResourceGroup(file, lockMap.NewDefaultLockGroup(5), nil),
	}
}

// TODO 待实现
//func ReadHttpType(connect net.Conn, buffer *bytes.Buffer) (buffers *bytes.Buffer, midReader io.Reader) {
//
//}
