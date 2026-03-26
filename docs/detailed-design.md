# NginxGo 详细设计文档

## 1. 项目结构设计

### 1.1 目录结构

```
nginxgo/
├── cmd/
│   └── nginxgo/
│       └── main.go           # 程序入口
├── pkg/
│   ├── core/                 # 核心模块
│   │   ├── cycle.go         # 运行周期
│   │   ├── process.go      # 进程管理
│   │   ├── signal.go       # 信号处理
│   │   └── module.go       # 模块系统
│   ├── config/              # 配置系统
│   │   ├── parser.go       # 配置解析器
│   │   ├── scanner.go     # 词法扫描
│   │   └── directive.go   # 指令定义
│   ├── event/              # 事件系统
│   │   ├── loop.go         # 事件循环
│   │   ├── connection.go   # 连接管理
│   │   ├── epoll.go        # epoll 实现
│   │   └── timer.go        # 定时器
│   ├── http/               # HTTP 模块
│   │   ├── server.go       # HTTP 服务器
│   │   ├── request.go      # 请求结构
│   │   ├── response.go     # 响应结构
│   │   ├── router.go       # 路由
│   │   ├── handler/        # 处理器
│   │   │   ├── static.go   # 静态文件
│   │   │   ├── proxy.go    # 反向代理
│   │   │   └── fastcgi.go  # FastCGI
│   │   └── filter/         # 过滤器
│   │       ├── gzip.go     # Gzip压缩
│   │       └── header.go   # 头处理
│   ├── upstream/           # 上游模块
│   │   ├── pool.go         # 连接池
│   │   ├── lb.go           # 负载均衡
│   │   └── health.go       # 健康检查
│   ├── log/                # 日志系统
│   │   └── logger.go       # 日志器
│   └── types/              # 类型定义
│       └── mime.go         # MIME类型
├── conf/
│   └── nginx.conf          # 默认配置
├── go.mod
└── go.sum
```

### 1.2 Go Module 配置

```go
// go.mod
module github.com/nginxgo/nginxgo

go 1.21
```

---

## 2. 核心模块详细设计

### 2.1 Core 模块

#### 2.1.1 Cycle (运行周期)

```go
// pkg/core/cycle.go

package core

import (
    "sync"
    "github.com/nginxgo/nginxgo/pkg/config"
    "github.com/nginxgo/nginxgo/pkg/log"
)

// Cycle 表示服务器运行周期
type Cycle struct {
    Config        *config.Config     // 配置
    Modules       []Module            // 已注册模块
    OldCycle      *Cycle              // 优雅重载时的旧周期
    Logger        *log.Logger         // 日志实例
    MemoryPool    sync.Pool           // 内存池
    
    mu            sync.RWMutex
    state         CycleState
}

// CycleState 运行周期状态
type CycleState int

const (
    CycleStateInit CycleState = iota
    CycleStateRunning
    CycleStateRunningWithOld
    CycleStateExiting
)

// NewCycle 创建新的运行周期
func NewCycle(cfg *config.Config) *Cycle {
    return &Cycle{
        Config:  cfg,
        Modules: make([]Module, 0),
        Logger:  log.NewLogger(cfg.Global.ErrorLog),
        state:   CycleStateInit,
    }
}

// Init 初始化周期
func (c *Cycle) Init() error {
    // 初始化模块
    for _, m := range c.Modules {
        if err := m.Init(c); err != nil {
            return err
        }
    }
    c.state = CycleStateRunning
    return nil
}

// NewCycleForReload 创建用于热重载的新周期
func (c *Cycle) NewCycleForReload(cfg *config.Config) *Cycle {
    old := *c
    old.state = CycleStateRunningWithOld
    
    newCycle := NewCycle(cfg)
    newCycle.OldCycle = &old
    return newCycle
}
```

#### 2.1.2 Process (进程管理)

```go
// pkg/core/process.go

package core

import (
    "os"
    "syscall"
    "time"
    "github.com/nginxgo/nginxgo/pkg/config"
)

const (
    ProcessMaster = iota
    ProcessWorker
    ProcessSingle
)

// ProcessInfo 进程信息
type ProcessInfo struct {
    PID     int
    Type    int
    Channel chan interface{}
}

// Master 主进程
type Master struct {
    cycle      *Cycle
    workers    []*Worker
    signalChan chan os.Signal
    done       chan struct{}
}

// Worker Worker 进程
type Worker struct {
    pid       int
    cycle     *Cycle
    eventLoop *EventLoop
}

// StartMaster 启动主进程
func (m *Master) Start() error {
    // 创建信号通道
    m.signalChan = make(chan os.Signal, 1)
    
    // 启动 Worker 进程
    workerCount := m.cycle.Config.Global.WorkerProcesses
    if workerCount <= 0 {
        workerCount = runtime.NumCPU()
    }
    
    for i := 0; i < workerCount; i++ {
        w := &Worker{cycle: m.cycle}
        if err := w.Start(); err != nil {
            return err
        }
        m.workers = append(m.workers, w)
    }
    
    // 进入信号处理循环
    return m.signalLoop()
}

// signalLoop 信号处理循环
func (m *Master) signalLoop() {
    for {
        select {
        case sig := <-m.signalChan:
            switch sig {
            case syscall.SIGTERM, syscall.SIGINT:
                m.Quit()
            case syscall.SIGHUP:
                m.Reload()
            case syscall.SIGUSR1:
                m.ReopenLog()
            }
        case <-m.done:
            return
        }
    }
}

// Quit 退出
func (m *Master) Quit() {
    for _, w := range m.workers {
        w.Quit()
    }
    m.done <- struct{}{}
}

// Reload 热重载
func (m *Master) Reload() {
    // 重新加载配置，创建新周期
    // 通知所有 Worker
}

// ReopenLog 重新打开日志
func (m *Master) ReopenLog() {
    for _, w := range m.workers {
        w.ReopenLog()
    }
}
```

