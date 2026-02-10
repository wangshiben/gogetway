package reader

import (
	"bytes"
	"errors"
	"io"
)

type ByteReader struct {
	reader io.Reader
}

func NewByteReader(reader io.Reader) *ByteReader {
	return &ByteReader{reader: reader}
}
func (b *ByteReader) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := b.reader.Read(buf)
	return buf[0], err
}
func (b *ByteReader) ReadUntilMatch(endBytes []byte) ([]byte, error) {
	if len(endBytes) == 0 {
		return nil, errors.New("you must give me endBytes")
	}

	var buffer bytes.Buffer
	// 用于滑动窗口匹配，只保留最多 len(endBytes) 个字节
	window := make([]byte, 0, len(endBytes))

	for {
		singleByte, err := b.ReadByte()
		if err != nil {
			return nil, err
		}
		buffer.WriteByte(singleByte)

		// 维护滑动窗口：只保留最后 len(endBytes) 个字节
		if len(window) == len(endBytes) {
			// 移除最旧的字节（左移）
			copy(window, window[1:])
			window[len(window)-1] = singleByte
		} else {
			window = append(window, singleByte)
		}

		// 检查是否匹配
		if len(window) == len(endBytes) && bytes.Equal(window, endBytes) {
			break
		}
	}

	return buffer.Bytes(), nil
}

func (b *ByteReader) ReadUtil(delim byte) ([]byte, error) {
	offset := 0
	buf := make([]byte, 1024)
	flag := true
	for flag {
		readByte, err := b.ReadByte()
		if err != nil {
			return nil, err
		}
		buf[offset] = readByte
		if offset >= len(buf) {
			tempBuf := make([]byte, len(buf)*2)
			copy(tempBuf, buf)
			buf = tempBuf
		}
		offset++
		flag = readByte != delim
	}
	return buf[:offset], nil
}
