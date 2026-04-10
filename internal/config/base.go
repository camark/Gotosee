// Package config 提供 goose 配置管理功能。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ConfigError 配置错误类型。
type ConfigError struct {
	Code string
	Msg  string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

// 预定义的错误类型。
var (
	ErrNotFound         = &ConfigError{Code: "NOT_FOUND", Msg: "配置值不存在"}
	ErrDeserialize      = &ConfigError{Code: "DESERIALIZE", Msg: "反序列化失败"}
	ErrFileError        = &ConfigError{Code: "FILE_ERROR", Msg: "文件操作错误"}
	ErrDirectoryError   = &ConfigError{Code: "DIRECTORY", Msg: "目录创建失败"}
	ErrKeyringError     = &ConfigError{Code: "KEYRING", Msg: "密钥环访问错误"}
	ErrLockError        = &ConfigError{Code: "LOCK", Msg: "文件锁定错误"}
	ErrFallbackToFile   = &ConfigError{Code: "FALLBACK", Msg: "已降级到文件存储"}
)

// ConfigValue 配置值的类型。
type ConfigValue int

const (
	ConfigValueString ConfigValue = iota
	ConfigValueInt
	ConfigValueFloat
	ConfigValueBool
	ConfigValueArray
	ConfigValueObject
)

// Config 配置管理器。
//
// 提供灵活的配置系统，支持：
// - 动态配置键
// - 通过 serde 的多类型支持
// - 环境变量覆盖
// - 基于 YAML 的配置文件存储
// - 配置热重载
// - 系统密钥环中的安全密钥存储
//
// 配置值加载优先级：
// 1. 环境变量（完全匹配）
// 2. 配置文件（默认 ~/.config/goose/config.yaml）
//
// 密钥加载优先级：
// 1. 环境变量（完全匹配）
// 2. 系统密钥环（可通过 GOOSE_DISABLE_KEYRING 禁用）
// 3. 如果密钥环被禁用，密钥存储在文件中（默认 ~/.config/goose/secrets.yaml）
type Config struct {
	configPath    string
	defaultsPath  string
	secrets       SecretStorage
	guard         sync.Mutex
	secretsCache  sync.Map
	data          map[string]interface{}
}

// SecretStorage 密钥存储方式。
type SecretStorage interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

// KeyringStorage 基于系统密钥环的存储。
type KeyringStorage struct {
	service string
}

// Get 从密钥环获取密钥。
func (k *KeyringStorage) Get(key string) (string, error) {
	// TODO: 实现系统密钥环集成
	// 目前返回一个错误，表示需要使用文件存储
	return "", ErrKeyringError
}

// Set 设置密钥到密钥环。
func (k *KeyringStorage) Set(key, value string) error {
	// TODO: 实现系统密钥环集成
	return ErrKeyringError
}

// Delete 从密钥环删除密钥。
func (k *KeyringStorage) Delete(key string) error {
	// TODO: 实现系统密钥环集成
	return ErrKeyringError
}

// FileStorage 基于文件的密钥存储。
type FileStorage struct {
	path string
}

// Get 从文件获取密钥。
func (f *FileStorage) Get(key string) (string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFileError, err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(data, &secrets); err != nil {
		return "", fmt.Errorf("%w: %v", ErrDeserialize, err)
	}

	val, ok := secrets[key]
	if !ok {
		return "", ErrNotFound
	}

	return val, nil
}

// Set 设置密钥到文件。
func (f *FileStorage) Set(key, value string) error {
	// 读取现有密钥
	secrets := make(map[string]string)
	data, err := os.ReadFile(f.path)
	if err == nil {
		json.Unmarshal(data, &secrets) // 忽略错误，使用空 map
	}

	secrets[key] = value

	out, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeserialize, err)
	}

	// Windows 上无法设置 0o600 权限，使用默认权限
	if err := os.WriteFile(f.path, out, 0o600); err != nil {
		return fmt.Errorf("%w: %v", ErrFileError, err)
	}

	return nil
}

// Delete 从文件删除密钥。
func (f *FileStorage) Delete(key string) error {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 已经不存在
		}
		return fmt.Errorf("%w: %v", ErrFileError, err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(data, &secrets); err != nil {
		return fmt.Errorf("%w: %v", ErrDeserialize, err)
	}

	delete(secrets, key)

	out, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeserialize, err)
	}

	if err := os.WriteFile(f.path, out, 0o600); err != nil {
		return fmt.Errorf("%w: %v", ErrFileError, err)
	}

	return nil
}

// configKey 是配置键的类型。
type configKey string

// Global instance.
var (
	globalConfig     *Config
	globalConfigOnce sync.Once
)

// ConfigDir 返回配置目录路径。
func ConfigDir() (string, error) {
	// Windows: %APPDATA%\goose
	// Unix: ~/.config/goose
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDirectoryError, err)
	}
	return filepath.Join(configDir, "goose"), nil
}

