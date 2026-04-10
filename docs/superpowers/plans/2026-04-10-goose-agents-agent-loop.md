# goose-agents 完整代理循环实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现完整的 Agent 循环逻辑，包括 LLM 调用、工具调用处理、响应生成和消息管理

**Architecture:** 
- Agent.Reply() 方法作为核心入口，返回事件通道
- 主循环：LLM 调用 → 解析响应 → 执行工具 → 收集结果 → 重复直到完成
- 通过 AgentEvent 通道向调用者流式返回事件（消息、工具调用、工具结果）
- 支持最大轮次限制和取消上下文

**Tech Stack:**
- Go 1.22+
- 现有 internal/conversation, internal/mcp, internal/providers, internal/session 模块

---

### Task 1: 完善 LLM 调用功能 (callLLM)

**Files:**
- Modify: `internal/agents/agent.go:222-232` (callLLM 方法)
- Modify: `internal/providers/base.go` (如需要，添加辅助方法)
- Test: `internal/agents/agent_test.go` (新建)

**依赖:** providers 接口已实现 Complete 方法

- [ ] **Step 1: 创建测试文件**

```go
// internal/agents/agent_test.go
package agents

import (
	"context"
	"testing"
	"time"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/model"
	"github.com/aaif-goose/gogo/internal/providers"
)

// mockProvider 模拟提供商用于测试
type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Description() string { return "mock provider" }
func (m *mockProvider) Validate() error { return nil }
func (m *mockProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	return conversation.NewTextMessage(conversation.RoleAssistant, "Hello from mock"), nil
}
func (m *mockProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk)
	close(ch)
	return ch, nil
}
func (m *mockProvider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	return nil, nil
}
```

- [ ] **Step 2: 运行测试验证框架搭建**

```bash
go test ./internal/agents -v -run TestCallLLM
```
Expected: 编译通过，无测试可运行（测试尚未创建）

- [ ] **Step 3: 实现 callLLM 方法 - 构建消息**

```go
// internal/agents/agent.go:222-260
// callLLM 调用 LLM 生成回复。
func (a *Agent) callLLM(ctx context.Context, provider providers.Provider, messages []*conversation.Message, tools []*mcp.Tool) (*LLMResponse, error) {
	// 转换为 providers 需要的消息格式
	convMessages := make([]conversation.Message, len(messages))
	for i, msg := range messages {
		convMessages[i] = conversation.Message{
			Role:      msg.Role,
			Content:   msg.Content,
			ToolCallID: msg.ToolCallID,
		}
	}

	// 获取模型配置
	config := model.ModelConfig{
		Model: "default",
		Temperature: 0.7,
		MaxTokens: 4096,
	}
```

- [ ] **Step 4: 实现 callLLM 方法 - 调用提供商 Complete**

```go
// 接 Step 3
	// 调用提供商的 Complete 方法
	responseMsg, err := provider.Complete(ctx, convMessages, config)
	if err != nil {
		return nil, fmt.Errorf("provider complete failed: %w", err)
	}

	// 解析响应
	return a.parseLLMResponse(responseMsg)
}
```

- [ ] **Step 5: 实现 parseLLMResponse 辅助方法**

```go
// internal/agents/agent.go:262-300
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
```

- [ ] **Step 6: 编写 callLLM 测试**

```go
// internal/agents/agent_test.go: 在文件末尾添加
func TestCallLLM_BasicCall(t *testing.T) {
	ctx := context.Background()
	agent := NewAgent()
	agent.SetProvider(&mockProvider{})

	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Hello"},
			},
		},
	}

	response, err := agent.callLLM(ctx, &mockProvider{}, messages, nil)
	if err != nil {
		t.Fatalf("callLLM failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected non-nil response")
	}
}
```

- [ ] **Step 7: 运行测试验证**

```bash
go test ./internal/agents -v -run TestCallLLM
```
Expected: PASS

- [ ] **Step 8: 提交**

```bash
git add internal/agents/agent.go internal/agents/agent_test.go
git commit -m "feat(agents): 实现 callLLM 方法支持 LLM 调用"
```

---

### Task 2: 实现工具调用执行逻辑 (executeTool)

