// Package config 提供 goose 配置管理功能。
package config

// GooseMode 定义 goose 的运行模式。
type GooseMode string

const (
	// ModeAuto 自动批准工具调用。
	ModeAuto GooseMode = "auto"
	// ModeApprove 每次工具调用前都询问。
	ModeApprove GooseMode = "approve"
	// ModeSmartApprove 仅对敏感工具调用询问。
	ModeSmartApprove GooseMode = "smart_approve"
	// ModeChat 仅聊天，不使用工具。
	ModeChat GooseMode = "chat"
)

// DefaultGooseMode 返回默认的 goose 模式。
func DefaultGooseMode() GooseMode {
	return ModeAuto
}

// IsValid 检查模式是否有效。
func (m GooseMode) IsValid() bool {
	switch m {
	case ModeAuto, ModeApprove, ModeSmartApprove, ModeChat:
		return true
	}
	return false
}

// String 返回模式的字符串表示。
func (m GooseMode) String() string {
	return string(m)
}
