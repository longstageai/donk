// agents 技能自动发现 Agent 模块
// Notifier Agent 负责通过 WebSocket 推送通知
package agents

import (
	"encoding/json"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// NotifierAgent 通知 Agent
// 负责在技能创建完成后通过 WebSocket 推送通知
type NotifierAgent struct {
	hub *websocket.Hub
}

// NewNotifierAgent 创建通知 Agent
// 参数:
//   - hub: WebSocket Hub 实例
//
// 返回:
//   - *NotifierAgent: Agent 实例
func NewNotifierAgent(hub *websocket.Hub) *NotifierAgent {
	return &NotifierAgent{
		hub: hub,
	}
}

// NotifySkillCreated 推送技能创建通知
// 参数:
//   - skill: 创建的技能
//   - source: 来源（analyzer/creative）
func (n *NotifierAgent) NotifySkillCreated(s *skill.Skill, source string) {
	if n.hub == nil {
		logger.Warn("WebSocket Hub 未初始化，跳过通知", map[string]interface{}{
			"skill_name": s.Name(),
		})
		return
	}

	logger.Info("推送技能创建通知", map[string]interface{}{
		"skill_name": s.Name(),
		"source":     source,
	})

	// 构建消息
	//msg := SkillCreatedMessage{
	//	Type:        string(MessageTypeSkillCreated),
	//	Name:        s.Name(),
	//	Description: s.Description(),
	//	CreatedAt:   time.Now(),
	//	Source:      source,
	//}
	msg := websocket.NewNotification(string(MessageTypeSkillCreated), s.Name(), s.Description())
	// 序列化为 JSON
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化通知消息失败", map[string]interface{}{
			"skill_name": s.Name(),
			"error":      err.Error(),
		})
		return
	}

	// 广播消息
	n.hub.BroadcastJSON(data)

	logger.Debug("技能创建通知已广播", map[string]interface{}{
		"skill_name": s.Name(),
		"message":    string(data),
	})
}

// NotifyDiscoveryCompleted 推送发现任务完成通知
// 参数:
//   - taskID: 任务ID
//   - createdCount: 创建数量
//   - skippedCount: 跳过数量
//   - createdSkills: 创建的技能名称列表
func (n *NotifierAgent) NotifyDiscoveryCompleted(
	taskID string,
	createdCount int,
	skippedCount int,
	createdSkills []string,
) {
	if n.hub == nil {
		logger.Warn("WebSocket Hub 未初始化，跳过通知", map[string]interface{}{
			"task_id": taskID,
		})
		return
	}

	logger.Info("推送发现任务完成通知", map[string]interface{}{
		"task_id":       taskID,
		"created_count": createdCount,
		"skipped_count": skippedCount,
	})

	// 构建消息
	msg := DiscoveryCompletedMessage{
		Type:          string(MessageTypeDiscoveryCompleted),
		TaskID:        taskID,
		CreatedCount:  createdCount,
		SkippedCount:  skippedCount,
		CreatedSkills: createdSkills,
	}

	// 序列化为 JSON
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化完成通知失败", map[string]interface{}{
			"task_id": taskID,
			"error":   err.Error(),
		})
		return
	}

	// 广播消息
	if false {
		n.hub.BroadcastJSON(data)
	}

	logger.Debug("发现任务完成通知已广播", map[string]interface{}{
		"task_id": taskID,
	})
}
