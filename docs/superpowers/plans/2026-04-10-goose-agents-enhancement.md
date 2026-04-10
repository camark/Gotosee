# Goose Agents Enhancement Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance the Go gogo agents module with missing features from Rust goose original implementation

**Architecture:** Build upon existing agent.go core with enhanced event streaming, tool confirmation routing, and permission management

**Tech Stack:** Go 1.22+, sync primitives, channel-based concurrency

---

### Task 1: Tool Confirmation Router

**Files:**
- Create: `internal/agents/tool_confirmation_router.go`
- Test: `internal/agents/tool_confirmation_router_test.go`

- [ ] **Step 1: Create tool confirmation router type**

```go
// Package agents 提供工具调用确认路由功能。
package agents

import (
	"sync"
	
	"github.com/aaif-goose/gogo/internal/permission"
)

// ToolConfirmationRouter 工具调用确认路由器。
type ToolConfirmationRouter struct {
	mu           sync.Mutex
	confirmations map[string]chan PermissionConfirmation
}

// PermissionConfirmation 权限确认。
type PermissionConfirmation struct {
	Permission permission.Permission
}

// NewToolConfirmationRouter 创建工具确认路由器。
func NewToolConfirmationRouter() *ToolConfirmationRouter {
	return &ToolConfirmationRouter{
		confirmations: make(map[string]chan PermissionConfirmation),
	}
}

// Register 注册一个工具调用请求的确认通道。
func (r *ToolConfirmationRouter) Register(requestID string) <-chan PermissionConfirmation {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	ch := make(chan PermissionConfirmation, 1)
	r.confirmations[requestID] = ch
	return ch
}

// Deliver 传递确认到指定请求。
func (r *ToolConfirmationRouter) Deliver(requestID string, confirmation PermissionConfirmation) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	ch, exists := r.confirmations[requestID]
	if !exists {
		return false
	}
	
	select {
	case ch <- confirmation:
		delete(r.confirmations, requestID)
		close(ch)
		return true
	default:
		return false
	}
}
```

- [ ] **Step 2: Create tests**

```go
package agents

import (
	"testing"
	"github.com/aaif-goose/gogo/internal/permission"
)

func TestToolConfirmationRouter_RegisterAndDeliver(t *testing.T) {
	router := NewToolConfirmationRouter()
	
	ch := router.Register("req-123")
	
	confirmation := PermissionConfirmation{
		Permission: permission.AllowOnce,
	}
	
	delivered := router.Deliver("req-123", confirmation)
	if !delivered {
		t.Fatal("Failed to deliver confirmation")
	}
	
	received := <-ch
	if received.Permission != permission.AllowOnce {
		t.Errorf("Expected AllowOnce, got %v", received.Permission)
	}
}

func TestToolConfirmationRouter_DeliverToNonExistent(t *testing.T) {
	router := NewToolConfirmationRouter()
	
	delivered := router.Deliver("non-existent", PermissionConfirmation{})
	if delivered {
		t.Error("Should not deliver to non-existent request")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/agents/tool_confirmation_router_test.go ./internal/agents/tool_confirmation_router.go -v
```
Expected: 2/2 tests passing

- [ ] **Step 4: Integrate with Agent**

Modify `internal/agents/agent.go`:
- Add `toolConfirmationRouter *ToolConfirmationRouter` field to Agent struct
- Initialize in `NewAgentWithConfig`: `toolConfirmationRouter: NewToolConfirmationRouter()`

- [ ] **Step 5: Commit**

```bash
git add internal/agents/tool_confirmation_router.go internal/agents/tool_confirmation_router_test.go internal/agents/agent.go
git commit -m "feat: add tool confirmation router for permission management"
```

### Task 2: Enhanced Agent Event Streaming

**Files:**
- Modify: `internal/agents/agent.go:92-229`
- Create: `internal/agents/reply_parts.go`

- [ ] **Step 1: Extract reply context preparation**

