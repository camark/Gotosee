package agents

import (
	"testing"

	"github.com/camark/Gotosee/internal/conversation"
)

func TestNeedsCompaction(t *testing.T) {
	messages := make([]*conversation.Message, 10)

	if needsCompaction(messages, 10) {
		t.Error("Should not need compaction when equal to threshold")
	}

	if needsCompaction(messages, 15) {
		t.Error("Should not need compaction when below threshold")
	}

	if !needsCompaction(messages, 5) {
		t.Error("Should need compaction when above threshold")
	}
}

func TestCompactMessages_Basic(t *testing.T) {
	messages := []*conversation.Message{
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg1"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply1"}}},
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg2"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply2"}}},
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg3"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply3"}}},
	}

	// 阈值设为 3，应该压缩
	compacted := compactMessages(messages, 3)

	// 应该保留系统消息（如果有）+ 压缩占位符 + 最近的消息
	if len(compacted) > 3 {
		t.Errorf("Expected at most 3 messages, got %d", len(compacted))
	}

	// 最后一条消息应该是原始的
	if len(compacted) > 0 {
		lastMsg := compacted[len(compacted)-1]
		if lastMsg.Role != conversation.RoleAssistant || lastMsg.Content[0].Text != "reply3" {
			t.Error("Last message should be preserved")
		}
	}
}

func TestCompactMessages_WithSystemMessage(t *testing.T) {
	messages := []*conversation.Message{
		{Role: conversation.RoleSystem, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "system"}}},
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg1"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply1"}}},
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg2"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply2"}}},
	}

	compacted := compactMessages(messages, 3)

	// 第一条应该是系统消息
	if len(compacted) == 0 || compacted[0].Role != conversation.RoleSystem {
		t.Error("First message should be system message")
	}
}

func TestCompactMessages_NoCompactionNeeded(t *testing.T) {
	messages := []*conversation.Message{
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg1"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply1"}}},
	}

	compacted := compactMessages(messages, 5)

	if len(compacted) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(compacted))
	}
}

func TestComputeCompactionThreshold(t *testing.T) {
	threshold := computeCompactionThreshold(1000, 0.8)
	if threshold != 800 {
		t.Errorf("Expected 800, got %d", threshold)
	}

	// 测试默认 ratio
	threshold = computeCompactionThreshold(1000, 0)
	if threshold != 800 {
		t.Errorf("Expected 800 with default ratio, got %d", threshold)
	}
}

func TestCompactMessages_PreservesOrder(t *testing.T) {
	messages := []*conversation.Message{
		{Role: conversation.RoleSystem, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "system"}}},
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg1"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply1"}}},
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg2"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply2"}}},
		{Role: conversation.RoleUser, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "msg3"}}},
		{Role: conversation.RoleAssistant, Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: "reply3"}}},
	}

	compacted := compactMessages(messages, 4)

	// 系统消息 + 压缩占位符 + 2 条最近消息 = 4 条或更多（取决于实现）
	if len(compacted) < 3 {
		t.Errorf("Expected at least 3 messages, got %d", len(compacted))
	}

	// 验证顺序：system -> compacted marker -> recent messages
	if compacted[0].Role != conversation.RoleSystem {
		t.Error("First should be system")
	}

	lastMsg := compacted[len(compacted)-1]
	if lastMsg.Content[0].Text != "reply3" {
		t.Error("Last should be reply3")
	}
}
