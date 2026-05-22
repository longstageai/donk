package agent

import (
	"github.com/longstageai/donk/donk/internal/creative"
)

// NewGoalValueReviewAgent 创建目标价值评审 Agent，负责评估目标的创造性和价值密度。
func NewGoalValueReviewAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("goal_value_review", "目标价值评审 Agent", creative.RoleGoalValueReview, []creative.EventType{creative.EventGoalValueReviewRequested}, promptSpec(goalValueReviewPrompt), llm, reviewOutput(creative.ArtifactValueReview, creative.EventGoalFeasibilityRequested, creative.EventGoalRefinementRequested, 70))
}
