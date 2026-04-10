// Package mcp 提供 Database MCP 服务器（数据库操作）。
package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// Database MCP 服务器
// ============================================================================

// DatabaseServer 数据库操作 MCP 服务器。
type DatabaseServer struct {
	*BaseServer
	connections map[string]*sql.DB
}

// NewDatabaseServer 创建新的数据库服务器。
func NewDatabaseServer() *DatabaseServer {
	server := &DatabaseServer{
		connections: make(map[string]*sql.DB),
	}

	server.BaseServer = &BaseServer{
		info: ServerInfo{
			Name:        "database-mcp",
			Version:     "1.0.0",
			Instructions: "数据库操作：连接、查询、执行 SQL 语句（支持 SQLite）",
		},
		capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{ListChanged: true},
		},
	}

	return server
}

// ListTools 列出所有工具。
func (s *DatabaseServer) ListTools(ctx context.Context) ([]Tool, error) {
	return []Tool{
		{
			Name:        "connect",
			Description: "连接到数据库",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "连接名称"},
					"driver": {"type": "string", "description": "数据库驱动", "default": "sqlite3"},
					"dsn": {"type": "string", "description": "数据源连接字符串"}
				},
				"required": ["name", "dsn"]
			}`),
		},
		{
			Name:        "disconnect",
			Description: "断开数据库连接",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "连接名称"}
				},
				"required": ["name"]
			}`),
		},
		{
			Name:        "query",
			Description: "执行查询语句",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"connection": {"type": "string", "description": "连接名称"},
					"sql": {"type": "string", "description": "SQL 查询语句"}
				},
				"required": ["connection", "sql"]
			}`),
		},
		{
			Name:        "execute",
			Description: "执行 SQL 语句（INSERT/UPDATE/DELETE）",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"connection": {"type": "string", "description": "连接名称"},
					"sql": {"type": "string", "description": "SQL 语句"}
				},
				"required": ["connection", "sql"]
			}`),
		},
		{
			Name:        "list_tables",
			Description: "列出所有表",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"connection": {"type": "string", "description": "连接名称"}
				},
				"required": ["connection"]
			}`),
		},
		{
			Name:        "describe_table",
			Description: "查看表结构",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"connection": {"type": "string", "description": "连接名称"},
					"table": {"type": "string", "description": "表名"}
				},
				"required": ["connection", "table"]
			}`),
		},
		{
			Name:        "list_connections",
			Description: "列出所有数据库连接",
			InputSchema: json.RawMessage(`{}`),
		},
	}, nil
}

// CallTool 调用工具。
func (s *DatabaseServer) CallTool(ctx context.Context, name string, params json.RawMessage) (*ToolResult, error) {
	switch name {
	case "connect":
		return s.connect(params)
	case "disconnect":
		return s.disconnect(params)
	case "query":
		return s.query(params)
	case "execute":
		return s.execute(params)
	case "list_tables":
		return s.listTables(params)
	case "describe_table":
		return s.describeTable(params)
	case "list_connections":
		return s.listConnections(params)
	default:
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("未知工具：%s", name)}},
		}, nil
	}
}

// connect 连接数据库。
func (s *DatabaseServer) connect(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
		DSN    string `json:"dsn"`
	}
	json.Unmarshal(params, &p)

	if p.Driver == "" {
		p.Driver = "sqlite3"
	}

	db, err := sql.Open(p.Driver, p.DSN)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("连接失败：%v", err)}},
		}, nil
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("无法连接数据库：%v", err)}},
		}, nil
	}

	s.connections[p.Name] = db

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已连接到数据库 '%s'", p.Name)}},
		Data: map[string]interface{}{
			"name":   p.Name,
			"driver": p.Driver,
		},
	}, nil
}

// disconnect 断开连接。
func (s *DatabaseServer) disconnect(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Name string `json:"name"`
	}
	json.Unmarshal(params, &p)

	db, ok := s.connections[p.Name]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("连接 '%s' 不存在", p.Name)}},
		}, nil
	}

	if err := db.Close(); err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("关闭失败：%v", err)}},
		}, nil
	}

	delete(s.connections, p.Name)

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 已断开连接 '%s'", p.Name)}},
		Data: map[string]interface{}{
			"name": p.Name,
		},
	}, nil
}

