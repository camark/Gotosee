// Package providers 提供 AI 模型提供商接口和实现。
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

// OpenAIProvider OpenAI 提供商实现。
type OpenAIProvider struct {
	BaseProvider
	apiKey        string
	baseURL       string
	organization  string
	project       string
	customHeaders map[string]string
	httpClient    *http.Client
}

// OpenAIConfig OpenAI 配置。
type OpenAIConfig struct {
	APIKey        string            `json:"api_key"`
	BaseURL       string            `json:"base_url,omitempty"`
	Organization  string            `json:"organization,omitempty"`
	Project       string            `json:"project,omitempty"`
	Timeout       int               `json:"timeout,omitempty"`
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`
}

// DefaultOpenAIBaseURL 默认 OpenAI API 地址。
const DefaultOpenAIBaseURL = "https://api.openai.com/v1/chat/completions"

// DefaultOpenAIModel 默认 OpenAI 模型。
const DefaultOpenAIModel = "gpt-4o"

// DefaultOpenAIFastModel 默认快速 OpenAI 模型。
const DefaultOpenAIFastModel = "gpt-4o-mini"

// KnownOpenAIModels 已知的 OpenAI 模型及其上下文窗口。
var KnownOpenAIModels = map[string]int{
	"gpt-4o":         128000,
	"gpt-4o-mini":    128000,
	"gpt-4.1":        128000,
	"gpt-4.1-mini":   128000,
	"o1":             200000,
	"o3":             200000,
	"gpt-3.5-turbo":  16385,
	"gpt-4-turbo":    128000,
	"o4-mini":        128000,
	"gpt-5-nano":     400000,
	"gpt-5.1-codex":  400000,
	"gpt-5-codex":    400000,
}

// NewOpenAIProvider 创建新的 OpenAI 提供商。
func NewOpenAIProvider(config OpenAIConfig, modelConfig model.ModelConfig) *OpenAIProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultOpenAIBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &OpenAIProvider{
		BaseProvider: NewBaseProvider("openai", "OpenAI", "OpenAI GPT 模型提供商", modelConfig),
		apiKey:       config.APIKey,
		baseURL:      baseURL,
		organization: config.Organization,
		project:      config.Project,
		customHeaders: config.CustomHeaders,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Name 返回提供商名称。
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Validate 验证配置。
func (p *OpenAIProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *OpenAIProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
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

	var apiResp OpenAICompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return conversation.Message{}, err
	}

	// 转换为 Message
	return p.responseToMessage(apiResp), nil
}

// Stream 执行流式请求。
func (p *OpenAIProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
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
			var chunkData OpenAICompletionResponse
			if err := json.Unmarshal([]byte(dataStr), &chunkData); err != nil {
				continue
			}

			if len(chunkData.Choices) > 0 && len(chunkData.Choices[0].Delta.Content) > 0 {
				delta := chunkData.Choices[0].Delta.Content[0].Text
				if delta != "" {
					ch <- StreamChunk{Text: delta}
				}
			}
		}
	}()

	return ch, nil
}

// ListModels 列出支持的模型。
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// 使用已知模型列表
	result := make([]ModelInfo, 0, len(KnownOpenAIModels))
	for modelID, contextWindow := range KnownOpenAIModels {
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
func (p *OpenAIProvider) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	if p.organization != "" {
		req.Header.Set("OpenAI-Organization", p.organization)
	}
	if p.project != "" {
		req.Header.Set("OpenAI-Project", p.project)
	}

	for k, v := range p.customHeaders {
		req.Header.Set(k, v)
	}
}

// buildCompletionRequest 构建完成请求体。
func (p *OpenAIProvider) buildCompletionRequest(messages []conversation.Message, config model.ModelConfig) ([]byte, error) {
	// 转换消息格式
	apiMessages := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = p.messageToAPI(msg)
	}

	model := config.Model
	if model == "" {
		model = DefaultOpenAIModel
	}

	req := OpenAICompletionRequest{
		Model:       model,
		Messages:    apiMessages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		Stream:      false,
	}

	return json.Marshal(req)
}

// messageToAPI 转换 Message 为 OpenAI 格式。
func (p *OpenAIProvider) messageToAPI(msg conversation.Message) OpenAIMessage {
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

	return OpenAIMessage{
		Role:    role,
		Content: content,
	}
}

// responseToMessage 转换 OpenAI 响应为 Message。
func (p *OpenAIProvider) responseToMessage(resp OpenAICompletionResponse) conversation.Message {
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
// OpenAI API 类型定义
// ============================================================================

// OpenAICompletionRequest OpenAI 完成请求。
type OpenAICompletionRequest struct {
	Model       string         `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64        `json:"temperature,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Tools       []OpenAITool   `json:"tools,omitempty"`
}

// OpenAIMessage OpenAI 消息格式。
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAITool OpenAI 工具定义。
type OpenAITool struct {
	Type     string          `json:"type"`
	Function *OpenAIFunction `json:"function,omitempty"`
}

// OpenAIFunction OpenAI 函数定义。
type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OpenAICompletionResponse OpenAI 完成响应。
type OpenAICompletionResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice OpenAI 响应选择。
type Choice struct {
	Index        int              `json:"index"`
	Message      OpenAIMessage    `json:"message"`
	Delta        OpenAIDelta      `json:"delta,omitempty"`
	FinishReason string           `json:"finish_reason"`
}

// OpenAIDelta OpenAI 流式响应增量。
type OpenAIDelta struct {
	Content []ContentDelta `json:"content,omitempty"`
}

// ContentDelta 内容增量。
type ContentDelta struct {
	Text string `json:"text"`
}

// Usage OpenAI 使用统计。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAISSEEvent OpenAI SSE 事件。
type OpenAISSEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}