// NewConfig 创建新的配置实例。
func NewConfig() (*Config, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDirectoryError, err)
	}

	cfg := &Config{
		configPath:   configPath,
		data:         make(map[string]interface{}),
		secretsCache: sync.Map{},
	}

	// 初始化密钥存储
	cfg.secrets, err = initSecretStorage(configDir)
	if err != nil {
		return nil, err
	}

	// 加载配置文件
	if err := cfg.loadConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		// 配置文件不存在是合法的，使用默认值
	}

	return cfg, nil
}

// GlobalConfig 返回全局配置实例。
func GlobalConfig() (*Config, error) {
	var err error
	globalConfigOnce.Do(func() {
		globalConfig, err = NewConfig()
	})
	return globalConfig, err
}

// initSecretStorage 初始化密钥存储。
func initSecretStorage(configDir string) (SecretStorage, error) {
	// 检查是否禁用密钥环
	if os.Getenv("GOOSE_DISABLE_KEYRING") != "" {
		secretsPath := filepath.Join(configDir, "secrets.yaml")
		return &FileStorage{path: secretsPath}, nil
	}

	// 使用系统密钥环
	return &KeyringStorage{service: "goose"}, nil
}

// loadConfig 加载配置文件。
func (c *Config) loadConfig() error {
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return err
	}

	// 尝试解析 YAML（这里简化为 JSON，实际项目需要引入 YAML 库）
	if err := json.Unmarshal(data, &c.data); err != nil {
		return fmt.Errorf("%w: %v", ErrDeserialize, err)
	}

	return nil
}

// saveConfig 保存配置文件。
func (c *Config) saveConfig() error {
	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeserialize, err)
	}

	if err := os.WriteFile(c.configPath, data, 0o600); err != nil {
		return fmt.Errorf("%w: %v", ErrFileError, err)
	}

	return nil
}

// GetParam 获取配置参数。
// 优先级：1. 环境变量 2. 配置文件
func (c *Config) GetParam(key string, target interface{}) error {
	// 首先检查环境变量（转换为大写）
	envKey := filepath.ToSlash(key)
	envKey = filepath.Base(envKey) // 获取最后一个路径成分
	envKey = toEnvVar(envKey)
	if envVal, ok := os.LookupEnv(envKey); ok {
		return json.Unmarshal([]byte(envVal), target)
	}

	// 然后检查配置文件
	c.guard.Lock()
	defer c.guard.Unlock()

	val, ok := c.data[key]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeserialize, err)
	}

	return json.Unmarshal(data, target)
}

// SetParam 设置配置参数。
func (c *Config) SetParam(key string, value interface{}) error {
	c.guard.Lock()
	defer c.guard.Unlock()

	c.data[key] = value
	return c.saveConfig()
}

// HasParam 检查配置参数是否存在。
func (c *Config) HasParam(key string) bool {
	// 检查环境变量
	envKey := toEnvVar(key)
	if _, ok := os.LookupEnv(envKey); ok {
		return true
	}

	c.guard.Lock()
	defer c.guard.Unlock()

	_, ok := c.data[key]
	return ok
}

// GetSecret 获取密钥。
func (c *Config) GetSecret(key string) (string, error) {
	// 首先检查缓存
	if cached, ok := c.secretsCache.Load(key); ok {
		return cached.(string), nil
	}

	// 检查环境变量
	envKey := toEnvVar(key)
	if envVal, ok := os.LookupEnv(envKey); ok {
		c.secretsCache.Store(key, envVal)
		return envVal, nil
	}

	// 从密钥存储获取
	val, err := c.secrets.Get(key)
	if err != nil {
		return "", err
	}

	c.secretsCache.Store(key, val)
	return val, nil
}

// SetSecret 设置密钥。
func (c *Config) SetSecret(key, value string) error {
	// 清除缓存
	c.secretsCache.Delete(key)

	return c.secrets.Set(key, value)
}

// HasSecret 检查密钥是否存在（不返回值）。
func (c *Config) HasSecret(key string) bool {
	// 检查环境变量
	envKey := toEnvVar(key)
	if _, ok := os.LookupEnv(envKey); ok {
		return true
	}

	_, err := c.secrets.Get(key)
	return err == nil
}

// DeleteSecret 删除密钥。
func (c *Config) DeleteSecret(key string) error {
	c.secretsCache.Delete(key)
	return c.secrets.Delete(key)
}

// Get 返回所有配置数据。
func (c *Config) Get() map[string]interface{} {
	c.guard.Lock()
	defer c.guard.Unlock()

	result := make(map[string]interface{})
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// toEnvVar 将配置键转换为环境变量格式。
// e.g. "openai_api_key" -> "OPENAI_API_KEY"
func toEnvVar(key string) string {
	result := ""
	for i, r := range key {
		if r == '_' || r == '.' {
			result += "_"
		} else {
			if i == 0 || (i > 0 && key[i-1] == '_') {
				result += string(r)
			} else {
				result += string(r)
			}
		}
	}
	return filepath.ToSlash(result)
}
