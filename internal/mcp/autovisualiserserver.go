// Package mcp 提供 AutoVisualiser MCP 服务器（图表和可视化生成）。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// ============================================================================
// AutoVisualiser MCP 服务器
// ============================================================================

// AutoVisualiserServer 图表和可视化生成 MCP 服务器。
type AutoVisualiserServer struct {
	*BaseServer
}

// NewAutoVisualiserServer 创建新的 AutoVisualiser 服务器。
func NewAutoVisualiserServer() *AutoVisualiserServer {
	server := &AutoVisualiserServer{}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "auto-visualiser-mcp",
			Version:     "1.0.0",
			Instructions: "AutoVisualiser：生成 ASCII 图表、数据可视化、统计图表",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *AutoVisualiserServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "bar_chart",
			Description: "生成 ASCII 条形图",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "图表标题"},
					"data": {"type": "object", "description": "键值对数据"},
					"width": {"type": "number", "description": "图表宽度", "default": 50}
				},
				"required": ["data"]
			}`),
		},
		{
			Name:        "line_chart",
			Description: "生成 ASCII 折线图",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "图表标题"},
					"values": {"type": "array", "description": "数值数组"},
					"labels": {"type": "array", "description": "X 轴标签"},
					"width": {"type": "number", "description": "图表宽度", "default": 60},
					"height": {"type": "number", "description": "图表高度", "default": 15}
				},
				"required": ["values"]
			}`),
		},
		{
			Name:        "pie_chart",
			Description: "生成 ASCII 饼图",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "图表标题"},
					"data": {"type": "object", "description": "键值对数据"},
					"radius": {"type": "number", "description": "饼图半径", "default": 10}
				},
				"required": ["data"]
			}`),
		},
		{
			Name:        "scatter_plot",
			Description: "生成 ASCII 散点图",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "图表标题"},
					"points": {"type": "array", "description": "点坐标数组 [{x, y}]"},
					"width": {"type": "number", "description": "图表宽度", "default": 50},
					"height": {"type": "number", "description": "图表高度", "default": 20}
				},
				"required": ["points"]
			}`),
		},
		{
			Name:        "histogram",
			Description: "生成 ASCII 直方图",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "图表标题"},
					"data": {"type": "array", "description": "数值数组"},
					"bins": {"type": "number", "description": "分组数量", "default": 10},
					"width": {"type": "number", "description": "图表宽度", "default": 50}
				},
				"required": ["data"]
			}`),
		},
		{
			Name:        "comparison_table",
			Description: "生成对比表格",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "表格标题"},
					"headers": {"type": "array", "description": "表头"},
					"rows": {"type": "array", "description": "数据行"}
				},
				"required": ["headers", "rows"]
			}`),
		},
		{
			Name:        "sparkline",
			Description: "生成迷你图 (sparkline)",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"values": {"type": "array", "description": "数值数组"},
					"title": {"type": "string", "description": "标题"}
				},
				"required": ["values"]
			}`),
		},
		{
			Name:        "gauge",
			Description: "生成仪表盘",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"value": {"type": "number", "description": "当前值"},
					"min": {"type": "number", "description": "最小值", "default": 0},
					"max": {"type": "number", "description": "最大值", "default": 100},
					"title": {"type": "string", "description": "标题"},
					"width": {"type": "number", "description": "宽度", "default": 40}
				},
				"required": ["value"]
			}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *AutoVisualiserServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "bar_chart":
		return s.barChart(params)
	case "line_chart":
		return s.lineChart(params)
	case "pie_chart":
		return s.pieChart(params)
	case "scatter_plot":
		return s.scatterPlot(params)
	case "histogram":
		return s.histogram(params)
	case "comparison_table":
		return s.comparisonTable(params)
	case "sparkline":
		return s.sparkline(params)
	case "gauge":
		return s.gauge(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// barChart 生成条形图。
func (s *AutoVisualiserServer) barChart(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Title string                 `json:"title"`
		Data  map[string]float64     `json:"data"`
		Width int                    `json:"width"`
	}
	json.Unmarshal(params, &p)

	if p.Width == 0 {
		p.Width = 50
	}

	var sb strings.Builder

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("╔════════════════════════════════════════╗\n"))
		sb.WriteString(fmt.Sprintf("║  %-36s  ║\n", p.Title))
		sb.WriteString(fmt.Sprintf("╚════════════════════════════════════════╝\n\n"))
	}

	// 找到最大值
	maxVal := 0.0
	for _, v := range p.Data {
		if v > maxVal {
			maxVal = v
		}
	}

	// 找到最长标签长度
	maxLabelLen := 0
	for k := range p.Data {
		if len(k) > maxLabelLen {
			maxLabelLen = len(k)
		}
	}

	// 生成条形
	for label, value := range p.Data {
		barLen := int((value / maxVal) * float64(p.Width))
		if barLen < 0 {
			barLen = 0
		}

		bar := strings.Repeat("█", barLen)
		sb.WriteString(fmt.Sprintf("%-*s |%s %v\n", maxLabelLen, label, bar, value))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":  "bar_chart",
			"title": p.Title,
			"count": len(p.Data),
		},
	}, nil
}

// lineChart 生成折线图。
func (s *AutoVisualiserServer) lineChart(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Title  string    `json:"title"`
		Values []float64 `json:"values"`
		Labels []string  `json:"labels"`
		Width  int       `json:"width"`
		Height int       `json:"height"`
	}
	json.Unmarshal(params, &p)

	if p.Width == 0 {
		p.Width = 60
	}
	if p.Height == 0 {
		p.Height = 15
	}

	var sb strings.Builder

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("┌─────────────────────────────────────────┐\n"))
		sb.WriteString(fmt.Sprintf("│  %-36s  │\n", p.Title))
		sb.WriteString(fmt.Sprintf("└─────────────────────────────────────────┘\n\n"))
	}

	if len(p.Values) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "错误：没有数据"}},
		}, nil
	}

	// 找到最大最小值
	maxVal := p.Values[0]
	minVal := p.Values[0]
	for _, v := range p.Values {
		if v > maxVal {
			maxVal = v
		}
		if v < minVal {
			minVal = v
		}
	}

	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}

	// 创建画布
	canvas := make([][]rune, p.Height)
	for i := range canvas {
		canvas[i] = make([]rune, p.Width)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// 绘制 Y 轴
	for i := 0; i < p.Height; i++ {
		canvas[i][0] = '│'
	}
	canvas[0][0] = '┌'
	canvas[p.Height-1][0] = '└'

	// 绘制 X 轴
	for j := 1; j < p.Width; j++ {
		canvas[p.Height-1][j] = '─'
	}

	// 绘制点
	var prevX, prevY int = -1, -1
	for i, v := range p.Values {
		x := 1 + int((float64(i) / float64(len(p.Values)-1)) * float64(p.Width-2))
		y := p.Height - 2 - int(((v - minVal) / rangeVal) * float64(p.Height-3))

		if x >= p.Width {
			x = p.Width - 2
		}
		if y < 1 {
			y = 1
		}
		if y >= p.Height-1 {
			y = p.Height - 2
		}

		if x < 1 {
			x = 1
		}

		canvas[y][x] = '●'

		// 连线
		if prevX >= 0 {
			s.drawLine(canvas, prevX, prevY, x, y)
		}
		prevX, prevY = x, y
	}

	// 输出画布
	for i := 0; i < p.Height; i++ {
		sb.WriteString(string(canvas[i]))
		sb.WriteString("\n")
	}

	// X 轴标签
	if len(p.Labels) > 0 {
		sb.WriteString("\n")
		for i, label := range p.Labels {
			if i == 0 || i == len(p.Labels)-1 || i == len(p.Labels)/2 {
				sb.WriteString(fmt.Sprintf("%-*s", p.Width/len(p.Labels), label))
			}
		}
		sb.WriteString("\n")
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":  "line_chart",
			"title": p.Title,
			"count": len(p.Values),
		},
	}, nil
}

// drawLine 在画布上绘制两点之间的线。
func (s *AutoVisualiserServer) drawLine(canvas [][]rune, x0, y0, x1, y1 int) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	for {
		if y0 >= 0 && y0 < len(canvas) && x0 >= 0 && x0 < len(canvas[0]) {
			if canvas[y0][x0] == ' ' {
				canvas[y0][x0] = '·'
			}
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// pieChart 生成饼图。
func (s *AutoVisualiserServer) pieChart(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Title  string             `json:"title"`
		Data   map[string]float64 `json:"data"`
		Radius int                `json:"radius"`
	}
	json.Unmarshal(params, &p)

	if p.Radius == 0 {
		p.Radius = 10
	}

	var sb strings.Builder

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("╔════════════════════════════════════════╗\n"))
		sb.WriteString(fmt.Sprintf("║  %-36s  ║\n", p.Title))
		sb.WriteString(fmt.Sprintf("╚════════════════════════════════════════╝\n\n"))
	}

	// 计算总和
	total := 0.0
	for _, v := range p.Data {
		total += v
	}

	if total == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "错误：数据总和为 0"}},
		}, nil
	}

	// 计算角度
	chars := []rune{'░', '▒', '▓', '█', '▄', '▀', '▌', '▐'}
	colors := make(map[string]rune)
	i := 0
	for k := range p.Data {
		colors[k] = chars[i%len(chars)]
		i++
	}

	// 简单的饼图表示
	sb.WriteString("图例:\n")
	for label, value := range p.Data {
		percent := (value / total) * 100
		sb.WriteString(fmt.Sprintf("  %c %s: %.1f%%\n", colors[label], label, percent))
	}

	sb.WriteString("\n")
	sb.WriteString("饼图 (简化表示):\n\n")

	// 绘制圆形饼图
	centerX, centerY := p.Radius*2+1, p.Radius+1
	canvas := make([][]rune, centerY*2+1)
	for i := range canvas {
		canvas[i] = make([]rune, centerX*2+1)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// 按顺序填充扇形
	currentAngle := 0.0
	angleStep := math.Pi / 180 // 1 度
	for label, value := range p.Data {
		endAngle := currentAngle + (value/total)*2*math.Pi
		char := colors[label]

		for a := currentAngle; a < endAngle; a += angleStep {
			for r := 0; r <= p.Radius; r++ {
				x := centerX + int(float64(r)*math.Cos(a)*2) // *2 因为字符宽高比
				y := centerY + int(float64(r) * math.Sin(a))

				if y >= 0 && y < len(canvas) && x >= 0 && x < len(canvas[0]) {
					canvas[y][x] = char
				}
			}
		}
		currentAngle = endAngle
	}

	// 输出
	for _, row := range canvas {
		sb.WriteString(string(row))
		sb.WriteString("\n")
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":  "pie_chart",
			"title": p.Title,
			"count": len(p.Data),
		},
	}, nil
}

// scatterPlot 生成散点图。
func (s *AutoVisualiserServer) scatterPlot(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Title  string `json:"title"`
		Points []struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"points"`
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	json.Unmarshal(params, &p)

	if p.Width == 0 {
		p.Width = 50
	}
	if p.Height == 0 {
		p.Height = 20
	}

	var sb strings.Builder

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("┌─────────────────────────────────────────┐\n"))
		sb.WriteString(fmt.Sprintf("│  %-36s  │\n", p.Title))
		sb.WriteString(fmt.Sprintf("└─────────────────────────────────────────┘\n\n"))
	}

	if len(p.Points) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "错误：没有数据点"}},
		}, nil
	}

	// 找到最大最小值
	minX, maxX := p.Points[0].X, p.Points[0].X
	minY, maxY := p.Points[0].Y, p.Points[0].Y
	for _, pt := range p.Points {
		if pt.X < minX {
			minX = pt.X
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
	}

	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	// 创建画布
	canvas := make([][]rune, p.Height)
	for i := range canvas {
		canvas[i] = make([]rune, p.Width)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}

	// 绘制轴
	for j := 0; j < p.Width; j++ {
		canvas[p.Height-1][j] = '─'
	}
	for i := 0; i < p.Height; i++ {
		canvas[i][0] = '│'
	}
	canvas[p.Height-1][0] = '└'

	// 绘制点
	for _, pt := range p.Points {
		x := 1 + int(((pt.X - minX) / rangeX) * float64(p.Width-2))
		y := p.Height - 2 - int(((pt.Y - minY) / rangeY) * float64(p.Height-3))

		if x >= 1 && x < p.Width-1 && y >= 1 && y < p.Height-1 {
			canvas[y][x] = '●'
		} else if x >= 0 && x < p.Width && y >= 0 && y < p.Height {
			canvas[y][x] = '●'
		}
	}

	// 输出画布
	for i := 0; i < p.Height; i++ {
		sb.WriteString(string(canvas[i]))
		sb.WriteString("\n")
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":  "scatter_plot",
			"title": p.Title,
			"count": len(p.Points),
		},
	}, nil
}

