// Package providers 提供 AI 模型提供商注册表功能。
package providers

import (
	"fmt"
	"sync"

	"github.com/aaif-goose/gogo/internal/model"
)

// Registry 提供商注册表。
type Registry struct {
	mu         sync.RWMutex
	providers  map[string]Provider
	factories  map[string]ProviderFactory
}

// ProviderFactory 提供商工厂函数。
type ProviderFactory func(config map[string]interface{}) (Provider, error)

var (
	globalRegistry     *Registry
	globalRegistryOnce sync.Once
)

// GlobalRegistry 返回全局提供商注册表。
func GlobalRegistry() *Registry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &Registry{
			providers: make(map[string]Provider),
			factories: make(map[string]ProviderFactory),
		}
	})
	return globalRegistry
}

// Register 注册提供商实例。
func (r *Registry) Register(provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := provider.Name()
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("provider %s already registered", id)
	}
	r.providers[id] = provider
	return nil
}

// RegisterFactory 注册提供商工厂。
func (r *Registry) RegisterFactory(id string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[id] = factory
}

// Get 获取提供商实例。
func (r *Registry) Get(id string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", id)
	}
	return provider, nil
}

// Create 通过工厂创建提供商实例。
func (r *Registry) Create(id string, config map[string]interface{}) (Provider, error) {
	r.mu.RLock()
	factory, ok := r.factories[id]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider factory %s not found", id)
	}

	return factory(config)
}

// List 列出所有已注册的提供商。
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// ListIDs 列出所有已注册的提供商 ID。
func (r *Registry) ListIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.providers))
	for id := range r.providers {
		result = append(result, id)
	}
	return result
}

// Unregister 注销提供商。
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, id)
}

// GetProvider 根据配置获取提供商实例。
func GetProvider(providerType, apiKey, baseURL, modelName string) (Provider, error) {
	switch providerType {
	case "openai":
		return NewOpenAIProvider(OpenAIConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
		}, model.ModelConfig{
			Provider: providerType,
			Model:    modelName,
		}), nil
	case "anthropic":
		return NewAnthropicProvider(apiKey, baseURL, &model.ModelConfig{
			Provider: providerType,
			Model:    modelName,
		}), nil
	case "ollama":
		return NewOllamaProvider(baseURL, &model.ModelConfig{
			Provider: providerType,
			Model:    modelName,
		}), nil
	case "google":
		return NewGoogleProvider(apiKey, baseURL, &model.ModelConfig{
			Provider: providerType,
			Model:    modelName,
		}), nil
	case "azure":
		return NewAzureProvider(apiKey, baseURL, modelName, &model.ModelConfig{
			Provider: providerType,
			Model:    modelName,
		}), nil
	case "openrouter":
		return NewOpenRouterProvider(apiKey, &model.ModelConfig{
			Provider: providerType,
			Model:    modelName,
		}), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}
