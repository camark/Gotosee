package agents

import (
	"testing"
)

func TestLifecycleManager_ValidTransitions(t *testing.T) {
	lm := NewLifecycleManager()

	if lm.GetState() != AgentLifecycleCreated {
		t.Errorf("expected Created state, got %v", lm.GetState())
	}

	// 初始化
	if err := lm.Transition(AgentLifecycleInitializing); err != nil {
		t.Fatalf("transition to Initializing failed: %v", err)
	}

	// 就绪
	if err := lm.Transition(AgentLifecycleReady); err != nil {
		t.Fatalf("transition to Ready failed: %v", err)
	}

	// 处理中
	if err := lm.Transition(AgentLifecycleProcessing); err != nil {
		t.Fatalf("transition to Processing failed: %v", err)
	}

	// 返回就绪
	if err := lm.Transition(AgentLifecycleReady); err != nil {
		t.Fatalf("transition back to Ready failed: %v", err)
	}
}

func TestLifecycleManager_InvalidTransition(t *testing.T) {
	lm := NewLifecycleManager()

	// 尝试非法转换
	err := lm.Transition(AgentLifecycleProcessing)
	if err == nil {
		t.Error("expected error for invalid transition")
	}
}
