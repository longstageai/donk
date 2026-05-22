package agent

import (
	"github.com/longstageai/donk/donk/internal/creative"
)

// NewPlanReviewAgent 创建规划审查 Agent，负责检查计划是否完整且可执行。
func NewPlanReviewAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("plan_review", "规划审查 Agent", creative.RolePlanReview, []creative.EventType{creative.EventPlanReviewRequested}, promptSpec(planReviewPrompt), llm, reviewOutput(creative.ArtifactPlanReview, creative.EventExecutionRequested, creative.EventPlanRevisionRequested, 30))
}
