package httpParser

import (
	"bufio"
	"net"
)

type PeekReader struct {
	reader  net.Conn
	scanner bufio.Scanner
	peek    []byte
}

func (p *PeekReader) FirstLine() []byte {
	if len(p.peek) != 0 {
		return p.peek
	}
	if p.scanner.Scan() {
		bytes := p.scanner.Bytes()
		p.peek = bytes
		return bytes
	}
	return nil
}

func (p *PeekReader) Read(data []byte) (int, error) {
	line := p.FirstLine()
	copy(data, line)
	return len(line), nil
}
func NewPeekReader(connect net.Conn) *PeekReader {
	scanner := bufio.NewScanner(connect)
	return &PeekReader{
		reader:  connect,
		scanner: *scanner,
	}
}
