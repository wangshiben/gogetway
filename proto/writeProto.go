package proto

import (
	"bytes"
	"encoding/binary"
	"gogetway/Types"
	"time"
)

func WriteProto(src []byte, PacType Types.ClientType, FromTo string) []byte {
	result := handleDataBody(src)

	return writeHeader(result, PacType, FromTo)
}

func handleDataBody(src []byte) []byte {
	// 要查找的模式：\n + 't'
	pattern := []byte{'\n', 't'}

	if len(src) < len(pattern) {
		return src
	}

	// 第一次遍历：统计非重叠匹配次数
	count := 0
	i := 0
	for i <= len(src)-len(pattern) {
		if src[i] == '\n' && src[i+1] == 't' {
			count++
			i += len(pattern) // 跳过整个 "\nt"，避免重叠
		} else {
			i++
		}
	}

	if count == 0 {
		return src // 无匹配，直接返回（或 copy）
	}

	// 每个匹配增加 1 字节（"\nt" → "\n\nt"）
	result := make([]byte, len(src)+count)
	j := 0
	i = 0

	// 第二次遍历：构建结果
	for i < len(src) {
		if i <= len(src)-2 && src[i] == '\n' && src[i+1] == 't' {
			// 写入 \n \n t
			result[j] = '\n'
			j++
			result[j] = '\n' // 插入额外的 \n
			j++
			result[j] = 't'
			j++
			i += 2 // 跳过原始的 \n t
		} else {
			result[j] = src[i]
			j++
			i++
		}
	}

	return result
}
func writeHeader(src []byte, PacType Types.ClientType, FromTo string) []byte {
	nowStamp := time.Now().UnixNano() // 获取当前时间戳(纳秒级)
	// 当前二进制写入包头格式为: MagicNum+时间戳(8byte)+包类型(1byte)+包长度(int64,8byte)+formTo+\n+包内容
	res := []byte(MagicHeader)
	var buffer bytes.Buffer
	buffer.Grow(len(res) + 8 + 1 + 8 + len(FromTo) + 1 + len(src))
	buffer.Write(res)
	TimeStamp := parseInt64ToBytes(nowStamp)
	buffer.Write(TimeStamp)
	buffer.WriteByte(byte(PacType))
	buffer.Write(parseInt64ToBytes(int64(len(src))))
	buffer.Write([]byte(FromTo))
	buffer.Write([]byte("\n"))
	buffer.Write(src)
	return buffer.Bytes()
}
func parseInt64ToBytes(data int64) []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint64(res[:], uint64(data))
	return res
}
func parseBytesToInt64(data []byte) int64 {
	res := binary.LittleEndian.Uint64(data)
	return int64(res)
}