#### 2.1.3 信号处理

```go
// pkg/core/signal.go

package core

import (
    "os"
    "syscall"
)

var (
    // 允许处理的信号
    handledSignals = []os.Signal{
        syscall.SIGTERM,
        syscall.SIGINT,
        syscall.SIGQUIT,
        syscall.SIGHUP,
        syscall.SIGUSR1,
        syscall.SIGUSR2,
        syscall.SIGWINCH,
    }
)

// SignalHandler 信号处理函数
func (m *Master) SetupSignal() {
    signal.Notify(m.signalChan, handledSignals...)
}

// SignalToString 信号转字符串
func SignalToString(s os.Signal) string {
    switch s {
    case syscall.SIGTERM:
        return "SIGTERM"
    case syscall.SIGINT:
        return "SIGINT"
    case syscall.SIGHUP:
        return "SIGHUP"
    case syscall.SIGQUIT:
        return "SIGQUIT"
    case syscall.SIGUSR1:
        return "SIGUSR1"
    case syscall.SIGUSR2:
        return "SIGUSR2"
    case syscall.SIGWINCH:
        return "SIGWINCH"
    default:
        return "UNKNOWN"
    }
}
```

#### 2.1.4 模块系统

```go
// pkg/core/module.go

package core

// ModuleType 模块类型
type ModuleType int

const (
    TypeCore ModuleType = iota
    TypeEvent
    TypeHTTP
    TypeUpstream
    TypeStream
    TypeMail
    TypeFilter
)

// Module 模块接口
type Module interface {
    Name() string
    Type() ModuleType
    Init(cycle *Cycle) error
    PostInit(cycle *Cycle) error
}

// BaseModule 基础模块实现
type BaseModule struct {
    name string
    moduleType ModuleType
}

func (m *BaseModule) Name() string {
    return m.name
}

func (m *BaseModule) Type() ModuleType {
    return m.moduleType
}

// modules 全局模块注册表
var modules = make(map[string]Module)

// RegisterModule 注册模块
func RegisterModule(m Module) {
    modules[m.Name()] = m
}

// GetModule 获取模块
func GetModule(name string) Module {
    return modules[name]
}

// InitModules 初始化所有模块
func (c *Cycle) InitModules() error {
    for _, m := range modules {
        c.Modules = append(c.Modules, m)
        if err := m.Init(c); err != nil {
            return err
        }
    }
    
    // 后初始化
    for _, m := range c.Modules {
        if err := m.PostInit(c); err != nil {
            return err
        }
    }
    return nil
}
```

---

### 2.2 Config 模块

#### 2.2.1 配置结构

```go
// pkg/config/config.go

package config

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

// ServerConfig 服务器配置
type ServerConfig struct {
    Listen       []string
    ServerName   []string
    Root         string
    Index        []string
    Locations    []*LocationConfig
}

// LocationConfig Location 配置
type LocationConfig {
    Path       string
    Modifier   string  // =, ^~, ~
    ProxyPass  string
    Root       string
    Index      []string
    Handlers   []HTTPHandler
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
```

#### 2.2.2 词法扫描

