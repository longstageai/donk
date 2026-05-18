// agents 任务结束Agent
package agents

import (
	"encoding/json"
	multiagentToken "github.com/longstageai/donk/donk/internal/multiagent/token"
	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// CompletionAgent 任务结束Agent
// 负责任务的收尾工作
type CompletionAgent struct {
	llm          types.LLMClient
	tokenManager *multiagentToken.Manager
	tokenStats   *token.TokenStats
	name         string
	description  string
	agentLogger  *AgentLogger
	hub          *websocket.Hub // WebSocket Hub，用于发送任务完成消息
}

// NewCompletionAgent 创建任务结束Agent（使用token.Manager）
func NewCompletionAgent(llm types.LLMClient, tokenManager *multiagentToken.Manager, log *logger.Logger) *CompletionAgent {
	return &CompletionAgent{
		llm:          llm,
		tokenManager: tokenManager,
		name:         "CompletionAgent",
		description:  "任务结束Agent - 完成任务收尾",
		agentLogger:  NewAgentLogger(log),
	}
}

// NewCompletionAgentWithHub 创建任务结束Agent（带WebSocket Hub）
func NewCompletionAgentWithHub(llm types.LLMClient, tokenManager *multiagentToken.Manager, log *logger.Logger, hub *websocket.Hub) *CompletionAgent {
	return &CompletionAgent{
		llm:          llm,
		tokenManager: tokenManager,
		name:         "CompletionAgent",
		description:  "任务结束Agent - 完成任务收尾",
		agentLogger:  NewAgentLogger(log),
		hub:          hub,
	}
}

// NewCompletionAgentWithStats 创建任务结束Agent（使用统一token.TokenStats）
func NewCompletionAgentWithStats(llm types.LLMClient, tokenStats *token.TokenStats, log *logger.Logger) *CompletionAgent {
	return &CompletionAgent{
		llm:         llm,
		tokenStats:  tokenStats,
		name:        "CompletionAgent",
		agentLogger: NewAgentLogger(log),
	}
}

// NewCompletionAgentWithStatsAndHub 创建任务结束Agent（使用统一token.TokenStats和WebSocket Hub）
func NewCompletionAgentWithStatsAndHub(llm types.LLMClient, tokenStats *token.TokenStats, log *logger.Logger, hub *websocket.Hub) *CompletionAgent {
	return &CompletionAgent{
		llm:         llm,
		tokenStats:  tokenStats,
		name:        "CompletionAgent",
		agentLogger: NewAgentLogger(log),
		hub:         hub,
	}
}

// GetName 获取Agent名称
func (a *CompletionAgent) GetName() string {
	return a.name
}

// GetDescription 获取Agent描述
func (a *CompletionAgent) GetDescription() string {
	return a.description
}

// Process 处理任务结束
// 通过 WebSocket 发送任务完成消息，不再调用 LLM
func (a *CompletionAgent) Process(ctx *types.TaskContext) error {
	a.agentLogger.Info("任务完成，准备发送通知", map[string]interface{}{
		"taskID":   ctx.TaskID,
		"title":    ctx.Task.Title,
		"blessing": ctx.Output.Blessing,
	})

	// 如果有 WebSocket Hub，广播消息
	if a.hub != nil {

		msg := websocket.NewNotification("multi-agent", ctx.Task.Title, ctx.Output.Message)
		// 序列化为 JSON
		data, err := json.Marshal(msg)
		if err != nil {
			logger.Error("广播消息失败", map[string]interface{}{
				"multi": ctx.Task.Title,
				"error": err.Error(),
			})

		}

		// 广播消息
		a.hub.BroadcastJSON(data)

	} else {
		a.agentLogger.Info("WebSocket Hub 未配置，跳过消息广播", nil)
	}

	// 填充任务上下文
	ctx.Delivered = true
	ctx.UpdateStatus(types.StatusCompleted)

	return nil
}
