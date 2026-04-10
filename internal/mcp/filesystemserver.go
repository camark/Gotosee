// Package mcp 提供 Filesystem MCP 服务器。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ============================================================================
// Filesystem MCP 服务器
// ============================================================================

// FilesystemServer 文件系统 MCP 服务器。
type FilesystemServer struct {
	*BaseServer
	allowedDirs []string // 允许的目录列表
}

// NewFilesystemServer 创建新的文件系统服务器。
func NewFilesystemServer(allowedDirs []string) *FilesystemServer {
	server := &FilesystemServer{
		allowedDirs: allowedDirs,
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "filesystem-mcp",
			Version:     "1.0.0",
			Instructions: "文件系统操作：读取、写入、删除、搜索文件",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *FilesystemServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "read_file",
			Description: "读取文件内容",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "文件路径"}
				},
				"required": ["path"]
			}`),
		},
		{
			Name:        "write_file",
			Description: "写入文件内容",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "文件路径"},
					"content": {"type": "string", "description": "文件内容"},
					"append": {"type": "boolean", "description": "追加模式", "default": false}
				},
				"required": ["path", "content"]
			}`),
		},
		{
			Name:        "delete_file",
			Description: "删除文件",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "文件路径"}
				},
				"required": ["path"]
			}`),
		},
		{
			Name:        "list_directory",
			Description: "列出目录内容",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "目录路径"}
				},
				"required": ["path"]
			}`),
		},
		{
			Name:        "create_directory",
			Description: "创建目录",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "目录路径"}
				},
				"required": ["path"]
			}`),
		},
		{
			Name:        "search_files",
			Description: "搜索文件",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "搜索起始路径"},
					"pattern": {"type": "string", "description": "文件匹配模式"},
					"recursive": {"type": "boolean", "description": "递归搜索", "default": true}
				},
				"required": ["pattern"]
			}`),
		},
		{
			Name:        "get_file_info",
			Description: "获取文件信息",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "文件路径"}
				},
				"required": ["path"]
			}`),
		},
		{
			Name:        "move_file",
			Description: "移动/重命名文件",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"source": {"type": "string", "description": "源文件路径"},
					"destination": {"type": "string", "description": "目标路径"}
				},
				"required": ["source", "destination"]
			}`),
		},
		{
			Name:        "copy_file",
			Description: "复制文件",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"source": {"type": "string", "description": "源文件路径"},
					"destination": {"type": "string", "description": "目标路径"}
				},
				"required": ["source", "destination"]
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *FilesystemServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "read_file":
		return s.readFile(params)
	case "write_file":
		return s.writeFile(params)
	case "delete_file":
		return s.deleteFile(params)
	case "list_directory":
		return s.listDirectory(params)
	case "create_directory":
		return s.createDirectory(params)
	case "search_files":
		return s.searchFiles(params)
	case "get_file_info":
		return s.getFileInfo(params)
	case "move_file":
		return s.moveFile(params)
	case "copy_file":
		return s.copyFile(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// validatePath 验证路径是否在允许的目录内。
func (s *FilesystemServer) validatePath(path string) error {
	if len(s.allowedDirs) == 0 {
		return nil // 没有限制
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	for _, dir := range s.allowedDirs {
		absDir, _ := filepath.Abs(dir)
		if strings.HasPrefix(absPath, absDir) {
			return nil
		}
	}

	return fmt.Errorf("路径 '%s' 不在允许的目录范围内", path)
}

// readFile 读取文件。
func (s *FilesystemServer) readFile(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Path string `json:"path"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	data, err := os.ReadFile(p.Path)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("读取失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: string(data)}},
		Data:    map[string]interface{}{"path": p.Path, "size": len(data)},
	}, nil
}

// writeFile 写入文件。
func (s *FilesystemServer) writeFile(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Append  bool   `json:"append"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	var err error
	if p.Append {
		err = os.WriteFile(p.Path, []byte(p.Content), 0644)
	} else {
		err = os.WriteFile(p.Path, []byte(p.Content), 0644)
	}

	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("写入失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已写入文件：%s", p.Path)}},
		Data:    map[string]interface{}{"path": p.Path},
	}, nil
}