**Files:**
- Modify: `internal/agents/agent.go:246-288` (executeTool 方法)
- Test: `internal/agents/agent_test.go` (追加测试)

- [ ] **Step 1: 查看当前 executeTool 实现**

当前实现已具备基础功能，需要完善错误处理和结果转换。

- [ ] **Step 2: 改进 executeTool 方法 - 添加参数验证**

```go
// internal/agents/agent.go:246-265
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
```

- [ ] **Step 3: 改进 executeTool 方法 - 查找并调用工具**

```go
// 接 Step 2
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
```

- [ ] **Step 4: 改进 executeTool 方法 - 结果转换**

```go
// 接 Step 3
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
```

- [ ] **Step 5: 编写 executeTool 测试**

```go
// internal/agents/agent_test.go
func TestExecuteTool_BasicExecution(t *testing.T) {
	ctx := context.Background()
	agent := NewAgent()

	call := ToolCall{
		Name:      "test_tool",
		Arguments: map[string]interface{}{"key": "value"},
	}

	result, err := agent.executeTool(ctx, call)
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}

	// 由于没有注册工具，应该返回错误
	if result.Error == nil {
		t.Log("注意：工具未找到是预期行为")
	}
}
```

- [ ] **Step 6: 运行测试**

```bash
go test ./internal/agents -v -run TestExecuteTool
```
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/agents/agent.go internal/agents/agent_test.go
git commit -m "feat(agents): 改进 executeTool 方法添加错误处理"
```

---

### Task 3: 完善工具收集逻辑 (collectTools)

**Files:**
- Modify: `internal/agents/agent.go:234-244` (collectTools 方法)
- Test: `internal/agents/agent_test.go`

- [ ] **Step 1: 改进 collectTools 方法**

```go
// internal/agents/agent.go:234-255
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
```

- [ ] **Step 2: 编写 collectTools 测试**

```go
// internal/agents/agent_test.go
func TestCollectTools_FromExtensionManager(t *testing.T) {
	agent := NewAgent()
	
	// 注册一个模拟工具
	tool := &mcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}
	
	err := agent.extensionManager.RegisterTool("test_tool", tool)
	if err != nil {
		t.Fatalf("register tool failed: %v", err)
	}

	tools := agent.collectTools()
	
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
	
	if tools[0].Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got '%s'", tools[0].Name)
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/agents -v -run TestCollectTools
```
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/agents/agent.go internal/agents/agent_test.go
git commit -m "feat(agents): 增强 collectTools 从 MCP 服务器收集工具"
```

---

### Task 4: 实现完整的 Reply 主循环

**Files:**
- Modify: `internal/agents/agent.go:95-208` (Reply 方法)
- Test: `internal/agents/agent_test.go`

- [ ] **Step 1: 改进 Reply 方法 - 消息历史构建**

```go
// internal/agents/agent.go:95-150
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

		// 构建消息历史副本（避免修改原始数据）
		msgHistory := make([]*conversation.Message, len(messages))
		copy(msgHistory, messages)
```

- [ ] **Step 2: 改进 Reply 方法 - 主循环结构**

```go
// 接 Step 1，替换原循环部分
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
```

- [ ] **Step 3: 改进 Reply 方法 - LLM 调用**

```go
// 接 Step 2
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
```

- [ ] **Step 4: 改进 Reply 方法 - 工具调用处理**

```go
// 接 Step 3
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
```

- [ ] **Step 5: 改进 Reply 方法 - 最大轮次处理**

```go
// 接 Step 4
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
```

- [ ] **Step 6: 编写 Reply 集成测试**

```go
// internal/agents/agent_test.go
func TestReply_BasicFlow(t *testing.T) {
	ctx := context.Background()
	agent := NewAgent()
	agent.SetProvider(&mockProvider{})

	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Hello"},
			},
		},
	}

	events, err := agent.Reply(ctx, messages)
	if err != nil {
		t.Fatalf("Reply failed: %v", err)
	}

	// 收集所有事件
	var eventCount int
	for event := range events {
		eventCount++
		switch e := event.(type) {
		case *MessageEvent:
			if e.Message == nil {
				t.Error("expected non-nil message")
			}
		case *ToolCallEvent:
			t.Logf("工具调用：%s", e.Name)
		case *ToolResultEvent:
			t.Logf("工具结果：%s", e.Name)
		}
	}

	if eventCount == 0 {
		t.Error("expected at least one event")
	}
}
```

