package http

import (
	"fmt"
	"net"
	"strings"
)

// Handler HTTP 处理器接口
type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}

// HandlerFunc 函数式处理器
type HandlerFunc func(w ResponseWriter, r *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

// Router 路由器
type Router struct {
	trees map[string]*node  // GET, POST, etc.
}

// NewRouter 创建路由器
func NewRouter() *Router {
	return &Router{
		trees: make(map[string]*node),
	}
}

// Handle 注册路由
func (r *Router) Handle(method, path string, handler Handler) {
	if r.trees[method] == nil {
		r.trees[method] = &node{}
	}
	r.trees[method].addRoute(path, handler)
}

// Get 注册 GET 路由
func (r *Router) Get(path string, handler Handler) {
	r.Handle("GET", path, handler)
}

// Post 注册 POST 路由
func (r *Router) Post(path string, handler Handler) {
	r.Handle("POST", path, handler)
}

// ServeHTTP 处理请求
func (r *Router) ServeHTTP(w ResponseWriter, req *Request) {
	method := req.Method
	path := req.Path

	tree := r.trees[method]
	if tree == nil {
		w.SetStatus(404)
		w.WriteString("Not Found")
		return
	}

	handler := tree.getRoute(path)
	if handler == nil {
		w.SetStatus(404)
		w.WriteString("Not Found")
		return
	}

	handler.ServeHTTP(w, req)
}

// node 路由树节点
type node struct {
	path     string
	handler  Handler
	children map[string]*node
	wildcard *node
}

func (n *node) addRoute(path string, handler Handler) {
	// 简化的路由实现
	if n.path == "" {
		n.path = path
		n.handler = handler
		return
	}

	// 简单的前缀匹配
	if !strings.HasPrefix(path, n.path) {
		return
	}

	if n.children == nil {
		n.children = make(map[string]*node)
	}

	// 处理通配符
	if path == "*" || strings.HasSuffix(path, ".*") {
		n.wildcard = &node{path: path, handler: handler}
		return
	}

	childPath := strings.TrimPrefix(path, n.path)
	if len(childPath) > 0 && childPath[0] == '/' {
		childPath = childPath[1:]
	}

	if n.children[childPath] == nil {
		n.children[childPath] = &node{path: childPath, handler: handler}
	}
}

func (n *node) getRoute(path string) Handler {
	if n.path == path {
		return n.handler
	}

	// 检查子节点
	if n.children != nil {
		for _, child := range n.children {
			if handler := child.getRoute(path); handler != nil {
				return handler
			}
		}
	}

	// 检查通配符
	if n.wildcard != nil {
		return n.wildcard.handler
	}

	return nil
}

// Server HTTP 服务器
type Server struct {
	Addr      string
	Port      int
	Name      string

	Handler   *Router
	TLSConfig interface{}
}

// NewServer 创建服务器
func NewServer(addr string, port int) *Server {
	return &Server{
		Addr:    addr,
		Port:    port,
		Handler: NewRouter(),
	}
}

// Serve 开始服务
func (s *Server) Serve() error {
	addr := fmt.Sprintf("%s:%d", s.Addr, s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	fmt.Printf("Server listening on %s\n", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		// 读取请求
		req, err := ParseRequest(reader)
		if err != nil {
			return
		}

		req.RemoteAddr = conn.RemoteAddr().String()
		req.Connection = &Connection{Conn: conn}

		// 创建响应
		resp := NewResponse(req, conn)

		// 路由处理
		s.Handler.ServeHTTP(resp, req)

		// 检查是否需要关闭连接
		connType := req.GetHeader("Connection")
		if strings.ToLower(connType) == "close" {
			return
		}
	}
}

// ResponseWriter 响应写入接口
type ResponseWriter interface {
	SetHeader(key, value string)
	SetContentType(ct string)
	SetContentLength(n int)
	SetStatus(code int)
	Write(b []byte) (int, error)
	WriteString(s string) (int, error)
	End() error
}