// deleteFile 删除文件。
func (s *FilesystemServer) deleteFile(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Path string `json:"path"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	if err := os.Remove(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("删除失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已删除：%s", p.Path)}},
		Data:    map[string]interface{}{"path": p.Path},
	}, nil
}

// listDirectory 列出目录。
func (s *FilesystemServer) listDirectory(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Path string `json:"path"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	entries, err := os.ReadDir(p.Path)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("读取目录失败：%v", err)}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("目录：%s\n\n", p.Path))
	for _, e := range entries {
		info, _ := e.Info()
		if e.IsDir() {
			sb.WriteString(fmt.Sprintf("📁 %s/\n", e.Name()))
		} else {
			sb.WriteString(fmt.Sprintf("📄 %s (%d bytes)\n", e.Name(), info.Size()))
		}
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"count": len(entries)},
	}, nil
}

// createDirectory 创建目录。
func (s *FilesystemServer) createDirectory(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Path string `json:"path"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	if err := os.MkdirAll(p.Path, 0755); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("创建目录失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已创建目录：%s", p.Path)}},
		Data:    map[string]interface{}{"path": p.Path},
	}, nil
}

// searchFiles 搜索文件。
func (s *FilesystemServer) searchFiles(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Path      string `json:"path"`
		Pattern   string `json:"pattern"`
		Recursive bool   `json:"recursive"`
	}
	json.Unmarshal(params, &p)

	if p.Path == "" {
		p.Path, _ = os.Getwd()
	}

	if err := s.validatePath(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	var matches []string

	var walkFn filepath.WalkFunc = func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		matched, _ := filepath.Match(p.Pattern, info.Name())
		if matched {
			matches = append(matches, path)
		}

		if !p.Recursive && !info.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}

	if err := filepath.Walk(p.Path, walkFn); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("搜索失败：%v", err)}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个匹配 '%s' 的文件:\n\n", len(matches), p.Pattern))
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("- %s\n", m))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"count": len(matches), "files": matches},
	}, nil
}

// getFileInfo 获取文件信息。
func (s *FilesystemServer) getFileInfo(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Path string `json:"path"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Path); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	info, err := os.Stat(p.Path)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("获取信息失败：%v", err)}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("文件信息：%s\n\n", p.Path))
	sb.WriteString(fmt.Sprintf("名称：%s\n", info.Name()))
	sb.WriteString(fmt.Sprintf("大小：%d bytes\n", info.Size()))
	sb.WriteString(fmt.Sprintf("模式：%s\n", info.Mode()))
	sb.WriteString(fmt.Sprintf("修改时间：%s\n", info.ModTime().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("是目录：%v\n", info.IsDir()))

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"name": info.Name(), "size": info.Size(), "isDir": info.IsDir()},
	}, nil
}

// moveFile 移动文件。
func (s *FilesystemServer) moveFile(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Source); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	if err := s.validatePath(p.Destination); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	if err := os.Rename(p.Source, p.Destination); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("移动失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已移动：%s -> %s", p.Source, p.Destination)}},
		Data:    map[string]interface{}{"source": p.Source, "destination": p.Destination},
	}, nil
}

// copyFile 复制文件。
func (s *FilesystemServer) copyFile(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	json.Unmarshal(params, &p)

	if err := s.validatePath(p.Source); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	if err := s.validatePath(p.Destination); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: err.Error()}},
		}, nil
	}

	srcFile, err := os.Open(p.Source)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("打开源文件失败：%v", err)}},
		}, nil
	}
	defer srcFile.Close()

	dstFile, err := os.Create(p.Destination)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("创建目标文件失败：%v", err)}},
		}, nil
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("复制失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已复制：%s -> %s", p.Source, p.Destination)}},
		Data:    map[string]interface{}{"source": p.Source, "destination": p.Destination},
	}, nil
}

// ListResources 列出资源。
func (s *FilesystemServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *FilesystemServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
