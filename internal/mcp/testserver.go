// Package mcp 提供测试用的 MCP 服务器。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// ============================================================================
// Test MCP 服务器
// ============================================================================

// TestServer 测试用 MCP 服务器。
type TestServer struct {
	*BaseServer
}

// NewTestServer 创建新的测试服务器。
func NewTestServer() *TestServer {
	server := &TestServer{}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "test-mcp",
			Version:     "1.0.0",
			Instructions: "测试用 MCP 服务器，提供简单的工具用于测试",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *TestServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "echo",
			Description: "回显输入的消息",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"message": {"type": "string", "description": "要回显的消息"}
				},
				"required": ["message"]
			}`),
		},
		{
			Name:        "add",
			Description: "加法运算",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"a": {"type": "number", "description": "第一个数"},
					"b": {"type": "number", "description": "第二个数"}
				},
				"required": ["a", "b"]
			}`),
		},
		{
			Name:        "multiply",
			Description: "乘法运算",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"a": {"type": "number", "description": "第一个数"},
					"b": {"type": "number", "description": "第二个数"}
				},
				"required": ["a", "b"]
			}`),
		},
		{
			Name:        "get_test_data",
			Description: "获取测试数据",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"count": {"type": "integer", "description": "返回数据的数量", "default": 3}
				}
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *TestServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "echo":
		return s.echo(params)
	case "add":
		return s.add(params)
	case "multiply":
		return s.multiply(params)
	case "get_test_data":
		return s.getTestData(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// echo 回显消息。
func (s *TestServer) echo(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("回显：%s", p.Message)}},
		Data: map[string]interface{}{
			"original": p.Message,
			"echoed":   true,
		},
	}, nil
}

// add 加法运算。
func (s *TestServer) add(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	result := p.A + p.B

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("%v + %v = %v", p.A, p.B, result)}},
		Data: map[string]interface{}{
			"a":      p.A,
			"b":      p.B,
			"result": result,
		},
	}, nil
}

// multiply 乘法运算。
func (s *TestServer) multiply(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	result := p.A * p.B

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("%v * %v = %v", p.A, p.B, result)}},
		Data: map[string]interface{}{
			"a":      p.A,
			"b":      p.B,
			"result": result,
		},
	}, nil
}

// getTestData 获取测试数据。
func (s *TestServer) getTestData(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Count int `json:"count"`
	}
	json.Unmarshal(params, &p)

	if p.Count <= 0 {
		p.Count = 3
	}
	if p.Count > 10 {
		p.Count = 10
	}

	items := make([]string, p.Count)
	for i := 0; i < p.Count; i++ {
		items[i] = fmt.Sprintf("测试数据 #%d", i+1)
	}

	var sb string
	sb += fmt.Sprintf("获取到 %d 条测试数据：\n", p.Count)
	for _, item := range items {
		sb += fmt.Sprintf("  - %s\n", item)
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb}},
		Data: map[string]interface{}{
			"count": p.Count,
			"items": items,
		},
	}, nil
}

// ListResources 列出资源。
func (s *TestServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:         []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *TestServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
