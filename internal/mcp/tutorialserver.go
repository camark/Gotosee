// Package mcp 提供 Tutorial MCP 服务器。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ============================================================================
// Tutorial MCP 服务器
// ============================================================================

// TutorialServer 教程 MCP 服务器。
type TutorialServer struct {
	*BaseServer
}

// NewTutorialServer 创建新的教程服务器。
func NewTutorialServer() *TutorialServer {
	server := &TutorialServer{}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "tutorial-mcp",
			Version:     "1.0.0",
			Instructions: "提供 Goose 使用教程和指南",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *TutorialServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "tutorial_get",
			Description: "获取指定主题的教程内容",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"topic": {"type": "string", "description": "教程主题"},
					"level": {"type": "string", "description": "难度级别", "enum": ["beginner", "intermediate", "advanced"]}
				},
				"required": ["topic"]
			}`),
		},
		{
			Name:        "tutorial_list",
			Description: "列出所有可用教程主题",
			InputSchema: json.RawMessage(`{}`),
		},
		{
			Name:        "tutorial_search",
			Description: "搜索教程内容",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "搜索关键词"}
				},
				"required": ["query"]
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *TutorialServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "tutorial_get":
		return s.getTutorial(params)
	case "tutorial_list":
		return s.listTutorials(params)
	case "tutorial_search":
		return s.searchTutorial(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// tutorialContent 教程内容结构。
type tutorialContent struct {
	Title       string   `json:"title"`
	Level       string   `json:"level"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Examples    []string `json:"examples,omitempty"`
	Related     []string `json:"related,omitempty"`
}

// tutorials 教程数据库。
var tutorials = map[string]tutorialContent{
	"getting_started": {
		Title:       "Goose 入门指南",
		Level:       "beginner",
		Description: "学习如何使用 Goose AI 代理框架的基础知识",
		Steps: []string{
			"1. 安装 Goose：go install github.com/camark/Gotosee/cmd/goose@latest",
			"2. 运行配置向导：goose configure",
			"3. 选择 AI 提供商（OpenAI、Anthropic、Ollama 等）",
			"4. 输入 API Key（Ollama 不需要）",
			"5. 开始对话：goose chat",
		},
		Examples: []string{
			"goose chat                          # 开始交互式对话",
			"goose recipe run translator.json    # 运行翻译配方",
			"goose schedule add                  # 添加定时任务",
		},
		Related: []string{"configuration", "chat", "recipes"},
	},
	"configuration": {
		Title:       "配置指南",
		Level:       "beginner",
		Description: "学习如何配置 Goose 以使用不同的 AI 提供商",
		Steps: []string{
			"1. 运行配置向导：goose configure",
			"2. 选择提供商类型",
			"3. 输入 API Key 和 Base URL",
			"4. 选择默认模型",
			"5. 测试配置",
		},
		Examples: []string{
			"goose configure                    # 交互式配置",
			"goose doctor                       # 检查配置状态",
			"goose info                         # 查看配置信息",
		},
		Related: []string{"getting_started", "providers"},
	},
	"recipes": {
		Title:       "配方使用指南",
		Level:       "intermediate",
		Description: "学习如何创建和使用自定义 AI 配方",
		Steps: []string{
			"1. 创建 recipes 目录",
			"2. 编写 JSON 格式的配方文件",
			"3. 定义设置项和指令",
			"4. 运行配方：goose recipe run <file>",
			"5. 传递设置参数：-s key=value",
		},
		Examples: []string{
			"goose recipe list                              # 列出配方",
			"goose recipe explain translator.json           # 查看配方说明",
			"goose recipe run translator.json -s tone=casual # 运行配方",
		},
		Related: []string{"getting_started", "advanced_usage"},
	},
	"scheduling": {
		Title:       "定时任务指南",
		Level:       "intermediate",
		Description: "学习如何创建和管理定时 AI 任务",
		Steps: []string{
			"1. 运行 goose schedule add",
			"2. 输入任务名称和描述",
			"3. 设置 Cron 表达式",
			"4. 选择执行命令或配方",
			"5. 查看任务列表：goose schedule",
		},
		Examples: []string{
			"goose schedule                        # 列出任务",
			"goose schedule run daily-report       # 立即运行任务",
			"goose schedule remove daily-report    # 删除任务",
		},
		Related: []string{"recipes", "automation"},
	},
	"providers": {
		Title:       "AI 提供商配置",
		Level:       "intermediate",
		Description: "了解如何配置不同的 AI 提供商",
		Steps: []string{
			"1. OpenAI: 需要 API Key",
			"2. Anthropic: 需要 API Key",
			"3. Ollama: 本地运行，无需 Key",
			"4. Google: 需要 API Key",
			"5. Azure: 需要部署名称和 Key",
		},
		Examples: []string{
			"goose chat -p openai -m gpt-4o       # 使用 OpenAI",
			"goose chat -p ollama -m llama2       # 使用 Ollama",
			"goose chat -p anthropic -m claude-3  # 使用 Anthropic",
		},
		Related: []string{"configuration", "models"},
	},
}

