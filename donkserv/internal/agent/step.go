package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/model"

	"github.com/longstageai/donk/donk/pkg/schema"
)

// step 执行ReAct的一步（非流式）
// 返回空字符串表示需要继续循环，返回非空字符串表示最终回复
func (a *Agent) step(ctx context.Context, step int) (string, error) {
	// 获取当前对话历史
	messages := a.sessionMemory.GetMessages()

	// 获取可用工具定义
	var toolDefs []schema.ToolDefinition
	if a.tools != nil {
		toolDefs = a.tools.GetToolDefinitions()
	}

	// 构建请求
	req := &schema.ChatRequest{
		Messages:    messages,
		Temperature: 0.7, // 适度随机性
		MaxTokens:   2048,
		Tools:       toolDefs,
	}

	// 调用模型
	resp, err := a.model.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("调用模型失败: %w", err)
	}

	// 记录Token消耗
	if a.tokenStats != nil && resp.Usage.TotalTokens > 0 {
		_ = a.tokenStats.Record(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	}

	// 检查模型错误
	if resp.Error != nil {
		return "", fmt.Errorf("模型返回错误: %s - %s", resp.Error.Code, resp.Error.Message)
	}

	// 如果模型调用了工具，执行工具
	if len(resp.ToolCalls) > 0 {
		a.sessionMemory.AddAssistantMessageWithToolCalls(resp.Content, resp.ToolCalls)

		for _, tc := range resp.ToolCalls {
			toolName := tc.Function.Name
			args := tc.Function.Arguments

			// 发送工具调用事件
			if a.onStream != nil {
				a.onStream(&StreamEvent{
					Type:      EventToolCall,
					ToolName:  toolName,
					ToolInput: args,
				})
			}

			// 解析工具参数
			var params map[string]any
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				params = make(map[string]any)
			}

			// 执行工具（内部会记录到 executionTrace）
			result := a.executeTool(toolName, params, step)

			// 记录工具结果到记忆（可选，解耦后可删除）
			toolResult := result.String()
			a.sessionMemory.AddToolMessage(toolResult, tc.ID)

			// 发送工具结果事件
			if a.onStream != nil {
				a.onStream(&StreamEvent{
					Type:       EventToolResult,
					ToolName:   toolName,
					ToolResult: toolResult,
				})
			}
		}
		// 工具执行完毕，继续下一轮循环
		return "", nil
	}

	// 模型返回文本回复，这是最终回复
	content := resp.Content
	a.sessionMemory.AddAssistantMessage(content)

	// 发送助手回复事件
	if a.onStream != nil {
		a.onStream(&StreamEvent{
			Type:    EventAssistant,
			Content: content,
		})
		a.onStream(&StreamEvent{Type: EventStop})
	}

	return content, nil
}

// stepStream 流式执行ReAct的一步
// 负责构建请求、处理流式响应、分发工具调用或发送最终回复
func (a *Agent) stepStream(ctx context.Context, step int) error {
	// 获取当前对话历史
	messages := a.sessionMemory.GetMessages()
	var toolDefs []schema.ToolDefinition
	if a.tools != nil {
		toolDefs = a.tools.GetToolDefinitions()
	}

	// 构建聊天请求
	req := &schema.ChatRequest{
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2048,
		Tools:       toolDefs,
	}

	// 调用模型获取流式响应
	streamResp, err := a.model.StreamChat(ctx, req)
	if err != nil {
		return fmt.Errorf("流式调用模型失败: %w", err)
	}

	// 处理流式事件
	result, err := a.processStreamEvents(ctx, streamResp)
	if err != nil {
		if errors.Is(err, ErrCanceled) {
			return ErrCanceled
		}
		return err
	}

	// 从 streamResp 获取 Usage（由模型适配器在流结束时设置）
	usage := streamResp.Usage

	// 记录Token消耗
	if a.tokenStats != nil && usage.TotalTokens > 0 {
		_ = a.tokenStats.Record(usage.PromptTokens, usage.CompletionTokens)
	}

	// 如果有工具调用，执行工具
	if result.HasTools {
		return a.executeToolCalls(ctx, result.ToolCalls, result.Content.String(), step)
	}

	// 没有工具调用，发送最终回复
	return a.sendFinalResponse(result.Content.String(), usage)
}

// streamProcessResult 流式处理结果
// 用于在处理过程中累积数据
type streamProcessResult struct {
	Content   strings.Builder   // 累积的回复内容
	Reasoning strings.Builder   // 累积的思考过程
	ToolCalls []schema.ToolCall // 收集的工具调用列表
	HasTools  bool              // 是否包含工具调用
}

// processStreamEvents 处理流式事件循环
// 从streamResp.Chunks读取chunk并处理，直到通道关闭或上下文取消
func (a *Agent) processStreamEvents(ctx context.Context, streamResp *model.StreamResponse) (*streamProcessResult, error) {
	result := &streamProcessResult{}

	for {
		select {
		case chunk, ok := <-streamResp.Chunks:
			// 通道关闭，表示流式响应结束
			if !ok {
				return result, nil
			}
			// 处理接收到的chunk
			a.handleChunk(chunk, result)
		case <-ctx.Done():
			// 上下文取消，退出循环
			return result, ctx.Err()
		case <-a.cancel:
			// 用户取消，退出循环
			return result, ErrCanceled
		}
	}
}

// handleChunk 处理单个流式数据块
// 解析chunk中的choice，提取思考过程、内容和工具调用
func (a *Agent) handleChunk(chunk *schema.StreamChunk, result *streamProcessResult) {
	for _, choice := range chunk.Choices {
		// 处理思考过程（如豆包模型的reasoning_content）
		if choice.Delta.ReasoningContent != "" {
			result.Reasoning.WriteString(choice.Delta.ReasoningContent)
			if a.onStream != nil {
				a.onStream(&StreamEvent{
					Type:             EventReasoningDelta,
					ReasoningContent: choice.Delta.ReasoningContent,
				})
			}
		}
		// 处理普通文本内容
		if choice.Delta.Content != "" {
			result.Content.WriteString(choice.Delta.Content)
			if a.onStream != nil {
				a.onStream(&StreamEvent{
					Type:    EventContentDelta,
					Content: choice.Delta.Content,
				})
			}
		}
		// 处理工具调用
		if len(choice.Delta.ToolCalls) > 0 {
			result.ToolCalls = append(result.ToolCalls, choice.Delta.ToolCalls...)
			result.HasTools = true
		}
	}
}
