// Package providers 提供 Qwen 提供商实现。
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

// QwenProvider Qwen 提供商实现（通义千问 - 阿里云）。
type QwenProvider struct {
	BaseProvider
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// QwenConfig Qwen 配置。
type QwenConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// DefaultQwenBaseURL 默认 Qwen API 地址（阿里云 DashScope）。
const DefaultQwenBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

// DefaultQwenModel 默认 Qwen 模型。
const DefaultQwenModel = "qwen-plus"

// KnownQwenModels 已知的 Qwen 模型。
var KnownQwenModels = map[string]int{
	"qwen-turbo":       131072,
	"qwen-plus":        131072,
	"qwen-max":         131072,
	"qwen-max-longcontext": 28000,
	"qwen2.5-72b-instruct": 131072,
	"qwen2.5-32b-instruct": 131072,
	"qwen2.5-14b-instruct": 131072,
	"qwen2.5-7b-instruct":  131072,
}

// NewQwenProvider 创建新的 Qwen 提供商。
func NewQwenProvider(config QwenConfig, modelConfig model.ModelConfig) *QwenProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultQwenBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &QwenProvider{
		BaseProvider: NewBaseProvider("qwen", "Qwen", "阿里云通义千问 Qwen 模型", modelConfig),
		apiKey:       config.APIKey,
		baseURL:      baseURL + "/chat/completions",
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Name 返回提供商名称。
func (p *QwenProvider) Name() string {
	return "qwen"
}

// GetModelConfig 获取当前模型配置。
func (p *QwenProvider) GetModelConfig() model.ModelConfig {
	return p.BaseProvider.Config()
}

// Validate 验证配置。
func (p *QwenProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *QwenProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	if err := p.Validate(); err != nil {
		return conversation.Message{}, err
	}

	reqBody, err := p.buildCompletionRequest(messages, config)
	if err != nil {
		return conversation.Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return conversation.Message{}, err
	}

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

	var apiResp QwenCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return conversation.Message{}, err
	}

	return p.responseToMessage(apiResp), nil
}

// Stream 执行流式请求。
func (p *QwenProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	ch := make(chan StreamChunk, 32)

	reqBody, err := p.buildCompletionRequest(messages, config)
	if err != nil {
		close(ch)
		return nil, err
	}

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

	p.setRequestHeaders(req)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		close(ch)
		return nil, err
	}

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

			var chunkData QwenCompletionResponse
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
func (p *QwenProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	result := make([]ModelInfo, 0, len(KnownQwenModels))
	for modelID, contextWindow := range KnownQwenModels {
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

func (p *QwenProvider) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func (p *QwenProvider) buildCompletionRequest(messages []conversation.Message, config model.ModelConfig) ([]byte, error) {
	apiMessages := make([]QwenMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = p.messageToAPI(msg)
	}

	model := config.Model
	if model == "" {
		model = DefaultQwenModel
	}

	req := QwenCompletionRequest{
		Model:       model,
		Messages:    apiMessages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
	}

	return json.Marshal(req)
}

func (p *QwenProvider) messageToAPI(msg conversation.Message) QwenMessage {
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

	return QwenMessage{
		Role:    role,
		Content: content,
	}
}

func (p *QwenProvider) responseToMessage(resp QwenCompletionResponse) conversation.Message {
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
// Qwen API 类型定义
// ============================================================================

type QwenCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []QwenMessage   `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type QwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type QwenCompletionResponse struct {
	ID      string          `json:"id"`
	Model   string          `json:"model"`
	Choices []QwenChoice    `json:"choices"`
	Usage   QwenUsage       `json:"usage"`
	Created int64           `json:"created"`
}

type QwenChoice struct {
	Index        int           `json:"index"`
	Message      QwenMessage   `json:"message"`
	Delta        QwenDelta     `json:"delta,omitempty"`
	FinishReason string        `json:"finish_reason"`
}

type QwenDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type QwenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