// histogram 生成直方图。
func (s *AutoVisualiserServer) histogram(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Title string    `json:"title"`
		Data  []float64 `json:"data"`
		Bins  int       `json:"bins"`
		Width int       `json:"width"`
	}
	json.Unmarshal(params, &p)

	if p.Bins == 0 {
		p.Bins = 10
	}
	if p.Width == 0 {
		p.Width = 50
	}

	var sb strings.Builder

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("╔════════════════════════════════════════╗\n"))
		sb.WriteString(fmt.Sprintf("║  %-36s  ║\n", p.Title))
		sb.WriteString(fmt.Sprintf("╚════════════════════════════════════════╝\n\n"))
	}

	if len(p.Data) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "错误：没有数据"}},
		}, nil
	}

	// 找到最大最小值
	minVal, maxVal := p.Data[0], p.Data[0]
	for _, v := range p.Data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// 创建 bins
	binWidth := (maxVal - minVal) / float64(p.Bins)
	if binWidth == 0 {
		binWidth = 1
	}
	bins := make([]int, p.Bins)
	for _, v := range p.Data {
		binIdx := int((v - minVal) / binWidth)
		if binIdx >= p.Bins {
			binIdx = p.Bins - 1
		}
		bins[binIdx]++
	}

	// 找到最大 bin 计数
	maxCount := 0
	for _, count := range bins {
		if count > maxCount {
			maxCount = count
		}
	}

	// 绘制直方图
	sb.WriteString("频数分布:\n\n")
	for i, count := range bins {
		binStart := minVal + float64(i)*binWidth
		binEnd := minVal + float64(i+1)*binWidth

		barLen := 0
		if maxCount > 0 {
			barLen = int((float64(count) / float64(maxCount)) * float64(p.Width))
		}

		bar := strings.Repeat("█", barLen)
		sb.WriteString(fmt.Sprintf("%8.2f-%8.2f |%s %d\n", binStart, binEnd, bar, count))
	}

	sb.WriteString(fmt.Sprintf("\n总数据点数：%d\n", len(p.Data)))

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":  "histogram",
			"title": p.Title,
			"count": len(p.Data),
			"bins":  p.Bins,
		},
	}, nil
}

