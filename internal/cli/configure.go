// Package cli 提供 configure 命令。
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// CLIConfig CLI 配置结构。
type CLIConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	APIKey     string `json:"api_key,omitempty"`
	BaseURL    string `json:"base_url,omitempty"`
	Deployment string `json:"deployment,omitempty"`
}

// DefaultCLIConfig 返回默认配置。
func DefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Provider: "openai",
		Model:    "gpt-4o",
	}
}

// DefaultConfigPath 返回默认配置文件路径。
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ".gogo.json"
	}
	return filepath.Join(home, ".config", "gogo", "config.json")
}

// LoadConfig 从文件加载配置。
func LoadConfig(path string) (*CLIConfig, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultCLIConfig(), nil
		}
		return nil, err
	}

	var config CLIConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// Save 保存配置到文件。
func (c *CLIConfig) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

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
	fmt.Println("6. OpenRouter (多模型聚合)")

	fmt.Print("请输入选项 (1-6): ")
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
	case "6":
		provider = "openrouter"
	default:
		fmt.Println("无效的选项")
		return nil
	}

	fmt.Printf("选择的提供商：%s\n", provider)

	// 获取模型
	var model string
	if provider == "ollama" {
		fmt.Print("请输入模型名称 (默认：llama3.1): ")
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			model = "llama3.1"
		}
	} else {
		fmt.Print("请输入模型名称 (默认：gpt-4o): ")
		model, _ = reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			model = "gpt-4o"
		}
	}

	// 获取 API Key (Ollama 除外)
	var apiKey string
	if provider != "ollama" {
		fmt.Print("请输入 API Key: ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
	}

	// 获取 BaseURL (可选)
	fmt.Print("请输入 API Base URL (可选，直接回车使用默认): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	// Azure 需要 deployment
	var deployment string
	if provider == "azure" {
		fmt.Print("请输入 Azure Deployment 名称：")
		deployment, _ = reader.ReadString('\n')
		deployment = strings.TrimSpace(deployment)
	}

	// 保存配置
	cfg := &CLIConfig{
		Provider:   provider,
		Model:      model,
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Deployment: deployment,
	}

	if err := cfg.Save(""); err != nil {
		return fmt.Errorf("保存配置失败：%w", err)
	}

	fmt.Println("\n配置已保存到:", DefaultConfigPath())
	fmt.Println("配置摘要:")
	fmt.Printf("  提供商：%s\n", provider)
	fmt.Printf("  模型：%s\n", model)
	if provider != "ollama" {
		fmt.Printf("  API Key: %s... (已隐藏)\n", maskString(apiKey))
	}

	return nil
}

func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
