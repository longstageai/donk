package agent

import (
	"github.com/longstageai/donk/donk/internal/creative"
)

// NewGoalFeasibilityAgent 创建目标可行性 Agent，负责判断目标是否处于 Agent 可执行能力范围内。
func NewGoalFeasibilityAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("goal_feasibility", "目标可行性 Agent", creative.RoleGoalFeasibility, []creative.EventType{creative.EventGoalFeasibilityRequested, creative.EventGoalFeasibilityRecheckRequested}, promptSpec(goalFeasibilityPrompt), llm, reviewOutput(creative.ArtifactFeasibilityReview, creative.EventGoalConvergenceRequested, creative.EventGoalRefinementRequested, 60))
}
