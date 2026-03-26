package core

import (
	"github.com/nginxgo/nginxgo/pkg/config"
	"github.com/nginxgo/nginxgo/pkg/log"
)

// Cycle 表示服务器运行周期
type Cycle struct {
	Config   *config.Config
	Modules  []Module
	OldCycle *Cycle
	Logger   *log.Logger
}

// NewCycle 创建新的运行周期
func NewCycle(cfg *config.Config) *Cycle {
	logger := log.NewLogger(cfg.Global.ErrorLog)
	return &Cycle{
		Config:  cfg,
		Modules: make([]Module, 0),
		Logger:  logger,
	}
}

// Init 初始化周期
func (c *Cycle) Init() error {
	c.Logger.Info("Initializing cycle")
	// 初始化模块
	for _, m := range c.Modules {
		if err := m.Init(c); err != nil {
			return err
		}
	}
	return nil
}

// NewCycleForReload 创建用于热重载的新周期
func (c *Cycle) NewCycleForReload(cfg *config.Config) *Cycle {
	old := *c
	newCycle := NewCycle(cfg)
	newCycle.OldCycle = &old
	return newCycle
}