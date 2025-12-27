package UsefullStructs

import (
	"errors"
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

func (n *NetTeeReader) Read(bytes []byte) (int, error) {
	read, err := n.teeReader.Read(bytes)
	if err != nil {
		return 0, err
	}
	if !n.isRead && read > 0 {
		n.readLock.Lock()

		index := n.lockIndex.Get()
		n.loadIndex = index
		n.isRead = true
		n.lockIndex.Set(index + uint64(1))
		n.readLock.Unlock()
	}
	return read, err
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
