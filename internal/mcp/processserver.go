// Package mcp 提供 Process MCP 服务器（进程管理）。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ============================================================================
// Process MCP 服务器
// ============================================================================

// ProcessServer 进程管理 MCP 服务器。
type ProcessServer struct {
	*BaseServer
	processes map[string]*exec.Cmd
}

// NewProcessServer 创建新的进程服务器。
func NewProcessServer() *ProcessServer {
	server := &ProcessServer{
		processes: make(map[string]*exec.Cmd),
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "process-mcp",
			Version:     "1.0.0",
			Instructions: "管理系统进程：启动、停止、查看进程",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *ProcessServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "run_command",
			Description: "运行系统命令",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string", "description": "要运行的命令"},
					"args": {"type": "array", "items": {"type": "string"}, "description": "命令参数"},
					"timeout": {"type": "integer", "description": "超时时间（秒）", "default": 30}
				},
				"required": ["command"]
			}`),
		},
		{
			Name:        "run_background",
			Description: "后台运行进程",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string", "description": "要运行的命令"},
					"args": {"type": "array", "items": {"type": "string"}, "description": "命令参数"},
					"name": {"type": "string", "description": "进程名称"}
				},
				"required": ["command", "name"]
			}`),
		},
		{
			Name:        "stop_process",
			Description: "停止后台进程",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "进程名称"}
				},
				"required": ["name"]
			}`),
		},
		{
			Name:        "list_processes",
			Description: "列出所有后台进程",
			InputSchema: json.RawMessage(`{}`),
		},
		{
			Name:        "get_process_status",
			Description: "获取进程状态",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "进程名称"}
				},
				"required": ["name"]
			}`),
		},
		{
			Name:        "kill_process",
			Description: "强制终止进程",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pid": {"type": "integer", "description": "进程 ID"}
				},
				"required": ["pid"]
			}`),
		},
		{
			Name:        "get_system_info",
			Description: "获取系统信息",
			InputSchema: json.RawMessage(`{}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *ProcessServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "run_command":
		return s.runCommand(params)
	case "run_background":
		return s.runBackground(params)
	case "stop_process":
		return s.stopProcess(params)
	case "list_processes":
		return s.listProcesses(params)
	case "get_process_status":
		return s.getProcessStatus(params)
	case "kill_process":
		return s.killProcess(params)
	case "get_system_info":
		return s.getSystemInfo(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// runCommand 运行命令。
func (s *ProcessServer) runCommand(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Timeout int      `json:"timeout"`
	}
	json.Unmarshal(params, &p)

	if p.Timeout <= 0 {
		p.Timeout = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.Command, p.Args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("命令执行超时（%d 秒）", p.Timeout)}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString("输出:\n")
	sb.WriteString(string(output))
	if err != nil {
		sb.WriteString(fmt.Sprintf("\n错误：%v", err))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"output": string(output),
			"error":  err,
		},
	}, nil
}

// runBackground 后台运行。
func (s *ProcessServer) runBackground(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Name    string   `json:"name"`
	}
	json.Unmarshal(params, &p)

	if p.Name == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "进程名称不能为空"}},
		}, nil
	}

	cmd := exec.Command(p.Command, p.Args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("启动失败：%v", err)}},
		}, nil
	}

	s.processes[p.Name] = cmd

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 进程 '%s' 已启动 (PID: %d)", p.Name, cmd.Process.Pid)}},
		Data: map[string]interface{}{
			"name": p.Name,
			"pid":  cmd.Process.Pid,
		},
	}, nil
}

// stopProcess 停止进程。
func (s *ProcessServer) stopProcess(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name string `json:"name"`
	}
	json.Unmarshal(params, &p)

	cmd, ok := s.processes[p.Name]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("进程 '%s' 不存在", p.Name)}},
		}, nil
	}

	if err := cmd.Process.Kill(); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("停止失败：%v", err)}},
		}, nil
	}

	delete(s.processes, p.Name)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 进程 '%s' 已停止", p.Name)}},
		Data: map[string]interface{}{
			"name": p.Name,
		},
	}, nil
}

// listProcesses 列出进程。
func (s *ProcessServer) listProcesses(params json.RawMessage) (*ToolResult, error) {
	if len(s.processes) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "暂无后台进程"}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("后台进程列表 (共 %d 个):\n\n", len(s.processes)))
	for name, cmd := range s.processes {
		status := "运行中"
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			status = "已结束"
		}
		sb.WriteString(fmt.Sprintf("- %s (PID: %d, 状态：%s)\n", name, cmd.Process.Pid, status))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"count": len(s.processes),
		},
	}, nil
}

// getProcessStatus 获取进程状态。
func (s *ProcessServer) getProcessStatus(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name string `json:"name"`
	}
	json.Unmarshal(params, &p)

	cmd, ok := s.processes[p.Name]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("进程 '%s' 不存在", p.Name)}},
		}, nil
	}

	status := "运行中"
	if cmd.ProcessState != nil {
		if cmd.ProcessState.Exited() {
			status = fmt.Sprintf("已结束 (退出码：%d)", cmd.ProcessState.ExitCode())
		} else {
			status = "已停止"
		}
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("进程 '%s': %s, PID: %d", p.Name, status, cmd.Process.Pid)}},
		Data: map[string]interface{}{
			"name":   p.Name,
			"pid":    cmd.Process.Pid,
			"status": status,
		},
	}, nil
}

// killProcess 强制终止进程。
func (s *ProcessServer) killProcess(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		PID int `json:"pid"`
	}
	json.Unmarshal(params, &p)

	process, err := os.FindProcess(p.PID)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("找不到进程：%v", err)}},
		}, nil
	}

	if err := process.Kill(); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("终止失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 进程 %d 已终止", p.PID)}},
		Data: map[string]interface{}{
			"pid": p.PID,
		},
	}, nil
}

// getSystemInfo 获取系统信息。
func (s *ProcessServer) getSystemInfo(params json.RawMessage) (*ToolResult, error) {
	var sb strings.Builder
	sb.WriteString("系统信息:\n\n")
	sb.WriteString(fmt.Sprintf("操作系统：%s\n", runtime.GOOS))
	sb.WriteString(fmt.Sprintf("架构：%s\n", runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("CPU 核心数：%d\n", runtime.NumCPU()))
	sb.WriteString(fmt.Sprintf("Go 版本：%s\n", runtime.Version()))

	// 获取主机名
	hostname, _ := os.Hostname()
	sb.WriteString(fmt.Sprintf("主机名：%s\n", hostname))

	// 获取工作目录
	wd, _ := os.Getwd()
	sb.WriteString(fmt.Sprintf("工作目录：%s\n", wd))

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
			"cpus":    runtime.NumCPU(),
			"go_version": runtime.Version(),
		},
	}, nil
}

// ListResources 列出资源。
func (s *ProcessServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *ProcessServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
