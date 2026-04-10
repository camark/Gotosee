// Package cli 提供 recipe 命令。
package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camark/Gotosee/internal/providers"
	"github.com/camark/Gotosee/internal/recipe"
	"github.com/spf13/cobra"
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "管理配方",
	Long:  "列出、查看、验证、运行配方",
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		dir, _ := cmd.Flags().GetString("dir")
		return listRecipes(global, dir)
	},
}

var recipeValidateCmd = &cobra.Command{
	Use:   "validate [recipe-file]",
	Short: "验证配方",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return validateRecipe(args[0])
	},
}

var recipeExplainCmd = &cobra.Command{
	Use:   "explain [recipe-file]",
	Short: "解释配方",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return explainRecipe(args[0])
	},
}

var recipeRunCmd = &cobra.Command{
	Use:   "run [recipe-file]",
	Short: "运行配方",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		modelName, _ := cmd.Flags().GetString("model")
		settings, _ := cmd.Flags().GetStringArray("setting")
		return runRecipe(args[0], provider, modelName, settings)
	},
}

func init() {
	recipeCmd.AddCommand(recipeValidateCmd)
	recipeCmd.AddCommand(recipeExplainCmd)
	recipeCmd.AddCommand(recipeRunCmd)
	recipeCmd.Flags().BoolP("global", "g", false, "使用全局配方目录")
	recipeCmd.Flags().String("dir", "", "指定配方目录")
	recipeRunCmd.Flags().StringP("provider", "p", "", "使用指定的提供商")
	recipeRunCmd.Flags().StringP("model", "m", "", "使用指定的模型")
	recipeRunCmd.Flags().StringArrayP("setting", "s", nil, "设置键值对 (格式：key=value)")
}

func listRecipes(global bool, dir string) error {
	recipeDir := dir
	if recipeDir == "" {
		if global {
			// 全局配方目录
			home, _ := os.UserHomeDir()
			recipeDir = filepath.Join(home, ".config", "goose", "recipes")
		} else {
			// 当前目录
			recipeDir = "./recipes"
		}
	}

	files, err := os.ReadDir(recipeDir)
	if err != nil {
		return fmt.Errorf("无法读取配方目录：%w", err)
	}

	fmt.Printf("配方列表 (%s):\n", recipeDir)
	fmt.Println("----------------------------------------")
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" || filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml" {
			fmt.Printf("  - %s\n", file.Name())
		}
	}

	return nil
}

func validateRecipe(path string) error {
	r, err := recipe.Load(path)
	if err != nil {
		return fmt.Errorf("加载配方失败：%w", err)
	}

	if err := r.Validate(); err != nil {
		return fmt.Errorf("验证失败：%w", err)
	}

	fmt.Printf("验证配方：%s\n", path)
	fmt.Printf("名称：%s\n", r.Name)
	fmt.Printf("描述：%s\n", r.Description)
	fmt.Printf("版本：%s\n", r.Version)
	fmt.Printf("设置项：%d\n", len(r.Settings))
	fmt.Println("✓ 配方格式正确")
	return nil
}

func explainRecipe(path string) error {
	r, err := recipe.Load(path)
	if err != nil {
		return fmt.Errorf("加载配方失败：%w", err)
	}

	fmt.Printf("配方：%s\n", r.Name)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("描述：%s\n", r.Description)
	fmt.Printf("版本：%s\n", r.Version)
	if r.Author.Name != "" {
		fmt.Printf("作者：%s\n", r.Author.Name)
	}
	fmt.Println()
	fmt.Println("设置项:")
	if len(r.Settings) == 0 {
		fmt.Println("  (无)")
	} else {
		for _, s := range r.Settings {
			required := ""
			if s.Required {
				required = " (必需)"
			}
			fmt.Printf("  - %s: %s%s\n", s.Key, s.Description, required)
			if s.Default != nil {
				fmt.Printf("    默认值：%v\n", s.Default)
			}
		}
	}
	fmt.Println()
	fmt.Println("指令:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(r.Instructions)
	return nil
}

func runRecipe(path string, providerName, modelName string, settings []string) error {
	// 加载配置
	cfg, err := LoadConfig("")
	if err != nil {
		return fmt.Errorf("加载配置失败：%w", err)
	}

	// 命令行参数覆盖
	if providerName != "" {
		cfg.Provider = providerName
	}
	if modelName != "" {
		cfg.Model = modelName
	}

	// 加载配方
	r, err := recipe.Load(path)
	if err != nil {
		return fmt.Errorf("加载配方失败：%w", err)
	}

	// 验证配方
	if err := r.Validate(); err != nil {
		return fmt.Errorf("配方验证失败：%w", err)
	}

	// 创建提供商
	provider, err := providers.GetProvider(cfg.Provider, cfg.APIKey, cfg.BaseURL, cfg.Model)
	if err != nil {
		return fmt.Errorf("创建提供商失败：%w", err)
	}

	// 创建运行器
	runner := recipe.NewRunner(r, provider, providerModelConfig(cfg))

	// 解析设置参数
	settingMap := make(map[string]any)
	for _, s := range settings {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("设置格式错误：%s (应为 key=value)", s)
		}
		settingMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	// 检查必需的设置
	required := runner.GetRequiredSettings()
	if len(required) > 0 {
		// 检查命令行是否提供了所有必需设置
		missing := []string{}
		for _, key := range required {
			if _, ok := settingMap[key]; !ok {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			fmt.Printf("此配方需要以下设置：%v\n", missing)
			fmt.Println("请使用 -s key=value 参数提供")
			return nil
		}
	}

	// 应用设置
	for k, v := range settingMap {
		runner.SetSetting(k, v)
	}

	fmt.Printf("运行配方：%s\n", r.Name)
	fmt.Println("输入 'quit' 退出")
	fmt.Println(strings.Repeat("-", 50))

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n你：")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "quit" {
			fmt.Println("再见!")
			return nil
		}

		ctx := context.Background()
		response, err := runner.Run(ctx, input)
		if err != nil {
			fmt.Printf("错误：%v\n", err)
			continue
		}

		fmt.Printf("\n%s: %s\n", r.Name, response)
	}
}
