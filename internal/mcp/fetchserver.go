// Package mcp 提供 Fetch MCP 服务器（网页抓取）。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ============================================================================
// Fetch MCP 服务器
// ============================================================================

// FetchServer 网页抓取 MCP 服务器。
type FetchServer struct {
	*BaseServer
	client *http.Client
}

// NewFetchServer 创建新的抓取服务器。
func NewFetchServer() *FetchServer {
	server := &FetchServer{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "fetch-mcp",
			Version:     "1.0.0",
			Instructions: "抓取网页内容，提取文本和信息",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *FetchServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "fetch_url",
			Description: "抓取网页内容并提取文本",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "要抓取的 URL"},
					"selector": {"type": "string", "description": "CSS 选择器（可选）"},
					"full_html": {"type": "boolean", "description": "是否返回完整 HTML", "default": false}
				},
				"required": ["url"]
			}`),
		},
		{
			Name:        "fetch_summary",
			Description: "抓取网页并生成摘要",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "要抓取的 URL"}
				},
				"required": ["url"]
			}`),
		},
		{
			Name:        "fetch_links",
			Description: "提取网页中的所有链接",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "要抓取的 URL"},
					"internal": {"type": "boolean", "description": "只返回内部链接", "default": false}
				},
				"required": ["url"]
			}`),
		},
		{
			Name:        "fetch_metadata",
			Description: "提取网页元数据（标题、描述、关键词等）",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"url": {"type": "string", "description": "要抓取的 URL"}
				},
				"required": ["url"]
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *FetchServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "fetch_url":
		return s.fetchURL(params)
	case "fetch_summary":
		return s.fetchSummary(params)
	case "fetch_links":
		return s.fetchLinks(params)
	case "fetch_metadata":
		return s.fetchMetadata(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// fetchURL 抓取网页。
func (s *FetchServer) fetchURL(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL      string `json:"url"`
		Selector string `json:"selector"`
		FullHTML bool   `json:"full_html"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	// 验证 URL
	if _, err := url.Parse(p.URL); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效 URL: %v", err)}},
		}, nil
	}

	// 发送请求
	req, err := http.NewRequest("GET", p.URL, nil)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("请求失败：%v", err)}},
		}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Goose/1.0; +https://github.com/camark/Gotosee)")

	resp, err := s.client.Do(req)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("抓取失败：%v", err)}},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("HTTP 错误：%d", resp.StatusCode)}},
		}, nil
	}

	// 解析 HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("解析 HTML 失败：%v", err)}},
		}, nil
	}

	if p.FullHTML {
		var sb strings.Builder
		if err := html.Render(&sb, doc); err != nil {
			return &ToolResult{
				IsError: true,
				Content: []Content{{Type: "text", Text: fmt.Sprintf("渲染 HTML 失败：%v", err)}},
			}, nil
		}
		return &ToolResult{
			Content: []Content{{Type: "text", Text: sb.String()}},
			Data:    map[string]interface{}{"url": p.URL, "status": resp.StatusCode},
		}, nil
	}

	// 提取文本
	text := extractText(doc, p.Selector)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: text}},
		Data:    map[string]interface{}{"url": p.URL, "status": resp.StatusCode, "length": len(text)},
	}, nil
}

// extractText 从 HTML 提取文本。
func extractText(n *html.Node, selector string) string {
	var sb strings.Builder
	var extract func(*html.Node)

	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteString("\n")
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(n)
	return sb.String()
}

// fetchSummary 抓取并生成摘要。
func (s *FetchServer) fetchSummary(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	// 验证 URL
	if _, err := url.Parse(p.URL); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效 URL: %v", err)}},
		}, nil
	}

	// 发送请求
	req, err := http.NewRequest("GET", p.URL, nil)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("请求失败：%v", err)}},
		}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Goose/1.0; +https://github.com/camark/Gotosee)")

	resp, err := s.client.Do(req)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("抓取失败：%v", err)}},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("HTTP 错误：%d", resp.StatusCode)}},
		}, nil
	}

	// 解析 HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("解析 HTML 失败：%v", err)}},
		}, nil
	}

	// 提取标题
	title := extractTitle(doc)

	// 提取正文（简化：提取所有段落）
	paragraphs := extractParagraphs(doc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(fmt.Sprintf("URL: %s\n\n", p.URL))
	sb.WriteString("## 内容摘要\n\n")
	for i, p := range paragraphs {
		if i >= 5 {
			break // 只返回前 5 段
		}
		sb.WriteString(fmt.Sprintf("%s\n\n", p))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"url": p.URL, "title": title, "paragraphs": len(paragraphs)},
	}, nil
}

