// Package mcp 提供 HTTP Client MCP 服务器。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ============================================================================
// HTTP Client MCP 服务器
// ============================================================================

// HttpClientServer HTTP 客户端 MCP 服务器。
type HttpClientServer struct {
	*BaseServer
	client *http.Client
}

// NewHttpClientServer 创建新的 HTTP 客户端服务器。
func NewHttpClientServer() *HttpClientServer {
	server := &HttpClientServer{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "http-client-mcp",
			Version:     "1.0.0",
			Instructions: "HTTP 客户端：发送 GET/POST/PUT/DELETE 请求",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *HttpClientServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "http_get",
			Description: "发送 HTTP GET 请求",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "请求 URL"},
					"headers": {"type": "object", "description": "请求头"}
				},
				"required": ["url"]
			}`),
		},
		{
			Name:        "http_post",
			Description: "发送 HTTP POST 请求",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "请求 URL"},
					"body": {"type": "string", "description": "请求体"},
					"content_type": {"type": "string", "description": "Content-Type", "default": "application/json"},
					"headers": {"type": "object", "description": "请求头"}
				},
				"required": ["url", "body"]
			}`),
		},
		{
			Name:        "http_put",
			Description: "发送 HTTP PUT 请求",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "请求 URL"},
					"body": {"type": "string", "description": "请求体"},
					"content_type": {"type": "string", "description": "Content-Type", "default": "application/json"},
					"headers": {"type": "object", "description": "请求头"}
				},
				"required": ["url", "body"]
			}`),
		},
		{
			Name:        "http_delete",
			Description: "发送 HTTP DELETE 请求",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "请求 URL"},
					"headers": {"type": "object", "description": "请求头"}
				},
				"required": ["url"]
			}`),
		},
		{
			Name:        "http_patch",
			Description: "发送 HTTP PATCH 请求",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "请求 URL"},
					"body": {"type": "string", "description": "请求体"},
					"headers": {"type": "object", "description": "请求头"}
				},
				"required": ["url", "body"]
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *HttpClientServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "http_get":
		return s.httpGet(params)
	case "http_post":
		return s.httpPost(params)
	case "http_put":
		return s.httpPut(params)
	case "http_delete":
		return s.httpDelete(params)
	case "http_patch":
		return s.httpPatch(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// doRequest 发送 HTTP 请求。
func (s *HttpClientServer) doRequest(method, urlStr string, body io.Reader, headers map[string]string) (*ToolResult, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("创建请求失败：%v", err)}},
		}, nil
	}

	// 设置默认 headers
	req.Header.Set("User-Agent", "Goose-MCP/1.0")

	// 设置自定义 headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("请求失败：%v", err)}},
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("读取响应失败：%v", err)}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HTTP %s %s\n", resp.Status, urlStr))
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	// 响应头
	sb.WriteString("响应头:\n")
	for k, vv := range resp.Header {
		for _, v := range vv {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	// 响应体
	sb.WriteString("\n响应体:\n")
	sb.WriteString(string(respBody))

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"status":     resp.Status,
			"statusCode": resp.StatusCode,
			"headers":    resp.Header,
			"body":       string(respBody),
		},
	}, nil
}

// httpGet GET 请求。
func (s *HttpClientServer) httpGet(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	json.Unmarshal(params, &p)

	// 验证 URL
	if _, err := url.Parse(p.URL); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效 URL: %v", err)}},
		}, nil
	}

	return s.doRequest("GET", p.URL, nil, p.Headers)
}

// httpPost POST 请求。
func (s *HttpClientServer) httpPost(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL         string            `json:"url"`
		Body        string            `json:"body"`
		ContentType string            `json:"content_type"`
		Headers     map[string]string `json:"headers"`
	}
	json.Unmarshal(params, &p)

	if p.ContentType == "" {
		p.ContentType = "application/json"
	}

	headers := p.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = p.ContentType

	return s.doRequest("POST", p.URL, strings.NewReader(p.Body), headers)
}

// httpPut PUT 请求。
func (s *HttpClientServer) httpPut(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL         string            `json:"url"`
		Body        string            `json:"body"`
		ContentType string            `json:"content_type"`
		Headers     map[string]string `json:"headers"`
	}
	json.Unmarshal(params, &p)

	if p.ContentType == "" {
		p.ContentType = "application/json"
	}

	headers := p.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = p.ContentType

	return s.doRequest("PUT", p.URL, strings.NewReader(p.Body), headers)
}

// httpDelete DELETE 请求。
func (s *HttpClientServer) httpDelete(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	json.Unmarshal(params, &p)

	return s.doRequest("DELETE", p.URL, nil, p.Headers)
}

// httpPatch PATCH 请求。
func (s *HttpClientServer) httpPatch(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL     string            `json:"url"`
		Body    string            `json:"body"`
		Headers map[string]string `json:"headers"`
	}
	json.Unmarshal(params, &p)

	headers := p.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/json"

	return s.doRequest("PATCH", p.URL, strings.NewReader(p.Body), headers)
}

// ListResources 列出资源。
func (s *HttpClientServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *HttpClientServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
