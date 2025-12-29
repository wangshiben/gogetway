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
	readHook  ReadHook
}
type ReadHook func(index uint64) (isContinue bool, err error)

func (n *NetTeeReader) Read(bytes []byte) (int, error) {
	//fmt.Println("readCalled")
	firstBytes := make([]byte, 1)
	hook := false
	isRead, err := n.teeReader.Read(firstBytes)
	if isRead > 0 {
		get, unlockFunc := n.lockIndex.LockGet()
		defer unlockFunc()
		index := get
		n.loadIndex = index
		n.isRead = true
		n.lockIndex.SetInLock(index+uint64(1), index)
		bytes[0] = firstBytes[0]
		hook = true
	}
	read, err := n.teeReader.Read(bytes[1:])
	if err != nil {
		return isRead, err
	}
	if hook {
		readHook, err := n.readHook(n.loadIndex)
		if err != nil {
			return 0, err
		}
		if !readHook {
			return 0, errors.New("filter req")
		}
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
func NewNetTeeReader(teeReader io.Reader, LockedValue *LockValue[uint64], readHook ReadHook) *NetTeeReader {
	return &NetTeeReader{
		teeReader: teeReader,
		lockIndex: LockedValue,
		readLock:  sync.RWMutex{},
		readHook:  readHook,
	}
}