```go
// ReplyContext 回复上下文。
type ReplyContext struct {
	Conversation     []*conversation.Message
	Tools            []*mcp.Tool
	SystemPrompt     string
	GooseMode        GooseMode
	ToolCallCutoff   int
	InitialMessages  []*conversation.Message
}

func (a *Agent) prepareReplyContext(
	sessionID string,
	messages []*conversation.Message,
	workingDir string,
) (*ReplyContext, error) {
	// Get tools
	tools := a.collectTools()
	
	// Get provider
	provider := a.provider.Get()
	if provider == nil {
		return nil, fmt.Errorf("provider not configured")
	}
	
	// Get model config
	modelConfig := provider.GetModelConfig()
	
	// Compute tool call cutoff
	contextLimit := modelConfig.ContextLimit
	if contextLimit == 0 {
		contextLimit = DEFAULT_CONTEXT_LIMIT
	}
	compactionThreshold := DEFAULT_COMPACTION_THRESHOLD
	toolCallCutoff := computeToolCallCutoff(contextLimit, compactionThreshold)
	
	return &ReplyContext{
		Conversation:    messages,
		Tools:           tools,
		SystemPrompt:    a.buildSystemPrompt(workingDir),
		GooseMode:       a.GetGooseMode(),
		ToolCallCutoff:  toolCallCutoff,
		InitialMessages: messages,
	}, nil
}

func computeToolCallCutoff(contextLimit, compactionThreshold int) int {
	// Simplified computation - adjust based on model context
	return contextLimit / 4
}

const DEFAULT_CONTEXT_LIMIT = 128000
```

- [ ] **Step 2: Categorize tool requests**

```go
// ToolCategorizeResult 工具分类结果。
type ToolCategorizeResult struct {
	FrontendRequests []*ToolRequest
	RemainingRequests []*ToolRequest
	FilteredResponse *conversation.Message
}

func (a *Agent) categorizeToolRequests(
	response *conversation.Message,
	tools []*mcp.Tool,
) *ToolCategorizeResult {
	var frontendRequests []*ToolRequest
	var remainingRequests []*ToolRequest
	
	// Build tool name set
	frontendTools := make(map[string]bool)
	// Check if tool is frontend tool
	// For now, all MCP tools are non-frontend
	
	for _, content := range response.Content {
		if content.Type == conversation.MessageContentToolUse {
			request := &ToolRequest{
				ID: content.ToolID,
				ToolCall: ToolCall{
					Name:      content.ToolName,
					Arguments: content.ToolArgs,
				},
			}
			
			if frontendTools[content.ToolName] {
				frontendRequests = append(frontendRequests, request)
			} else {
				remainingRequests = append(remainingRequests, request)
			}
		}
	}
	
	return &ToolCategorizeResult{
		FrontendRequests: frontendRequests,
		RemainingRequests: remainingRequests,
		FilteredResponse: response,
	}
}
```

- [ ] **Step 3: Refactor Reply method to use new structure**

- [ ] **Step 4: Commit**

```bash
git add internal/agents/reply_parts.go internal/agents/agent.go
git commit -m "feat: extract reply context preparation and tool categorization"
```

### Task 3: Permission Manager Integration

**Files:**
- Create: `internal/permission/permission.go`
- Modify: `internal/agents/agent.go`

- [ ] **Step 1: Create permission types**

```go
// Package permission 提供权限管理功能。
package permission

// Permission 权限级别。
type Permission string

const (
	// AllowOnce 允许一次。
	AllowOnce Permission = "allow_once"
	// AlwaysAllow 总是允许。
	AlwaysAllow Permission = "always_allow"
	// DenyOnce 拒绝一次。
	DenyOnce Permission = "deny_once"
	// AlwaysDeny 总是拒绝。
	AlwaysDeny Permission = "always_deny"
)

// IsValid 检查权限级别是否有效。
func (p Permission) IsValid() bool {
	switch p {
	case AllowOnce, AlwaysAllow, DenyOnce, AlwaysDeny:
		return true
	}
	return false
}

// PermissionLevel 权限级别（用于持久化）。
type PermissionLevel string

const (
	PermissionLevelAsk       PermissionLevel = "ask"
	PermissionLevelAlwaysAllow PermissionLevel = "always_allow"
	PermissionLevelNeverAllow PermissionLevel = "never_allow"
)
```

- [ ] **Step 2: Create permission manager**

