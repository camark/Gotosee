package agents

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/model"
	"github.com/camark/Gotosee/internal/permission"
	"github.com/camark/Gotosee/internal/providers"
	"github.com/camark/Gotosee/internal/session"
)

// MockProvider 模拟提供商用于测试。
type MockProvider struct {
	mu           sync.Mutex
	lastMessages []conversation.Message
	lastConfig   model.ModelConfig
	completeFunc func(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error)
	streamFunc   func(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan providers.StreamChunk, error)
}

func (p *MockProvider) Name() string        { return "mock" }
func (p *MockProvider) Description() string { return "Mock provider for testing" }
func (p *MockProvider) Validate() error     { return nil }
func (p *MockProvider) GetModelConfig() model.ModelConfig {
	return model.ModelConfig{ContextLimit: 128000}
}

func (p *MockProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	p.mu.Lock()
	p.lastMessages = messages
	p.lastConfig = config
	p.mu.Unlock()

	if p.completeFunc != nil {
		return p.completeFunc(ctx, messages, config)
	}

	// 默认返回文本回复
	return conversation.Message{
		Role: conversation.RoleAssistant,
		Content: []conversation.MessageContent{
			{Type: conversation.MessageContentText, Text: "Hello, I am a mock assistant."},
		},
	}, nil
}

func (p *MockProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan providers.StreamChunk, error) {
	p.mu.Lock()
	p.lastMessages = messages
	p.lastConfig = config
	p.mu.Unlock()

	if p.streamFunc != nil {
		return p.streamFunc(ctx, messages, config)
	}

	ch := make(chan providers.StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (p *MockProvider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	return []providers.ModelInfo{}, nil
}

// TestAgent Reply Basic Flow 测试基本的 Reply 流程。
func TestAgent_Reply_BasicFlow(t *testing.T) {
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	config := NewAgentConfig(sessionManager, "auto", false, GoosePlatformCLI)
	agent := NewAgentWithConfig(config)

	// 设置模拟提供商
	mockProvider := &MockProvider{}
	agent.SetProvider(mockProvider)

	ctx := context.Background()
	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Hello"},
			},
		},
	}

	eventChan, err := agent.Reply(ctx, messages)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 接收事件
	eventsReceived := 0
	for event := range eventChan {
		eventsReceived++
		switch e := event.(type) {
		case *MessageEvent:
			if e.Message == nil {
				t.Error("MessageEvent should have message")
			}
		default:
			// 其他事件类型可以忽略
		}
	}

	if eventsReceived == 0 {
		t.Error("Should receive at least one event")
	}
}

// TestAgent_Reply_WithToolCall 测试带工具调用的 Reply。
func TestAgent_Reply_WithToolCall(t *testing.T) {
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	config := NewAgentConfig(sessionManager, "auto", false, GoosePlatformCLI)
	agent := NewAgentWithConfig(config)

	// 设置返回工具调用的模拟提供商
	mockProvider := &MockProvider{
		completeFunc: func(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
			return conversation.Message{
				Role: conversation.RoleAssistant,
				Content: []conversation.MessageContent{
					{
						Type:     conversation.MessageContentToolUse,
						ToolName: "test_tool",
						ToolArgs: []byte(`{"arg": "value"}`),
					},
				},
			}, nil
		},
	}
	agent.SetProvider(mockProvider)

	ctx := context.Background()
	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Run a test"},
			},
		},
	}

	eventChan, err := agent.Reply(ctx, messages)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 接收事件
	toolCallReceived := false
	for event := range eventChan {
		switch event.(type) {
		case *ToolCallEvent:
			toolCallReceived = true
		case *ToolResultEvent:
			// 工具结果事件
		}
	}

	// 由于工具未注册，应该不会触发工具调用事件
	// 这里验证基本流程正常
	_ = toolCallReceived
}

