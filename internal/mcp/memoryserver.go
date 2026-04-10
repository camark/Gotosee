// Package mcp 提供 Memory MCP 服务器。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// Memory MCP 服务器
// ============================================================================

// MemoryServer 记忆管理 MCP 服务器。
type MemoryServer struct {
	*BaseServer
	memories map[string][]MemoryEntry
	filePath string
}

// MemoryEntry 记忆条目。
type MemoryEntry struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	Metadata  Metadata  `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Metadata 记忆元数据。
type Metadata map[string]interface{}

// NewMemoryServer 创建新的记忆服务器。
func NewMemoryServer() *MemoryServer {
	server := &MemoryServer{
		memories: make(map[string][]MemoryEntry),
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "memory-mcp",
			Version:     "1.0.0",
			Instructions: "管理用户记忆，支持添加、查询、删除记忆条目",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	// 加载记忆文件
	home, _ := os.UserHomeDir()
	server.filePath = filepath.Join(home, ".config", "gogo", "memory.json")
	server.loadMemories()

	return server
}

// memoryFile 记忆文件结构。
type memoryFile struct {
	Categories map[string][]MemoryEntry `json:"categories"`
}

// loadMemories 从文件加载记忆。
func (s *MemoryServer) loadMemories() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return // 文件不存在则使用空数据
	}

	var file memoryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return
	}

	s.memories = file.Categories
}

// saveMemories 保存记忆到文件。
func (s *MemoryServer) saveMemories() error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	file := memoryFile{
		Categories: s.memories,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0600)
}

// ListTools 列出所有工具。
func (s *MemoryServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "memory_add",
			Description: "添加新的记忆条目到指定分类",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {"type": "string", "description": "记忆分类"},
					"content": {"type": "string", "description": "记忆内容"},
					"metadata": {"type": "object", "description": "元数据"}
				},
				"required": ["category", "content"]
			}`),
		},
		{
			Name:        "memory_query",
			Description: "查询指定分类的记忆",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {"type": "string", "description": "记忆分类"},
					"limit": {"type": "integer", "description": "返回数量限制", "default": 10}
				},
				"required": ["category"]
			}`),
		},
		{
			Name:        "memory_list",
			Description: "列出所有记忆分类",
			InputSchema: json.RawMessage(`{}`),
		},
		{
			Name:        "memory_delete",
			Description: "删除指定 ID 的记忆",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {"type": "string", "description": "记忆分类"},
					"id": {"type": "string", "description": "记忆 ID"}
				},
				"required": ["category", "id"]
			}`),
		},
		{
			Name:        "memory_clear",
			Description: "清空指定分类的所有记忆",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {"type": "string", "description": "记忆分类"}
				},
				"required": ["category"]
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *MemoryServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "memory_add":
		return s.addMemory(params)
	case "memory_query":
		return s.queryMemory(params)
	case "memory_list":
		return s.listMemoryCategories(params)
	case "memory_delete":
		return s.deleteMemory(params)
	case "memory_clear":
		return s.clearMemory(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// addMemory 添加记忆。
func (s *MemoryServer) addMemory(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Category string                 `json:"category"`
		Content  string                 `json:"content"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	if p.Category == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "分类不能为空"}},
		}, nil
	}

	if p.Content == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "内容不能为空"}},
		}, nil
	}

	now := time.Now()
	entry := MemoryEntry{
		ID:        generateID(),
		Category:  p.Category,
		Content:   p.Content,
		Metadata:  p.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.memories[p.Category] = append(s.memories[p.Category], entry)

	if err := s.saveMemories(); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("保存失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已添加记忆到 '%s' (ID: %s)", p.Category, entry.ID)}},
		Data:    map[string]interface{}{"id": entry.ID, "category": p.Category},
	}, nil
}

// queryMemory 查询记忆。
func (s *MemoryServer) queryMemory(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	if p.Limit <= 0 {
		p.Limit = 10
	}

	entries, ok := s.memories[p.Category]
	if !ok || len(entries) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("分类 '%s' 中没有记忆", p.Category)}},
		}, nil
	}

	// 按更新时间倒序排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
	})

	if p.Limit > len(entries) {
		p.Limit = len(entries)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("分类 '%s' 的记忆 (共 %d 条，显示 %d 条):\n\n", p.Category, len(entries), p.Limit))

	for i := 0; i < p.Limit; i++ {
		e := entries[i]
		sb.WriteString(fmt.Sprintf("[%s] %s\n", e.ID, e.Content))
		if len(e.Metadata) > 0 {
			sb.WriteString(fmt.Sprintf("  元数据：%v\n", e.Metadata))
		}
		sb.WriteString(fmt.Sprintf("  创建：%s, 更新：%s\n\n", e.CreatedAt.Format("2006-01-02"), e.UpdatedAt.Format("2006-01-02")))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"count": p.Limit, "total": len(entries)},
	}, nil
}

// listMemoryCategories 列出分类。
func (s *MemoryServer) listMemoryCategories(params json.RawMessage) (*ToolResult, error) {
	if len(s.memories) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "暂无记忆分类"}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString("记忆分类列表:\n\n")

	categories := make([]string, 0, len(s.memories))
	for cat := range s.memories {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	for _, cat := range categories {
		sb.WriteString(fmt.Sprintf("- %s: %d 条记忆\n", cat, len(s.memories[cat])))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"categories": categories},
	}, nil
}

// deleteMemory 删除记忆。
func (s *MemoryServer) deleteMemory(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Category string `json:"category"`
		ID       string `json:"id"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	entries, ok := s.memories[p.Category]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("分类 '%s' 不存在", p.Category)}},
		}, nil
	}

	found := -1
	for i, e := range entries {
		if e.ID == p.ID {
			found = i
			break
		}
	}

	if found < 0 {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未找到 ID 为 '%s' 的记忆", p.ID)}},
		}, nil
	}

	s.memories[p.Category] = append(entries[:found], entries[found+1:]...)

	if err := s.saveMemories(); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("保存失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已删除记忆 %s", p.ID)}},
		Data:    map[string]interface{}{"deleted": p.ID},
	}, nil
}

// clearMemory 清空分类。
func (s *MemoryServer) clearMemory(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Category string `json:"category"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	if _, ok := s.memories[p.Category]; !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("分类 '%s' 不存在", p.Category)}},
		}, nil
	}

	count := len(s.memories[p.Category])
	s.memories[p.Category] = nil
	delete(s.memories, p.Category)

	if err := s.saveMemories(); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("保存失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已清空分类 '%s' (%d 条记忆)", p.Category, count)}},
		Data:    map[string]interface{}{"cleared": count},
	}, nil
}

// ListResources 列出资源（Memory 服务器暂无资源）。
func (s *MemoryServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *MemoryServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}

// generateID 生成唯一 ID。
func generateID() string {
	return fmt.Sprintf("mem-%d", time.Now().UnixNano())
}