```go
// PermissionManager 权限管理器。
type PermissionManager struct {
	mu          sync.RWMutex
	permissions map[string]PermissionLevel // tool_name -> level
}

func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		permissions: make(map[string]PermissionLevel),
	}
}

func (pm *PermissionManager) GetPermission(toolName string) PermissionLevel {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.permissions[toolName]
}

func (pm *PermissionManager) SetPermission(toolName string, level PermissionLevel) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.permissions[toolName] = level
}
```

- [ ] **Step 3: Integrate with Agent**

- [ ] **Step 4: Commit**

```bash
git add internal/permission/permission.go internal/agents/agent.go
git commit -m "feat: add permission manager for tool access control"
```

### Task 4: Enhanced Stream Response Support

**Files:**
- Modify: `internal/providers/base.go` (add stream interface)
- Modify: `internal/agents/agent.go` (add stream handling)

- [ ] **Step 1: Add stream chunk type**

```go
// StreamChunk 流式响应块。
type StreamChunk struct {
	Text string
	ToolCalls []ToolCall
	Done bool
	Err error
}

// Provider interface enhancement
type Provider interface {
	// ... existing methods ...
	Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error)
}
```

- [ ] **Step 2: Implement stream handling in agent**

- [ ] **Step 3: Update OpenAI provider with proper stream**

- [ ] **Step 4: Commit**

```bash
git add internal/providers/base.go internal/providers/openai.go internal/agents/agent.go
git commit -m "feat: enhance stream response support in agent loop"
```

### Task 5: Extension State Persistence

**Files:**
- Modify: `internal/agents/extension.go`
- Modify: `internal/session/session.go`

- [ ] **Step 1: Add extension state types**

```go
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

func NewEnabledExtensionsState(configs []*ExtensionConfig) *EnabledExtensionsState {
	states := make([]ExtensionState, len(configs))
	for i, config := range configs {
		states[i] = ExtensionState{
			Name:    config.Name,
			Config:  config.ConfigData,
			Enabled: config.Enabled,
		}
	}
	return &EnabledExtensionsState{Extensions: states}
}
```

- [ ] **Step 2: Add serialization methods**

- [ ] **Step 3: Integrate with session persistence**

- [ ] **Step 4: Commit**

```bash
git add internal/agents/extension.go internal/session/session.go
git commit -m "feat: add extension state persistence to session"
```

### Task 6: Message Compaction

**Files:**
- Create: `internal/agents/compaction.go`
- Modify: `internal/agents/agent.go`

- [ ] **Step 1: Create compaction logic**

```go
// Check if compaction is needed
func needsCompaction(messages []*conversation.Message, threshold int) bool {
	return len(messages) > threshold
}

// Compact messages - keep system messages and recent conversation
func compactMessages(messages []*conversation.Message) []*conversation.Message {
	// Keep first system message
	// Keep last N messages (based on threshold)
	// Add compaction summary placeholder
}
```

- [ ] **Step 2: Integrate compaction into reply loop**

- [ ] **Step 3: Commit**

```bash
git add internal/agents/compaction.go internal/agents/agent.go
git commit -m "feat: add message compaction for long conversations"
```

### Task 7: Final Output Tool

**Files:**
- Create: `internal/agents/final_output_tool.go`

- [ ] **Step 1: Create final output tool**

```go
// FinalOutputTool 最终输出工具。
type FinalOutputTool struct {
	response Response
}

func NewFinalOutputTool(response Response) *FinalOutputTool {
	return &FinalOutputTool{response: response}
}

func (t *FinalOutputTool) Execute(args map[string]interface{}) (*ToolCallResult, error) {
	// Store final output
	// Signal completion
}
```

- [ ] **Step 2: Integrate with agent lifecycle**

- [ ] **Step 3: Commit**

```bash
git add internal/agents/final_output_tool.go internal/agents/agent.go
git commit -m "feat: add final output tool for structured responses"
```

### Task 8: Integration Tests

**Files:**
- Create: `internal/agents/agent_integration_test.go`

- [ ] **Step 1: Create agent integration test**

```go
func TestAgentReplyFlow(t *testing.T) {
	// Create agent
	// Set up mock provider
	// Send message
	// Verify event stream
}
```

- [ ] **Step 2: Run all tests**

```bash
go test ./internal/agents/... -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/agents/agent_integration_test.go
git commit -m "test: add agent integration tests"
```
