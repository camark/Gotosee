// Package providers 提供 Ollama 提供商实现。
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/model"
)

const (
	OllamaProviderName = "ollama"
	OllamaDefaultURL   = "http://localhost:11434"
)

// OllamaProvider Ollama 提供商。
type OllamaProvider struct {
	baseURL    string
	httpClient *http.Client
	model      *model.ModelConfig
}

// ollamaRequest Ollama 请求格式。
type ollamaRequest struct {
	Model    string         `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  ollamaOptions  `json:"options,omitempty"`
}

// ollamaMessage Ollama 消息格式。
type ollamaMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content"`
	Images    []string      `json:"images,omitempty"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

// ollamaToolCall Ollama 工具调用。
type ollamaToolCall struct {
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ollamaOptions Ollama 选项。
type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// ollamaResponse Ollama 响应格式。
type ollamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool   `json:"done"`
}

// ollamaModelResponse Ollama 模型列表响应。
type ollamaModelResponse struct {
	Models []ollamaModelItem `json:"models"`
}

// ollamaModelItem Ollama 模型项。
type ollamaModelItem struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	ModifiedAt string `json:"modified_at"`
}

// NewOllamaProvider 创建 Ollama 提供商。
func NewOllamaProvider(baseURL string, modelConfig *model.ModelConfig) *OllamaProvider {
	if baseURL == "" {
		baseURL = OllamaDefaultURL
	}
	if modelConfig == nil {
		modelConfig = &model.ModelConfig{
			Provider: OllamaProviderName,
			Model:    "llama3.1",
		}
	}
	return &OllamaProvider{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		model:      modelConfig,
	}
}

// Name 返回提供商名称。
func (p *OllamaProvider) Name() string {
	return OllamaProviderName
}

// GetModelConfig 获取当前模型配置。
func (p *OllamaProvider) GetModelConfig() model.ModelConfig {
	if p.model != nil {
		return *p.model
	}
	return model.ModelConfig{}
}

// Description 返回提供商描述。
func (p *OllamaProvider) Description() string {
	return "Ollama 本地模型"
}

// Validate 验证配置是否有效。
func (p *OllamaProvider) Validate() error {
	// Ollama 不需要 API key
	return nil
}

// Complete 执行完成请求。
func (p *OllamaProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	reqBody := &ollamaRequest{
		Model:    config.Model,
		Messages: p.convertMessages(messages),
		Stream:   false,
		Options: ollamaOptions{
			Temperature: config.Temperature,
			NumPredict:  config.MaxTokens,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return conversation.Message{}, err
	}

	url := p.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return conversation.Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")

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
		return conversation.Message{}, fmt.Errorf("Ollama request failed: %s", string(respBody))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return conversation.Message{}, err
	}

	return conversation.Message{
		Role: conversation.RoleAssistant,
		Content: []conversation.MessageContent{
			{Type: conversation.MessageContentText, Text: ollamaResp.Message.Content},
		},
	}, nil
}

// Stream 执行流式请求。
func (p *OllamaProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
	streamChan := make(chan StreamChunk, 32)

	go func() {
		defer close(streamChan)

		reqBody := &ollamaRequest{
			Model:    config.Model,
			Messages: p.convertMessages(messages),
			Stream:   true,
			Options: ollamaOptions{
				Temperature: config.Temperature,
				NumPredict:  config.MaxTokens,
			},
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}

		url := p.baseURL + "/api/chat"
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			streamChan <- StreamChunk{Err: fmt.Errorf("Ollama request failed: %s", string(body))}
			return
		}

		// 解析 NDJSON 流
		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk ollamaResponse
			if err := decoder.Decode(&chunk); err != nil {
				if err != io.EOF {
					streamChan <- StreamChunk{Err: err}
				}
				return
			}

			if chunk.Message.Content != "" {
				streamChan <- StreamChunk{Text: chunk.Message.Content}
			}

			if chunk.Done {
				streamChan <- StreamChunk{Done: true}
				return
			}
		}
	}()

	return streamChan, nil
}

// ListModels 列出可用的模型。
func (p *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	url := p.baseURL + "/api/tags"
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

	var modelResp ollamaModelResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelResp); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(modelResp.Models))
	for _, m := range modelResp.Models {
		models = append(models, ModelInfo{
			ID:            m.Name,
			Name:          m.Name,
			ContextWindow: 4096, // Ollama 默认上下文
			SupportsStream: true,
			SupportsTools:  false, // Ollama 工具支持有限
		})
	}

	return models, nil
}

func (p *OllamaProvider) convertMessages(messages []conversation.Message) []ollamaMessage {
	ollamaMsgs := make([]ollamaMessage, 0, len(messages))
	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		} else if msg.Role == "system" {
			role = "system"
		}

		// 拼接所有内容块为文本
		var contentBuilder strings.Builder
		for _, c := range msg.Content {
			if c.Type == conversation.MessageContentText {
				contentBuilder.WriteString(c.Text)
			}
		}

		ollamaMsgs = append(ollamaMsgs, ollamaMessage{
			Role:    role,
			Content: contentBuilder.String(),
		})
	}
	return ollamaMsgs
}