```go
// pkg/config/scanner.go

package config

import (
    "unicode"
    "bytes"
)

// TokenType Token 类型
type TokenType int

const (
    TokenEOF TokenType = iota
    TokenLeftBrace     // {
    TokenRightBrace    // }
    TokenSemicolon     // ;
    TokenIdent         // 标识符
    TokenString        // 字符串
    TokenVariable      // 变量 $xxx
)

// Token Token 结构
type Token struct {
    Type    TokenType
    Value   string
    Line    int
    Column  int
}

// Scanner 词法扫描器
type Scanner struct {
    input  []byte
    pos    int
    line   int
    column int
}

// NewScanner 创建扫描器
func NewScanner(input []byte) *Scanner {
    return &Scanner{
        input:  input,
        line:   1,
        column: 1,
    }
}

// Next 获取下一个 Token
func (s *Scanner) Next() Token {
    s.skipWhitespace()
    
    if s.pos >= len(s.input) {
        return Token{Type: TokenEOF, Line: s.line, Column: s.column}
    }
    
    ch := s.input[s.pos]
    
    switch {
    case ch == '{':
        s.advance()
        return s.makeToken(TokenLeftBrace, "{")
    case ch == '}':
        s.advance()
        return s.makeToken(TokenRightBrace, "}")
    case ch == ';':
        s.advance()
        return s.makeToken(TokenSemicolon, ";")
    case ch == '"':
        return s.scanString()
    case ch == '$':
        return s.scanVariable()
    case unicode.IsLetter(ch) || ch == '_':
        return s.scanIdent()
    default:
        s.advance()
        return s.makeToken(TokenIdent, string(ch))
    }
}

func (s *Scanner) skipWhitespace() {
    for s.pos < len(s.input) {
        ch := s.input[s.pos]
        if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
            if ch == '\n' {
                s.line++
                s.column = 1
            }
            s.pos++
        } else if ch == '#' {
            s.skipComment()
        } else {
            break
        }
    }
}

func (s *Scanner) scanIdent() string {
    start := s.pos
    for s.pos < len(s.input) {
        ch := s.input[s.pos]
        if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-' {
            s.pos++
        } else {
            break
        }
    }
    return string(s.input[start:s.pos])
}
```

#### 2.2.3 配置解析

```go
// pkg/config/parser.go

package config

// Parser 解析器
type Parser struct {
    scanner *Scanner
    tokens  []Token
    pos     int
}

// NewParser 创建解析器
func NewParser(content []byte) *Parser {
    scanner := NewScanner(content)
    tokens := make([]Token, 0)
    
    for {
        tok := scanner.Next()
        tokens = append(tokens, tok)
        if tok.Type == TokenEOF {
            break
        }
    }
    
    return &Parser{
        scanner: scanner,
        tokens:   tokens,
        pos:      0,
    }
}

// Parse 解析配置
func (p *Parser) Parse() (*Config, error) {
    cfg := &Config{
        Global: GlobalConfig{
            WorkerProcesses: 1,
            ErrorLog:        "logs/error.log",
            PIDFile:         "logs/nginx.pid",
        },
        Events: EventsConfig{
            WorkerConnections: 512,
            MultiAccept:      true,
        },
        HTTP: HTTPConfig{
            Servers:   make([]*ServerConfig, 0),
            Upstreams: make([]*UpstreamConfig, 0),
            MIMETypes: DefaultMIMETypes,
        },
    }
    
    // 解析配置块
    for p.pos < len(p.tokens) {
        tok := p.peek()
        
        if tok.Type == TokenEOF {
            break
        }
        
        if tok.Type != TokenIdent {
            p.error("unexpected token: %v", tok)
            continue
        }
        
        // 解析 directive
        if err := p.parseDirective(cfg); err != nil {
            return nil, err
        }
    }
    
    return cfg, nil
}

func (p *Parser) parseDirective(cfg *Config) error {
    name := p.next().Value
    
    switch name {
    case "events":
        return p.parseEventsBlock(&cfg.Events)
    case "http":
        return p.parseHTTPBlock(&cfg.HTTP)
    case "worker_processes":
        return p.parseWorkerProcesses(&cfg.Global)
    case "error_log":
        return p.parseErrorLog(&cfg.Global)
    default:
        return p.skipBlock()
    }
}

func (p *Parser) parseEventsBlock(cfg *EventsConfig) error {
    p.expect(TokenLeftBrace)
    
    for {
        tok := p.peek()
        if tok.Type == TokenRightBrace {
            p.next()
            break
        }
        
        name := p.next().Value
        switch name {
        case "worker_connections":
            val := p.next()
            cfg.WorkerConnections = p.parseInt(val.Value)
        case "multi_accept":
            val := p.next()
            cfg.MultiAccept = val.Value == "on"
        }
        p.expect(TokenSemicolon)
    }
    
    return nil
}

func (p *Parser) parseHTTPBlock(cfg *HTTPConfig) error {
    p.expect(TokenLeftBrace)
    
    for {
        tok := p.peek()
        if tok.Type == TokenRightBrace {
            p.next()
            break
        }
        
        name := p.next().Value
        switch name {
        case "server":
            server, err := p.parseServerBlock()
            if err != nil {
                return err
            }
            cfg.Servers = append(cfg.Servers, server)
        case "upstream":
            upstream, err := p.parseUpstreamBlock()
            if err != nil {
                return err
            }
            cfg.Upstreams = append(cfg.Upstreams, upstream)
        default:
            p.skipBlock()
        }
    }
    
    return nil
}

func (p *Parser) next() Token {
    t := p.tokens[p.pos]
    p.pos++
    return t
}

func (p *Parser) peek() Token {
    return p.tokens[p.pos]
}

func (p *Parser) expect(t TokenType) {
    tok := p.next()
    if tok.Type != t {
        p.error("expected %v but got %v", t, tok.Type)
    }
}

func (p *Parser) error(format string, args ...interface{}) {
    // 实现错误处理
}
```

---

### 2.3 Event 模块

