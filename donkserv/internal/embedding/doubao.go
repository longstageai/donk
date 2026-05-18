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

// DoubaoEmbedder 豆包向量嵌入实现
// 使用字节火山引擎 Embedding API 生成文本向量
type DoubaoEmbedder struct {
	model     string
	apiKey    string
	baseURL   string
	dimension int
	client    *http.Client
	isVision  bool
}

// NewDoubaoEmbedder 创建豆包向量嵌入器
//
// 参数:
//
//	apiKey: API 密钥
//	model: Embedding 模型名称，如 "doubao-embedding-vision-250328"
//	baseURL: API 基础地址
func NewDoubaoEmbedder(apiKey, model, baseURL string) *DoubaoEmbedder {
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal"
	}

	// 根据模型名称推断维度
	// doubao-embedding-vision-250328/250615: 默认 2048，支持 1024/2048
	// doubao-embedding-large-240915: 默认 2048，支持 512/1024/2048/4096
	// doubao-embedding-text-240715: 默认 2048，支持 512/1024/2048 (最高2560)
	// doubao-embedding-text-240515: 默认 2048，支持 512/1024
	dimension := 2048 // 默认维度

	return &DoubaoEmbedder{
		model:     model,
		apiKey:    apiKey,
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		dimension: dimension,
		client:    &http.Client{},
		isVision:  true,
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
func (e *DoubaoEmbedder) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := e.GetEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("未获取到嵌入向量")
	}
	return embeddings[0], nil
}

// maxDoubaoInputLength 是豆包 embedding API 的最大输入长度限制 (token 数)
// doubao-embedding-vision 模型支持 8192 tokens (标准版) 或 128k tokens (最新版)
// 使用 8192 作为安全默认值
const maxDoubaoInputLength = 8192

// truncateDoubaoText 截断文本到指定长度（按字符数估算）
func truncateDoubaoText(text string, maxTokens int) string {
	// 保守估计：按字符数截断，保留约 maxTokens * 3/4 个字符以确保安全
	// 假设平均每个 token 约 1.3 个字符
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
func (e *DoubaoEmbedder) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	type EmbeddingRequest struct {
		Model          string      `json:"model"`
		Input          interface{} `json:"input"`
		EncodingFormat string      `json:"encoding_format,omitempty"`
	}

	type EmbeddingResponse struct {
		Data interface{} `json:"data"`
	}

	endpoint := e.baseURL

	// 处理输入文本：截断超长文本，过滤空文本
	processedTexts := make([]string, 0, len(texts))
	for _, text := range texts {
		// 过滤空文本或纯空白文本
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		// 截断超长文本
		processedTexts = append(processedTexts, truncateDoubaoText(trimmed, maxDoubaoInputLength))
	}

	// 如果所有文本都被过滤掉了，返回错误
	if len(processedTexts) == 0 {
		return nil, fmt.Errorf("没有有效的输入文本（文本为空或过长）")
	}

	input := make([]map[string]interface{}, len(processedTexts))
	for i, text := range processedTexts {
		input[i] = map[string]interface{}{
			"type": "text",
			"text": text,
		}
	}

	reqBody := EmbeddingRequest{
		Model:          e.model,
		Input:          input,
		EncodingFormat: "float",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
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

	var embeddings [][]float64
	switch data := result.Data.(type) {
	case []interface{}:
		for _, item := range data {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			emb, ok := itemMap["embedding"].([]interface{})
			if !ok {
				continue
			}
			var vec []float64
			for _, v := range emb {
				vec = append(vec, v.(float64))
			}
			embeddings = append(embeddings, vec)
		}
	case map[string]interface{}:
		emb, ok := data["embedding"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("无法提取embedding字段")
		}
		var vec []float64
		for _, v := range emb {
			vec = append(vec, v.(float64))
		}
		embeddings = append(embeddings, vec)
	default:
		return nil, fmt.Errorf("未知data格式: %T", result.Data)
	}

	return embeddings, nil
}

// Dimension 返回向量维度
//
// 返回:
//
//	向量维度 (int)
func (e *DoubaoEmbedder) Dimension() int {
	return e.dimension
}
