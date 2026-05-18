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

	"github.com/longstageai/donk/donk/pkg/schema"
)

type OpenAIAdapter struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAIAdapter(apiKey, model, baseURL string) *OpenAIAdapter {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/chat/completions"
	}
	return &OpenAIAdapter{
		model:   model,
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{},
	}
}

func (o *OpenAIAdapter) Name() string {
	return "openai"
}

func (o *OpenAIAdapter) Chat(ctx context.Context, req *schema.ChatRequest) (*schema.ChatResponse, error) {
	req.Model = o.model

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
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

	// OpenAI 响应结构
	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				Role      string `json:"role"`
				ToolCalls []struct {
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

	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if openAIResp.Error != nil {
		return &schema.ChatResponse{
			Error: &schema.ResponseError{
				Code:    openAIResp.Error.Code,
				Message: openAIResp.Error.Message,
				Type:    openAIResp.Error.Type,
			},
		}, nil
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("响应为空")
	}

	msg := openAIResp.Choices[0].Message

	chatResp := &schema.ChatResponse{
		Content:      msg.Content,
		FinishReason: openAIResp.Choices[0].FinishReason,
		Model:        o.model,
		Usage: schema.UsageInfo{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}

	// 处理ToolCalls
	if len(msg.ToolCalls) > 0 {
		chatResp.ToolCalls = make([]schema.ToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			chatResp.ToolCalls = append(chatResp.ToolCalls, schema.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: schema.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return chatResp, nil
}

func (o *OpenAIAdapter) StreamChat(ctx context.Context, req *schema.ChatRequest) (*StreamResponse, error) {
	req.Model = o.model
	req.Stream = true
	// 启用流式响应中的usage统计
	if req.StreamOptions == nil {
		req.StreamOptions = &schema.StreamOptions{}
	}
	req.StreamOptions.IncludeUsage = true

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
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

	go o.processStream(resp.Body, streamResp, ctx)

	return streamResp, nil
}

// processStream 处理流式响应
func (o *OpenAIAdapter) processStream(body io.ReadCloser, streamResp *StreamResponse, ctx context.Context) {
	var lastUsage schema.UsageInfo
	toolCallsAccumulator := make(map[int]schema.ToolCall)

	defer body.Close()
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

		// OpenAI 流式响应结构
		var chunk struct {
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
					Content   string `json:"content"`
					Role      string `json:"role"`
					ToolCalls []struct {
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

		if err := json.Unmarshal(data, &chunk); err != nil {
			continue
		}

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

		// 记录最后一个usage
		if streamChunk.Usage.TotalTokens > 0 {
			lastUsage = streamChunk.Usage
		}

		for _, c := range chunk.Choices {
			choice := schema.Choice{
				Index:        c.Index,
				FinishReason: c.FinishReason,
			}
			choice.Delta = schema.Delta{
				Content: c.Delta.Content,
				Role:    c.Delta.Role,
			}
			streamChunk.Choices = append(streamChunk.Choices, choice)
		}

		// 处理工具调用累加
		for _, c := range chunk.Choices {
			for _, tc := range c.Delta.ToolCalls {
				index := tc.Index

				if tc.ID != "" && tc.Function.Name != "" {
					toolCallsAccumulator[index] = schema.ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: schema.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				} else if existing, ok := toolCallsAccumulator[index]; ok {
					existing.Function.Arguments += tc.Function.Arguments
					toolCallsAccumulator[index] = existing
				}
			}
		}

		// 如果finish_reason为tool_calls，完成工具调用
		for _, c := range chunk.Choices {
			if c.FinishReason == "tool_calls" && len(toolCallsAccumulator) > 0 {
				if len(streamChunk.Choices) > 0 {
					for _, tc := range toolCallsAccumulator {
						streamChunk.Choices[0].Delta.ToolCalls = append(streamChunk.Choices[0].Delta.ToolCalls, tc)
					}
				}
				for k := range toolCallsAccumulator {
					delete(toolCallsAccumulator, k)
				}
			}
		}

		select {
		case streamResp.Chunks <- streamChunk:
		case <-ctx.Done():
			return
		}
	}
}

// SetConfig 设置配置参数
// 用于在运行时动态更新模型配置
func (o *OpenAIAdapter) SetConfig(model, apiKey, baseURL string) {
	o.model = model
	o.apiKey = apiKey
	if baseURL != "" {
		o.baseURL = strings.TrimSuffix(baseURL, "/")
	}
}