#### 2.3.1 事件循环

```go
// pkg/event/loop.go

package event

import (
    "runtime"
    "syscall"
    "time"
)

// EventLoop 事件循环
type EventLoop struct {
    epoll   *Epoll
    timer   *Timer
    conns   map[int]*Connection
    events  chan *Event
    quit    chan struct{}
}

// NewEventLoop 创建事件循环
func NewEventLoop() *EventLoop {
    ep, err := NewEpoll()
    if err != nil {
        panic(err)
    }
    
    return &EventLoop{
        epoll:   ep,
        timer:   NewTimer(),
        conns:   make(map[int]*Connection, 1024),
        events:  make(chan *Event, 1024),
        quit:    make(chan struct{}),
    }
}

// Start 启动事件循环
func (el *EventLoop) Start(acceptFn func() (*Connection, error)) error {
    for {
        // 等待事件
        nfds, err := el.epoll.Wait()
        if err != nil {
            if err == syscall.EINTR {
                continue
            }
            return err
        }
        
        // 处理事件
        for i := 0; i < nfds; i++ {
            fd := el.epoll.GetEvent(i).Fd
            
            if fd == el.epoll.ListenFD() {
                // 新连接
                conn, err := acceptFn()
                if err != nil {
                    continue
                }
                el.addConnection(conn)
            } else {
                // 处理现有连接
                conn, ok := el.conns[fd]
                if !ok {
                    continue
                }
                el.handleConnection(conn)
            }
        }
        
        // 处理定时器
        el.timer.Process()
    }
}

func (el *EventLoop) addConnection(conn *Connection) {
    el.conns[conn.FD] = conn
    
    // 注册读事件
    el.epoll.Add(conn.FD, conn.ReadEvent)
}

func (el *EventLoop) handleConnection(conn *Connection) {
    ev, err := el.epoll.WaitFor(conn.FD)
    if err != nil {
        el.removeConnection(conn)
        return
    }
    
    if ev&syscall.EPOLLIN != 0 {
        conn.ReadEvent.Handler()
    }
    
    if ev&syscall.EPOLLOUT != 0 {
        conn.WriteEvent.Handler()
    }
}

func (el *EventLoop) removeConnection(conn *Connection) {
    el.epoll.Del(conn.FD)
    delete(el.conns, conn.FD)
    conn.Close()
}
```

#### 2.3.2 Epoll 实现

```go
// pkg/event/epoll.go

package event

import (
    "syscall"
    "unsafe"
)

const (
    EPOLLIN  = 0x001
    EPOLLOUT = 0x004
    EPOLLET  = 0x80000000
)

// Epoll epoll 实现
type Epoll struct {
    fd     int
    events []syscall.EpollEvent
}

// NewEpoll 创建 epoll
func NewEpoll() (*Epoll, error) {
    fd, err := syscall.EpollCreate1(0)
    if err != nil {
        return nil, err
    }
    
    return &Epoll{
        fd:     fd,
        events: make([]syscall.EpollEvent, 128),
    }, nil
}

// Add 添加文件描述符到 epoll
func (e *Epoll) Add(fd int, event *Event) error {
    var ev syscall.EpollEvent
    ev.Fd = int32(fd)
    ev.Events = EPOLLIN | EPOLLET
    
    if event.WriteEnabled {
        ev.Events |= EPOLLOUT
    }
    
    return syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &ev)
}

// Del 从 epoll 删除
func (e *Epoll) Del(fd int) error {
    return syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
}

// Mod 修改事件
func (e *Epoll) Mod(fd int, event *Event) error {
    var ev syscall.EpollEvent
    ev.Fd = int32(fd)
    ev.Events = EPOLLIN | EPOLLET
    
    if event.WriteEnabled {
        ev.Events |= EPOLLOUT
    }
    
    return syscall.EpollCtl(e.fd, syscall.EPOLL_CTL_MOD, fd, &ev)
}

// Wait 等待事件
func (e *Epoll) Wait() (int, error) {
    n, err := syscall.EpollWait(e.fd, e.events, -1)
    if err != nil {
        return 0, err
    }
    return n, nil
}

// WaitFor 等待特定 fd 的事件
func (e *Epoll) WaitFor(fd int) (uint32, error) {
    evs := make([]syscall.EpollEvent, 1)
    n, err := syscall.EpollWait(e.fd, evs, 0)
    if err != nil {
        return 0, err
    }
    if n == 0 {
        return 0, nil
    }
    return evs[0].Events, nil
}

// ListenFD 获取监听 fd
func (e *Epoll) ListenFD() int {
    return -1 // 需要在初始化时设置
}
```

#### 2.3.3 连接管理

