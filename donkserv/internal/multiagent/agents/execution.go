// agents 任务执行Agent
package agents

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/multiagent/prompts"
	multiagentToken "github.com/longstageai/donk/donk/internal/multiagent/token"
	"github.com/longstageai/donk/donk/internal/multiagent/tools"
	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// ExecutionAgent 任务执行Agent
// 负责按照规划逐个步骤执行任务
type ExecutionAgent struct {
	llm          types.LLMClient
	tokenManager *multiagentToken.Manager
	tokenStats   *token.TokenStats
	toolRegistry *tools.Registry
	name         string
	description  string
	agentLogger  *AgentLogger
}

// NewExecutionAgent 创建任务执行Agent（使用token.Manager）
func NewExecutionAgent(llm types.LLMClient, tokenManager *multiagentToken.Manager, toolRegistry *tools.Registry, log *logger.Logger) *ExecutionAgent {
	return &ExecutionAgent{
		llm:          llm,
		tokenManager: tokenManager,
		toolRegistry: toolRegistry,
		name:         "ExecutionAgent",
		description:  "任务执行Agent - 逐个步骤执行计划",
		agentLogger:  NewAgentLogger(log),
	}
}

// NewExecutionAgentWithStats 创建任务执行Agent（使用统一token.TokenStats）
func NewExecutionAgentWithStats(llm types.LLMClient, tokenStats *token.TokenStats, toolRegistry *tools.Registry, log *logger.Logger) *ExecutionAgent {
	return &ExecutionAgent{
		llm:          llm,
		tokenStats:   tokenStats,
		toolRegistry: toolRegistry,
		name:         "ExecutionAgent",
		description:  "任务执行Agent - 逐个步骤执行计划",
		agentLogger:  NewAgentLogger(log),
	}
}

// GetName 获取Agent名称
func (a *ExecutionAgent) GetName() string {
	return a.name
}

// GetDescription 获取Agent描述
func (a *ExecutionAgent) GetDescription() string {
	return a.description
}

// Process 处理任务执行 - 逐个步骤执行
func (a *ExecutionAgent) Process(ctx *types.TaskContext) error {
	if len(ctx.Plan) == 0 {
		return fmt.Errorf("没有可执行的规划步骤")
	}

	// 初始化待办事项列表
	ctx.Todos = make([]*types.TodoItem, 0, len(ctx.Plan))

	// 逐个步骤执行
	for i, step := range ctx.Plan {
		if err := a.executeStep(ctx, step, i+1, len(ctx.Plan)); err != nil {
			return fmt.Errorf("执行步骤%d失败: %w", step.Step, err)
		}
	}

	//// 所有步骤执行完成，生成最终输出
	if err := a.generateFinalOutput(ctx); err != nil {
		return fmt.Errorf("生成最终输出失败: %w", err)
	}

	ctx.UpdateStatus(types.StatusExecuting)
	return nil
}

