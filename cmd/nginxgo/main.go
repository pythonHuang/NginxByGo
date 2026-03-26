package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nginxgo/nginxgo/pkg/core"
	"github.com/nginxgo/nginxgo/pkg/config"
)

var (
	version   = "0.1.0"
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

	// 创建主进程
	master := core.NewMaster(cfg)

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

	go func() {
		for sig := range sigChan {
			fmt.Printf("Received signal: %s\n", sig.String())
			master.HandleSignal(sig)
		}
	}()

	// 启动主进程
	if err := master.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start: %v\n", err)
		os.Exit(1)
	}
}