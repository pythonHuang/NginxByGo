package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"io"
	"github.com/nginxgo/nginxgo/pkg/http"
)

// ProxyHandler 反向代理处理器
type ProxyHandler struct {
	Target   string      // 上游服务器地址
	Client   *http.Client
}

// NewProxyHandler 创建反向代理处理器
func NewProxyHandler(target string) *ProxyHandler {
	return &ProxyHandler{
		Target: target,
		Client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// ServeHTTP 处理代理请求
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 解析目标 URL
	targetURL, err := url.Parse(h.Target)
	if err != nil {
		w.SetStatus(500)
		w.WriteString("Bad Gateway")
		return
	}

	// 构建转发请求
	req, err := http.NewRequest(r.Method, r.URI, nil)
	if err != nil {
		w.SetStatus(500)
		w.WriteString("Bad Gateway")
		return
	}

	// 设置 headers
	for k, v := range r.Headers {
		if k == "Host" {
			req.Host = targetURL.Host
		} else {
			req.Header.Set(k, v)
		}
	}

	// 添加代理头
	req.Header.Set("X-Real-IP", r.RemoteAddr)
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)
	req.Header.Set("X-Forwarded-Proto", "http")

	// 发送请求
	resp, err := h.Client.Do(req)
	if err != nil {
		w.SetStatus(502)
		w.WriteString("Bad Gateway")
		return
	}
	defer resp.Body.Close()

	// 复制响应状态
	w.SetStatus(resp.StatusCode)

	// 复制 headers
	for k, v := range resp.Header {
		for _, val := range v {
			w.SetHeader(k, val)
		}
	}

	// 复制 body
	io.Copy(w, resp.Body)
}

// ProxyPassReverseHandler 反向代理处理器 (重构版本)
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
	// 构建上游 URL
	upstreamURL := h.Upstream + r.Path

	// 创建请求
	req, _ := http.NewRequest(r.Method, upstreamURL, nil)

	// 转发 headers
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	// 添加代理头
	req.Header.Set("X-Real-IP", r.RemoteAddr)
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)

	// 使用 http.DefaultClient
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.SetStatus(502)
		w.WriteString(fmt.Sprintf("Bad Gateway: %v", err))
		return
	}
	defer resp.Body.Close()

	// 复制响应
	w.SetStatus(resp.StatusCode)
	for k, v := range resp.Header {
		w.SetHeader(k, v[0])
	}

	io.Copy(w, resp.Body)
}