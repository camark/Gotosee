// Package recipe 提供配方（Recipe）管理功能。
package recipe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Recipe 配方定义。
type Recipe struct {
	// 配方名称
	Name string `json:"name"`
	// 配方描述
	Description string `json:"description,omitempty"`
	// 配方版本
	Version string `json:"version,omitempty"`
	// 作者信息
	Author Author `json:"author,omitempty"`
	// 设置项
	Settings []Setting `json:"settings,omitempty"`
	// 响应配置
	Responses []Response `json:"responses,omitempty"`
	// 配方内容（指令）
	Instructions string `json:"instructions"`
}

// Author 作者信息。
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Setting 设置项定义。
type Setting struct {
	// 设置键
	Key string `json:"key"`
	// 设置类型
	Type string `json:"type"` // string, int, bool, etc.
	// 描述
	Description string `json:"description,omitempty"`
	// 默认值
	Default any `json:"default,omitempty"`
	// 是否必需
	Required bool `json:"required,omitempty"`
}

// Response 响应配置。
type Response struct {
	// 匹配模式
	Match string `json:"match"`
	// 响应内容
	Content string `json:"content"`
}

// Load 从文件加载配方。
func Load(path string) (*Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read recipe file: %w", err)
	}

	var recipe Recipe
	if err := json.Unmarshal(data, &recipe); err != nil {
		return nil, fmt.Errorf("failed to parse recipe: %w", err)
	}

	return &recipe, nil
}

// Save 保存配方到文件。
func (r *Recipe) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal recipe: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write recipe: %w", err)
	}

	return nil
}

// Validate 验证配方是否有效。
func (r *Recipe) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("recipe name is required")
	}
	if r.Instructions == "" {
		return fmt.Errorf("recipe instructions is required")
	}

	// 验证设置项
	for i, setting := range r.Settings {
		if setting.Key == "" {
			return fmt.Errorf("setting[%d] key is required", i)
		}
		if setting.Type == "" {
			return fmt.Errorf("setting[%d] type is required", i)
		}
	}

	return nil
}

// GetSettingDefault 获取设置的默认值。
func (r *Recipe) GetSettingDefault(key string) (any, bool) {
	for _, setting := range r.Settings {
		if setting.Key == key {
			return setting.Default, true
		}
	}
	return nil, false
}

// ListRecipes 列出目录中的所有配方。
func ListRecipes(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var recipes []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".json" || ext == ".yaml" || ext == ".yml" {
			recipes = append(recipes, entry.Name())
		}
	}

	return recipes, nil
}

// DefaultRecipe 返回默认配方。
func DefaultRecipe() *Recipe {
	return &Recipe{
		Name:        "default",
		Description: "Default recipe",
		Version:     "1.0.0",
		Instructions: "You are a helpful AI assistant. Help the user with their questions and tasks.",
	}
}
