package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/nginxgo/nginxgo/pkg/config"
	"github.com/nginxgo/nginxgo/pkg/http"
	"github.com/nginxgo/nginxgo/pkg/http/handler"
	"github.com/nginxgo/nginxgo/pkg/log"
)

var (
	version    = "0.1.0"
	configFile = flag.String("c", "conf/nginx.conf", "Path to configuration file")
	showVersion = flag.Bool("v", false, "Show version")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("NginxGo version %s\n", version)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 创建日志器
	logger := log.NewLogger(cfg.Global.ErrorLog)
	logger.Info("Starting NginxGo %s", version)

	// 创建 HTTP 服务器
	httpServer := http.NewServer("0.0.0.0", 80)

	// 配置路由
	setupRoutes(httpServer, cfg, logger)

	// 启动服务器
	go func() {
		if err := httpServer.Serve(); err != nil {
			logger.Error("Server error: %v", err)
		}
	}()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, 
		syscall.SIGTERM, 
		syscall.SIGINT, 
		syscall.SIGQUIT, 
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)

	// 等待信号
	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT:
			logger.Info("Shutting down...")
			os.Exit(0)
		case syscall.SIGHUP:
			logger.Info("Reloading configuration...")
			// TODO: 实现配置重载
		case syscall.SIGUSR1:
			logger.Info("Reopening log files...")
		}
	}
}

// setupRoutes 配置路由
func setupRoutes(srv *http.Server, cfg *config.Config, logger *log.Logger) {
	// 配置 HTTP 服务器
	for _, serverCfg := range cfg.HTTP.Servers {
		for _, loc := range serverCfg.Locations {
			if loc.ProxyPass != "" {
				// 反向代理
				proxyHandler := handler.NewProxyHandler(loc.ProxyPass)
				srv.Handler.Get(loc.Path, proxyHandler)
				logger.Info("Registered proxy: %s -> %s", loc.Path, loc.ProxyPass)
			} else {
				// 静态文件
				root := loc.Root
				if root == "" {
					root = serverCfg.Root
				}
				if root == "" {
					root = "html"
				}
				
				index := loc.Index
				if len(index) == 0 {
					index = []string{"index.html"}
				}
				
				staticHandler := handler.NewStaticHandler(root, index)
				srv.Handler.Get(loc.Path, staticHandler)
				logger.Info("Registered static: %s -> %s", loc.Path, root)
			}
		}
	}

	// 默认路由
	srv.Handler.Get("/", handler.NewStaticHandler("html", []string{"index.html"}))
}

// WorkerCount 获取 Worker 进程数
func workerCount(cfg *config.Config) int {
	count := cfg.Global.WorkerProcesses
	if count <= 0 {
		count = runtime.NumCPU()
	}
	return count
}