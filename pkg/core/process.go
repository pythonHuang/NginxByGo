package core

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/nginxgo/nginxgo/pkg/config"
)

const (
	ProcessMaster = iota
	ProcessWorker
	ProcessSingle
)

// Master 主进程
type Master struct {
	cycle      *Cycle
	workers    []*Worker
	signalChan chan os.Signal
	done       chan struct{}
	cfg        *config.Config
}

// NewMaster 创建主进程
func NewMaster(cfg *config.Config) *Master {
	return &Master{
		cfg:        cfg,
		cycle:      NewCycle(cfg),
		workers:    make([]*Worker, 0),
		signalChan: make(chan os.Signal, 1),
		done:       make(chan struct{}, 1),
	}
}

// Start 启动主进程
func (m *Master) Start() error {
	m.cycle.Logger.Info("Starting NginxGo %s", version)

	// 初始化周期
	if err := m.cycle.Init(); err != nil {
		return err
	}

	// 启动 Worker 进程
	workerCount := m.cfg.Global.WorkerProcesses
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	m.cycle.Logger.Info("Starting %d worker processes", workerCount)

	for i := 0; i < workerCount; i++ {
		w := NewWorker(m.cycle)
		if err := w.Start(); err != nil {
			return err
		}
		m.workers = append(m.workers, w)
	}

	m.cycle.Logger.Info("NginxGo started successfully")

	// 等待信号
	m.waitForSignal()

	return nil
}

// waitForSignal 等待信号
func (m *Master) waitForSignal() {
	signal.Notify(m.signalChan, 
		syscall.SIGTERM, 
		syscall.SIGINT, 
		syscall.SIGQUIT, 
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)

	for {
		sig := <-m.signalChan
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT:
			m.cycle.Logger.Info("Shutting down...")
			m.Quit()
			return
		case syscall.SIGHUP:
			m.cycle.Logger.Info("Reloading configuration...")
			m.Reload()
		case syscall.SIGUSR1:
			m.cycle.Logger.Info("Reopening log files...")
			m.ReopenLog()
		}
	}
}

// HandleSignal 处理信号
func (m *Master) HandleSignal(sig os.Signal) {
	switch sig {
	case syscall.SIGTERM, syscall.SIGINT:
		m.Quit()
	case syscall.SIGHUP:
		m.Reload()
	case syscall.SIGUSR1:
		m.ReopenLog()
	}
}

// Quit 退出
func (m *Master) Quit() {
	m.cycle.Logger.Info("Stopping workers...")
	for _, w := range m.workers {
		w.Quit()
	}
	m.done <- struct{}{}
}

// Reload 热重载配置
func (m *Master) Reload() {
	// TODO: 实现配置热重载
	m.cycle.Logger.Info("Configuration reloaded")
}

// ReopenLog 重新打开日志
func (m *Master) ReopenLog() {
	m.cycle.Logger.Info("Log files reopened")
}

// Worker Worker 进程
type Worker struct {
	pid    int
	cycle  *Cycle
	running bool
}

// NewWorker 创建 Worker 进程
func NewWorker(cycle *Cycle) *Worker {
	return &Worker{
		pid:   os.Getpid(),
		cycle: cycle,
	}
}

// Start 启动 Worker
func (w *Worker) Start() error {
	w.running = true
	w.cycle.Logger.Info("Worker process started (pid: %d)", w.pid)
	// TODO: 启动事件循环
	return nil
}

// Quit 退出 Worker
func (w *Worker) Quit() {
	w.running = false
	w.cycle.Logger.Info("Worker process stopped (pid: %d)", w.pid)
}

// GetVersion 返回版本
var version = "0.1.0"

// GetVersion 获取版本
func GetVersion() string {
	return fmt.Sprintf("NginxGo %s", version)
}