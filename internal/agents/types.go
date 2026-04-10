// Package agents 提供代理逻辑、工具调度和消息处理功能。
package agents

import (
	"strings"
	"sync"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/mcp"
	"github.com/aaif-goose/gogo/internal/providers"
	"github.com/aaif-goose/gogo/internal/session"
)

// ============================================================================
// 基础类型
// ============================================================================

// ToolResultReceiver 工具结果接收器。
type ToolResultReceiver chan *ToolCallResult

// SharedProvider 共享的提供商（线程安全）。
type SharedProvider struct {
	mu       sync.RWMutex
	provider providers.Provider
}

// NewSharedProvider 创建共享提供商。
func NewSharedProvider(p providers.Provider) *SharedProvider {
	return &SharedProvider{
		provider: p,
	}
}

// Get 获取提供商。
func (sp *SharedProvider) Get() providers.Provider {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.provider
}

// Set 设置提供商。
func (sp *SharedProvider) Set(p providers.Provider) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.provider = p
}

// ToolCallResult 工具调用结果。
type ToolCallResult struct {
	ToolName string
	Content  []*mcp.Content
	Error    error
}

// Text 将结果转换为文本。
func (t *ToolCallResult) Text() string {
	var sb strings.Builder
	for _, c := range t.Content {
		sb.WriteString(c.Text)
	}
	return sb.String()
}

// ============================================================================
// 配置类型
// ============================================================================

// RetryConfig 重试配置。
type RetryConfig struct {
	// 最大重试次数
	MaxRetries uint32 `json:"max_retries"`
	// 成功检查列表
	Checks []SuccessCheck `json:"checks"`
	// 失败时执行的 shell 命令
	OnFailure *string `json:"on_failure,omitempty"`
	// 单个命令的超时时间（秒）
	TimeoutSeconds *uint64 `json:"timeout_seconds,omitempty"`
	// 失败命令的超时时间（秒）
	OnFailureTimeoutSeconds *uint64 `json:"on_failure_timeout_seconds,omitempty"`
}

// Validate 验证重试配置。
func (c *RetryConfig) Validate() error {
	if c.MaxRetries == 0 {
		return &ConfigError{Field: "max_retries", Message: "must be greater than 0"}
	}
	if c.TimeoutSeconds != nil && *c.TimeoutSeconds == 0 {
		return &ConfigError{Field: "timeout_seconds", Message: "must be greater than 0 if specified"}
	}
	if c.OnFailureTimeoutSeconds != nil && *c.OnFailureTimeoutSeconds == 0 {
		return &ConfigError{Field: "on_failure_timeout_seconds", Message: "must be greater than 0 if specified"}
	}
	return nil
}

// SuccessCheck 成功检查。
type SuccessCheck struct {
	// Shell 命令检查
	Command string `json:"command"`
}

// SessionConfig 会话配置。
type SessionConfig struct {
	// 会话 ID
	ID string `json:"id"`
	// 调度 ID（如果由调度触发）
	ScheduleID *string `json:"schedule_id,omitempty"`
	// 最大轮次限制
	MaxTurns *uint32 `json:"max_turns,omitempty"`
	// 重试配置
	RetryConfig *RetryConfig `json:"retry_config,omitempty"`
}

// FrontendTool 前端工具（由前端执行）。
type FrontendTool struct {
	Name string
	Tool *mcp.Tool
}

// ConfigError 配置错误。
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}

// ============================================================================
// Agent 事件类型
// ============================================================================

// AgentEvent 代理事件。
type AgentEvent interface {
	isAgentEvent()
}

// MessageEvent 消息事件。
type MessageEvent struct {
	Message *conversation.Message
}

func (e MessageEvent) isAgentEvent() {}

// McpNotificationEvent MCP 通知事件。
type McpNotificationEvent struct {
	ExtensionName string
	Notification  *mcp.ServerNotification
}

func (e McpNotificationEvent) isAgentEvent() {}

// HistoryReplacedEvent 历史替换事件。
type HistoryReplacedEvent struct {
	Conversation *conversation.Conversation
}

func (e HistoryReplacedEvent) isAgentEvent() {}

// ToolCallEvent 工具调用事件。
type ToolCallEvent struct {
	Name      string
	Arguments map[string]interface{}
}

func (e ToolCallEvent) isAgentEvent() {}

// ToolResultEvent 工具结果事件。
type ToolResultEvent struct {
	Name    string
	Content []*mcp.Content
	Error   error
}

func (e ToolResultEvent) isAgentEvent() {}

// ============================================================================
// Goose 平台类型
// ============================================================================

// GoosePlatform Goose 平台类型。
type GoosePlatform string

const (
	GoosePlatformDesktop GoosePlatform = "goose-desktop"
	GoosePlatformCLI     GoosePlatform = "goose-cli"
)

// ============================================================================
// Agent 配置
// ============================================================================

// AgentConfig Agent 配置。
type AgentConfig struct {
	SessionManager      *session.SessionManager
	PermissionManager   interface{} // TODO: 实现 permission manager
	SchedulerService    interface{} // TODO: 实现 scheduler trait
	GooseMode           string
	DisableSessionNaming bool
	GoosePlatform       GoosePlatform
}

// NewAgentConfig 创建 Agent 配置。
func NewAgentConfig(
	sessionManager *session.SessionManager,
	gooseMode string,
	disableSessionNaming bool,
	platform GoosePlatform,
) *AgentConfig {
	return &AgentConfig{
		SessionManager:       sessionManager,
		GooseMode:            gooseMode,
		DisableSessionNaming: disableSessionNaming,
		GoosePlatform:        platform,
	}
}

// ============================================================================
// 常量
// ============================================================================

const (
	// DefaultMaxTurns 默认最大轮次。
	DefaultMaxTurns uint32 = 1000

	// CompactionThinkingText  compaciton 时的思考文本。
	CompactionThinkingText = "goose is compacting the conversation..."

	// DefaultCompactionThreshold 默认压缩阈值。
	DefaultCompactionThreshold = 100

	// FinalOutputToolName 最终输出工具名称。
	FinalOutputToolName = "final_output"

	// ManageExtensionsToolName 扩展管理工具名称。
	ManageExtensionsToolName = "manage_extensions"

	// PlatformManageScheduleToolName 调度管理工具名称。
	PlatformManageScheduleToolName = "platform_manage_schedule"
)