// getTutorial 获取教程。
func (s *TutorialServer) getTutorial(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Topic string `json:"topic"`
		Level string `json:"level"`
		_     struct{} `json:"$schema"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	tutorial, ok := tutorials[p.Topic]
	if !ok {
		// 模糊匹配
		for key, t := range tutorials {
			if strings.Contains(key, strings.ToLower(p.Topic)) {
				tutorial = t
				break
			}
		}
		if tutorial.Title == "" {
			return &ToolResult{
				IsError: true,
				Content: []Content{{Type: "text", Text: fmt.Sprintf("未找到主题 '%s' 的教程", p.Topic)}},
			}, nil
		}
	}

	if p.Level != "" && tutorial.Level != p.Level {
		// 级别不匹配，但仍然返回
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", tutorial.Title))
	sb.WriteString(fmt.Sprintf("**难度**: %s\n\n", tutorial.Level))
	sb.WriteString(fmt.Sprintf("%s\n\n", tutorial.Description))
	sb.WriteString("## 步骤\n\n")
	for _, step := range tutorial.Steps {
		sb.WriteString(fmt.Sprintf("%s\n", step))
	}
	sb.WriteString("\n## 示例\n\n")
	for _, example := range tutorial.Examples {
		sb.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", example))
	}
	if len(tutorial.Related) > 0 {
		sb.WriteString("\n## 相关内容\n\n")
		sb.WriteString(strings.Join(tutorial.Related, ", "))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"topic": p.Topic, "level": tutorial.Level},
	}, nil
}

// listTutorials 列出教程。
func (s *TutorialServer) listTutorials(params json.RawMessage) (*ToolResult, error) {
	var sb strings.Builder
	sb.WriteString("可用教程列表:\n\n")

	// 按级别分组
	levels := map[string][]string{
		"beginner":     {},
		"intermediate": {},
		"advanced":     {},
	}

	for key, t := range tutorials {
		levels[t.Level] = append(levels[t.Level], key)
	}

	for _, level := range []string{"beginner", "intermediate", "advanced"} {
		levelName := map[string]string{
			"beginner":     "入门",
			"intermediate": "中级",
			"advanced":     "高级",
		}[level]

		sb.WriteString(fmt.Sprintf("### %s (%d 个)\n", levelName, len(levels[level])))
		for _, key := range levels[level] {
			t := tutorials[key]
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", t.Title, t.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n使用 'goose mcp tutorial tutorial_get <topic>' 查看详细教程\n")

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"count": len(tutorials)},
	}, nil
}

// searchTutorial 搜索教程。
func (s *TutorialServer) searchTutorial(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal(params, &p); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("参数解析失败：%v", err)}},
		}, nil
	}

	query := strings.ToLower(p.Query)
	var results []string

	for key, t := range tutorials {
		// 搜索标题、描述、步骤
		if strings.Contains(strings.ToLower(t.Title), query) ||
			strings.Contains(strings.ToLower(t.Description), query) ||
			strings.Contains(strings.ToLower(t.Steps[0]), query) {
			results = append(results, key)
		}
	}

	if len(results) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未找到与 '%s' 相关的教程", p.Query)}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个相关教程:\n\n", len(results)))
	for _, key := range results {
		t := tutorials[key]
		sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", t.Title, key, t.Description))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data:    map[string]interface{}{"results": results, "count": len(results)},
	}, nil
}

// ListResources 列出资源。
func (s *TutorialServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *TutorialServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
