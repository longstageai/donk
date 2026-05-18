package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// QwenEmbedder 通义千问向量嵌入实现
// 使用阿里云百炼 Embedding API 生成文本向量
// 支持 text-embedding-v3/v4 等模型
type QwenEmbedder struct {
	model     string // Embedding 模型名称
	apiKey    string // 阿里云百炼 API 密钥
	baseURL   string // API 基础地址
	dimension int    // 向量维度
	client    *http.Client
}

// NewQwenEmbedder 创建通义千问向量嵌入器
//
// 参数:
//
//	apiKey: 阿里云百炼 API 密钥
//	model: Embedding 模型名称，如 "text-embedding-v3"
//	baseURL: 可选的 API 基础地址，默认使用阿里云百炼 OpenAI 兼容端点
func NewQwenEmbedder(apiKey, model, baseURL string) *QwenEmbedder {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings"
	}

	// 根据模型名称自动推断向量维度
	// text-embedding-v4 支持: 2048、1536、1024(默认)、768、512、256、128、64
	// text-embedding-v3 支持: 1024(默认)、768、512、256、128、64
	// text-embedding-v2 固定: 1536
	// text-embedding-v1 固定: 1536
	dimension := 1024 // 默认维度
	if strings.Contains(model, "v4") {
		dimension = 1024
	} else if strings.Contains(model, "v3") {
		dimension = 1024
	} else if strings.Contains(model, "v2") || strings.Contains(model, "v1") {
		dimension = 1536
	}

	return &QwenEmbedder{
		model:     model,
		apiKey:    apiKey,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		dimension: dimension,
		client:    &http.Client{},
	}
}

// GetEmbedding 获取单个文本的向量嵌入
//
// 参数:
//
//	ctx: 上下文
//	text: 待嵌入的文本
//
// 返回:
//
//	向量嵌入 ([]float64)
func (e *QwenEmbedder) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.GetEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("未获取到嵌入向量")
	}
	return embeddings[0], nil
}

// maxInputLength 是阿里云百炼 embedding API 的最大输入长度限制 (token 数)
const maxInputLength = 8192

// truncateText 截断文本到指定长度（按字符数估算，中文按 1-2 token/字符）
func truncateText(text string, maxTokens int) string {
	// 简单估算：按字符数截断，保留约 maxTokens/2 个字符以确保安全
	// 实际 token 数取决于分词器，这里做保守估计
	maxChars := maxTokens * 3 / 4 // 保守估计，假设平均每个 token 约 1.3 个字符
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars]
}

// GetEmbeddings 批量获取文本的向量嵌入
//
// 参数:
//
//	ctx: 上下文
//	texts: 待嵌入的文本列表
//
// 返回:
//
//	向量嵌入列表 ([][]float64)
func (e *QwenEmbedder) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	type EmbeddingRequest struct {
		Model      string   `json:"model"`
		Input      []string `json:"input"`
		Dimensions int      `json:"dimensions,omitempty"`
	}

	type EmbeddingResponse struct {
		Object string `json:"object"`
		Data   []struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	// 处理输入文本：截断超长文本，过滤空文本
	processedTexts := make([]string, 0, len(texts))
	for _, text := range texts {
		// 过滤空文本或纯空白文本
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		// 截断超长文本
		processedTexts = append(processedTexts, truncateText(trimmed, maxInputLength))
	}

	// 如果所有文本都被过滤掉了，返回错误
	if len(processedTexts) == 0 {
		return nil, fmt.Errorf("没有有效的输入文本（文本为空或过长）")
	}

	// 构建请求
	reqBody := EmbeddingRequest{
		Model:      e.model,
		Input:      processedTexts,
		Dimensions: e.dimension,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result EmbeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取向量嵌入
	var embeddings [][]float64
	for _, item := range result.Data {
		embeddings = append(embeddings, item.Embedding)
	}

	return embeddings, nil
}

// Dimension 返回向量维度
//
// 返回:
//
//	向量维度 (int)
func (e *QwenEmbedder) Dimension() int {
	return e.dimension
}
