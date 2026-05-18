package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/pkg/schema"
)

// DoubaoAdapter 豆包模型适配器
// 实现 Adapter 接口，支持流式和非流式调用
type DoubaoAdapter struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewDoubaoAdapter 创建豆包模型适配器
func NewDoubaoAdapter(apiKey, model, baseURL string) *DoubaoAdapter {
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	}
	return &DoubaoAdapter{
		model:   model,
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

// Name 返回适配器名称
func (d *DoubaoAdapter) Name() string {
	return "doubao"
}

// Chat 非流式聊天
func (d *DoubaoAdapter) Chat(ctx context.Context, req *schema.ChatRequest) (*schema.ChatResponse, error) {
	req.Model = d.model

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", d.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.client.Do(httpReq)
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

	var doubaoResp doubaoResponse
	if err := json.Unmarshal(body, &doubaoResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if doubaoResp.Error != nil {
		return &schema.ChatResponse{
			Error: &schema.ResponseError{
				Code:    doubaoResp.Error.Code,
				Message: doubaoResp.Error.Message,
				Type:    doubaoResp.Error.Type,
			},
		}, nil
	}

	if len(doubaoResp.Choices) == 0 {
		return nil, fmt.Errorf("响应为空")
	}

	return d.buildChatResponse(doubaoResp), nil
}

// StreamChat 流式聊天
func (d *DoubaoAdapter) StreamChat(ctx context.Context, req *schema.ChatRequest) (*StreamResponse, error) {
	req.Model = d.model
	req.Stream = true
	req.StreamOptions = &schema.StreamOptions{
		ChunkIncludeUsage: true,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", d.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API 返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	streamResp := &StreamResponse{
		Chunks: make(chan *schema.StreamChunk, 100),
		Done:   make(chan struct{}),
	}

	go d.processStream(resp.Body, streamResp, ctx)

	return streamResp, nil
}

// processStream 处理流式响应
func (d *DoubaoAdapter) processStream(body io.Reader, streamResp *StreamResponse, ctx context.Context) {
	var lastUsage schema.UsageInfo
	toolCallsAccumulator := make(map[int]schema.ToolCall)

	defer close(streamResp.Done)
	defer streamResp.Close()
	defer func() {
		streamResp.Usage = lastUsage
	}()

	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			streamResp.Error = fmt.Errorf("读取流失败: %w", err)
			return
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		chunk := d.parseChunk(data)
		if chunk == nil {
			continue
		}

		streamChunk := d.convertToStreamChunk(chunk)
		lastUsage = streamChunk.Usage

		d.processToolCallsAccumulator(chunk, toolCallsAccumulator)
		d.completeToolCallsIfNeeded(chunk, toolCallsAccumulator, streamChunk)

		select {
		case streamResp.Chunks <- streamChunk:
		case <-ctx.Done():
			return
		}
	}
}

// doubaoResponse 豆包API响应结构
type doubaoResponse struct {
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Role             string `json:"role"`
			ToolCalls        []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
	} `json:"error,omitempty"`
}

// doubaoChunk 豆包流式chunk结构
type doubaoChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Delta        struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Role             string `json:"role"`
			ToolCalls        []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Index    int    `json:"index"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

// buildChatResponse 构建聊天响应
func (d *DoubaoAdapter) buildChatResponse(resp doubaoResponse) *schema.ChatResponse {
	result := &schema.ChatResponse{
		Model:        d.model,
		FinishReason: resp.Choices[0].FinishReason,
		Usage: schema.UsageInfo{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	msg := resp.Choices[0].Message
	if msg.Content != "" {
		result.Content = msg.Content
	} else if msg.ReasoningContent != "" {
		result.Content = msg.ReasoningContent
	}

	// 处理ToolCalls
	if len(msg.ToolCalls) > 0 {
		result.ToolCalls = make([]schema.ToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			result.ToolCalls = append(result.ToolCalls, schema.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: schema.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return result
}

// parseChunk 解析SSE数据为chunk
func (d *DoubaoAdapter) parseChunk(data []byte) *doubaoChunk {
	var chunk doubaoChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil
	}
	return &chunk
}

// convertToStreamChunk 转换为schema.StreamChunk
func (d *DoubaoAdapter) convertToStreamChunk(chunk *doubaoChunk) *schema.StreamChunk {
	streamChunk := &schema.StreamChunk{
		ID:      chunk.ID,
		Object:  chunk.Object,
		Created: chunk.Created,
		Model:   chunk.Model,
		Usage: schema.UsageInfo{
			PromptTokens:     chunk.Usage.PromptTokens,
			CompletionTokens: chunk.Usage.CompletionTokens,
			TotalTokens:      chunk.Usage.TotalTokens,
		},
	}

	for _, c := range chunk.Choices {
		choice := schema.Choice{
			Index:        c.Index,
			FinishReason: c.FinishReason,
		}
		choice.Delta = schema.Delta{
			Content:          c.Delta.Content,
			ReasoningContent: c.Delta.ReasoningContent,
			Role:             c.Delta.Role,
		}
		streamChunk.Choices = append(streamChunk.Choices, choice)
	}

	return streamChunk
}

// processToolCallsAccumulator 处理工具调用累加
func (d *DoubaoAdapter) processToolCallsAccumulator(chunk *doubaoChunk, accumulator map[int]schema.ToolCall) {
	for _, c := range chunk.Choices {
		for _, tc := range c.Delta.ToolCalls {
			index := tc.Index

			if tc.ID != "" && tc.Function.Name != "" {
				accumulator[index] = schema.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: schema.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			} else if existing, ok := accumulator[index]; ok {
				existing.Function.Arguments += tc.Function.Arguments
				accumulator[index] = existing
			}
		}
	}
}

// completeToolCallsIfNeeded 完成工具调用（如果finish_reason为tool_calls）
func (d *DoubaoAdapter) completeToolCallsIfNeeded(chunk *doubaoChunk, accumulator map[int]schema.ToolCall, streamChunk *schema.StreamChunk) {
	for _, c := range chunk.Choices {
		if c.FinishReason == "tool_calls" && len(accumulator) > 0 {
			for _, tc := range accumulator {
				streamChunk.Choices[0].Delta.ToolCalls = append(streamChunk.Choices[0].Delta.ToolCalls, tc)
			}
			for k := range accumulator {
				delete(accumulator, k)
			}
		}
	}
}

// SetConfig 设置配置参数
// 用于在运行时动态更新模型配置
func (d *DoubaoAdapter) SetConfig(model, apiKey, baseURL string) {
	d.model = model
	d.apiKey = apiKey
	if baseURL != "" {
		d.baseURL = strings.TrimSuffix(baseURL, "/")
	}
}
