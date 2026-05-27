# Model 模块架构

## 概述

Model 模块负责与各大语言模型(LLM)提供商的交互，提供统一的接口抽象，支持灵活切换不同的模型实现。

## 目录结构

```
internal/model/
├── llm.go        # LLM 接口定义
├── adapter.go    # 适配器工厂函数 + StreamResponse
├── openai.go     # OpenAI 模型实现
├── qwen.go       # 通义千问模型实现
├── deepseek.go   # DeepSeek 模型实现
└── doubao.go     # 豆包模型实现
```

## 核心接口

```go
// LLM 对话模型接口
type LLM interface {
    Name() string
    Chat(ctx context.Context, req *schema.ChatRequest) (*schema.ChatResponse, error)
    StreamChat(ctx context.Context, req *schema.ChatRequest) (*StreamResponse, error)
}
```

## 设计模式

### 1. 接口抽象
- 定义 `LLM` 接口，统一不同模型的行为
- 支持同步对话(`Chat`)和流式对话(`StreamChat`)

### 2. 工厂模式
- `NewAdapter()` 工厂函数根据 provider 参数创建对应的模型适配器
- 支持的提供商: openai, qwen, deepseek, doubao

### 3. 类型别名
- `type Adapter = LLM` 保持向后兼容

## 使用示例

```go
// 创建模型适配器
adapter, err := model.NewAdapter("openai", apiKey, "gpt-4", "")

// 同步对话
resp, err := adapter.Chat(ctx, req)

// 流式对话
stream, err := adapter.StreamChat(ctx, req)
for chunk := range stream.Chunks {
    // 处理流式输出
}
```

## 扩展新的模型

只需新增一个实现文件:

```go
// internal/model/claude.go
type ClaudeAdapter struct { ... }

func NewClaudeAdapter(apiKey, model, baseURL string) *ClaudeAdapter { ... }
func (c *ClaudeAdapter) Name() string { ... }
func (c *ClaudeAdapter) Chat(...) (*ChatResponse, error) { ... }
func (c *ClaudeAdapter) StreamChat(...) (*StreamResponse, error) { ... }
```

然后在 `adapter.go` 的 `NewAdapter` 函数中添加 case 即可。
