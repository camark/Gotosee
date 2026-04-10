// Package agents 提供代理核心逻辑。
package agents

import (
	"context"
	"sync"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/providers"
	"github.com/aaif-goose/gogo/internal/session"
)

// Agent 主要的 goose 代理。
type Agent struct {
	provider             *SharedProvider
	config               *AgentConfig
	currentGooseMode     sync.Mutex
	extensionManager     *ExtensionManager
	finalOutputTool      *FinalOutputTool
	frontendTools        sync.Map // map[string]*FrontendTool
	frontendInstructions *string
	promptManager        sync.Mutex
	toolResultTx         ToolResultReceiver
	toolResultRx         *ToolResultReceiver
	retryManager         *RetryManager
	container            *Container
}

// FinalOutputTool 最终输出工具（简化版）。
type FinalOutputTool struct {
	// TODO: 实现最终输出逻辑
}

// Container 容器（用于隔离执行环境）。
type Container struct {
	// TODO: 实现容器逻辑
}

// NewAgent 创建新的 Agent。
func NewAgent() *Agent {
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	config := NewAgentConfig(
		sessionManager,
		"auto",
		false,
		GoosePlatformCLI,
	)
	return NewAgentWithConfig(config)
}

// NewAgentWithConfig 使用配置创建 Agent。
func NewAgentWithConfig(config *AgentConfig) *Agent {
	toolTx := make(ToolResultReceiver, 32)
	toolRx := toolTx

	retryManager := NewRetryManager()

	return &Agent{
		provider:         NewSharedProvider(nil),
		config:           config,
		extensionManager: NewExtensionManager(),
		finalOutputTool:  nil,
		toolResultTx:     toolTx,
		toolResultRx:     &toolRx,
		retryManager:     retryManager,
		container:        nil,
	}
}

// ResetRetryAttempts 重置重试计数。
func (a *Agent) ResetRetryAttempts() {
	a.retryManager.mu.Lock()
	defer a.retryManager.mu.Unlock()
	a.retryManager.retryAttempts = 0
}

// IncrementRetryAttempts 增加重试计数并返回新值。
func (a *Agent) IncrementRetryAttempts() uint32 {
	a.retryManager.mu.Lock()
	defer a.retryManager.mu.Unlock()
	a.retryManager.retryAttempts++
	return a.retryManager.retryAttempts
}

// GetRetryAttempts 获取当前重试计数。
func (a *Agent) GetRetryAttempts() uint32 {
	a.retryManager.mu.Lock()
	defer a.retryManager.mu.Unlock()
	return a.retryManager.retryAttempts
}

// Reply 处理用户消息并生成回复（核心方法）。
// 这是一个简化版本，实际实现需要更多逻辑。
func (a *Agent) Reply(ctx context.Context, messages []*conversation.Message) (<-chan AgentEvent, error) {
	eventChan := make(chan AgentEvent, 32)

	// TODO: 实现完整的 reply 逻辑
	// 1. 调用 LLM 生成回复
	// 2. 解析工具调用
	// 3. 执行工具
	// 4. 处理结果
	// 5. 循环直到完成

	go func() {
		defer close(eventChan)
		// 简化实现：仅发送消息事件
		for _, msg := range messages {
			eventChan <- &MessageEvent{Message: msg}
		}
	}()

	return eventChan, nil
}

// SetProvider 设置提供商。
func (a *Agent) SetProvider(p interface{}) {
	a.provider.Set(p.(providers.Provider))
}

// GetProvider 获取提供商。
func (a *Agent) GetProvider() interface{} {
	return a.provider.Get()
}

// GetExtensionManager 获取扩展管理器。
func (a *Agent) GetExtensionManager() *ExtensionManager {
	return a.extensionManager
}

// SwitchGooseMode 切换 Goose 模式。
func (a *Agent) SwitchGooseMode(mode string) {
	a.currentGooseMode.Lock()
	defer a.currentGooseMode.Unlock()
	a.config.GooseMode = mode
}

// GetGooseMode 获取当前 Goose 模式。
func (a *Agent) GetGooseMode() string {
	a.currentGooseMode.Lock()
	defer a.currentGooseMode.Unlock()
	return a.config.GooseMode
}
