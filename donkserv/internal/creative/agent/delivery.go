package agent

import (
	"context"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewDeliveryAgent 创建任务交付 Agent，负责整合最终交付内容并结束 Session。
func NewDeliveryAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("delivery", "任务交付 Agent", creative.RoleDelivery, []creative.EventType{creative.EventDeliveryRequested}, promptSpec(deliveryPrompt), llm, deliveryOutput)
}

func deliveryOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	delivery := creative.FinalDelivery{ID: creative.NextID("delivery"), GoalID: input.Session.FinalGoalID, PlanID: input.Session.PlanID, ResultID: input.Session.ExecutionID, Summary: content, CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactFinalDelivery, Data: delivery}}, Events: []creative.EventDraft{{Type: creative.EventDeliveryCompleted, DispatchMode: creative.DispatchExclusive, Priority: 1}, {Type: creative.EventLoopCompleted, DispatchMode: creative.DispatchExclusive, Priority: 1}, {Type: creative.EventSessionCompleted, DispatchMode: creative.DispatchExclusive, Priority: 1}}}
}
