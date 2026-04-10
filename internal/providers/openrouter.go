// Package providers 提供 OpenRouter 提供商实现。
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
	OpenRouterProviderName = "openrouter"
	OpenRouterDefaultModel = "meta-llama/llama-3.1-405b-instruct"
	OpenRouterBaseURL      = "https://openrouter.ai/api/v1/chat/completions"
)

// OpenRouterProvider OpenRouter 提供商。
type OpenRouterProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      *model.ModelConfig
}

// openrouterRequest OpenRouter 请求格式。
type openrouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openrouterMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
}

// openrouterMessage OpenRouter 消息格式。
type openrouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openrouterResponse OpenRouter 响应格式。
type openrouterResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int             `json:"index"`
		Message      openrouterMessage `json:"message"`
		FinishReason string          `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// openrouterModel OpenRouter 模型信息。
type openrouterModel struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	ContextWindow    int     `json:"context_length"`
	TopProvider      string  `json:"top_provider"`
	Pricing          struct {
		Prompt     float64 `json:"prompt"`
		Completion float64 `json:"completion"`
	} `json:"pricing"`
}

// openrouterModelsResponse OpenRouter 模型列表响应。
type openrouterModelsResponse struct {
	Data []openrouterModel `json:"data"`
}

// NewOpenRouterProvider 创建 OpenRouter 提供商。
func NewOpenRouterProvider(apiKey string, modelConfig *model.ModelConfig) *OpenRouterProvider {
	if modelConfig == nil {
		modelConfig = &model.ModelConfig{
			Provider: OpenRouterProviderName,
			Model:    OpenRouterDefaultModel,
		}
	}
	return &OpenRouterProvider{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		model:      modelConfig,
	}
}

// Name 返回提供商名称。
func (p *OpenRouterProvider) Name() string {
	return OpenRouterProviderName
}

// GetModelConfig 获取当前模型配置。
func (p *OpenRouterProvider) GetModelConfig() model.ModelConfig {
	if p.model != nil {
		return *p.model
	}
	return model.ModelConfig{}
}

// Description 返回提供商描述。
func (p *OpenRouterProvider) Description() string {
	return "OpenRouter - 多模型聚合平台"
}

// Validate 验证配置是否有效。
func (p *OpenRouterProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *OpenRouterProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	reqBody := &openrouterRequest{
		Model:  config.Model,
		Messages: make([]openrouterMessage, 0, len(messages)),
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	for _, msg := range messages {
		var contentBuilder strings.Builder
		for _, c := range msg.Content {
			if c.Type == conversation.MessageContentText {
				contentBuilder.WriteString(c.Text)
			}
		}
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		} else if msg.Role == "system" {
			role = "system"
		}
		reqBody.Messages = append(reqBody.Messages, openrouterMessage{
			Role:    role,
			Content: contentBuilder.String(),
		})
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return conversation.Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", OpenRouterBaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return conversation.Message{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/camark/Gotosee")
	req.Header.Set("X-Title", "gogo")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return conversation.Message{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return conversation.Message{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return conversation.Message{}, fmt.Errorf("API request failed: %s - %s", resp.Status, string(respBody))
	}

	var apiResp openrouterResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return conversation.Message{}, err
	}

	if len(apiResp.Choices) == 0 {
		return conversation.Message{}, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content
	return conversation.NewTextMessage("assistant", content), nil
}

// Stream 执行流式请求。
func (p *OpenRouterProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
	streamChan := make(chan StreamChunk, 32)

	go func() {
		defer close(streamChan)

		reqBody := &openrouterRequest{
			Model:  config.Model,
			Messages: make([]openrouterMessage, 0, len(messages)),
			MaxTokens:   config.MaxTokens,
			Temperature: config.Temperature,
			Stream:      true,
		}

		for _, msg := range messages {
			var contentBuilder strings.Builder
			for _, c := range msg.Content {
				if c.Type == conversation.MessageContentText {
					contentBuilder.WriteString(c.Text)
				}
			}
			role := "user"
			if msg.Role == "assistant" {
				role = "assistant"
			}
			reqBody.Messages = append(reqBody.Messages, openrouterMessage{
				Role:    role,
				Content: contentBuilder.String(),
			})
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", OpenRouterBaseURL, bytes.NewReader(jsonBody))
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
		req.Header.Set("HTTP-Referer", "https://github.com/camark/Gotosee")
		req.Header.Set("X-Title", "gogo")
		req.Header.Set("Accept", "text/event-stream")

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
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
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

						var chunk openrouterResponse
						if err := json.Unmarshal([]byte(jsonData), &chunk); err == nil {
							for _, choice := range chunk.Choices {
								if choice.Message.Content != "" {
									streamChan <- StreamChunk{Text: choice.Message.Content}
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
	}()

	return streamChan, nil
}

// ListModels 列出支持的模型。
func (p *OpenRouterProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	url := "https://openrouter.ai/api/v1/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: %s", resp.Status)
	}

	var modelResp openrouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelResp); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(modelResp.Data))
	for _, m := range modelResp.Data {
		models = append(models, ModelInfo{
			ID:             m.ID,
			Name:           m.Name,
			Description:    m.Description,
			ContextWindow:  m.ContextWindow,
			SupportsStream: true,
			SupportsTools:  true,
		})
	}

	return models, nil
}
