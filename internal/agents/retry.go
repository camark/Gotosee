// Package agents 提供重试管理功能。
package agents

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// RetryManager 重试管理器。
type RetryManager struct {
	mu            sync.Mutex
	retryAttempts uint32
}

// RetryResult 重试结果。
type RetryResult struct {
	ShouldRetry bool
	Error       error
}

// NewRetryManager 创建重试管理器。
func NewRetryManager() *RetryManager {
	return &RetryManager{
		retryAttempts: 0,
	}
}

// Reset 重置重试计数。
func (rm *RetryManager) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.retryAttempts = 0
}

// Increment 增加重试计数并返回新值。
func (rm *RetryManager) Increment() uint32 {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.retryAttempts++
	return rm.retryAttempts
}

// Get 获取当前重试计数。
func (rm *RetryManager) Get() uint32 {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.retryAttempts
}

// ShouldRetry 检查是否应该重试。
func (rm *RetryManager) ShouldRetry(config *RetryConfig) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.retryAttempts < config.MaxRetries
}

// ExecuteWithRetry 带重试地执行操作。
func (rm *RetryManager) ExecuteWithRetry(
	ctx context.Context,
	config *RetryConfig,
	operation func() error,
	checkSuccess func() bool,
) error {
	if err := config.Validate(); err != nil {
		return err
	}

	rm.Reset()

	for {
		// 执行操作
		if err := operation(); err != nil {
			return err
		}

		// 检查是否成功
		if checkSuccess() {
			return nil
		}

		// 检查重试次数
		rm.mu.Lock()
		rm.retryAttempts++
		attempts := rm.retryAttempts
		rm.mu.Unlock()

		if attempts >= config.MaxRetries {
			// 达到最大重试次数，执行失败处理
			if config.OnFailure != nil {
				timeout := DEFAULT_ON_FAILURE_TIMEOUT_SECONDS
				if config.OnFailureTimeoutSeconds != nil {
					timeout = *config.OnFailureTimeoutSeconds
				}
				execCmdWithTimeout(*config.OnFailure, timeout)
			}
			return fmt.Errorf("max retries (%d) exceeded", config.MaxRetries)
		}
	}
}

// CheckSuccess 检查成功条件。
func (rm *RetryManager) CheckSuccess(checks []SuccessCheck) bool {
	for _, check := range checks {
		if !executeSuccessCheck(check) {
			return false
		}
	}
	return true
}

// executeSuccessCheck 执行单个成功检查。
func executeSuccessCheck(check SuccessCheck) bool {
	cmd := exec.Command("sh", "-c", check.Command)
	err := cmd.Run()
	return err == nil
}

// execCmdWithTimeout 执行命令并超时。
func execCmdWithTimeout(command string, timeoutSeconds uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Run()
}

// 默认常量
const (
	DEFAULT_RETRY_TIMEOUT_SECONDS       uint64 = 300 // 5 分钟
	DEFAULT_ON_FAILURE_TIMEOUT_SECONDS  uint64 = 600 // 10 分钟
)