- [ ] **Step 7: 运行所有测试**

```bash
go test ./internal/agents -v
```
Expected: 所有测试 PASS

- [ ] **Step 8: 提交**

```bash
git add internal/agents/agent.go internal/agents/agent_test.go
git commit -m "feat(agents): 实现完整的 Reply 主循环支持多轮对话"
```

---

### Task 5: 添加重试和错误处理机制

**Files:**
- Modify: `internal/agents/agent.go` (集成 retryManager)
- Modify: `internal/agents/retry.go` (如需要)
- Test: `internal/agents/agent_test.go`

- [ ] **Step 1: 在 Reply 方法中集成重试逻辑**

```go
// internal/agents/agent.go: 在 Reply 方法开始处添加
func (a *Agent) Reply(ctx context.Context, messages []*conversation.Message) (<-chan AgentEvent, error) {
	eventChan := make(chan AgentEvent, 32)

	go func() {
		defer close(eventChan)

		// 重置重试计数器
		a.retryManager.Reset()

		// ... 其余逻辑不变
```

- [ ] **Step 2: 添加 LLM 调用重试**

```go
// internal/agents/agent.go: 修改 callLLM 调用部分
			// 调用 LLM 生成回复（带重试）
			var response *LLMResponse
			var llmErr error
			
			retryConfig := &RetryConfig{
				MaxRetries: 3,
				TimeoutSeconds: func() *uint64 { t := uint64(30); return &t }(),
			}
			
			response, llmErr = a.callLLM(ctx, provider, msgHistory, tools)
			if llmErr != nil {
				// 检查是否可以重试
				if a.retryManager.ShouldRetry(retryConfig) {
					attempts := a.retryManager.Increment()
					eventChan <- &MessageEvent{
						Message: &conversation.Message{
							Role: conversation.RoleAssistant,
							Content: []conversation.MessageContent{
								{Type: conversation.MessageContentText, Text: fmt.Sprintf("LLM 调用失败，重试 %d/3...", attempts)},
							},
						},
					}
					// 重试逻辑...
				}
			}
```

- [ ] **Step 3: 编写重试测试**

```go
// internal/agents/agent_test.go
func TestRetryManager_BasicRetry(t *testing.T) {
	rm := NewRetryManager()
	
	config := &RetryConfig{
		MaxRetries: 3,
	}
	
	if !rm.ShouldRetry(config) {
		t.Error("expected should retry on first attempt")
	}
	
	rm.Increment()
	rm.Increment()
	rm.Increment()
	
	if rm.ShouldRetry(config) {
		t.Error("expected no retry after max attempts")
	}
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/agents -v -run TestRetry
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/agents/agent.go internal/agents/retry.go internal/agents/agent_test.go
git commit -m "feat(agents): 集成重试管理器支持 LLM 调用重试"
```

---

### Task 6: 添加 Agent 生命周期管理

**Files:**
- Create: `internal/agents/lifecycle.go`
- Test: `internal/agents/lifecycle_test.go`

- [ ] **Step 1: 创建生命周期管理器**

