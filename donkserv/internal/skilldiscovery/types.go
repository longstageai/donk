// skilldiscovery 技能自动发现模块
// 负责从对话历史中分析用户需求，自动创建新的 Skill
package skilldiscovery

import (
	"time"

	"github.com/longstageai/donk/donk/internal/skilldiscovery/agents"
)

// SkillCandidate 技能候选
// 表示从对话中分析出的潜在技能需求
type SkillCandidate = agents.SkillCandidate

// SkillPlan 技能规划
// 包含创建技能所需的完整规划信息
type SkillPlan = agents.SkillPlan

// DuplicateCheckResult 重复检查结果
type DuplicateCheckResult = agents.DuplicateCheckResult

// DiscoveryResult 发现任务执行结果
type DiscoveryResult = agents.DiscoveryResult

// SkippedSkillInfo 被跳过的技能信息
type SkippedSkillInfo = agents.SkippedSkillInfo

// SkillCreatedMessage 技能创建消息
type SkillCreatedMessage = agents.SkillCreatedMessage

// DiscoveryCompletedMessage 发现任务完成消息
type DiscoveryCompletedMessage = agents.DiscoveryCompletedMessage

// WebSocketMessageType WebSocket 消息类型
type WebSocketMessageType = agents.WebSocketMessageType

const (
	// MessageTypeSkillCreated 技能创建消息
	MessageTypeSkillCreated = agents.MessageTypeSkillCreated
	// MessageTypeDiscoveryCompleted 发现任务完成消息
	MessageTypeDiscoveryCompleted = agents.MessageTypeDiscoveryCompleted
)

// Config 技能发现配置
type Config struct {
	// Interval 执行间隔（默认2小时）
	Interval time.Duration

	// SimilarityThreshold 重复检测相似度阈值（默认0.6）
	SimilarityThreshold float64

	// MaxSkillsPerRun 每次最大创建技能数（默认5）
	MaxSkillsPerRun int

	// CreativeSkillsCount 无需求时生成创意技能数量（默认1）
	CreativeSkillsCount int

	// EnableNotification 是否启用通知（默认true）
	EnableNotification bool

	// ConversationLookback 对话回溯时间（默认2小时）
	ConversationLookback time.Duration

	// SkillsDir 技能存储目录
	SkillsDir string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Interval:             2 * time.Hour,
		SimilarityThreshold:  0.6,
		MaxSkillsPerRun:      1,
		CreativeSkillsCount:  1,
		EnableNotification:   true,
		ConversationLookback: 2 * time.Hour,
	}
}
