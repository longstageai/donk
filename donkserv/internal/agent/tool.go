package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/tool"

	"github.com/longstageai/donk/donk/pkg/schema"
)

// executeToolCalls 执行工具调用列表
// 依次执行每个工具并发送相应事件
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []schema.ToolCall, content string, step int) error {
	// 将助手回复添加到记忆（包含工具调用）
	a.sessionMemory.AddAssistantMessageWithToolCalls(content, toolCalls)

	// 发送助手消息事件
	if a.onStream != nil {
		a.onStream(&StreamEvent{Type: EventAssistant, Content: content})
	}

	// 逐个执行工具调用
	for _, tc := range toolCalls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-a.cancel:
			return ErrCanceled
		default:
		}
		a.executeSingleToolCall(ctx, tc, step)
	}

	return nil
}

// executeSingleToolCall 执行单个工具调用
// 发送工具调用事件、解析参数、执行工具、记录结果
func (a *Agent) executeSingleToolCall(ctx context.Context, tc schema.ToolCall, step int) {
	toolName := tc.Function.Name
	args := tc.Function.Arguments

	// 发送工具调用开始事件
	if a.onStream != nil {
		a.onStream(&StreamEvent{
			Type:      EventToolCall,
			ToolName:  toolName,
			ToolInput: args,
		})
	}

	// 解析JSON参数
	params := a.parseToolParams(args)
	// 执行工具
	result := a.executeTool(toolName, params, step)

	// 获取结果字符串
	toolResult := result.String()
	// 将工具结果添加到记忆
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

// parseToolParams 解析工具参数
// 将JSON字符串参数解析为map
func (a *Agent) parseToolParams(args string) map[string]any {
	var params map[string]any
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		params = make(map[string]any)
	}
	return params
}

// executeTool 执行工具
// 调用工具注册表执行工具并返回结果
func (a *Agent) executeTool(name string, params map[string]any, step int) *tool.Result {
	// 记录开始时间
	startTime := time.Now()

	if name == "command_executor" && a.workspace != "" {
		params["working_dir"] = a.workspace
	}

	// 执行工具
	result, err := a.tools.Execute(name, params)
	// 计算执行时长
	duration := time.Since(startTime).Milliseconds()

	// 序列化输入参数
	inputJSON, _ := json.Marshal(params)

	// 记录到 sessionToolCalls（兼容旧代码）
	a.sessionToolCalls = append(a.sessionToolCalls, memory.ToolCallInfo{
		ID:       fmt.Sprintf("call_%d", len(a.sessionToolCalls)),
		Name:     name,
		Input:    string(inputJSON),
		Output:   result.String(),
		Duration: duration,
		Round:    step,
	})

	// 记录到 executionTrace（解耦后的执行轨迹）
	if a.executionTrace != nil {
		a.executionTrace.Add(memory.ExecutionStep{
			Round:    step,
			Action:   name,
			Input:    string(inputJSON),
			Output:   result.String(),
			Success:  err == nil,
			Duration: duration,
		})
	}

	// 如果执行出错，返回错误结果
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error())
	}
	return result
}

// sendFinalResponse 发送最终回复
// 将最终回复添加到记忆，发送助手消息和停止事件
func (a *Agent) sendFinalResponse(content string, usage schema.UsageInfo) error {
	// 添加到工作记忆
	a.sessionMemory.AddAssistantMessage(content)

	// 保存到历史记录
	a.saveToHistory(a.currentInput, content, a.sessionToolCalls...)

	// 发送助手回复和停止事件
	if a.onStream != nil {
		a.onStream(&StreamEvent{
			Type:    EventAssistant,
			Content: content,
			Usage:   usage,
		})
		a.onStream(&StreamEvent{Type: EventStop})
	}

	return ErrTaskCompleted
}