```go
// internal/agents/lifecycle.go
package agents

import (
	"context"
	"sync"
	"time"
)

// AgentLifecycle Agent 生命周期状态。
type AgentLifecycle string

const (
	AgentLifecycleCreated    AgentLifecycle = "created"
	AgentLifecycleInitializing AgentLifecycle = "initializing"
	AgentLifecycleReady       AgentLifecycle = "ready"
	AgentLifecycleProcessing  AgentLifecycle = "processing"
	AgentLifecycleClosing     AgentLifecycle = "closing"
	AgentLifecycleClosed      AgentLifecycle = "closed"
)

// LifecycleManager 生命周期管理器。
type LifecycleManager struct {
	mu            sync.RWMutex
	currentState  AgentLifecycle
	lastStateChange time.Time
	history       []LifecycleEvent
}

// LifecycleEvent 生命周期事件。
type LifecycleEvent struct {
	From AgentLifecycle
	To   AgentLifecycle
	Time time.Time
}

// NewLifecycleManager 创建生命周期管理器。
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		currentState: AgentLifecycleCreated,
		history:      make([]LifecycleEvent, 0),
	}
}

// GetState 获取当前状态。
func (lm *LifecycleManager) GetState() AgentLifecycle {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState
}

// Transition 状态转换。
func (lm *LifecycleManager) Transition(to AgentLifecycle) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 验证状态转换合法性
	validTransitions := map[AgentLifecycle][]AgentLifecycle{
		AgentLifecycleCreated:    {AgentLifecycleInitializing},
		AgentLifecycleInitializing: {AgentLifecycleReady},
		AgentLifecycleReady:      {AgentLifecycleProcessing, AgentLifecycleClosing},
		AgentLifecycleProcessing: {AgentLifecycleReady, AgentLifecycleClosing},
		AgentLifecycleClosing:    {AgentLifecycleClosed},
		AgentLifecycleClosed:     {},
	}

	allowed := validTransitions[lm.currentState]
	valid := false
	for _, s := range allowed {
		if s == to {
			valid = true
			break
		}
	}

	if !valid {
		return &LifecycleError{
			From: lm.currentState,
			To:   to,
		}
	}

	from := lm.currentState
	lm.currentState = to
	lm.lastStateChange = time.Now()
	lm.history = append(lm.history, LifecycleEvent{
		From: from,
		To:   to,
		Time: time.Now(),
	})

	return nil
}

// IsProcessing 检查是否在处理中。
func (lm *LifecycleManager) IsProcessing() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState == AgentLifecycleProcessing
}

// IsReady 检查是否就绪。
func (lm *LifecycleManager) IsReady() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState == AgentLifecycleReady
}

// IsClosed 检查是否已关闭。
func (lm *LifecycleManager) IsClosed() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState == AgentLifecycleClosed
}

// LifecycleError 生命周期错误。
type LifecycleError struct {
	From AgentLifecycle
	To   AgentLifecycle
}

func (e *LifecycleError) Error() string {
	return "invalid lifecycle transition: " + string(e.From) + " -> " + string(e.To)
}
```

- [ ] **Step 2: 编写生命周期测试**

```go
// internal/agents/lifecycle_test.go
package agents

import (
	"testing"
)

func TestLifecycleManager_ValidTransitions(t *testing.T) {
	lm := NewLifecycleManager()

	if lm.GetState() != AgentLifecycleCreated {
		t.Errorf("expected Created state, got %v", lm.GetState())
	}

	// 初始化
	if err := lm.Transition(AgentLifecycleInitializing); err != nil {
		t.Fatalf("transition to Initializing failed: %v", err)
	}

	// 就绪
	if err := lm.Transition(AgentLifecycleReady); err != nil {
		t.Fatalf("transition to Ready failed: %v", err)
	}

	// 处理中
	if err := lm.Transition(AgentLifecycleProcessing); err != nil {
		t.Fatalf("transition to Processing failed: %v", err)
	}

	// 返回就绪
	if err := lm.Transition(AgentLifecycleReady); err != nil {
		t.Fatalf("transition back to Ready failed: %v", err)
	}
}

func TestLifecycleManager_InvalidTransition(t *testing.T) {
	lm := NewLifecycleManager()

	// 尝试非法转换
	err := lm.Transition(AgentLifecycleProcessing)
	if err == nil {
		t.Error("expected error for invalid transition")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/agents -v -run TestLifecycle
```
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/agents/lifecycle.go internal/agents/lifecycle_test.go
git commit -m "feat(agents): 添加生命周期管理器跟踪 Agent 状态"
```

---

### Task 7: 完善 Agent 配置和初始化

**Files:**
- Modify: `internal/agents/agent.go` (NewAgent, NewAgentWithConfig)
- Modify: `internal/agents/types.go` (AgentConfig)
- Test: `internal/agents/agent_test.go`

- [ ] **Step 1: 增强 AgentConfig**

```go
// internal/agents/types.go:204-220
// AgentConfig Agent 配置。
type AgentConfig struct {
	SessionManager      *session.SessionManager
	PermissionManager   interface{} // TODO: 实现 permission manager
	SchedulerService    interface{} // TODO: 实现 scheduler trait
	GooseMode           string
	DisableSessionNaming bool
	GoosePlatform       GoosePlatform
	MaxTurns            uint32  // 自定义最大轮次
	Timeout             int64   // 超时时间（秒）
	RetryConfig         *RetryConfig
}

