# NginxGo

使用 Go 语言重构的高性能 Web 服务器，兼容 Nginx 配置。

## 特性

- ⚡ 高性能 Go 语言实现
- 🏗️ Master-Worker 架构
- 📁 静态文件服务 (支持 ETag, Last-Modified)
- 🌐 反向代理
- ⚖️ 负载均衡 (轮询、最少连接)
- 📦 Gzip 压缩
- 📝 Nginx 兼容配置

## 快速开始

### 运行服务器

```bash
# 编译
go build -o nginxgo ./cmd/nginxgo

# 运行
./nginxgo -c conf/nginx.conf

# 或者使用默认配置
./nginxgo
```

### 测试

```bash
# 访问首页
curl http://localhost:80/

# 访问静态文件
curl http://localhost:80/html/index.html
```

## 配置文件

### 基本配置 (`conf/nginx.conf`)

```nginx
# NginxGo 配置文件

# Worker 进程数
worker_processes 1;

# 错误日志
error_log logs/error.log;

# PID 文件
pid logs/nginx.pid;

# 事件配置
events {
    worker_connections 1024;
}

# HTTP 配置
http {
    # 默认 MIME 类型
    default_type application/octet-stream;

    # 发送文件
    sendfile on;
    
    # 连接超时
    keepalive_timeout 65;

    # Gzip 压缩
    gzip on;

    # 服务器配置
    server {
        listen 80;
        server_name localhost;

        # 根目录
        root html;
        index index.html;

        # 静态文件
        location / {
            root html;
            index index.html;
        }
        
        # 反向代理示例
        # location /api/ {
        #     proxy_pass http://localhost:8080;
        # }
    }
}
```

### 配置指令

| 指令 | 说明 | 示例 |
|------|------|------|
| `worker_processes` | Worker 进程数量 | `worker_processes 4;` |
| `error_log` | 错误日志路径 | `error_log logs/error.log;` |
| `listen` | 监听端口 | `listen 80;` |
| `server_name` | 服务器名称 | `server_name localhost;` |
| `root` | 文档根目录 | `root html;` |
| `index` | 默认首页 | `index index.html;` |
| `location` | URL 路由 | `location / {}` |
| `proxy_pass` | 反向代理目标 | `proxy_pass http://localhost:8080;` |
| `gzip` | 启用 Gzip 压缩 | `gzip on;` |

## 命令行参数

```bash
nginxgo [-c configfile] [-v]
```

- `-c` : 配置文件路径 (默认: `conf/nginx.conf`)
- `-v` : 显示版本信息

## 项目结构

```
nginxgo/
├── cmd/nginxgo/       # 程序入口
├── pkg/
│   ├── config/        # 配置解析
│   ├── core/          # 核心模块
│   ├── event/         # 事件循环
│   ├── http/          # HTTP 服务器
│   │   ├── handler/   # 处理器
│   │   └── filter/    # 过滤器
│   ├── upstream/      # 负载均衡
│   └── log/           # 日志系统
├── conf/              # 配置文件
├── html/              # 静态文件
└── go.mod
```

## 开发

### 运行测试

```bash
go test ./...
```

### 生成文档

```bash
# 查看 API 文档或使用 godoc
godoc -http=:6060
```

## 许可证

MIT License
