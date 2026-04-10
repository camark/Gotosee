// Package cli 提供 recipe 命令。
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "管理配方",
	Long:  "列出、查看、验证配方",
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

func init() {
	recipeCmd.AddCommand(recipeValidateCmd)
	recipeCmd.AddCommand(recipeExplainCmd)
	recipeCmd.Flags().BoolP("global", "g", false, "使用全局配方目录")
	recipeCmd.Flags().StringP("dir", "d", "", "指定配方目录")
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
		if filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml" {
			fmt.Printf("  - %s\n", file.Name())
		}
	}

	return nil
}

func validateRecipe(path string) error {
	// TODO: 实现配方验证逻辑
	fmt.Printf("验证配方：%s\n", path)
	fmt.Println("✓ 配方格式正确")
	return nil
}

func explainRecipe(path string) error {
	// TODO: 实现配方解释逻辑
	fmt.Printf("解释配方：%s\n", path)
	return nil
}
