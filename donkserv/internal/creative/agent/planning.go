package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewPlanningAgent 创建任务规划 Agent，负责将最终目标拆解为可执行计划。
func NewPlanningAgent(llm CreativeLLMClient) creative.Agent {
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildPlanningPrompt(input)
	}
	return NewLLMAgentWithDynamicPrompt("planning", "任务规划 Agent", creative.RolePlanning, []creative.EventType{creative.EventPlanRequested, creative.EventPlanRevisionRequested}, promptBuilder, llm, planningOutput)
}

// buildPlanningPrompt 构建任务规划 Agent 的完整提示词
func buildPlanningPrompt(input creative.AgentInput) PromptSpec {
	// 从 input 中提取最终可执行目标
	var goalTitle, goalDescription, goalExpectedDelivery string
	for _, artifact := range input.Artifacts {
		if artifact.Type == creative.ArtifactFinalExecutableGoal {
			if goal, ok := artifact.Data.(creative.FinalExecutableGoal); ok {
				goalTitle = goal.Title
				goalDescription = goal.Description
				goalExpectedDelivery = goal.ExpectedDelivery
			}
		}
	}

	// 系统提示词：使用 prompts.go 中的模板
	systemPrompt := fmt.Sprintf(planningPromptTemplate, goalTitle, goalDescription, goalExpectedDelivery)

	// 用户提示词：Agent 自己管理，只保留必要运行时信息
	userPrompt := fmt.Sprintf(`请根据系统提示词中的信息完成任务规划。

当前事件类型：%s
当前阶段：%s

输出要求：
请严格按系统提示词中的输出格式回答，生成可执行计划。`, input.Event.Type, input.Session.CurrentPhase)

	return PromptSpec{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		OutputFormat: "请严格按系统提示词中的输出格式回答，生成可执行计划。",
	}
}

func planningOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	plan := creative.ExecutablePlan{ID: creative.NextID("plan"), GoalID: input.Session.FinalGoalID, Steps: []creative.PlanStep{{ID: creative.NextID("step"), Title: firstNonEmpty(extractJSONField(content, "step_title"), "执行 LLM 规划步骤"), Description: content}}, CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactExecutablePlan, Data: plan}}, Events: []creative.EventDraft{{Type: creative.EventPlanReviewRequested, DispatchMode: creative.DispatchExclusive, Priority: 40}}}
}
