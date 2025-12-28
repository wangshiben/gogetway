package UsefullStructs

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

type NetTeeReader struct {
	teeReader io.Reader
	lockIndex *LockValue[uint64]
	isRead    bool
	loadIndex uint64
	readLock  sync.RWMutex
}
type ReadHook func(index uint64)

func (n *NetTeeReader) Read(bytes []byte) (int, error) {
	fmt.Println("readCalled")
	firstBytes := make([]byte, 1)
	isRead, err := n.teeReader.Read(firstBytes)
	if !n.isRead && isRead > 0 {
		get, unlockFunc := n.lockIndex.LockGet()
		defer unlockFunc()
		n.readLock.Lock()
		index := get
		n.loadIndex = index
		n.isRead = true
		n.lockIndex.SetInLock(index+uint64(1), index)
		n.readLock.Unlock()

	}
	if isRead > 0 {
		bytes[0] = firstBytes[0]
	}
	read, err := n.teeReader.Read(bytes[1:])
	if err != nil {
		return isRead, err
	}

	return read + isRead, err
}
func (n *NetTeeReader) GetIndexed() (uint64, error) {
	n.readLock.RLock()
	defer n.readLock.RUnlock()
	if !n.isRead {
		return 0, errors.New("not read")
	}
	return n.loadIndex, nil
}
func NewNetTeeReader(teeReader io.Reader, LockedValue *LockValue[uint64]) *NetTeeReader {
	return &NetTeeReader{
		teeReader: teeReader,
		lockIndex: LockedValue,
		readLock:  sync.RWMutex{},
	}
}
