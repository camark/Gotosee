// Package acp 提供 ACP (Agent Client Protocol) 协议实现。
package acp

import (
	"encoding/json"
	"time"
)

// ============================================================================
// ACP 基础类型
// ============================================================================

// SessionID 会话 ID。
type SessionID string

// ToolCallID 工具调用 ID。
type ToolCallID string

// ModelID 模型 ID。
type ModelID string

// SessionMode 会话模式。
type SessionMode string

const (
	ModeAuto         SessionMode = "auto"
	ModeApprove      SessionMode = "approve"
	ModeSmartApprove SessionMode = "smart_approve"
	ModeChat         SessionMode = "chat"
)

// ============================================================================
// 初始化
// ============================================================================

// InitializeRequest 初始化请求。
type InitializeRequest struct {
	ProtocolVersion string `json:"protocolVersion"`
	ClientInfo      ClientInfo `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// InitializeResponse 初始化响应。
type InitializeResponse struct {
	ProtocolVersion string           `json:"protocolVersion"`
	ServerInfo      ServerInfo       `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

// ClientInfo 客户端信息。
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo 服务器信息。
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities 客户端能力。
type ClientCapabilities struct {
	Tools      *ToolCapabilities      `json:"tools,omitempty"`
	Resources  *ResourceCapabilities  `json:"resources,omitempty"`
	Prompts    *PromptCapabilities    `json:"prompts,omitempty"`
	Sessions   *SessionCapabilities   `json:"sessions,omitempty"`
}

// ServerCapabilities 服务器能力。
type ServerCapabilities struct {
	Tools      *ToolCapabilities      `json:"tools,omitempty"`
	Resources  *ResourceCapabilities  `json:"resources,omitempty"`
	Prompts    *PromptCapabilities    `json:"prompts,omitempty"`
	Sessions   *SessionCapabilities   `json:"sessions,omitempty"`
	Models     *ModelCapabilities     `json:"models,omitempty"`
}

// ToolCapabilities 工具能力。
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapabilities 资源能力。
type ResourceCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptCapabilities 提示能力。
type PromptCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SessionCapabilities 会话能力。
type SessionCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ModelCapabilities 模型能力。
type ModelCapabilities struct{}

// ============================================================================
// 会话管理
// ============================================================================

// NewSessionRequest 创建会话请求。
type NewSessionRequest struct {
	Mode          *SessionMode `json:"mode,omitempty"`
	Model         *ModelInfo   `json:"model,omitempty"`
	Configuration *json.RawMessage `json:"configuration,omitempty"`
}

// NewSessionResponse 创建会话响应。
type NewSessionResponse struct {
	SessionID SessionID `json:"sessionId"`
}

// CloseSessionRequest 关闭会话请求。
type CloseSessionRequest struct {
	SessionID SessionID `json:"sessionId"`
}

// CloseSessionResponse 关闭会话响应。
type CloseSessionResponse struct{}

// ListSessionsRequest 列出会话请求。
type ListSessionsRequest struct{}

// ListSessionsResponse 列出会话响应。
type ListSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

// SessionInfo 会话信息。
type SessionInfo struct {
	ID        SessionID  `json:"id"`
	Mode      SessionMode `json:"mode"`
	Model     *ModelInfo `json:"model,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Status    SessionStatus `json:"status"`
}

// SessionStatus 会话状态。
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusIdle     SessionStatus = "idle"
	SessionStatusClosed   SessionStatus = "closed"
	SessionStatusError    SessionStatus = "error"
)

// SetSessionModeRequest 设置会话模式请求。
type SetSessionModeRequest struct {
	SessionID SessionID   `json:"sessionId"`
	Mode      SessionMode `json:"mode"`
}

// SetSessionModeResponse 设置会话模式响应。
type SetSessionModeResponse struct{}

// ForkSessionRequest 复制会话请求。
type ForkSessionRequest struct {
	SessionID SessionID `json:"sessionId"`
	FromMessageID string `json:"fromMessageId,omitempty"`
}

// ForkSessionResponse 复制会话响应。
type ForkSessionResponse struct {
	SessionID SessionID `json:"sessionId"`
}

// LoadSessionRequest 加载会话请求。
type LoadSessionRequest struct {
	SessionID SessionID `json:"sessionId"`
}

// LoadSessionResponse 加载会话响应。
type LoadSessionResponse struct {
	Session SessionInfo `json:"session"`
	Messages []Message `json:"messages"`
}

// ============================================================================
// 模型管理
// ============================================================================

// ModelInfo 模型信息。
type ModelInfo struct {
	ID            ModelID  `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	ContextWindow int      `json:"context_window,omitempty"`
}

// SetSessionModelRequest 设置会话模型请求。
type SetSessionModelRequest struct {
	SessionID SessionID `json:"sessionId"`
	Model     ModelInfo `json:"model"`
}

// SetSessionModelResponse 设置会话模型响应。
type SetSessionModelResponse struct {
	CurrentModel ModelInfo `json:"current_model"`
}

// ============================================================================
// 工具调用
// ============================================================================

// ToolCall 工具调用请求。
type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	ToolCallID ToolCallID     `json:"tool_call_id,omitempty"`
}

