package setting

import "time"

// Config 统一配置结构
// 对应数据库表 config，存储所有配置信息
type Config struct {
	ID                     int64     `json:"id" db:"id"`                                               // 主键ID
	LLMProvider            string    `json:"llm_provider" db:"llm_provider"`                           // LLM模型提供商（openai/deepseek/qwen/doubao）
	LLMModel               string    `json:"llm_model" db:"llm_model"`                                 // LLM模型名称
	LLMAPISKey             string    `json:"llm_api_key" db:"llm_api_key"`                             // LLM API密钥
	LLMBaseURL             string    `json:"llm_base_url" db:"llm_base_url"`                           // LLM API基础URL
	LLMTemperature         float64   `json:"llm_temperature" db:"llm_temperature"`                     // LLM温度参数（0-2）
	LLMMaxTokens           int       `json:"llm_max_tokens" db:"llm_max_tokens"`                       // LLM最大输出token数
	EmbeddingProvider      string    `json:"embedding_provider" db:"embedding_provider"`               // Embedding模型提供商
	EmbeddingModel         string    `json:"embedding_model" db:"embedding_model"`                     // Embedding模型名称
	EmbeddingAPISKey       string    `json:"embedding_api_key" db:"embedding_api_key"`                 // Embedding API密钥
	EmbeddingBaseURL       string    `json:"embedding_base_url" db:"embedding_base_url"`               // Embedding API基础URL
	EmbeddingDimension     int       `json:"embedding_dimension" db:"embedding_dimension"`             // Embedding向量维度
	AgentName              string    `json:"agent_name" db:"agent_name"`                               // Agent名称
	AgentMaxLoop           int       `json:"agent_max_loop" db:"agent_max_loop"`                       // Agent最大循环次数
	AgentConvergeAfter     int       `json:"agent_converge_after" db:"agent_converge_after"`           // Agent连续无工具调用终止数
	AgentTimeout           int       `json:"agent_timeout" db:"agent_timeout"`                         // Agent超时时间（秒）
	AgentDailyTokenLimit   int       `json:"agent_daily_token_limit" db:"agent_daily_token_limit"`     // Agent每日Token限额（-1表示不限）
	AgentHistoryMaxEntries int       `json:"agent_history_max_entries" db:"agent_history_max_entries"` // Agent历史记录最大条目数
	AgentHistoryMaxDays    int       `json:"agent_history_max_days" db:"agent_history_max_days"`       // Agent历史记录保留天数
	KnowledgeEnabled       bool      `json:"knowledge_enabled" db:"knowledge_enabled"`                 // 知识库是否启用
	CreatedAt              time.Time `json:"created_at" db:"created_at"`                               // 创建时间
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`                               // 更新时间
}

// LLMConfigRequest LLM配置请求结构（用于API请求）
type LLMConfigRequest struct {
	Provider    string  `json:"provider" binding:"required"` // 模型提供商
	Model       string  `json:"model" binding:"required"`    // 模型名称
	APIKey      string  `json:"api_key"`                     // API密钥
	BaseURL     string  `json:"base_url"`                    // API基础URL
	Temperature float64 `json:"temperature"`                 // 温度参数
	MaxTokens   int     `json:"max_tokens"`                  // 最大输出token数
}

// EmbeddingConfigRequest Embedding配置请求结构（用于API请求）
type EmbeddingConfigRequest struct {
	Provider  string `json:"provider" binding:"required"` // 模型提供商
	Model     string `json:"model" binding:"required"`    // 模型名称
	APIKey    string `json:"api_key"`                     // API密钥
	BaseURL   string `json:"base_url"`                    // API基础URL
	Dimension int    `json:"dimension"`                   // 向量维度
}

// AgentConfigRequest Agent配置请求结构（用于API请求）
type AgentConfigRequest struct {
	Name              string `json:"name" binding:"required"` // Agent名称
	MaxLoop           int    `json:"max_loop"`                // 最大循环次数
	ConvergeAfter     int    `json:"converge_after"`          // 连续无工具调用终止数
	Timeout           int    `json:"timeout"`                 // 超时时间（秒）
	DailyTokenLimit   int    `json:"daily_token_limit"`       // 每日Token限额
	HistoryMaxEntries int    `json:"history_max_entries"`     // 历史记录最大条目数
	HistoryMaxDays    int    `json:"history_max_days"`        // 历史记录保留天数
}

// KnowledgeConfigRequest 知识库配置请求结构（用于API请求）
type KnowledgeConfigRequest struct {
	Enabled bool `json:"enabled"` // 知识库是否启用
}

// ConfigRequest 完整配置请求结构（用于API请求-全量更新）
type ConfigRequest struct {
	LLMProvider            string  `json:"llm_provider"`              // LLM提供商
	LLMModel               string  `json:"llm_model"`                 // LLM模型名称
	LLMAPISKey             string  `json:"llm_api_key"`               // LLM API密钥
	LLMBaseURL             string  `json:"llm_base_url"`              // LLM API基础URL
	LLMTemperature         float64 `json:"llm_temperature"`           // 温度参数
	LLMMaxTokens           int     `json:"llm_max_tokens"`            // 最大token数
	EmbeddingProvider      string  `json:"embedding_provider"`        // Embedding提供商
	EmbeddingModel         string  `json:"embedding_model"`           // Embedding模型名称
	EmbeddingAPISKey       string  `json:"embedding_api_key"`         // Embedding API密钥
	EmbeddingBaseURL       string  `json:"embedding_base_url"`        // Embedding API基础URL
	EmbeddingDimension     int     `json:"embedding_dimension"`       // 向量维度
	AgentName              string  `json:"agent_name"`                // Agent名称
	AgentMaxLoop           int     `json:"agent_max_loop"`            // 最大循环次数
	AgentConvergeAfter     int     `json:"agent_converge_after"`      // 收敛判定次数
	AgentTimeout           int     `json:"agent_timeout"`             // 超时时间（秒）
	AgentDailyTokenLimit   int     `json:"agent_daily_token_limit"`   // 每日Token限额
	AgentHistoryMaxEntries int     `json:"agent_history_max_entries"` // 历史记录最大条目数
	AgentHistoryMaxDays    int     `json:"agent_history_max_days"`    // 历史记录保留天数
	KnowledgeEnabled       bool    `json:"knowledge_enabled"`         // 知识库是否启用
}

// ConfigUpdateRequest 完整配置更新请求结构（用于API请求-部分更新）
// 所有字段均为指针类型，nil 表示该字段不更新
type ConfigUpdateRequest struct {
	LLMProvider            *string  `json:"llm_provider"`              // LLM提供商
	LLMModel               *string  `json:"llm_model"`                 // LLM模型名称
	LLMAPISKey             *string  `json:"llm_api_key"`               // LLM API密钥
	LLMBaseURL             *string  `json:"llm_base_url"`              // LLM API基础URL
	LLMTemperature         *float64 `json:"llm_temperature"`           // 温度参数
	LLMMaxTokens           *int     `json:"llm_max_tokens"`            // 最大token数
	EmbeddingProvider      *string  `json:"embedding_provider"`        // Embedding提供商
	EmbeddingModel         *string  `json:"embedding_model"`           // Embedding模型名称
	EmbeddingAPISKey       *string  `json:"embedding_api_key"`         // Embedding API密钥
	EmbeddingBaseURL       *string  `json:"embedding_base_url"`        // Embedding API基础URL
	EmbeddingDimension     *int     `json:"embedding_dimension"`       // 向量维度
	AgentName              *string  `json:"agent_name"`                // Agent名称
	AgentMaxLoop           *int     `json:"agent_max_loop"`            // 最大循环次数
	AgentConvergeAfter     *int     `json:"agent_converge_after"`      // 收敛判定次数
	AgentTimeout           *int     `json:"agent_timeout"`             // 超时时间（秒）
	AgentDailyTokenLimit   *int     `json:"agent_daily_token_limit"`   // 每日Token限额
	AgentHistoryMaxEntries *int     `json:"agent_history_max_entries"` // 历史记录最大条目数
	AgentHistoryMaxDays    *int     `json:"agent_history_max_days"`    // 历史记录保留天数
	KnowledgeEnabled       *bool    `json:"knowledge_enabled"`         // 知识库是否启用
}
