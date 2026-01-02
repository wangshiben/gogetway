package UsefullStructs

import (
	"sync"
	"sync/atomic"
)

type SafeMap[T any] struct {
	data   map[string]T
	opTime atomic.Int32
	lock   sync.RWMutex
}

func (s *SafeMap[T]) Get(key string) (value T, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	value, ok = s.data[key]
	return
}
func (s *SafeMap[T]) Set(key string, value T) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.opTime.Add(1)
	s.data[key] = value
}

// Iterator : no lock called
func (s *SafeMap[T]) Iterator(funcCall IteratorFunc[T]) {
	s.lock.Lock()
	for key, value := range s.data {
		funcCall(key, value)
	}
	s.lock.Unlock()
}
func (s *SafeMap[T]) Delete(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.opTime.Add(1)
	delete(s.data, key)
}

type IteratorFunc[T any] func(key string, value T)

func NewSafeMap[T any]() *SafeMap[T] {
	return &SafeMap[T]{
		data:   make(map[string]T),
		opTime: atomic.Int32{},
	}
}
