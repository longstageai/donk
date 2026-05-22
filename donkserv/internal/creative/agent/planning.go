package agent

import (
	"context"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewPlanningAgent 创建任务规划 Agent，负责将最终目标拆解为可执行计划。
func NewPlanningAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("planning", "任务规划 Agent", creative.RolePlanning, []creative.EventType{creative.EventPlanRequested, creative.EventPlanRevisionRequested}, promptSpec(planningPrompt), llm, planningOutput)
}

func planningOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	plan := creative.ExecutablePlan{ID: creative.NextID("plan"), GoalID: input.Session.FinalGoalID, Steps: []creative.PlanStep{{ID: creative.NextID("step"), Title: firstNonEmpty(extractJSONField(content, "step_title"), "执行 LLM 规划步骤"), Description: content}}, CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactExecutablePlan, Data: plan}}, Events: []creative.EventDraft{{Type: creative.EventPlanReviewRequested, DispatchMode: creative.DispatchExclusive, Priority: 40}}}
}
