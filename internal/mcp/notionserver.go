// Package mcp 提供 Notion MCP 服务器（Notion 集成）。
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ============================================================================
// Notion MCP 服务器
// ============================================================================

// NotionServer Notion 集成 MCP 服务器。
type NotionServer struct {
	*BaseServer
	apiKey     string
	httpClient *http.Client
}

// NewNotionServer 创建新的 Notion 服务器。
func NewNotionServer() *NotionServer {
	server := &NotionServer{
		apiKey: os.Getenv("NOTION_API_KEY"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "notion-mcp",
			Version:     "1.0.0",
			Instructions: "Notion 集成：创建页面、获取内容、搜索、更新数据库",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *NotionServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "notion_create_page",
			Description: "在 Notion 中创建新页面",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"parent_database_id": {"type": "string", "description": "父数据库 ID"},
					"title": {"type": "string", "description": "页面标题"},
					"content": {"type": "string", "description": "页面内容"}
				},
				"required": ["parent_database_id", "title"]
			}`),
		},
		{
			Name:        "notion_get_page",
			Description: "获取 Notion 页面内容",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"page_id": {"type": "string", "description": "页面 ID"}
				},
				"required": ["page_id"]
			}`),
		},
		{
			Name:        "notion_search",
			Description: "搜索 Notion 页面和数据库",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "搜索关键词"},
					"filter": {"type": "string", "description": "过滤条件 (page/database)", "default": "page"}
				}
			}`),
		},
		{
			Name:        "notion_update_page",
			Description: "更新 Notion 页面内容",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"page_id": {"type": "string", "description": "页面 ID"},
					"title": {"type": "string", "description": "新标题"},
					"content": {"type": "string", "description": "新内容"}
				},
				"required": ["page_id"]
			}`),
		},
		{
			Name:        "notion_delete_page",
			Description: "删除（归档）Notion 页面",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"page_id": {"type": "string", "description": "页面 ID"}
				},
				"required": ["page_id"]
			}`),
		},
		{
			Name:        "notion_list_databases",
			Description: "列出所有数据库",
			InputSchema: json.RawMessage(`{}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *NotionServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "notion_create_page":
		return s.createPage(ctx, params)
	case "notion_get_page":
		return s.getPage(ctx, params)
	case "notion_search":
		return s.search(ctx, params)
	case "notion_update_page":
		return s.updatePage(ctx, params)
	case "notion_delete_page":
		return s.deletePage(ctx, params)
	case "notion_list_databases":
		return s.listDatabases(ctx, params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// CreatePageRequest 创建页面请求。
type CreatePageRequest struct {
	Parent   ParentInfo `json:"parent"`
	Properties struct {
		Title []TitleInfo `json:"title"`
	} `json:"properties"`
	Children []BlockContent `json:"children,omitempty"`
}

// ParentInfo 父级信息。
type ParentInfo struct {
	DatabaseID string `json:"database_id"`
}

// TitleInfo 标题信息。
type TitleInfo struct {
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
}

// BlockContent 块内容。
type BlockContent struct {
	Object    string `json:"object"`
	Type      string `json:"type"`
	Paragraph struct {
		RichText []RichText `json:"rich_text"`
	} `json:"paragraph"`
}

// RichText 富文本。
type RichText struct {
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
}

// RichTextContent 富文本内容。
type RichTextContent struct {
	Content string `json:"content"`
}

// createPage 创建页面。
func (s *NotionServer) createPage(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var p struct {
		ParentDatabaseID string `json:"parent_database_id"`
		Title            string `json:"title"`
		Content          string `json:"content"`
	}
	json.Unmarshal(params, &p)

	if p.ParentDatabaseID == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "错误：缺少 parent_database_id"}},
		}, nil
	}

	reqBody := CreatePageRequest{}
	reqBody.Parent.DatabaseID = p.ParentDatabaseID
	reqBody.Properties.Title = []TitleInfo{{
		Text: RichTextContent{Content: p.Title},
	}}

	// 如果有内容，添加块
	if p.Content != "" {
		reqBody.Children = []BlockContent{{
			Object: "block",
			Type:   "paragraph",
			Paragraph: struct {
				RichText []RichText `json:"rich_text"`
			}{
				RichText: []RichText{{
					Text: RichTextContent{Content: p.Content},
				}},
			},
		}}
	}

	body, _ := json.Marshal(reqBody)
	respBody, err := s.notionRequest(ctx, "POST", "https://api.notion.com/v1/pages", body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("创建失败：%v", err)}},
		}, nil
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	pageURL, _ := result["url"].(string)
	pageID, _ := result["id"].(string)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 页面已创建\n标题：%s\n链接：%s", p.Title, pageURL)}},
		Data: map[string]interface{}{
			"id":  pageID,
			"url": pageURL,
		},
	}, nil
}

