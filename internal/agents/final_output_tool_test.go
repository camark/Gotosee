package agents

import (
	"encoding/json"
	"testing"

	"github.com/aaif-goose/gogo/internal/conversation"
)

func TestFinalOutputTool_Execute(t *testing.T) {
	tool := NewFinalOutputTool(&Response{
		Name:        "test",
		Description: "test response",
	})

	args := map[string]interface{}{
		"final_output": "This is the final result",
		"success":      true,
		"metadata": map[string]interface{}{
			"key": "value",
		},
	}

	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ToolName != FINAL_OUTPUT_TOOL_NAME {
		t.Errorf("Expected tool name %s, got %s", FINAL_OUTPUT_TOOL_NAME, result.ToolName)
	}

	if !tool.IsExecuted() {
		t.Error("Tool should be marked as executed")
	}

	output := tool.GetOutput()
	if output == nil {
		t.Error("Output should not be nil")
	}
}

func TestFinalOutputTool_ExecuteWithMinimalArgs(t *testing.T) {
	tool := NewFinalOutputTool(&Response{})

	args := map[string]interface{}{
		"final_output": "Minimal result",
	}

	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Error != nil {
		t.Errorf("Unexpected error in result: %v", result.Error)
	}
}

func TestFinalOutputTool_ExecuteWithInvalidArgs(t *testing.T) {
	tool := NewFinalOutputTool(&Response{})

	// Test with invalid JSON (should still work since we marshal/unmarshal internally)
	args := map[string]interface{}{
		"final_output": 123, // wrong type but should be handled
	}

	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should still execute, just with different output
	if result.ToolName != FINAL_OUTPUT_TOOL_NAME {
		t.Error("Should still execute")
	}
}

func TestFinalOutputTool_ToolDefinition(t *testing.T) {
	tool := NewFinalOutputTool(&Response{})

	def := tool.ToolDefinition()

	if def.Name != FINAL_OUTPUT_TOOL_NAME {
		t.Errorf("Expected tool name %s, got %s", FINAL_OUTPUT_TOOL_NAME, def.Name)
	}

	if def.InputSchema == nil {
		t.Error("InputSchema should not be nil")
	}

	// 反序列化 schema 进行检查
	var schema map[string]interface{}
	if err := json.Unmarshal(def.InputSchema, &schema); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Check required fields
	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("required should be a slice")
	}

	found := false
	for _, r := range required {
		if r == "final_output" {
			found = true
			break
		}
	}
	if !found {
		t.Error("final_output should be required")
	}
}

func TestFinalOutputTool_SystemPrompt(t *testing.T) {
	tool := NewFinalOutputTool(&Response{})

	prompt := tool.SystemPrompt()

	if prompt == "" {
		t.Error("SystemPrompt should not be empty")
	}

	// Check for key phrases
	expectedPhrases := []string{
		"Final Output",
		FINAL_OUTPUT_TOOL_NAME,
		"final output",
		"completion",
	}

	for _, phrase := range expectedPhrases {
		contains := false
		for i := 0; i < len(prompt)-len(phrase); i++ {
			if prompt[i:i+len(phrase)] == phrase {
				contains = true
				break
			}
		}
		if !contains {
			t.Errorf("SystemPrompt should contain '%s'", phrase)
		}
	}
}

func TestHasFinalOutputTool_Found(t *testing.T) {
	messages := []*conversation.Message{
		{
			Role: "user",
			Content: []conversation.MessageContent{
				{Type: "text", Text: "Hello"},
			},
		},
		{
			Role: "assistant",
			Content: []conversation.MessageContent{
				{
					Type:     "tool_use",
					ToolName: FINAL_OUTPUT_TOOL_NAME,
				},
			},
		},
	}

	if !HasFinalOutputTool(messages) {
		t.Error("Should detect final output tool")
	}
}

func TestHasFinalOutputTool_NotFound(t *testing.T) {
	messages := []*conversation.Message{
		{
			Role: "user",
			Content: []conversation.MessageContent{
				{Type: "text", Text: "Hello"},
			},
		},
		{
			Role: "assistant",
			Content: []conversation.MessageContent{
				{Type: "text", Text: "Hi there"},
			},
		},
	}

	if HasFinalOutputTool(messages) {
		t.Error("Should not detect final output tool")
	}
}
