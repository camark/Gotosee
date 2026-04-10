// Package cli 提供 schedule 命令。
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "管理定时任务",
	Long:  "添加、列出、删除定时任务",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listSchedules()
	},
}

var scheduleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加定时任务",
	RunE: func(cmd *cobra.Command, args []string) error {
		return addSchedule()
	},
}

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "删除定时任务",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeSchedule(args[0])
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleAddCmd)
	scheduleCmd.AddCommand(scheduleRemoveCmd)
	scheduleCmd.Flags().BoolP("all", "a", false, "列出所有任务")
}

func listSchedules() error {
	fmt.Println("定时任务列表:")
	fmt.Println("  (暂无任务)")
	return nil
}

func addSchedule() error {
	fmt.Println("添加定时任务 (待实现)")
	return nil
}

func removeSchedule(id string) error {
	fmt.Printf("删除定时任务：%s (待实现)\n", id)
	return nil
}
