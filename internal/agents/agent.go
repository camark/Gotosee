// Package agents 提供代理核心逻辑。
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/mcp"
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
func (a *Agent) Reply(ctx context.Context, messages []*conversation.Message) (<-chan AgentEvent, error) {
	eventChan := make(chan AgentEvent, 32)

	go func() {
		defer close(eventChan)

		// 获取提供商
		provider := a.provider.Get()
		if provider == nil {
			eventChan <- &MessageEvent{
				Message: &conversation.Message{
					Role: conversation.RoleAssistant,
					Content: []conversation.MessageContent{
						{Type: conversation.MessageContentText, Text: "错误：未配置 AI 提供商"},
					},
				},
			}
			return
		}

		// 获取所有可用工具
		tools := a.collectTools()

		// 主循环
		maxTurns := DefaultMaxTurns
		for turn := 0; turn < int(maxTurns); turn++ {
			select {
			case <-ctx.Done():
				eventChan <- &MessageEvent{
					Message: &conversation.Message{
						Role: conversation.RoleAssistant,
						Content: []conversation.MessageContent{
							{Type: conversation.MessageContentText, Text: "会话已取消"},
						},
					},
				}
				return
			default:
			}

			// 调用 LLM 生成回复
			response, err := a.callLLM(ctx, provider, messages, tools)
			if err != nil {
				eventChan <- &MessageEvent{
					Message: &conversation.Message{
						Role: conversation.RoleAssistant,
						Content: []conversation.MessageContent{
							{Type: conversation.MessageContentText, Text: fmt.Sprintf("错误：%v", err)},
						},
					},
				}
				return
			}

			// 发送回复消息
			if response.Content != "" {
				eventChan <- &MessageEvent{
					Message: &conversation.Message{
						Role: conversation.RoleAssistant,
						Content: []conversation.MessageContent{
							{Type: conversation.MessageContentText, Text: response.Content},
						},
					},
				}
			}

			// 处理工具调用
			if len(response.ToolCalls) > 0 {
				for _, toolCall := range response.ToolCalls {
					eventChan <- &ToolCallEvent{
						Name:      toolCall.Name,
						Arguments: toolCall.Arguments,
					}

					result, err := a.executeTool(ctx, toolCall)
					if err != nil {
						eventChan <- &ToolResultEvent{
							Name:  toolCall.Name,
							Error: err,
						}
					} else {
						eventChan <- &ToolResultEvent{
							Name:    toolCall.Name,
							Content: result.Content,
						}

						// 将工具结果添加到消息历史
						messages = append(messages, &conversation.Message{
							Role: conversation.RoleAssistant,
							Content: []conversation.MessageContent{
								{Type: conversation.MessageContentToolResult, ToolResult: result.Text()},
							},
						})
					}
				}
			} else {
				// 没有工具调用，完成
				return
			}
		}

		eventChan <- &MessageEvent{
			Message: &conversation.Message{
				Role: conversation.RoleAssistant,
				Content: []conversation.MessageContent{
					{Type: conversation.MessageContentText, Text: fmt.Sprintf("达到最大轮次限制 (%d)", maxTurns)},
				},
			},
		}
	}()

	return eventChan, nil
}

// LLMResponse LLM 响应。
type LLMResponse struct {
	Content   string
	ToolCalls []ToolCall
}

// ToolCall 工具调用。
type ToolCall struct {
	Name      string
	Arguments map[string]interface{}
}

// callLLM 调用 LLM 生成回复。
func (a *Agent) callLLM(ctx context.Context, provider providers.Provider, messages []*conversation.Message, tools []*mcp.Tool) (*LLMResponse, error) {
	// TODO: 实现完整的 LLM 调用逻辑
	// 这需要提供商实现 Chat 方法

	// 简化实现：返回空响应
	return &LLMResponse{
		Content:   "LLM 调用功能正在实现中...",
		ToolCalls: []ToolCall{},
	}, nil
}

// collectTools 收集所有可用工具。
func (a *Agent) collectTools() []*mcp.Tool {
	var tools []*mcp.Tool

	// 从扩展管理器获取工具
	for _, tool := range a.extensionManager.ListTools() {
		tools = append(tools, tool)
	}

	return tools
}

// executeTool 执行工具调用。
func (a *Agent) executeTool(ctx context.Context, call ToolCall) (*ToolCallResult, error) {
	// 调用工具
	args, _ := json.Marshal(call.Arguments)

	// 遍历所有注册的 MCP 服务器，找到包含该工具的服务器
	serverNames := mcp.List()
	var result *mcp.ToolResult
	for _, serverName := range serverNames {
		server, err := mcp.Get(serverName)
		if err != nil {
			continue
		}

		// 尝试调用工具
		result, err = server.CallTool(ctx, call.Name, args)
		if err == nil {
			break
		}
	}

	if result == nil {
		return &ToolCallResult{
			ToolName: call.Name,
			Error:    fmt.Errorf("tool not found in any server: %s", call.Name),
		}, nil
	}

	// 转换结果
	content := make([]*mcp.Content, 0, len(result.Content))
	for _, c := range result.Content {
		content = append(content, &mcp.Content{
			Type: c.Type,
			Text: c.Text,
		})
	}

	return &ToolCallResult{
		ToolName: call.Name,
		Content:  content,
		Error:    nil,
	}, nil
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
