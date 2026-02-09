package httpParser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
)

type HttpRequestReader struct {
	connection     net.Conn
	headData       []byte
	peekLength     int
	bodyData       []byte
	requestVersion int // 1: 1.x 2: 2.x 3: 3.x
	dataLength     int //
}

func (h *HttpRequestReader) ReadHeader(peekData []byte) ([]byte, error) {
	if len(h.headData) != 0 {
		return h.headData, nil
	}

	realHeader := make([]byte, len(peekData))
	h.peekLength = len(peekData)
	copy(realHeader, peekData)
	header, err := h.readOriginHeader()
	if err != nil {
		return nil, err
	}
	realHeader = append(realHeader, header...)
	h.headData = realHeader
	return realHeader, nil
}
func (h *HttpRequestReader) readOriginHeader() ([]byte, error) {
	res := make([]byte, 0)
	reader := bufio.NewReader(h.connection)
	for {
		line, readErr := reader.ReadBytes('\n')
		if readErr != nil {
			return nil, readErr
		}

		res = append(res, line...)

		// 检查是否是空行（即 "\r\n"）
		if len(line) == 2 && line[0] == '\r' && line[1] == '\n' {
			break // HTTP 头部结束
		}
	}
	return res, nil
}

func (h *HttpRequestReader) ParseHeader(peekData []byte) (http.Header, error) {
	header, err := h.ReadHeader(peekData)
	if err != nil {
		return nil, err
	}
	httpHeader, err := parseHTTPHeader(header)
	if err != nil {
		return nil, err
	}
	length := httpHeader.Get("Content-Length")
	if length != "" {
		h.dataLength, err = fmt.Sscanf(length, "%d", &h.dataLength)
		if err != nil {
			return nil, err
		}
	}
	return httpHeader, nil
}

func (h *HttpRequestReader) ReadBody() ([]byte, error) {
	if len(h.headData) == 0 {
		return nil, errors.New("you must read header first")
	}
	if len(h.bodyData) != 0 && h.dataLength != 0 {
		bodyData := make([]byte, h.dataLength)
		reader := bufio.NewReader(h.connection)
		_, err := io.ReadFull(reader, bodyData)
		if err != nil {
			return nil, err
		}
		h.bodyData = bodyData
		return bodyData, nil
	}
	return h.bodyData, nil

}
func NewHttpParser(connect net.Conn) *HttpRequestReader {
	return &HttpRequestReader{
		connection: connect,
		headData:   make([]byte, 0),
		bodyData:   make([]byte, 0),
	}
}

// parseHTTPHeader 从完整的 HTTP 请求头字节中提取 http.Header
// 注意：headerBytes 应包含请求行 + 头部 + 结尾的 \r\n\r\n
func parseHTTPHeader(headerBytes []byte) (http.Header, error) {
	reader := bufio.NewReader(bytes.NewReader(headerBytes))

	// 1. 读取并丢弃请求行（第一行）
	_, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read request line: %w", err)
	}

	// 2. 使用 textproto.Reader 解析 MIME 头部
	tpReader := textproto.NewReader(reader)
	mimeHeader, err := tpReader.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}

	// 3. 转换为 http.Header（其实 mimeHeader 就是 http.Header 的底层类型）
	return http.Header(mimeHeader), nil
}
