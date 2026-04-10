// Package model 提供 AI 模型配置功能。
package model

import (
	"encoding/json"
	"fmt"
)

// ModelConfig AI 模型配置。
type ModelConfig struct {
	// Provider ID
	Provider string `json:"provider"`
	// 模型名称
	Model string `json:"model"`
	// 上下文长度限制
	ContextLimit int `json:"context_limit,omitempty"`
	// 温度参数
	Temperature float64 `json:"temperature,omitempty"`
	// 最大 token 数
	MaxTokens int `json:"max_tokens,omitempty"`
	// 额外参数
	ExtraParams map[string]interface{} `json:"extra_params,omitempty"`
}

// DefaultModelConfig 返回默认的模型配置。
func DefaultModelConfig() ModelConfig {
	return ModelConfig{
		ContextLimit:  4096,
		Temperature:   0.7,
		MaxTokens:     2048,
		ExtraParams:   make(map[string]interface{}),
	}
}

// NewModelConfig 创建新的模型配置。
func NewModelConfig(provider, model string) ModelConfig {
	return ModelConfig{
		Provider:      provider,
		Model:         model,
		ContextLimit:  4096,
		Temperature:   0.7,
		MaxTokens:     2048,
		ExtraParams:   make(map[string]interface{}),
	}
}

// String 返回配置的字符串表示。
func (c *ModelConfig) String() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

// Validate 验证配置是否有效。
func (c *ModelConfig) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.ContextLimit <= 0 {
		return fmt.Errorf("context_limit must be positive")
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if c.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	return nil
}

// WithContextLimit 设置上下文限制。
func (c *ModelConfig) WithContextLimit(limit int) *ModelConfig {
	c.ContextLimit = limit
	return c
}

// WithTemperature 设置温度。
func (c *ModelConfig) WithTemperature(temp float64) *ModelConfig {
	c.Temperature = temp
	return c
}

// WithMaxTokens 设置最大 token 数。
func (c *ModelConfig) WithMaxTokens(tokens int) *ModelConfig {
	c.MaxTokens = tokens
	return c
}

// WithParam 设置额外参数。
func (c *ModelConfig) WithParam(key string, value interface{}) *ModelConfig {
	if c.ExtraParams == nil {
		c.ExtraParams = make(map[string]interface{})
	}
	c.ExtraParams[key] = value
	return c
}