// ToolCallResult 工具调用结果。
type ToolCallResult struct {
	ToolCallID ToolCallID      `json:"tool_call_id"`
	Status     ToolCallStatus  `json:"status"`
	Content    []ContentBlock  `json:"content,omitempty"`
	Error      *ToolCallError  `json:"error,omitempty"`
}

// ToolCallStatus 工具调用状态。
type ToolCallStatus string

const (
	ToolCallStatusPending   ToolCallStatus = "pending"
	ToolCallStatusExecuting ToolCallStatus = "executing"
	ToolCallStatusComplete  ToolCallStatus = "complete"
	ToolCallStatusError     ToolCallStatus = "error"
	ToolCallStatusCancelled ToolCallStatus = "cancelled"
)

// ToolCallError 工具调用错误。
type ToolCallError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ToolCallUpdate 工具调用更新通知。
type ToolCallUpdate struct {
	Fields ToolCallUpdateFields `json:"fields"`
}

// ToolCallUpdateFields 工具调用更新字段。
type ToolCallUpdateFields struct {
	ToolCallID ToolCallID     `json:"tool_call_id"`
	Status     ToolCallStatus `json:"status"`
	Content    []ContentBlock `json:"content,omitempty"`
}

// ============================================================================
// 权限管理
// ============================================================================

// PermissionOptionKind 权限选项类型。
type PermissionOptionKind string

const (
	PermissionApproveOnce   PermissionOptionKind = "approve_once"
	PermissionApproveAlways PermissionOptionKind = "approve_always"
	PermissionDeny          PermissionOptionKind = "deny"
	PermissionTrustSession  PermissionOptionKind = "trust_session"
)

// PermissionOption 权限选项。
type PermissionOption struct {
	Kind        PermissionOptionKind `json:"kind"`
	Label       string               `json:"label"`
	Description string               `json:"description,omitempty"`
}

// RequestPermissionRequest 请求权限请求。
type RequestPermissionRequest struct {
	SessionID      SessionID        `json:"sessionId"`
	ToolCallID     ToolCallID       `json:"tool_call_id"`
	ToolName       string           `json:"tool_name"`
	Arguments      json.RawMessage  `json:"arguments,omitempty"`
	PermissionOpts []PermissionOption `json:"permission_options"`
}

// RequestPermissionResponse 请求权限响应。
type RequestPermissionResponse struct {
	Outcome RequestPermissionOutcome `json:"outcome"`
}

// RequestPermissionOutcome 请求权限结果。
type RequestPermissionOutcome struct {
	Decision       *PermissionDecision  `json:"decision,omitempty"`
	ApplyToSession bool                 `json:"apply_to_session,omitempty"`
}

// PermissionDecision 权限决定。
type PermissionDecision struct {
	Approve bool `json:"approve"`
	ApplyToSimilarTools bool `json:"apply_to_similar_tools,omitempty"`
}

// ============================================================================
// 内容类型
// ============================================================================

// ContentBlock 内容块。
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	ImageData *ImageData `json:"image_data,omitempty"`
	Resource *EmbeddedResource `json:"resource,omitempty"`
}

// ImageData 图片数据。
type ImageData struct {
	Data     string `json:"data"` // base64
	MIMEType string `json:"mime_type"`
}

// EmbeddedResource 嵌入式资源。
type EmbeddedResource struct {
	Type     string      `json:"type"`
	Resource interface{} `json:"resource"`
}

// TextResourceContents 文本资源内容。
type TextResourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mime_type,omitempty"`
	Text     string `json:"text"`
}

// BlobResourceContents 二进制资源内容。
type BlobResourceContents struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mime_type,omitempty"`
	Blob     string `json:"blob"` // base64
}

// ResourceLink 资源链接。
type ResourceLink struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// ============================================================================
// 消息类型
// ============================================================================

// Message ACP 消息。
type Message struct {
	ID        string        `json:"id"`
	Role      MessageRole   `json:"role"`
	Content   []ContentBlock `json:"content"`
	Timestamp time.Time     `json:"timestamp"`
	Meta      *Meta         `json:"meta,omitempty"`
}

// MessageRole 消息角色。
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// Meta 消息元数据。
type Meta struct {
	Data map[string]interface{} `json:"data,omitempty"`
}

// Content 内容类型。
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ============================================================================
// 通知
// ============================================================================

// SessionNotification 会话通知。
type SessionNotification struct {
	SessionID SessionID `json:"sessionId"`
	Type      string    `json:"type"`
	Data      interface{} `json:"data,omitempty"`
}

// CancelNotification 取消通知。
type CancelNotification struct {
	Reason string `json:"reason,omitempty"`
}

// ============================================================================
// 文件系统能力
// ============================================================================

