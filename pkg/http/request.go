package http

import (
	"net"
	"bufio"
	"strings"
)

// Request HTTP 请求
type Request struct {
	Method  string            // GET, POST, etc.
	URI    string            // /path?query
	Proto  string            // HTTP/1.1

	Headers map[string]string
	Body    []byte

	RemoteAddr string
	Host      string

	Connection *Connection
	Server     *Server

	// 解析后的字段
	Path       string
	Query      string
}

// NewRequest 创建请求
func NewRequest() *Request {
	return &Request{
		Headers: make(map[string]string),
	}
}

// ParseRequest 解析请求
func ParseRequest(reader *bufio.Reader) (*Request, error) {
	req := NewRequest()

	// 读取请求行
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")

	// 解析请求行: GET /path HTTP/1.1
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return nil, ErrBadRequest
	}

	req.Method = parts[0]
	req.URI = parts[1]
	req.Proto = parts[2]

	// 解析 URL
	if idx := strings.Index(parts[1], "?"); idx != -1 {
		req.Path = parts[1][:idx]
		req.Query = parts[1][idx+1:]
	} else {
		req.Path = parts[1]
	}

	// 读取请求头
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		
		if line == "" {
			break
		}

		// 解析 header
		if idx := strings.Index(line, ":"); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			req.Headers[key] = value
		}
	}

	// 提取常用字段
	req.Host = req.Headers["Host"]
	if addr, ok := req.Headers["X-Real-IP"]; ok {
		req.RemoteAddr = addr
	}

	return req, nil
}

// GetHeader 获取请求头
func (r *Request) GetHeader(key string) string {
	return r.Headers[key]
}

// SetHeader 设置请求头
func (r *Request) SetHeader(key, value string) {
	r.Headers[key] = value
}