# Nginx 技术文档

## 1. Nginx 概述

Nginx (engine x) 是一个高性能的 HTTP 和反向代理服务器，由俄罗斯工程师 Igor Sysoev 于2004年开发。最初旨在解决 C10K 问题（处理10000个并发连接），现已成为世界上最流行的 Web 服务器之一。

### 1.1 主要特性

- **高性能**: 事件驱动架构，支持数万并发连接
- **低资源消耗**: 内存占用低，CPU 使用效率高
- **模块化设计**: 丰富的模块生态系统
- **热重载**: 配置更新无需重启服务
- **反向代理**: 内置负载均衡和健康检查
- **静态文件服务**: 高效的静态内容处理
- **SSL/TLS 支持**: 完整的 HTTPS 支持

### 1.2 应用场景

1. Web 服务器
2. 反向代理服务器
3. 负载均衡器
4. 缓存服务器
5. API 网关

---

## 2. 架构设计

### 2.1 事件驱动架构

Nginx 采用事件驱动的非阻塞 I/O 模型，与传统线程/进程模型不同：

```
传统模型:                    事件驱动模型:
┌─────────────┐             ┌─────────────┐
│   Thread    │             │   Worker    │
├─────────────┤             ├─────────────┤
│ Request 1  │             │  Event Loop │
│ Request 2  │──────────▶  │   (epoll)   │
│ Request 3  │             │              │
│ ...        │             │  Request 1  │
│ Request N  │             │  Request 2  │
└─────────────┘             │  Request 3  │
   (阻塞等待)                │  ...        │
                            └─────────────┘
                            (非阻塞)
```

### 2.2 Master-Worker 进程模型

```
                    ┌──────────────┐
                    │   Master     │
                    │   Process    │
                    │              │
                    │ - 配置管理   │
                    │ - 进程监控   │
                    │ - 信号处理   │
                    └──────┬───────┘
                           │ fork
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
    │ Worker  │      │ Worker  │      │ Worker  │
    │   1     │      │   2     │      │   N     │
    │         │      │         │      │         │
    │ 处理请求 │      │ 处理请求 │      │ 处理请求 │
    │ 事件循环 │      │ 事件循环 │      │ 事件循环 │
    └─────────┘      └─────────┘      └─────────┘
```

#### Master 进程职责
- 读取和验证配置文件
- 创建/管理 Worker 进程
- 接收信号（reload, quit 等）
- 监控 Worker 状态
- 打开监听端口

#### Worker 进程职责
- 处理客户端请求
- 事件循环处理
- 与上游服务器通信
- 发送响应

### 2.3 核心数据结构

#### ngx_cycle_t
服务器运行周期，包含完整状态信息：
- `conf_ctx`: 配置上下文数组
- `pool`: 内存池
- `modules`: 已注册模块数组
- `connections`: 连接池
- `listening`: 监听套接字数组

#### ngx_connection_t
连接表示：
- `fd`: 文件描述符
- `read`: 读事件
- `write`: 写事件
- `remote_addr`: 客户端地址
- `server`: 所属服务器

#### ngx_event_t
事件结构：
- `handler`: 回调处理函数
- `timer`: 定时器
- `active`: 活跃状态标志

---

## 3. 模块系统

### 3.1 模块类型

| 模块类型 | 说明 | 示例 |
|---------|------|------|
| Core Module | 核心功能 | ngx_core_module |
| Event Module | 事件处理 | ngx_epoll_module, ngx_kqueue_module |
| HTTP Module | HTTP 协议 | ngx_http_module, ngx_http_static_module |
| Mail Module | 邮件代理 | ngx_mail_module |
| Stream Module | TCP/UDP 代理 | ngx_stream_module |
| Filter Module | 输出过滤 | ngx_http_gzip_filter |
| Upstream Module | 上游代理 | ngx_http_proxy_module |

### 3.2 模块结构

```c
typedef struct ngx_module_s {
    ngx_uint_t            version;
    void                 *ctx;
    ngx_command_t        *commands;
    ngx_uint_t            type;
    // ... 生命周期回调
} ngx_module_t;
```

### 3.3 核心模块列表

