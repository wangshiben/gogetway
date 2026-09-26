package UsefullStructs

import (
	"testing"
	"time"
)

func TestSafeMapIteratorCallbackCanDelete(t *testing.T) {
	values := NewSafeMap[int]()
	values.Set("one", 1)
	values.Set("two", 2)

	done := make(chan struct{})
	go func() {
		values.Iterator(func(key string, _ int) {
			values.Delete(key)
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("iterator callback deadlocked while deleting from the same map")
	}

	if _, ok := values.Get("one"); ok {
		t.Fatal("expected first value to be deleted")
	}
	if _, ok := values.Get("two"); ok {
		t.Fatal("expected second value to be deleted")
	}
}
