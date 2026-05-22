package agent

import (
	"context"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewExecutionAgent 创建任务执行 Agent，负责基于计划产出执行结果。
func NewExecutionAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("execution", "任务执行 Agent", creative.RoleExecution, []creative.EventType{creative.EventExecutionRequested, creative.EventExecutionRevisionRequested}, promptSpec(executionPrompt), llm, executionOutput)
}

func executionOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	result := creative.ExecutionResult{ID: creative.NextID("execution"), PlanID: input.Session.PlanID, StepResults: []creative.StepResult{{Status: "done", Output: content}}, CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactExecutionResult, Data: result}}, Events: []creative.EventDraft{{Type: creative.EventResultReviewRequested, DispatchMode: creative.DispatchExclusive, Priority: 20}}}
}
