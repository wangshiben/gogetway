package proto

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"gogetway/Types"
	"io"
	"strings"
)

type Packet struct {
	Data         []byte
	Type         Types.ClientType
	From         string
	To           string
	reqTimestamp int64
}

func (p *Packet) Marshal() []byte {
	body := handleDataBody(p.Data)
	return writeHeader(body, p.Type, fmt.Sprintf("%s...%s", p.From, p.To))
}
func (p *Packet) Timestamp() int64 {
	return p.reqTimestamp
}
func NewPacket(data []byte, From, To string, PacketType Types.ClientType) *Packet {
	return &Packet{
		Data:         data,
		Type:         PacketType,
		From:         From,
		To:           To,
		reqTimestamp: 0,
	}
}

func UnMarshal(data []byte) (*Packet, error) {
	headers := []byte(MagicHeader)
	if bytes.HasPrefix(data, headers) {
		if len(data) <= len(MagicHeader)+17 {
			return nil, errors.New("packet too short")
		}
		index := len(MagicHeader)
		// 读时间戳
		timeStamp := make([]byte, 8)
		for i := 0; i < len(timeStamp); i++ {
			timeStamp[i] = data[index]
			index++
		}
		PacakgeType := data[index]
		index++
		PackLength := make([]byte, 8)
		for i := 0; i < len(PackLength); i++ {
			PackLength[i] = data[index]
			index++
		}
		Length := parseBytesToInt64(PackLength)
		FromTo := make([]byte, 0)
		currentByte := data[index]
		for currentByte != '\n' {
			if index >= len(data) {
				return nil, errors.New("not my packet you may read wrong bytes")
			}
			FromTo = append(FromTo, currentByte)
			index++
			currentByte = data[index]

		}
		FromToStr := string(FromTo)
		FromToArr := strings.SplitN(FromToStr, "...", 2)
		index++
		dataNeedParse := data[index:]
		after := removeNewlineAfter(dataNeedParse)
		if len(after) != int(Length) {
			return nil, errors.New("packet length error")
		}
		return &Packet{
			Data:         after,
			Type:         Types.ClientType(PacakgeType),
			From:         FromToArr[0],
			To:           FromToArr[1],
			reqTimestamp: parseBytesToInt64(timeStamp),
		}, nil
	}
	return nil, errors.New("not my packet you may read wrong bytes")
}

func removeNewlineAfter(data []byte) []byte {
	magicOriginal := []byte(MagicHeader)                  // e.g., "PREFIX\nt"
	magicPrefix := magicOriginal[:len(magicOriginal)-2]   // e.g., "PREFIX"
	magicExpanded := append(magicPrefix, '\n', '\n', 't') // "PREFIX\n\n t"

	return bytes.ReplaceAll(data, magicExpanded, magicOriginal)

}
func ReadProtoFromReader(reader io.Reader) (*bufio.Scanner, error) {
	reads := bufio.NewReader(reader)
	delim := NewChunkScannerWithDelim(reads, []byte(MagicHeader))
	return delim, nil
}

// NewChunkScannerWithDelim 创建一个 scanner，每次返回以 delim 开头的完整块（包含 delim）
func NewChunkScannerWithDelim(reader *bufio.Reader, delim []byte) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if len(delim) == 0 {
			return 0, nil, fmt.Errorf("empty delimiter")
		}

		// 第一次：找到第一个 delim
		firstIdx := bytes.Index(data, delim)
		if firstIdx == -1 {
			if atEOF {
				// 没有找到任何分隔符，返回空
				return len(data), nil, nil
			}
			return 0, nil, nil // 需要更多数据
		}

		// 从第一个 delim 开始的位置
		start := firstIdx
		searchFrom := start + len(delim)

		// 在剩余部分查找下一个 delim
		nextIdx := bytes.Index(data[searchFrom:], delim)
		if nextIdx == -1 {
			if atEOF {
				// 到达文件末尾，返回从 start 到 EOF
				return len(data), data[start:], nil
			}
			// 需要更多数据才能确定是否还有下一个 delim
			return 0, nil, nil
		}

		// 找到下一个 delim，当前 chunk 是 [start, searchFrom + nextIdx)
		end := searchFrom + nextIdx
		return end, data[start:end], nil
	})

	return scanner
}
