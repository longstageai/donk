package memory

import (
	"fmt"
	"sync"
)

// ExecutionStep 执行步骤
// 表示一次工具调用的完整信息
type ExecutionStep struct {
	Round    int    `json:"round"`    // 轮次编号
	Action   string `json:"action"`   // 工具/动作名称
	Input    string `json:"input"`    // 输入参数
	Output   string `json:"output"`   // 输出结果
	Success  bool   `json:"success"`  // 是否执行成功
	Duration int64  `json:"duration"` // 执行耗时(毫秒)
}

// ExecutionTrace 执行轨迹
// 用于记录和追踪Agent的执行过程，与对话消息分离存储
type ExecutionTrace struct {
	mu    sync.RWMutex
	steps []ExecutionStep
}

// NewExecutionTrace 创建新的执行轨迹记录器
func NewExecutionTrace() *ExecutionTrace {
	return &ExecutionTrace{
		steps: make([]ExecutionStep, 0),
	}
}

// Add 添加一个执行步骤
func (e *ExecutionTrace) Add(step ExecutionStep) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.steps = append(e.steps, step)
}

// GetSteps 获取所有执行步骤
func (e *ExecutionTrace) GetSteps() []ExecutionStep {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]ExecutionStep, len(e.steps))
	copy(result, e.steps)
	return result
}

// GetLastStep 获取最后一个执行步骤
func (e *ExecutionTrace) GetLastStep() *ExecutionStep {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.steps) == 0 {
		return nil
	}
	return &e.steps[len(e.steps)-1]
}

// Clear 清空执行轨迹
func (e *ExecutionTrace) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.steps = make([]ExecutionStep, 0)
}

// GetSummary 获取执行轨迹摘要
// 返回格式化的执行记录字符串，用于展示或调试
func (e *ExecutionTrace) GetSummary() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.steps) == 0 {
		return ""
	}

	summary := "## 执行记录\n"
	for _, step := range e.steps {
		status := "✓"
		if !step.Success {
			status = "✗"
		}
		summary += fmt.Sprintf("- %s %s (%.2fs)\n", status, step.Action, float64(step.Duration)/1000)
	}
	return summary
}
