package config

import (
	"os"
	"io/ioutil"
)

// Config 全局配置
type Config struct {
	Global  GlobalConfig
	Events  EventsConfig
	HTTP    HTTPConfig
	Stream  StreamConfig
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	User             string
	WorkerProcesses  int
	WorkerCPUAffinity bool
	WorkerRlimitNofile int
	ErrorLog         string
	PIDFile          string
}

// EventsConfig 事件配置
type EventsConfig struct {
	WorkerConnections int
	Use               string
	MultiAccept       bool
}

// HTTPConfig HTTP 配置
type HTTPConfig struct {
	Servers    []*ServerConfig
	Upstreams  []*UpstreamConfig
	MIMETypes  map[string]string
}

// StreamConfig 流配置
type StreamConfig struct {
	Servers   []*StreamServerConfig
	Upstreams []*UpstreamConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Listen      []string
	ServerName  []string
	Root        string
	Index       []string
	Locations   []*LocationConfig
}

// LocationConfig Location 配置
type LocationConfig struct {
	Path       string
	Modifier   string // =, ^~, ~
	ProxyPass  string
	Root       string
	Index      []string
}

// StreamServerConfig 流服务器配置
type StreamServerConfig struct {
	Listen string
}

// UpstreamConfig 上游服务器配置
type UpstreamConfig struct {
	Name     string
	Servers  []*UpstreamServer
	LB       LoadBalancerType
}

// UpstreamServer 上游服务器
type UpstreamServer struct {
	Address  string
	Weight   int
	MaxFails int
	Backup   bool
}

// LoadBalancerType 负载均衡类型
type LoadBalancerType int

const (
	LBRoundRobin LoadBalancerType = iota
	LBLeastConn
	LBIPHash
)

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 使用默认配置
		return DefaultConfig(), nil
	}

	// 读取文件内容
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 解析配置
	parser := NewParser(content)
	return parser.Parse()
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			WorkerProcesses: 1,
			ErrorLog:        "logs/error.log",
			PIDFile:         "logs/nginx.pid",
		},
		Events: EventsConfig{
			WorkerConnections: 1024,
			MultiAccept:       true,
		},
		HTTP: HTTPConfig{
			Servers:   make([]*ServerConfig, 0),
			Upstreams: make([]*UpstreamConfig, 0),
			MIMETypes: DefaultMIMETypes,
		},
	}
}