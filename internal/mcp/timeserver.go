// Package mcp 提供 Time MCP 服务器（时间管理）。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// Time MCP 服务器
// ============================================================================

// TimeServer 时间管理 MCP 服务器。
type TimeServer struct {
	*BaseServer
	location *time.Location
}

// NewTimeServer 创建新的时间服务器。
func NewTimeServer() *TimeServer {
	server := &TimeServer{
		location: time.Local,
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "time-mcp",
			Version:     "1.0.0",
			Instructions: "提供时间相关功能：当前时间、时区转换、时间计算等",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// SetLocation 设置时区。
func (s *TimeServer) SetLocation(tz string) error {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	s.location = loc
	return nil
}

// ListTools 列出所有工具。
func (s *TimeServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "get_current_time",
			Description: "获取当前时间",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"timezone": {"type": "string", "description": "时区名称，如 'Asia/Shanghai' 或 'UTC'"}
				}
			}`),
		},
		{
			Name:        "convert_timezone",
			Description: "时区转换",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"time": {"type": "string", "description": "时间 (RFC3339 格式)"},
					"from_tz": {"type": "string", "description": "源时区"},
					"to_tz": {"type": "string", "description": "目标时区"}
				},
				"required": ["time", "from_tz", "to_tz"]
			}`),
		},
		{
			Name:        "format_time",
			Description: "格式化时间",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"time": {"type": "string", "description": "时间 (RFC3339 格式)"},
					"format": {"type": "string", "description": "格式化模板"}
				},
				"required": ["time"]
			}`),
		},
		{
			Name:        "parse_time",
			Description: "解析时间字符串",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"time_str": {"type": "string", "description": "时间字符串"},
					"format": {"type": "string", "description": "解析格式"}
				},
				"required": ["time_str"]
			}`),
		},
		{
			Name:        "calculate_duration",
			Description: "计算两个时间之间的差值",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"start": {"type": "string", "description": "开始时间 (RFC3339 格式)"},
					"end": {"type": "string", "description": "结束时间 (RFC3339 格式)"}
				},
				"required": ["start", "end"]
			}`),
		},
		{
			Name:        "add_duration",
			Description: "在时间上增加一段时长",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"time": {"type": "string", "description": "时间 (RFC3339 格式)"},
					"hours": {"type": "integer", "description": "小时数"},
					"minutes": {"type": "integer", "description": "分钟数"},
					"days": {"type": "integer", "description": "天数"}
				},
				"required": ["time"]
			}`),
		},
		{
			Name:        "get_week_info",
			Description: "获取指定日期是一周的第几天",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"time": {"type": "string", "description": "时间 (RFC3339 格式)"}
				}
			}`),
		},
		{
			Name:        "is_weekend",
			Description: "判断是否是周末",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"time": {"type": "string", "description": "时间 (RFC3339 格式)"}
				}
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *TimeServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "get_current_time":
		return s.getCurrentTime(params)
	case "convert_timezone":
		return s.convertTimezone(params)
	case "format_time":
		return s.formatTime(params)
	case "parse_time":
		return s.parseTime(params)
	case "calculate_duration":
		return s.calculateDuration(params)
	case "add_duration":
		return s.addDuration(params)
	case "get_week_info":
		return s.getWeekInfo(params)
	case "is_weekend":
		return s.isWeekend(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// getCurrentTime 获取当前时间。
func (s *TimeServer) getCurrentTime(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Timezone string `json:"timezone"`
	}
	json.Unmarshal(params, &p)

	loc := s.location
	if p.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(p.Timezone)
		if err != nil {
			return &ToolResult{
				IsError: true,
				Content: []Content{{Type: "text", Text: fmt.Sprintf("无效时区：%v", err)}},
			}, nil
		}
	}

	now := time.Now().In(loc)

	var sb string
	sb += fmt.Sprintf("当前时间：%s\n", now.Format("2006-01-02 15:04:05"))
	sb += fmt.Sprintf("时区：%s\n", loc.String())
	sb += fmt.Sprintf("星期：%s\n", now.Weekday())
	sb += fmt.Sprintf("一年中的第 %d 天\n", now.YearDay())

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb}},
		Data: map[string]interface{}{
			"time":     now.Format(time.RFC3339),
			"timezone": loc.String(),
			"weekday":  now.Weekday().String(),
		},
	}, nil
}

// convertTimezone 转换时区。
func (s *TimeServer) convertTimezone(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Time   string `json:"time"`
		FromTZ string `json:"from_tz"`
		ToTZ   string `json:"to_tz"`
	}
	json.Unmarshal(params, &p)

	t, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效时间格式：%v", err)}},
		}, nil
	}

	fromLoc, err := time.LoadLocation(p.FromTZ)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效源时区：%v", err)}},
		}, nil
	}

	toLoc, err := time.LoadLocation(p.ToTZ)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效目标时区：%v", err)}},
		}, nil
	}

	converted := t.In(fromLoc).In(toLoc)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("%s (%s) 转换为 %s 时区后：%s", p.Time, p.FromTZ, p.ToTZ, converted.Format("2006-01-02 15:04:05"))}},
		Data: map[string]interface{}{
			"original":   p.Time,
			"converted":  converted.Format(time.RFC3339),
			"from_tz":    p.FromTZ,
			"to_tz":      p.ToTZ,
		},
	}, nil
}

// formatTime 格式化时间。
func (s *TimeServer) formatTime(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Time   string `json:"time"`
		Format string `json:"format"`
	}
	json.Unmarshal(params, &p)

	t, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效时间格式：%v", err)}},
		}, nil
	}

	format := p.Format
	if format == "" {
		format = "2006-01-02 15:04:05"
	}

	formatted := t.Format(format)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("格式化结果：%s", formatted)}},
		Data: map[string]interface{}{
			"formatted": formatted,
		},
	}, nil
}

// parseTime 解析时间。
func (s *TimeServer) parseTime(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		TimeStr string `json:"time_str"`
		Format  string `json:"format"`
	}
	json.Unmarshal(params, &p)

	format := p.Format
	if format == "" {
		format = "2006-01-02 15:04:05"
	}

	t, err := time.Parse(format, p.TimeStr)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("解析失败：%v", err)}},
		}, nil
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("解析结果：%s", t.Format(time.RFC3339))}},
		Data: map[string]interface{}{
			"parsed": t.Format(time.RFC3339),
		},
	}, nil
}

// calculateDuration 计算时长。
func (s *TimeServer) calculateDuration(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	json.Unmarshal(params, &p)

	start, err := time.Parse(time.RFC3339, p.Start)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效开始时间：%v", err)}},
		}, nil
	}

	end, err := time.Parse(time.RFC3339, p.End)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效结束时间：%v", err)}},
		}, nil
	}

	duration := end.Sub(start)

	var sb string
	sb += fmt.Sprintf("时间差：%s\n", duration.String())
	sb += fmt.Sprintf("天数：%.2f\n", duration.Hours()/24)
	sb += fmt.Sprintf("小时数：%.2f\n", duration.Hours())
	sb += fmt.Sprintf("分钟数：%.2f\n", duration.Minutes())
	sb += fmt.Sprintf("秒数：%.2f\n", duration.Seconds())

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb}},
		Data: map[string]interface{}{
			"duration_seconds": duration.Seconds(),
			"duration_minutes": duration.Minutes(),
			"duration_hours":   duration.Hours(),
			"duration_days":    duration.Hours() / 24,
		},
	}, nil
}

// addDuration 增加时长。
func (s *TimeServer) addDuration(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Time    string `json:"time"`
		Hours   int    `json:"hours"`
		Minutes int    `json:"minutes"`
		Days    int    `json:"days"`
	}
	json.Unmarshal(params, &p)

	t, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效时间：%v", err)}},
		}, nil
	}

	result := t.Add(time.Duration(p.Days) * 24 * time.Hour)
	result = result.Add(time.Duration(p.Hours) * time.Hour)
	result = result.Add(time.Duration(p.Minutes) * time.Minute)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("增加时长后：%s", result.Format("2006-01-02 15:04:05"))}},
		Data: map[string]interface{}{
			"result": result.Format(time.RFC3339),
		},
	}, nil
}

// getWeekInfo 获取周信息。
func (s *TimeServer) getWeekInfo(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Time string `json:"time"`
	}
	json.Unmarshal(params, &p)

	t, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效时间：%v", err)}},
		}, nil
	}

	weekday := t.Weekday()
	weekdayName := weekday.String()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	var sb string
	sb += fmt.Sprintf("日期：%s\n", t.Format("2006-01-02"))
	sb += fmt.Sprintf("星期：%s\n", weekdayName)
	sb += fmt.Sprintf("一周的第 %d 天\n", int(weekday)+1)
	if isWeekend {
		sb += "这是周末\n"
	} else {
		sb += "这是工作日\n"
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb}},
		Data: map[string]interface{}{
			"weekday":   int(weekday),
			"weekday_name": weekdayName,
			"is_weekend":  isWeekend,
		},
	}, nil
}

// isWeekend 判断是否是周末。
func (s *TimeServer) isWeekend(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Time string `json:"time"`
	}
	json.Unmarshal(params, &p)

	t, err := time.Parse(time.RFC3339, p.Time)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无效时间：%v", err)}},
		}, nil
	}

	weekday := t.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	result := "否"
	if isWeekend {
		result = "是"
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("%s 是周末吗？%s", t.Format("2006-01-02"), result)}},
		Data: map[string]interface{}{
			"is_weekend": isWeekend,
		},
	}, nil
}

// ListResources 列出资源。
func (s *TimeServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *TimeServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