```go
// pkg/event/connection.go

package event

import (
    "net"
    "time"
)

// Connection 连接
type Connection struct {
    FD       int
    Conn     net.Conn
    ReadEvent  *Event
    WriteEvent *Event
    
    RemoteAddr string
    LocalAddr  string
    
    Server     interface{}  // 所属服务器
    Request    interface{}  // 当前请求
    
    CreatedAt  time.Time
    LastActive time.Time
}

// NewConnection 创建连接
func NewConnection(fd int, conn net.Conn) *Connection {
    return &Connection{
        FD:        fd,
        Conn:      conn,
        ReadEvent:  &Event{Handler: nil},
        WriteEvent: &Event{Handler: nil, WriteEnabled: true},
        
        RemoteAddr: conn.RemoteAddr().String(),
        LocalAddr:  conn.LocalAddr().String(),
        
        CreatedAt:  time.Now(),
        LastActive: time.Now(),
    }
}

// Close 关闭连接
func (c *Connection) Close() error {
    if c.Conn != nil {
        return c.Conn.Close()
    }
    return nil
}

// Read 读取数据
func (c *Connection) Read(p []byte) (int, error) {
    return c.Conn.Read(p)
}

// Write 写入数据
func (c *Connection) Write(p []byte) (int, error) {
    return c.Conn.Write(p)
}
```

#### 2.3.4 定时器

```go
// pkg/event/timer.go

package event

import (
    "container/heap"
    "time"
)

// Timer 定时器
type Timer struct {
    tasks *priorityQueue
}

// TimerTask 定时任务
type TimerTask struct {
    ID        int
    ExpiresAt time.Time
    Handler   func()
    Interval  time.Duration  // 0 表示不重复
    index     int
}

// priorityQueue 优先级队列
type priorityQueue []*TimerTask

func (pq priorityQueue) Len() int {
    return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
    return pq[i].ExpiresAt.Before(pq[j].ExpiresAt)
}

func (pq priorityQueue) Swap(i, j int) {
    pq[i], pq[j] = pq[j], pq[i]
    pq[i].index = i
    pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
    task := x.(*TimerTask)
    task.index = len(*pq)
    *pq = append(*pq, task)
}

func (pq *priorityQueue) Pop() interface{} {
    old := *pq
    n := len(old)
    task := old[n-1]
    *pq = old[0 : n-1]
    return task
}

// NewTimer 创建定时器
func NewTimer() *Timer {
    pq := make(priorityQueue, 0)
    heap.Init(&pq)
    
    return &Timer{
        tasks: &pq,
    }
}

// Add 添加定时任务
func (t *Timer) Add(task *TimerTask) {
    heap.Push(t.tasks, task)
}

// Process 处理到期任务
func (t *Timer) Process() {
    now := time.Now()
    
    for t.tasks.Len() > 0 {
        task := (*t.tasks)[0]
        
        if now.Before(task.ExpiresAt) {
            break
        }
        
        // 执行任务
        heap.Pop(t.tasks)
        go task.Handler()
        
        // 如果是重复任务，重新添加
        if task.Interval > 0 {
            task.ExpiresAt = now.Add(task.Interval)
            heap.Push(t.tasks, task)
        }
    }
}
```

---

### 2.4 HTTP 模块

#### 2.4.1 请求结构

```go
// pkg/http/request.go

package http

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
    Form       map[string]string
    Cookies    map[string]string
}

// NewRequest 创建请求
func NewRequest() *Request {
    return &Request{
        Headers: make(map[string]string),
        Form:    make(map[string]string),
        Cookies: make(map[string]string),
    }
}

// ParseRequestLine 解析请求行
func (r *Request) ParseRequestLine(line string) error {
    // GET /path HTTP/1.1
    parts := split(line, " ")
    if len(parts) != 3 {
        return ErrBadRequest
    }
    
    r.Method = parts[0]
    r.URI = parts[1]
    r.Proto = parts[2]
    
    // 解析 URL
    if idx := find(parts[1], "?"); idx != -1 {
        r.Path = parts[1][:idx]
        r.Query = parts[1][idx+1:]
    } else {
        r.Path = parts[1]
    }
    
    return nil
}

// ParseHeaders 解析请求头
func (r *Request) ParseHeaders(lines []string) error {
    for _, line := range lines {
        idx := find(line, ":")
        if idx == -1 {
            continue
        }
        
        key := trim(line[:idx])
        value := trim(line[idx+1:])
        r.Headers[key] = value
    }
    
    // 提取常用字段
    r.Host = r.Headers["Host"]
    return nil
}
```

#### 2.4.2 响应结构

```go
// pkg/http/response.go

package http

// Response HTTP 响应
type Response struct {
    StatusCode int            // 200, 404, etc.
    StatusText string         // OK, Not Found
    
    Headers map[string]string
    Body    []byte
    
    Request *Request
    conn    *Connection
    
    sentHeaders bool
}

// NewResponse 创建响应
func NewResponse(req *Request, conn *Connection) *Response {
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
    r.SetHeader("Content-Length", itoa(n))
}

// WriteHeader 写入响应头
func (r *Response) WriteHeader() error {
    if r.sentHeaders {
        return nil
    }
    
    // 构建状态行
    statusLine := sprintf("%s %d %s\r\n", r.Request.Proto, r.StatusCode, r.StatusText)
    
    // 构建头部
    var header bytes.Buffer
    header.WriteString(statusLine)
    
    for k, v := range r.Headers {
        header.WriteString(sprintf("%s: %s\r\n", k, v))
    }
    
    header.WriteString("\r\n")
    
    // 发送
    r.conn.Write(header.Bytes())
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
```

