package agent

import (
	"github.com/longstageai/donk/donk/internal/creative"
)

// RegisterLLMDefaultAgents 注册真实 LLM Agent。每个 Agent 的具体实现分散在独立文件中，方便维护单个角色的提示词和输出逻辑。
func RegisterLLMDefaultAgents(registry *creative.AgentRegistry, llm CreativeLLMClient) {
	if registry == nil {
		return
	}
	registry.Register(NewGoalCreativeAgent(llm))
	registry.Register(NewGoalDedupAgent(llm))
	registry.Register(NewGoalValueReviewAgent(llm))
	registry.Register(NewGoalFeasibilityAgent(llm))
	registry.Register(NewGoalConvergenceAgent(llm))
	registry.Register(NewPlanningAgent(llm))
	registry.Register(NewPlanReviewAgent(llm))
	registry.Register(NewExecutionAgent(llm))
	registry.Register(NewResultReviewAgent(llm))
	registry.Register(NewDeliveryAgent(llm))
}
