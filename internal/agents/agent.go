// Package agents 提供代理核心逻辑。
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/mcp"
	"github.com/aaif-goose/gogo/internal/model"
	"github.com/aaif-goose/gogo/internal/permission"
	"github.com/aaif-goose/gogo/internal/providers"
	"github.com/aaif-goose/gogo/internal/session"
)

// Agent 主要的 goose 代理。
type Agent struct {
	provider               *SharedProvider
	config                 *AgentConfig
	currentGooseMode       sync.RWMutex
	extensionManager       *ExtensionManager
	finalOutputTool        *FinalOutputTool
	frontendTools          sync.Map // map[string]*FrontendTool
	frontendInstructions   *string
	promptManager          sync.Mutex
	toolResultTx           ToolResultReceiver
	toolResultRx           *ToolResultReceiver
	retryManager           *RetryManager
	lifecycleManager       *LifecycleManager
	toolConfirmationRouter *ToolConfirmationRouter
	permissionManager      *permission.PermissionManager
	container              *Container
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
	toolChan := make(ToolResultReceiver, 32)

	retryManager := NewRetryManager()
	lifecycleManager := NewLifecycleManager()
	toolConfirmationRouter := NewToolConfirmationRouter()
	permissionManager := permission.NewPermissionManager()

	return &Agent{
		provider:               NewSharedProvider(nil),
		config:                 config,
		extensionManager:       NewExtensionManager(),
		finalOutputTool:        nil,
		toolResultTx:           toolChan,
		toolResultRx:           &toolChan,
		retryManager:           retryManager,
		lifecycleManager:       lifecycleManager,
		toolConfirmationRouter: toolConfirmationRouter,
		permissionManager:      permissionManager,
		container:              nil,
	}
}

// ResetRetryAttempts 重置重试计数。
func (a *Agent) ResetRetryAttempts() {
	a.retryManager.Reset()
}

// IncrementRetryAttempts 增加重试计数并返回新值。
func (a *Agent) IncrementRetryAttempts() uint32 {
	return a.retryManager.Increment()
}

// GetRetryAttempts 获取当前重试计数。
func (a *Agent) GetRetryAttempts() uint32 {
	return a.retryManager.Get()
}

// Reply 处理用户消息并生成回复（核心方法）。
func (a *Agent) Reply(ctx context.Context, messages []*conversation.Message) (<-chan AgentEvent, error) {
	eventChan := make(chan AgentEvent, 32)

	go func() {
		defer close(eventChan)

		// 重置重试计数器
		a.retryManager.Reset()

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

		// 构建消息历史副本（避免修改原始数据）
		msgHistory := make([]*conversation.Message, len(messages))
		copy(msgHistory, messages)

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

			// 获取所有可用工具
			tools := a.collectTools()

			// 调用 LLM 生成回复
			response, err := a.callLLM(ctx, provider, msgHistory, tools)
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

			// 发送文本回复消息
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
					// 发送工具调用事件
					eventChan <- &ToolCallEvent{
						Name:      toolCall.Name,
						Arguments: toolCall.Arguments,
					}

					// 执行工具
					result, err := a.executeTool(ctx, toolCall)
					if err != nil {
						eventChan <- &ToolResultEvent{
							Name:  toolCall.Name,
							Error: err,
						}
						// 将错误添加到消息历史
						msgHistory = append(msgHistory, &conversation.Message{
							Role: conversation.RoleAssistant,
							Content: []conversation.MessageContent{
								{
									Type:       conversation.MessageContentToolResult,
									ToolName:   toolCall.Name,
									ToolResult: fmt.Sprintf("执行错误：%v", err),
								},
							},
						})
					} else {
						eventChan <- &ToolResultEvent{
							Name:    toolCall.Name,
							Content: result.Content,
						}

						// 将工具结果添加到消息历史
						msgHistory = append(msgHistory, &conversation.Message{
							Role: conversation.RoleAssistant,
							Content: []conversation.MessageContent{
								{
									Type:       conversation.MessageContentToolResult,
									ToolName:   toolCall.Name,
									ToolResult: result.Text(),
								},
							},
						})
					}
				}
			} else {
				// 没有工具调用，完成
				return
			}
		}

		// 达到最大轮次
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
	// 转换为 providers 需要的消息格式
	convMessages := make([]conversation.Message, len(messages))
	for i, msg := range messages {
		convMessages[i] = conversation.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
	}

	// 获取模型配置
	config := model.ModelConfig{
		Model:       "default",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// 调用提供商的 Complete 方法
	responseMsg, err := provider.Complete(ctx, convMessages, config)
	if err != nil {
		return nil, fmt.Errorf("provider complete failed: %w", err)
	}

	// 解析响应
	return a.parseLLMResponse(responseMsg)
}

// parseLLMResponse 解析 LLM 响应消息。
func (a *Agent) parseLLMResponse(msg conversation.Message) (*LLMResponse, error) {
	response := &LLMResponse{
		Content:   "",
		ToolCalls: []ToolCall{},
	}

	for _, content := range msg.Content {
		switch content.Type {
		case conversation.MessageContentText:
			response.Content = content.Text

		case conversation.MessageContentToolUse:
			// 解析工具调用
			var args map[string]interface{}
			if content.ToolArgs != nil {
				if err := json.Unmarshal(content.ToolArgs, &args); err != nil {
					return nil, fmt.Errorf("parse tool args: %w", err)
				}
			}
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				Name:      content.ToolName,
				Arguments: args,
			})
		}
	}

	return response, nil
}

