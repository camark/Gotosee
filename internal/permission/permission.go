// Package permission 提供权限管理功能。
package permission

import (
	"sync"
)

// Permission 权限级别。
type Permission string

const (
	// AllowOnce 允许一次。
	AllowOnce Permission = "allow_once"
	// AlwaysAllow 总是允许。
	AlwaysAllow Permission = "always_allow"
	// DenyOnce 拒绝一次。
	DenyOnce Permission = "deny_once"
	// AlwaysDeny 总是拒绝。
	AlwaysDeny Permission = "always_deny"
)

// IsValid 检查权限级别是否有效。
func (p Permission) IsValid() bool {
	switch p {
	case AllowOnce, AlwaysAllow, DenyOnce, AlwaysDeny:
		return true
	}
	return false
}

// String 返回权限的字符串表示。
func (p Permission) String() string {
	return string(p)
}

// PermissionLevel 权限级别（用于持久化）。
type PermissionLevel string

const (
	// PermissionLevelAsk 询问用户。
	PermissionLevelAsk PermissionLevel = "ask"
	// PermissionLevelAlwaysAllow 总是允许。
	PermissionLevelAlwaysAllow PermissionLevel = "always_allow"
	// PermissionLevelNeverAllow 从不允许。
	PermissionLevelNeverAllow PermissionLevel = "never_allow"
)

// IsValid 检查权限级别是否有效。
func (p PermissionLevel) IsValid() bool {
	switch p {
	case PermissionLevelAsk, PermissionLevelAlwaysAllow, PermissionLevelNeverAllow:
		return true
	}
	return false
}

// String 返回权限级别的字符串表示。
func (p PermissionLevel) String() string {
	return string(p)
}

// PermissionManager 权限管理器。
type PermissionManager struct {
	mu          sync.RWMutex
	permissions map[string]PermissionLevel // tool_name -> level
}

// NewPermissionManager 创建权限管理器。
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		permissions: make(map[string]PermissionLevel),
	}
}

// GetPermission 获取工具的权限级别。
func (pm *PermissionManager) GetPermission(toolName string) PermissionLevel {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.permissions[toolName]
}

// SetPermission 设置工具的权限级别。
func (pm *PermissionManager) SetPermission(toolName string, level PermissionLevel) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.permissions[toolName] = level
}

// ShouldAutoApprove 检查工具是否应该自动批准。
func (pm *PermissionManager) ShouldAutoApprove(toolName string) bool {
	return pm.GetPermission(toolName) == PermissionLevelAlwaysAllow
}

// ShouldAutoDeny 检查工具是否应该自动拒绝。
func (pm *PermissionManager) ShouldAutoDeny(toolName string) bool {
	return pm.GetPermission(toolName) == PermissionLevelNeverAllow
}
