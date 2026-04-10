// Package conversation 提供对话管理功能。
package conversation

import (
	"encoding/json"
	"time"
)

// MessageContent 消息内容的类型。
type MessageContentType string

const (
	MessageContentText       MessageContentType = "text"
	MessageContentImage      MessageContentType = "image"
	MessageContentToolUse    MessageContentType = "tool_use"
	MessageContentToolResult MessageContentType = "tool_result"
	MessageContentActionRequired MessageContentType = "action_required"
)

// MessageContent 单条消息内容。
type MessageContent struct {
	Type       MessageContentType `json:"type"`
	Text       string             `json:"text,omitempty"`
	ImageURL   string             `json:"image_url,omitempty"`
	ImageData  []byte             `json:"image_data,omitempty"`
	ToolName   string             `json:"tool_name,omitempty"`
	ToolArgs   json.RawMessage    `json:"tool_args,omitempty"`
	ToolResult string             `json:"tool_result,omitempty"`
	// ActionRequired 用于需要用户确认的操作
	ActionRequired *ActionRequiredData `json:"action_required,omitempty"`
}

// ActionRequiredData 行动请求数据。
type ActionRequiredData struct {
	Type      string                 `json:"type"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MessageRole 消息角色。
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

// Message 对话消息。
type Message struct {
	// 消息 ID
	ID string `json:"id"`
	// 角色
	Role MessageRole `json:"role"`
	// 消息内容列表
	Content []MessageContent `json:"content"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 关联的工具调用 ID
	ToolCallID string `json:"tool_call_id,omitempty"`
	// 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewMessage 创建新消息。
func NewMessage(role MessageRole, content []MessageContent) Message {
	return Message{
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewTextMessage 创建文本消息。
func NewTextMessage(role MessageRole, text string) Message {
	return Message{
		Role: role,
		Content: []MessageContent{
			{Type: MessageContentText, Text: text},
		},
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// Conversation 对话记录。
type Conversation struct {
	// 会话 ID
	SessionID string `json:"session_id"`
	// 标题
	Title string `json:"title,omitempty"`
	// 消息列表
	Messages []Message `json:"messages"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
	// 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewConversation 创建新对话。
func NewConversation(sessionID string) *Conversation {
	now := time.Now()
	return &Conversation{
		SessionID: sessionID,
		Messages:  make([]Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}
}

// AddMessage 添加消息。
func (c *Conversation) AddMessage(msg Message) {
	c.Messages = append(c.Messages, msg)
	c.UpdatedAt = time.Now()
}

// LastMessage 返回最后一条消息。
func (c *Conversation) LastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return &c.Messages[len(c.Messages)-1]
}

// MessageCount 返回消息数量。
func (c *Conversation) MessageCount() int {
	return len(c.Messages)
}

// Clear 清空对话。
func (c *Conversation) Clear() {
	c.Messages = make([]Message, 0)
	c.UpdatedAt = time.Now()
}

// ToJSON 将对话转换为 JSON。
func (c *Conversation) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// FromJSON 从 JSON 加载对话。
func FromJSON(data []byte) (*Conversation, error) {
	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, err
	}
	return &conv, nil
}
