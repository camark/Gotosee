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

	"github.com/aaif-goose/gogo/internal/conversation"
	"github.com/aaif-goose/gogo/internal/model"
	"github.com/aaif-goose/gogo/internal/providers"
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
	fmt.Println("输入 'quit' 或 'exit' 退出，'clear' 清空对话")
	fmt.Println(strings.Repeat("-", 50))

	// 保存对话历史
	var messages []conversation.Message

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

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
		}

		// 添加用户消息
		userMsg := conversation.NewTextMessage(conversation.RoleUser, input)
		messages = append(messages, userMsg)

		// 调用 AI
		fmt.Print("🤖 AI：")

		if streamMode {
			// 流式输出
			stream, err := provider.Stream(ctx, messages, providerModelConfig(cfg))
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
				aiMsg := conversation.NewTextMessage(conversation.RoleAssistant, fullResponse.String())
				messages = append(messages, aiMsg)
			}
		} else {
			// 非流式输出
			response, err := provider.Complete(ctx, messages, providerModelConfig(cfg))
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
				aiMsg := conversation.NewTextMessage(conversation.RoleAssistant, textContent)
				messages = append(messages, aiMsg)
			}
		}
	}
}

func providerModelConfig(cfg *CLIConfig) model.ModelConfig {
	return model.ModelConfig{
		Provider:    cfg.Provider,
		Model:       cfg.Model,
		Temperature: 0.7,
		MaxTokens:   4096,
	}
}
