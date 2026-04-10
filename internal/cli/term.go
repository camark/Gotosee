// Package cli 提供 term 命令。
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/camark/Gotosee/internal/conversation"
	"github.com/camark/Gotosee/internal/providers"
	"github.com/spf13/cobra"
)

var termCmd = &cobra.Command{
	Use:   "term",
	Short: "管理终端会话",
	Long:  "运行、初始化终端会话",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTerm()
	},
}

var termRunCmd = &cobra.Command{
	Use:   "run [session-id]",
	Short: "运行终端会话",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTermSession(args[0])
	},
}

var termInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化新会话",
	RunE: func(cmd *cobra.Command, args []string) error {
		return initTermSession()
	},
}

func init() {
	termCmd.AddCommand(termRunCmd)
	termCmd.AddCommand(termInitCmd)
	termCmd.Flags().StringP("provider", "p", "", "使用指定的提供商")
	termCmd.Flags().StringP("model", "m", "", "使用指定的模型")
}

// TermSession 终端会话定义。
type TermSession struct {
	// 会话 ID
	ID string `json:"id"`
	// 会话名称
	Name string `json:"name"`
	// 会话描述
	Description string `json:"description,omitempty"`
	// 使用的提供商
	Provider string `json:"provider"`
	// 使用的模型
	Model string `json:"model"`
	// 对话历史
	Messages []conversation.Message `json:"messages,omitempty"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
	// 最后更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// getTermDir 获取终端会话目录。
func getTermDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".config", "gogo", "terms")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	return dir, nil
}

// getTermSessionFile 获取会话文件路径。
func getTermSessionFile(id string) (string, error) {
	dir, err := getTermDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

// loadTermSession 加载终端会话。
func loadTermSession(id string) (*TermSession, error) {
	file, err := getTermSessionFile(id)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("会话 '%s' 不存在", id)
		}
		return nil, err
	}

	var session TermSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveTermSession 保存终端会话。
func saveTermSession(session *TermSession) error {
	file, err := getTermSessionFile(session.ID)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0600)
}

// runTerm 运行终端（列出可用会话）。
func runTerm() error {
	dir, err := getTermDir()
	if err != nil {
		return fmt.Errorf("获取终端目录失败：%w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("暂无终端会话")
			fmt.Println("使用 'gogo term init' 创建新会话")
			return nil
		}
		return err
	}

	if len(entries) == 0 {
		fmt.Println("暂无终端会话")
		fmt.Println("使用 'gogo term init' 创建新会话")
		return nil
	}

	fmt.Println("终端会话列表:")
	fmt.Println(strings.Repeat("=", 60))

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		session, err := loadTermSession(id)
		if err != nil {
			continue
		}

		fmt.Printf("ID: %s\n", session.ID)
		fmt.Printf("名称：%s\n", session.Name)
		fmt.Printf("提供商：%s (%s)\n", session.Provider, session.Model)
		fmt.Printf("最后更新：%s\n", session.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("消息数：%d\n", len(session.Messages))
		fmt.Println(strings.Repeat("-", 40))
	}

	fmt.Println("\n使用 'gogo term run <id>' 进入会话")

	return nil
}

// runTermSession 运行终端会话。
func runTermSession(id string) error {
	// 加载配置
	cfg, err := LoadConfig("")
	if err != nil {
		return fmt.Errorf("加载配置失败：%w", err)
	}

	// 加载会话
	session, err := loadTermSession(id)
	if err != nil {
		return err
	}

	// 命令行参数覆盖配置
	providerFlag, _ := termCmd.Flags().GetString("provider")
	modelFlag, _ := termCmd.Flags().GetString("model")

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

	fmt.Printf("进入终端会话：%s\n", session.Name)
	fmt.Printf("提供商：%s (%s)\n", provider.Name(), cfg.Model)
	fmt.Println("输入 'quit' 退出，'clear' 清空对话，'save' 保存")
	fmt.Println(strings.Repeat("-", 50))

	// 恢复对话历史
	messages := session.Messages
	if messages == nil {
		messages = []conversation.Message{}
	}

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("\n你：")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)

		// 处理命令
		switch strings.ToLower(input) {
		case "quit", "exit":
			fmt.Println("正在退出...")
			// 自动保存
			session.Messages = messages
			session.UpdatedAt = time.Now()
			if err := saveTermSession(session); err != nil {
				fmt.Printf("警告：保存会话失败：%v\n", err)
			} else {
				fmt.Println("会话已保存")
			}
			return nil
		case "clear":
			messages = []conversation.Message{}
			fmt.Println("对话已清空")
			continue
		case "save":
			session.Messages = messages
			session.UpdatedAt = time.Now()
			if err := saveTermSession(session); err != nil {
				fmt.Printf("保存失败：%v\n", err)
			} else {
				fmt.Println("会话已保存")
			}
			continue
		case "":
			continue
		}

		// 添加用户消息
		userMsg := conversation.NewTextMessage(conversation.RoleUser, input)
		messages = append(messages, userMsg)

		// 调用 AI
		fmt.Print("AI: ")

		response, err := provider.Complete(ctx, messages, providerModelConfig(cfg))
		if err != nil {
			fmt.Printf("\n错误：%v\n", err)
			// 移除失败的用户消息
			if len(messages) > 0 {
				messages = messages[:len(messages)-1]
			}
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

		// 添加 AI 回复
		if textContent != "" {
			aiMsg := conversation.NewTextMessage(conversation.RoleAssistant, textContent)
			messages = append(messages, aiMsg)
		}
	}
}

// initTermSession 初始化新会话。
func initTermSession() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("初始化新的终端会话")
	fmt.Println(strings.Repeat("-", 50))

	// 输入会话名称
	fmt.Print("会话名称：")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("会话名称不能为空")
	}

	// 输入描述
	fmt.Print("会话描述（可选）：")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	// 加载配置
	cfg, err := LoadConfig("")
	if err != nil {
		return fmt.Errorf("加载配置失败：%w", err)
	}

	// 确认配置
	fmt.Println("\n当前配置:")
	fmt.Printf("  提供商：%s\n", cfg.Provider)
	fmt.Printf("  模型：%s\n", cfg.Model)
	fmt.Print("\n使用当前配置？[Y/n]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm == "n" || confirm == "no" {
		fmt.Print("新的提供商：")
		newProvider, _ := reader.ReadString('\n')
		newProvider = strings.TrimSpace(newProvider)
		if newProvider != "" {
			cfg.Provider = newProvider
		}

		fmt.Print("新的模型：")
		newModel, _ := reader.ReadString('\n')
		newModel = strings.TrimSpace(newModel)
		if newModel != "" {
			cfg.Model = newModel
		}
	}

	// 生成会话 ID
	id := fmt.Sprintf("term-%d", time.Now().Unix())

	// 创建会话
	now := time.Now()
	session := &TermSession{
		ID:          id,
		Name:        name,
		Description: description,
		Provider:    cfg.Provider,
		Model:       cfg.Model,
		Messages:    []conversation.Message{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 保存会话
	if err := saveTermSession(session); err != nil {
		return fmt.Errorf("保存会话失败：%w", err)
	}

	fmt.Printf("\n✓ 终端会话 '%s' 已创建\n", name)
	fmt.Printf("  会话 ID: %s\n", id)
	fmt.Printf("  使用 'gogo term run %s' 进入会话\n", id)

	// 询问是否立即进入
	fmt.Print("\n是否立即进入会话？[Y/n]: ")
	enter, _ := reader.ReadString('\n')
	enter = strings.TrimSpace(strings.ToLower(enter))

	if enter == "" || enter == "y" || enter == "yes" {
		return runTermSession(id)
	}

	return nil
}
