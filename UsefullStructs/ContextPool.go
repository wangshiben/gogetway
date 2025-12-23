package UsefullStructs

import (
	"context"
	"sync"
)

type ContextPool struct {
	pool *sync.Pool
}

func (c *ContextPool) GetContext() context.Context {
	return c.pool.Get().(context.Context)
}

func (c *ContextPool) Put(ctx context.Context) {
	ctx2, ok := ctx.(*Contexts)
	if ok {
		ctx2.Clear()
		c.pool.Put(ctx)
	}
}

func NewContextPool() *ContextPool {
	return &ContextPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return &Contexts{
					context: context.Background(),
					kvMap:   make(map[string]any),
				}
			},
		},
	}
}