// comparisonTable 生成对比表格。
func (s *AutoVisualiserServer) comparisonTable(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Title   string   `json:"title"`
		Headers []string `json:"headers"`
		Rows    [][]string `json:"rows"`
	}
	json.Unmarshal(params, &p)

	var sb strings.Builder

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("╔════════════════════════════════════════╗\n"))
		sb.WriteString(fmt.Sprintf("║  %-36s  ║\n", p.Title))
		sb.WriteString(fmt.Sprintf("╚════════════════════════════════════════╝\n\n"))
	}

	if len(p.Headers) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "错误：没有表头"}},
		}, nil
	}

	// 计算列宽
	colWidths := make([]int, len(p.Headers))
	for i, h := range p.Headers {
		colWidths[i] = len(h)
	}
	for _, row := range p.Rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// 生成表格
	sb.WriteString("┌")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("┐\n")

	// 表头
	sb.WriteString("│")
	for i, h := range p.Headers {
		sb.WriteString(fmt.Sprintf(" %- *s │", colWidths[i], h))
	}
	sb.WriteString("\n")

	// 分隔线
	sb.WriteString("├")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			sb.WriteString("┼")
		}
	}
	sb.WriteString("┤\n")

	// 数据行
	for _, row := range p.Rows {
		sb.WriteString("│")
		for i := range p.Headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			sb.WriteString(fmt.Sprintf(" %- *s │", colWidths[i], cell))
		}
		sb.WriteString("\n")
	}

	// 底部
	sb.WriteString("└")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("┘\n")

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":  "comparison_table",
			"title": p.Title,
			"rows":  len(p.Rows),
			"cols":  len(p.Headers),
		},
	}, nil
}

