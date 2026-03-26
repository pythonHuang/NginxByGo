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
	name       string
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