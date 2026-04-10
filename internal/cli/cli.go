// Package cli 提供 CLI 命令行接口。
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version 版本号。
var Version = "1.0.0"

// RootCmd 根命令。
var RootCmd = &cobra.Command{
	Use:   "gogo",
	Short: "gogo - AI 代理框架",
	Long: `gogo 是一个用 Go 语言编写的 AI 代理框架，
支持多种 AI 提供商、MCP 扩展和 ACP 协议。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 默认显示帮助
		return cmd.Help()
	},
}

// Execute 执行 CLI。
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	// 全局标志
	RootCmd.PersistentFlags().BoolP("debug", "d", false, "启用调试输出")
	RootCmd.PersistentFlags().BoolP("version", "v", false, "显示版本号")

	// 添加子命令
	RootCmd.AddCommand(
		configureCmd,
		sessionCmd,
		recipeCmd,
		scheduleCmd,
		termCmd,
		projectCmd,
		doctorCmd,
		infoCmd,
		versionCmd,
		chatCmd,
	)
}

// versionCmd 版本命令。
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本号",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gogo version %s\n", Version)
	},
}