// sparkline 生成迷你图。
func (s *AutoVisualiserServer) sparkline(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Values []float64 `json:"values"`
		Title  string    `json:"title"`
	}
	json.Unmarshal(params, &p)

	var sb strings.Builder

	if len(p.Values) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "错误：没有数据"}},
		}, nil
	}

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("%s\n", p.Title))
	}

	// 找到最大最小值
	maxVal := p.Values[0]
	minVal := p.Values[0]
	for _, v := range p.Values {
		if v > maxVal {
			maxVal = v
		}
		if v < minVal {
			minVal = v
		}
	}

	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}

	// 生成 sparkline
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	sb.WriteString("  ")
	for _, v := range p.Values {
		idx := int(((v - minVal) / rangeVal) * float64(len(chars)-1))
		sb.WriteRune(chars[idx])
	}
	sb.WriteString("\n")

	// 统计信息
	sb.WriteString(fmt.Sprintf("  最小值：%.2f  最大值：%.2f  数据点：%d\n", minVal, maxVal, len(p.Values)))

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":  "sparkline",
			"title": p.Title,
			"count": len(p.Values),
		},
	}, nil
}

// gauge 生成仪表盘。
func (s *AutoVisualiserServer) gauge(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Value float64 `json:"value"`
		Min   float64 `json:"min"`
		Max   float64 `json:"max"`
		Title string  `json:"title"`
		Width int     `json:"width"`
	}
	json.Unmarshal(params, &p)

	if p.Width == 0 {
		p.Width = 40
	}
	if p.Max <= p.Min {
		p.Max = 100
	}

	var sb strings.Builder

	// 标题
	if p.Title != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", p.Title))
	}

	// 计算百分比
	percent := (p.Value - p.Min) / (p.Max - p.Min)
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	// 绘制仪表盘
	barWidth := p.Width - 4
	filledWidth := int(percent * float64(barWidth))

	sb.WriteString("  ┌")
	sb.WriteString(strings.Repeat("─", barWidth))
	sb.WriteString("┐\n")

	sb.WriteString("  │")
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			sb.WriteString("█")
		} else {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("│\n")

	sb.WriteString("  └")
	sb.WriteString(strings.Repeat("─", barWidth))
	sb.WriteString("┘\n")

	// 标签
	sb.WriteString(fmt.Sprintf("  %*.*f", filledWidth+1, filledWidth, p.Min+percent*(p.Max-p.Min)))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  范围：%.2f - %.2f (%.1f%%)\n", p.Min, p.Max, percent*100))

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"type":      "gauge",
			"title":     p.Title,
			"value":     p.Value,
			"percent":   percent * 100,
		},
	}, nil
}

// abs 返回绝对值。
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ListResources 列出资源。
func (s *AutoVisualiserServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *AutoVisualiserServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}
