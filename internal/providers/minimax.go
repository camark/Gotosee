// Package providers 提供 MiniMax 提供商实现。
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

// MiniMaxProvider MiniMax 提供商实现（名之梦）。
type MiniMaxProvider struct {
	BaseProvider
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// MiniMaxConfig MiniMax 配置。
type MiniMaxConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// DefaultMiniMaxBaseURL 默认 MiniMax API 地址。
const DefaultMiniMaxBaseURL = "https://api.minimax.chat"

// DefaultMiniMaxModel 默认 MiniMax 模型。
const DefaultMiniMaxModel = "abab6.5s-chat"

// KnownMiniMaxModels 已知的 MiniMax 模型。
var KnownMiniMaxModels = map[string]int{
	"abab6.5s-chat":  256000,
	"abab6.5-chat":   256000,
	"abab6-chat":     256000,
	"abab5.5s-chat":  8192,
	"abab5.5-chat":   16384,
}

// NewMiniMaxProvider 创建新的 MiniMax 提供商。
func NewMiniMaxProvider(config MiniMaxConfig, modelConfig model.ModelConfig) *MiniMaxProvider {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultMiniMaxBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &MiniMaxProvider{
		BaseProvider: NewBaseProvider("minimax", "MiniMax", "名之梦 MiniMax AI 模型", modelConfig),
		apiKey:       config.APIKey,
		baseURL:      baseURL + "/v1/text/chatcompletion_v2",
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Name 返回提供商名称。
func (p *MiniMaxProvider) Name() string {
	return "minimax"
}

// GetModelConfig 获取当前模型配置。
func (p *MiniMaxProvider) GetModelConfig() model.ModelConfig {
	return p.BaseProvider.Config()
}

// Validate 验证配置。
func (p *MiniMaxProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *MiniMaxProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
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

	var apiResp MiniMaxCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return conversation.Message{}, err
	}

	if apiResp.BaseResp.StatusCode != 0 {
		return conversation.Message{}, fmt.Errorf("MiniMax API error: %s", apiResp.BaseResp.StatusMsg)
	}

	return p.responseToMessage(apiResp), nil
}

// Stream 执行流式请求。
func (p *MiniMaxProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
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

			var chunkData MiniMaxStreamResponse
			if err := json.Unmarshal([]byte(dataStr), &chunkData); err != nil {
				continue
			}

			if len(chunkData.Choices) > 0 {
				delta := chunkData.Choices[0].Text
				if delta != "" {
					ch <- StreamChunk{Text: delta}
				}
			}
		}
	}()

	return ch, nil
}

// ListModels 列出支持的模型。
func (p *MiniMaxProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	result := make([]ModelInfo, 0, len(KnownMiniMaxModels))
	for modelID, contextWindow := range KnownMiniMaxModels {
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

func (p *MiniMaxProvider) setRequestHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func (p *MiniMaxProvider) buildCompletionRequest(messages []conversation.Message, config model.ModelConfig) ([]byte, error) {
	apiMessages := make([]MiniMaxMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = p.messageToAPI(msg)
	}

	model := config.Model
	if model == "" {
		model = DefaultMiniMaxModel
	}

	req := MiniMaxCompletionRequest{
		Model:       model,
		Messages:    apiMessages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
	}

	return json.Marshal(req)
}

func (p *MiniMaxProvider) messageToAPI(msg conversation.Message) MiniMaxMessage {
	var role string
	switch msg.Role {
	case conversation.RoleUser:
		role = "USER"
	case conversation.RoleAssistant:
		role = "BOT"
	case conversation.RoleSystem:
		role = "SYSTEM"
	default:
		role = "USER"
	}

	content := ""
	for _, c := range msg.Content {
		if c.Type == conversation.MessageContentText {
			content += c.Text
		}
	}

	return MiniMaxMessage{
		SenderType: role,
		SenderName: role,
		Text:       content,
	}
}

func (p *MiniMaxProvider) responseToMessage(resp MiniMaxCompletionResponse) conversation.Message {
	content := conversation.MessageContent{
		Type: conversation.MessageContentText,
	}

	if len(resp.Choices) > 0 {
		content.Text = resp.Choices[0].Message.Text
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
// MiniMax API 类型定义
// ============================================================================

type MiniMaxCompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []MiniMaxMessage  `json:"messages"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

type MiniMaxMessage struct {
	SenderType string `json:"sender_type"`
	SenderName string `json:"sender_name"`
	Text       string `json:"text"`
}

type MiniMaxCompletionResponse struct {
	ID      string                 `json:"id"`
	Model   string                 `json:"model"`
	Choices []MiniMaxChoice        `json:"choices"`
	Usage   MiniMaxUsage           `json:"usage"`
	Created int64                  `json:"created"`
	BaseResp MiniMaxBaseResponse  `json:"base_resp"`
}

type MiniMaxBaseResponse struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type MiniMaxChoice struct {
	Index        int              `json:"index"`
	Message      MiniMaxMessage   `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

type MiniMaxDelta struct {
	SenderType string `json:"sender_type,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
	Text       string `json:"text,omitempty"`
}

type MiniMaxUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type MiniMaxStreamResponse struct {
	Choices []MiniMaxDelta `json:"choices"`
	Usage   MiniMaxUsage   `json:"usage"`
}
