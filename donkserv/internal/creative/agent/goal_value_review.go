package agent

import (
	"fmt"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewGoalValueReviewAgent 创建目标价值评审 Agent，负责评估目标的创造性和价值密度。
func NewGoalValueReviewAgent(llm CreativeLLMClient) creative.Agent {
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildGoalValueReviewPrompt(input)
	}
	return NewLLMAgentWithDynamicPrompt("goal_value_review", "目标价值评审 Agent", creative.RoleGoalValueReview, []creative.EventType{creative.EventGoalValueReviewRequested}, promptBuilder, llm, reviewOutput(creative.ArtifactValueReview, creative.EventGoalValueReviewPassed, creative.EventGoalValueReviewRejected, creative.EventGoalFeasibilityRequested, creative.EventGoalRefinementRequested, 70))
}

// buildGoalValueReviewPrompt 构建目标价值评审 Agent 的完整提示词
func buildGoalValueReviewPrompt(input creative.AgentInput) PromptSpec {
	// 从 input 中提取候选目标信息
	var title, description string
	for _, artifact := range input.Artifacts {
		if artifact.Type == creative.ArtifactCandidateGoal {
			if goalData, ok := artifact.Data.(creative.CandidateGoal); ok {
				title = goalData.Title
				description = goalData.Description
			}
		}
	}

	// 系统提示词：使用 prompts.go 中的模板
	systemPrompt := fmt.Sprintf(goalValueReviewPromptTemplate, title, description)

	// 用户提示词：Agent 自己管理，只保留必要运行时信息
	userPrompt := fmt.Sprintf(`请根据系统提示词中的信息完成目标价值评审。

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
