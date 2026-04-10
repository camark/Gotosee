// Package cli 提供 session 命令。
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/camark/Gotosee/internal/session"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "管理会话",
	Long:  "列出、查看、删除、导出、导入会话",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		limit, _ := cmd.Flags().GetInt("limit")
		return listSessions(jsonOutput, limit)
	},
}

var sessionRemoveCmd = &cobra.Command{
	Use:   "remove [session-id]",
	Short: "删除会话",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeSession(args[0])
	},
}

var sessionExportCmd = &cobra.Command{
	Use:   "export [session-id]",
	Short: "导出会话",
	Long:  "将会话导出为 JSON 文件",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("output")
		return exportSession(args[0], outputFile)
	},
}

var sessionImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "导入会话",
	Long:  "从 JSON 文件导入会话",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return importSession(args[0])
	},
}

func init() {
	sessionCmd.AddCommand(sessionRemoveCmd)
	sessionCmd.AddCommand(sessionExportCmd)
	sessionCmd.AddCommand(sessionImportCmd)
	sessionCmd.Flags().BoolP("json", "j", false, "以 JSON 格式输出")
	sessionCmd.Flags().IntP("limit", "n", 20, "限制显示的会话数量")

	sessionExportCmd.Flags().StringP("output", "o", "", "输出文件路径（默认：stdout）")
}

func listSessions(jsonOutput bool, limit int) error {
	sm, err := session.NewSessionManager(session.DefaultSessionManagerConfig())
	if err != nil {
		return fmt.Errorf("failed to create session manager: %w", err)
	}
	defer sm.Close()

	ctx := context.Background()
	sessions, err := sm.ListSessions(ctx, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(sessions, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(sessions) == 0 {
		fmt.Println("没有会话")
		return nil
	}

	fmt.Printf("最近 %d 个会话:\n", len(sessions))
	fmt.Println("----------------------------------------")
	for _, s := range sessions {
		status := "✓"
		if s.SessionType == session.SessionTypeScheduled {
			status = "⏰"
		}
		fmt.Printf("%s %-12s %s\n", status, s.ID[:8], s.Name)
		fmt.Printf("   目录：%s\n", s.WorkingDir)
		fmt.Printf("   时间：%s\n", s.UpdatedAt.Format(time.RFC3339))
		fmt.Println()
	}

	return nil
}

func removeSession(id string) error {
	sm, err := session.NewSessionManager(session.DefaultSessionManagerConfig())
	if err != nil {
		return fmt.Errorf("failed to create session manager: %w", err)
	}
	defer sm.Close()

	ctx := context.Background()
	if err := sm.DeleteSession(ctx, id); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	fmt.Printf("会话 %s 已删除\n", id)
	return nil
}

func exportSession(id, outputFile string) error {
	sm, err := session.NewSessionManager(session.DefaultSessionManagerConfig())
	if err != nil {
		return fmt.Errorf("failed to create session manager: %w", err)
	}
	defer sm.Close()

	ctx := context.Background()
	exp, err := sm.ExportSession(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to export session: %w", err)
	}

	data, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("会话已导出到：%s\n", outputFile)
	} else {
		fmt.Println(string(data))
	}

	return nil
}

func importSession(inputFile string) error {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var exp session.SessionExport
	if err := json.Unmarshal(data, &exp); err != nil {
		return fmt.Errorf("failed to parse export: %w", err)
	}

	sm, err := session.NewSessionManager(session.DefaultSessionManagerConfig())
	if err != nil {
		return fmt.Errorf("failed to create session manager: %w", err)
	}
	defer sm.Close()

	ctx := context.Background()
	if err := sm.ImportSession(ctx, &exp); err != nil {
		return fmt.Errorf("failed to import session: %w", err)
	}

	fmt.Printf("会话 %s 已导入\n", exp.Session.ID)
	return nil
}