// query 执行查询。
func (s *DatabaseServer) query(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Connection string `json:"connection"`
		SQL        string `json:"sql"`
	}
	json.Unmarshal(params, &p)

	db, ok := s.connections[p.Connection]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("连接 '%s' 不存在", p.Connection)}},
		}, nil
	}

	rows, err := db.Query(p.SQL)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("查询失败：%v", err)}},
		}, nil
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return &ToolResult{
				IsError: true,
				Content: []Content{{Type: "text", Text: fmt.Sprintf("扫描失败：%v", err)}},
			}, nil
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("查询结果 (%d 行):\n\n", len(results)))

	// 表头
	sb.WriteString("|")
	for _, col := range cols {
		sb.WriteString(fmt.Sprintf(" %s |", col))
	}
	sb.WriteString("\n")

	// 分隔线
	sb.WriteString("|")
	for range cols {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// 数据
	for _, row := range results {
		sb.WriteString("|")
		for _, col := range cols {
			sb.WriteString(fmt.Sprintf(" %v |", row[col]))
		}
		sb.WriteString("\n")
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"columns": cols,
			"rows":    results,
			"count":   len(results),
		},
	}, nil
}

// execute 执行 SQL 语句。
func (s *DatabaseServer) execute(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Connection string `json:"connection"`
		SQL        string `json:"sql"`
	}
	json.Unmarshal(params, &p)

	db, ok := s.connections[p.Connection]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("连接 '%s' 不存在", p.Connection)}},
		}, nil
	}

	result, err := db.Exec(p.SQL)
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("执行失败：%v", err)}},
		}, nil
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return &ToolResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("✓ 执行成功\n影响行数：%d\n最后插入 ID: %d", rowsAffected, lastInsertID)}},
		Data: map[string]interface{}{
			"rows_affected": rowsAffected,
			"last_insert_id": lastInsertID,
		},
	}, nil
}

// listTables 列出表。
func (s *DatabaseServer) listTables(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Connection string `json:"connection"`
	}
	json.Unmarshal(params, &p)

	db, ok := s.connections[p.Connection]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("连接 '%s' 不存在", p.Connection)}},
		}, nil
	}

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("查询失败：%v", err)}},
		}, nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("表列表 (共 %d 个):\n\n", len(tables)))
	for _, t := range tables {
		sb.WriteString(fmt.Sprintf("- %s\n", t))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"tables": tables,
			"count":  len(tables),
		},
	}, nil
}

// describeTable 查看表结构。
func (s *DatabaseServer) describeTable(params json.RawMessage) (*ToolResult, error) {
	var p struct {
		Connection string `json:"connection"`
		Table      string `json:"table"`
	}
	json.Unmarshal(params, &p)

	db, ok := s.connections[p.Connection]
	if !ok {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("连接 '%s' 不存在", p.Connection)}},
		}, nil
	}

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", p.Table))
	if err != nil {
		return &ToolResult{
			IsError: true,
			Content: []Content{{Type: "text", Text: fmt.Sprintf("查询失败：%v", err)}},
		}, nil
	}
	defer rows.Close()

	cols := []string{"cid", "name", "type", "notnull", "default", "pk"}
	var results []map[string]interface{}

	for rows.Next() {
		var cid, notnull, pk int
		var name, ttype, defval sql.NullString
		if err := rows.Scan(&cid, &name, &ttype, &notnull, &defval, &pk); err != nil {
			continue
		}

		row := map[string]interface{}{
			"cid":     cid,
			"name":    name.String,
			"type":    ttype.String,
			"notnull": notnull == 1,
			"default": defval.String,
			"pk":      pk > 0,
		}
		results = append(results, row)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("表 '%s' 结构:\n\n", p.Table))
	for _, r := range results {
		pk := ""
		if r["pk"].(bool) {
			pk = " [PK]"
		}
		sb.WriteString(fmt.Sprintf("- %s: %s%s\n", r["name"], r["type"], pk))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"columns": results,
		},
	}, nil
}

// listConnections 列出连接。
func (s *DatabaseServer) listConnections(params json.RawMessage) (*ToolResult, error) {
	if len(s.connections) == 0 {
		return &ToolResult{
			Content: []Content{{Type: "text", Text: "暂无数据库连接"}},
		}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("数据库连接列表 (共 %d 个):\n\n", len(s.connections)))
	for name := range s.connections {
		sb.WriteString(fmt.Sprintf("- %s\n", name))
	}

	return &ToolResult{
		Content: []Content{{Type: "text", Text: sb.String()}},
		Data: map[string]interface{}{
			"count": len(s.connections),
		},
	}, nil
}

// ListResources 列出资源。
func (s *DatabaseServer) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	return &ListResourcesResult{
		Resources:       []Resource{},
		ResourceTemplates: []ResourceTemplate{},
	}, nil
}

// ReadResource 读取资源。
func (s *DatabaseServer) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	return nil, ErrResourceNotFound
}

// Close 关闭所有连接。
func (s *DatabaseServer) Close() {
	for name, db := range s.connections {
		db.Close()
		delete(s.connections, name)
	}
}
