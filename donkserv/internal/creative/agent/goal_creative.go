package agent

import (
	"context"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewGoalCreativeAgent 创建目标创意 Agent，负责生成候选目标并推进去重检查。
func NewGoalCreativeAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("goal_creative", "目标创意 Agent", creative.RoleGoalCreative, []creative.EventType{creative.EventGoalRequested, creative.EventGoalRegenerationRequested, creative.EventGoalRefinementRequested}, promptSpec(goalCreativePrompt), llm, goalCreativeOutput)
}

func goalCreativeOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	goal := creative.CandidateGoal{ID: creative.NextID("candidate_goal"), Title: firstNonEmpty(extractJSONField(content, "title"), "LLM 生成的候选目标"), Description: firstNonEmpty(extractJSONField(content, "description"), content), Motivation: extractJSONField(content, "value"), ContextBasis: []string{extractJSONField(content, "context_basis")}, ExpectedOutput: firstNonEmpty(extractJSONField(content, "expected_output"), "标准化候选目标"), CreatedBy: "goal_creative", CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactCandidateGoal, Data: goal}}, Events: []creative.EventDraft{{Type: creative.EventGoalDedupRequested, DispatchMode: creative.DispatchExclusive, Priority: 90}}}
}
