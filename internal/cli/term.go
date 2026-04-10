// Package cli 提供 term 命令。
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var termCmd = &cobra.Command{
	Use:   "term",
	Short: "终端命令",
	Long:  "运行、初始化、管理终端会话",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTerm()
	},
}

var termInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化终端",
	RunE: func(cmd *cobra.Command, args []string) error {
		return initTerm()
	},
}

var termRunCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "运行命令",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTermCommand(args[0])
	},
}

func init() {
	termCmd.AddCommand(termInitCmd)
	termCmd.AddCommand(termRunCmd)
	termCmd.Flags().StringP("shell", "s", "bash", "指定 shell 类型")
}

func runTerm() error {
	fmt.Println("启动终端会话 (待实现)")
	return nil
}

func initTerm() error {
	fmt.Println("初始化终端 (待实现)")
	return nil
}

func runTermCommand(cmd string) error {
	fmt.Printf("运行命令：%s (待实现)\n", cmd)
	return nil
}
