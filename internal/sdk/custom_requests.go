// Package sdk 提供 goose SDK 类型定义。
// 这些类型用于 goose 与服务器之间的 JSON-RPC 通信。
package sdk

import "encoding/json"

// CustomMethodSchema 描述自定义方法的模式。
// 对应 Rust 中的 CustomMethodSchema 结构体。
type CustomMethodSchema struct {
	Method           string                 `json:"method"`
	ParamsSchema     map[string]interface{} `json:"params_schema,omitempty"`
	ParamsTypeName   string                 `json:"params_type_name,omitempty"`
	ResponseSchema   map[string]interface{} `json:"response_schema,omitempty"`
	ResponseTypeName string                 `json:"response_type_name,omitempty"`
}

// ============================================================================
// 扩展管理请求/响应
// ============================================================================

// AddExtensionRequest 添加扩展请求。
// JSON-RPC method: _goose/extensions/add
type AddExtensionRequest struct {
	SessionID string          `json:"sessionId"`
	Config    json.RawMessage `json:"config,omitempty"`
}

// RemoveExtensionRequest 移除扩展请求。
// JSON-RPC method: _goose/extensions/remove
type RemoveExtensionRequest struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name"`
}

// GetToolsRequest 获取工具列表请求。
// JSON-RPC method: _goose/tools
type GetToolsRequest struct {
	SessionID string `json:"sessionId"`
}

// ToolInfo 工具信息。
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Permission  string                 `json:"permission,omitempty"`
}

// GetToolsResponse 获取工具列表响应。
type GetToolsResponse struct {
	Tools []ToolInfo `json:"tools"`
}

// ============================================================================
// 资源管理请求/响应
// ============================================================================

// ReadResourceRequest 读取资源请求。
// JSON-RPC method: _goose/resource/read
type ReadResourceRequest struct {
	SessionID     string `json:"sessionId"`
	URI           string `json:"uri"`
	ExtensionName string `json:"extension_name"`
}

// ReadResourceResponse 读取资源响应。
type ReadResourceResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
}

// ============================================================================
// 会话管理请求/响应
// ============================================================================

// UpdateWorkingDirRequest 更新工作目录请求。
// JSON-RPC method: _goose/working_dir/update
type UpdateWorkingDirRequest struct {
	SessionID   string `json:"sessionId"`
	WorkingDir  string `json:"working_dir"`
}

// DeleteSessionRequest 删除会话请求。
// JSON-RPC method: session/delete
type DeleteSessionRequest struct {
	SessionID string `json:"sessionId"`
}

// ArchiveSessionRequest 归档会话请求。
// JSON-RPC method: _goose/session/archive
type ArchiveSessionRequest struct {
	SessionID string `json:"sessionId"`
}

// UnarchiveSessionRequest 取消归档会话请求。
// JSON-RPC method: _goose/session/unarchive
type UnarchiveSessionRequest struct {
	SessionID string `json:"sessionId"`
}

// ExportSessionRequest 导出会话请求。
// JSON-RPC method: _goose/session/export
type ExportSessionRequest struct {
	SessionID string `json:"sessionId"`
}

// ExportSessionResponse 导出会话响应。
type ExportSessionResponse struct {
	Data string `json:"data"`
}

// ImportSessionRequest 导入会话请求。
// JSON-RPC method: _goose/session/import
type ImportSessionRequest struct {
	Data string `json:"data"`
}

// ImportSessionResponse 导入会话响应。
type ImportSessionResponse struct {
	SessionID     string  `json:"sessionId"`
	Title         *string `json:"title,omitempty"`
	UpdatedAt     *string `json:"updated_at,omitempty"`
	MessageCount  uint64  `json:"message_count"`
}

// ============================================================================
// 配置管理请求/响应
// ============================================================================

// GetExtensionsRequest 获取扩展列表请求。
// JSON-RPC method: _goose/config/extensions
type GetExtensionsRequest struct{}

// ExtensionEntry 扩展条目。
type ExtensionEntry struct {
	Enabled bool               `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// GetExtensionsResponse 获取扩展列表响应。
type GetExtensionsResponse struct {
	Extensions []ExtensionEntry `json:"extensions"`
	Warnings   []string         `json:"warnings"`
}

// ReadConfigRequest 读取配置请求。
// JSON-RPC method: _goose/config/read
type ReadConfigRequest struct {
	Key string `json:"key"`
}

// ReadConfigResponse 读取配置响应。
type ReadConfigResponse struct {
	Value json.RawMessage `json:"value,omitempty"`
}

// UpsertConfigRequest 更新/插入配置请求。
// JSON-RPC method: _goose/config/upsert
type UpsertConfigRequest struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// RemoveConfigRequest 移除配置请求。
// JSON-RPC method: _goose/config/remove
type RemoveConfigRequest struct {
	Key string `json:"key"`
}

// ============================================================================
// 密钥管理请求/响应
// ============================================================================

// CheckSecretRequest 检查密钥是否存在请求。
// JSON-RPC method: _goose/secret/check
type CheckSecretRequest struct {
	Key string `json:"key"`
}

// CheckSecretResponse 检查密钥响应。
type CheckSecretResponse struct {
	Exists bool `json:"exists"`
}

// UpsertSecretRequest 更新/插入密钥请求。
// JSON-RPC method: _goose/secret/upsert
type UpsertSecretRequest struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// RemoveSecretRequest 移除密钥请求。
// JSON-RPC method: _goose/secret/remove
type RemoveSecretRequest struct {
	Key string `json:"key"`
}

// ============================================================================
// Provider 管理请求/响应
// ============================================================================

// ListProvidersRequest 列出 Provider 请求。
// JSON-RPC method: _goose/providers/list
type ListProvidersRequest struct{}

// ProviderListEntry Provider 列表条目。
type ProviderListEntry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ListProvidersResponse Provider 列表响应。
type ListProvidersResponse struct {
	Providers []ProviderListEntry `json:"providers"`
}

// UpdateProviderRequest 更新 Provider 请求。
// JSON-RPC method: _goose/session/provider/update
type UpdateProviderRequest struct {
	SessionID      string                 `json:"sessionId"`
	Provider       string                 `json:"provider"`
	Model          *string                `json:"model,omitempty"`
	ContextLimit   *int                   `json:"context_limit,omitempty"`
	RequestParams  map[string]interface{} `json:"request_params,omitempty"`
}

// UpdateProviderResponse 更新 Provider 响应。
type UpdateProviderResponse struct {
	ConfigOptions []map[string]interface{} `json:"config_options"`
}

// ============================================================================
// 通用响应
// ============================================================================

// EmptyResponse 空响应，用于不返回数据的操作。
type EmptyResponse struct{}
