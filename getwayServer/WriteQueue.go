package getwayServer

import (
	"context"
	"fmt"
	"gogetway/UsefullStructs"
	"gogetway/logger"
	"sync"
)

type WriteQueue struct {
	handleQueues       *QueueItem
	waitingQueueHeader *QueueItem
	CurrentIndex       *UsefullStructs.LockValue[uint64] // startFrom 0
	ctx                context.Context
}

func (w *WriteQueue) AddItem(ctx context.Context, Data []byte, Index uint64, HookWrite WriteFunc) {
	logger.LogInfo(fmt.Sprintf("AddItem: %d FROM To %s", Index, ctx.Value(FromTo).(string)))
	pushItem := &QueueItem{Data: Data, Index: Index, HookWrite: HookWrite, Lock: sync.Mutex{}, ctx: ctx}
	findPrev := w.waitingQueueHeader
	flag := true
	for flag {
		if findPrev.Next == nil {
			flag = false
			continue
		}
		if findPrev.Next.Index > Index && findPrev.Index < Index {
			//findPrev=findPrev.Next
			flag = false
			continue
		}
		findPrev = findPrev.Next
	}
	next := findPrev.Next
	findPrev.SetNext(pushItem)
	pushItem.SetNext(next)
	CanContinue := w.ContinueHandle() // 尝试启动队列处理
	if CanContinue {
		w.HandleQueue()
	}
}
func (w *WriteQueue) HandleQueue() {
	w.handleQueues.HookWrite(w.handleQueues.Data, w.handleQueues.ctx)
	w.handleQueues = nil
	//w.CurrentIndex.Set(w.CurrentIndex.Get() + 1)
	canNext := w.ContinueHandle()
	if canNext {
		w.HandleQueue()
	}

}

func NewWriteQueue(ctx context.Context) *WriteQueue {
	return &WriteQueue{
		handleQueues: nil,
		waitingQueueHeader: &QueueItem{
			Data:      nil,
			Index:     0,
			HookWrite: nil,
			Next:      nil,
			Lock:      sync.Mutex{},
		},
		CurrentIndex: UsefullStructs.NewLockValue(uint64(1)),
	}
}

func (w *WriteQueue) ContinueHandle() bool {
	next := w.waitingQueueHeader.Next
	// 尝试加入队列: 当前索引和队列索引相同且执行任务为空(即无正在执行任务)
	if next != nil && next.Index == w.CurrentIndex.Get() && w.handleQueues == nil {
		w.handleQueues = next
		w.waitingQueueHeader.SetNext(next.Next)
		w.CurrentIndex.Set(w.CurrentIndex.Get() + 1)
		next.SetNext(nil)
		return true
	}
	return false
}

type QueueItem struct {
	Data      []byte
	Index     uint64
	HookWrite WriteFunc
	Next      *QueueItem
	ctx       context.Context
	//Prev      *QueueItem
	Lock sync.Mutex
}

//	func (q *QueueItem) SetPrev(newPrv *QueueItem) {
//		q.Lock.Lock()
//		defer q.Lock.Unlock()
//		q.Prev = newPrv
//	}
func (q *QueueItem) SetNext(newNxt *QueueItem) {
	q.Lock.Lock()
	defer q.Lock.Unlock()
	q.Next = newNxt
}