// getPage 获取页面。
func (s *NotionServer) getPage(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var p struct {
		PageID string `json:"page_id"`
	}
	json.Unmarshal(params, &p)

	if p.PageID == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "错误：缺少 page_id"}},
		}, nil
	}

	// 获取页面属性
	respBody, err := s.notionRequest(ctx, "GET", fmt.Sprintf("https://api.notion.com/v1/pages/%s", p.PageID), nil)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("获取失败：%v", err)}},
		}, nil
	}

	var page map[string]interface{}
	json.Unmarshal(respBody, &page)

	// 获取页面块
	blocksResp, err := s.notionRequest(ctx, "GET", fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children", p.PageID), nil)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("获取内容失败：%v", err)}},
		}, nil
	}

	var blocks map[string]interface{}
	json.Unmarshal(blocksResp, &blocks)

	var sb strings.Builder
	sb.WriteString("页面内容:\n\n")

	// 解析块
	if results, ok := blocks["results"].([]interface{}); ok {
		for _, block := range results {
			if b, ok := block.(map[string]interface{}); ok {
				if para, ok := b["paragraph"].(map[string]interface{}); ok {
					if richText, ok := para["rich_text"].([]interface{}); ok {
						for _, rt := range richText {
							if r, ok := rt.(map[string]interface{}); ok {
								if text, ok := r["text"].(map[string]interface{}); ok {
									if content, ok := text["content"].(string); ok {
										sb.WriteString(content)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    page,
	}, nil
}

// search 搜索。
func (s *NotionServer) search(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Query  string `json:"query"`
		Filter string `json:"filter"`
	}
	json.Unmarshal(params, &p)

	reqBody := map[string]interface{}{}
	if p.Query != "" {
		reqBody["query"] = p.Query
	}
	if p.Filter != "" {
		reqBody["filter"] = map[string]string{"value": p.Filter, "property": "object"}
	}

	body, _ := json.Marshal(reqBody)
	respBody, err := s.notionRequest(ctx, "POST", "https://api.notion.com/v1/search", body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("搜索失败：%v", err)}},
		}, nil
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	var sb strings.Builder
	sb.WriteString("搜索结果:\n\n")

	if results, ok := result["results"].([]interface{}); ok {
		for i, item := range results {
			if m, ok := item.(map[string]interface{}); ok {
				title := "无标题"
				if props, ok := m["properties"].(map[string]interface{}); ok {
					if t, ok := props["title"].(map[string]interface{}); ok {
						if id, ok := t["id"].(string); ok {
							title = id
						}
					}
				}
				url, _ := m["url"].(string)
				objType, _ := m["object"].(string)
				sb.WriteString(fmt.Sprintf("%d. [%s] %s\n   %s\n\n", i+1, objType, title, url))
			}
		}
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    result,
	}, nil
}

// updatePage 更新页面。
func (s *NotionServer) updatePage(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var p struct {
		PageID  string `json:"page_id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	json.Unmarshal(params, &p)

	if p.PageID == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "错误：缺少 page_id"}},
		}, nil
	}

	reqBody := map[string]interface{}{}
	properties := make(map[string]interface{})

	if p.Title != "" {
		properties["title"] = []TitleInfo{{
			Text: RichTextContent{Content: p.Title},
		}}
	}

	if len(properties) > 0 {
		reqBody["properties"] = properties
	}

	body, _ := json.Marshal(reqBody)
	respBody, err := s.notionRequest(ctx, "PATCH", fmt.Sprintf("https://api.notion.com/v1/pages/%s", p.PageID), body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("更新失败：%v", err)}},
		}, nil
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: "✓ 页面已更新"}},
		Data:    result,
	}, nil
}

// deletePage 删除页面。
func (s *NotionServer) deletePage(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var p struct {
		PageID string `json:"page_id"`
	}
	json.Unmarshal(params, &p)

	if p.PageID == "" {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: "错误：缺少 page_id"}},
		}, nil
	}

	reqBody := map[string]interface{}{
		"archived": true,
	}

	body, _ := json.Marshal(reqBody)
	_, err := s.notionRequest(ctx, "PATCH", fmt.Sprintf("https://api.notion.com/v1/pages/%s", p.PageID), body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("删除失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: "✓ 页面已归档"}},
	}, nil
}

// listDatabases 列出数据库。
func (s *NotionServer) listDatabases(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	reqBody := map[string]interface{}{
		"filter": map[string]string{"value": "database", "property": "object"},
	}

	body, _ := json.Marshal(reqBody)
	respBody, err := s.notionRequest(ctx, "POST", "https://api.notion.com/v1/search", body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("搜索失败：%v", err)}},
		}, nil
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	var sb strings.Builder
	sb.WriteString("数据库列表:\n\n")

	if results, ok := result["results"].([]interface{}); ok {
		for i, item := range results {
			if m, ok := item.(map[string]interface{}); ok {
				title := "无标题"
				if t, ok := m["title"].([]interface{}); ok && len(t) > 0 {
					if r, ok := t[0].(map[string]interface{}); ok {
						if text, ok := r["plain_text"].(string); ok {
							title = text
						}
					}
				}
				id, _ := m["id"].(string)
				sb.WriteString(fmt.Sprintf("%d. %s\n   ID: %s\n\n", i+1, title, id))
			}
		}
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    result,
	}, nil
}

// notionRequest 发送 Notion API 请求。
func (s *NotionServer) notionRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Notion-Version", "2022-06-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Notion API error: %s", string(respBody))
	}

	return respBody, nil
}

// ListResources 列出资源。
func (s *NotionServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{}, nil
}

// ReadResource 读取资源。
func (s *NotionServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
