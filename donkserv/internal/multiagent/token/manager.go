// token Token管理模块
// 负责统计Token消耗，使用setting模块的每日限额配置
package token

import (
	"fmt"
	"sync"

	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/internal/setting"
)

// Manager Token管理器
// 只保留系统级每日限额检查，从setting模块读取配置
type Manager struct {
	mu sync.RWMutex

	// 当前任务使用统计（仅用于展示，不做限制）
	currentTask *types.TaskTokenUsage

	// 是否已超限
	limitExceeded bool
}

// NewManager 创建Token管理器
func NewManager() *Manager {
	return &Manager{
		currentTask:   types.NewTaskTokenUsage(),
		limitExceeded: false,
	}
}

// ResetTaskUsage 重置当前任务使用统计
func (m *Manager) ResetTaskUsage() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTask = types.NewTaskTokenUsage()
	m.limitExceeded = false
}

// RecordUsage 记录Token使用
// agentType: generation/planning/planReview/execution/taskReview/completion
func (m *Manager) RecordUsage(agentType string, usage types.TokenUsage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch agentType {
	case "generation":
		m.currentTask.Generation = addUsage(m.currentTask.Generation, usage)
	case "planning":
		m.currentTask.Planning = addUsage(m.currentTask.Planning, usage)
	case "planReview":
		m.currentTask.PlanReview = addUsage(m.currentTask.PlanReview, usage)
	case "execution":
		m.currentTask.Execution = addUsage(m.currentTask.Execution, usage)
	case "taskReview":
		m.currentTask.TaskReview = addUsage(m.currentTask.TaskReview, usage)
	case "completion":
		m.currentTask.Completion = addUsage(m.currentTask.Completion, usage)
	}

	// 更新总计
	m.currentTask.Total = m.currentTask.Generation.TotalTokens +
		m.currentTask.Planning.TotalTokens +
		m.currentTask.PlanReview.TotalTokens +
		m.currentTask.Execution.TotalTokens +
		m.currentTask.TaskReview.TotalTokens +
		m.currentTask.Completion.TotalTokens
}

// addUsage 累加Token使用
func addUsage(a, b types.TokenUsage) types.TokenUsage {
	return types.TokenUsage{
		PromptTokens:     a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
		TotalTokens:      a.TotalTokens + b.TotalTokens,
	}
}

// IsLimitExceeded 检查是否已超出限制
// 从setting模块读取每日限额配置
func (m *Manager) IsLimitExceeded() bool {
	provider := setting.GetProvider()
	if provider == nil {
		return false
	}

	cfg, err := provider.GetConfig()
	if err != nil || cfg == nil {
		return false
	}

	// -1 或 0 表示不限制
	if cfg.AgentDailyTokenLimit <= 0 {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.limitExceeded
}

// SetLimitExceeded 设置超限标志
// 由外部token统计器调用（如internal/token/stats.go）
func (m *Manager) SetLimitExceeded(exceeded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limitExceeded = exceeded
}

// GetTaskUsage 获取当前任务使用统计
func (m *Manager) GetTaskUsage() *types.TaskTokenUsage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本
	return &types.TaskTokenUsage{
		Generation: m.currentTask.Generation,
		Planning:   m.currentTask.Planning,
		PlanReview: m.currentTask.PlanReview,
		Execution:  m.currentTask.Execution,
		TaskReview: m.currentTask.TaskReview,
		Completion: m.currentTask.Completion,
		Total:      m.currentTask.Total,
	}
}

// GenerateReport 生成使用报告
func (m *Manager) GenerateReport() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取每日限额
	dailyLimit := -1
	provider := setting.GetProvider()
	if provider != nil {
		if cfg, err := provider.GetConfig(); err == nil && cfg != nil {
			dailyLimit = cfg.AgentDailyTokenLimit
		}
	}

	limitStr := "不限"
	if dailyLimit > 0 {
		limitStr = fmt.Sprintf("%d", dailyLimit)
	}

	return fmt.Sprintf(`
========================================
Token消耗统计报告
========================================
任务生成Agent:   %d tokens
任务规划Agent:   %d tokens
规划审查Agent:   %d tokens
任务执行Agent:   %d tokens
任务审查Agent:   %d tokens
任务结束Agent:   %d tokens
----------------------------------------
当前任务总计:   %d tokens
每日限额:       %s
========================================
`,
		m.currentTask.Generation.TotalTokens,
		m.currentTask.Planning.TotalTokens,
		m.currentTask.PlanReview.TotalTokens,
		m.currentTask.Execution.TotalTokens,
		m.currentTask.TaskReview.TotalTokens,
		m.currentTask.Completion.TotalTokens,
		m.currentTask.Total,
		limitStr,
	)
}

// GetDailyLimit 获取每日限额
// 从setting模块读取
func (m *Manager) GetDailyLimit() int {
	provider := setting.GetProvider()
	if provider == nil {
		return -1
	}

	cfg, err := provider.GetConfig()
	if err != nil || cfg == nil {
		return -1
	}

	return cfg.AgentDailyTokenLimit
}
