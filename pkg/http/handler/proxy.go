package handler

import (
	"fmt"
	"net"
	"github.com/nginxgo/nginxgo/pkg/http"
)

// ProxyHandler 反向代理处理器
type ProxyHandler struct {
	Target string // 上游服务器地址
}

// NewProxyHandler 创建反向代理处理器
func NewProxyHandler(target string) *ProxyHandler {
	return &ProxyHandler{
		Target: target,
	}
}

// ServeHTTP 处理代理请求
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 连接到上游服务器
	conn, err := net.Dial("tcp", h.Target)
	if err != nil {
		w.SetStatus(502)
		w.WriteString("Bad Gateway")
		return
	}
	defer conn.Close()

	// 构建请求行
	requestLine := fmt.Sprintf("%s %s %s\r\n", r.Method, r.URI, r.Proto)

	// 构建请求头
	var headers string
	for k, v := range r.Headers {
		headers += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	// 添加代理头
	headers += fmt.Sprintf("X-Real-IP: %s\r\n", r.RemoteAddr)
	headers += fmt.Sprintf("X-Forwarded-For: %s\r\n", r.RemoteAddr)

	// 发送请求
	req := requestLine + headers + "\r\n"
	conn.Write([]byte(req))

	// 读取响应
	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil {
		w.SetStatus(502)
		w.WriteString("Bad Gateway")
		return
	}

	// 解析响应并发送
	w.Write(buf[:n])
}

// ProxyPassReverseHandler 反向代理处理器 (基于配置的)
type ProxyPassReverseHandler struct {
	Upstream string
}

// NewProxyPassReverseHandler 创建反向代理处理器
func NewProxyPassReverseHandler(upstream string) *ProxyPassReverseHandler {
	return &ProxyPassReverseHandler{
		Upstream: upstream,
	}
}

func (h *ProxyPassReverseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 简单实现：直接转发
	// 实际应该使用 http.Client
	w.SetStatus(502)
	w.WriteString("Proxy not implemented")
}