#### 2.4.3 服务器

```go
// pkg/http/server.go

package http

import (
    "net"
    "time"
)

// Server HTTP 服务器
type Server struct {
    Addr      string
    Port      int
    Name      string
    
    Handler   http.Handler  // 路由处理器
    Locations []*LocationConfig
    
    TLSConfig *tls.Config
}

// NewServer 创建服务器
func NewServer(addr string, port int) *Server {
    return &Server{
        Addr:    addr,
        Port:    port,
        Handler: NewRouter(),
        Locations: make([]*LocationConfig, 0),
    }
}

// Serve 开始服务
func (s *Server) Serve() error {
    addr := sprintf("%s:%d", s.Addr, s.Port)
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }
    
    for {
        conn, err := ln.Accept()
        if err != nil {
            if err == syscall.EINTR {
                continue
            }
            return err
        }
        
        go s.handleConnection(conn)
    }
}

func (s *Server) handleConnection(conn net.Conn) {
    defer conn.Close()
    
    c := NewConnection(int(conn.(*net.TCPConn).File().Fd()), conn)
    
    // 读取请求
    buf := make([]byte, 4096)
    n, err := c.Read(buf)
    if err != nil {
        return
    }
    
    // 解析请求
    req := NewRequest()
    lines := split(string(buf[:n]), "\r\n")
    if len(lines) == 0 {
        return
    }
    
    req.ParseRequestLine(lines[0])
    
    // 解析 header
    headerEnd := 0
    for i := 1; i < len(lines); i++ {
        if lines[i] == "" {
            headerEnd = i
            break
        }
    }
    if headerEnd > 1 {
        req.ParseHeaders(lines[1:headerEnd])
    }
    
    // 路由匹配
    handler := s.Handler.ServeHTTP(req)
    
    // 处理请求
    resp := NewResponse(req, c)
    handler.ServeHTTP(resp, req)
}
```

#### 2.4.4 路由

```go
// pkg/http/router.go

package http

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

// ServeHTTP 处理请求
func (r *Router) ServeHTTP(w ResponseWriter, req *Request) {
    method := req.Method
    path := req.Path
    
    tree := r.trees[method]
    if tree == nil {
        w.WriteHeader(404)
        return
    }
    
    handler := tree.getRoute(path)
    if handler == nil {
        w.WriteHeader(404)
        return
    }
    
    handler.ServeHTTP(w, req)
}

// node 路由树节点
type node struct {
    path     string
    handler  Handler
    children map[string]*node
}

func (n *node) addRoute(path string, handler Handler) {
    // 简化的路由实现
    n.path = path
    n.handler = handler
    if n.children == nil {
        n.children = make(map[string]*node)
    }
}

func (n *node) getRoute(path string) Handler {
    if n.path == path {
        return n.handler
    }
    return nil
}
```

---

### 2.5 Handler 实现

#### 2.5.1 静态文件处理器

```go
// pkg/http/handler/static.go

package handler

import (
    "os"
    "path"
    "mime"
)

// StaticHandler 静态文件处理器
type StaticHandler struct {
    Root     string      // 文档根目录
    Index    []string    // 默认首页
}

// NewStaticHandler 创建静态文件处理器
func NewStaticHandler(root string, index []string) *StaticHandler {
    return &StaticHandler{
        Root:  root,
        Index: index,
    }
}

// ServeHTTP 处理静态文件请求
func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 构建文件路径
    filePath := path.Join(h.Root, r.Path)
    
    // 检查路径安全（防止目录遍历）
    if !strings.HasPrefix(filePath, h.Root) {
        w.WriteHeader(403)
        return
    }
    
    // 获取文件信息
    stat, err := os.Stat(filePath)
    if err != nil {
        if os.IsNotExist(err) {
            w.WriteHeader(404)
        } else {
            w.WriteHeader(500)
        }
        return
    }
    
    // 如果是目录，查找默认首页
    if stat.IsDir() {
        for _, indexFile := range h.Index {
            indexPath := path.Join(filePath, indexFile)
            if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
                filePath = indexPath
                stat = info
                break
            }
        }
    }
    
    // 检查是否是文件
    if stat.IsDir() {
        w.WriteHeader(403)
        return
    }
    
    // 设置响应头
    w.SetContentType(mime.TypeByExtension(path.Ext(filePath)))
    w.SetContentLength(int(stat.Size()))
    
    // ETag
    etag := fmt.Sprintf(`"%x-%x"`, stat.Size(), stat.ModTime().Unix())
    w.SetHeader("ETag", etag)
    
    // Last-Modified
    w.SetHeader("Last-Modified", stat.ModTime().Format(http.TimeFormat))
    
    // 发送文件
    http.ServeFile(w, r, filePath)
}
```

