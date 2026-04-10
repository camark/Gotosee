// Package mcp 提供 MCP (Model Context Protocol) 服务器功能。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ============================================================================
// MCP 基础类型
// ============================================================================

// ServerInfo MCP 服务器信息。
type ServerInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Instructions string                `json:"instructions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ServerCapabilities MCP 服务器能力。
type ServerCapabilities struct {
	Tools      *ToolCapabilities      `json:"tools,omitempty"`
	Resources  *ResourceCapabilities  `json:"resources,omitempty"`
	Prompts    *PromptCapabilities    `json:"prompts,omitempty"`
	Completions *CompletionCapabilities `json:"completions,omitempty"`
}

// ToolCapabilities 工具能力。
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapabilities 资源能力。
type ResourceCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptCapabilities 提示能力。
type PromptCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// CompletionCapabilities 完成能力。
type CompletionCapabilities struct{}

// InitializeResult MCP 初始化响应。
type InitializeResult struct {
	ProtocolVersion string           `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo       `json:"serverInfo"`
}

// ============================================================================
// 工具类型
// ============================================================================

// Tool MCP 工具定义。
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolResult 工具执行结果。
type ToolResult struct {
	Content []Content      `json:"content"`
	IsError bool           `json:"isError,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Content MCP 内容类型。
type Content struct {
	Type     string          `json:"type"` // "text", "image", "resource"
	Text     string          `json:"text,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	MIMEType string          `json:"mimeType,omitempty"`
	Resource *ResourceContent `json:"resource,omitempty"`
}

// ServerNotification MCP 服务器通知。
type ServerNotification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// ResourceContent 资源内容。
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// ============================================================================
// 资源类型
// ============================================================================

// Resource MCP 资源定义。
type Resource struct {
	URI         string          `json:"uri"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	MIMEType    string          `json:"mimeType,omitempty"`
	Metadata    json.RawMessage `json:"_metadata,omitempty"`
}

// ResourceTemplate 资源模板。
type ResourceTemplate struct {
	URITemplate string          `json:"uriTemplate"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	MIMEType    string          `json:"mimeType,omitempty"`
	Metadata    json.RawMessage `json:"_metadata,omitempty"`
}

// ListResourcesResult 资源列表响应。
type ListResourcesResult struct {
	Resources       []Resource         `json:"resources"`
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates,omitempty"`
	NextCursor      string             `json:"nextCursor,omitempty"`
}

// ReadResourceResult 资源读取响应。
type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

// ResourceContents 资源内容。
type ResourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 encoded
}

// ============================================================================
// JSON-RPC 类型
// ============================================================================

// JSONRPCRequest JSON-RPC 请求。
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应。
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// JSONRPCNotification JSON-RPC 通知。
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCError RPC 错误。
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// RPC 错误码。
const (
	ErrParse      = -32700
	ErrInvalidReq = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams = -32602
	ErrInternal   = -32603
)

// NewRPCError 创建新的 RPC 错误。
func NewRPCError(code int, message string, data interface{}) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// ============================================================================
// 错误处理
// ============================================================================

// MCPError MCP 错误类型。
type MCPError struct {
	Code    int
	Message string
	Data    interface{}
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// 预定义的错误。
var (
	ErrToolNotFound     = &MCPError{Code: ErrMethodNotFound, Message: "tool not found"}
	ErrResourceNotFound = &MCPError{Code: ErrMethodNotFound, Message: "resource not found"}
	ErrMCPInvalidParams = &MCPError{Code: ErrInvalidParams, Message: "invalid parameters"}
)

// ============================================================================
// 服务器接口
// ============================================================================

// Server MCP 服务器接口。
type Server interface {
	// Initialize 初始化服务器
	Initialize(ctx context.Context) (*InitializeResult, error)

	// ListTools 列出所有工具
	ListTools(ctx context.Context) ([]Tool, error)

	// CallTool 调用工具
	CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error)

	// ListResources 列出所有资源
	ListResources(ctx context.Context) (*ListResourcesResult, error)

	// ReadResource 读取资源
	ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error)

	// Shutdown 关闭服务器
	Shutdown(ctx context.Context) error
}

// BaseServer MCP 服务器基础实现。
type BaseServer struct {
	info         ServerInfo
	capabilities ServerCapabilities
}

// NewBaseServer 创建基础服务器。
func NewBaseServer(name, version string) *BaseServer {
	return &BaseServer{
		info: ServerInfo{
			Name:    name,
			Version: version,
		},
	}
}

// Initialize 初始化服务器。
func (s *BaseServer) Initialize(ctx context.Context) (*InitializeResult, error) {
	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    s.capabilities,
		ServerInfo:      s.info,
	}, nil
}

// Shutdown 关闭服务器。
func (s *BaseServer) Shutdown(ctx context.Context) error {
	return nil
}

// ListTools 列出工具（默认返回空）。
func (s *BaseServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{}, nil
}

// CallTool 调用工具（默认返回错误）。
func (s *BaseServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	return nil, ErrToolNotFound
}

// ListResources 列出资源（默认返回空）。
func (s *BaseServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{}, nil
}

// ReadResource 读取资源（默认返回错误）。
func (s *BaseServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}

// TextContent 将 Content 转换为文本。
func (c *Content) TextContent() string {
	return c.Text
}

// Text 将 ToolResult 转换为文本。
func (t *ToolResult) Text() string {
	var sb strings.Builder
	for _, content := range t.Content {
		sb.WriteString(content.Text)
	}
	return sb.String()
}
