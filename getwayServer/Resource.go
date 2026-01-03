package getwayServer

import (
	"gogetway/lockMap"
	"io"
)

type DefaultResource struct {
	writer     io.Writer
	writeFunc  WriteFunc
	writeQueue *WriteQueue
	lock       lockMap.Lock
	writeType  string
}

func (d DefaultResource) Writer() io.Writer {
	return d.writer
}

func (d DefaultResource) WriteFunc() WriteFunc {
	return d.writeFunc
}

func (d DefaultResource) WriteQueue() *WriteQueue {
	return d.writeQueue
}

func (d DefaultResource) GetLock() lockMap.Lock {
	return d.lock
}

func (d DefaultResource) WriteType() string {
	return d.writeType
}
