// Package agents 提供消息压缩功能。
package agents

import (
	"fmt"

	"github.com/aaif-goose/gogo/internal/conversation"
)

// needsCompaction 检查是否需要压缩消息。
func needsCompaction(messages []*conversation.Message, threshold int) bool {
	return len(messages) > threshold
}

// compactMessages 压缩消息列表。
// 保留：
// 1. 第一条系统消息（如果有）
// 2. 最近的 N 条消息（基于阈值）
// 3. 添加压缩占位符
func compactMessages(messages []*conversation.Message, threshold int) []*conversation.Message {
	if len(messages) <= threshold {
		return messages
	}

	var result []*conversation.Message

	// 保留第一条系统消息
	var systemMessage *conversation.Message
	remainingMessages := messages

	for i, msg := range messages {
		if msg.Role == conversation.RoleSystem {
			systemMessage = msg
			remainingMessages = append(messages[:i], messages[i+1:]...)
			break
		}
	}

	if systemMessage != nil {
		result = append(result, systemMessage)
	}

	// 计算保留的消息数量
	// 保留最近的 threshold-1 条消息（减去系统消息）
	keepCount := threshold - 1
	if keepCount < 1 {
		keepCount = 1
	}

	startIndex := len(remainingMessages) - keepCount
	if startIndex < 0 {
		startIndex = 0
	}

	// 添加压缩占位符
	if startIndex > 0 {
		compactedCount := startIndex
		result = append(result, &conversation.Message{
			Role: conversation.RoleAssistant,
			Content: []conversation.MessageContent{
				{
					Type: conversation.MessageContentText,
					Text: fmt.Sprintf("[... %d 条消息被压缩以节省上下文 ...]", compactedCount),
				},
			},
		})
	}

	// 添加保留的消息
	result = append(result, remainingMessages[startIndex:]...)

	return result
}

// computeCompactionThreshold 计算压缩阈值。
func computeCompactionThreshold(contextLimit int, compactionRatio float64) int {
	// 默认使用 80% 的上下文限制作为压缩阈值
	if compactionRatio <= 0 {
		compactionRatio = 0.8
	}
	return int(float64(contextLimit) * compactionRatio)
}

// checkAndCompact 检查并压缩消息。
func (a *Agent) checkAndCompact(messages []*conversation.Message) []*conversation.Message {
	// 使用默认阈值进行压缩
	if needsCompaction(messages, DEFAULT_COMPACTION_THRESHOLD) {
		return compactMessages(messages, DEFAULT_COMPACTION_THRESHOLD)
	}
	return messages
}

const (
	// COMPACTION_THINKING_TEXT 压缩时的思考文本。
	COMPACTION_THINKING_TEXT = "goose is compacting the conversation..."
)
