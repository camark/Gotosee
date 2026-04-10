// Package cli 提供 project 和 doctor 命令。
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

var projectAddCmd = &cobra.Command{
	Use:   "add [project-path]",
	Short: "添加项目",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return addProject(args)
	},
}

var projectRemoveCmd = &cobra.Command{
	Use:   "remove [project-name]",
	Short: "删除项目",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeProject(args[0])
	},
}

var projectUseCmd = &cobra.Command{
	Use:   "use [project-name]",
	Short: "切换项目",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return useProject(args[0])
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
	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectUseCmd)
	projectCmd.Flags().BoolP("set-default", "s", false, "设置当前目录为默认项目")
}

// Project 项目配置。
type Project struct {
	// 项目名称
	Name string `json:"name"`
	// 项目路径
	Path string `json:"path"`
	// 使用的提供商
	Provider string `json:"provider,omitempty"`
	// 使用的模型
	Model string `json:"model,omitempty"`
	// 配方目录
	RecipeDir string `json:"recipe_dir,omitempty"`
	// 添加时间
	AddedAt time.Time `json:"added_at"`
	// 最后使用时间
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// getProjectFile 获取项目配置文件路径。
func getProjectFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".config", "gogo")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(dir, "projects.json"), nil
}

// loadProjects 加载所有项目。
func loadProjects() ([]Project, error) {
	file, err := getProjectFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// saveProjects 保存所有项目。
func saveProjects(projects []Project) error {
	file, err := getProjectFile()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0600)
}

// listProjects 列出所有项目。
func listProjects() error {
	projects, err := loadProjects()
	if err != nil {
		return fmt.Errorf("加载项目失败：%w", err)
	}

	if len(projects) == 0 {
		fmt.Println("暂无项目")
		fmt.Println("使用 'gogo project add' 添加项目")
		return nil
	}

	fmt.Println("项目列表:")
	fmt.Println(strings.Repeat("=", 60))

	for i, p := range projects {
		current := ""
		if i == 0 {
			current = " (当前)"
		}
		fmt.Printf("[%d]%s %s\n", i+1, current, p.Name)
		fmt.Printf("    路径：%s\n", p.Path)
		if p.Provider != "" {
			fmt.Printf("    提供商：%s\n", p.Provider)
		}
		if p.Model != "" {
			fmt.Printf("    模型：%s\n", p.Model)
		}
		if p.RecipeDir != "" {
			fmt.Printf("    配方目录：%s\n", p.RecipeDir)
		}
		if p.LastUsedAt != nil {
			fmt.Printf("    最后使用：%s\n", p.LastUsedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	return nil
}

// addProject 添加项目。
func addProject(args []string) error {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		// 使用当前目录
		var err error
		path, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("获取当前目录失败：%w", err)
		}
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("路径无效：%w", err)
	}

	// 检查目录是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("目录不存在：%s", absPath)
	}

	// 项目名称使用目录名
	name := filepath.Base(absPath)

	// 加载现有项目
	projects, err := loadProjects()
	if err != nil {
		return err
	}

	// 检查是否已存在
	for _, p := range projects {
		if p.Path == absPath {
			return fmt.Errorf("项目已存在：%s", p.Name)
		}
	}

	// 加载配置获取默认提供商和模型
	cfg, err := LoadConfig("")
	if err == nil {
		// 创建新项目
		now := time.Now()
		project := Project{
			Name:      name,
			Path:      absPath,
			Provider:  cfg.Provider,
			Model:     cfg.Model,
			AddedAt:   now,
			LastUsedAt: &now,
		}

		// 添加到列表开头
		projects = append([]Project{project}, projects...)
	} else {
		now := time.Now()
		project := Project{
			Name:      name,
			Path:      absPath,
			AddedAt:   now,
			LastUsedAt: &now,
		}
		projects = append([]Project{project}, projects...)
	}

	if err := saveProjects(projects); err != nil {
		return fmt.Errorf("保存项目失败：%w", err)
	}

	fmt.Printf("✓ 已添加项目：%s\n", name)
	fmt.Printf("  路径：%s\n", absPath)

	return nil
}

// removeProject 删除项目。
func removeProject(name string) error {
	projects, err := loadProjects()
	if err != nil {
		return err
	}

	found := -1
	for i, p := range projects {
		if p.Name == name {
			found = i
			break
		}
	}

	if found < 0 {
		return fmt.Errorf("未找到项目：%s", name)
	}

	projects = append(projects[:found], projects[found+1:]...)

	if err := saveProjects(projects); err != nil {
		return err
	}

	fmt.Printf("✓ 已删除项目：%s\n", name)
	return nil
}

// useProject 切换项目。
func useProject(name string) error {
	projects, err := loadProjects()
	if err != nil {
		return err
	}

	found := -1
	for i, p := range projects {
		if p.Name == name {
			found = i
			break
		}
	}

	if found < 0 {
		return fmt.Errorf("未找到项目：%s", name)
	}

	// 将选中的项目移到列表开头
	selected := projects[found]
	projects = append(projects[:found], projects[found+1:]...)
	now := time.Now()
	selected.LastUsedAt = &now
	projects = append([]Project{selected}, projects...)

	if err := saveProjects(projects); err != nil {
		return err
	}

	fmt.Printf("✓ 已切换到项目：%s\n", name)
	fmt.Printf("  路径：%s\n", selected.Path)

	return nil
}

