package agent

import (
	"github.com/longstageai/donk/donk/internal/creative"
	donksql "github.com/longstageai/donk/donk/internal/sql"
)

// RegisterLLMDefaultAgents 注册真实 LLM Agent。每个 Agent 的具体实现分散在独立文件中，方便维护单个角色的提示词和输出逻辑。
// goalCreativeDeps 是目标创意 Agent 的依赖配置，如果为 nil 则该 Agent 的工具将使用零值初始化。
// executionAgentDeps 是任务执行 Agent 的依赖配置，如果为 nil 则该 Agent 的工具将使用零值初始化。
// db 是数据库连接，用于目标去重 Agent（必传）。
func RegisterLLMDefaultAgents(registry *creative.AgentRegistry, llm CreativeLLMClient, goalCreativeDeps *GoalCreativeAgentDeps, executionAgentDeps *ExecutionAgentDeps, db *donksql.DB) {
	if registry == nil {
		return
	}
	registry.Register(NewGoalCreativeAgent(llm, goalCreativeDeps))

	// 目标去重 Agent 需要数据库连接
	if db != nil {
		registry.Register(NewGoalDedupAgent(llm, &GoalDedupAgentDeps{DB: db}))
	}

	registry.Register(NewGoalValueReviewAgent(llm))
	registry.Register(NewGoalFeasibilityAgent(llm))
	registry.Register(NewGoalConvergenceAgent(llm))
	registry.Register(NewPlanningAgent(llm))
	registry.Register(NewPlanReviewAgent(llm))
	registry.Register(NewExecutionAgent(llm, executionAgentDeps))
	registry.Register(NewResultReviewAgent(llm))
	registry.Register(NewDeliveryAgent(llm))
}
