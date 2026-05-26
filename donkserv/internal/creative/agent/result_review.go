package agent

import (
	"fmt"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewResultReviewAgent 创建成果审查 Agent，负责检查执行结果是否满足目标和计划。
func NewResultReviewAgent(llm CreativeLLMClient) creative.Agent {
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildResultReviewPrompt(input)
	}
	return NewLLMAgentWithDynamicPrompt("result_review", "成果审查 Agent", creative.RoleResultReview, []creative.EventType{creative.EventResultReviewRequested}, promptBuilder, llm, reviewOutput(creative.ArtifactResultReview, creative.EventResultReviewPassed, creative.EventResultReviewRejected, creative.EventDeliveryRequested, creative.EventExecutionRevisionRequested, 10))
}

// buildResultReviewPrompt 构建成果审查 Agent 的完整提示词
func buildResultReviewPrompt(input creative.AgentInput) PromptSpec {
	// 从 input 中提取最终目标、计划和执行结果
	var goalTitle, goalDescription string
	var planDescription string
	var executionResult string

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
		case creative.ArtifactExecutionResult:
			if result, ok := artifact.Data.(creative.ExecutionResult); ok && len(result.StepResults) > 0 {
				for _, stepResult := range result.StepResults {
					executionResult += fmt.Sprintf("- 状态：%s\n输出：%s\n", stepResult.Status, stepResult.Output)
				}
			}
		}
	}

	// 系统提示词：使用 prompts.go 中的模板
	systemPrompt := fmt.Sprintf(resultReviewPromptTemplate, goalTitle, goalDescription, planDescription, executionResult)

	// 用户提示词：Agent 自己管理，只保留必要运行时信息
	userPrompt := fmt.Sprintf(`请根据系统提示词中的信息完成成果审查。

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
