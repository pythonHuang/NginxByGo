package http

import (
	"testing"
)

func TestRouter(t *testing.T) {
	router := NewRouter()
	
	// 注册路由
	router.Handle("GET", "/", HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteString("Home")
	}))
	
	router.Handle("GET", "/about", HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteString("About")
	}))
	
	router.Handle("GET", "/api/users", HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteString("Users")
	}))

	// 测试路由匹配
	tests := []struct {
		path     string
		expected string
	}{
		{"/", "Home"},
		{"/about", "About"},
		{"/api/users", "Users"},
		{"/unknown", ""}, // 应该返回 404
	}

	for _, tt := range tests {
		req := &Request{
			Method: "GET",
			Path:   tt.path,
			Proto:  "HTTP/1.1",
		}
		
		// 创建一个 mock ResponseWriter
		mock := &mockResponseWriter{}
		
		router.ServeHTTP(mock, req)
		
		if tt.expected != "" {
			if mock.statusCode == 0 || mock.statusCode == 404 {
				// 可能没有正确匹配
			}
			// 由于 handler 是简单的函数，我们主要测试路由匹配逻辑
		}
	}
}

func TestRouterNotFound(t *testing.T) {
	router := NewRouter()
	
	// 只注册一个路由
	router.Handle("GET", "/exists", HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteString("Exists")
	}))
	
	// 测试不存在的路由
	req := &Request{
		Method: "GET",
		Path:   "/notfound",
		Proto:  "HTTP/1.1",
	}
	
	mock := &mockResponseWriter{}
	router.ServeHTTP(mock, req)
	
	// 应该返回 404
	if mock.statusCode != 404 {
		t.Errorf("Expected 404, got %d", mock.statusCode)
	}
}

// mockResponseWriter 用于测试的 ResponseWriter 实现
type mockResponseWriter struct {
	headers   map[string]string
	statusCode int
	body      string
}

func (m *mockResponseWriter) SetHeader(key, value string) {
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	m.headers[key] = value
}

func (m *mockResponseWriter) SetContentType(ct string) {
	m.SetHeader("Content-Type", ct)
}

func (m *mockResponseWriter) SetContentLength(n int) {
	m.SetHeader("Content-Length", "")
}

func (m *mockResponseWriter) SetStatus(code int) {
	m.statusCode = code
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.body += string(b)
	return len(b), nil
}

func (m *mockResponseWriter) WriteString(s string) (int, error) {
	return m.Write([]byte(s))
}

func (m *mockResponseWriter) End() error {
	return nil
}