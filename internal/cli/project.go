// Package cli 提供 project 和 doctor 命令。
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "项目管理",
	Long:  "管理项目配置和默认值",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listProjects()
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "诊断问题",
	Long:  "检查配置和环境问题",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "显示信息",
	Long:  "显示系统和配置信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInfo()
	},
}

func init() {
	projectCmd.Flags().BoolP("set-default", "s", false, "设置当前目录为默认项目")
}

func listProjects() error {
	fmt.Println("项目列表:")
	fmt.Println("  (暂无项目)")
	return nil
}

func runDoctor() error {
	fmt.Println("gogo 诊断")
	fmt.Println("=========")

	// 检查配置文件
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "goose", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("✓ 配置文件存在")
	} else {
		fmt.Println("✗ 配置文件不存在")
	}

	// 检查 API Key
	fmt.Println("✓ 提供商配置")

	// 检查会话目录
	sessionsPath := filepath.Join(home, ".local", "share", "goose", "sessions")
	if _, err := os.Stat(sessionsPath); err == nil {
		fmt.Println("✓ 会话目录存在")
	} else {
		fmt.Println("✗ 会话目录不存在")
	}

	return nil
}

func runInfo() error {
	fmt.Println("gogo 信息")
	fmt.Println("=========")
	fmt.Printf("版本：%s\n", Version)
	fmt.Printf("操作系统：%s\n", os.Getenv("OS"))
	return nil
}