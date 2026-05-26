package agent

import (
	"fmt"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewPlanReviewAgent 创建规划审查 Agent，负责检查计划是否完整且可执行。
func NewPlanReviewAgent(llm CreativeLLMClient) creative.Agent {
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildPlanReviewPrompt(input)
	}
	return NewLLMAgentWithDynamicPrompt("plan_review", "规划审查 Agent", creative.RolePlanReview, []creative.EventType{creative.EventPlanReviewRequested}, promptBuilder, llm, reviewOutput(creative.ArtifactPlanReview, creative.EventPlanReviewPassed, creative.EventPlanReviewRejected, creative.EventExecutionRequested, creative.EventPlanRevisionRequested, 30))
}

// buildPlanReviewPrompt 构建规划审查 Agent 的完整提示词
func buildPlanReviewPrompt(input creative.AgentInput) PromptSpec {
	// 从 input 中提取最终目标和可执行计划
	var goalTitle, goalDescription string
	var planDescription string

	for _, artifact := range input.Artifacts {
		switch artifact.Type {
		case creative.ArtifactFinalExecutableGoal:
			if goal, ok := artifact.Data.(creative.FinalExecutableGoal); ok {
				goalTitle = goal.Title
				goalDescription = goal.Description
			}
		case creative.ArtifactExecutablePlan:
			if plan, ok := artifact.Data.(creative.ExecutablePlan); ok && len(plan.Steps) > 0 {
				for _, step := range plan.Steps {
					planDescription += fmt.Sprintf("- %s: %s\n", step.Title, step.Description)
				}
			}
		}
	}

	// 系统提示词：使用 prompts.go 中的模板
	systemPrompt := fmt.Sprintf(planReviewPromptTemplate, goalTitle, goalDescription, planDescription)

	// 用户提示词：Agent 自己管理，只保留必要运行时信息
	userPrompt := fmt.Sprintf(`请根据系统提示词中的信息完成规划审查。

当前事件类型：%s
当前阶段：%s

输出要求：
请严格按系统提示词中的输出格式回答。必须明确写出 判定结果：通过 或 判定结果：不通过，并给出原因。`, input.Event.Type, input.Session.CurrentPhase)

	return PromptSpec{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		OutputFormat: "请严格按系统提示词中的输出格式回答。必须明确写出 判定结果：通过 或 判定结果：不通过，并给出原因。",
	}
}
