// background 后台Agent模块
package background

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// TaskExecutor 任务执行器
// 独立实现Agent执行逻辑，不依赖internal/agent包的内部方法
type TaskExecutor struct {
	model        model.Adapter     // LLM模型适配器
	tools        *tool.Registry    // 工具注册表
	maxLoop      int               // 最大循环次数
	timeout      time.Duration     // 超时时间
	systemPrompt string            // 系统提示词
	db           *sql.DB           // 数据库连接，用于Token统计
	tokenStats   *token.TokenStats // Token统计器
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Output     string        // 执行输出
	Iterations int           // 实际迭代次数
	Duration   time.Duration // 执行耗时
	Error      string        // 错误信息（如果有）
	TokenUsage TokenUsage    // Token使用统计
}

// TokenUsage Token使用统计
type TokenUsage struct {
	PromptTokens     int // 输入Token数
	CompletionTokens int // 输出Token数
	TotalTokens      int // 总Token数
}

// NewTaskExecutor 创建任务执行器
// model: LLM模型适配器
// tools: 工具注册表
// maxLoop: 最大循环次数
// timeout: 超时时间
// systemPrompt: 系统提示词
// db: 数据库连接
// 返回任务执行器实例
func NewTaskExecutor(model model.Adapter, tools *tool.Registry, maxLoop int, timeout time.Duration, systemPrompt string, db *sql.DB) *TaskExecutor {
	// 创建Token统计器
	var tokenStats *token.TokenStats
	if db != nil {
		var err error
		tokenStats, err = token.NewTokenStats(db)
		if err != nil {
			logger.Error("创建Token统计器失败", map[string]interface{}{
				"error": err.Error(),
			})
			tokenStats = nil
		}
	}

	return &TaskExecutor{
		model:        model,
		tools:        tools,
		maxLoop:      maxLoop,
		timeout:      timeout,
		systemPrompt: systemPrompt,
		db:           db,
		tokenStats:   tokenStats,
	}
}

// checkTokenBudget 检查Token预算
// 在执行前检查是否超出每日Token限额
// 返回错误（如果超出预算）
func (e *TaskExecutor) checkTokenBudget() error {
	if e.tokenStats == nil {
		logger.Debug("Token统计器未初始化，跳过预算检查", nil)
		return nil
	}

	// 检查预算
	ok, remaining := e.tokenStats.CheckBudget()
	if !ok {
		logger.Error("Token预算已超出", map[string]interface{}{
			"remaining": remaining,
		})
		return fmt.Errorf("Token预算已超出，剩余: %d", remaining)
	}

	// 如果剩余预算少于10000，记录警告
	if remaining > 0 && remaining < 10000 {
		logger.Warn("Token预算即将耗尽", map[string]interface{}{
			"remaining": remaining,
		})
	}

	logger.Debug("Token预算检查通过", map[string]interface{}{
		"remaining": remaining,
	})
	return nil
}

