package upstream

import (
	"sync"
	"time"
)

// LoadBalancerType 负载均衡类型
type LoadBalancerType int

const (
	LBRoundRobin LoadBalancerType = iota
	LBLeastConn
	LBIPHash
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Next() *Server
	Servers() []*Server
}

// RoundRobin 轮询负载均衡
type RoundRobin struct {
	servers []*Server
	current int
	mu      sync.Mutex
}

// NewRoundRobin 创建轮询负载均衡
func NewRoundRobin(servers []*Server) *RoundRobin {
	return &RoundRobin{
		servers: servers,
		current: 0,
	}
}

func (r *RoundRobin) Next() *Server {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.servers) == 0 {
		return nil
	}

	server := r.servers[r.current]
	r.current = (r.current + 1) % len(r.servers)

	return server
}

func (r *RoundRobin) Servers() []*Server {
	return r.servers
}

// LeastConn 最少连接
type LeastConn struct {
	servers []*Server
	mu      sync.Mutex
}

func NewLeastConn(servers []*Server) *LeastConn {
	return &LeastConn{servers: servers}
}

func (l *LeastConn) Next() *Server {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.servers) == 0 {
		return nil
	}

	var minConn *Server
	minVal := 1<<31 - 1

	for _, s := range l.servers {
		s.mu.Lock()
		conns := s.ActiveConns
		s.mu.Unlock()
		
		if conns < minVal {
			minVal = conns
			minConn = s
		}
	}

	if minConn != nil {
		minConn.mu.Lock()
		minConn.ActiveConns++
		minConn.mu.Unlock()
	}

	return minConn
}

// IPHash IP 哈希负载均衡
type IPHash struct {
	servers  []*Server
	mu       sync.Mutex
}

func NewIPHash(servers []*Server) *IPHash {
	return &IPHash{
		servers: servers,
	}
}

func (i *IPHash) Next() *Server {
	return nil // TODO: 实现
}

// Server 上游服务器
type Server struct {
	URL         string
	Weight      int
	MaxFails    int
	FailTimeout time.Duration

	mu          sync.RWMutex
	ActiveConns int
	Fails       int
	LastFail    time.Time

	Down bool
}

// NewServer 创建上游服务器
func NewServer(url string, weight int) *Server {
	return &Server{
		URL:         url,
		Weight:      weight,
		MaxFails:    1,
		FailTimeout: 10 * time.Second,
	}
}

// MarkFailed 标记失败
func (s *Server) MarkFailed() {
	s.mu.Lock()
	s.Fails++
	s.LastFail = time.Now()
	s.mu.Unlock()
}

// IsAvailable 检查是否可用
func (s *Server) IsAvailable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.Down {
		return false
	}
	
	if s.MaxFails > 0 && s.Fails >= s.MaxFails {
		if time.Since(s.LastFail) < s.FailTimeout {
			return false
		}
		// 重置失败计数
		s.Fails = 0
	}
	
	return true
}