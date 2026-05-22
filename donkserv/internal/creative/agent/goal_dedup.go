package agent

import (
	"github.com/longstageai/donk/donk/internal/creative"
)

// NewGoalDedupAgent 创建目标去重 Agent，负责检测候选目标是否与历史任务重复。
func NewGoalDedupAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("goal_dedup", "目标去重 Agent", creative.RoleGoalDedup, []creative.EventType{creative.EventGoalDedupRequested}, promptSpec(goalDedupPrompt), llm, reviewOutput(creative.ArtifactDedupReview, creative.EventGoalValueReviewRequested, creative.EventGoalRegenerationRequested, 80))
}
