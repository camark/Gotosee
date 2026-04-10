// Package mcp 提供 MCP 服务器功能。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// ============================================================================
// ComputerControllerServer 计算机控制服务器
// ============================================================================

// ComputerControllerServer 计算机控制 MCP 服务器。
type ComputerControllerServer struct {
	name       string
	version    string
	automation Automation
}

// NewComputerControllerServer 创建计算机控制服务器。
func NewComputerControllerServer() *ComputerControllerServer {
	return &ComputerControllerServer{
		name:       "computer-controller",
		version:    "1.0.0",
		automation: createSystemAutomation(),
	}
}

// Automation 系统自动化接口。
type Automation interface {
	// Screenshot 截取屏幕
	Screenshot() ([]byte, error)
	// Click 点击坐标
	Click(x, y int) error
	// Type 输入文本
	Type(text string) error
	// PressKey 按下按键
	PressKey(key string) error
	// MoveMouse 移动鼠标
	MoveMouse(x, y int) error
	// DragMouse 拖拽鼠标
	DragMouse(fromX, fromY, toX, toY int) error
}

// ListTools 列出所有工具。
func (s *ComputerControllerServer) ListTools(ctx context.Context) ([]Tool, error) {
	tools := []Tool{
		{
			Name:        "screenshot",
			Description: "Take a screenshot of the current screen",
			InputSchema: json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		},
		{
			Name:        "click",
			Description: "Click at a specific coordinate",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"integer"},"y":{"type":"integer"}},"required":["x","y"]}`),
		},
		{
			Name:        "type",
			Description: "Type text using the keyboard",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		},
		{
			Name:        "press_key",
			Description: "Press a specific key",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}},"required":["key"]}`),
		},
		{
			Name:        "move_mouse",
			Description: "Move the mouse to a specific coordinate",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"integer"},"y":{"type":"integer"}},"required":["x","y"]}`),
		},
		{
			Name:        "drag_mouse",
			Description: "Drag the mouse from one coordinate to another",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"fromX":{"type":"integer"},"fromY":{"type":"integer"},"toX":{"type":"integer"},"toY":{"type":"integer"}},"required":["fromX","fromY","toX","toY"]}`),
		},
	}
	return tools, nil
}

// CallTool 调用工具。
func (s *ComputerControllerServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "screenshot":
		return s.handleScreenshot(ctx)
	case "click":
		return s.handleClick(ctx, params)
	case "type":
		return s.handleType(ctx, params)
	case "press_key":
		return s.handlePressKey(ctx, params)
	case "move_mouse":
		return s.handleMoveMouse(ctx, params)
	case "drag_mouse":
		return s.handleDragMouse(ctx, params)
	default:
		return nil, ErrToolNotFound
	}
}

// handleScreenshot 处理截图请求。
func (s *ComputerControllerServer) handleScreenshot(ctx context.Context) (*ToolResult, error) {
	data, err := s.automation.Screenshot()
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Screenshot failed: %v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{
			{
				Type:     "image",
				Data:     json.RawMessage(fmt.Sprintf(`"data:image/png;base64,%s"`, encodeBase64(data))),
				MIMEType: "image/png",
			},
		},
	}, nil
}

// handleClick 处理点击请求。
func (s *ComputerControllerServer) handleClick(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, ErrMCPInvalidParams
	}

	if err := s.automation.Click(req.X, req.Y); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Click failed: %v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Clicked at (%d, %d)", req.X, req.Y)}},
	}, nil
}

// handleType 处理输入请求。
func (s *ComputerControllerServer) handleType(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, ErrMCPInvalidParams
	}

	if err := s.automation.Type(req.Text); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Type failed: %v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Typed: %s", req.Text)}},
	}, nil
}

// handlePressKey 处理按键请求。
func (s *ComputerControllerServer) handlePressKey(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var req struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, ErrMCPInvalidParams
	}

	if err := s.automation.PressKey(req.Key); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("PressKey failed: %v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Pressed key: %s", req.Key)}},
	}, nil
}

// handleMoveMouse 处理移动鼠标请求。
func (s *ComputerControllerServer) handleMoveMouse(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, ErrMCPInvalidParams
	}

	if err := s.automation.MoveMouse(req.X, req.Y); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("MoveMouse failed: %v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Moved mouse to (%d, %d)", req.X, req.Y)}},
	}, nil
}