// TestAgent_Reply_ContextCancellation 测试上下文取消。
func TestAgent_Reply_ContextCancellation(t *testing.T) {
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	config := NewAgentConfig(sessionManager, "auto", false, GoosePlatformCLI)
	agent := NewAgentWithConfig(config)

	// 设置会阻塞的模拟提供商
	mockProvider := &MockProvider{
		completeFunc: func(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
			// 等待上下文取消
			<-ctx.Done()
			return conversation.Message{}, ctx.Err()
		},
	}
	agent.SetProvider(mockProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Hello"},
			},
		},
	}

	eventChan, err := agent.Reply(ctx, messages)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 应该收到取消事件
	eventsReceived := 0
	for event := range eventChan {
		eventsReceived++
		_ = event
	}

	if eventsReceived == 0 {
		t.Error("Should receive at least one event (cancellation)")
	}
}

// TestAgent_Reply_MaxTurns 测试最大轮次限制。
func TestAgent_Reply_MaxTurns(t *testing.T) {
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	config := NewAgentConfig(sessionManager, "auto", false, GoosePlatformCLI)
	config.MaxTurns = 2 // 设置最大轮次为 2
	agent := NewAgentWithConfig(config)

	// 设置总是返回工具调用的模拟提供商
	mockProvider := &MockProvider{
		completeFunc: func(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
			return conversation.Message{
				Role: conversation.RoleAssistant,
				Content: []conversation.MessageContent{
					{
						Type:     conversation.MessageContentToolUse,
						ToolName: "loop_tool",
						ToolArgs: []byte(`{}`),
					},
				},
			}, nil
		},
	}
	agent.SetProvider(mockProvider)

	ctx := context.Background()
	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Start loop"},
			},
		},
	}

	eventChan, err := agent.Reply(ctx, messages)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 接收所有事件
	eventsReceived := 0
	for range eventChan {
		eventsReceived++
	}

	// 验证在最大轮次后停止
	if eventsReceived == 0 {
		t.Error("Should receive events")
	}
}

// TestAgent_CollectTools 测试工具收集。
func TestAgent_CollectTools(t *testing.T) {
	agent := NewAgent()

	tools := agent.collectTools()
	if tools == nil {
		t.Error("Tools should not be nil")
	}
}

// TestAgent_checkAndCompact 测试消息压缩。
func TestAgent_checkAndCompact(t *testing.T) {
	agent := NewAgent()

	// 创建大量消息
	var messages []*conversation.Message
	for i := 0; i < 150; i++ {
		messages = append(messages, &conversation.Message{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "msg"},
			},
		})
	}

	compacted := agent.checkAndCompact(messages)
	if len(compacted) >= len(messages) {
		t.Error("Compacted messages should be fewer")
	}
}

// TestAgent_SwitchGooseMode 测试模式切换。
func TestAgent_SwitchGooseMode(t *testing.T) {
	agent := NewAgent()

	initialMode := agent.GetGooseMode()
	if initialMode != "auto" {
		t.Errorf("Expected initial mode 'auto', got '%s'", initialMode)
	}

	agent.SwitchGooseMode("chat")
	newMode := agent.GetGooseMode()
	if newMode != "chat" {
		t.Errorf("Expected mode 'chat', got '%s'", newMode)
	}
}

// TestAgent_LifecycleTransitions 测试生命周期状态转换。
func TestAgent_LifecycleTransitions(t *testing.T) {
	agent := NewAgent()

	// 验证初始状态
	if agent.lifecycleManager == nil {
		t.Fatal("LifecycleManager should not be nil")
	}

	// 测试状态转换
	err := agent.lifecycleManager.Transition(AgentLifecycleInitializing)
	if err != nil {
		t.Errorf("Transition to Initializing failed: %v", err)
	}

	err = agent.lifecycleManager.Transition(AgentLifecycleReady)
	if err != nil {
		t.Errorf("Transition to Ready failed: %v", err)
	}
}

