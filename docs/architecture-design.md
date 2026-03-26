# Nginx-Go 系统架构设计文档

## 1. 项目概述

### 1.1 项目背景

本项目旨在使用 Go 语言重新实现 Nginx 的核心功能，创建一个高性能的事件驱动型 HTTP 服务器和反向代理服务器。项目代号：NginxGo。

### 1.2 设计目标

- **高性能**: 事件驱动架构，支持数万并发连接
- **低资源消耗**: 优化的内存管理，高效的 CPU 利用率
- **模块化设计**: 插件式架构，便于功能扩展
- **配置兼容**: 保持与 Nginx 配置语法的高度兼容
- **热重载**: 支持零停机配置更新

---

## 2. 整体架构

### 2.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         NginxGo 系统架构                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────────────────────────────┐  │
│  │   Master    │    │            Worker 进程               │  │
│  │   Process   │    │  ┌─────────────────────────────────┐  │  │
│  │             │    │  │       Event Loop (epoll)        │  │  │
│  │ - 配置解析   │    │  │  ┌─────────┐  ┌─────────┐      │  │  │
│  │ - 进程管理   │    │  │  │ Conn 1  │  │ Conn N  │      │  │  │
│  │ - 信号处理   │    │  │  └────┬────┘  └────┬────┘      │  │  │
│  │ - 日志管理   │    │  └──────┼────────────┼───────────┘  │  │
│  │ - 热重载     │    │         │            │              │  │
│  └──────┬──────┘    │    ┌─────┴────────────┴─────────┐    │  │
│         │           │    │     HTTP 处理管道          │    │  │
│         └───────────┼───▶│  路由 → 过滤 → 处理器      │    │  │
│                     │    └────────────────────────────┘    │  │
│                     └───────────────────────────────────────┘  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                         共享层                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│  │  配置    │ │  模块    │ │  连接池  │ │  内存池  │        │
│  │  管理   │ │  系统    │ │          │ │          │        │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 进程模型

| 进程类型 | 数量 | 职责 |
|---------|------|------|
| Master | 1 | 配置管理、进程监控、信号处理、热重载 |
| Worker | N (CPU核数) | 请求处理、事件循环、连接管理 |

---

## 3. 核心模块设计

### 3.1 模块结构图

```
┌─────────────────────────────────────────────────────────────┐
│                      模块系统架构                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  Core Module (核心)                   │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐          │   │
│  │  │ 配置解析  │ │ 进程管理  │ │ 信号处理  │          │   │
│  │  └───────────┘ └───────────┘ └───────────┘          │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                  │
│  ┌────────────────────────┼────────────────────────────────┐ │
│  │              Event Module (事件层)                      │ │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐              │ │
│  │  │ epoll/kqueue │ │ Timer    │ │ Connection │            │ │
│  │  └───────────┘ └───────────┘ └───────────┘              │ │
│  └────────────────────────────────────────────────────────┘ │
│                           │                                  │
│  ┌────────────────────────┼────────────────────────────────┐ │
│  │              HTTP Module (应用层)                       │ │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐              │ │
│  │  │  Router   │ │ Handlers  │ │  Filters  │              │ │
│  │  └───────────┘ └───────────┘ └───────────┘              │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Upstream Module (上游)                  │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐          │   │
│  │  │  负载均衡  │ │  健康检查 │ │  连接池   │          │   │
│  │  └───────────┘ └───────────┘ └───────────┘          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 核心模块说明

#### 3.2.1 Core 模块
- **配置解析器**: 支持 Nginx 风格配置语法
- **进程管理器**: Master/Worker 进程生命周期管理
- **信号处理器**: 处理系统信号（TERM, HUP, USR1 等）
- **日志系统**: 多级别日志输出

#### 3.2.2 Event 模块
- **事件循环**: 基于 epoll/kqueue 的事件驱动
- **连接管理**: 连接池管理，复用连接资源
- **定时器管理**: 支持高精度定时任务

#### 3.2.3 HTTP 模块
- **路由器**: URL 到处理器的映射
- **请求解析**: HTTP 头部、正文解析
- **处理器**: 静态文件、反向代理、FastCGI 等
- **过滤器**: Gzip 压缩、缓存控制等

#### 3.2.4 Upstream 模块
- **负载均衡**: 轮询、最少连接、IP Hash
- **健康检查**: 上游服务器状态检测
- **连接池**: 上游连接复用

---

## 4. 数据结构设计

### 4.1 核心数据结构

#### 4.1.1 Cycle (运行周期)
```go
type Cycle struct {
    ConfigContext []interface{}       // 配置上下文数组
    Modules       []Module            // 已注册模块
    Connections   []*Connection       // 连接池
    Listening     []*Listener         // 监听套接字
    Logger        *Logger             // 日志实例
    MemoryPool    *sync.Pool          // 内存池
}
```

#### 4.1.2 Connection (连接)
```go
type Connection struct {
    FD           int                   // 文件描述符
    RemoteAddr   string                // 客户端地址
    LocalAddr    string                // 服务器地址
    ReadEvent    *Event               // 读事件
    WriteEvent   *Event               // 写事件
    Request      *Request             // 当前请求
    Server       *Server              // 所属服务器
}
```

#### 4.1.3 Request (请求)
```go
type Request struct {
    Method     string                 // HTTP 方法
    URI        string                // 请求 URI
    Proto      string                // 协议版本
    Headers    map[string]string     // 请求头
    Body       []byte                // 请求体
    Connection  *Connection           // 关联连接
}
```

#### 4.1.4 Server (服务器)
```go
type Server struct {
    Name       string                 // 服务器名称
    Addresses  []string               // 监听地址
    Locations  []*Location            // Location 配置
    Upstreams  map[string]*Upstream   // 上游服务器组
}
```

### 4.2 事件结构

```go
type Event struct {
    Handler  func()                  // 回调处理函数
    Timer    *time.Timer             // 定时器
    Active   bool                    // 活跃状态
    Read     bool                    // 可读标志
    Write    bool                    // 可写标志
}
```

---

## 5. 接口设计

### 5.1 模块接口

```go
// 模块接口
type Module interface {
    Name() string                      // 模块名称
    Type() ModuleType                 // 模块类型
    Init(cycle *Cycle) error          // 初始化
    PostInit(cycle *Cycle) error      // 后初始化
}