// FileSystemCapabilities 文件系统能力。
type FileSystemCapabilities struct {
	Supported bool `json:"supported"`
}

// ============================================================================
// MCP 能力
// ============================================================================

// McpCapabilities MCP 能力。
type McpCapabilities struct {
	Servers []McpServer `json:"servers,omitempty"`
}

// McpServer MCP 服务器。
type McpServer struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ============================================================================
// Agent 能力
// ============================================================================

// AgentCapabilities Agent 能力。
type AgentCapabilities struct {
	Autonomy string `json:"autonomy,omitempty"`
}

// ============================================================================
// 会话配置选项
// ============================================================================

// SessionConfigOption 会话配置选项。
type SessionConfigOption struct {
	Category SessionConfigOptionCategory `json:"category"`
	Key      string                      `json:"key"`
	Label    string                      `json:"label"`
	Value    interface{}                 `json:"value"`
}

// SessionConfigOptionCategory 配置选项类别。
type SessionConfigOptionCategory string

const (
	CategoryModel     SessionConfigOptionCategory = "model"
	CategoryMode      SessionConfigOptionCategory = "mode"
	CategoryExtension SessionConfigOptionCategory = "extension"
)

// SessionConfigSelectOption 配置选择选项。
type SessionConfigSelectOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// ConfigOptionUpdate 配置选项更新。
type ConfigOptionUpdate struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// SetSessionConfigOptionRequest 设置会话配置选项请求。
type SetSessionConfigOptionRequest struct {
	SessionID SessionID           `json:"sessionId"`
	Updates   []ConfigOptionUpdate `json:"updates"`
}

// SetSessionConfigOptionResponse 设置会话配置选项响应。
type SetSessionConfigOptionResponse struct{}

// ============================================================================
// Prompt
// ============================================================================

// PromptRequest Prompt 请求。
type PromptRequest struct {
	Name       string                 `json:"name"`
	Arguments  map[string]interface{} `json:"arguments,omitempty"`
}

// PromptResponse Prompt 响应。
type PromptResponse struct {
	Messages []Message `json:"messages"`
}

// ============================================================================
// 认证
// ============================================================================

// AuthMethod 认证方法。
type AuthMethod string

const (
	AuthMethodBearerToken AuthMethod = "bearer_token"
	AuthMethodOAuth2      AuthMethod = "oauth2"
	AuthMethodNone        AuthMethod = "none"
)

// AuthMethodAgent 认证方法 Agent。
type AuthMethodAgent struct {
	Type   AuthMethod `json:"type"`
	Config *json.RawMessage `json:"config,omitempty"`
}

// AuthenticateRequest 认证请求。
type AuthenticateRequest struct {
	Method AuthMethod `json:"method"`
	Config *json.RawMessage `json:"config,omitempty"`
}

// AuthenticateResponse 认证响应。
type AuthenticateResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
}

// ============================================================================
// 停止原因
// ============================================================================

// StopReason 停止原因。
type StopReason string

const (
	StopReasonEndTurn     StopReason = "end_turn"
	StopReasonToolCalls   StopReason = "tool_calls"
	StopReasonMaxTokens   StopReason = "max_tokens"
	StopReasonStopSequence StopReason = "stop_sequence"
	StopReasonCancelled   StopReason = "cancelled"
	StopReasonError       StopReason = "error"
)

// CurrentModeUpdate 当前模式更新。
type CurrentModeUpdate struct {
	Mode SessionMode `json:"mode"`
}

// SessionModeState 会话模式状态。
type SessionModeState struct {
	Mode SessionMode `json:"mode"`
}

// SessionModelState 会话模型状态。
type SessionModelState struct {
	Model ModelInfo `json:"model"`
}

// SessionCloseCapabilities 会话关闭能力。
type SessionCloseCapabilities struct{}

// SessionListCapabilities 会话列表能力。
type SessionListCapabilities struct{}

// SessionUpdate 会话更新。
type SessionUpdate struct {
	Fields map[string]interface{} `json:"fields"`
}

// SessionNotification 会话通知类型。
type SessionNotificationType struct {
	Type string `json:"type"`
}

// Meta 元数据。
type MetaData struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// ContentChunk 内容块。
type ContentChunk struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolKind 工具类型。
type ToolKind string

const (
	ToolKindBuiltin ToolKind = "builtin"
	ToolKindMCP     ToolKind = "mcp"
	ToolKindCustom  ToolKind = "custom"
)

// ToolCallContent 工具调用内容。
type ToolCallContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ImageData *ImageData      `json:"image_data,omitempty"`
	Resource  *EmbeddedResource `json:"resource,omitempty"`
}

// ToolCallLocation 工具调用位置。
type ToolCallLocation struct {
	Line      int `json:"line,omitempty"`
	Column    int `json:"column,omitempty"`
	Offset    int `json:"offset,omitempty"`
}

// ActionRequiredData 需要操作的数据。
type ActionRequiredData struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Data        interface{} `json:"data"`
}