// executeStep 执行单个步骤
func (a *ExecutionAgent) executeStep(ctx *types.TaskContext, step *types.PlanStep, currentStep, totalSteps int) error {
	// 构建步骤执行提示词
	config := prompts.NewConfig(ctx.CoreTheme)
	systemPrompt := prompts.GetExecutionAgentPrompt(config)

	stepInfo := fmt.Sprintf(`
任务信息：
- 主题: %s
- 标题: %s
- 描述: %s

当前执行步骤 (%d/%d)：
- 步骤%d: %s
- 描述: %s
- 工具: %s

%s

请执行当前步骤，并返回执行结果。`,
		ctx.Task.Theme,
		ctx.Task.Title,
		ctx.Task.Description,
		currentStep,
		totalSteps,
		step.Step,
		step.Action,
		step.Description,
		step.Tool,
		a.buildProgressInfo(ctx),
	)

	// 添加用户画像信息
	if ctx.UserProfile != nil && ctx.UserProfile.Name != "" {
		stepInfo += fmt.Sprintf("\n\n用户画像:\n")
		stepInfo += fmt.Sprintf("- 姓名: %s\n", ctx.UserProfile.Name)
		if len(ctx.UserProfile.Hobbies) > 0 {
			stepInfo += fmt.Sprintf("- 兴趣爱好: %v\n", ctx.UserProfile.Hobbies)
		}
	}

	// 获取工具定义
	toolDefs := a.toolRegistry.GetAllToolDefinitions()

	// 构建消息
	messages := []types.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: stepInfo},
	}

	// 打印LLM输入参数（第一次调用）
	a.agentLogger.LogLLMInput("ExecutionAgent", messages, toolDefs)

	// 调用LLM执行步骤
	resp, err := a.llm.Chat(messages, toolDefs)
	if err != nil {
		return fmt.Errorf("LLM调用失败: %w", err)
	}

	// 打印LLM输出参数（第一次调用）
	a.agentLogger.LogLLMOutput("ExecutionAgent", resp)

	// 记录Token使用
	if a.tokenStats != nil {
		a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "execution")
		if a.tokenStats.IsBudgetExceeded() {
			return fmt.Errorf("Token预算已超出限额")
		}
	} else if a.tokenManager != nil {
		a.tokenManager.RecordUsage("execution", types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	// 处理工具调用
	if len(resp.ToolCalls) > 0 {
		// 添加assistant消息（包含工具调用请求）
		messages = append(messages, types.Message{
			Role:      "assistant",
			Content:   "", // 工具调用时content为空
			ToolCalls: resp.ToolCalls,
		})

		// 执行所有工具调用
		for _, toolCall := range resp.ToolCalls {
			toolResult, _ := a.executeTool(toolCall)

			// 处理用户画像工具结果
			if toolCall.Function.Name == "get_user_profile" {
				a.handleUserProfileResult(toolResult)
			}

			// 将工具结果添加到消息中（无论成功失败都反馈）
			messages = append(messages, types.Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: toolCall.ID,
			})
		}

		// 打印LLM输入参数（第二次调用）
		a.agentLogger.LogLLMInput("ExecutionAgent", messages, nil)

		// 再次调用LLM获取执行结果
		resp, err = a.llm.Chat(messages, nil)
		if err != nil {
			return fmt.Errorf("LLM二次调用失败: %w", err)
		}

		// 打印LLM输出参数（第二次调用）
		a.agentLogger.LogLLMOutput("ExecutionAgent", resp)

		// 记录Token使用
		if a.tokenStats != nil {
			a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "execution")
			if a.tokenStats.IsBudgetExceeded() {
				return fmt.Errorf("Token预算已超出限额")
			}
		} else if a.tokenManager != nil {
			a.tokenManager.RecordUsage("execution", types.TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			})
		}
	}

	// 解析步骤执行结果
	todoItem := &types.TodoItem{
		Step:   step.Step,
		Action: step.Action,
		Status: types.TodoDone,
		Result: resp.Content,
	}

	// 尝试解析JSON结果
	var stepResult struct {
		Result  string `json:"result"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &stepResult); err == nil {
		if stepResult.Result != "" {
			todoItem.Result = stepResult.Result
		}
		if stepResult.Status == "failed" {
			todoItem.Status = types.TodoFailed
		}
	}

	ctx.Todos = append(ctx.Todos, todoItem)
	return nil
}

// executeTool 执行工具调用
func (a *ExecutionAgent) executeTool(toolCall types.ToolCall) (string, error) {
	result, err := a.toolRegistry.Execute(toolCall.Function.Name, toolCall.Function.Arguments)

	response := map[string]interface{}{
		"tool_call_id": toolCall.ID,
		"tool_name":    toolCall.Function.Name,
	}

	if err != nil {
		response["status"] = "error"
		response["error"] = err.Error()
		resultJSON, _ := json.Marshal(response)
		return string(resultJSON), err
	}

	response["status"] = "success"
	response["result"] = result
	resultJSON, _ := json.Marshal(response)
	return string(resultJSON), nil
}

// handleUserProfileResult 处理用户画像工具返回结果
func (a *ExecutionAgent) handleUserProfileResult(toolResult string) {
	var result struct {
		Status string                 `json:"status"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(toolResult), &result); err != nil {
		return
	}

	if result.Status != "success" || result.Result == nil {
		return
	}

	// 检查是否找到用户画像
	found, ok := result.Result["found"].(bool)
	if !ok {
		return
	}

	if found {
		// 解析用户画像信息
		profile := &types.UserProfile{
			UserID: getStringFromMap(result.Result, "user_id"),
			Name:   getStringFromMap(result.Result, "name"),
			Gender: getStringFromMap(result.Result, "gender"),
		}

		// 解析年龄
		if age, ok := result.Result["age"].(float64); ok {
			profile.Age = int(age)
		}

		// 解析职业
		profile.Occupation = getStringFromMap(result.Result, "occupation")

		// 解析兴趣爱好
		if hobbies, ok := result.Result["hobbies"].([]interface{}); ok {
			profile.Hobbies = make([]string, 0, len(hobbies))
			for _, h := range hobbies {
				if hobby, ok := h.(string); ok {
					profile.Hobbies = append(profile.Hobbies, hobby)
				}
			}
		}

		// 解析偏好设置
		if prefs, ok := result.Result["preferences"].(map[string]interface{}); ok {
			profile.Preferences = make(map[string]string)
			for k, v := range prefs {
				if str, ok := v.(string); ok {
					profile.Preferences[k] = str
				}
			}
		}

		a.agentLogger.Info("获取到用户画像", map[string]interface{}{
			"user_id": profile.UserID,
			"name":    profile.Name,
		})
	} else {
		// 未找到用户画像，设为nil
		a.agentLogger.Info("未找到用户画像，将使用通用策略", nil)
	}
}