// 模块类型
type ModuleType int
const (
    TypeCore ModuleType = iota
    TypeEvent
    TypeHTTP
    TypeUpstream
    TypeStream
    TypeMail
)
```

### 5.2 HTTP 处理器接口

```go
type HTTPHandler interface {
    ServeHTTP(w ResponseWriter, r *Request)
}

// 处理器注册
type HandlerRegistry struct {
    Handlers map[string]HTTPHandler
}
```

### 5.3 过滤器接口

```go
type HTTPFilter interface {
    Filter(w ResponseWriter, r *Request) error
}

// 过滤器链
type FilterChain struct {
    Filters []HTTPFilter
}
```

---

## 6. 配置系统设计

### 6.1 配置解析流程

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ 读取配置  │───▶│ 词法分析  │───▶│ 语法分析  │───▶│ 构建对象 │
│  文件   │    │ (Tokenize)│   │ (Parse)  │    │ (Build) │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
```

### 6.2 配置结构

```go
type Config struct {
    Global   GlobalConfig              // 全局配置
    Events   EventConfig              // 事件配置
    HTTP     HTTPConfig               // HTTP 配置
    Stream   StreamConfig             // 流配置
}

type GlobalConfig struct {
    WorkerProcesses    int             // Worker 进程数
    WorkerCPUAffinity  bool            // CPU 亲和
    ErrorLog          string          // 错误日志路径
    PIDFile           string          // PID 文件路径
}

type HTTPConfig struct {
    Servers   []*ServerConfig         // 服务器配置
    Upstreams []*UpstreamConfig       // 上游配置
}
```

---

## 7. 事件处理设计

### 7.1 事件循环流程

```
开始
  │
  ▼
┌─────────────────┐
│  epoll_wait    │◀─────────────┐
│  (阻塞等待)     │              │
└────────┬────────┘              │
         │ 有事件                │
         ▼                      │
    ┌────────────┐              │
    │ 处理就绪事件 │              │
    └─────┬──────┘              │
          │                     │
          ▼                     │
    ┌────────────┐              │
    │  分发到     │──────────────┤
    │  处理器    │
    └────────────┘
          │
          ▼
    ┌────────────┐
    │ 处理完成   │
    │ 返回 epoll │
    └────────────┘
```

### 7.2 连接生命周期

```
Accept (接受连接) → Register (注册事件) → 
Read (读取请求) → Process (处理请求) → 
Write (发送响应) → Close (关闭连接)
```

---

## 8. 通信机制

### 8.1 IPC 机制

| 机制 | 用途 |
|------|------|
| Unix Socket | Master-Worker 通信 |
| Channel | Go routine 间通信 |
| Shared Memory | 配置共享 |

### 8.2 信号处理

| 信号 | 动作 |
|------|------|
| SIGTERM/SIGINT | 快速停止服务 |
| SIGQUIT | 优雅停止服务 |
| SIGHUP | 重新加载配置 |
| SIGUSR1 | 重新打开日志 |
| SIGUSR2 | 平滑升级 |

---

## 9. 扩展性设计

### 9.1 模块加载机制

```go
// 模块注册
var modules = make(map[string]Module)

func RegisterModule(m Module) {
    modules[m.Name()] = m
}
```

### 9.2 Handler 注册

```go
// 内置处理器
var builtInHandlers = map[string]func() HTTPHandler{
    "static":   NewStaticHandler,
    "proxy":    NewProxyHandler,
    "fastcgi":  NewFastCGIHandler,
    "uwsgi":    NewUWSGIHandler,
}
```

---

## 10. 性能优化策略

### 10.1 内存优化

- 连接池预分配，避免运行时分配
- 对象池复用，减少 GC 压力
- 减少内存碎片

### 10.2 CPU 优化

- 多核亲和性
- 无锁数据结构
- 高效的事件分发

### 10.3 网络优化

- TCP 零拷贝
- TCP_NOPUSH/TCP_NODELAY
- 连接复用

---

## 11. 版本规划

### Phase 1: 核心框架
- Master/Worker 进程模型
- 事件循环 (epoll)
- 基础 HTTP 处理

### Phase 2: 核心功能
- 静态文件服务
- 反向代理
- 负载均衡

### Phase 3: 高级功能
- SSL/TLS 支持
- Gzip 压缩
- 缓存机制

### Phase 4: 优化增强
- 性能调优
- 监控指标
- 故障恢复

---

## 12. 兼容性设计

### 12.1 配置兼容

- 支持 Nginx 配置语法
- 支持常见 Nginx 指令
- 平滑迁移

### 12.2 API 兼容

- 兼容 Nginx 模块 API
- 支持第三方模块移植