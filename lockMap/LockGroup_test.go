package lockMap

import (
	"context"
	"testing"
	"time"
)

func TestCheckLocksRemovesReleasableLockWithoutDeadlock(t *testing.T) {
	group := NewDefaultLockGroup(5)
	if _, err := group.CreateLock(context.Background(), "stale"); err != nil {
		t.Fatalf("create lock: %v", err)
	}

	done := make(chan struct{})
	go func() {
		group.CheckLocks(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("lock cleanup deadlocked")
	}

	if _, ok := group.lockMap.Get("stale"); ok {
		t.Fatal("expected releasable lock to be removed")
	}
}
