// configs 应用配置模块
// 定义应用程序的配置文件结构，支持 YAML 格式解析
package configs

// ServerConfig 服务器配置
// 用于配置 HTTP 服务的监听地址和端口
type ServerConfig struct {
	Host string `yaml:"host"` // 服务监听地址
	Port int    `yaml:"port"` // 服务监听端口
}

// Model 模型配置
// 用于配置 AI 模型提供商的连接信息
type Model struct {
	Provider string `json:"provider" yaml:"provider"`                     // 模型提供商：openai/deepseek/qwen/doubao
	Model    string `json:"model" yaml:"model"`                           // 具体模型名称
	APIKey   string `json:"api_key" yaml:"api_key"`                       // API密钥
	BaseURL  string `json:"base_url,omitempty" yaml:"base_url,omitempty"` // API基础URL（可选，部分提供商需要）
}

// AgentConfig Agent配置
// 用于配置单Agent系统的运行参数
type AgentConfig struct {
	Name              string `json:"name" yaml:"name"`                               // Agent名称
	MaxLoop           int    `json:"max_loop" yaml:"max_loop"`                       // 最大循环次数（防止无限循环）
	ConvergeAfter     int    `json:"converge_after" yaml:"converge_after"`           // 连续N轮无工具调用则终止
	Timeout           int    `json:"timeout" yaml:"timeout"`                         // 超时时间（秒）
	HistoryMaxEntries int    `json:"history_max_entries" yaml:"history_max_entries"` // 历史记录最大条目数
	HistoryMaxDays    int    `json:"history_max_days" yaml:"history_max_days"`       // 历史记录保留天数
	DailyTokenLimit   int    `json:"daily_token_limit" yaml:"daily_token_limit"`     // 每日Token限额（-1:不限制但记录, 0:不记录不限制, >0:限制且记录）
}

// Conf 应用程序完整配置
// 包含服务器、AI模型、Agent的所有配置项
type Conf struct {
	Server    ServerConfig `yaml:"server"`    // 服务器配置
	Llm       Model        `yaml:"llm"`       // 大语言模型配置
	Embedding Model        `yaml:"embedding"` // 向量嵌入模型配置
	Agent     AgentConfig  `yaml:"agent"`     // 单Agent系统配置
}
