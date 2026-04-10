// Package providers 提供 Azure OpenAI 提供商实现。
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
	AzureProviderName      = "azure"
	AzureDefaultModel      = "gpt-4o"
	AzureAPIVersion        = "2024-02-15-preview"
)

// AzureProvider Azure OpenAI 提供商。
type AzureProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      *model.ModelConfig
	deployment string
}

// azureRequest Azure 请求格式（与 OpenAI 兼容）。
type azureRequest struct {
	Messages    []azureMessage `json:"messages"`
	Model       string         `json:"model"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Tools       []azureTool    `json:"tools,omitempty"`
}

// azureMessage Azure 消息格式。
type azureMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content,omitempty"`
	ToolCalls []azureToolCall `json:"tool_calls,omitempty"`
}

// azureToolCall Azure 工具调用。
type azureToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// azureTool Azure 工具格式。
type azureTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Parameters  any    `json:"parameters,omitempty"`
	} `json:"function"`
}

// azureResponse Azure 响应格式。
type azureResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      azureMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// azureStreamChunk Azure 流式块。
type azureStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string `json:"role,omitempty"`
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				Index    int `json:"index"`
				ID       string `json:"id,omitempty"`
				Type     string `json:"type,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// NewAzureProvider 创建 Azure 提供商。
func NewAzureProvider(apiKey, baseURL, deployment string, modelConfig *model.ModelConfig) *AzureProvider {
	if modelConfig == nil {
		modelConfig = &model.ModelConfig{
			Provider: AzureProviderName,
			Model:    AzureDefaultModel,
		}
	}
	return &AzureProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
		model:      modelConfig,
		deployment: deployment,
	}
}

// Name 返回提供商名称。
func (p *AzureProvider) Name() string {
	return AzureProviderName
}

// GetModelConfig 获取当前模型配置。
func (p *AzureProvider) GetModelConfig() model.ModelConfig {
	if p.model != nil {
		return *p.model
	}
	return model.ModelConfig{}
}

// Description 返回提供商描述。
func (p *AzureProvider) Description() string {
	return "Azure OpenAI Service"
}

// Validate 验证配置是否有效。
func (p *AzureProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if p.baseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if p.deployment == "" {
		return fmt.Errorf("deployment name is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *AzureProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
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
func (p *AzureProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
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
			body, _ := io.ReadAll(resp.Body)
			streamChan <- StreamChunk{Err: fmt.Errorf("API request failed: %s - %s", resp.Status, string(body))}
			return
		}

		// 解析 SSE 流
		p.parseStream(resp.Body, streamChan)
	}()

	return streamChan, nil
}

// ListModels 列出支持的模型。
func (p *AzureProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Azure 返回部署的模型，这里列出常见的
	models := []ModelInfo{
		{
			ID:             "gpt-4o",
			Name:           "GPT-4o",
			ContextWindow:  128000,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "gpt-4o-mini",
			Name:           "GPT-4o Mini",
			ContextWindow:  128000,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "gpt-4-turbo",
			Name:           "GPT-4 Turbo",
			ContextWindow:  128000,
			SupportsStream: true,
			SupportsTools:  true,
		},
	}
	return models, nil
}

func (p *AzureProvider) buildRequest(messages []conversation.Message, config model.ModelConfig, streaming bool) (*azureRequest, error) {
	azureMsgs := make([]azureMessage, 0, len(messages))

	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		} else if msg.Role == "system" {
			role = "system"
		}

		// 拼接文本内容
		var contentBuilder strings.Builder
		for _, c := range msg.Content {
			if c.Type == conversation.MessageContentText {
				contentBuilder.WriteString(c.Text)
			}
		}

		azureMsg := azureMessage{
			Role:    role,
			Content: contentBuilder.String(),
		}

		if len(azureMsgs) > 0 || azureMsg.Content != "" {
			azureMsgs = append(azureMsgs, azureMsg)
		}
	}

	return &azureRequest{
		Messages:    azureMsgs,
		Model:       p.deployment,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		Stream:      streaming,
	}, nil
}

func (p *AzureProvider) createRequest(ctx context.Context, body *azureRequest, streaming bool) (*http.Request, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// Azure OpenAI API 格式：{endpoint}/openai/deployments/{deployment}/chat/completions?api-version={version}
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.baseURL, p.deployment, AzureAPIVersion)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	return req, nil
}

func (p *AzureProvider) sendRequest(ctx context.Context, body *azureRequest, streaming bool) (*azureResponse, error) {
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
		return nil, fmt.Errorf("API request failed: %s - %s", resp.Status, string(respBody))
	}

	var apiResp azureResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

func (p *AzureProvider) parseResponse(resp *azureResponse) (conversation.Message, error) {
	msg := conversation.NewTextMessage("assistant", "")

	if len(resp.Choices) == 0 {
		return msg, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	if choice.Message.Content != "" {
		msg.Content = append(msg.Content, conversation.MessageContent{
			Type: conversation.MessageContentText,
			Text: choice.Message.Content,
		})
	}

	return msg, nil
}

func (p *AzureProvider) parseStream(body io.Reader, streamChan chan<- StreamChunk) {
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

					var chunk azureStreamChunk
					if err := json.Unmarshal([]byte(jsonData), &chunk); err == nil {
						for _, choice := range chunk.Choices {
							if choice.Delta.Content != "" {
								streamChan <- StreamChunk{Text: choice.Delta.Content}
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
