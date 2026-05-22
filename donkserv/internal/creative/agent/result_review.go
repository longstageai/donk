package agent

import (
	"github.com/longstageai/donk/donk/internal/creative"
)

// NewResultReviewAgent 创建成果审查 Agent，负责检查执行结果是否满足目标和计划。
func NewResultReviewAgent(llm CreativeLLMClient) creative.Agent {
	return NewLLMAgent("result_review", "成果审查 Agent", creative.RoleResultReview, []creative.EventType{creative.EventResultReviewRequested}, promptSpec(resultReviewPrompt), llm, reviewOutput(creative.ArtifactResultReview, creative.EventDeliveryRequested, creative.EventExecutionRevisionRequested, 10))
}