#### 2.5.2 反向代理处理器

```go
// pkg/http/handler/proxy.go

package handler

import (
    "net/http"
    "net/url"
    "strings"
)

// ProxyHandler 反向代理处理器
type ProxyHandler struct {
    Upstream *upstream.Pool
}

// NewProxyHandler 创建反向代理处理器
func NewProxyHandler(upstream *upstream.Pool) *ProxyHandler {
    return &ProxyHandler{
        Upstream: upstream,
    }
}

// ServeHTTP 处理代理请求
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 获取上游服务器
    backend := h.Upstream.Get()
    if backend == nil {
        w.WriteHeader(502)
        w.Write([]byte("Bad Gateway"))
        return
    }
    defer backend.Release()
    
    // 构建上游请求
    req, err := h.buildRequest(r, backend.URL)
    if err != nil {
        w.WriteHeader(500)
        return
    }
    
    // 发送请求
    resp, err := backend.Client.Do(req)
    if err != nil {
        w.WriteHeader(502)
        return
    }
    
    // 复制响应
    h.copyResponse(w, resp)
}

func (h *ProxyHandler) buildRequest(r *http.Request, backendURL string) (*http.Request, error) {
    // 解析后端 URL
    url, err := url.Parse(backendURL)
    if err != nil {
        return nil, err
    }
    
    // 构建新请求
    req, _ := http.NewRequest(r.Method, r.URI, r.Body)
    
    // 设置 headers
    for k, v := range r.Headers {
        if k == "Host" {
            req.Host = url.Host
        } else {
            req.Header.Set(k, v)
        }
    }
    
    // 添加代理头
    req.Header.Set("X-Real-IP", r.RemoteAddr)
    req.Header.Set("X-Forwarded-For", r.RemoteAddr)
    req.Header.Set("X-Forwarded-Proto", "http")
    
    return req, nil
}

func (h *ProxyHandler) copyResponse(w http.ResponseWriter, resp *http.Response) {
    // 复制状态码
    w.WriteHeader(resp.StatusCode)
    
    // 复制 headers
    for k, v := range resp.Header {
        for _, val := range v {
            w.SetHeader(k, val)
        }
    }
    
    // 复制 body
    io.Copy(w, resp.Body)
}
```

---

### 2.6 Upstream 模块

#### 2.6.1 负载均衡

```go
// pkg/upstream/lb.go

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

func (l *LeastConn) Next() *Server {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    var minConn *Server
    var minVal int = 1<<31 - 1
    
    for _, s := range l.servers {
        if s.ActiveConns < minVal {
            minVal = s.ActiveConns
            minConn = s
        }
    }
    
    if minConn != nil {
        minConn.ActiveConns++
    }
    
    return minConn
}

// IPHash IP 哈希
type IPHash struct {
    servers  []*Server
    mu       sync.Mutex
    hashFunc func(string) uint32
}

func NewIPHash(servers []*Server) *IPHash {
    return &IPHash{
        servers: servers,
        hashFunc: func(ip string) uint32 {
            h := uint32(0)
            for _, c := range ip {
                h = 31*h + uint32(c)
            }
            return h
        },
    }
}

func (i *IPHash) Next() *Server {
    // 需要实现
    return nil
}
```

#### 2.6.2 连接池

```go
// pkg/upstream/pool.go

package upstream

import (
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
    
    Client      *http.Client
}

// NewPool 创建连接池
func NewPool(name string, servers []*Server, lbType LoadBalancerType) *Pool {
    lb := newLoadBalancer(lbType, servers)
    
    return &Pool{
        Name:    name,
        Servers: servers,
        LB:      lb,
        timeout: 10 * time.Second,
    }
}

func newLoadBalancer(t LoadBalancerType, servers []*Server) LoadBalancer {
    switch t {
    case LBRoundRobin:
        return NewRoundRobin(servers)
    case LBLeastConn:
        return &LeastConn{servers: servers}
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
        Request: p.newRequest(server),
    }
}

// Backend 后端连接
type Backend struct {
    Server  *Server
    Pool    *Pool
    Request *http.Request
    
    resp *http.Response
}

// Release 释放连接
func (b *Backend) Release() {
    b.Server.mu.Lock()
    b.Server.ActiveConns--
    b.Server.mu.Unlock()
}

// Do 执行请求
func (b *Backend) Do(req *http.Request) (*http.Response, error) {
    return b.Server.Client.Do(req)
}
```

---

### 2.7 日志系统