1. **ngx_core_module**: 基本服务器指令
2. **ngx_events_module**: 事件模块配置
3. **ngx_http_module**: HTTP 核心功能
4. **ngx_stream_module**: 流代理功能
5. **ngx_mail_module**: 邮件代理功能

---

## 4. HTTP 处理流程

```
请求流程:
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  接收   │───▶│  解析   │───▶│  处理   │───▶│  响应   │
│ Request │    │ Headers │    │ Handler │    │ Response│
└─────────┘    └─────────┘    └─────────┘    └─────────┘
                     │              │
                     │              ▼
               ┌─────┴─────┐   ┌─────────┐
               │  URL解析   │   │  过滤   │
               │  Route   │──▶│ Filters │
               └───────────┘   └─────────┘
```

### 4.1 HTTP 模块阶段

1. **NGX_HTTP_FIND_CONFIG_PHASE**: URL 匹配 location
2. **NGX_HTTP_REWRITE_PHASE**: URL 重写
3. **NGX_HTTP_ACCESS_PHASE**: 访问控制
4. **NGX_HTTP_CONTENT_PHASE**: 内容生成
5. **NGX_HTTP_LOG_PHASE**: 日志记录

---

## 5. 事件处理

### 5.1 多路复用机制

| 平台 | 实现 |
|------|------|
| Linux | epoll |
| FreeBSD/macOS | kqueue |
| Solaris | event ports |
| Windows | IOCP |
| 通用 | select/poll |

### 5.2 连接生命周期

```
创建连接 → 注册读事件 → 等待就绪 → 处理数据 → 关闭连接
    │                                         │
    └─────────── 循环复用 ─────────────────────┘
```

---

## 6. 配置系统

### 6.1 配置层次

```
main (全局)
  ├── events { }
  │
  ├── http {
  │    ├── server {
  │    │    ├── location / {
  │    │    │    └── 配置项
  │    │    }
  │    }
  │    └── upstream backend { }
  }
  │
  └── stream { }
```

### 6.2 配置指令类型

- **标记指令**: 无值 (如 `daemon on;`)
- **字符串指令**: 单值 (如 `worker_processes 4;`)
- **数组指令**: 多值 (如 `server_name example.com;`)
- **块指令**: 嵌套配置 (如 `location { }`)

---

## 7. 内存管理

### 7.1 内存池特性

- 批量分配，对象仅在池销毁时释放
- 快速分配，指针递增方式
- 自动清理，注册清理回调
- 避免内存碎片

### 7.2 主要内存池

- **cycle pool**: 周期级内存池
- **request pool**: 请求级内存池
- **temporary pool**: 临时内存池

---

## 8. 日志系统

### 8.1 日志级别

| 级别 | 值 | 说明 |
|------|----|------|
| EMERG | 1 | 紧急，系统不可用 |
| ALERT | 2 | 需要立即处理 |
| CRIT | 3 | 严重错误 |
| ERR | 4 | 错误条件 |
| WARN | 5 | 警告 |
| NOTICE | 6 | 通知 |
| INFO | 7 | 信息 |
| DEBUG | 8 | 调试 |

---

## 9. 常用功能配置

### 9.1 反向代理

```nginx
location / {
    proxy_pass http://backend;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

### 9.2 负载均衡

```nginx
upstream backend {
    server 192.168.1.1:8080;
    server 192.168.1.2:8080;
    least_conn;
}

location / {
    proxy_pass http://backend;
}
```

### 9.3 静态文件服务

```nginx
server {
    root /var/www/html;
    index index.html;
    
    location / {
        try_files $uri $uri/ =404;
    }
    
    location ~* \.(jpg|jpeg|png|gif)$ {
        expires 30d;
    }
}
```

---

## 10. 信号处理

| 信号 | 作用 |
|------|------|
| TERM, INT | 快速停止 |
| QUIT | 优雅停止 |
| HUP | 重新加载配置 |
| USR1 | 重新打开日志 |
| USR2 | 平滑升级 |
| WINCH | 优雅退出 Worker |

---

## 参考资料

- [Nginx 官方文档](https://nginx.org/en/docs/)
- [Nginx 官方博客](https://blog.nginx.org/)
- [Nginx 源码](https://github.com/nginx/nginx)