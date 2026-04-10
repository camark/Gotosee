// Package agents 提供代理核心逻辑。
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/mcp"
	"github.com/camark/Gotosee/internal/permission"
	"github.com/camark/Gotosee/internal/providers"
	"github.com/camark/Gotosee/internal/session"
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

// marshalArgs 序列化工具参数。
func marshalArgs(args map[string]interface{}) []byte {
	if len(args) == 0 {
		return []byte("{}")
	}
	data, _ := json.Marshal(args)
	return data
}

// Reply 处理用户消息并生成回复（核心方法）。
func (a *Agent) Reply(ctx context.Context, messages []*conversation.Message) (<-chan AgentEvent, error) {
	eventChan := make(chan AgentEvent, 32)

	// 初始化生命周期状态（如果需要）
	if a.lifecycleManager.GetState() == AgentLifecycleCreated {
		a.lifecycleManager.Transition(AgentLifecycleInitializing)
		a.lifecycleManager.Transition(AgentLifecycleReady)
	}

	go func() {
		defer close(eventChan)

		// 转换到处理中状态
		a.lifecycleManager.Transition(AgentLifecycleProcessing)
		defer func() {
			// 确保返回到就绪状态
			if a.lifecycleManager.GetState() == AgentLifecycleProcessing {
				a.lifecycleManager.Transition(AgentLifecycleReady)
			}
		}()

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

		// 构建当前轮次的工作消息历史（只用于本轮次的工具调用）
		// 从原始 messages 开始，不修改原始数据
		workingHistory := make([]*conversation.Message, len(messages))
		copy(workingHistory, messages)

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

			// 检查并压缩消息以节省上下文
			workingHistory = a.checkAndCompact(workingHistory)

			// 调用 LLM 生成回复
			response, err := a.callLLM(ctx, provider, workingHistory, tools)
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

			// 构建 Assistant 消息内容：文本 + 所有 tool_calls
			assistantMsgContent := []conversation.MessageContent{}

			// 添加文本内容（如果有）
			if response.Content != "" {
				assistantMsgContent = append(assistantMsgContent, conversation.MessageContent{
					Type: conversation.MessageContentText,
					Text: response.Content,
				})

				// 发送文本回复事件
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
				// 首先：将所有 tool_calls 添加到 Assistant 消息内容中
				for _, toolCall := range response.ToolCalls {
					toolCallID := toolCall.ID
					if toolCallID == "" {
						toolCallID = fmt.Sprintf("call_%s_%d", toolCall.Name, turn)
					}

					assistantMsgContent = append(assistantMsgContent, conversation.MessageContent{
						Type:       conversation.MessageContentToolUse,
						ToolName:   toolCall.Name,
						ToolArgs:   marshalArgs(toolCall.Arguments),
						ToolCallID: toolCallID,
					})
				}

				// 将包含所有内容的 Assistant 消息添加到历史
				workingHistory = append(workingHistory, &conversation.Message{
					Role:    conversation.RoleAssistant,
					Content: assistantMsgContent,
				})

				// 然后：依次执行每个工具，添加对应的 Tool 消息
				for _, toolCall := range response.ToolCalls {
					toolCallID := toolCall.ID
					if toolCallID == "" {
						toolCallID = fmt.Sprintf("call_%s_%d", toolCall.Name, turn)
					}

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
						// 将错误添加到消息历史（不要 ToolName 字段）
						workingHistory = append(workingHistory, &conversation.Message{
							Role: conversation.RoleTool,
							Content: []conversation.MessageContent{
								{
									Type:       conversation.MessageContentToolResult,
									ToolResult: fmt.Sprintf("执行错误：%v", err),
								},
							},
							ToolCallID: toolCallID,
						})
					} else {
						eventChan <- &ToolResultEvent{
							Name:    toolCall.Name,
							Content: result.Content,
						}

						// 将工具结果添加到消息历史（不要 ToolName 字段）
						var toolResultText string
						if len(result.Content) > 0 && result.Content[0].Text != "" {
							toolResultText = result.Content[0].Text
						} else {
							toolResultText = "执行完成"
						}
						workingHistory = append(workingHistory, &conversation.Message{
							Role: conversation.RoleTool,
							Content: []conversation.MessageContent{
								{
									Type:       conversation.MessageContentToolResult,
									ToolResult: toolResultText,
								},
							},
							ToolCallID: toolCallID,
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
	ID        string
	Name      string
	Arguments map[string]interface{}
}

// callLLM 调用 LLM 生成回复。
func (a *Agent) callLLM(ctx context.Context, provider providers.Provider, messages []*conversation.Message, tools []*mcp.Tool) (*LLMResponse, error) {
	// 构建消息列表，首先添加系统提示词
	convMessages := make([]conversation.Message, 0, len(messages)+1)

	// 添加系统提示词，告诉 LLM 它可以使用工具
	if len(tools) > 0 {
		convMessages = append(convMessages, conversation.Message{
			Role: conversation.RoleSystem,
			Content: []conversation.MessageContent{
				{
					Type: conversation.MessageContentText,
					Text: "You are an AI assistant with access to tools. WHEN THE USER ASKS ABOUT FILES, DIRECTORIES, SYSTEM INFORMATION, OR ANY DATA THAT REQUIRES TOOL ACCESS, YOU MUST CALL THE APPROPRIATE TOOL. DO NOT describe what you would do - ACTUALLY CALL THE TOOL. Use the same language as the user's message for your response.",
				},
			},
		})
	}

	// 转换其余消息为 providers 需要的格式
	for _, msg := range messages {
		convMessages = append(convMessages, conversation.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		})
	}

	// 从 provider 获取模型配置
	config := provider.GetModelConfig()
	if config.Model == "" {
		config.Model = "gpt-4o" // 默认模型
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	// 将工具信息添加到 config 中（通过 ExtraParams 传递）
	if len(tools) > 0 {
		if config.ExtraParams == nil {
			config.ExtraParams = make(map[string]interface{})
		}
		config.ExtraParams["tools"] = tools
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
				// 尝试直接解析
				if err := json.Unmarshal(content.ToolArgs, &args); err != nil {
					// 如果失败，尝试修复常见的 JSON 转义问题
					fixedJSON := fixInvalidJSONEscapes(string(content.ToolArgs))
					if err2 := json.Unmarshal([]byte(fixedJSON), &args); err2 != nil {
						return nil, fmt.Errorf("parse tool args: %w (original error: %v)", err2, err)
					}
				}
			}
			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:        content.ToolCallID, // 从响应中获取工具调用 ID
				Name:      content.ToolName,
				Arguments: args,
			})
		}
	}

	return response, nil
}

// fixInvalidJSONEscapes 修复常见的 JSON 无效转义序列
func fixInvalidJSONEscapes(s string) string {
	// 替换无效的转义序列
	// 例如: \D → D, \_ → _, 等等
	// 但保留有效的转义序列: \", \\, \/, \b, \f, \n, \r, \t, \uXXXX
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			// 检查是否是有效的转义字符
			switch next {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't', 'u':
				// 有效的转义序列，保留
				result = append(result, '\\', next)
			default:
				// 无效的转义序列，只保留后面的字符
				result = append(result, next)
			}
			i += 2
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
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

// CollectTools 收集所有可用工具（公开方法）。
func (a *Agent) CollectTools() []*mcp.Tool {
	return a.collectTools()
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
		if callErr == nil && result != nil && !result.IsError {
			break
		}
	}

	if result == nil {
		return &ToolCallResult{
			ToolName: call.Name,
			Error:    fmt.Errorf("tool not found: %s (last error: %v)", call.Name, callErr),
		}, nil
	}

	// 检查工具执行是否出错
	if result.IsError {
		var content []*mcp.Content
		for _, c := range result.Content {
			content = append(content, &c)
		}
		return &ToolCallResult{
			ToolName: call.Name,
			Content:  content,
			Error:    fmt.Errorf("tool execution failed: %s", result.Content[0].Text),
		}, nil
	}

	// 转换 MCP 结果为 ToolCallResult
	content := make([]*mcp.Content, 0, len(result.Content))
	for _, c := range result.Content {
		content = append(content, &c)
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
				Role: conversation.RoleAssistant,
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