// Execute 执行任务
// ctx: 上下文
// 返回执行结果
func (e *TaskExecutor) Execute(ctx context.Context) *ExecutionResult {
	startTime := time.Now()
	result := &ExecutionResult{
		Iterations: 0,
		TokenUsage: TokenUsage{},
	}

	logger.Info("开始执行任务", map[string]interface{}{
		"max_loop": e.maxLoop,
		"timeout":  e.timeout.Seconds(),
	})

	// 1. 检查Token预算
	if err := e.checkTokenBudget(); err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		logger.Error("Token预算检查失败，任务终止", map[string]interface{}{
			"error": err.Error(),
		})
		return result
	}

	// 构建消息列表
	// 添加时间上下文，让Agent能够根据时间提供个性化问候
	timeContext := e.buildTimeContext()

	messages := []schema.Message{
		{
			Role:    "system",
			Content: e.systemPrompt,
		},
		{
			Role:    "user",
			Content: timeContext, // 使用时间上下文作为输入
		},
	}

	// ReAct循环
	for i := 0; i < e.maxLoop; i++ {
		result.Iterations = i + 1

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			result.Error = "任务被取消或超时"
			result.Duration = time.Since(startTime)
			logger.Warn("任务执行被取消", map[string]interface{}{
				"iterations": i,
			})
			return result
		default:
		}

		// 每轮循环前检查Token预算
		if err := e.checkTokenBudget(); err != nil {
			result.Error = err.Error()
			result.Duration = time.Since(startTime)
			logger.Error("Token预算检查失败，终止循环", map[string]interface{}{
				"iteration": i + 1,
				"error":     err.Error(),
			})
			return result
		}

		logger.Debug("执行第%d轮迭代", map[string]interface{}{
			"iteration": i + 1,
		})

		// 执行一步
		stepResult, done, usage, err := e.executeStep(ctx, messages)

		// 累加Token使用
		result.TokenUsage.PromptTokens += usage.PromptTokens
		result.TokenUsage.CompletionTokens += usage.CompletionTokens
		result.TokenUsage.TotalTokens += usage.TotalTokens

		if err != nil {
			result.Error = fmt.Sprintf("执行失败: %v", err)
			result.Duration = time.Since(startTime)
			logger.Error("执行步骤失败", map[string]interface{}{
				"iteration": i + 1,
				"error":     err.Error(),
			})
			// 记录Token消耗
			e.recordTokenUsage(result.TokenUsage)
			return result
		}

		if done {
			// 任务完成，获取最终结果
			result.Output = stepResult
			result.Duration = time.Since(startTime)
			logger.Info("任务执行完成", map[string]interface{}{
				"iterations": result.Iterations,
				"duration":   result.Duration.Milliseconds(),
				"tokens":     result.TokenUsage.TotalTokens,
			})
			// 记录Token消耗
			e.recordTokenUsage(result.TokenUsage)
			return result
		}

		// 继续下一轮
		logger.Debug("继续下一轮迭代", map[string]interface{}{
			"iteration": i + 1,
		})
	}

	// 达到最大循环次数
	result.Error = fmt.Sprintf("达到最大循环次数(%d)，任务未完成", e.maxLoop)
	result.Duration = time.Since(startTime)
	logger.Warn("达到最大循环次数", map[string]interface{}{
		"max_loop": e.maxLoop,
		"tokens":   result.TokenUsage.TotalTokens,
	})

	// 记录Token消耗
	e.recordTokenUsage(result.TokenUsage)

	return result
}

// executeStep 执行单步
// ctx: 上下文
// messages: 消息列表
// 返回：结果内容、是否完成、Token使用、错误
func (e *TaskExecutor) executeStep(ctx context.Context, messages []schema.Message) (string, bool, TokenUsage, error) {
	usage := TokenUsage{}

	// 获取可用工具定义
	var toolDefs []schema.ToolDefinition
	if e.tools != nil {
		toolDefs = e.tools.GetToolDefinitions()
	}

	logger.Debug("调用LLM", map[string]interface{}{
		"message_count": len(messages),
		"tool_count":    len(toolDefs),
	})

	// 构建请求
	req := &schema.ChatRequest{
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2048,
		Tools:       toolDefs,
	}

	// 调用模型
	resp, err := e.model.Chat(ctx, req)
	if err != nil {
		return "", false, usage, fmt.Errorf("调用模型失败: %w", err)
	}

	// 检查模型错误
	if resp.Error != nil {
		return "", false, usage, fmt.Errorf("模型返回错误: %s - %s", resp.Error.Code, resp.Error.Message)
	}

	// 记录Token消耗
	if resp.Usage.TotalTokens > 0 {
		usage.PromptTokens = resp.Usage.PromptTokens
		usage.CompletionTokens = resp.Usage.CompletionTokens
		usage.TotalTokens = resp.Usage.TotalTokens
		logger.Debug("Token消耗", map[string]interface{}{
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
		})
	}

	// 如果模型调用了工具，执行工具
	if len(resp.ToolCalls) > 0 {
		logger.Info("模型调用工具", map[string]interface{}{
			"tool_count": len(resp.ToolCalls),
		})

		// 添加助手消息
		messages = append(messages, schema.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: convertToolCalls(resp.ToolCalls),
		})

		// 执行每个工具调用
		for _, tc := range resp.ToolCalls {
			toolName := tc.Function.Name
			args := tc.Function.Arguments

			logger.Info("执行工具", map[string]interface{}{
				"tool": toolName,
				"args": args,
			})

			// 解析工具参数
			var params map[string]any
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				params = make(map[string]any)
			}

			// 执行工具
			toolResult := e.executeTool(ctx, toolName, params)

			// 添加工具结果到消息列表
			messages = append(messages, schema.Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
			})

			logger.Debug("工具执行完成", map[string]interface{}{
				"tool":   toolName,
				"result": toolResult,
			})
		}

		// 工具执行完毕，继续下一轮（返回false表示未完成）
		return "", false, usage, nil
	}

	// 模型返回文本回复，这是最终结果
	content := resp.Content
	logger.Info("模型返回最终结果", map[string]interface{}{
		"content_length": len(content),
	})

	return content, true, usage, nil
}

