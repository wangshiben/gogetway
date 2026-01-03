package getwayServer

import (
	"context"
	"gogetway/lockMap"
	"io"
	"net"
)

type ConnectResource interface {
	Writer() io.Writer
	WriteFunc() WriteFunc
	WriteQueue() *WriteQueue
	//currentIndex() *UsefullStructs.LockValue[uint64]
	GetLock() lockMap.Lock
	WriteType() string
}

type ResourceGroup interface {
	GetResource(ctx context.Context, Connect net.Conn) (resource ConnectResource, err error)
	NewResourceFunc(ctx context.Context, From string) NewResourceFunc
}
type NewResourceFunc func(ctx context.Context, From string) (resource ConnectResource, err error)
