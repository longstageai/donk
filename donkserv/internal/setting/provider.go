package setting

import (
	"sync"
)

// 全局变量声明
var (
	globalProvider *ConfigProvider // 全局配置提供者实例
	once           sync.Once       // 保证初始化只执行一次
)

// ConfigProvider 全局配置提供者
// 负责从数据库读取配置，提供给其他模块使用
// 使用单例模式，确保整个应用只有一个实例
type ConfigProvider struct {
	storage *Storage     // 存储层实例
	mu      sync.RWMutex // 读写锁，保证并发安全
}

// InitProvider 初始化全局配置提供者
// 参数:
//   - storage: 存储层实例
//
// 返回:
//   - *ConfigProvider: 配置提供者实例
//
// 注意: 此函数只会执行一次，多次调用不会重复初始化
func InitProvider(storage *Storage) *ConfigProvider {
	once.Do(func() {
		globalProvider = &ConfigProvider{
			storage: storage,
		}
	})
	return globalProvider
}

// GetProvider 获取全局配置提供者实例
// 返回:
//   - *ConfigProvider: 配置提供者实例
//
// 注意: 必须先调用 InitProvider 初始化后才能获取有效实例
func GetProvider() *ConfigProvider {
	return globalProvider
}

// GetConfig 获取完整配置
// 返回:
//   - *Config: 完整配置结构
//   - error: 错误信息
func (p *ConfigProvider) GetConfig() (*Config, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.storage.GetConfig()
}

// GetLLMConfig 获取 LLM 配置
// 返回:
//   - *LLMConfigRequest: LLM 配置结构
//   - error: 错误信息
func (p *ConfigProvider) GetLLMConfig() (*LLMConfigRequest, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.storage.GetLLMConfig()
}

// GetEmbeddingConfig 获取 Embedding 配置
// 返回:
//   - *EmbeddingConfigRequest: Embedding 配置结构
//   - error: 错误信息
func (p *ConfigProvider) GetEmbeddingConfig() (*EmbeddingConfigRequest, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.storage.GetEmbeddingConfig()
}

// GetAgentConfig 获取 Agent 配置
// 返回:
//   - *AgentConfigRequest: Agent 配置结构
//   - error: 错误信息
func (p *ConfigProvider) GetAgentConfig() (*AgentConfigRequest, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.storage.GetAgentConfig()
}

// LLMProvider 获取 LLM 配置参数
// 返回:
//   - string: 提供商名称
//   - string: 模型名称
//   - string: API 密钥
//   - string: API 基础 URL
//   - error: 错误信息
func (p *ConfigProvider) LLMProvider() (string, string, string, string, error) {
	cfg, err := p.GetLLMConfig()
	if err != nil || cfg == nil {
		return "", "", "", "", err
	}
	return cfg.Provider, cfg.Model, cfg.APIKey, cfg.BaseURL, nil
}

// EmbeddingProvider 获取 Embedding 配置参数
// 返回:
//   - string: 提供商名称
//   - string: 模型名称
//   - string: API 密钥
//   - string: API 基础 URL
//   - int: 向量维度
//   - error: 错误信息
func (p *ConfigProvider) EmbeddingProvider() (string, string, string, string, int, error) {
	cfg, err := p.GetEmbeddingConfig()
	if err != nil || cfg == nil {
		return "", "", "", "", 0, err
	}
	return cfg.Provider, cfg.Model, cfg.APIKey, cfg.BaseURL, cfg.Dimension, nil
}

// AgentConfig 获取 Agent 配置参数
// 返回:
//   - string: Agent 名称
//   - int: 最大循环次数
//   - int: 收敛判定次数
//   - int: 超时时间（秒）
//   - int: 每日 Token 限额
//   - error: 错误信息
func (p *ConfigProvider) AgentConfig() (string, int, int, int, int, error) {
	cfg, err := p.GetAgentConfig()
	if err != nil || cfg == nil {
		return "", 0, 0, 0, 0, err
	}
	return cfg.Name, cfg.MaxLoop, cfg.ConvergeAfter, cfg.Timeout, cfg.DailyTokenLimit, nil
}
