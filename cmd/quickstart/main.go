package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nginxgo/nginxgo/pkg/config"
	"github.com/nginxgo/nginxgo/pkg/http/handler"
	"github.com/nginxgo/nginxgo/pkg/log"
)

// QuickStart 一键启动脚本
func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║           🚀 NginxGo 快速启动脚本                      ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	// 检查配置文件
	if _, err := os.Stat("conf/nginx.conf"); os.IsNotExist(err) {
		fmt.Println("⚠️  配置文件不存在，使用默认配置")
	}

	// 加载配置
	cfg, err := config.LoadConfig("conf/nginx.conf")
	if err != nil {
		fmt.Printf("⚠️  配置加载失败，使用默认配置: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// 创建日志器
	logger := log.NewLogger(cfg.Global.ErrorLog)
	logger.Info("Starting NginxGo...")

	// 创建 HTTP 服务器
	addr := "0.0.0.0"
	port := 8080
	
	if len(cfg.HTTP.Servers) > 0 {
		if cfg.HTTP.Servers[0].Listen != "" {
			fmt.Sscanf(cfg.HTTP.Servers[0].Listen, "%d", &port)
		}
	}
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      createHandler(cfg),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	fmt.Printf("🌐 服务器启动: http://localhost:%d\n", port)
	fmt.Println("📁 静态文件目录: html/")
	fmt.Println("📝 配置目录: conf/nginx.conf")
	fmt.Println()
	fmt.Println("按 Ctrl+C 停止服务器")
	fmt.Println()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func createHandler(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	
	// 注册静态文件处理
	htmlDir := "html"
	if len(cfg.HTTP.Servers) > 0 && cfg.HTTP.Servers[0].Root != "" {
		htmlDir = cfg.HTTP.Servers[0].Root
	}
	
	mux.Handle("/", http.FileServer(http.Dir(htmlDir)))
	
	// 注册代理
	for _, srv := range cfg.HTTP.Servers {
		for _, loc := range srv.Locations {
			if loc.ProxyPass != "" {
				mux.HandleFunc(loc.Path, func(w http.ResponseWriter, r *http.Request) {
					handler.NewProxyHandler(loc.ProxyPass).ServeHTTP(
						&mockResponseWriter{w}, r)
				})
			}
		}
	}
	
	return mux
}

// mockResponseWriter 实现 http.ResponseWriter
type mockResponseWriter struct {
	http.ResponseWriter
}

func (m *mockResponseWriter) Header() http.Header {
	return m.ResponseWriter.Header()
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.ResponseWriter.Write(b)
}

func (m *mockResponseWriter) WriteHeader(code int) {
	m.ResponseWriter.WriteHeader(code)
}