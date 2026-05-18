// background 后台Agent模块
package background

// TaskCompleteMessage 任务完成消息
// 用于WebSocket推送任务执行结果
type TaskCompleteMessage struct {
	Type       string `json:"type"`        // 消息类型：background_task_complete / background_task_error
	RunnerID   string `json:"runner_id"`   // Runner唯一标识
	RunnerName string `json:"runner_name"` // Runner显示名称
	Status     string `json:"status"`      // 状态：success / failed
	Output     string `json:"output"`      // 执行输出（成功时）
	ErrorType  string `json:"error_type"`  // 错误类型（失败时）
	Error      string `json:"error"`       // 错误信息（失败时）
	Duration   int64  `json:"duration"`    // 执行耗时（毫秒）
	Tokens     int    `json:"tokens"`      // Token消耗
	Timestamp  int64  `json:"timestamp"`   // 时间戳（秒）

	// 扩展字段 - 详细执行信息
	Iterations       int `json:"iterations,omitempty"`        // 实际迭代次数
	PromptTokens     int `json:"prompt_tokens,omitempty"`     // 输入Token数
	CompletionTokens int `json:"completion_tokens,omitempty"` // 输出Token数
	TotalTokens      int `json:"total_tokens,omitempty"`      // 总Token数
}

// RunnerStatsMessage Runner统计消息
// 用于查询Runner运行状态
type RunnerStatsMessage struct {
	Type         string `json:"type"`          // 消息类型：background_runner_stats
	RunnerID     string `json:"runner_id"`     // Runner唯一标识
	RunnerName   string `json:"runner_name"`   // Runner显示名称
	RunCount     int    `json:"run_count"`     // 总执行次数
	SuccessCount int    `json:"success_count"` // 成功次数
	FailCount    int    `json:"fail_count"`    // 失败次数
	TotalTokens  int    `json:"total_tokens"`  // 累计Token消耗
	LastRunAt    int64  `json:"last_run_at"`   // 上次执行时间戳
	IsRunning    bool   `json:"is_running"`    // 是否正在运行
}

// TokenBudgetMessage Token预算消息
// 用于通知Token预算状态
type TokenBudgetMessage struct {
	Type       string `json:"type"`        // 消息类型：background_token_budget
	RunnerID   string `json:"runner_id"`   // Runner唯一标识
	RunnerName string `json:"runner_name"` // Runner显示名称
	DailyLimit int    `json:"daily_limit"` // 每日Token限额
	TodayUsage int    `json:"today_usage"` // 今日已使用Token数
	Remaining  int    `json:"remaining"`   // 剩余可用Token数
	IsExceeded bool   `json:"is_exceeded"` // 是否已超出预算
	Timestamp  int64  `json:"timestamp"`   // 时间戳（秒）
}

// 消息类型常量
const (
	// MessageTypeTaskComplete 任务完成消息
	MessageTypeTaskComplete = "background_task_complete"
	// MessageTypeTaskError 任务错误消息
	MessageTypeTaskError = "background_task_error"
	// MessageTypeRunnerStats Runner统计消息
	MessageTypeRunnerStats = "background_runner_stats"
	// MessageTypeTokenBudget Token预算消息
	MessageTypeTokenBudget = "background_token_budget"
)
