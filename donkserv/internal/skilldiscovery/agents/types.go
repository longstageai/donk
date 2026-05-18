// agents 技能自动发现 Agent 模块
// 共享类型定义
package agents

import (
	"time"

	"github.com/longstageai/donk/donk/internal/skill"
)

// SkillCandidate 技能候选
// 表示从对话中分析出的潜在技能需求
type SkillCandidate struct {
	Name        string   `json:"name"`        // 技能名称
	Description string   `json:"description"` // 技能描述
	Trigger     string   `json:"trigger"`     // 触发场景
	Confidence  float64  `json:"confidence"`  // 置信度 (0.0-1.0)
	Evidence    []string `json:"evidence"`    // 证据原文
}

// SkillPlan 技能规划
// 包含创建技能所需的完整规划信息
type SkillPlan struct {
	Name         string         `json:"name"`          // 技能名称
	Description  string         `json:"description"`   // 技能描述
	Instructions string         `json:"instructions"`  // 核心指令
	Tags         []string       `json:"tags"`          // 标签
	AllowedTools []string       `json:"allowed_tools"` // 允许使用的工具
	Examples     []string       `json:"examples"`      // 使用示例
	Metadata     map[string]any `json:"metadata"`      // 元数据
}

// DuplicateCheckResult 重复检查结果
type DuplicateCheckResult struct {
	IsDuplicate bool              `json:"is_duplicate"` // 是否重复
	Reason      string            `json:"reason"`       // 原因说明
	Existing    *skill.SkillState `json:"existing"`     // 已存在的技能
	Similarity  float64           `json:"similarity"`   // 相似度
}

// DiscoveryResult 发现任务执行结果
type DiscoveryResult struct {
	TaskID        string             `json:"task_id"`        // 任务ID
	StartTime     time.Time          `json:"start_time"`     // 开始时间
	EndTime       time.Time          `json:"end_time"`       // 结束时间
	Duration      int64              `json:"duration_ms"`    // 执行耗时(毫秒)
	CreatedSkills []string           `json:"created_skills"` // 创建的技能列表
	SkippedSkills []SkippedSkillInfo `json:"skipped_skills"` // 跳过的技能列表
	Errors        []string           `json:"errors"`         // 错误信息
}

// SkippedSkillInfo 被跳过的技能信息
type SkippedSkillInfo struct {
	Name   string `json:"name"`   // 技能名称
	Reason string `json:"reason"` // 跳过原因
}

// Conversation 对话记录
type Conversation struct {
	ID        string
	Content   string
	Timestamp time.Time
}

// SkillCreatedMessage 技能创建消息
type SkillCreatedMessage struct {
	Type        string    `json:"type"`        // 消息类型
	Name        string    `json:"name"`        // 技能名称
	Description string    `json:"description"` // 技能描述
	CreatedAt   time.Time `json:"created_at"`  // 创建时间
	Source      string    `json:"source"`      // 来源（analyzer/creative）
}

// DiscoveryCompletedMessage 发现任务完成消息
type DiscoveryCompletedMessage struct {
	Type          string   `json:"type"`           // 消息类型
	TaskID        string   `json:"task_id"`        // 任务ID
	CreatedCount  int      `json:"created_count"`  // 创建数量
	SkippedCount  int      `json:"skipped_count"`  // 跳过数量
	CreatedSkills []string `json:"created_skills"` // 创建的技能列表
}

// WebSocketMessageType WebSocket 消息类型
type WebSocketMessageType string

const (
	// MessageTypeSkillCreated 技能创建消息
	MessageTypeSkillCreated WebSocketMessageType = "skill_created"
	// MessageTypeDiscoveryCompleted 发现任务完成消息
	MessageTypeDiscoveryCompleted WebSocketMessageType = "discovery_completed"
)
