package UsefullStructs

import (
	"bytes"
	"sync"
)

type BufferPool struct {
	pool *sync.Pool
}

func (b *BufferPool) Get() *bytes.Buffer {
	return b.pool.Get().(*bytes.Buffer)
}

func (b *BufferPool) Put(buffer *bytes.Buffer) {
	b.pool.Put(buffer)
}

func NewBufferPool(initSize int) *BufferPool {
	bufpool := &BufferPool{
		pool: &sync.Pool{
			New: func() any {
				buffer := bytes.NewBuffer(make([]byte, 0))
				buffer.Grow(1024 * 4)
				return buffer
			},
		},
	}
	for i := 0; i < initSize; i++ {
		bufpool.pool.Put(bufpool.pool.New())
	}
	return bufpool
}
