// agents 任务规划Agent
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

// PlanningAgent 任务规划Agent
// 负责将任务拆解为可执行的步骤
type PlanningAgent struct {
	llm          types.LLMClient
	tokenManager *multiagentToken.Manager
	tokenStats   *token.TokenStats
	name         string
	description  string
	agentLogger  *AgentLogger
}

// NewPlanningAgent 创建任务规划Agent（使用token.Manager）
func NewPlanningAgent(llm types.LLMClient, tokenManager *multiagentToken.Manager, log *logger.Logger) *PlanningAgent {
	return &PlanningAgent{
		llm:          llm,
		tokenManager: tokenManager,
		name:         "PlanningAgent",
		description:  "任务规划Agent - 将任务拆解为可执行步骤",
		agentLogger:  NewAgentLogger(log),
	}
}

// NewPlanningAgentWithStats 创建任务规划Agent（使用统一token.TokenStats）
func NewPlanningAgentWithStats(llm types.LLMClient, tokenStats *token.TokenStats, log *logger.Logger) *PlanningAgent {
	return &PlanningAgent{
		llm:         llm,
		tokenStats:  tokenStats,
		name:        "PlanningAgent",
		agentLogger: NewAgentLogger(log),
	}
}

// GetName 获取Agent名称
func (a *PlanningAgent) GetName() string {
	return a.name
}

// GetDescription 获取Agent描述
func (a *PlanningAgent) GetDescription() string {
	return a.description
}

// Process 处理任务规划
func (a *PlanningAgent) Process(ctx *types.TaskContext) error {
	// 构建提示词
	config := prompts.NewConfig(ctx.CoreTheme)
	systemPrompt := prompts.GetPlanningAgentPrompt(config)

	// 构建任务信息
	taskInfo := fmt.Sprintf(`
任务信息：
- 主题: %s
- 标题: %s
- 描述: %s
- 核心主题原因: %s
- 核心要素: %v
`,
		ctx.Task.Theme,
		ctx.Task.Title,
		ctx.Task.Description,
		ctx.Task.CoreThemeReason,
		ctx.Task.CoreElements,
	)

	// 构建消息
	messages := []types.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: taskInfo},
	}

	// 打印LLM输入参数
	a.agentLogger.LogLLMInput("PlanningAgent", messages, nil)

	// 调用LLM
	resp, err := a.llm.Chat(messages, nil)
	if err != nil {
		return fmt.Errorf("LLM调用失败: %w", err)
	}

	// 打印LLM输出参数
	a.agentLogger.LogLLMOutput("PlanningAgent", resp)

	// 记录Token使用
	if a.tokenStats != nil {
		a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "planning")
		if a.tokenStats.IsBudgetExceeded() {
			return fmt.Errorf("Token预算已超出限额")
		}
	} else if a.tokenManager != nil {
		a.tokenManager.RecordUsage("planning", types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	// 解析响应
	var result struct {
		Plan []*types.PlanStep `json:"plan"`
	}

	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		content := extractJSON(resp.Content)
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return fmt.Errorf("解析任务规划结果失败: %w", err)
		}
	}

	// 填充任务上下文
	ctx.Plan = result.Plan
	ctx.UpdateStatus(types.StatusPlanned)

	return nil
}