// extractTitle 提取网页标题。
func extractTitle(n *html.Node) string {
	var title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil {
				title = n.FirstChild.Data
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return title
}

// extractParagraphs 提取段落。
func extractParagraphs(n *html.Node) []string {
	var paragraphs []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			var sb strings.Builder
			var extractText func(*html.Node)
			extractText = func(node *html.Node) {
				if node.Type == html.TextNode {
					sb.WriteString(node.Data)
				}
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					extractText(c)
				}
			}
			extractText(n)
			text := strings.TrimSpace(sb.String())
			if text != "" {
				paragraphs = append(paragraphs, text)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return paragraphs
}

// fetchLinks 提取链接。
func (s *FetchServer) fetchLinks(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL      string `json:"url"`
		Internal bool   `json:"internal"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	// 验证 URL
	baseURL, err := url.Parse(p.URL)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效 URL: %v", err)}},
		}, nil
	}

	// 发送请求
	req, err := http.NewRequest("GET", p.URL, nil)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("请求失败：%v", err)}},
		}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Goose/1.0; +https://github.com/camark/Gotosee)")

	resp, err := s.client.Do(req)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("抓取失败：%v", err)}},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("HTTP 错误：%d", resp.StatusCode)}},
		}, nil
	}

	// 解析 HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("解析 HTML 失败：%v", err)}},
		}, nil
	}

	// 提取链接
	var links []map[string]string
	seen := make(map[string]bool)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href := attr.Val
					// 转换为绝对 URL
					absURL, err := baseURL.Parse(href)
					if err != nil {
						continue
					}

					// 检查是否只返回内部链接
					if p.Internal && absURL.Host != baseURL.Host {
						continue
					}

					urlStr := absURL.String()
					if !seen[urlStr] {
						seen[urlStr] = true

						// 获取链接文本
						var textSB strings.Builder
						var extractText func(*html.Node)
						extractText = func(node *html.Node) {
							if node.Type == html.TextNode {
								textSB.WriteString(node.Data)
							}
							for c := node.FirstChild; c != nil; c = c.NextSibling {
								extractText(c)
							}
						}
						extractText(n)
						text := strings.TrimSpace(textSB.String())

						links = append(links, map[string]string{
							"url":  urlStr,
							"text": text,
						})
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到的链接 (%d 个):\n\n", len(links)))
	for _, link := range links {
		sb.WriteString(fmt.Sprintf("- [%s](%s)\n", link["text"], link["url"]))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"count": len(links), "links": links},
	}, nil
}

// fetchMetadata 提取元数据。
func (s *FetchServer) fetchMetadata(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	// 验证 URL
	if _, err := url.Parse(p.URL); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效 URL: %v", err)}},
		}, nil
	}

	// 发送请求
	req, err := http.NewRequest("GET", p.URL, nil)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("请求失败：%v", err)}},
		}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Goose/1.0; +https://github.com/camark/Gotosee)")

	resp, err := s.client.Do(req)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("抓取失败：%v", err)}},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("HTTP 错误：%d", resp.StatusCode)}},
		}, nil
	}

	// 解析 HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("解析 HTML 失败：%v", err)}},
		}, nil
	}

	// 提取元数据
	metadata := make(map[string]string)

	// 标题
	metadata["title"] = extractTitle(doc)

	// Meta 标签
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, content, property string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "name":
					name = attr.Val
				case "content":
					content = attr.Val
				case "property":
					property = attr.Val
				}
			}
			if content != "" {
				if name != "" {
					metadata[name] = content
				} else if property != "" {
					metadata[property] = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("网页元数据 (%s):\n\n", p.URL))
	for key, value := range metadata {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"metadata": metadata},
	}, nil
}

// ListResources 列出资源。
func (s *FetchServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *FetchServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
