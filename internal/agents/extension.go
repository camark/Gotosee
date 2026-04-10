// Package agents 提供扩展管理功能。
package agents

import (
	"fmt"
	"sync"

	"github.com/camark/Gotosee/internal/mcp"
	"github.com/camark/Gotosee/internal/session"
)

// ExtensionConfig 扩展配置。
type ExtensionConfig struct {
	Name       string                 `json:"name"`
	Type       ExtensionType          `json:"type"`
	Enabled    bool                   `json:"enabled"`
	ConfigData map[string]interface{} `json:"config,omitempty"`
}

// ExtensionType 扩展类型。
type ExtensionType string

const (
	ExtensionTypeMCP     ExtensionType = "mcp"
	ExtensionTypeBuiltin ExtensionType = "builtin"
)

// ExtensionError 扩展错误。
type ExtensionError struct {
	Name    string
	Message string
	Code    string
}

func (e *ExtensionError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Name, e.Message)
}

// ExtensionLoadResult 扩展加载结果。
type ExtensionLoadResult struct {
	Name    string  `json:"name"`
	Success bool    `json:"success"`
	Error   *string `json:"error,omitempty"`
}

// ExtensionManagerCapabilities 扩展管理器能力。
type ExtensionManagerCapabilities struct {
	MCPUI bool `json:"mcpui"`
}

// ExtensionManager 扩展管理器。
type ExtensionManager struct {
	mu           sync.RWMutex
	extensions   map[string]*ExtensionConfig
	tools        map[string]*mcp.Tool
	capabilities ExtensionManagerCapabilities
	platform     string
}

// NewExtensionManager 创建扩展管理器。
func NewExtensionManager() *ExtensionManager {
	return &ExtensionManager{
		extensions: make(map[string]*ExtensionConfig),
		tools:      make(map[string]*mcp.Tool),
		capabilities: ExtensionManagerCapabilities{
			MCPUI: false,
		},
		platform: "goose-cli",
	}
}

// RegisterExtension 注册扩展。
func (em *ExtensionManager) RegisterExtension(config *ExtensionConfig) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.extensions[config.Name]; exists {
		return &ExtensionError{
			Name:    config.Name,
			Message: "extension already registered",
			Code:    "ALREADY_EXISTS",
		}
	}

	em.extensions[config.Name] = config
	return nil
}

// UnregisterExtension 注销扩展。
func (em *ExtensionManager) UnregisterExtension(name string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.extensions[name]; !exists {
		return &ExtensionError{
			Name:    name,
			Message: "extension not found",
			Code:    "NOT_FOUND",
		}
	}

	delete(em.extensions, name)
	return nil
}

// GetExtension 获取扩展配置。
func (em *ExtensionManager) GetExtension(name string) (*ExtensionConfig, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	ext, exists := em.extensions[name]
	if !exists {
		return nil, &ExtensionError{
			Name:    name,
			Message: "extension not found",
			Code:    "NOT_FOUND",
		}
	}

	return ext, nil
}

// ListExtensions 列出所有扩展。
func (em *ExtensionManager) ListExtensions() []*ExtensionConfig {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]*ExtensionConfig, 0, len(em.extensions))
	for _, ext := range em.extensions {
		result = append(result, ext)
	}
	return result
}

// EnableExtension 启用扩展。
func (em *ExtensionManager) EnableExtension(name string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	ext, exists := em.extensions[name]
	if !exists {
		return &ExtensionError{
			Name:    name,
			Message: "extension not found",
			Code:    "NOT_FOUND",
		}
	}

	ext.Enabled = true
	return nil
}

// DisableExtension 禁用扩展。
func (em *ExtensionManager) DisableExtension(name string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	ext, exists := em.extensions[name]
	if !exists {
		return &ExtensionError{
			Name:    name,
			Message: "extension not found",
			Code:    "NOT_FOUND",
		}
	}

	ext.Enabled = false
	return nil
}

// RegisterTool 注册工具。
func (em *ExtensionManager) RegisterTool(name string, tool *mcp.Tool) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.tools[name] = tool
	return nil
}

// UnregisterTool 注销工具。
func (em *ExtensionManager) UnregisterTool(name string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.tools[name]; !exists {
		return &ExtensionError{
			Name:    name,
			Message: "tool not found",
			Code:    "NOT_FOUND",
		}
	}

	delete(em.tools, name)
	return nil
}

// GetTool 获取工具。
func (em *ExtensionManager) GetTool(name string) (*mcp.Tool, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	tool, exists := em.tools[name]
	if !exists {
		return nil, &ExtensionError{
			Name:    name,
			Message: "tool not found",
			Code:    "NOT_FOUND",
		}
	}

	return tool, nil
}

// ListTools 列出所有工具。
func (em *ExtensionManager) ListTools() []*mcp.Tool {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]*mcp.Tool, 0, len(em.tools))
	for _, tool := range em.tools {
		result = append(result, tool)
	}
	return result
}

// SetCapabilities 设置能力。
func (em *ExtensionManager) SetCapabilities(caps ExtensionManagerCapabilities) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.capabilities = caps
}

// GetCapabilities 获取能力。
func (em *ExtensionManager) GetCapabilities() ExtensionManagerCapabilities {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.capabilities
}

// SetPlatform 设置平台。
func (em *ExtensionManager) SetPlatform(platform string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.platform = platform
}

// GetPlatform 获取平台。
func (em *ExtensionManager) GetPlatform() string {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.platform
}

// GetExtensionConfigs 获取所有扩展配置。
func (em *ExtensionManager) GetExtensionConfigs() []*ExtensionConfig {
	em.mu.RLock()
	defer em.mu.RUnlock()

	result := make([]*ExtensionConfig, 0, len(em.extensions))
	for _, ext := range em.extensions {
		result = append(result, ext)
	}
	return result
}

// SaveExtensionState 保存扩展状态到会话。
func (em *ExtensionManager) SaveExtensionState(sess *session.Session) error {
	configs := em.GetExtensionConfigs()
	state := NewEnabledExtensionsState(configs)

	if sess.ExtensionData == nil {
		sess.ExtensionData = session.NewExtensionData()
	}

	return state.ToExtensionData(sess.ExtensionData)
}

// LoadExtensionState 从会话加载扩展状态。
func (em *ExtensionManager) LoadExtensionState(sess *session.Session) error {
	state, err := FromExtensionData(sess.ExtensionData)
	if err != nil {
		return err
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	for _, extState := range state.Extensions {
		config := &ExtensionConfig{
			Name:       extState.Name,
			Enabled:    extState.Enabled,
			ConfigData: make(map[string]interface{}),
		}

		if extState.Config != nil {
			if cfgMap, ok := extState.Config.(map[string]interface{}); ok {
				config.ConfigData = cfgMap
			}
		}

		em.extensions[extState.Name] = config
	}

	return nil
}