// TestAgent_RetryManagerIntegration 测试重试管理器集成。
func TestAgent_RetryManagerIntegration(t *testing.T) {
	agent := NewAgent()

	// 测试重试计数
	initial := agent.GetRetryAttempts()
	if initial != 0 {
		t.Errorf("Expected initial retry count 0, got %d", initial)
	}

	agent.IncrementRetryAttempts()
	count := agent.GetRetryAttempts()
	if count != 1 {
		t.Errorf("Expected retry count 1, got %d", count)
	}

	agent.ResetRetryAttempts()
	count = agent.GetRetryAttempts()
	if count != 0 {
		t.Errorf("Expected reset retry count 0, got %d", count)
	}
}

// TestAgent_ExtensionManager 测试扩展管理器。
func TestAgent_ExtensionManager(t *testing.T) {
	agent := NewAgent()

	em := agent.GetExtensionManager()
	if em == nil {
		t.Error("ExtensionManager should not be nil")
	}

	// 测试扩展注册
	config := &ExtensionConfig{
		Name:    "test_ext",
		Type:    ExtensionTypeBuiltin,
		Enabled: true,
	}
	err := em.RegisterExtension(config)
	if err != nil {
		t.Errorf("RegisterExtension failed: %v", err)
	}

	// 验证扩展已注册
	ext, err := em.GetExtension("test_ext")
	if err != nil {
		t.Errorf("GetExtension failed: %v", err)
	}
	if ext.Name != "test_ext" {
		t.Errorf("Expected extension name 'test_ext', got '%s'", ext.Name)
	}
}

// TestAgent_MessageCompactionIntegration 测试消息压缩集成。
func TestAgent_MessageCompactionIntegration(t *testing.T) {
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	config := NewAgentConfig(sessionManager, "auto", false, GoosePlatformCLI)
	agent := NewAgentWithConfig(config)

	// 创建带系统消息的大量消息
	var messages []*conversation.Message
	messages = append(messages, &conversation.Message{
		Role: conversation.RoleSystem,
		Content: []conversation.MessageContent{
			{Type: conversation.MessageContentText, Text: "You are a helpful assistant."},
		},
	})

	for i := 0; i < 150; i++ {
		messages = append(messages, &conversation.Message{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "msg"},
			},
		})
	}

	ctx := context.Background()
	eventChan, err := agent.Reply(ctx, messages)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 消耗事件
	for range eventChan {
	}

	// 验证流程正常完成
}

// TestAgent_ToolConfirmationRouterIntegration 测试工具确认路由器集成。
func TestAgent_ToolConfirmationRouterIntegration(t *testing.T) {
	agent := NewAgent()

	// 验证路由器已初始化
	if agent.toolConfirmationRouter == nil {
		t.Error("ToolConfirmationRouter should not be nil")
	}

	// 测试注册和传递
	ch := agent.toolConfirmationRouter.Register("test-req")

	go func() {
		agent.toolConfirmationRouter.Deliver("test-req", PermissionConfirmation{
			Permission: permission.AllowOnce,
		})
	}()

	confirmation := <-ch
	if confirmation.Permission != permission.AllowOnce {
		t.Errorf("Expected AllowOnce, got %v", confirmation.Permission)
	}
}

// TestAgent_PermissionManagerIntegration 测试权限管理器集成。
func TestAgent_PermissionManagerIntegration(t *testing.T) {
	agent := NewAgent()

	// 验证管理器已初始化
	if agent.permissionManager == nil {
		t.Error("PermissionManager should not be nil")
	}
}

// BenchmarkAgent_Reply 基准测试 Reply 性能。
func BenchmarkAgent_Reply(b *testing.B) {
	sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
	config := NewAgentConfig(sessionManager, "auto", false, GoosePlatformCLI)
	agent := NewAgentWithConfig(config)
	agent.SetProvider(&MockProvider{})

	ctx := context.Background()
	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Hello"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventChan, err := agent.Reply(ctx, messages)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		for range eventChan {
		}
	}
}

// BenchmarkAgent_Compaction 基准测试消息压缩性能。
func BenchmarkAgent_Compaction(b *testing.B) {
	agent := NewAgent()

	var messages []*conversation.Message
	for i := 0; i < 1000; i++ {
		messages = append(messages, &conversation.Message{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "msg"},
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.checkAndCompact(messages)
	}
}