// runDoctor 运行诊断。
func runDoctor() error {
	fmt.Println("gogo 诊断")
	fmt.Println(strings.Repeat("=", 50))

	allPassed := true

	// 1. 检查配置文件
	fmt.Println("\n[1/6] 配置文件检查...")
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "gogo", "config.json")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("  ✓ 配置文件存在")
	} else {
		fmt.Println("  ✗ 配置文件不存在")
		allPassed = false
	}

	// 2. 检查 API Key 配置
	fmt.Println("\n[2/6] API Key 检查...")
	cfg, err := LoadConfig("")
	if err == nil {
		if cfg.Provider == "" {
			fmt.Println("  ⚠ 未配置提供商")
			allPassed = false
		} else {
			fmt.Printf("  ✓ 已配置提供商：%s\n", cfg.Provider)
		}
		if cfg.APIKey == "" && cfg.Provider != "ollama" {
			fmt.Println("  ⚠ 未配置 API Key")
			allPassed = false
		} else if cfg.Provider != "ollama" {
			fmt.Println("  ✓ API Key 已配置")
		}
	} else {
		fmt.Println("  ✗ 无法加载配置")
		allPassed = false
	}

	// 3. 检查配置目录
	fmt.Println("\n[3/6] 配置目录检查...")
	configDir := filepath.Join(home, ".config", "gogo")
	if _, err := os.Stat(configDir); err == nil {
		fmt.Println("  ✓ 配置目录存在")
	} else {
		fmt.Println("  ✗ 配置目录不存在")
		if err := os.MkdirAll(configDir, 0700); err == nil {
			fmt.Println("  ✓ 已创建配置目录")
		} else {
			fmt.Println("  ✗ 无法创建配置目录")
			allPassed = false
		}
	}

	// 4. 检查会话目录
	fmt.Println("\n[4/6] 会话目录检查...")
	sessionsDir := filepath.Join(home, ".config", "gogo", "sessions")
	if _, err := os.Stat(sessionsDir); err == nil {
		fmt.Println("  ✓ 会话目录存在")
	} else {
		fmt.Println("  ⚠ 会话目录不存在 (可选)")
	}

	// 5. 检查配方目录
	fmt.Println("\n[5/6] 配方目录检查...")
	recipesDir := filepath.Join(home, ".config", "gogo", "recipes")
	if _, err := os.Stat(recipesDir); err == nil {
		fmt.Println("  ✓ 配方目录存在")
	} else {
		fmt.Println("  ⚠ 配方目录不存在 (可选)")
	}

	// 6. 检查系统信息
	fmt.Println("\n[6/6] 系统信息...")
	fmt.Printf("  操作系统：%s\n", runtime.GOOS)
	fmt.Printf("  架构：%s\n", runtime.GOARCH)
	fmt.Printf("  Go 版本：%s\n", runtime.Version())

	// 总结
	fmt.Println("\n" + strings.Repeat("=", 50))
	if allPassed {
		fmt.Println("✓ 所有检查通过")
	} else {
		fmt.Println("⚠ 部分检查未通过，请检查配置")
		fmt.Println("  使用 'gogo configure' 进行配置")
	}

	return nil
}

// runInfo 显示信息。
func runInfo() error {
	fmt.Println("gogo 信息")
	fmt.Println(strings.Repeat("=", 50))

	// 版本信息
	fmt.Printf("版本：%s\n", Version)
	fmt.Printf("Go 版本：%s\n", runtime.Version())
	fmt.Printf("操作系统：%s\n", runtime.GOOS)
	fmt.Printf("架构：%s\n", runtime.GOARCH)

	// 配置信息
	fmt.Println("\n配置信息:")
	cfg, err := LoadConfig("")
	if err == nil {
		fmt.Printf("  提供商：%s\n", cfg.Provider)
		fmt.Printf("  模型：%s\n", cfg.Model)
		if cfg.BaseURL != "" {
			fmt.Printf("  API 地址：%s\n", cfg.BaseURL)
		}
		if cfg.Deployment != "" {
			fmt.Printf("  部署名称：%s\n", cfg.Deployment)
		}
	} else {
		fmt.Println("  (未配置)")
	}

	// 目录信息
	home, _ := os.UserHomeDir()
	fmt.Println("\n目录信息:")
	fmt.Printf("  配置目录：%s\n", filepath.Join(home, ".config", "gogo"))
	fmt.Printf("  会话目录：%s\n", filepath.Join(home, ".config", "gogo", "sessions"))
	fmt.Printf("  配方目录：%s\n", filepath.Join(home, ".config", "gogo", "recipes"))

	// 项目信息
	fmt.Println("\n项目信息:")
	projects, err := loadProjects()
	if err == nil && len(projects) > 0 {
		fmt.Printf("  项目数量：%d\n", len(projects))
		if len(projects) > 0 {
			fmt.Printf("  当前项目：%s\n", projects[0].Name)
		}
	} else {
		fmt.Println("  (无项目)")
	}

	return nil
}
