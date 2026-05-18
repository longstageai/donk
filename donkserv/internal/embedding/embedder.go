package embedding

import (
	"context"
	"errors"
)

// Embedder 向量嵌入器接口
// 用于将文本转换为向量表示，支持语义搜索
type Embedder interface {
	// GetEmbedding 获取单个文本的向量嵌入
	GetEmbedding(ctx context.Context, text string) ([]float64, error)
	// GetEmbeddings 批量获取多个文本的向量嵌入
	GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
	// Dimension 返回向量维度
	Dimension() int
}

func NewEmbedding(provider, apiKey, model, baseURL string) (Embedder, error) {
	switch provider {
	case "doubao":
		return NewDoubaoEmbedder(apiKey, model, baseURL), nil
	case "openai":
		return NewOpenAIEmbedder(apiKey, model, baseURL), nil
	case "qwen":
		return NewQwenEmbedder(apiKey, model, baseURL), nil
	default:
		return nil, errors.New("创建失败")
	}
}
