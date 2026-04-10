// Package providers 提供 AI 模型提供商接口和实现。
package providers

import (
	"context"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/model"
)

// Provider AI 提供商接口。
// 所有提供商必须实现此接口。
type Provider interface {
	// Name 返回提供商名称。
	Name() string

	// Description 返回提供商描述。
	Description() string

	// Validate 验证配置是否有效。
	Validate() error

	// Complete 执行完成请求，返回响应消息。
	Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error)

	// Stream 执行流式请求，返回响应通道。
	Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error)

	// ListModels 列出支持的模型。
	ListModels(ctx context.Context) ([]ModelInfo, error)

	// GetModelConfig 获取当前模型配置。
	GetModelConfig() model.ModelConfig
}

// StreamChunk 流式响应块。
type StreamChunk struct {
	// 文本内容
	Text string
	// 工具调用信息
	ToolName string
	// 工具参数
	ToolArgs string
	// 是否结束
	Done bool
	// 错误信息
	Err error
}

// ModelInfo 模型信息。
type ModelInfo struct {
	// 模型 ID
	ID string `json:"id"`
	// 模型名称
	Name string `json:"name"`
	// 模型描述
	Description string `json:"description,omitempty"`
	// 上下文窗口大小
	ContextWindow int `json:"context_window"`
	// 是否支持流式
	SupportsStream bool `json:"supports_stream"`
	// 是否支持工具调用
	SupportsTools bool `json:"supports_tools"`
}

// BaseProvider 提供商基础实现。
type BaseProvider struct {
	id          string
	label       string
	description string
	config      model.ModelConfig
}

// NewBaseProvider 创建基础提供商。
func NewBaseProvider(id, label, description string, config model.ModelConfig) BaseProvider {
	return BaseProvider{
		id:          id,
		label:       label,
		description: description,
		config:      config,
	}
}

// ID 返回提供商 ID。
func (b *BaseProvider) ID() string {
	return b.id
}

// Label 返回提供商标签。
func (b *BaseProvider) Label() string {
	return b.label
}

// Description 返回提供商描述。
func (b *BaseProvider) Description() string {
	return b.description
}

// Config 返回模型配置。
func (b *BaseProvider) Config() model.ModelConfig {
	return b.config
}

// GetModelConfig 获取当前模型配置。
func (b *BaseProvider) GetModelConfig() model.ModelConfig {
	return b.config
}

// SetConfig 设置模型配置。
func (b *BaseProvider) SetConfig(config model.ModelConfig) {
	b.config = config
}
