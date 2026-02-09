package reader

import "io"

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