// collectTools 收集所有可用工具。
func (a *Agent) collectTools() []*mcp.Tool {
	var tools []*mcp.Tool

	// 从扩展管理器获取工具
	extensionTools := a.extensionManager.ListTools()
	tools = append(tools, extensionTools...)

	// 从 MCP 服务器注册中心获取工具
	serverNames := mcp.List()
	for _, serverName := range serverNames {
		server, err := mcp.Get(serverName)
		if err != nil {
			continue
		}

		serverTools, err := server.ListTools(context.Background())
		if err != nil {
			continue
		}

		// 转换 MCP Tool 为内部类型
		for _, tool := range serverTools {
			t := tool // 创建副本避免引用问题
			tools = append(tools, &t)
		}
	}

	return tools
}

// executeTool 执行工具调用。
func (a *Agent) executeTool(ctx context.Context, call ToolCall) (*ToolCallResult, error) {
	// 参数验证
	if call.Name == "" {
		return &ToolCallResult{
			ToolName: call.Name,
			Error:    fmt.Errorf("tool name is required"),
		}, nil
	}

	// 序列化参数
	var args json.RawMessage
	if len(call.Arguments) > 0 {
		var err error
		args, err = json.Marshal(call.Arguments)
		if err != nil {
			return &ToolCallResult{
				ToolName: call.Name,
				Error:    fmt.Errorf("marshal arguments: %w", err),
			}, nil
		}
	}

	// 遍历所有注册的 MCP 服务器，找到包含该工具的服务器
	serverNames := mcp.List()
	var result *mcp.ToolResult
	var callErr error

	for _, serverName := range serverNames {
		server, err := mcp.Get(serverName)
		if err != nil {
			continue
		}

		// 尝试调用工具
		result, callErr = server.CallTool(ctx, call.Name, args)
		if callErr == nil {
			break
		}
	}

	if result == nil {
		return &ToolCallResult{
			ToolName: call.Name,
			Error:    fmt.Errorf("tool not found: %s (last error: %v)", call.Name, callErr),
		}, nil
	}

	// 转换 MCP 结果为 ToolCallResult
	content := make([]*mcp.Content, 0, len(result.Content))
	for _, c := range result.Content {
		content = append(content, &mcp.Content{
			Type:     c.Type,
			Text:     c.Text,
			Data:     c.Data,
			MIMEType: c.MIMEType,
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
	a.currentGooseMode.RLock()
	defer a.currentGooseMode.RUnlock()
	return a.config.GooseMode
}

// handleApprovalToolRequests 处理需要审批的工具请求。
func (a *Agent) handleApprovalToolRequests(
	toolRequests []*ToolRequest,
	toolFutures *map[string]*ToolCallResult,
	requestToResponseMap map[string]*conversation.Message,
	sessionID string,
) error {
	for _, request := range toolRequests {
		// 检查权限管理器
		if a.permissionManager.ShouldAutoApprove(request.ToolCall.Name) {
			// 自动批准
			result, err := a.executeTool(context.Background(), request.ToolCall)
			(*toolFutures)[request.ID] = result
			if err != nil {
				return err
			}
			continue
		}

		if a.permissionManager.ShouldAutoDeny(request.ToolCall.Name) {
			// 自动拒绝
			if response, ok := requestToResponseMap[request.ID]; ok {
				response.Content = append(response.Content, conversation.MessageContent{
					Type:       conversation.MessageContentToolResult,
					ToolName:   request.ToolCall.Name,
					ToolResult: "用户已拒绝运行此工具",
				})
			}
			continue
		}

		// 需要用户确认
		// 发送行动请求事件
		eventChan := make(chan AgentEvent, 1)
		eventChan <- &MessageEvent{
			Message: &conversation.Message{
				Role: conversation.RoleUser,
				Content: []conversation.MessageContent{
					{
						Type: conversation.MessageContentActionRequired,
						ActionRequired: &conversation.ActionRequiredData{
							Type:      "elicitation",
							ToolName:  request.ToolCall.Name,
							Arguments: request.ToolCall.Arguments,
						},
					},
				},
			},
		}

		// 注册确认通道
		confirmationCh := a.toolConfirmationRouter.Register(request.ID)

		// 等待确认
		confirmation := <-confirmationCh

		if confirmation.Permission == permission.AllowOnce || confirmation.Permission == permission.AlwaysAllow {
			// 执行工具
			result, err := a.executeTool(context.Background(), request.ToolCall)
			(*toolFutures)[request.ID] = result
			if err != nil {
				return err
			}

			// 如果总是允许，更新权限管理器
			if confirmation.Permission == permission.AlwaysAllow {
				a.permissionManager.SetPermission(request.ToolCall.Name, permission.PermissionLevelAlwaysAllow)
			}
		} else {
			// 拒绝
			if response, ok := requestToResponseMap[request.ID]; ok {
				response.Content = append(response.Content, conversation.MessageContent{
					Type:       conversation.MessageContentToolResult,
					ToolName:   request.ToolCall.Name,
					ToolResult: "用户已拒绝运行此工具",
				})
			}

			// 如果总是拒绝，更新权限管理器
			if confirmation.Permission == permission.AlwaysDeny {
				a.permissionManager.SetPermission(request.ToolCall.Name, permission.PermissionLevelNeverAllow)
			}
		}
	}

	return nil
}
