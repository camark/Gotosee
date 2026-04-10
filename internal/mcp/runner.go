// Package mcp 提供 MCP 服务器运行功能。
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// MCPServerRunner MCP 服务器运行器。
type MCPServerRunner struct {
	server  Server
	mu      sync.Mutex
	running bool
}

// NewMCPServerRunner 创建新的 MCP 服务器运行器。
func NewMCPServerRunner(server Server) *MCPServerRunner {
	return &MCPServerRunner{
		server: server,
	}
}

// Run 运行 MCP 服务器（stdio 传输）。
func (r *MCPServerRunner) Run(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	r.running = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
	}()

	// 初始化服务器（仅记录，不使用结果）
	_, err := r.server.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// 从 stdin 读取请求，写入 stdout
	reader := bufio.NewReader(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return r.server.Shutdown(ctx)
		default:
		}

		// 读取一行（JSON-RPC 消息）
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return r.server.Shutdown(ctx)
			}
			return fmt.Errorf("failed to read request: %w", err)
		}

		// 解析请求
		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   NewRPCError(ErrParse, "Parse error", nil),
			}
			encoder.Encode(resp)
			continue
		}

		// 处理请求
		resp := r.handleRequest(ctx, req)

		// 发送响应
		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("failed to encode response: %w", err)
		}
	}
}

// handleRequest 处理 JSON-RPC 请求。
func (r *MCPServerRunner) handleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	var result interface{}
	var rpcErr *RPCError

	switch req.Method {
	case "initialize":
		result, rpcErr = r.handleInitialize(ctx, req.Params)
	case "tools/list":
		result, rpcErr = r.handleToolsList(ctx, req.Params)
	case "tools/call":
		result, rpcErr = r.handleToolsCall(ctx, req.Params)
	case "resources/list":
		result, rpcErr = r.handleResourcesList(ctx, req.Params)
	case "resources/read":
		result, rpcErr = r.handleResourcesRead(ctx, req.Params)
	case "shutdown":
		err := r.server.Shutdown(ctx)
		if err != nil {
			rpcErr = NewRPCError(ErrInternal, err.Error(), nil)
		} else {
			result = struct{}{}
		}
	default:
		rpcErr = NewRPCError(ErrMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}

	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}

	return resp
}

// handleInitialize 处理初始化请求。
func (r *MCPServerRunner) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
	result, err := r.server.Initialize(ctx)
	if err != nil {
		return nil, NewRPCError(ErrInternal, err.Error(), nil)
	}
	return result, nil
}

// handleToolsList 处理工具列表请求。
func (r *MCPServerRunner) handleToolsList(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
	tools, err := r.server.ListTools(ctx)
	if err != nil {
		return nil, NewRPCError(ErrInternal, err.Error(), nil)
	}
	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall 处理工具调用请求。
func (r *MCPServerRunner) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
	var req struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, NewRPCError(ErrInvalidParams, "Invalid parameters", nil)
	}

	// 将参数重新编码为 JSON
	args, _ := json.Marshal(req.Arguments)

	result, err := r.server.CallTool(ctx, req.Name, args)
	if err != nil {
		return nil, NewRPCError(ErrInternal, err.Error(), nil)
	}

	return map[string]interface{}{
		"content": result.Content,
		"isError": result.IsError,
	}, nil
}

// handleResourcesList 处理资源列表请求。
func (r *MCPServerRunner) handleResourcesList(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
	result, err := r.server.ListResources(ctx)
	if err != nil {
		return nil, NewRPCError(ErrInternal, err.Error(), nil)
	}
	return result, nil
}

// handleResourcesRead 处理资源读取请求。
func (r *MCPServerRunner) handleResourcesRead(ctx context.Context, params json.RawMessage) (interface{}, *RPCError) {
	var req struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, NewRPCError(ErrInvalidParams, "Invalid parameters", nil)
	}

	result, err := r.server.ReadResource(ctx, req.URI)
	if err != nil {
		return nil, NewRPCError(ErrInternal, err.Error(), nil)
	}

	return map[string]interface{}{
		"contents": result.Contents,
	}, nil
}

// ============================================================================
// MCP 命令枚举
// ============================================================================

// McpCommand MCP 命令类型。
type McpCommand string

const (
	McpCommandAutoVisualiser     McpCommand = "autovisualiser"
	McpCommandComputerController McpCommand = "computercontroller"
	McpCommandMemory             McpCommand = "memory"
	McpCommandTutorial           McpCommand = "tutorial"
	McpCommandFetch              McpCommand = "fetch"
	McpCommandGit                McpCommand = "git"
	McpCommandFilesystem         McpCommand = "filesystem"
	McpCommandTime               McpCommand = "time"
	McpCommandEnvironment        McpCommand = "environment"
	McpCommandProcess            McpCommand = "process"
	McpCommandDatabase           McpCommand = "database"
	McpCommandHttpClient         McpCommand = "http-client"
)

// String 返回命令字符串。
func (c McpCommand) String() string {
	return string(c)
}

// ParseMcpCommand 解析 MCP 命令。
func ParseMcpCommand(s string) (McpCommand, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "autovisualiser":
		return McpCommandAutoVisualiser, nil
	case "computercontroller", "computer_controller":
		return McpCommandComputerController, nil
	case "memory":
		return McpCommandMemory, nil
	case "tutorial":
		return McpCommandTutorial, nil
	case "fetch":
		return McpCommandFetch, nil
	case "git":
		return McpCommandGit, nil
	case "filesystem", "file":
		return McpCommandFilesystem, nil
	case "time":
		return McpCommandTime, nil
	case "environment", "env":
		return McpCommandEnvironment, nil
	case "process", "proc":
		return McpCommandProcess, nil
	case "database", "db":
		return McpCommandDatabase, nil
	case "http-client", "http", "curl":
		return McpCommandHttpClient, nil
	default:
		return "", fmt.Errorf("unknown MCP command: %s", s)
	}
}

// NewServerForCommand 为命令创建服务器。
func NewServerForCommand(cmd McpCommand) Server {
	// 使用注册中心获取服务器
	server, err := Get(string(cmd))
	if err != nil {
		// 回退到简单服务器
		return &SimpleServer{name: cmd.String(), version: "1.0.0"}
	}
	return server
}

// SimpleServer 简单的服务器实现。
type SimpleServer struct {
	name    string
	version string
}

// Initialize 初始化服务器。
func (s *SimpleServer) Initialize(ctx context.Context) (*InitializeResult, error) {
	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}, nil
}

// ListTools 列出工具。
func (s *SimpleServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{}, nil
}

// CallTool 调用工具。
func (s *SimpleServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Simple server: no tools available")}},
	}, nil
}

// ListResources 列出资源。
func (s *SimpleServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{}, nil
}

// ReadResource 读取资源。
func (s *SimpleServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return &ReadResourceResult{}, nil
}

// Shutdown 关闭服务器。
func (s *SimpleServer) Shutdown(ctx context.Context) error {
	return nil
}
