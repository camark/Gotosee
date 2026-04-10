// Package recipe 提供配方运行器功能。
package recipe

import (
	"context"
	"fmt"
	"strings"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/model"
	"github.com/camark/Gotosee/internal/providers"
)

// Runner 配方运行器。
type Runner struct {
	recipe   *Recipe
	provider providers.Provider
	config   model.ModelConfig
	settings map[string]any
}

// NewRunner 创建新的配方运行器。
func NewRunner(recipe *Recipe, provider providers.Provider, config model.ModelConfig) *Runner {
	return &Runner{
		recipe:   recipe,
		provider: provider,
		config:   config,
		settings: make(map[string]any),
	}
}

// SetSetting 设置配方参数。
func (r *Runner) SetSetting(key string, value any) {
	r.settings[key] = value
}

// GetSetting 获取配方参数。
func (r *Runner) GetSetting(key string) (any, bool) {
	val, ok := r.settings[key]
	if !ok {
		// 尝试从默认值获取
		return r.recipe.GetSettingDefault(key)
	}
	return val, ok
}

// Run 运行配方。
func (r *Runner) Run(ctx context.Context, userInput string) (string, error) {
	// 构建初始消息
	messages := r.buildInitialMessages(userInput)

	// 调用 AI
	response, err := r.provider.Complete(ctx, messages, r.config)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}

	// 提取文本内容
	var textContent strings.Builder
	for _, c := range response.Content {
		if c.Type == conversation.MessageContentText {
			textContent.WriteString(c.Text)
		}
	}

	return textContent.String(), nil
}

// RunWithSettings 使用设置运行配方。
func (r *Runner) RunWithSettings(ctx context.Context, userInput string, settings map[string]any) (string, error) {
	for k, v := range settings {
		r.SetSetting(k, v)
	}
	return r.Run(ctx, userInput)
}

// buildInitialMessages 构建初始消息。
func (r *Runner) buildInitialMessages(userInput string) []conversation.Message {
	var messages []conversation.Message

	// 添加系统提示（配方指令）
	systemPrompt := r.buildSystemPrompt()
	if systemPrompt != "" {
		messages = append(messages, conversation.NewTextMessage(conversation.RoleSystem, systemPrompt))
	}

	// 添加用户输入
	messages = append(messages, conversation.NewTextMessage(conversation.RoleUser, userInput))

	return messages
}

// buildSystemPrompt 构建系统提示。
func (r *Runner) buildSystemPrompt() string {
	var sb strings.Builder

	// 添加配方指令
	if r.recipe.Instructions != "" {
		sb.WriteString(r.recipe.Instructions)
		sb.WriteString("\n\n")
	}

	// 添加设置信息
	if len(r.settings) > 0 {
		sb.WriteString("Current settings:\n")
		for k, v := range r.settings {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ValidateSettings 验证设置是否完整。
func (r *Runner) ValidateSettings() error {
	for _, setting := range r.recipe.Settings {
		if setting.Required {
			if _, exists := r.settings[setting.Key]; !exists {
				// 检查是否有默认值
				if setting.Default == nil {
					return fmt.Errorf("required setting '%s' is missing", setting.Key)
				}
			}
		}
	}
	return nil
}

// GetRequiredSettings 获取所有必需的设置项。
func (r *Runner) GetRequiredSettings() []string {
	var required []string
	for _, setting := range r.recipe.Settings {
		if setting.Required && setting.Default == nil {
			if _, exists := r.settings[setting.Key]; !exists {
				required = append(required, setting.Key)
			}
		}
	}
	return required
}
