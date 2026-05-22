package agent

import (
	"context"

	"github.com/longstageai/donk/donk/internal/creative"
)

func reviewOutput(artifactType creative.ArtifactType, nextEvent creative.EventType, retryEvent creative.EventType, defaultPriority int) func(context.Context, creative.AgentInput, string, creative.TokenUsage) creative.AgentOutput {
	return func(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
		decision := parseAgentDecision(content)
		status := creative.AgentRunSucceeded
		if decision == creative.DecisionRejected {
			status = creative.AgentRunRejected
		}
		events := []creative.EventDraft{{Type: nextEvent, DispatchMode: creative.DispatchExclusive, Priority: defaultPriority}}
		if decision != creative.DecisionSucceeded && retryEvent != "" {
			events = []creative.EventDraft{{Type: retryEvent, DispatchMode: creative.DispatchExclusive, Priority: defaultPriority}}
		}
		return creative.AgentOutput{Status: status, Decision: decision, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: artifactType, Data: map[string]any{"decision": decision, "content": content}}}, Events: events}
	}
}