// getStringFromMap 从map中安全获取字符串
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// buildProgressInfo 构建进度信息
func (a *ExecutionAgent) buildProgressInfo(ctx *types.TaskContext) string {
	var info strings.Builder

	// 添加已完成步骤信息
	if len(ctx.Todos) > 0 {
		info.WriteString("\n已完成的步骤:\n")
		for _, todo := range ctx.Todos {
			status := "✓"
			if todo.Status == types.TodoFailed {
				status = "✗"
			}
			info.WriteString(fmt.Sprintf("%s 步骤%d: %s\n", status, todo.Step, todo.Action))
		}
	}

	// 添加用户画像信息
	if ctx.UserProfile != nil && ctx.UserProfile.Name != "" {
		info.WriteString("\n用户画像:\n")
		info.WriteString(fmt.Sprintf("- 姓名: %s\n", ctx.UserProfile.Name))
		if ctx.UserProfile.Gender != "" {
			info.WriteString(fmt.Sprintf("- 性别: %s\n", ctx.UserProfile.Gender))
		}
		if ctx.UserProfile.Age > 0 {
			info.WriteString(fmt.Sprintf("- 年龄: %d\n", ctx.UserProfile.Age))
		}
		if ctx.UserProfile.Occupation != "" {
			info.WriteString(fmt.Sprintf("- 职业: %s\n", ctx.UserProfile.Occupation))
		}
		if len(ctx.UserProfile.Hobbies) > 0 {
			info.WriteString(fmt.Sprintf("- 兴趣爱好: %v\n", ctx.UserProfile.Hobbies))
		}
	} else {
		info.WriteString("\n注意：未找到该用户的画像信息，请使用通用策略生成内容。\n")
		info.WriteString("通用策略建议：\n")
		info.WriteString("- 使用温馨、友好的通用祝福语\n")
		info.WriteString("- 避免使用针对特定个人的称呼\n")
		info.WriteString("- 内容适合大多数用户接收\n")
	}

	if info.Len() == 0 {
		return "之前没有已完成的步骤。"
	}
	return info.String()
}

// generateFinalOutput 生成最终输出
func (a *ExecutionAgent) generateFinalOutput(ctx *types.TaskContext) error {
	systemPrompt := prompts.GenerateFinalOutput()

	// 构建执行摘要
	summary := fmt.Sprintf(`
任务执行摘要：

任务信息：
- 主题: %s
- 标题: %s
- 描述: %s

执行结果：
`,
		ctx.Task.Theme,
		ctx.Task.Title,
		ctx.Task.Description,
	)

	for _, todo := range ctx.Todos {
		status := "完成"
		if todo.Status == types.TodoFailed {
			status = "失败"
		}
		summary += fmt.Sprintf("- 步骤%d %s: %s (结果: %s)\n", todo.Step, todo.Action, status, todo.Result)
	}

	messages := []types.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: summary},
	}

	// 打印LLM输入参数
	a.agentLogger.LogLLMInput("ExecutionAgent-Final", messages, nil)

	resp, err := a.llm.Chat(messages, nil)
	if err != nil {
		return fmt.Errorf("生成最终输出失败: %w", err)
	}

	// 打印LLM输出参数
	a.agentLogger.LogLLMOutput("ExecutionAgent-Final", resp)

	// 记录Token使用
	if a.tokenStats != nil {
		a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "completion")
		if a.tokenStats.IsBudgetExceeded() {
			return fmt.Errorf("Token预算已超出限额")
		}
	} else if a.tokenManager != nil {
		a.tokenManager.RecordUsage("completion", types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	// 解析输出
	var output types.TaskOutput
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		content := extractJSON(resp.Content)
		if err := json.Unmarshal([]byte(content), &output); err != nil {
			// 如果解析失败，使用默认输出
			output = types.TaskOutput{
				Blessing: "祝你一切顺利！",
				Message:  "任务已执行完成。",
			}
		}
	}

	ctx.Output = &output
	return nil
}
