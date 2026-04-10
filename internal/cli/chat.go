// Package cli 提供 chat 命令。
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aaif-goose/gogo/internal/agents"
	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/model"
	"github.com/aaif-goose/gogo/internal/providers"
	"github.com/aaif-goose/gogo/internal/session"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "开始对话",
	Long:  "与 AI 代理进行交互式对话",
	RunE: runChat,
}

func init() {
	chatCmd.Flags().StringP("provider", "p", "", "使用指定的提供商")
	chatCmd.Flags().StringP("model", "m", "", "使用指定的模型")
	chatCmd.Flags().BoolP("stream", "s", true, "使用流式输出")
	chatCmd.Flags().BoolP("debug", "d", false, "启用调试模式")
	chatCmd.Flags().BoolP("agent", "a", false, "使用 Agent 模式（支持工具调用）")
}

func runChat(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := LoadConfig("")
	if err != nil {
		return fmt.Errorf("加载配置失败：%w", err)
	}

	// 命令行参数覆盖配置
	providerFlag, _ := cmd.Flags().GetString("provider")
	modelFlag, _ := cmd.Flags().GetString("model")
	streamMode, _ := cmd.Flags().GetBool("stream")
	useAgent, _ := cmd.Flags().GetBool("agent") // 使用 Agent 模式

	if providerFlag != "" {
		cfg.Provider = providerFlag
	}
	if modelFlag != "" {
		cfg.Model = modelFlag
	}

	// 创建提供商
	provider, err := providers.GetProvider(cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.Model)
	if err != nil {
		return fmt.Errorf("创建提供商失败：%w", err)
	}

	fmt.Printf("连接到 %s (%s)...\n", provider.Name(), cfg.Model)
	if useAgent {
		fmt.Println("🤖 Agent 模式已启用 - 支持工具调用")
	}
	fmt.Println("输入 'quit' 或 'exit' 退出，'clear' 清空对话，'tools' 查看可用工具")
	fmt.Println(strings.Repeat("-", 50))

	// 保存对话历史
	var messages []*conversation.Message

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	// 如果启用 Agent 模式，创建 Agent 实例
	var agent *agents.Agent
	if useAgent {
		sessionManager, _ := session.NewSessionManager(session.DefaultSessionManagerConfig())
		agentConfig := agents.NewAgentConfig(sessionManager, "auto", false, agents.GoosePlatformCLI)
		agent = agents.NewAgentWithConfig(agentConfig)
		agent.SetProvider(provider)
	}

	for {
		// 读取用户输入
		fmt.Print("\n👤 你：")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)

		// 处理命令
		switch strings.ToLower(input) {
		case "quit", "exit":
			fmt.Println("再见!")
			return nil
		case "clear":
			messages = nil
			fmt.Println("对话已清空")
			continue
		case "":
			continue
		case "tools":
			// 显示可用工具
			if useAgent && agent != nil {
				tools := agent.CollectTools()
				if len(tools) == 0 {
					fmt.Println("暂无可用工具")
				} else {
					fmt.Printf("可用工具 (%d):\n", len(tools))
					for _, tool := range tools {
						fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
					}
				}
			} else {
				fmt.Println("提示：使用 --agent 标志启用 Agent 模式以查看完整工具列表")
			}
			continue
		}

		// 添加用户消息
		userMsg := &conversation.Message{
			Role: conversation.RoleUser,
			Content: []conversation.MessageContent{
				{Type: conversation.MessageContentText, Text: input},
			},
		}
		messages = append(messages, userMsg)

		// 调用 AI
		fmt.Print("🤖 AI：")

		if useAgent && agent != nil {
			// 使用 Agent 模式（支持工具调用）
			eventChan, err := agent.Reply(ctx, messages)
			if err != nil {
				fmt.Printf("\n错误：%v\n", err)
				continue
			}

			for event := range eventChan {
				switch e := event.(type) {
				case *agents.MessageEvent:
					if e.Message != nil {
						for _, content := range e.Message.Content {
							if content.Type == conversation.MessageContentText {
								fmt.Print(content.Text)
							}
						}
					}
				case *agents.ToolCallEvent:
					fmt.Printf("\n🔧 调用工具：%s", e.Name)
				case *agents.ToolResultEvent:
					if e.Error != nil {
						fmt.Printf(" [错误：%v]", e.Error)
					} else {
						fmt.Printf(" [完成]")
					}
				}
			}
			fmt.Println()

			// 添加 AI 回复到历史（从事件通道获取最终消息）
			// 简化处理：将当前 messages 更新
		} else if streamMode {
			// 流式输出
			stream, err := provider.Stream(ctx, messagesToSlice(messages), providerModelConfig(cfg))
			if err != nil {
				fmt.Printf("\n错误：%v\n", err)
				continue
			}

			var fullResponse strings.Builder
			for chunk := range stream {
				if chunk.Err != nil {
					fmt.Printf("\n错误：%v\n", chunk.Err)
					break
				}
				if chunk.Done {
					break
				}
				if chunk.Text != "" {
					fmt.Print(chunk.Text)
					fullResponse.WriteString(chunk.Text)
				}
			}
			fmt.Println()

			// 添加 AI 回复到历史
			if fullResponse.Len() > 0 {
				aiMsg := &conversation.Message{
					Role:    conversation.RoleAssistant,
					Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: fullResponse.String()}},
				}
				messages = append(messages, aiMsg)
			}
		} else {
			// 非流式输出
			response, err := provider.Complete(ctx, messagesToSlice(messages), providerModelConfig(cfg))
			if err != nil {
				fmt.Printf("\n错误：%v\n", err)
				continue
			}

			// 提取文本内容
			var textContent string
			for _, c := range response.Content {
				if c.Type == conversation.MessageContentText {
					textContent += c.Text
				}
			}

			fmt.Println(textContent)
			fmt.Println()

			// 添加 AI 回复到历史
			if textContent != "" {
				aiMsg := &conversation.Message{
					Role:    conversation.RoleAssistant,
					Content: []conversation.MessageContent{{Type: conversation.MessageContentText, Text: textContent}},
				}
				messages = append(messages, aiMsg)
			}
		}
	}
}

// messagesToSlice 转换消息类型
func messagesToSlice(msgs []*conversation.Message) []conversation.Message {
	result := make([]conversation.Message, len(msgs))
	for i, msg := range msgs {
		result[i] = conversation.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
	}
	return result
}

func providerModelConfig(cfg *CLIConfig) model.ModelConfig {
	return model.ModelConfig{
		Provider:    cfg.Provider,
		Model:       cfg.Model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}
}
