// Package agents 提供扩展状态持久化功能。
package agents

import (
	"encoding/json"
	"sync"

	"github.com/camark/Gotosee/internal/session"
)

// ExtensionState 扩展状态。
type ExtensionState struct {
	Name    string      `json:"name"`
	Config  interface{} `json:"config"`
	Enabled bool        `json:"enabled"`
}

// EnabledExtensionsState 已启用扩展状态。
type EnabledExtensionsState struct {
	Extensions []ExtensionState `json:"extensions"`
}

// NewEnabledExtensionsState 从扩展配置创建状态。
func NewEnabledExtensionsState(configs []*ExtensionConfig) *EnabledExtensionsState {
	states := make([]ExtensionState, 0, len(configs))
	for _, config := range configs {
		states = append(states, ExtensionState{
			Name:    config.Name,
			Config:  config.ConfigData,
			Enabled: config.Enabled,
		})
	}
	return &EnabledExtensionsState{Extensions: states}
}

// ToExtensionData 转换为 ExtensionData。
func (s *EnabledExtensionsState) ToExtensionData(data *session.ExtensionData) error {
	if data == nil {
		return nil
	}

	marshaled, err := json.Marshal(s)
	if err != nil {
		return err
	}

	// 存储为 "extensions.state" 键
	data.SetExtensionState("extensions", "v1", marshaled)
	return nil
}

// FromExtensionData 从 ExtensionData 加载状态。
func FromExtensionData(data *session.ExtensionData) (*EnabledExtensionsState, error) {
	if data == nil {
		return &EnabledExtensionsState{Extensions: []ExtensionState{}}, nil
	}

	raw := data.GetExtensionState("extensions", "v1")
	if raw == nil {
		return &EnabledExtensionsState{Extensions: []ExtensionState{}}, nil
	}

	var state EnabledExtensionsState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// ExtensionStateCache 扩展状态缓存（线程安全）。
type ExtensionStateCache struct {
	mu     sync.RWMutex
	states map[string]*EnabledExtensionsState // session_id -> state
}

// NewExtensionStateCache 创建扩展状态缓存。
func NewExtensionStateCache() *ExtensionStateCache {
	return &ExtensionStateCache{
		states: make(map[string]*EnabledExtensionsState),
	}
}

// Get 获取会话的扩展状态。
func (c *ExtensionStateCache) Get(sessionID string) *EnabledExtensionsState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.states[sessionID]
}

// Set 设置会话的扩展状态。
func (c *ExtensionStateCache) Set(sessionID string, state *EnabledExtensionsState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.states[sessionID] = state
}

// Delete 删除会话的扩展状态。
func (c *ExtensionStateCache) Delete(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.states, sessionID)
}
