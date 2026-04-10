// Package providers 提供 Google 提供商实现。
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/model"
)

const (
	GoogleProviderName  = "google"
	GoogleDefaultModel  = "gemini-2.0-flash"
	GoogleBaseURL       = "https://generativelanguage.googleapis.com"
	GoogleAPIVersion    = "v1beta"
)

// GoogleProvider Google 提供商。
type GoogleProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      *model.ModelConfig
}

// googleRequest Google 请求格式。
type googleRequest struct {
	Contents     []googleContent `json:"contents"`
	GenerationConfig *googleGenerationConfig `json:"generationConfig,omitempty"`
	Tools        []googleTool    `json:"tools,omitempty"`
}

// googleContent Google 内容格式。
type googleContent struct {
	Role  string            `json:"role"`
	Parts []googlePart      `json:"parts"`
}

// googlePart Google 部分格式。
type googlePart struct {
	Text       string `json:"text,omitempty"`
	InlineData *googleBlob `json:"inlineData,omitempty"`
	FunctionCall *googleFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *googleFunctionResponse `json:"functionResponse,omitempty"`
}

// googleBlob Google 图片数据。
type googleBlob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// googleFunctionCall Google 函数调用。
type googleFunctionCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

// googleFunctionResponse Google 函数响应。
type googleFunctionResponse struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

// googleGenerationConfig Google 生成配置。
type googleGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
}

// googleTool Google 工具格式。
type googleTool struct {
	FunctionDeclarations []googleFunction `json:"functionDeclarations,omitempty"`
}

// googleFunction Google 函数。
type googleFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// googleResponse Google 响应格式。
type googleResponse struct {
	Candidates []googleCandidate `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// googleCandidate Google 候选。
type googleCandidate struct {
	Content      googleContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

// NewGoogleProvider 创建 Google 提供商。
func NewGoogleProvider(apiKey, baseURL string, modelConfig *model.ModelConfig) *GoogleProvider {
	if baseURL == "" {
		baseURL = GoogleBaseURL
	}
	if modelConfig == nil {
		modelConfig = &model.ModelConfig{
			Provider: GoogleProviderName,
			Model:    GoogleDefaultModel,
		}
	}
	return &GoogleProvider{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
		model:      modelConfig,
	}
}

// Name 返回提供商名称。
func (p *GoogleProvider) Name() string {
	return GoogleProviderName
}

// GetModelConfig 获取当前模型配置。
func (p *GoogleProvider) GetModelConfig() model.ModelConfig {
	if p.model != nil {
		return *p.model
	}
	return model.ModelConfig{}
}

// Description 返回提供商描述。
func (p *GoogleProvider) Description() string {
	return "Google Gemini AI 模型"
}

// Validate 验证配置是否有效。
func (p *GoogleProvider) Validate() error {
	if p.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// Complete 执行完成请求。
func (p *GoogleProvider) Complete(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (conversation.Message, error) {
	reqBody, err := p.buildRequest(messages, config)
	if err != nil {
		return conversation.Message{}, err
	}

	resp, err := p.sendRequest(ctx, reqBody)
	if err != nil {
		return conversation.Message{}, err
	}

	return p.parseResponse(resp)
}

// Stream 执行流式请求。
func (p *GoogleProvider) Stream(ctx context.Context, messages []conversation.Message, config model.ModelConfig) (<-chan StreamChunk, error) {
	streamChan := make(chan StreamChunk, 32)

	go func() {
		defer close(streamChan)

		reqBody, err := p.buildRequest(messages, config)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}

		url := fmt.Sprintf("%s/%s/models/%s:streamGenerateContent?key=%s&alt=sse",
			p.baseURL, GoogleAPIVersion, config.Model, p.apiKey)

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			streamChan <- StreamChunk{Err: err}
			return
		}

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
			streamChan <- StreamChunk{Err: fmt.Errorf("API request failed: %s - %s", resp.Status, string(body))}
			return
		}

		// 解析 SSE 流
		decoder := json.NewDecoder(resp.Body)
		for {
			var resp googleResponse
			if err := decoder.Decode(&resp); err != nil {
				if err != io.EOF {
					streamChan <- StreamChunk{Err: err}
				}
				return
			}

			for _, candidate := range resp.Candidates {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						streamChan <- StreamChunk{Text: part.Text}
					}
				}
			}
		}
	}()

	return streamChan, nil
}

// ListModels 列出支持的模型。
func (p *GoogleProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	models := []ModelInfo{
		{
			ID:             "gemini-2.0-flash",
			Name:           "Gemini 2.0 Flash",
			ContextWindow:  1048576,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "gemini-1.5-pro",
			Name:           "Gemini 1.5 Pro",
			ContextWindow:  2097152,
			SupportsStream: true,
			SupportsTools:  true,
		},
		{
			ID:             "gemini-1.5-flash",
			Name:           "Gemini 1.5 Flash",
			ContextWindow:  1048576,
			SupportsStream: true,
			SupportsTools:  true,
		},
	}
	return models, nil
}

func (p *GoogleProvider) buildRequest(messages []conversation.Message, config model.ModelConfig) (*googleRequest, error) {
	contents := make([]googleContent, 0, len(messages))

	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		parts := make([]googlePart, 0)

		for _, c := range msg.Content {
			switch c.Type {
			case conversation.MessageContentText:
				parts = append(parts, googlePart{Text: c.Text})
			case conversation.MessageContentImage:
				if c.ImageData != nil {
					parts = append(parts, googlePart{
						InlineData: &googleBlob{
							MimeType: "image/png",
							Data:     string(c.ImageData),
						},
					})
				}
			case conversation.MessageContentToolUse:
				parts = append(parts, googlePart{
					FunctionCall: &googleFunctionCall{
						Name: c.ToolName,
						Args: c.ToolArgs,
					},
				})
			case conversation.MessageContentToolResult:
				parts = append(parts, googlePart{
					FunctionResponse: &googleFunctionResponse{
						Name: c.ToolName,
						Response: map[string]string{"result": c.ToolResult},
					},
				})
			}
		}

		if len(parts) > 0 {
			contents = append(contents, googleContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	req := &googleRequest{
		Contents: contents,
		GenerationConfig: &googleGenerationConfig{
			Temperature:     config.Temperature,
			MaxOutputTokens: config.MaxTokens,
		},
	}

	return req, nil
}

func (p *GoogleProvider) sendRequest(ctx context.Context, body *googleRequest) (*googleResponse, error) {
	url := fmt.Sprintf("%s/%s/models/%s:generateContent?key=%s",
		p.baseURL, GoogleAPIVersion, p.model.Model, p.apiKey)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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

	var apiResp googleResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

func (p *GoogleProvider) parseResponse(resp *googleResponse) (conversation.Message, error) {
	msg := conversation.NewTextMessage("assistant", "")

	if len(resp.Candidates) == 0 {
		return msg, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			msg.Content = append(msg.Content, conversation.MessageContent{
				Type: conversation.MessageContentText,
				Text: part.Text,
			})
		} else if part.FunctionCall != nil {
			var argsRaw json.RawMessage
			if part.FunctionCall.Args != nil {
				argsRaw, _ = json.Marshal(part.FunctionCall.Args)
			}
			msg.Content = append(msg.Content, conversation.MessageContent{
				Type:     conversation.MessageContentToolUse,
				ToolName: part.FunctionCall.Name,
				ToolArgs: argsRaw,
			})
			msg.ToolCallID = part.FunctionCall.Name
		}
	}

	return msg, nil
}
