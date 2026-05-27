# Embedding 模块架构

## 概述

Embedding 模块负责文本向量化的处理，提供统一的接口抽象，支持灵活切换不同的 embedding 模型。

## 目录结构

```
internal/embedding/
├── embedder.go  # Embedder 接口定义
├── openai.go    # OpenAI embedding 实现
└── doubao.go    # 豆包 embedding 实现
```

## 核心接口

```go
// Embedder 向量化接口
type Embedder interface {
    Name() string
    Dimension() int
    Embed(ctx context.Context, text string) ([]float32, error)
    Embeds(ctx context.Context, texts []string) ([][]float32, error)
}
```

## 设计模式

### 1. 接口抽象
- 定义 `Embedder` 接口，统一不同 embedding 模型的行为
- 支持单文本(`Embed`)和批量文本(`Embeds`)向量化

### 2. 工厂模式
- `NewEmbedder()` 工厂函数根据 provider 参数创建对应的 embedding 实现

## 使用示例

```go
// 创建 embedding 器
emb, err := embedding.NewEmbedder("openai", apiKey, "text-embedding-3-small", "")

// 单文本向量化
vec, err := emb.Embed(ctx, "Hello world")

// 批量向量化
vecs, err := emb.Embeds(ctx, []string{"Hello", "World"})
```

## 扩展新的 Embedding 模型

只需新增一个实现文件，然后在工厂函数中添加 case 即可。

## 与 Model 模块的区别

| 模块 | 职责 |
|-----|------|
| model | 大语言模型交互(对话、推理) |
| embedding | 文本向量化(语义搜索、相似度计算) |
