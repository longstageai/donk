package scheduler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/robfig/cron/v3"
)

// TaskType 任务类型定义
type TaskType string

const (
	TaskTypeCron  TaskType = "cron"  // 定时循环任务
	TaskTypeDelay TaskType = "delay" // 延迟执行任务
	TaskTypeOnce  TaskType = "once"  // 单次执行任务
)

// ExecutorType 执行器类型定义
type ExecutorType string

const (
	ExecutorScript ExecutorType = "script" // 脚本/命令执行器
	ExecutorAPI    ExecutorType = "api"    // HTTP API 执行器
	ExecutorAgent  ExecutorType = "agent"  // LLM Agent 执行器
)

// TaskStatus 任务状态定义
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 等待执行
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusDone      TaskStatus = "done"      // 执行完成
	TaskStatusFailed    TaskStatus = "failed"    // 执行失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// TaskConfig 任务执行配置，用于存储不同执行器所需的配置参数
type TaskConfig map[string]interface{}

func (c TaskConfig) GetString(key string) string {
	if v, ok := c[key].(string); ok {
		return v
	}
	return ""
}

func (c TaskConfig) GetInt(key string) int {
	switch v := c[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func (c TaskConfig) GetBool(key string) bool {
	if v, ok := c[key].(bool); ok {
		return v
	}
	return false
}

func (c TaskConfig) GetMap(key string) map[string]interface{} {
	if v, ok := c[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// TaskResult 任务执行结果
type TaskResult struct {
	Output   string `json:"output"`    // 执行输出
	Error    string `json:"error"`     // 错误信息
	ExitCode int    `json:"exit_code"` // 退出码
	DoneAt   int64  `json:"done_at"`   // 完成时间戳
	Duration int64  `json:"duration"`  // 执行耗时(毫秒)
}

// Task 定时任务定义
type Task struct {
	ID string `json:"id" gorm:"primaryKey"` // 任务唯一标识(UUID)

	// 调度配置
	Name      string   `json:"name"`        // 任务名称
	TaskType  TaskType `json:"task_type"`   // 任务类型 (cron/delay/once)
	Schedule  string   `json:"schedule"`    // 调度表达式 (cron表达式/延迟时间/时间戳)
	NextRunAt int64    `json:"next_run_at"` // 下次执行时间戳
	LastRunAt int64    `json:"last_run_at"` // 上次执行时间戳

	// 执行配置
	Executor ExecutorType `json:"executor"` // 执行器类型 (script/api/agent)
	Config   TaskConfig   `json:"config"`   // 执行器配置参数

	// 状态与结果
	Status     TaskStatus  `json:"status"`      // 当前状态
	Result     *TaskResult `json:"result"`      // 执行结果
	Retries    int         `json:"retries"`     // 当前重试次数
	MaxRetries int         `json:"max_retries"` // 最大重试次数

	// 元数据
	CreatedBy string `json:"created_by"` // 创建者标识
	CreatedAt int64  `json:"created_at"` // 创建时间
	UpdatedAt int64  `json:"updated_at"` // 更新时间
}

// BeforeCreate 创建前设置默认值
func (t *Task) BeforeCreate() error {
	now := time.Now().Unix()
	if t.CreatedAt == 0 {
		t.CreatedAt = now
	}
	if t.UpdatedAt == 0 {
		t.UpdatedAt = now
	}
	if t.Status == "" {
		t.Status = TaskStatusPending
	}
	if t.MaxRetries == 0 {
		t.MaxRetries = 3
	}
	if t.NextRunAt == 0 {
		t.NextRunAt = t.CalculateNextRunAt()
	}
	return nil
}

// BeforeUpdate 更新前设置更新时间
func (t *Task) BeforeUpdate() error {
	t.UpdatedAt = time.Now().Unix()
	return nil
}

// CalculateNextRunAt 根据任务类型和调度表达式计算下次执行时间
// 返回下次执行的Unix时间戳
func (t *Task) CalculateNextRunAt() int64 {
	now := time.Now().Unix()

	switch t.TaskType {
	case TaskTypeCron:
		// 解析 cron 表达式并计算下次执行时间
		return t.calculateCronNextRun()
	case TaskTypeDelay:
		// 延迟任务：当前时间 + 延迟时间
		delay, err := time.ParseDuration(t.Schedule)
		if err != nil {
			return now
		}
		return now + int64(delay.Seconds())
	case TaskTypeOnce:
		// 单次任务：解析时间戳
		if t.Schedule == "" {
			return now
		}
		// 支持直接的时间戳或 RFC3339 格式
		if ts, err := time.Parse(time.RFC3339, t.Schedule); err == nil {
			return ts.Unix()
		}
		// 尝试作为时间戳解析
		if ts, err := strconv.ParseInt(t.Schedule, 10, 64); err == nil {
			return ts
		}
		return now
	default:
		return now
	}
}

// calculateCronNextRun 计算 cron 表达式的下次执行时间
// 使用 robfig/cron 库进行解析
func (t *Task) calculateCronNextRun() int64 {
	// 验证 cron 表达式格式
	if err := ValidateCronExpression(t.Schedule); err != nil {
		logger.Warn("cron 表达式无效", map[string]interface{}{"schedule": t.Schedule, "error": err.Error()})
		return 0
	}

	schedule, err := cron.ParseStandard(t.Schedule)
	if err != nil {
		logger.Warn("cron 表达式解析失败", map[string]interface{}{"schedule": t.Schedule, "error": err.Error()})
		return 0
	}
	return schedule.Next(time.Now()).Unix()
}

// ValidateCronExpression 验证 cron 表达式是否有效
func ValidateCronExpression(expr string) error {
	if expr == "" {
		return fmt.Errorf("cron 表达式不能为空")
	}

	// 尝试解析，验证格式是否正确
	_, err := cron.ParseStandard(expr)
	if err != nil {
		return fmt.Errorf("无效的 cron 表达式: %v", err)
	}
	return nil
}

// IsRetryable 判断任务是否可重试
func (t *Task) IsRetryable() bool {
	if t.Status != TaskStatusFailed {
		return false
	}
	return t.Retries < t.MaxRetries
}

// CanExecute 判断任务是否可以执行
func (t *Task) CanExecute() bool {
	return t.Status == TaskStatusPending && t.NextRunAt <= time.Now().Unix()
}

// Clone 创建任务的深拷贝
func (t *Task) Clone() *Task {
	clone := *t
	if t.Result != nil {
		result := *t.Result
		clone.Result = &result
	}
	// Config 是 map，已经是指针拷贝，但为了安全也做一次
	if t.Config != nil {
		config := make(TaskConfig)
		for k, v := range t.Config {
			config[k] = v
		}
		clone.Config = config
	}
	return &clone
}

// MarshalJSON 自定义 JSON 序列化
func (t Task) MarshalJSON() ([]byte, error) {
	type Alias Task
	aux := struct {
		Alias
		ResultJSON string `json:"result,omitempty"`
	}{
		Alias: Alias(t),
	}
	if t.Result != nil {
		resultBytes, err := json.Marshal(t.Result)
		if err != nil {
			return nil, err
		}
		aux.ResultJSON = string(resultBytes)
	}
	return json.Marshal(aux)
}

// UnmarshalJSON 自定义 JSON 反序列化
func (t *Task) UnmarshalJSON(data []byte) error {
	type Alias Task
	aux := struct {
		Alias
		ResultJSON string `json:"result,omitempty"`
	}{
		Alias: Alias(*t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = Task(aux.Alias)
	if aux.ResultJSON != "" {
		var result TaskResult
		if err := json.Unmarshal([]byte(aux.ResultJSON), &result); err != nil {
			return err
		}
		t.Result = &result
	}
	return nil
}

// TaskRunStatus 任务执行记录状态定义
type TaskRunStatus string

const (
	TaskRunStatusRunning TaskRunStatus = "running" // 执行中
	TaskRunStatusDone    TaskRunStatus = "done"    // 执行成功
	TaskRunStatusFailed  TaskRunStatus = "failed"  // 执行失败
)

// TaskRun 任务执行记录，用于记录每次任务执行的详细信息
type TaskRun struct {
	ID       string `json:"id" gorm:"primaryKey"` // 执行记录唯一标识(UUID)
	TaskID   string `json:"task_id" gorm:"index"` // 关联的任务ID
	TaskName string `json:"task_name"`            // 任务名称(冗余存储，便于查询)

	// 执行器信息
	Executor ExecutorType `json:"executor"` // 执行器类型(script/api/agent)
	Input    string       `json:"input"`    // 输入参数(JSON格式存储)

	// 执行结果
	Status    TaskRunStatus `json:"status"`     // 执行状态(running/done/failed)
	StartTime int64         `json:"start_time"` // 开始执行时间戳
	EndTime   int64         `json:"end_time"`   // 结束执行时间戳
	Duration  int64         `json:"duration"`   // 执行时长(毫秒)
	Output    string        `json:"output"`     // 执行输出内容
	Error     string        `json:"error"`      // 错误信息(如有)
	ExitCode  int           `json:"exit_code"`  // 进程退出码

	// 重试信息
	RetryCount int `json:"retry_count"` // 当前重试次数

	// 元数据
	CreatedAt int64 `json:"created_at"` // 记录创建时间
	UpdatedAt int64 `json:"updated_at"` // 记录更新时间
}

// BeforeCreate 创建前设置默认值
func (r *TaskRun) BeforeCreate() error {
	now := time.Now().Unix()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = now
	}
	if r.Status == "" {
		r.Status = TaskRunStatusRunning
	}
	return nil
}

// BeforeUpdate 更新前设置更新时间
func (r *TaskRun) BeforeUpdate() error {
	r.UpdatedAt = time.Now().Unix()
	return nil
}

// CalculateDuration 计算执行时长(毫秒)
func (r *TaskRun) CalculateDuration() int64 {
	if r.EndTime > 0 && r.StartTime > 0 {
		return r.EndTime - r.StartTime
	}
	return 0
}
