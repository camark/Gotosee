// Package mcp 提供 Environment MCP 服务器（环境变量管理）。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ============================================================================
// Environment MCP 服务器
// ============================================================================

// EnvironmentServer 环境变量管理 MCP 服务器。
type EnvironmentServer struct {
	*BaseServer
}

// NewEnvironmentServer 创建新的环境变量服务器。
func NewEnvironmentServer() *EnvironmentServer {
	server := &EnvironmentServer{}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "environment-mcp",
			Version:     "1.0.0",
			Instructions: "管理环境变量：查看、设置、获取环境变量",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *EnvironmentServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "list_env_vars",
			Description: "列出所有环境变量",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prefix": {"type": "string", "description": "按前缀过滤"}
				}
			}`),
		},
		{
			Name:        "get_env_var",
			Description: "获取环境变量值",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "环境变量名称"}
				},
				"required": ["name"]
			}`),
		},
		{
			Name:        "set_env_var",
			Description: "设置环境变量",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "环境变量名称"},
					"value": {"type": "string", "description": "环境变量值"}
				},
				"required": ["name", "value"]
			}`),
		},
		{
			Name:        "delete_env_var",
			Description: "删除环境变量",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "环境变量名称"}
				},
				"required": ["name"]
			}`),
		},
		{
			Name:        "get_env_subset",
			Description: "获取相关环境变量组",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pattern": {"type": "string", "description": "匹配模式"}
				},
				"required": ["pattern"]
			}`),
		},
		{
			Name:        "check_env_exists",
			Description: "检查环境变量是否存在",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "环境变量名称"}
				},
				"required": ["name"]
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *EnvironmentServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "list_env_vars":
		return s.listEnvVars(params)
	case "get_env_var":
		return s.getEnvVar(params)
	case "set_env_var":
		return s.setEnvVar(params)
	case "delete_env_var":
		return s.deleteEnvVar(params)
	case "get_env_subset":
		return s.getEnvSubset(params)
	case "check_env_exists":
		return s.checkEnvExists(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// listEnvVars 列出环境变量。
func (s *EnvironmentServer) listEnvVars(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Prefix string `json:"prefix"`
	}
	json.Unmarshal(params, &p)

	env := os.Environ()
	sort.Strings(env)

	var filtered []string
	for _, e := range env {
		if p.Prefix != "" {
			if !strings.HasPrefix(e, p.Prefix+"=") {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("环境变量列表 (共 %d 个):\n\n", len(filtered)))
	for _, e := range filtered {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			sb.WriteString(fmt.Sprintf("%s=%s\n", parts[0], parts[1]))
		}
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"count": len(filtered),
			"vars":  filtered,
		},
	}, nil
}

// getEnvVar 获取环境变量。
func (s *EnvironmentServer) getEnvVar(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name string `json:"name"`
	}
	json.Unmarshal(params, &p)

	value, exists := os.LookupEnv(p.Name)
	if !exists {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("环境变量 '%s' 不存在", p.Name)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("%s=%s", p.Name, value)}},
		Data: map[string]interface{}{
			"name":  p.Name,
			"value": value,
		},
	}, nil
}

// setEnvVar 设置环境变量。
func (s *EnvironmentServer) setEnvVar(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	json.Unmarshal(params, &p)

	if p.Name == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "环境变量名称不能为空"}},
		}, nil
	}

	os.Setenv(p.Name, p.Value)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已设置环境变量：%s=%s", p.Name, p.Value)}},
		Data: map[string]interface{}{
			"name":  p.Name,
			"value": p.Value,
		},
	}, nil
}

// deleteEnvVar 删除环境变量。
func (s *EnvironmentServer) deleteEnvVar(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name string `json:"name"`
	}
	json.Unmarshal(params, &p)

	if p.Name == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "环境变量名称不能为空"}},
		}, nil
	}

	_, exists := os.LookupEnv(p.Name)
	if !exists {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("环境变量 '%s' 不存在", p.Name)}},
		}, nil
	}

	os.Unsetenv(p.Name)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已删除环境变量：%s", p.Name)}},
		Data: map[string]interface{}{
			"name": p.Name,
		},
	}, nil
}

// getEnvSubset 获取相关环境变量组。
func (s *EnvironmentServer) getEnvSubset(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Pattern string `json:"pattern"`
	}
	json.Unmarshal(params, &p)

	if p.Pattern == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "匹配模式不能为空"}},
		}, nil
	}

	env := os.Environ()
	var matches []string

	for _, e := range env {
		if strings.Contains(e, p.Pattern) {
			matches = append(matches, e)
		}
	}

	sort.Strings(matches)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("包含 '%s' 的环境变量 (共 %d 个):\n\n", p.Pattern, len(matches)))
	for _, e := range matches {
		sb.WriteString(fmt.Sprintf("%s\n", e))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"count": len(matches),
			"vars":  matches,
		},
	}, nil
}

// checkEnvExists 检查环境变量是否存在。
func (s *EnvironmentServer) checkEnvExists(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name string `json:"name"`
	}
	json.Unmarshal(params, &p)

	if p.Name == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "环境变量名称不能为空"}},
		}, nil
	}

	_, exists := os.LookupEnv(p.Name)

	result := "不存在"
	if exists {
		result = "存在"
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("环境变量 '%s' %s", p.Name, result)}},
		Data: map[string]interface{}{
			"name":   p.Name,
			"exists": exists,
		},
	}, nil
}

// ListResources 列出资源。
func (s *EnvironmentServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *EnvironmentServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