// executeTool 执行工具
// ctx: 上下文
// toolName: 工具名称
// params: 工具参数
// 返回工具执行结果
func (e *TaskExecutor) executeTool(ctx context.Context, toolName string, params map[string]any) string {
	if e.tools == nil {
		return "错误：工具注册表未初始化"
	}

	// 查找工具
	t, ok := e.tools.Get(toolName)
	if !ok || t == nil {
		return fmt.Sprintf("错误：未找到工具 '%s'", toolName)
	}

	// 创建工具上下文
	toolCtx := &tool.Context{
		ToolName: toolName,
		Params:   params,
		Metadata: map[string]any{
			"context": ctx,
		},
	}

	// 执行工具
	result, err := t.Execute(toolCtx)
	if err != nil {
		return fmt.Sprintf("工具执行错误: %v", err)
	}

	// 返回结果字符串
	return result.String()
}

// recordTokenUsage 记录Token使用量到数据库
// usage: Token使用统计
func (e *TaskExecutor) recordTokenUsage(usage TokenUsage) {
	if e.tokenStats == nil || usage.TotalTokens <= 0 {
		logger.Debug("Token统计器未初始化或无Token消耗，跳过记录", map[string]interface{}{
			"total_tokens": usage.TotalTokens,
		})
		return
	}

	err := e.tokenStats.Record(usage.PromptTokens, usage.CompletionTokens)
	if err != nil {
		logger.Error("记录Token消耗失败", map[string]interface{}{
			"error":         err.Error(),
			"prompt_tokens": usage.PromptTokens,
			"output_tokens": usage.CompletionTokens,
			"total_tokens":  usage.TotalTokens,
		})
	} else {
		logger.Info("Token消耗已记录", map[string]interface{}{
			"prompt_tokens": usage.PromptTokens,
			"output_tokens": usage.CompletionTokens,
			"total_tokens":  usage.TotalTokens,
		})
	}
}

// buildTimeContext 构建时间上下文
// 只提供原始时间信息，让Agent自己决定如何回应，保持灵活性
// 返回包含时间信息的上下文字符串
func (e *TaskExecutor) buildTimeContext() string {
	now := time.Now()

	// 只提供原始时间信息，不做任何业务预设
	context := fmt.Sprintf("当前系统时间：%s\n", now.Format("2006年1月2日 15:04:05"))
	context += fmt.Sprintf("星期：%s\n", now.Weekday().String())
	context += fmt.Sprintf("小时：%d\n", now.Hour())

	logger.Debug("构建时间上下文", map[string]interface{}{
		"datetime": now.Format("2006-01-02 15:04:05"),
		"weekday":  now.Weekday().String(),
	})

	return context
}

// convertToolCalls 转换工具调用格式
// toolCalls: 原始工具调用列表
// 返回转换后的工具调用列表
func convertToolCalls(toolCalls []schema.ToolCall) []schema.ToolCall {
	// 已经是相同类型，直接返回
	return toolCalls
}

// GetDailyTokenLimit 获取每日Token限额
// 返回每日Token限额，-1表示不限制
func GetDailyTokenLimit() int {
	provider := setting.GetProvider()
	if provider == nil {
		return -1
	}

	cfg, err := provider.GetAgentConfig()
	if err != nil || cfg == nil {
		return -1
	}

	return cfg.DailyTokenLimit
}
