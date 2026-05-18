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

// OpenAIEmbedder OpenAI 向量嵌入实现
// 使用 OpenAI Embedding API 生成文本向量
type OpenAIEmbedder struct {
	model     string // Embedding 模型名称
	apiKey    string // OpenAI API 密钥
	baseURL   string // API 基础地址
	dimension int    // 向量维度
	client    *http.Client
}

// NewOpenAIEmbedder 创建 OpenAI 向量嵌入器
//
// 参数:
//
//	apiKey: OpenAI API 密钥
//	model: Embedding 模型名称，如 "text-embedding-3-small"
//	baseURL: 可选的 API 基础地址，默认使用 OpenAI 官方地址
func NewOpenAIEmbedder(apiKey, model, baseURL string) *OpenAIEmbedder {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/embeddings"
	}

	// 根据模型名称自动推断向量维度
	dimension := 1536
	if strings.Contains(model, "3-large") {
		dimension = 3072
	}

	return &OpenAIEmbedder{
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
func (e *OpenAIEmbedder) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.GetEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("未获取到嵌入向量")
	}
	return embeddings[0], nil
}

// maxOpenAIInputLength 是 OpenAI embedding API 的最大输入长度限制 (token 数)
// text-embedding-ada-002 和 text-embedding-3 系列模型的上下文长度均为 8192 tokens
const maxOpenAIInputLength = 8192

// truncateOpenAIText 截断文本到指定长度（按字符数估算）
func truncateOpenAIText(text string, maxTokens int) string {
	// 保守估计：按字符数截断，保留约 maxTokens * 3/4 个字符以确保安全
	maxChars := maxTokens * 3 / 4
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
func (e *OpenAIEmbedder) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	type EmbeddingRequest struct {
		Input []string `json:"input"`
		Model string   `json:"model"`
	}

	type EmbeddingResponse struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
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
		processedTexts = append(processedTexts, truncateOpenAIText(trimmed, maxOpenAIInputLength))
	}

	// 如果所有文本都被过滤掉了，返回错误
	if len(processedTexts) == 0 {
		return nil, fmt.Errorf("没有有效的输入文本（文本为空或过长）")
	}

	// 构建请求
	reqBody := EmbeddingRequest{
		Input: processedTexts,
		Model: e.model,
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
func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}
