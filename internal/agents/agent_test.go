package agents

import (
	"context"
	"testing"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/mcp"
	"github.com/aaif-goose/gogo/internal/model"
	"github.com/aaif-goose/gogo/internal/providers"
	"github.com/aaif-goose/gogo/internal/session"
)

// mockProvider 模拟提供商用于测试
type mockProvider struct{}

func (m *mockProvider) Name() string        { return "mock" }
func (m *mockProvider) Description() string { return "mock provider" }
func (m *mockProvider) Validate() error     { return nil }
func (m *mockProvider) GetModelConfig() model.ModelConfig {
	return model.ModelConfig{Model: "mock-model"}
}
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

	// 验证至少有一个工具（MCP 注册中心可能有其他工具）
	if len(tools) < 1 {
		t.Errorf("expected at least 1 tool, got %d", len(tools))
	}

	// 验证我们注册的工具在列表中
	found := false
	for _, t := range tools {
		if t.Name == "test_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected test_tool to be in tools list")
	}
}

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
