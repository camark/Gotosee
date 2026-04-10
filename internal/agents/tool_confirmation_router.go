// Package agents 提供工具调用确认路由功能。
package agents

import (
	"sync"

	"github.com/camark/Gotosee/internal/permission"
)

// PermissionConfirmation 权限确认。
type PermissionConfirmation struct {
	Permission permission.Permission
}

// ToolConfirmationRouter 工具调用确认路由器。
type ToolConfirmationRouter struct {
	mu            sync.Mutex
	confirmations map[string]chan PermissionConfirmation
}

// NewToolConfirmationRouter 创建工具确认路由器。
func NewToolConfirmationRouter() *ToolConfirmationRouter {
	return &ToolConfirmationRouter{
		confirmations: make(map[string]chan PermissionConfirmation),
	}
}

// Register 注册一个工具调用请求的确认通道。
func (r *ToolConfirmationRouter) Register(requestID string) <-chan PermissionConfirmation {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan PermissionConfirmation, 1)
	r.confirmations[requestID] = ch
	return ch
}

// Deliver 传递确认到指定请求。
func (r *ToolConfirmationRouter) Deliver(requestID string, confirmation PermissionConfirmation) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch, exists := r.confirmations[requestID]
	if !exists {
		return false
	}

	select {
	case ch <- confirmation:
		delete(r.confirmations, requestID)
		close(ch)
		return true
	default:
		return false
	}
}
