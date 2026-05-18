// agents 规划审查Agent
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

// PlanReviewAgent 规划审查Agent
// 负责审查任务规划的质量
type PlanReviewAgent struct {
	llm          types.LLMClient
	tokenManager *multiagentToken.Manager
	tokenStats   *token.TokenStats
	name         string
	description  string
	agentLogger  *AgentLogger
}

// NewPlanReviewAgent 创建规划审查Agent（使用token.Manager）
func NewPlanReviewAgent(llm types.LLMClient, tokenManager *multiagentToken.Manager, log *logger.Logger) *PlanReviewAgent {
	return &PlanReviewAgent{
		llm:          llm,
		tokenManager: tokenManager,
		name:         "PlanReviewAgent",
		description:  "规划审查Agent - 审查任务规划质量",
		agentLogger:  NewAgentLogger(log),
	}
}

// NewPlanReviewAgentWithStats 创建规划审查Agent（使用统一token.TokenStats）
func NewPlanReviewAgentWithStats(llm types.LLMClient, tokenStats *token.TokenStats, log *logger.Logger) *PlanReviewAgent {
	return &PlanReviewAgent{
		llm:         llm,
		tokenStats:  tokenStats,
		name:        "PlanReviewAgent",
		agentLogger: NewAgentLogger(log),
	}
}

// GetName 获取Agent名称
func (a *PlanReviewAgent) GetName() string {
	return a.name
}

// GetDescription 获取Agent描述
func (a *PlanReviewAgent) GetDescription() string {
	return a.description
}

// Process 处理规划审查
func (a *PlanReviewAgent) Process(ctx *types.TaskContext) error {
	// 构建提示词
	config := prompts.NewConfig(ctx.CoreTheme)
	systemPrompt := prompts.GetPlanReviewAgentPrompt(config)

	// 构建任务和规划信息
	planInfo := fmt.Sprintf(`
任务信息：
- 主题: %s
- 标题: %s
- 描述: %s
- 核心主题原因: %s
- 核心要素: %v

执行规划：
`,
		ctx.Task.Theme,
		ctx.Task.Title,
		ctx.Task.Description,
		ctx.Task.CoreThemeReason,
		ctx.Task.CoreElements,
	)

	for _, step := range ctx.Plan {
		planInfo += fmt.Sprintf("\n步骤%d: %s\n", step.Step, step.Action)
		planInfo += fmt.Sprintf("  描述: %s\n", step.Description)
		planInfo += fmt.Sprintf("  工具: %s\n", step.Tool)
		planInfo += fmt.Sprintf("  输入: %v\n", step.Input)
		planInfo += fmt.Sprintf("  输出: %v\n", step.Output)
		planInfo += fmt.Sprintf("  依赖: %v\n", step.Dependencies)
	}

	// 添加上次审查反馈（如果有）
	if ctx.PlanReview.Attempt > 0 && !ctx.PlanReview.Passed {
		planInfo += fmt.Sprintf("\n上次审查反馈:\n%s\n", ctx.PlanReview.Feedback)
	}

	// 构建消息
	messages := []types.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: planInfo},
	}

	// 打印LLM输入参数
	a.agentLogger.LogLLMInput("PlanReviewAgent", messages, nil)

	// 调用LLM
	resp, err := a.llm.Chat(messages, nil)
	if err != nil {
		return fmt.Errorf("LLM调用失败: %w", err)
	}

	// 打印LLM输出参数
	a.agentLogger.LogLLMOutput("PlanReviewAgent", resp)

	// 记录Token使用
	if a.tokenStats != nil {
		a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "planReview")
		if a.tokenStats.IsBudgetExceeded() {
			return fmt.Errorf("Token预算已超出限额")
		}
	} else if a.tokenManager != nil {
		a.tokenManager.RecordUsage("planReview", types.TokenUsage{
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
			return fmt.Errorf("解析规划审查结果失败: %w", err)
		}
	}

	// 更新尝试次数
	result.Attempt = ctx.PlanReview.Attempt + 1

	// 填充任务上下文
	ctx.PlanReview = &result
	ctx.UpdateStatus(types.StatusPlanReviewing)

	return nil
}
