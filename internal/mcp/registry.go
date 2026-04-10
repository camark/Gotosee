package mcp

import (
	"fmt"
	"sync"
)

// ============================================================================
// MCP 注册中心
// ============================================================================

// Registry MCP 服务器注册中心。
type Registry struct {
	mu       sync.RWMutex
	servers  map[string]Server
	factories map[string]ServerFactory
}

// ServerFactory 服务器工厂函数。
type ServerFactory func() Server

// globalRegistry 全局注册中心。
var globalRegistry = NewRegistry()

// NewRegistry 创建新的注册中心。
func NewRegistry() *Registry {
	return &Registry{
		servers:   make(map[string]Server),
		factories: make(map[string]ServerFactory),
	}
}

// Register 注册服务器工厂。
func (r *Registry) Register(name string, factory ServerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Get 获取服务器实例。
func (r *Registry) Get(name string) (Server, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 首先检查已创建的服务器
	if server, ok := r.servers[name]; ok {
		return server, nil
	}

	// 检查工厂
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("MCP 服务器 '%s' 未注册", name)
	}

	// 创建新实例
	server := factory()
	r.servers[name] = server
	return server, nil
}

// List 列出所有可用的服务器。
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Register 注册服务器工厂到全局注册中心。
func Register(name string, factory ServerFactory) {
	globalRegistry.Register(name, factory)
}

// Get 从全局注册中心获取服务器。
func Get(name string) (Server, error) {
	return globalRegistry.Get(name)
}

// List 列出全局注册中心的所有服务器。
func List() []string {
	return globalRegistry.List()
}

// ============================================================================
// 内置服务器注册
// ============================================================================

func init() {
	// 注册计算机控制服务器
	Register("computer-controller", func() Server {
		return NewComputerController()
	})

	// 注册文档工具服务器
	Register("doctools", func() Server {
		return NewDocTools()
	})

	// 注册记忆服务器
	Register("memory", func() Server {
		return NewMemoryServer()
	})

	// 注册教程服务器
	Register("tutorial", func() Server {
		return NewTutorialServer()
	})
}
