package UsefullStructs

import (
	"context"
	"sync"
	"time"
)

type Contexts struct {
	context context.Context
	lock    sync.RWMutex
	kvMap   map[string]any
}

func (c *Contexts) Deadline() (deadline time.Time, ok bool) {
	return c.context.Deadline()
}

func (c *Contexts) Done() <-chan struct{} {
	return c.context.Done()
}

func (c *Contexts) Err() error {
	return c.context.Err()
}
func (c *Contexts) Value(key any) any {
	s, ok := key.(string)
	if ok {
		c.lock.RLock()
		defer c.lock.RUnlock()
		return c.kvMap[s]
	}
	return c.context.Value(key)
}
func (c *Contexts) Clear() {
	c.kvMap = make(map[string]any)
	c.context = context.Background()
}
func (c *Contexts) Put(key string, value any) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.kvMap[key] = value
}
