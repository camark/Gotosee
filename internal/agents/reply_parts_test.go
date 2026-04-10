package agents

import (
	"context"
	"errors"
	"testing"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/model"
	"github.com/camark/Gotosee/internal/providers"
)

var (
	ErrStreamFailed       = errors.New("stream failed")
	ErrStreamNotSupported = errors.New("stream not supported")
)

// mockStreamProvider 支持流式的模拟提供商
type mockStreamProvider struct {
	chunks []providers.StreamChunk
}

func (m *mockStreamProvider) Name() string        { return "mock-stream" }
func (m *mockStreamProvider) Description() string { return "mock stream provider" }
func (m *mockStreamProvider) Validate() error     { return nil }
func (m *mockStreamProvider) GetModelConfig() model.ModelConfig {
	return model.ModelConfig{ContextLimit: 128000}
}

func (m *mockStreamProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	return conversation.Message{
		Role: conversation.RoleAssistant,
		Content: []conversation.MessageContent{
			{Type: conversation.MessageContentText, Text: "Complete response"},
		},
	}, nil
}

func (m *mockStreamProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, len(m.chunks))
	for _, chunk := range m.chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func (m *mockStreamProvider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	return nil, nil
}

func TestStreamResponseFromProvider_BasicStream(t *testing.T) {
	agent := NewAgent()

	provider := &mockStreamProvider{
		chunks: []providers.StreamChunk{
			{Text: "Hello ", Done: false},
			{Text: "world", Done: false},
			{Done: true},
		},
	}

	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Test"},
			},
		},
	}

	ctx := context.Background()
	response, err := agent.streamResponseFromProvider(ctx, provider, messages, nil, "")
	if err != nil {
		t.Fatalf("streamResponseFromProvider failed: %v", err)
	}

	if len(response.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	if response.Content[0].Type != conversation.MessageContentText {
		t.Errorf("expected text content, got %s", response.Content[0].Type)
	}

	if response.Content[0].Text != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", response.Content[0].Text)
	}
}

func TestStreamResponseFromProvider_WithToolCall(t *testing.T) {
	agent := NewAgent()

	provider := &mockStreamProvider{
		chunks: []providers.StreamChunk{
			{Text: "Let me check...", Done: false},
			{
				ToolName: "test_tool",
				ToolArgs: `{"key": "value"}`,
				Done:     false,
			},
			{Done: true},
		},
	}

	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Run tool"},
			},
		},
	}

	ctx := context.Background()
	response, err := agent.streamResponseFromProvider(ctx, provider, messages, nil, "")
	if err != nil {
		t.Fatalf("streamResponseFromProvider failed: %v", err)
	}

	if len(response.Content) != 2 {
		t.Fatalf("expected 2 content items, got %d", len(response.Content))
	}

	// 第一个是文本
	if response.Content[0].Type != conversation.MessageContentText {
		t.Errorf("expected text, got %s", response.Content[0].Type)
	}

	// 第二个是工具调用
	if response.Content[1].Type != conversation.MessageContentToolUse {
		t.Errorf("expected tool use, got %s", response.Content[1].Type)
	}

	if response.Content[1].ToolName != "test_tool" {
		t.Errorf("expected 'test_tool', got '%s'", response.Content[1].ToolName)
	}
}

func TestStreamResponseFromProvider_StreamError(t *testing.T) {
	agent := NewAgent()

	provider := &mockStreamProvider{
		chunks: []providers.StreamChunk{
			{Err: ErrStreamFailed, Done: false},
		},
	}

	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Test"},
			},
		},
	}

	ctx := context.Background()
	_, err := agent.streamResponseFromProvider(ctx, provider, messages, nil, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStreamResponseFromProvider_FallbackToComplete(t *testing.T) {
	agent := NewAgent()

	// 创建一个 Stream 返回错误的提供商
	provider := &mockErrorStreamProvider{}

	messages := []*conversation.Message{
		{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: "Test"},
			},
		},
	}

	ctx := context.Background()
	response, err := agent.streamResponseFromProvider(ctx, provider, messages, nil, "")
	if err != nil {
		t.Fatalf("streamResponseFromProvider failed: %v", err)
	}

	if len(response.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	if response.Content[0].Text != "Complete response" {
		t.Errorf("expected 'Complete response', got '%s'", response.Content[0].Text)
	}
}

// mockErrorStreamProvider Stream 方法返回错误的模拟提供商
type mockErrorStreamProvider struct{}

func (m *mockErrorStreamProvider) Name() string                      { return "mock-error" }
func (m *mockErrorStreamProvider) Description() string               { return "mock error provider" }
func (m *mockErrorStreamProvider) Validate() error                   { return nil }
func (m *mockErrorStreamProvider) GetModelConfig() model.ModelConfig { return model.ModelConfig{} }

func (m *mockErrorStreamProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	return conversation.Message{
		Role: conversation.RoleAssistant,
		Content: []conversation.MessageContent{
			{Type: conversation.MessageContentText, Text: "Complete response"},
		},
	}, nil
}

func (m *mockErrorStreamProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan providers.StreamChunk, error) {
	return nil, ErrStreamNotSupported
}

func (m *mockErrorStreamProvider) ListModels(ctx context.Context) ([]providers.ModelInfo, error) {
	return nil, nil
}
