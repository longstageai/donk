package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewGoalConvergenceAgent 创建目标收敛 Agent，负责整合前序评审并生成最终可执行目标。
func NewGoalConvergenceAgent(llm CreativeLLMClient) creative.Agent {
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildGoalConvergencePrompt(input)
	}
	return NewLLMAgentWithDynamicPrompt("goal_convergence", "目标收敛 Agent", creative.RoleGoalConvergence, []creative.EventType{creative.EventGoalConvergenceRequested}, promptBuilder, llm, goalConvergenceOutput)
}

// buildGoalConvergencePrompt 构建目标收敛 Agent 的完整提示词
func buildGoalConvergencePrompt(input creative.AgentInput) PromptSpec {
	// 从 input 中提取候选目标、去重评审结果、价值评审结果、可行性评审结果
	var title, description string
	var dedupReview, valueReview, feasibilityReview string

	for _, artifact := range input.Artifacts {
		switch artifact.Type {
		case creative.ArtifactCandidateGoal:
			if goalData, ok := artifact.Data.(creative.CandidateGoal); ok {
				title = goalData.Title
				description = goalData.Description
			}
		case creative.ArtifactDedupReview:
			if review, ok := artifact.Data.(map[string]any); ok {
				dedupReview = fmt.Sprintf("判定结果：%v\n内容：%v", review["decision"], review["content"])
			}
		case creative.ArtifactValueReview:
			if review, ok := artifact.Data.(map[string]any); ok {
				valueReview = fmt.Sprintf("判定结果：%v\n内容：%v", review["decision"], review["content"])
			}
		case creative.ArtifactFeasibilityReview:
			if review, ok := artifact.Data.(map[string]any); ok {
				feasibilityReview = fmt.Sprintf("判定结果：%v\n内容：%v", review["decision"], review["content"])
			}
		}
	}

	// 系统提示词：使用 prompts.go 中的模板
	systemPrompt := fmt.Sprintf(goalConvergencePromptTemplate, title, description, dedupReview, valueReview, feasibilityReview)

	// 用户提示词：Agent 自己管理，只保留必要运行时信息
	userPrompt := fmt.Sprintf(`请根据系统提示词中的信息完成目标收敛。

当前事件类型：%s
当前阶段：%s

输出要求：
请严格按系统提示词中的输出格式回答，生成最终可执行目标。`, input.Event.Type, input.Session.CurrentPhase)

	return PromptSpec{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		OutputFormat: "请严格按系统提示词中的输出格式回答，生成最终可执行目标。",
	}
}

func goalConvergenceOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	goal := creative.FinalExecutableGoal{ID: creative.NextID("final_goal"), Title: firstNonEmpty(extractJSONField(content, "title"), "LLM 收敛后的最终目标"), Description: firstNonEmpty(extractJSONField(content, "description"), content), WhyNow: extractJSONField(content, "why_now"), ExpectedDelivery: firstNonEmpty(extractJSONField(content, "expected_delivery"), "可执行计划与最终交付物"), CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactFinalExecutableGoal, Data: goal}}, Events: []creative.EventDraft{{Type: creative.EventPlanRequested, DispatchMode: creative.DispatchExclusive, Priority: 50}}}
}
