// agents 任务审查Agent
package agents

import (
	"encoding/json"
	"fmt"

	"github.com/longstageai/donk/donk/internal/multiagent/prompts"
	multiagentToken "github.com/longstageai/donk/donk/internal/multiagent/token"
	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// TaskReviewAgent 任务审查Agent
// 负责审查任务执行成果的质量
type TaskReviewAgent struct {
	llm          types.LLMClient
	tokenManager *multiagentToken.Manager
	tokenStats   *token.TokenStats
	name         string
	description  string
	agentLogger  *AgentLogger
}

// NewTaskReviewAgent 创建任务审查Agent（使用token.Manager）
func NewTaskReviewAgent(llm types.LLMClient, tokenManager *multiagentToken.Manager, log *logger.Logger) *TaskReviewAgent {
	return &TaskReviewAgent{
		llm:          llm,
		tokenManager: tokenManager,
		name:         "TaskReviewAgent",
		description:  "任务审查Agent - 审查执行成果质量",
		agentLogger:  NewAgentLogger(log),
	}
}

// NewTaskReviewAgentWithStats 创建任务审查Agent（使用统一token.TokenStats）
func NewTaskReviewAgentWithStats(llm types.LLMClient, tokenStats *token.TokenStats, log *logger.Logger) *TaskReviewAgent {
	return &TaskReviewAgent{
		llm:         llm,
		tokenStats:  tokenStats,
		name:        "TaskReviewAgent",
		agentLogger: NewAgentLogger(log),
	}
}

// GetName 获取Agent名称
func (a *TaskReviewAgent) GetName() string {
	return a.name
}

// GetDescription 获取Agent描述
func (a *TaskReviewAgent) GetDescription() string {
	return a.description
}

// Process 处理任务审查
func (a *TaskReviewAgent) Process(ctx *types.TaskContext) error {
	// 构建提示词
	config := prompts.NewConfig(ctx.CoreTheme)
	systemPrompt := prompts.GetTaskReviewAgentPrompt(config)

	// 构建任务和执行信息
	reviewInfo := fmt.Sprintf(`
任务信息：
- 主题: %s
- 标题: %s
- 描述: %s
- 核心主题原因: %s
- 核心要素: %v

执行结果：
`,
		ctx.Task.Theme,
		ctx.Task.Title,
		ctx.Task.Description,
		ctx.Task.CoreThemeReason,
		ctx.Task.CoreElements,
	)

	for _, todo := range ctx.Todos {
		reviewInfo += fmt.Sprintf("\n步骤%d: %s - %s\n", todo.Step, todo.Action, todo.Status)
		if todo.Result != "" {
			reviewInfo += fmt.Sprintf("  结果: %s\n", todo.Result)
		}
		if todo.Error != "" {
			reviewInfo += fmt.Sprintf("  错误: %s\n", todo.Error)
		}
	}

	reviewInfo += "\n最终输出:\n"
	if ctx.Output.Blessing != "" {
		reviewInfo += fmt.Sprintf("祝福语: %s\n", ctx.Output.Blessing)
	}
	if ctx.Output.Message != "" {
		reviewInfo += fmt.Sprintf("消息: %s\n", ctx.Output.Message)
	}
	if ctx.Output.CardImage != "" {
		reviewInfo += fmt.Sprintf("图片: %s\n", ctx.Output.CardImage)
	}

	// 添加上次审查反馈（如果有）
	if ctx.ExecutionReview.Attempt > 0 && !ctx.ExecutionReview.Passed {
		reviewInfo += fmt.Sprintf("\n上次审查反馈:\n%s\n", ctx.ExecutionReview.Feedback)
	}

	// 构建消息
	messages := []types.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: reviewInfo},
	}

	// 打印LLM输入参数
	a.agentLogger.LogLLMInput("TaskReviewAgent", messages, nil)

	// 调用LLM
	resp, err := a.llm.Chat(messages, nil)
	if err != nil {
		return fmt.Errorf("LLM调用失败: %w", err)
	}

	// 打印LLM输出参数
	a.agentLogger.LogLLMOutput("TaskReviewAgent", resp)

	// 记录Token使用
	if a.tokenStats != nil {
		a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "taskReview")
		if a.tokenStats.IsBudgetExceeded() {
			return fmt.Errorf("Token预算已超出限额")
		}
	} else if a.tokenManager != nil {
		a.tokenManager.RecordUsage("taskReview", types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	// 解析响应
	var result types.ReviewResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		content := extractJSON(resp.Content)
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return fmt.Errorf("解析任务审查结果失败: %w", err)
		}
	}

	// 更新尝试次数
	result.Attempt = ctx.ExecutionReview.Attempt + 1

	// 填充任务上下文
	ctx.ExecutionReview = &result
	ctx.UpdateStatus(types.StatusReviewing)

	return nil
}
