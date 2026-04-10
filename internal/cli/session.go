// Package cli 提供 session 命令。
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aaif-goose/gogo/internal/session"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "管理会话",
	Long:  "列出、查看、删除会话",
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

func init() {
	sessionCmd.AddCommand(sessionRemoveCmd)
	sessionCmd.Flags().BoolP("json", "j", false, "以 JSON 格式输出")
	sessionCmd.Flags().IntP("limit", "n", 20, "限制显示的会话数量")
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
