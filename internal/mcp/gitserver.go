// Package mcp 提供 Git MCP 服务器。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ============================================================================
// Git MCP 服务器
// ============================================================================

// GitServer Git 操作 MCP 服务器。
type GitServer struct {
	*BaseServer
	workDir string
}

// NewGitServer 创建新的 Git 服务器。
func NewGitServer() *GitServer {
	server := &GitServer{
		workDir: "",
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "git-mcp",
			Version:     "1.0.0",
			Instructions: "执行 Git 操作，查看状态、日志、分支等",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// SetWorkDir 设置工作目录。
func (s *GitServer) SetWorkDir(dir string) {
	s.workDir = dir
}

// ListTools 列出所有工具。
func (s *GitServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "git_status",
			Description: "查看 Git 仓库状态",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径（可选，默认当前目录）"}
				}
			}`),
		},
		{
			Name:        "git_log",
			Description: "查看 Git 提交日志",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径"},
					"limit": {"type": "integer", "description": "显示数量", "default": 10},
					"oneline": {"type": "boolean", "description": "单行格式", "default": true}
				}
			}`),
		},
		{
			Name:        "git_branch",
			Description: "查看 Git 分支",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径"},
					"all": {"type": "boolean", "description": "显示所有分支", "default": false}
				}
			}`),
		},
		{
			Name:        "git_diff",
			Description: "查看 Git 差异",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径"},
					"cached": {"type": "boolean", "description": "查看暂存区差异", "default": false},
					"path": {"type": "string", "description": "文件路径（可选）"}
				}
			}`),
		},
		{
			Name:        "git_add",
			Description: "添加文件到暂存区",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径"},
					"files": {"type": "array", "items": {"type": "string"}, "description": "文件列表"}
				},
				"required": ["files"]
			}`),
		},
		{
			Name:        "git_commit",
			Description: "创建 Git 提交",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径"},
					"message": {"type": "string", "description": "提交信息"},
					"all": {"type": "boolean", "description": "自动添加所有修改", "default": false}
				},
				"required": ["message"]
			}`),
		},
		{
			Name:        "git_push",
			Description: "推送 Git 提交",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径"},
					"remote": {"type": "string", "description": "远程仓库", "default": "origin"},
					"branch": {"type": "string", "description": "分支名称", "default": "当前分支"}
				}
			}`),
		},
		{
			Name:        "git_pull",
			Description: "拉取 Git 提交",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_path": {"type": "string", "description": "仓库路径"},
					"remote": {"type": "string", "description": "远程仓库", "default": "origin"}
				}
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *GitServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "git_status":
		return s.gitStatus(params)
	case "git_log":
		return s.gitLog(params)
	case "git_branch":
		return s.gitBranch(params)
	case "git_diff":
		return s.gitDiff(params)
	case "git_add":
		return s.gitAdd(params)
	case "git_commit":
		return s.gitCommit(params)
	case "git_push":
		return s.gitPush(params)
	case "git_pull":
		return s.gitPull(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// runGit 执行 Git 命令。
func (s *GitServer) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.workDir
	if s.workDir == "" {
		cmd.Dir, _ = os.Getwd()
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return string(output), nil
}

// runGitErr 执行 Git 命令（不返回输出）。
func (s *GitServer) runGitErr(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.workDir
	if s.workDir == "" {
		cmd.Dir, _ = os.Getwd()
	}

	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

// gitStatus 查看状态。
func (s *GitServer) gitStatus(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string `json:"repo_path"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}

	output, err := s.runGit("status")
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: output}},
		Data:    map[string]interface{}{"status": output},
	}, nil
}

// gitLog 查看日志。
func (s *GitServer) gitLog(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string `json:"repo_path"`
		Limit    int    `json:"limit"`
		Oneline  bool   `json:"oneline"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}
	if p.Limit <= 0 {
		p.Limit = 10
	}

	args := []string{"log"}
	if p.Oneline {
		args = append(args, "--oneline")
	}
	args = append(args, fmt.Sprintf("-n %d", p.Limit))

	output, err := s.runGit(args...)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: output}},
		Data:    map[string]interface{}{"log": output},
	}, nil
}

// gitBranch 查看分支。
func (s *GitServer) gitBranch(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string `json:"repo_path"`
		All      bool   `json:"all"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}

	args := []string{"branch"}
	if p.All {
		args = append(args, "-a")
	}

	output, err := s.runGit(args...)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: output}},
		Data:    map[string]interface{}{"branches": output},
	}, nil
}

// gitDiff 查看差异。
func (s *GitServer) gitDiff(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string `json:"repo_path"`
		Cached   bool   `json:"cached"`
		Path     string `json:"path"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}

	args := []string{"diff"}
	if p.Cached {
		args = append(args, "--cached")
	}
	if p.Path != "" {
		args = append(args, "--", p.Path)
	}

	output, err := s.runGit(args...)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: output}},
		Data:    map[string]interface{}{"diff": output},
	}, nil
}

// gitAdd 添加文件。
func (s *GitServer) gitAdd(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string   `json:"repo_path"`
		Files    []string `json:"files"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}

	if len(p.Files) == 0 {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "文件列表不能为空"}},
		}, nil
	}

	args := append([]string{"add"}, p.Files...)
	if err := s.runGitErr(args...); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已添加 %d 个文件到暂存区", len(p.Files))}},
		Data:    map[string]interface{}{"added": p.Files},
	}, nil
}

// gitCommit 创建提交。
func (s *GitServer) gitCommit(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string `json:"repo_path"`
		Message  string `json:"message"`
		All      bool   `json:"all"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}

	if p.Message == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "提交信息不能为空"}},
		}, nil
	}

	args := []string{"commit"}
	if p.All {
		args = append(args, "-a")
	}
	args = append(args, "-m", p.Message)

	if err := s.runGitErr(args...); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已创建提交：%s", p.Message)}},
		Data:    map[string]interface{}{"message": p.Message},
	}, nil
}

// gitPush 推送提交。
func (s *GitServer) gitPush(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string `json:"repo_path"`
		Remote   string `json:"remote"`
		Branch   string `json:"branch"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}
	if p.Remote == "" {
		p.Remote = "origin"
	}

	args := []string{"push", p.Remote}
	if p.Branch != "" {
		args = append(args, p.Branch)
	}

	output, err := s.runGit(args...)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: output}},
		Data:    map[string]interface{}{"output": output},
	}, nil
}

// gitPull 拉取提交。
func (s *GitServer) gitPull(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		RepoPath string `json:"repo_path"`
		Remote   string `json:"remote"`
	}
	json.Unmarshal(params, &p)

	if p.RepoPath != "" {
		s.workDir = p.RepoPath
	}
	if p.Remote == "" {
		p.Remote = "origin"
	}

	output, err := s.runGit("pull", p.Remote)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: output}},
		Data:    map[string]interface{}{"output": output},
	}, nil
}

// ListResources 列出资源。
func (s *GitServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *GitServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}

// isGitRepo 检查目录是否为 Git 仓库。
func isGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}
