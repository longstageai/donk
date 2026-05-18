// types 多Agent系统类型定义
// 独立的类型包，避免循环导入
package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskStatus 任务状态枚举
type TaskStatus string

const (
	StatusCreated       TaskStatus = "CREATED"        // 任务已创建
	StatusPlanned       TaskStatus = "PLANNED"        // 规划已完成
	StatusPlanReviewing TaskStatus = "PLAN_REVIEWING" // 规划审查中
	StatusExecuting     TaskStatus = "EXECUTING"      // 执行中
	StatusReviewing     TaskStatus = "REVIEWING"      // 成果审查中
	StatusCompleted     TaskStatus = "COMPLETED"      // 任务已完成
	StatusFailed        TaskStatus = "FAILED"         // 任务失败
)

// TodoStatus 待办事项状态枚举
type TodoStatus string

const (
	TodoPending TodoStatus = "pending" // 待执行
	TodoDoing   TodoStatus = "doing"   // 执行中
	TodoDone    TodoStatus = "done"    // 已完成
	TodoFailed  TodoStatus = "failed"  // 失败
)

// TaskContext 多Agent任务上下文
// 贯穿整个任务生命周期，在所有Agent间传递
type TaskContext struct {
	TaskID    string     `json:"taskId"`    // 任务唯一ID
	CoreTheme string     `json:"coreTheme"` // 核心主题（如"让用户感受到温暖"）
	Status    TaskStatus `json:"status"`    // 当前状态
	CreatedAt time.Time  `json:"createdAt"` // 创建时间
	UpdatedAt time.Time  `json:"updatedAt"` // 更新时间

	// 任务信息（由任务生成Agent填充）
	Task *TaskInfo `json:"task"`

	// 规划信息（由任务规划Agent填充）
	Plan []*PlanStep `json:"plan"`

	// 规划审查结果（由规划审查Agent填充）
	PlanReview *ReviewResult `json:"planReview"`

	// 待办事项（由任务执行Agent维护）
	Todos []*TodoItem `json:"todos"`

	// 执行审查结果（由任务审查Agent填充）
	ExecutionReview *ReviewResult `json:"executionReview"`

	// 任务输出（由任务执行Agent填充）
	Output *TaskOutput `json:"output"`

	// Token使用统计
	TokenUsage *TaskTokenUsage `json:"tokenUsage"`

	// 是否已交付
	Delivered bool `json:"delivered"`

	// 用户画像（用于个性化）
	UserProfile *UserProfile `json:"userProfile"`

	// 对话历史记录（用于分析用户需求）
	Conversations []*Conversation `json:"conversations"`
}

// Conversation 对话记录
// 用于记录用户与系统的对话历史
type Conversation struct {
	ID        string    `json:"id"`        // 对话ID
	Content   string    `json:"content"`   // 对话内容
	Timestamp time.Time `json:"timestamp"` // 对话时间
	Role      string    `json:"role"`      // 角色：user/assistant/system
}

// NewTaskContext 创建新的任务上下文
func NewTaskContext(coreTheme string) *TaskContext {
	now := time.Now()
	return &TaskContext{
		TaskID:    generateTaskID(),
		CoreTheme: coreTheme,
		Status:    StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
		Task:      &TaskInfo{},
		Plan:      make([]*PlanStep, 0),
		PlanReview: &ReviewResult{
			DimensionScores: make(map[string]float64),
			Suggestions:     make([]string, 0),
			Issues:          make([]ReviewIssue, 0),
		},
		Todos: make([]*TodoItem, 0),
		ExecutionReview: &ReviewResult{
			DimensionScores: make(map[string]float64),
			Suggestions:     make([]string, 0),
			Issues:          make([]ReviewIssue, 0),
		},
		Output:      &TaskOutput{},
		TokenUsage:  NewTaskTokenUsage(),
		Delivered:   false,
		UserProfile: &UserProfile{},
	}
}

// UpdateStatus 更新任务状态
func (tc *TaskContext) UpdateStatus(status TaskStatus) {
	tc.Status = status
	tc.UpdatedAt = time.Now()
}

// ToJSON 转换为JSON字符串
func (tc *TaskContext) ToJSON() string {
	data, _ := json.MarshalIndent(tc, "", "  ")
	return string(data)
}

// TaskInfo 任务基本信息
// 由任务生成Agent创建，描述要做什么
type TaskInfo struct {
	Theme           string   `json:"theme"`           // 任务主题，如birthday_card
	Title           string   `json:"title"`           // 任务标题
	Description     string   `json:"description"`     // 任务描述
	CoreThemeReason string   `json:"coreThemeReason"` // 为什么这个能达成核心主题
	CoreElements    []string `json:"coreElements"`    // 核心要素列表
}

// PlanStep 规划步骤
// 由任务规划Agent生成，描述如何执行任务
type PlanStep struct {
	Step         int      `json:"step"`         // 步骤序号
	Action       string   `json:"action"`       // 动作标识
	Description  string   `json:"description"`  // 步骤描述
	Tool         string   `json:"tool"`         // 使用的工具名称
	Input        []string `json:"input"`        // 输入参数列表
	Output       []string `json:"output"`       // 输出结果列表
	Dependencies []int    `json:"dependencies"` // 依赖的步骤序号
}

// TodoItem 待办事项
// 由任务执行Agent维护，跟踪执行状态
type TodoItem struct {
	Step        int        `json:"step"`        // 步骤序号
	Action      string     `json:"action"`      // 动作标识
	Status      TodoStatus `json:"status"`      // 状态
	Result      string     `json:"result"`      // 执行结果
	Error       string     `json:"error"`       // 错误信息
	StartedAt   *time.Time `json:"startedAt"`   // 开始时间
	CompletedAt *time.Time `json:"completedAt"` // 完成时间
}

