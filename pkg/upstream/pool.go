package upstream

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Pool 连接池
type Pool struct {
	Name     string
	Servers  []*Server
	LB       LoadBalancer
	timeout  time.Duration

	mu       sync.RWMutex
}

// NewPool 创建连接池
func NewPool(name string, servers []*Server, lbType LoadBalancerType) *Pool {
	lb := NewLoadBalancer(lbType, servers)

	return &Pool{
		Name:    name,
		Servers: servers,
		LB:      lb,
		timeout: 10 * time.Second,
	}
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(t LoadBalancerType, servers []*Server) LoadBalancer {
	switch t {
	case LBLeastConn:
		return NewLeastConn(servers)
	case LBIPHash:
		return NewIPHash(servers)
	default:
		return NewRoundRobin(servers)
	}
}

// Get 获取后端服务器
func (p *Pool) Get() *Backend {
	server := p.LB.Next()
	if server == nil {
		return nil
	}

	return &Backend{
		Server:  server,
		Pool:    p,
		Request: nil,
	}
}

// Backend 后端连接
type Backend struct {
	Server  *Server
	Pool    *Pool
	Request *http.Request

	resp *http.Response
	conn net.Conn
}

// Release 释放连接
func (b *Backend) Release() {
	b.Server.mu.Lock()
	b.Server.ActiveConns--
	b.Server.mu.Unlock()
}

// Do 执行请求
func (b *Backend) Do(req *http.Request) (*http.Response, error) {
	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: b.Pool.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 设置代理头
	req.Header.Set("X-Real-IP", b.Server.URL)
	req.Header.Set("X-Forwarded-For", b.Server.URL)

	// 发送请求
	return client.Do(req)
}

// Ping 检查服务器健康状态
func (s *Server) Ping() bool {
	conn, err := net.DialTimeout("tcp", s.URL, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// NewHealthCheck 创建健康检查
func NewHealthCheck(pool *Pool, interval time.Duration) *HealthCheck {
	return &HealthCheck{
		Pool:     pool,
		Interval: interval,
		stop:    make(chan struct{}),
	}
}

// HealthCheck 健康检查
type HealthCheck struct {
	Pool     *Pool
	Interval time.Duration
	stop     chan struct{}
}

// Start 启动健康检查
func (h *HealthCheck) Start() {
	go func() {
		ticker := time.NewTicker(h.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.check()
			case <-h.stop:
				return
			}
		}
	}()
}

// Stop 停止健康检查
func (h *HealthCheck) Stop() {
	close(h.stop)
}

func (h *HealthCheck) check() {
	for _, server := range h.Pool.Servers {
		if server.Ping() {
			server.mu.Lock()
			server.Fails = 0
			server.Down = false
			server.mu.Unlock()
		} else {
			server.MarkFailed()
		}
	}
}

// String 实现 Stringer 接口
func (p *Pool) String() string {
	return fmt.Sprintf("upstream %s (%d servers)", p.Name, len(p.Servers))
}