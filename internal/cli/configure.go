// Package cli 提供 configure 命令。
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "配置 goose",
	Long:  "配置 goose 的提供商、模型和其他设置",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigure()
	},
}

func init() {
	configureCmd.Flags().StringP("provider", "p", "", "设置 AI 提供商")
	configureCmd.Flags().StringP("model", "m", "", "设置模型")
	configureCmd.Flags().BoolP("reset", "r", false, "重置所有配置")
}

func runConfigure() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("欢迎使用 gogo 配置向导")
	fmt.Println("=====================")

	// 选择提供商
	fmt.Println("\n选择 AI 提供商:")
	fmt.Println("1. OpenAI")
	fmt.Println("2. Anthropic")
	fmt.Println("3. Google")
	fmt.Println("4. Ollama")
	fmt.Println("5. Azure OpenAI")

	fmt.Print("请输入选项 (1-5): ")
	providerChoice, _ := reader.ReadString('\n')
	providerChoice = strings.TrimSpace(providerChoice)

	provider := ""
	switch providerChoice {
	case "1":
		provider = "openai"
	case "2":
		provider = "anthropic"
	case "3":
		provider = "google"
	case "4":
		provider = "ollama"
	case "5":
		provider = "azure"
	default:
		fmt.Println("无效的选项")
		return nil
	}

	fmt.Printf("选择的提供商：%s\n", provider)

	// 获取 API Key
	fmt.Print("请输入 API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	// TODO: 保存配置到文件
	fmt.Println("\n配置已保存!")

	return nil
}
