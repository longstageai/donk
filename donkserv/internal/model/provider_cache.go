package model

import (
	"fmt"
	"sync"
)

// ProviderCache LLM Provider 缓存管理器
// 启动时预创建所有支持的 Provider Adapter，对话时通过 SetConfig 动态设置参数
type ProviderCache struct {
	adapters map[string]Adapter // provider名称 -> Adapter映射
	mu       sync.RWMutex       // 读写锁保护缓存
}

// NewProviderCache 创建 Provider 缓存管理器
// 启动时预创建所有支持的 Provider Adapter
func NewProviderCache() *ProviderCache {
	cache := &ProviderCache{
		adapters: make(map[string]Adapter),
	}

	// 预创建所有支持的 Provider（使用空配置，后续通过 SetConfig 设置）
	cache.adapters["openai"] = NewOpenAIAdapter("", "", "")
	cache.adapters["deepseek"] = NewDeepSeekAdapter("", "", "")
	cache.adapters["qwen"] = NewQwenAdapter("", "", "")
	cache.adapters["doubao"] = NewDoubaoAdapter("", "", "")

	return cache
}

// Get 获取指定名称的 Provider Adapter
//
// 参数:
//   - name: Provider名称（如"openai", "deepseek"等）
//
// 返回:
//   - Adapter: 模型适配器，如果不存在返回nil
//   - bool: 是否存在
func (c *ProviderCache) Get(name string) (Adapter, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	adapter, exists := c.adapters[name]
	return adapter, exists
}

// List 获取所有已缓存的 Provider 名称列表
//
// 返回:
//   - []string: Provider名称列表
func (c *ProviderCache) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.adapters))
	for name := range c.adapters {
		names = append(names, name)
	}
	return names
}

// SetConfig 为指定 Provider 设置配置
// 在每次对话前调用，动态更新 Provider 的参数
//
// 参数:
//   - provider: Provider名称
//   - model: 模型名称
//   - apiKey: API密钥
//   - baseURL: API基础地址
//
// 返回:
//   - error: 错误信息
func (c *ProviderCache) SetConfig(provider, model, apiKey, baseURL string) error {
	c.mu.RLock()
	adapter, exists := c.adapters[provider]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("不支持的Provider: %s", provider)
	}

	adapter.SetConfig(model, apiKey, baseURL)
	return nil
}