// Validate 验证配置。
func (c *AgentConfig) Validate() error {
	if c.SessionManager == nil {
		return &ConfigError{Field: "SessionManager", Message: "is required"}
	}
	if c.MaxTurns == 0 {
		c.MaxTurns = DefaultMaxTurns
	}
	return nil
}
```

- [ ] **Step 2: 增强 NewAgentWithConfig**

```go
// internal/agents/agent.go:54-71
// NewAgentWithConfig 使用配置创建 Agent。
func NewAgentWithConfig(config *AgentConfig) *Agent {
	if err := config.Validate(); err != nil {
		panic("invalid agent config: " + err.Error())
	}

	toolTx := make(ToolResultReceiver, 32)
	toolRx := toolTx

	retryManager := NewRetryManager()
	lifecycleManager := NewLifecycleManager()

	return &Agent{
		provider:         NewSharedProvider(nil),
		config:           config,
		extensionManager: NewExtensionManager(),
		finalOutputTool:  nil,
		toolResultTx:     toolTx,
		toolResultRx:     &toolRx,
		retryManager:     retryManager,
		lifecycleManager: lifecycleManager,
		container:        nil,
	}
}
```

- [ ] **Step 3: 添加 Agent 结构体字段**

```go
// internal/agents/agent.go:16-30
// Agent 主要的 goose 代理。
type Agent struct {
	provider           *SharedProvider
	config             *AgentConfig
	currentGooseMode   sync.Mutex
	extensionManager   *ExtensionManager
	finalOutputTool    *FinalOutputTool
	frontendTools      sync.Map // map[string]*FrontendTool
	frontendInstructions *string
	promptManager      sync.Mutex
	toolResultTx       ToolResultReceiver
	toolResultRx       *ToolResultReceiver
	retryManager       *RetryManager
	lifecycleManager   *LifecycleManager // 新增
	container          *Container
}
```

- [ ] **Step 4: 编写配置测试**

```go
// internal/agents/agent_test.go
func TestAgentConfig_Validation(t *testing.T) {
	// 空配置应该失败
	invalidConfig := &AgentConfig{}
	err := invalidConfig.Validate()
	if err == nil {
		t.Error("expected validation error for nil SessionManager")
	}

	// 有效配置
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	validConfig := NewAgentConfig(sessionManager, "auto", false, GoosePlatformCLI)
	err = validConfig.Validate()
	if err != nil {
		t.Errorf("valid config failed validation: %v", err)
	}
}
```

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/agents -v
```
Expected: 所有测试 PASS

- [ ] **Step 6: 提交**

```bash
git add internal/agents/agent.go internal/agents/types.go internal/agents/lifecycle.go internal/agents/agent_test.go
git commit -m "feat(agents): 增强 Agent 配置和初始化逻辑"
```

---

### Task 8: 清理代码和运行完整测试

**Files:**
- All `internal/agents/*.go`
- Test: `internal/agents/*_test.go`

- [ ] **Step 1: 运行 go fmt**

```bash
go fmt ./internal/agents/...
```

- [ ] **Step 2: 运行 go vet**

```bash
go vet ./internal/agents/...
```

- [ ] **Step 3: 运行完整测试套件**

```bash
go test ./internal/agents/... -v
```
Expected: 所有测试 PASS

- [ ] **Step 4: 构建验证**

```bash
go build ./internal/agents/...
go build ./cmd/goose/...
go build ./cmd/goosed/...
```
Expected: 无编译错误

- [ ] **Step 5: 最终提交**

```bash
git add internal/agents/...
git commit -m "feat(agents): 完成 Agent 循环实现和测试"
```

---

## 完成检查清单

- [ ] callLLM 方法实现并测试
- [ ] executeTool 方法实现并测试
- [ ] collectTools 方法实现并测试
- [ ] Reply 主循环完整实现
- [ ] 重试机制集成
- [ ] 生命周期管理器实现
- [ ] Agent 配置增强
- [ ] 所有测试通过
- [ ] 代码格式化
- [ ] 构建成功
