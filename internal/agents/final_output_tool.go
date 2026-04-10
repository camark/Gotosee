// Package agents 提供最终输出工具功能。
package agents

import (
	"encoding/json"
	"fmt"

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/mcp"
)

// FinalOutputTool 最终输出工具。
// 用于结构化地完成代理任务并返回最终结果。
type FinalOutputTool struct {
	Response *Response
	Executed bool
	Output   interface{}
}

// Response 响应结构。
type Response struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NewFinalOutputTool 创建新的最终输出工具。
func NewFinalOutputTool(response *Response) *FinalOutputTool {
	return &FinalOutputTool{
		Response: response,
		Executed: false,
	}
}

// FinalOutputArgs 最终输出工具的参数。
type FinalOutputArgs struct {
	// FinalOutput 最终输出内容
	FinalOutput string `json:"final_output"`
	// Success 是否成功
	Success bool `json:"success,omitempty"`
	// Metadata 额外元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Execute 执行最终输出工具。
func (t *FinalOutputTool) Execute(args map[string]interface{}) (*ToolCallResult, error) {
	t.Executed = true

	// 解析参数
	var finalArgs FinalOutputArgs

	// 序列化参数
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return &ToolCallResult{
			ToolName: FINAL_OUTPUT_TOOL_NAME,
			Error:    fmt.Errorf("marshal arguments: %w", err),
		}, nil
	}

	// 反序列化
	if err := json.Unmarshal(argsJSON, &finalArgs); err != nil {
		return &ToolCallResult{
			ToolName: FINAL_OUTPUT_TOOL_NAME,
			Error:    fmt.Errorf("unmarshal arguments: %w", err),
		}, nil
	}

	// 存储输出
	t.Output = finalArgs

	// 返回结果
	return &ToolCallResult{
		ToolName: FINAL_OUTPUT_TOOL_NAME,
		Content: []*mcp.Content{
			{
				Type: "text",
				Text: fmt.Sprintf("Final output recorded: %s", finalArgs.FinalOutput),
			},
		},
		Error: nil,
	}, nil
}

// IsExecuted 检查工具是否已执行。
func (t *FinalOutputTool) IsExecuted() bool {
	return t.Executed
}

// GetOutput 获取输出。
func (t *FinalOutputTool) GetOutput() interface{} {
	return t.Output
}

// ToolDefinition 返回工具定义。
func (t *FinalOutputTool) ToolDefinition() *mcp.Tool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"final_output": map[string]interface{}{
				"type":        "string",
				"description": "The final output or result of your work",
			},
			"success": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the task was completed successfully",
			},
			"metadata": map[string]interface{}{
				"type":        "object",
				"description": "Additional metadata about the result",
			},
		},
		"required": []string{"final_output"},
	}

	schemaJSON, _ := json.Marshal(schema)

	return &mcp.Tool{
		Name:        FINAL_OUTPUT_TOOL_NAME,
		Description: "Use this tool to submit the final result of your work. Call this when you have completed the user's request.",
		InputSchema: schemaJSON,
	}
}

// SystemPrompt 返回系统提示片段。
func (t *FinalOutputTool) SystemPrompt() string {
	return fmt.Sprintf(`## Final Output

When you have completed the user's request, you MUST call the **%s** tool.
This tool allows you to:
1. Submit your final output or result
2. Indicate whether the task was successful
3. Add any relevant metadata

Example usage:
- After writing code: submit the code summary
- After analysis: submit the findings
- After research: submit the conclusions

IMPORTANT: Always call this tool at the end of your work to signal completion.`, FINAL_OUTPUT_TOOL_NAME)
}

const (
	// FINAL_OUTPUT_TOOL_NAME 最终输出工具名称。
	FINAL_OUTPUT_TOOL_NAME = "final_output"
)

// HasFinalOutputTool 检查消息中是否包含最终输出工具调用。
func HasFinalOutputTool(messages []*conversation.Message) bool {
	for _, msg := range messages {
		for _, content := range msg.Content {
			if content.Type == conversation.MessageContentToolUse && content.ToolName == FINAL_OUTPUT_TOOL_NAME {
				return true
			}
		}
	}
	return false
}
