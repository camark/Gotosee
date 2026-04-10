// Package providers 提供 DeepSeek 提供商实现。
package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/model"
)

// DeepSeekProvider DeepSeek 提供商实现。
type DeepSeekProvider struct {
	BaseProvider
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// DeepSeekConfig DeepSeek 配置。
type DeepSeekConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// DefaultDeepSeekBaseURL 默认 DeepSeek API 地址。
const DefaultDeepSeekBaseURL = "https://api.deepseek.com"

// DefaultDeepSeekEndpoint DeepSeek API 端点。
const DefaultDeepSeekEndpoint = "/v1/chat/completions"

// DefaultDeepSeekModel 默认 DeepSeek 模型。
const DefaultDeepSeekModel = "deepseek-chat"

// KnownDeepSeekModels 已知的 DeepSeek 模型。
var KnownDeepSeekModels = map[string]int{
	"deepseek-chat":     128000,
	"deepseek-coder":    128000,
	"deepseek-reasoner": 128000,
}

// NewDeepSeekProvider 创建新的 DeepSeek 提供商。
func NewDeepSeekProvider(config DeepSeekConfig, modelConfig model.ModelConfig) *DeepSeekProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultDeepSeekBaseURL
	}
	// 移除末尾的斜杠
	baseURL = strings.TrimSuffix(baseURL, "/")
	// 添加 API 端点
	apiURL := baseURL + DefaultDeepSeekEndpoint

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &DeepSeekProvider{
		BaseProvider: NewBaseProvider("deepseek", "DeepSeek", "DeepSeek AI 模型提供商", modelConfig),
		apiKey:       config.APIKey,
		baseURL:      apiURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Name 返回提供商名称。
func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

// GetModelConfig 获取当前模型配置。
func (p *DeepSeekProvider) GetModelConfig() model.ModelConfig {
	return p.BaseProvider.Config()
}

// Validate 验证配置。
func (p *DeepSeekProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *DeepSeekProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	if err := p.Validate(); err != nil {
		return conversation.Message{}, err
	}

	// 构建请求
	reqBody, err := p.buildCompletionRequest(messages, config)
	if err != nil {
		return conversation.Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return conversation.Message{}, err
	}

	// 设置请求头
	p.setRequestHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return conversation.Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return conversation.Message{}, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var apiResp DeepSeekCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return conversation.Message{}, err
	}

	// 转换为 Message
	return p.responseToMessage(apiResp), nil
}

// Stream 执行流式请求。
func (p *DeepSeekProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	ch := make(chan StreamChunk, 32)

	// 构建流式请求
	reqBody, err := p.buildCompletionRequest(messages, config)
	if err != nil {
		close(ch)
		return nil, err
	}

	// 修改为流式模式
	var streamReq map[string]interface{}
	if err := json.Unmarshal(reqBody, &streamReq); err != nil {
		close(ch)
		return nil, err
	}
	streamReq["stream"] = true

	reqBody, err = json.Marshal(streamReq)
	if err != nil {
		close(ch)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		close(ch)
		return nil, err
	}

	// 设置请求头
	p.setRequestHeaders(req)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		close(ch)
		return nil, err
	}

	// 启动 goroutine 处理流式响应
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				ch <- StreamChunk{Done: true, Err: ctx.Err()}
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					ch <- StreamChunk{Done: true}
					return
				}
				ch <- StreamChunk{Done: true, Err: err}
				return
			}

			lineStr := strings.TrimSpace(string(line))
			if !strings.HasPrefix(lineStr, "data: ") {
				continue
			}

			dataStr := strings.TrimPrefix(lineStr, "data: ")
			if dataStr == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}

			// 解析 SSE 数据
			var chunkData DeepSeekCompletionResponse
			if err := json.Unmarshal([]byte(dataStr), &chunkData); err != nil {
				continue
			}

			if len(chunkData.Choices) > 0 {
				delta := chunkData.Choices[0].Delta.Content
				if delta != "" {
					ch <- StreamChunk{Text: delta}
				}
			}
		}
	}()

	return ch, nil
}

// ListModels 列出支持的模型。
func (p *DeepSeekProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	result := make([]ModelInfo, 0, len(KnownDeepSeekModels))
	for modelID, contextWindow := range KnownDeepSeekModels {
		result = append(result, ModelInfo{
			ID:             modelID,
			Name:           modelID,
			ContextWindow:  contextWindow,
			SupportsStream: true,
			SupportsTools:  true,
		})
	}
	return result, nil
}

// setRequestHeaders 设置请求头。
func (p *DeepSeekProvider) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

// buildCompletionRequest 构建完成请求体。
func (p *DeepSeekProvider) buildCompletionRequest(messages []conversation.Message, config model.ModelConfig) ([]byte, error) {
	// 转换消息格式
	apiMessages := make([]DeepSeekMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = p.messageToAPI(msg)
	}

	model := config.Model
	if model == "" {
		model = DefaultDeepSeekModel
	}

	req := DeepSeekCompletionRequest{
		Model:       model,
		Messages:    apiMessages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		Stream:      false,
	}

	return json.Marshal(req)
}

// messageToAPI 转换 Message 为 DeepSeek 格式。
func (p *DeepSeekProvider) messageToAPI(msg conversation.Message) DeepSeekMessage {
	var role string
	switch msg.Role {
	case conversation.RoleUser:
		role = "user"
	case conversation.RoleAssistant:
		role = "assistant"
	case conversation.RoleSystem:
		role = "system"
	default:
		role = "user"
	}

	content := ""
	for _, c := range msg.Content {
		if c.Type == conversation.MessageContentText {
			content += c.Text
		}
	}

	return DeepSeekMessage{
		Role:    role,
		Content: content,
	}
}

// responseToMessage 转换 DeepSeek 响应为 Message。
func (p *DeepSeekProvider) responseToMessage(resp DeepSeekCompletionResponse) conversation.Message {
	content := conversation.MessageContent{
		Type: conversation.MessageContentText,
	}

	if len(resp.Choices) > 0 {
		content.Text = resp.Choices[0].Message.Content
	}

	return conversation.Message{
		Role: conversation.RoleAssistant,
		Content: []conversation.MessageContent{content},
		Metadata: map[string]interface{}{
			"model": resp.Model,
			"usage": resp.Usage,
		},
	}
}

// ============================================================================
// DeepSeek API 类型定义
// ============================================================================

// DeepSeekCompletionRequest DeepSeek 完成请求。
type DeepSeekCompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []DeepSeekMessage `json:"messages"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

// DeepSeekMessage DeepSeek 消息格式。
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekCompletionResponse DeepSeek 完成响应。
type DeepSeekCompletionResponse struct {
	ID      string           `json:"id"`
	Model   string           `json:"model"`
	Choices []DeepSeekChoice `json:"choices"`
	Usage   DeepSeekUsage    `json:"usage"`
}

// DeepSeekChoice DeepSeek 响应选择。
type DeepSeekChoice struct {
	Index        int             `json:"index"`
	Message      DeepSeekMessage `json:"message"`
	Delta        DeepSeekDelta   `json:"delta,omitempty"`
	FinishReason string          `json:"finish_reason"`
}

// DeepSeekDelta DeepSeek 流式响应增量。
type DeepSeekDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// DeepSeekUsage DeepSeek 使用统计。
type DeepSeekUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
