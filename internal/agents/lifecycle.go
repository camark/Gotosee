package agents

import (
	"sync"
	"time"
)

// AgentLifecycle Agent 生命周期状态。
type AgentLifecycle string

const (
	AgentLifecycleCreated      AgentLifecycle = "created"
	AgentLifecycleInitializing AgentLifecycle = "initializing"
	AgentLifecycleReady        AgentLifecycle = "ready"
	AgentLifecycleProcessing   AgentLifecycle = "processing"
	AgentLifecycleClosing      AgentLifecycle = "closing"
	AgentLifecycleClosed       AgentLifecycle = "closed"
)

// LifecycleManager 生命周期管理器。
type LifecycleManager struct {
	mu              sync.RWMutex
	currentState    AgentLifecycle
	lastStateChange time.Time
	history         []LifecycleEvent
}

// LifecycleEvent 生命周期事件。
type LifecycleEvent struct {
	From AgentLifecycle
	To   AgentLifecycle
	Time time.Time
}

// NewLifecycleManager 创建生命周期管理器。
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		currentState: AgentLifecycleCreated,
		history:      make([]LifecycleEvent, 0),
	}
}

// GetState 获取当前状态。
func (lm *LifecycleManager) GetState() AgentLifecycle {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState
}

// Transition 状态转换。
func (lm *LifecycleManager) Transition(to AgentLifecycle) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 验证状态转换合法性
	validTransitions := map[AgentLifecycle][]AgentLifecycle{
		AgentLifecycleCreated:      {AgentLifecycleInitializing},
		AgentLifecycleInitializing: {AgentLifecycleReady},
		AgentLifecycleReady:        {AgentLifecycleProcessing, AgentLifecycleClosing},
		AgentLifecycleProcessing:   {AgentLifecycleReady, AgentLifecycleClosing},
		AgentLifecycleClosing:      {AgentLifecycleClosed},
		AgentLifecycleClosed:       {},
	}

	allowed := validTransitions[lm.currentState]
	valid := false
	for _, s := range allowed {
		if s == to {
			valid = true
			break
		}
	}

	if !valid {
		return &LifecycleError{
			From: lm.currentState,
			To:   to,
		}
	}

	from := lm.currentState
	lm.currentState = to
	lm.lastStateChange = time.Now()
	lm.history = append(lm.history, LifecycleEvent{
		From: from,
		To:   to,
		Time: time.Now(),
	})

	return nil
}

// IsProcessing 检查是否在处理中。
func (lm *LifecycleManager) IsProcessing() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState == AgentLifecycleProcessing
}

// IsReady 检查是否就绪。
func (lm *LifecycleManager) IsReady() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState == AgentLifecycleReady
}

// IsClosed 检查是否已关闭。
func (lm *LifecycleManager) IsClosed() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.currentState == AgentLifecycleClosed
}

// LifecycleError 生命周期错误。
type LifecycleError struct {
	From AgentLifecycle
	To   AgentLifecycle
}

func (e *LifecycleError) Error() string {
	return "invalid lifecycle transition: " + string(e.From) + " -> " + string(e.To)
}