// ReviewIssue 审查发现的问题
type ReviewIssue struct {
	Dimension   string `json:"dimension"`   // 问题维度
	Description string `json:"description"` // 问题描述
	Suggestion  string `json:"suggestion"`  // 改进建议
}

// ReviewResult 审查结果
// 由规划审查Agent或任务审查Agent生成
type ReviewResult struct {
	Score           float64            `json:"score"`           // 总分
	Passed          bool               `json:"passed"`          // 是否通过
	DimensionScores map[string]float64 `json:"dimensionScores"` // 各维度得分
	Feedback        string             `json:"feedback"`        // 评价反馈
	Suggestions     []string           `json:"suggestions"`     // 改进建议
	Issues          []ReviewIssue      `json:"issues"`          // 问题列表
	Attempt         int                `json:"attempt"`         // 第几次尝试
}

// TaskOutput 任务输出
// 由任务执行Agent生成，包含最终成果
type TaskOutput struct {
	CardImage string `json:"cardImage"` // 贺卡图片URL或内容
	Blessing  string `json:"blessing"`  // 祝福语文本
	Message   string `json:"message"`   // 消息内容
}

// UserProfile 用户画像
// 用于个性化内容生成
type UserProfile struct {
	UserID             string               `json:"userId"`             // 用户ID
	Name               string               `json:"name"`               // 姓名
	Gender             string               `json:"gender"`             // 性别
	Age                int                  `json:"age"`                // 年龄
	Occupation         string               `json:"occupation"`         // 职业
	Hobbies            []string             `json:"hobbies"`            // 兴趣爱好
	Preferences        map[string]string    `json:"preferences"`        // 偏好设置
	InteractionHistory []*InteractionRecord `json:"interactionHistory"` // 互动历史
	RawContent         string               `json:"rawContent"`         // 原始画像文本（ToPrompt生成）
}

// InteractionRecord 互动记录
type InteractionRecord struct {
	Timestamp time.Time `json:"timestamp"` // 时间戳
	Type      string    `json:"type"`      // 发送/接收
	Content   string    `json:"content"`   // 内容
	Feedback  string    `json:"feedback"`  // 用户反馈
}

// Agent 接口定义
// 所有Agent必须实现此接口
type Agent interface {
	// GetName 获取Agent名称
	GetName() string
	// GetDescription 获取Agent描述
	GetDescription() string
	// Process 处理任务上下文
	Process(ctx *TaskContext) error
}

// LLMClient LLM客户端接口
// 用于与LLM服务交互
type LLMClient interface {
	// Chat 发送聊天请求
	Chat(messages []Message, tools []ToolDefinition) (*LLMResponse, error)
	// ChatStream 流式聊天请求
	ChatStream(messages []Message, tools []ToolDefinition, callback StreamCallback) error
}

// Message 消息结构
type Message struct {
	Role       string     `json:"role"`       // system/user/assistant
	Content    string     `json:"content"`    // 消息内容
	ToolCalls  []ToolCall `json:"toolCalls"`  // 工具调用
	ToolCallID string     `json:"toolCallId"` // 工具调用ID
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string `json:"id"`   // 调用ID
	Type     string `json:"type"` // 类型
	Function struct {
		Name      string `json:"name"`      // 函数名
		Arguments string `json:"arguments"` // 参数(JSON字符串)
	} `json:"function"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string       `json:"type"`     // 类型，通常为"function"
	Function FunctionInfo `json:"function"` // 函数信息
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name        string                 `json:"name"`        // 函数名
	Description string                 `json:"description"` // 函数描述
	Parameters  map[string]interface{} `json:"parameters"`  // 参数定义(JSON Schema)
}

// LLMResponse LLM响应
type LLMResponse struct {
	Content   string     `json:"content"`   // 响应内容
	Reasoning string     `json:"reasoning"` // 思考过程
	ToolCalls []ToolCall `json:"toolCalls"` // 工具调用请求
	Usage     TokenUsage `json:"usage"`     // Token使用统计
}

// StreamCallback 流式回调函数
type StreamCallback func(chunk *StreamChunk)

// StreamChunk 流式数据块
type StreamChunk struct {
	Content   string `json:"content"`   // 内容增量
	Reasoning string `json:"reasoning"` // 思考过程增量
	Done      bool   `json:"done"`      // 是否结束
}

// TokenUsage Token使用统计
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`     // 输入token数
	CompletionTokens int `json:"completionTokens"` // 输出token数
	TotalTokens      int `json:"totalTokens"`      // 总token数
}

// TaskTokenUsage 任务Token使用统计
type TaskTokenUsage struct {
	Generation TokenUsage `json:"generation"` // 任务生成Agent
	Planning   TokenUsage `json:"planning"`   // 任务规划Agent
	PlanReview TokenUsage `json:"planReview"` // 规划审查Agent
	Execution  TokenUsage `json:"execution"`  // 任务执行Agent
	TaskReview TokenUsage `json:"taskReview"` // 任务审查Agent
	Completion TokenUsage `json:"completion"` // 任务结束Agent
	Total      int        `json:"total"`      // 任务总计
}

// NewTaskTokenUsage 创建新的任务Token使用统计
func NewTaskTokenUsage() *TaskTokenUsage {
	return &TaskTokenUsage{}
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
