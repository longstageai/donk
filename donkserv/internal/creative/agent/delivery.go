package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
)

// NewDeliveryAgent 创建任务交付 Agent，负责整合最终交付内容并结束 Session。
func NewDeliveryAgent(llm CreativeLLMClient) creative.Agent {
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildDeliveryPrompt(input)
	}
	return NewLLMAgentWithDynamicPrompt("delivery", "任务交付 Agent", creative.RoleDelivery, []creative.EventType{creative.EventDeliveryRequested}, promptBuilder, llm, deliveryOutput)
}

// buildDeliveryPrompt 构建任务交付 Agent 的完整提示词
func buildDeliveryPrompt(input creative.AgentInput) PromptSpec {
	// 从 input 中提取最终目标、计划、执行结果
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
	systemPrompt := fmt.Sprintf(deliveryPromptTemplate, goalTitle, goalDescription, planDescription, executionResult)

	// 用户提示词：Agent 自己管理，只保留必要运行时信息
	userPrompt := fmt.Sprintf(`请根据系统提示词中的信息完成最终交付。

当前事件类型：%s
当前阶段：%s

输出要求：
请严格按系统提示词中的输出格式回答，生成最终交付内容。`, input.Event.Type, input.Session.CurrentPhase)

	return PromptSpec{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		OutputFormat: "请严格按系统提示词中的输出格式回答，生成最终交付内容。",
	}
}

func deliveryOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	delivery := creative.FinalDelivery{ID: creative.NextID("delivery"), GoalID: input.Session.FinalGoalID, PlanID: input.Session.PlanID, ResultID: input.Session.ExecutionID, Summary: content, CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactFinalDelivery, Data: delivery}}, Events: []creative.EventDraft{{Type: creative.EventDeliveryCompleted, DispatchMode: creative.DispatchExclusive, Priority: 1}, {Type: creative.EventLoopCompleted, DispatchMode: creative.DispatchExclusive, Priority: 1}, {Type: creative.EventSessionCompleted, DispatchMode: creative.DispatchExclusive, Priority: 1}}}
}