```go
// pkg/log/logger.go

package log

import (
    "os"
    "fmt"
    "time"
    "sync"
    "bytes"
)

// Level 日志级别
type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
    LevelFatal
)

// Logger 日志器
type Logger struct {
    level  Level
    output *os.File
    mu     sync.Mutex
}

// NewLogger 创建日志器
func NewLogger(path string) *Logger {
    output, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        output = os.Stderr
    }
    
    return &Logger{
        level:  LevelInfo,
        output: output,
    }
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
    l.level = level
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
    l.log(LevelDebug, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
    l.log(LevelInfo, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
    l.log(LevelWarn, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
    l.log(LevelError, format, args...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
    l.log(LevelFatal, format, args...)
    os.Exit(1)
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
    if level < l.level {
        return
    }
    
    l.mu.Lock()
    defer l.mu.Unlock()
    
    buf := new(bytes.Buffer)
    buf.WriteString(time.Now().Format("2006/01/02 15:04:05"))
    buf.WriteString(" [")
    buf.WriteString(levelToString(level))
    buf.WriteString("] ")
    buf.WriteString(fmt.Sprintf(format, args...))
    buf.WriteString("\n")
    
    l.output.Write(buf.Bytes())
}

func levelToString(level Level) string {
    switch level {
    case LevelDebug:
        return "DEBUG"
    case LevelInfo:
        return "INFO"
    case LevelWarn:
        return "WARN"
    case LevelError:
        return "ERROR"
    case LevelFatal:
        return "FATAL"
    default:
        return "UNKNOWN"
    }
}
```

---

## 3. 启动流程

```
┌─────────────────────────────────────────────────────────────┐
│                       启动流程                                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. main()                                                  │
│     │                                                        │
│     ▼                                                        │
│  2. ParseFlags() - 解析命令行参数                            │
│     │                                                        │
│     ▼                                                        │
│  3. LoadConfig() - 加载配置文件                             │
│     │                                                        │
│     ▼                                                        │
│  4. NewMaster() - 创建主进程                                │
│     │                                                        │
│     ▼                                                        │
│  5. master.Start()                                          │
│     │                                                        │
│     ▼                                                        │
│  6. cycle.Init() - 初始化周期                               │
│     │                                                        │
│     ▼                                                        │
│  7. cycle.InitModules() - 初始化模块                        │
│     │                                                        │
│     ▼                                                        │
│  8. master.spawnWorkers() - 启动 Worker 进程               │
│     │                                                        │
│     ▼                                                        │
│  9. worker.Start() - Worker 进入事件循环                    │
│     │                                                        │
│     ▼                                                        │
│ 10. eventLoop.Start(accept) - 事件循环启动                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 4. 请求处理流程

```
┌─────────────────────────────────────────────────────────────┐
│                     请求处理流程                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Client ──▶ TCP Connect                                    │
│               │                                             │
│               ▼                                             │
│          Accept()                                           │
│               │                                             │
│               ▼                                             │
│          Read() - 读取 HTTP 请求                            │
│               │                                             │
│               ▼                                             │
│          Parse Request                                      │
│          - 请求行解析                                        │
│          - Header 解析                                       │
│          - Body 解析(如有)                                   │
│               │                                             │
│               ▼                                             │
│          Router.Match() - 路由匹配                          │
│               │                                             │
│               ▼                                             │
│          Handler.ServeHTTP()                               │
│          - 静态文件处理                                      │
│          - 反向代理                                          │
│          - FastCGI                                          │
│               │                                             │
│               ▼                                             │
│          Filters - 过滤器链                                 │
│          - Gzip 压缩                                        │
│          - 添加响应头                                        │
│               │                                             │
│               ▼                                             │
│          Write Response                                      │
│               │                                             │
│               ▼                                             │
│          Close (或 Keep-Alive)                              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. 配置热重载流程

```
┌─────────────────────────────────────────────────────────────┐
│                      热重载流程                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  SIGHUP 信号                                               │
│      │                                                      │
│      ▼                                                      │
│  Master 接收信号                                            │
│      │                                                      │
│      ▼                                                      │
│  解析新配置文件                                             │
│      │                                                      │
│      ▼                                                      │
│  创建新 Cycle                                              │
│      │                                                      │
│      ▼                                                      │
│  初始化新模块                                              │
│      │                                                      │
│      ▼                                                      │
│  通知所有 Worker (通过 channel)                            │
│      │                                                      │
│      ▼                                                      │
│  Worker:                                                    │
│    - 处理完当前请求                                         │
│    - 切换到新 Cycle                                         │
│    - 关闭旧监听（已由新 Cycle 接管）                        │
│                                                             │
│  Master:                                                    │
│    - 更新 PID 文件                                          │
│    - 清理旧 Cycle (延迟)                                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 6. 错误处理策略

| 错误类型 | 处理方式 |
|---------|---------|
| 配置错误 | 启动失败，输出错误信息 |
| 监听失败 | 输出错误日志，继续尝试 |
| 请求解析错误 | 返回 400 Bad Request |
| 文件不存在 | 返回 404 Not Found |
| 上游错误 | 返回 502 Bad Gateway |
| 内部错误 | 返回 500 Internal Server Error |

---

## 7. 性能优化要点

1. **连接池预分配**: 启动时预分配连接对象
2. **内存池**: 使用 sync.Pool 减少 GC
3. **零拷贝**: 尽可能使用 sendfile
4. **Epoll ET 模式**: 边缘触发，减少唤醒次数
5. **减少锁竞争**: 使用原子操作