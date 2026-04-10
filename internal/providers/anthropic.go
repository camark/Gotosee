// Package providers 提供 Anthropic 提供商实现。
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/model"
)

const (
	AnthropicProviderName  = "anthropic"
	AnthropicDefaultModel  = "claude-sonnet-4-5"
	AnthropicDefaultFastModel = "claude-haiku-4-5"
	AnthropicAPIVersion    = "2023-06-01"
	AnthropicBaseURL       = "https://api.anthropic.com"
)

// AnthropicProvider Anthropic 提供商。
type AnthropicProvider struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	model       *model.ModelConfig
	customModels []string
}

// anthropicMessage Anthropic 消息格式。
type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

// anthropicContent Anthropic 内容格式。
type anthropicContent struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	Thoughts  string `json:"thoughts,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   []anthropicContent `json:"content,omitempty"`
}

// anthropicRequest Anthropic 请求格式。
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

// anthropicTool Anthropic 工具格式。
type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

// anthropicResponse Anthropic 响应格式。
type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// NewAnthropicProvider 创建 Anthropic 提供商。
func NewAnthropicProvider(apiKey, baseURL string, modelConfig *model.ModelConfig) *AnthropicProvider {
	if baseURL == "" {
		baseURL = AnthropicBaseURL
	}
	if modelConfig == nil {
		modelConfig = &model.ModelConfig{
			Provider: AnthropicProviderName,
			Model:    AnthropicDefaultModel,
		}
	}
	return &AnthropicProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
		model:      modelConfig,
	}
}

// Name 返回提供商名称。
func (p *AnthropicProvider) Name() string {
	return AnthropicProviderName
}

// GetModelConfig 获取当前模型配置。
func (p *AnthropicProvider) GetModelConfig() model.ModelConfig {
	if p.model != nil {
		return *p.model
	}
	return model.ModelConfig{}
}

// Description 返回提供商描述。
func (p *AnthropicProvider) Description() string {
	return "Anthropic Claude AI 模型"
}

// Validate 验证配置是否有效。
func (p *AnthropicProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *AnthropicProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	reqBody, err := p.buildRequest(messages, config, false)
	if err != nil {
		return conversation.Message{}, err
	}

	resp, err := p.sendRequest(ctx, reqBody, false)
	if err != nil {
		return conversation.Message{}, err
	}

	return p.parseResponse(resp)
}

// Stream 执行流式请求。
func (p *AnthropicProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
	streamChan := make(chan StreamChunk, 32)

	go func() {
		defer close(streamChan)

		reqBody, err := p.buildRequest(messages, config, true)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}

		req, err := p.createRequest(ctx, reqBody, true)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			streamChan <- StreamChunk{Err: fmt.Errorf("API request failed: %s", resp.Status)}
			return
		}

		// 解析 SSE 流
		p.parseStream(resp.Body, streamChan)
	}()

	return streamChan, nil
}

// ListModels 列出支持的模型。
func (p *AnthropicProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	models := []ModelInfo{
		{
			ID:             "claude-opus-4-6",
			Name:           "Claude Opus 4.6",
			ContextWindow:  200000,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "claude-sonnet-4-6",
			Name:           "Claude Sonnet 4.6",
			ContextWindow:  200000,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "claude-sonnet-4-5",
			Name:           "Claude Sonnet 4.5",
			ContextWindow:  200000,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "claude-haiku-4-5",
			Name:           "Claude Haiku 4.5",
			ContextWindow:  200000,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "claude-opus-4-5",
			Name:           "Claude Opus 4.5",
			ContextWindow:  200000,
			SupportsStream: true,
			SupportsTools:  true,
		},
	}

	if p.customModels != nil {
		for _, modelID := range p.customModels {
			models = append(models, ModelInfo{
				ID:             modelID,
				Name:           modelID,
				ContextWindow:  200000,
				SupportsStream: true,
				SupportsTools:  true,
			})
		}
	}

	return models, nil
}

func (p *AnthropicProvider) buildRequest(messages []conversation.Message, config model.ModelConfig, streaming bool) (*anthropicRequest, error) {
	anthropicMsgs := make([]anthropicMessage, 0, len(messages))

	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		} else if msg.Role == "system" {
			role = "user" // Anthropic 将 system 消息合并到 user
		}

		content := make([]anthropicContent, 0)

		// 处理内容列表
		for _, c := range msg.Content {
			switch c.Type {
			case conversation.MessageContentText:
				content = append(content, anthropicContent{
					Type: "text",
					Text: c.Text,
				})
			case conversation.MessageContentImage:
				content = append(content, anthropicContent{
					Type: "image",
					// TODO: 处理图片数据
				})
			case conversation.MessageContentToolUse:
				content = append(content, anthropicContent{
					Type: "tool_use",
					ID:   msg.ToolCallID,
					Name: c.ToolName,
					Input: c.ToolArgs,
				})
			case conversation.MessageContentToolResult:
				content = append(content, anthropicContent{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Content:   []anthropicContent{{Type: "text", Text: c.ToolResult}},
				})
			}
		}

		if len(content) > 0 {
			anthropicMsgs = append(anthropicMsgs, anthropicMessage{
				Role:    role,
				Content: content,
			})
		}
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &anthropicRequest{
		Model:     config.Model,
		MaxTokens: maxTokens,
		Messages:  anthropicMsgs,
		Stream:    streaming,
	}, nil
}

func (p *AnthropicProvider) createRequest(ctx context.Context, body *anthropicRequest, streaming bool) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := p.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", AnthropicAPIVersion)
	if streaming {
		req.Header.Set("Accept", "text/event-stream")
	}

	return req, nil
}

func (p *AnthropicProvider) sendRequest(ctx context.Context, body *anthropicRequest, streaming bool) (*anthropicResponse, error) {
	req, err := p.createRequest(ctx, body, streaming)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed: %s, body: %s", resp.Status, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

func (p *AnthropicProvider) parseResponse(resp *anthropicResponse) (conversation.Message, error) {
	msg := conversation.NewTextMessage("assistant", "")

	for _, content := range resp.Content {
		switch content.Type {
		case "text":
			msg.Content = append(msg.Content, conversation.MessageContent{
				Type: conversation.MessageContentText,
				Text: content.Text,
			})
		case "tool_use":
			msg.Content = append(msg.Content, conversation.MessageContent{
				Type:     conversation.MessageContentToolUse,
				ToolName: content.Name,
				ToolArgs: json.RawMessage(content.Input.(json.RawMessage)),
			})
			msg.ToolCallID = content.ID
		}
	}

	return msg, nil
}

func (p *AnthropicProvider) parseStream(body io.Reader, streamChan chan<- StreamChunk) {
	// 简化的 SSE 解析器
	buf := make([]byte, 4096)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			data := string(buf[:n])
			lines := strings.Split(data, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					jsonData := strings.TrimPrefix(line, "data: ")
					if jsonData == "[DONE]" {
						streamChan <- StreamChunk{Done: true}
						return
					}

					var event anthropicResponse
					if err := json.Unmarshal([]byte(jsonData), &event); err == nil {
						for _, content := range event.Content {
							if content.Type == "text" {
								streamChan <- StreamChunk{Text: content.Text}
							}
						}
					}
				}
			}
		}
		if err != nil {
			if err != io.EOF {
				streamChan <- StreamChunk{Err: err}
			}
			return
		}
	}
}