// handleDragMouse 处理拖拽鼠标请求。
func (s *ComputerControllerServer) handleDragMouse(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var req struct {
		FromX int `json:"fromX"`
		FromY int `json:"fromY"`
		ToX   int `json:"toX"`
		ToY   int `json:"toY"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, ErrMCPInvalidParams
	}

	if err := s.automation.DragMouse(req.FromX, req.FromY, req.ToX, req.ToY); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("DragMouse failed: %v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Dragged mouse")}},
	}, nil
}

// Initialize 初始化服务器。
func (s *ComputerControllerServer) Initialize(ctx context.Context) (*InitializeResult, error) {
	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
		Capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}, nil
}

// ListResources 列出资源。
func (s *ComputerControllerServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{}, nil
}

// ReadResource 读取资源。
func (s *ComputerControllerServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return &ReadResourceResult{}, nil
}

// Shutdown 关闭服务器。
func (s *ComputerControllerServer) Shutdown(ctx context.Context) error {
	return nil
}

// ============================================================================
// 平台相关实现
// ============================================================================

// createSystemAutomation 创建平台相关的自动化实现。
func createSystemAutomation() Automation {
	switch runtime.GOOS {
	case "darwin":
		return &MacOSAutomation{}
	case "windows":
		return &WindowsAutomation{}
	case "linux":
		return &LinuxAutomation{}
	default:
		return &NoopAutomation{}
	}
}

// NoopAutomation 无操作实现。
type NoopAutomation struct{}

func (a *NoopAutomation) Screenshot() ([]byte, error)          { return nil, fmt.Errorf("not implemented") }
func (a *NoopAutomation) Click(x, y int) error                  { return fmt.Errorf("not implemented") }
func (a *NoopAutomation) Type(text string) error                { return nil }
func (a *NoopAutomation) PressKey(key string) error             { return nil }
func (a *NoopAutomation) MoveMouse(x, y int) error              { return nil }
func (a *NoopAutomation) DragMouse(fromX, fromY, toX, toY int) error { return nil }

// MacOSAutomation macOS 实现。
type MacOSAutomation struct{}

func (a *MacOSAutomation) Screenshot() ([]byte, error) {
	// 使用 screencapture 命令
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("screenshot_%d.png", time.Now().UnixNano()))
	cmd := exec.Command("screencapture", "-x", tmpFile)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile)
	return os.ReadFile(tmpFile)
}

func (a *MacOSAutomation) Click(x, y int) error {
	script := fmt.Sprintf(`
		tell application "System Events"
			click at {%d, %d}
		end tell
	`, x, y)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func (a *MacOSAutomation) Type(text string) error {
	// 使用 osascript 输入文本
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, text))
	return cmd.Run()
}

func (a *MacOSAutomation) PressKey(key string) error {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "System Events" to key code %s`, key))
	return cmd.Run()
}

func (a *MacOSAutomation) MoveMouse(x, y int) error {
	script := fmt.Sprintf(`
		tell application "System Events"
			set mouse location to {%d, %d}
		end tell
	`, x, y)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func (a *MacOSAutomation) DragMouse(fromX, fromY, toX, toY int) error {
	// 先移动到起始位置，然后拖拽
	if err := a.MoveMouse(fromX, fromY); err != nil {
		return err
	}
	script := fmt.Sprintf(`
		tell application "System Events"
			mousedown
			set mouse location to {%d, %d}
			mouseup
		end tell
	`, toX, toY)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// WindowsAutomation Windows 实现。
type WindowsAutomation struct{}

func (a *WindowsAutomation) Screenshot() ([]byte, error) {
	// TODO: 使用 PowerShell 或 Go 库截取屏幕
	return nil, fmt.Errorf("not implemented for Windows")
}

func (a *WindowsAutomation) Click(x, y int) error {
	// TODO: 使用 PowerShell 或 Go 库点击
	return nil
}

func (a *WindowsAutomation) Type(text string) error {
	// TODO: 使用 PowerShell 或 Go 库输入
	return nil
}

func (a *WindowsAutomation) PressKey(key string) error {
	// TODO: 使用 PowerShell 或 Go 库按键
	return nil
}

func (a *WindowsAutomation) MoveMouse(x, y int) error {
	// TODO: 使用 PowerShell 或 Go 库移动鼠标
	return nil
}

func (a *WindowsAutomation) DragMouse(fromX, fromY, toX, toY int) error {
	// TODO: 使用 PowerShell 或 Go 库拖拽
	return nil
}

// LinuxAutomation Linux 实现。
type LinuxAutomation struct{}

func (a *LinuxAutomation) Screenshot() ([]byte, error) {
	// 使用 scrot 或 gnome-screenshot
	cmd := exec.Command("scrot", "-s", "/tmp/screenshot.png")
	if err := cmd.Run(); err != nil {
		// 尝试 gnome-screenshot
		cmd = exec.Command("gnome-screenshot", "-f", "/tmp/screenshot.png")
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	}
	return os.ReadFile("/tmp/screenshot.png")
}

func (a *LinuxAutomation) Click(x, y int) error {
	// 使用 xdotool
	cmd := exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", x), fmt.Sprintf("%d", y), "click", "1")
	return cmd.Run()
}

func (a *LinuxAutomation) Type(text string) error {
	cmd := exec.Command("xdotool", "type", "--", text)
	return cmd.Run()
}

func (a *LinuxAutomation) PressKey(key string) error {
	cmd := exec.Command("xdotool", "key", key)
	return cmd.Run()
}

func (a *LinuxAutomation) MoveMouse(x, y int) error {
	cmd := exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))
	return cmd.Run()
}

func (a *LinuxAutomation) DragMouse(fromX, fromY, toX, toY int) error {
	// 使用 xdotool 拖拽
	cmd := exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", fromX), fmt.Sprintf("%d", fromY))
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("xdotool", "mousedown", "1")
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", toX), fmt.Sprintf("%d", toY))
	if err := cmd.Run(); err != nil {
		return err
	}
	return exec.Command("xdotool", "mouseup", "1").Run()
}

// encodeBase64 Base64 编码。
func encodeBase64(data []byte) string {
	// 简单实现，实际应使用 base64 包
	return fmt.Sprintf("%x", data) // 占位符
}
