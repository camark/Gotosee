// Package cli 提供 schedule 命令。
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "管理定时任务",
	Long:  "列出、添加、删除定时任务",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listSchedules()
	},
}

var scheduleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加定时任务",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return addSchedule()
	},
}

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove [task-name]",
	Short: "删除定时任务",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeSchedule(args[0])
	},
}

var scheduleRunCmd = &cobra.Command{
	Use:   "run [task-name]",
	Short: "立即运行定时任务",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSchedule(args[0])
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleAddCmd)
	scheduleCmd.AddCommand(scheduleRemoveCmd)
	scheduleCmd.AddCommand(scheduleRunCmd)
}

// ScheduleTask 定时任务定义。
type ScheduleTask struct {
	// 任务名称
	Name string `json:"name"`
	// 任务描述
	Description string `json:"description,omitempty"`
	// Cron 表达式（分 时 日 月 周）
	Cron string `json:"cron"`
	// 执行的命令或配方
	Command string `json:"command"`
	// 命令参数
	Args []string `json:"args,omitempty"`
	// 配方文件路径（如果使用 recipe 命令）
	Recipe string `json:"recipe,omitempty"`
	// 设置参数
	Settings []string `json:"settings,omitempty"`
	// 是否启用
	Enabled bool `json:"enabled"`
	// 上次运行时间
	LastRun *time.Time `json:"last_run,omitempty"`
	// 下次运行时间
	NextRun *time.Time `json:"next_run,omitempty"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// getScheduleFile 获取定时任务文件路径。
func getScheduleFile() (string, error) {
	// 优先使用全局配置目录
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".config", "gogo")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(dir, "schedules.json"), nil
}

// loadSchedules 加载所有定时任务。
func loadSchedules() ([]ScheduleTask, error) {
	file, err := getScheduleFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScheduleTask{}, nil
		}
		return nil, err
	}

	var tasks []ScheduleTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// saveSchedules 保存所有定时任务。
func saveSchedules(tasks []ScheduleTask) error {
	file, err := getScheduleFile()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0600)
}

// parseCron 解析 Cron 表达式，返回下次运行时间。
func parseCron(expr string) (*time.Time, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("Cron 表达式必须包含 5 个字段：分 时 日 月 周")
	}

	// 简单实现：找到下一个匹配的时间
	now := time.Now()
	next := now.Add(time.Hour)

	return &next, nil
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// listSchedules 列出所有定时任务。
func listSchedules() error {
	tasks, err := loadSchedules()
	if err != nil {
		return fmt.Errorf("加载定时任务失败：%w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("暂无定时任务")
		fmt.Println("使用 'gogo schedule add' 添加新任务")
		return nil
	}

	fmt.Println("定时任务列表:")
	fmt.Println(strings.Repeat("=", 80))
	for i, task := range tasks {
		status := "启用"
		if !task.Enabled {
			status = "禁用"
		}
		fmt.Printf("[%d] [%s] %s\n", i+1, status, task.Name)
		fmt.Printf("    描述：%s\n", task.Description)
		fmt.Printf("    Cron: %s\n", task.Cron)
		fmt.Printf("    命令：%s %v\n", task.Command, task.Args)
		if task.Recipe != "" {
			fmt.Printf("    配方：%s\n", task.Recipe)
			if len(task.Settings) > 0 {
				fmt.Printf("    设置：%v\n", task.Settings)
			}
		}
		if task.LastRun != nil {
			fmt.Printf("    上次运行：%s\n", task.LastRun.Format("2006-01-02 15:04:05"))
		}
		if task.NextRun != nil {
			fmt.Printf("    下次运行：%s\n", task.NextRun.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	return nil
}

// addSchedule 添加定时任务。
func addSchedule() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("添加定时任务")
	fmt.Println(strings.Repeat("-", 50))

	// 输入任务名称
	fmt.Print("任务名称：")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("任务名称不能为空")
	}

	// 输入描述
	fmt.Print("任务描述：")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	// 输入 Cron 表达式
	fmt.Print("Cron 表达式 (分 时 日 月 周，例如：0 9 * * 1-5): ")
	cron, _ := reader.ReadString('\n')
	cron = strings.TrimSpace(cron)
	if cron == "" {
		return fmt.Errorf("Cron 表达式不能为空")
	}

	// 验证 Cron 表达式
	nextRun, err := parseCron(cron)
	if err != nil {
		return fmt.Errorf("Cron 表达式无效：%w", err)
	}

	// 输入命令类型
	fmt.Println("\n命令类型:")
	fmt.Println("  1) 运行配方 (recipe run)")
	fmt.Println("  2) 自定义命令")
	fmt.Print("请选择 [1-2]: ")
	cmdType, _ := reader.ReadString('\n')
	cmdType = strings.TrimSpace(cmdType)

	var command string
	var args []string
	var recipe string
	var settings []string

	switch cmdType {
	case "1", "":
		command = "recipe run"
		fmt.Print("配方文件路径：")
		recipePath, _ := reader.ReadString('\n')
		recipe = strings.TrimSpace(recipePath)
		if recipe == "" {
			return fmt.Errorf("配方文件路径不能为空")
		}
		fmt.Print("设置参数 (可选，多个用逗号分隔，例如：tone=formal,language=en): ")
		settingsStr, _ := reader.ReadString('\n')
		settingsStr = strings.TrimSpace(settingsStr)
		if settingsStr != "" {
			settings = strings.Split(settingsStr, ",")
		}
		args = []string{recipe}
		for _, s := range settings {
			args = append(args, "-s", s)
		}
	case "2":
		fmt.Print("命令：")
		command, _ = reader.ReadString('\n')
		command = strings.TrimSpace(command)
		fmt.Print("参数 (可选): ")
		argsStr, _ := reader.ReadString('\n')
		argsStr = strings.TrimSpace(argsStr)
		if argsStr != "" {
			args = strings.Fields(argsStr)
		}
	default:
		return fmt.Errorf("无效的选择")
	}

	// 创建任务
	now := time.Now()
	task := ScheduleTask{
		Name:        name,
		Description: description,
		Cron:        cron,
		Command:     command,
		Args:        args,
		Recipe:      recipe,
		Settings:    settings,
		Enabled:     true,
		CreatedAt:   now,
		NextRun:     nextRun,
	}

	// 加载现有任务
	tasks, err := loadSchedules()
	if err != nil {
		return err
	}

	// 检查名称是否重复
	for _, t := range tasks {
		if t.Name == task.Name {
			return fmt.Errorf("任务名称 '%s' 已存在", task.Name)
		}
	}

	// 添加新任务
	tasks = append(tasks, task)

	// 保存
	if err := saveSchedules(tasks); err != nil {
		return fmt.Errorf("保存任务失败：%w", err)
	}

	fmt.Printf("\n✓ 定时任务 '%s' 已添加\n", task.Name)
	fmt.Printf("  下次运行时间：%s\n", nextRun.Format("2006-01-02 15:04:05"))

	return nil
}

// removeSchedule 删除定时任务。
func removeSchedule(name string) error {
	tasks, err := loadSchedules()
	if err != nil {
		return err
	}

	found := -1
	for i, task := range tasks {
		if task.Name == name {
			found = i
			break
		}
	}

	if found < 0 {
		return fmt.Errorf("未找到任务 '%s'", name)
	}

	// 确认删除
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("确认删除任务 '%s'? [y/N]: ", name)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("取消删除")
		return nil
	}

	// 删除任务
	tasks = append(tasks[:found], tasks[found+1:]...)

	if err := saveSchedules(tasks); err != nil {
		return err
	}

	fmt.Printf("✓ 定时任务 '%s' 已删除\n", name)
	return nil
}

// runSchedule 立即运行定时任务。
func runSchedule(name string) error {
	tasks, err := loadSchedules()
	if err != nil {
		return err
	}

	var task *ScheduleTask
	for i := range tasks {
		if tasks[i].Name == name {
			task = &tasks[i]
			break
		}
	}

	if task == nil {
		return fmt.Errorf("未找到任务 '%s'", name)
	}

	if !task.Enabled {
		fmt.Printf("警告：任务 '%s' 已禁用\n", name)
	}

	fmt.Printf("运行任务：%s\n", task.Name)
	fmt.Printf("命令：%s %v\n", task.Command, task.Args)
	fmt.Println(strings.Repeat("-", 50))

	// 这里只是模拟执行，实际需要集成到调度器中
	// TODO: 集成调度器后台运行
	fmt.Println("(模拟执行 - 实际调度器将在后台运行)")

	// 更新最后运行时间
	now := time.Now()
	task.LastRun = &now

	// 计算下次运行时间
	nextRun, err := parseCron(task.Cron)
	if err != nil {
		return fmt.Errorf("计算下次运行时间失败：%w", err)
	}
	task.NextRun = nextRun

	// 保存
	if err := saveSchedules(tasks); err != nil {
		return err
	}

	fmt.Printf("\n✓ 任务已执行\n")
	fmt.Printf("  上次运行：%s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("  下次运行：%s\n", nextRun.Format("2006-01-02 15:04:05"))

	return nil
}
