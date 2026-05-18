package schema

import "time"

// Message 表示对话中的一条消息
// 对应 OpenAI ChatML 格式的消息结构
type Message struct {
	Role       string     `json:"role"`                   // 角色：user/assistant/system/tool
	Content    string     `json:"content"`                // 消息内容
	Name       string     `json:"name,omitempty"`         // 可选：消息发送者名称
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 可选：工具调用列表
	ToolCallID string     `json:"tool_call_id,omitempty"` // 可选：工具调用ID（用于tool类型消息）
	Timestamp  time.Time  `json:"timestamp,omitempty"`    // 时间戳
}

// ToolCall 表示模型调用的一个工具
type ToolCall struct {
	ID       string       `json:"id"`       // 工具调用ID
	Type     string       `json:"type"`     // 调用类型（固定为function）
	Function FunctionCall `json:"function"` // 函数调用详情
}

// FunctionCall 表示具体的函数调用
type FunctionCall struct {
	Name      string `json:"name"`      // 函数名称
	Arguments string `json:"arguments"` // 函数参数（JSON字符串）
}

// ToolDefinition 工具定义，用于告诉模型有哪些工具可用
type ToolDefinition struct {
	Type     string           `json:"type"`     // 工具类型（固定为function）
	Function FunctionProperty `json:"function"` // 函数属性
}

// FunctionProperty 函数的名称、描述和参数模式
type FunctionProperty struct {
	Name        string         `json:"name"`                 // 函数名称
	Description string         `json:"description"`          // 函数功能描述（会告诉模型何时使用）
	Parameters  map[string]any `json:"parameters,omitempty"` // JSON Schema 格式的参数定义
}

// UsageInfo Token使用量统计
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入消耗的token数
	CompletionTokens int `json:"completion_tokens"` // 输出消耗的token数
	TotalTokens      int `json:"total_tokens"`      // 总token数
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Model         string           `json:"model"`                    // 模型名称
	Messages      []Message        `json:"messages"`                 // 消息列表
	Temperature   float64          `json:"temperature,omitempty"`    // 温度参数（0-2），控制随机性
	MaxTokens     int              `json:"max_tokens,omitempty"`     // 最大输出token数
	Tools         []ToolDefinition `json:"tools,omitempty"`          // 可用工具列表
	Stream        bool             `json:"stream,omitempty"`         // 是否开启流式输出
	StreamOptions *StreamOptions   `json:"stream_options,omitempty"` // 流式输出选项
}

// StreamOptions 流式输出选项
type StreamOptions struct {
	IncludeUsage      bool `json:"include_usage,omitempty"`       // 是否在流结束时返回usage信息
	ChunkIncludeUsage bool `json:"chunk_include_usage,omitempty"` // 是否在每个chunk中返回usage信息
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Content      string         `json:"content"`              // 文本内容
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"` // 工具调用列表
	FinishReason string         `json:"finish_reason"`        // 完成原因（stop/tool_calls）
	Usage        UsageInfo      `json:"usage"`                // Token使用量
	Model        string         `json:"model"`                // 实际使用的模型
	Error        *ResponseError `json:"error,omitempty"`      // 错误信息
}

// ResponseError API错误详情
type ResponseError struct {
	Code    string `json:"code,omitempty"` // 错误码
	Message string `json:"message"`        // 错误信息
	Type    string `json:"type,omitempty"` // 错误类型
}

// StreamChunk 流式响应的数据块
type StreamChunk struct {
	ID      string    `json:"id"`              // 本次请求ID
	Object  string    `json:"object"`          // 对象类型（chat.completion.chunk）
	Created int64     `json:"created"`         // 创建时间戳
	Model   string    `json:"model"`           // 模型名称
	Choices []Choice  `json:"choices"`         // 响应选项
	Usage   UsageInfo `json:"usage,omitempty"` // Token使用量
}

// Choice 流式响应中的一个选项
type Choice struct {
	Index        int    `json:"index"`         // 选项索引
	Delta        Delta  `json:"delta"`         // 增量内容
	FinishReason string `json:"finish_reason"` // 完成原因
}

// Delta 流式响应中的增量内容
type Delta struct {
	Content          string     `json:"content,omitempty"`           // 增量文本
	ReasoningContent string     `json:"reasoning_content,omitempty"` // 思考过程（豆包等模型）
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`        // 增量工具调用
	Role             string     `json:"role,omitempty"`              // 角色
}

// ModelConfig 模型配置
type ModelConfig struct {
	Provider       string `json:"provider"`                  // 模型提供商：openai/deepseek/qwen/doubao
	Model          string `json:"model"`                     // 具体模型名称
	APIKey         string `json:"api_key"`                   // API密钥
	BaseURL        string `json:"base_url,omitempty"`        // API基础URL（可选）
	EmbeddingModel string `json:"embedding_model,omitempty"` // Embedding 模型名称
}

// AgentConfig Agent配置
type AgentConfig struct {
	Name              string `json:"name"`                // Agent名称
	MaxLoop           int    `json:"max_loop"`            // 最大循环次数（防止无限循环）
	ConvergeAfter     int    `json:"converge_after"`      // 连续N轮无工具调用则终止
	Timeout           int    `json:"timeout"`             // 超时时间（秒）
	MemoryDir         string `json:"memory_dir"`          // 长期记忆存储目录
	HistoryDir        string `json:"history_dir"`         // 历史记录存储目录
	HistoryMaxEntries int    `json:"history_max_entries"` // 历史记录最大条目数
	HistoryMaxDays    int    `json:"history_max_days"`    // 历史记录保留天数
	ProfileDir        string `json:"profile_dir"`         // 用户画像存储目录
	DailyTokenLimit   int    `json:"daily_token_limit"`   // 每日Token限额（-1:不限制但记录, 0:不记录不限制, >0:限制且记录）
}

// Config 完整配置结构
type Config struct {
	Agent AgentConfig `json:"agent"` // Agent配置
	Model ModelConfig `json:"model"` // 模型配置
}
