// Package agents 提供 Agent Reply 功能的辅助类型和函数。
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/mcp"
	"github.com/aaif-goose/gogo/internal/model"
	"github.com/aaif-goose/gogo/internal/providers"
)

// ReplyContext 回复上下文。
type ReplyContext struct {
	Conversation    []*conversation.Message
	Tools           []*mcp.Tool
	SystemPrompt    string
	GooseMode       string
	ToolCallCutoff  int
	InitialMessages []*conversation.Message
}

// computeToolCallCutoff 计算工具调用截断阈值。
func computeToolCallCutoff(contextLimit, compactionThreshold int) int {
	// 使用简化的计算方法 - 保留上下文限制的 25% 用于工具调用
	return contextLimit / 4
}

const (
	// DEFAULT_CONTEXT_LIMIT 默认上下文限制。
	DEFAULT_CONTEXT_LIMIT = 128000
	// DEFAULT_COMPACTION_THRESHOLD 默认压缩阈值。
	DEFAULT_COMPACTION_THRESHOLD = 100
)

// prepareReplyContext 准备回复上下文。
func (a *Agent) prepareReplyContext(
	sessionID string,
	messages []*conversation.Message,
	workingDir string,
) (*ReplyContext, error) {
	// 获取工具
	tools := a.collectTools()

	// 获取提供商
	provider := a.provider.Get()
	if provider == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	// 计算工具调用截断
	contextLimit := DEFAULT_CONTEXT_LIMIT
	toolCallCutoff := computeToolCallCutoff(contextLimit, DEFAULT_COMPACTION_THRESHOLD)

	return &ReplyContext{
		Conversation:    messages,
		Tools:           tools,
		SystemPrompt:    a.buildSystemPrompt(workingDir),
		GooseMode:       a.GetGooseMode(),
		ToolCallCutoff:  toolCallCutoff,
		InitialMessages: messages,
	}, nil
}

// buildSystemPrompt 构建系统提示。
func (a *Agent) buildSystemPrompt(workingDir string) string {
	var sb strings.Builder

	sb.WriteString("You are goose, an AI coding assistant.")
	sb.WriteString("\n\n")

	// 添加扩展信息
	extensionsInfo := a.extensionManager.GetExtensionsInfo()
	if len(extensionsInfo) > 0 {
		sb.WriteString("## Available Extensions\n\n")
		for _, ext := range extensionsInfo {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", ext.Name, ext.Description))
		}
		sb.WriteString("\n")
	}

	// 添加工具信息
	tools := a.collectTools()
	sb.WriteString(fmt.Sprintf("You have access to %d tools.\n", len(tools)))
	sb.WriteString("Use tools when needed to accomplish user requests.\n")
	sb.WriteString("Always explain your reasoning before calling tools.\n")

	return sb.String()
}

// ExtensionInfo 扩展信息。
type ExtensionInfo struct {
	Name        string
	Description string
}

// GetExtensionsInfo 获取扩展信息列表。
func (em *ExtensionManager) GetExtensionsInfo() []ExtensionInfo {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var result []ExtensionInfo
	for _, config := range em.extensions {
		result = append(result, ExtensionInfo{
			Name:        config.Name,
			Description: fmt.Sprintf("Extension of type %s", config.Type),
		})
	}
	return result
}

// ToolCategorizeResult 工具分类结果。
type ToolCategorizeResult struct {
	FrontendRequests  []*ToolRequest
	RemainingRequests []*ToolRequest
	FilteredResponse  *conversation.Message
}

// ToolRequest 工具请求。
type ToolRequest struct {
	ID       string
	ToolCall ToolCall
	Metadata map[string]interface{}
}

// categorizeToolRequests 分类工具请求。
func (a *Agent) categorizeToolRequests(
	response *conversation.Message,
	tools []*mcp.Tool,
) *ToolCategorizeResult {
	var frontendRequests []*ToolRequest
	var remainingRequests []*ToolRequest

	// 构建前端工具名称集合
	frontendTools := make(map[string]bool)
	a.frontendTools.Range(func(key, value interface{}) bool {
		if name, ok := key.(string); ok {
			frontendTools[name] = true
		}
		return true
	})

	for _, content := range response.Content {
		if content.Type == conversation.MessageContentToolUse {
			var args map[string]interface{}
			if content.ToolArgs != nil {
				json.Unmarshal(content.ToolArgs, &args)
			}

			request := &ToolRequest{
				ID:       fmt.Sprintf("tool-%s", content.ToolName),
				ToolCall: ToolCall{Name: content.ToolName, Arguments: args},
				Metadata: make(map[string]interface{}),
			}

			if frontendTools[content.ToolName] {
				frontendRequests = append(frontendRequests, request)
			} else {
				remainingRequests = append(remainingRequests, request)
			}
		}
	}

	return &ToolCategorizeResult{
		FrontendRequests:  frontendRequests,
		RemainingRequests: remainingRequests,
		FilteredResponse:  response,
	}
}

// streamResponseFromProvider 从提供商获取流式响应。
func (a *Agent) streamResponseFromProvider(
	ctx context.Context,
	provider providers.Provider,
	messages []*conversation.Message,
	tools []*mcp.Tool,
	systemPrompt string,
) (*conversation.Message, error) {
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

	// 尝试使用流式响应
	streamChan, err := provider.Stream(ctx, convMessages, config)
	if err != nil {
		// 如果流式不支持，回退到 Complete
		responseMsg, err := provider.Complete(ctx, convMessages, config)
		if err != nil {
			return nil, fmt.Errorf("provider complete failed: %w", err)
		}
		return &responseMsg, nil
	}

	// 收集流式响应块
	var textBuilder strings.Builder
	var toolCalls []conversation.MessageContent

	for chunk := range streamChan {
		if chunk.Err != nil {
			return nil, fmt.Errorf("stream error: %w", chunk.Err)
		}

		if chunk.Text != "" {
			textBuilder.WriteString(chunk.Text)
		}

		if chunk.ToolName != "" {
			toolCalls = append(toolCalls, conversation.MessageContent{
				Type:     conversation.MessageContentToolUse,
				ToolName: chunk.ToolName,
				ToolArgs: []byte(chunk.ToolArgs),
			})
		}

		if chunk.Done {
			break
		}
	}

	// 构建响应消息
	response := conversation.Message{
		Role:    conversation.RoleAssistant,
		Content: []conversation.MessageContent{},
	}

	if textBuilder.Len() > 0 {
		response.Content = append(response.Content, conversation.MessageContent{
			Type: conversation.MessageContentText,
			Text: textBuilder.String(),
		})
	}

	response.Content = append(response.Content, toolCalls...)

	return &response, nil
}
