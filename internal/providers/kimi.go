// Package providers 提供 Kimi 提供商实现。
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

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/model"
)

// KimiProvider Kimi 提供商实现（月之暗面）。
type KimiProvider struct {
	BaseProvider
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// KimiConfig Kimi 配置。
type KimiConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// DefaultKimiBaseURL 默认 Kimi API 地址。
const DefaultKimiBaseURL = "https://api.moonshot.cn"

// DefaultKimiModel 默认 Kimi 模型。
const DefaultKimiModel = "moonshot-v1-8k"

// KnownKimiModels 已知的 Kimi 模型。
var KnownKimiModels = map[string]int{
	"moonshot-v1-8k":   8192,
	"moonshot-v1-32k":  32768,
	"moonshot-v1-128k": 131072,
}

// NewKimiProvider 创建新的 Kimi 提供商。
func NewKimiProvider(config KimiConfig, modelConfig model.ModelConfig) *KimiProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultKimiBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &KimiProvider{
		BaseProvider: NewBaseProvider("kimi", "Kimi", "月之暗面 Kimi 智能助手", modelConfig),
		apiKey:       config.APIKey,
		baseURL:      baseURL + "/v1/chat/completions",
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Name 返回提供商名称。
func (p *KimiProvider) Name() string {
	return "kimi"
}

// GetModelConfig 获取当前模型配置。
func (p *KimiProvider) GetModelConfig() model.ModelConfig {
	return p.BaseProvider.Config()
}

// Validate 验证配置。
func (p *KimiProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *KimiProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
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

	var apiResp KimiCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return conversation.Message{}, err
	}

	return p.responseToMessage(apiResp), nil
}

// Stream 执行流式请求。
func (p *KimiProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
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

			var chunkData KimiCompletionResponse
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
func (p *KimiProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	result := make([]ModelInfo, 0, len(KnownKimiModels))
	for modelID, contextWindow := range KnownKimiModels {
		result = append(result, ModelInfo{
			ID:             modelID,
			Name:           modelID,
			ContextWindow:  contextWindow,
			SupportsStream: true,
			SupportsTools:  false,
		})
	}
	return result, nil
}

func (p *KimiProvider) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func (p *KimiProvider) buildCompletionRequest(messages []conversation.Message, config model.ModelConfig) ([]byte, error) {
	apiMessages := make([]KimiMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = p.messageToAPI(msg)
	}

	model := config.Model
	if model == "" {
		model = DefaultKimiModel
	}

	req := KimiCompletionRequest{
		Model:       model,
		Messages:    apiMessages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
	}

	return json.Marshal(req)
}

func (p *KimiProvider) messageToAPI(msg conversation.Message) KimiMessage {
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

	return KimiMessage{
		Role:    role,
		Content: content,
	}
}

func (p *KimiProvider) responseToMessage(resp KimiCompletionResponse) conversation.Message {
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
// Kimi API 类型定义
// ============================================================================

type KimiCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []KimiMessage   `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type KimiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type KimiCompletionResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []KimiChoice   `json:"choices"`
	Usage   KimiUsage      `json:"usage"`
	Created int64          `json:"created"`
}

type KimiChoice struct {
	Index        int           `json:"index"`
	Message      KimiMessage   `json:"message"`
	Delta        KimiDelta     `json:"delta,omitempty"`
	FinishReason string        `json:"finish_reason"`
}

type KimiDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type KimiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
