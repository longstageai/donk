package agent

import (
	"context"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewGoalConvergenceAgent 创建目标收敛 Agent，负责整合前序评审并生成最终可执行目标。
func NewGoalConvergenceAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("goal_convergence", "目标收敛 Agent", creative.RoleGoalConvergence, []creative.EventType{creative.EventGoalConvergenceRequested}, promptSpec(goalConvergencePrompt), llm, goalConvergenceOutput)
}

func goalConvergenceOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	goal := creative.FinalExecutableGoal{ID: creative.NextID("final_goal"), Title: firstNonEmpty(extractJSONField(content, "title"), "LLM 收敛后的最终目标"), Description: firstNonEmpty(extractJSONField(content, "description"), content), WhyNow: extractJSONField(content, "why_now"), ExpectedDelivery: firstNonEmpty(extractJSONField(content, "expected_delivery"), "可执行计划与最终交付物"), CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactFinalExecutableGoal, Data: goal}}, Events: []creative.EventDraft{{Type: creative.EventPlanRequested, DispatchMode: creative.DispatchExclusive, Priority: 50}}}
}
