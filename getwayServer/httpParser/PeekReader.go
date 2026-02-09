package httpParser

import (
	"gogetway/reader"
	"net"
)

type PeekReader struct {
	reader  net.Conn
	scanner *reader.ByteReader
	peek    []byte
}

func (p *PeekReader) FirstLine() []byte {
	if len(p.peek) != 0 {
		return p.peek
	}
	bytes, err := p.scanner.ReadUtil('\n')
	if err != nil {
		return nil
	}
	p.peek = bytes
	return p.peek
}

func (p *PeekReader) Read(data []byte) (int, error) {
	line := p.FirstLine()
	copy(data, line)
	return len(line), nil
}
func NewPeekReader(connect net.Conn) *PeekReader {
	scanner := reader.NewByteReader(connect)
	return &PeekReader{
		reader:  connect,
		scanner: scanner,
	}
}
