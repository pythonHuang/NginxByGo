package http

import (
	"bufio"
	"net"
	"fmt"
	"strconv"
	"strings"
)

// Response HTTP 响应
type Response struct {
	StatusCode int            // 200, 404, etc.
	StatusText string         // OK, Not Found

	Headers map[string]string
	Body    []byte

	Request *Request
	conn    net.Conn

	sentHeaders bool
}

// NewResponse 创建响应
func NewResponse(req *Request, conn net.Conn) *Response {
	return &Response{
		StatusCode: 200,
		StatusText: "OK",
		Headers:    make(map[string]string),
		Request:    req,
		conn:       conn,
	}
}

// SetHeader 设置响应头
func (r *Response) SetHeader(key, value string) {
	r.Headers[key] = value
}

// SetContentType 设置 Content-Type
func (r *Response) SetContentType(ct string) {
	r.SetHeader("Content-Type", ct)
}

// SetContentLength 设置 Content-Length
func (r *Response) SetContentLength(n int) {
	r.SetHeader("Content-Length", strconv.Itoa(n))
}

// SetStatus 设置状态码
func (r *Response) SetStatus(code int) {
	r.StatusCode = code
	r.StatusText = StatusText(code)
}

// WriteHeader 写入响应头
func (r *Response) WriteHeader() error {
	if r.sentHeaders {
		return nil
	}

	// 构建状态行
	statusLine := fmt.Sprintf("%s %d %s\r\n", r.Request.Proto, r.StatusCode, r.StatusText)

	// 构建头部
	var header strings.Builder
	header.WriteString(statusLine)

	for k, v := range r.Headers {
		header.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	header.WriteString("\r\n")

	// 发送
	r.conn.Write([]byte(header.String()))
	r.sentHeaders = true

	return nil
}

// Write 写入响应体
func (r *Response) Write(b []byte) (int, error) {
	if !r.sentHeaders {
		r.SetContentLength(len(b))
		r.WriteHeader()
	}

	return r.conn.Write(b)
}

// WriteString 写入字符串
func (r *Response) WriteString(s string) (int, error) {
	return r.Write([]byte(s))
}

// End 结束响应
func (r *Response) End() error {
	if !r.sentHeaders && len(r.Body) > 0 {
		r.SetContentLength(len(r.Body))
		r.WriteHeader()
		if len(r.Body) > 0 {
			r.conn.Write(r.Body)
		}
	}
	return nil
}

// Flush 刷新缓冲区
func (r *Response) Flush() {
	if writer, ok := r.conn.(*bufio.Writer); ok {
		writer.Flush()
	}
}

// StatusText 返回状态码对应的文本
func StatusText(code int) string {
	text, ok := statusText[code]
	if !ok {
		return "Unknown"
	}
	return text
}

var statusText = map[int]string{
	100: "Continue",
	101: "Switching Protocols",
	200: "OK",
	201: "Created",
	202: "Accepted",
	204: "No Content",
	301: "Moved Permanently",
	302: "Found",
	304: "Not Modified",
	400: "Bad Request",
	401: "Unauthorized",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	408: "Request Timeout",
	413: "Payload Too Large",
	414: "URI Too Long",
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
}