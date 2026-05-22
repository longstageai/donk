package creative

import "time"

// ID 类型统一表示系统内 Session、Room、Event、AgentRun 等对象的唯一标识。
type ID string

// TriggerType 表示创建一次创意多 Agent 协作 Session 的触发来源。
type TriggerType string

const (
	TriggerSystemStarted        TriggerType = "SystemStarted"
	TriggerUserTriggered        TriggerType = "UserTriggered"
	TriggerContextChanged       TriggerType = "ContextChanged"
	TriggerTimerTriggered       TriggerType = "TimerTriggered"
	TriggerPreviousLoopComplete TriggerType = "PreviousLoopCompleted"
)

// Trigger 表示一次 Session 的启动请求。
type Trigger struct {
	Type    TriggerType // 触发类型
	Payload any         // 触发载荷数据
}

// SessionStatus 表示一次任务生命周期的业务状态。
type SessionStatus string

const (
	SessionRunning   SessionStatus = "running"
	SessionBlocked   SessionStatus = "blocked"
	SessionCompleted SessionStatus = "completed"
	SessionFailed    SessionStatus = "failed"
	SessionCancelled SessionStatus = "cancelled"
)

// LoopStatus 表示事件循环本身的运行状态，与 SessionStatus 分离。
type LoopStatus string

const (
	LoopCreated  LoopStatus = "created"
	LoopStarting LoopStatus = "starting"
	LoopRunning  LoopStatus = "running"
	LoopPausing  LoopStatus = "pausing"
	LoopPaused   LoopStatus = "paused"
	LoopStopping LoopStatus = "stopping"
	LoopStopped  LoopStatus = "stopped"
	LoopFailed   LoopStatus = "failed"
)

// Phase 表示当前 Session 所处的全局阶段。
type Phase string

const (
	PhaseIdle            Phase = "idle"
	PhaseGoalGeneration  Phase = "goal_generation"
	PhaseGoalReview      Phase = "goal_review"
	PhaseGoalConvergence Phase = "goal_convergence"
	PhasePlanning        Phase = "planning"
	PhasePlanReview      Phase = "plan_review"
	PhaseExecution       Phase = "execution"
	PhaseResultReview    Phase = "result_review"
	PhaseDelivery        Phase = "delivery"
	PhaseCompleted       Phase = "completed"
	PhaseBlocked         Phase = "blocked"
)

// EventType 表示事件流中的事件类型。
type EventType string

const (
	EventSessionStarted   EventType = "SessionStarted"
	EventSessionCompleted EventType = "SessionCompleted"
	EventSessionFailed    EventType = "SessionFailed"
	EventSessionBlocked   EventType = "SessionBlocked"
	EventSessionCancelled EventType = "SessionCancelled"

	EventGoalRequested             EventType = "GoalRequested"
	EventCandidateGoalCreated      EventType = "CandidateGoalCreated"
	EventGoalDedupRequested        EventType = "GoalDedupRequested"
	EventGoalDedupPassed           EventType = "GoalDedupPassed"
	EventGoalDedupRejected         EventType = "GoalDedupRejected"
	EventGoalValueReviewRequested  EventType = "GoalValueReviewRequested"
	EventGoalValueReviewPassed     EventType = "GoalValueReviewPassed"
	EventGoalValueReviewRejected   EventType = "GoalValueReviewRejected"
	EventGoalFeasibilityRequested  EventType = "GoalFeasibilityRequested"
	EventGoalFeasibilityPassed     EventType = "GoalFeasibilityPassed"
	EventGoalFeasibilityRejected   EventType = "GoalFeasibilityRejected"
	EventGoalConvergenceRequested  EventType = "GoalConvergenceRequested"
	EventFinalGoalCreated          EventType = "FinalGoalCreated"
	EventGoalRegenerationRequested EventType = "GoalRegenerationRequested"
	EventGoalRefinementRequested   EventType = "GoalRefinementRequested"

	EventPlanRequested                   EventType = "PlanRequested"
	EventPlanCreated                     EventType = "PlanCreated"
	EventPlanReviewRequested             EventType = "PlanReviewRequested"
	EventPlanReviewPassed                EventType = "PlanReviewPassed"
	EventPlanReviewRejected              EventType = "PlanReviewRejected"
	EventPlanRevisionRequested           EventType = "PlanRevisionRequested"
	EventGoalFeasibilityRecheckRequested EventType = "GoalFeasibilityRecheckRequested"

	EventExecutionRequested         EventType = "ExecutionRequested"
	EventExecutionStepStarted       EventType = "ExecutionStepStarted"
	EventExecutionStepCompleted     EventType = "ExecutionStepCompleted"
	EventExecutionCompleted         EventType = "ExecutionCompleted"
	EventExecutionRevisionRequested EventType = "ExecutionRevisionRequested"
	EventResultReviewRequested      EventType = "ResultReviewRequested"
	EventResultReviewPassed         EventType = "ResultReviewPassed"
	EventResultReviewRejected       EventType = "ResultReviewRejected"
	EventDeliveryRequested          EventType = "DeliveryRequested"
	EventDeliveryCompleted          EventType = "DeliveryCompleted"
	EventLoopCompleted              EventType = "LoopCompleted"

	EventToolCallRequested EventType = "ToolCallRequested"
	EventToolCallStarted   EventType = "ToolCallStarted"
	EventToolCallCompleted EventType = "ToolCallCompleted"
	EventToolCallFailed    EventType = "ToolCallFailed"

	EventTokenUsageRecorded              EventType = "TokenUsageRecorded"
	EventTokenBudgetWarning              EventType = "TokenBudgetWarning"
	EventTokenBudgetExceeded             EventType = "TokenBudgetExceeded"
	EventTokenContextCompactionRequested EventType = "TokenContextCompactionRequested"
	EventTokenModelDowngradeRequested    EventType = "TokenModelDowngradeRequested"
	EventTokenApprovalRequested          EventType = "TokenApprovalRequested"
	EventTokenSessionPauseRequested      EventType = "TokenSessionPauseRequested"
	EventTokenSessionStopRequested       EventType = "TokenSessionStopRequested"

	EventLoopStartRequested  EventType = "LoopStartRequested"
	EventLoopStarted         EventType = "LoopStarted"
	EventLoopPauseRequested  EventType = "LoopPauseRequested"
	EventLoopPaused          EventType = "LoopPaused"
	EventLoopResumeRequested EventType = "LoopResumeRequested"
	EventLoopResumed         EventType = "LoopResumed"
	EventLoopStopRequested   EventType = "LoopStopRequested"
	EventLoopStopped         EventType = "LoopStopped"
	EventLoopFailed          EventType = "LoopFailed"
)

// EventStatus 表示单个事件在事件循环中的处理状态。
type EventStatus string

const (
	EventPending    EventStatus = "pending"
	EventClaimed    EventStatus = "claimed"
	EventProcessing EventStatus = "processing"
	EventSucceeded  EventStatus = "succeeded"
	EventRejected   EventStatus = "rejected"
	EventFailed     EventStatus = "failed"
	EventSkipped    EventStatus = "skipped"
	EventExpired    EventStatus = "expired"
)

// DispatchMode 表示事件分发方式。
type DispatchMode string

const (
	DispatchExclusive   DispatchMode = "exclusive"
	DispatchCompetitive DispatchMode = "competitive"
	DispatchBroadcast   DispatchMode = "broadcast"
)

// AgentRole 表示 Agent 在协作房间中的角色。
type AgentRole string

const (
	RoleGoalCreative    AgentRole = "goal_creative"
	RoleGoalDedup       AgentRole = "goal_dedup"
	RoleGoalValueReview AgentRole = "goal_value_review"
	RoleGoalFeasibility AgentRole = "goal_feasibility"
	RoleGoalConvergence AgentRole = "goal_convergence"
	RolePlanning        AgentRole = "planning"
	RolePlanReview      AgentRole = "plan_review"
	RoleExecution       AgentRole = "execution"
	RoleResultReview    AgentRole = "result_review"
	RoleDelivery        AgentRole = "delivery"
)

// MessageRole 表示消息在群聊上下文中的来源角色。
type MessageRole string

const (
	MessageRoleSystem MessageRole = "system"
	MessageRoleAgent  MessageRole = "agent"
	MessageRoleTool   MessageRole = "tool"
)

// ArtifactType 表示结构化产物类型。
type ArtifactType string

const (
	ArtifactCandidateGoal       ArtifactType = "CandidateGoal"
	ArtifactDedupReview         ArtifactType = "DedupReview"
	ArtifactValueReview         ArtifactType = "ValueReview"
	ArtifactFeasibilityReview   ArtifactType = "FeasibilityReview"
	ArtifactFinalExecutableGoal ArtifactType = "FinalExecutableGoal"
	ArtifactExecutablePlan      ArtifactType = "ExecutablePlan"
	ArtifactPlanReview          ArtifactType = "PlanReview"
	ArtifactExecutionResult     ArtifactType = "ExecutionResult"
	ArtifactResultReview        ArtifactType = "ResultReview"
	ArtifactFinalDelivery       ArtifactType = "FinalDelivery"
)

// Session 是一次从触发到交付、失败、阻塞或取消的完整多 Agent 协作生命周期。
type Session struct {
	ID             ID                // Session唯一标识
	TriggerType    TriggerType       // 触发类型
	TriggerPayload any               // 触发载荷数据
	Status         SessionStatus     // Session业务状态
	LoopStatus     LoopStatus        // 事件循环运行状态
	CurrentPhase   Phase             // 当前所处阶段
	RoomID         ID                // 关联的房间ID
	RootEventID    ID                // 根事件ID
	CurrentEventID ID                // 当前正在处理的事件ID
	FinalGoalID    ID                // 最终目标ID
	PlanID         ID                // 执行计划ID
	ExecutionID    ID                // 执行结果ID
	DeliveryID     ID                // 交付产物ID
	Tick           int               // 当前Tick计数
	FailureCounts  map[string]int    // 各阶段失败次数统计
	RetryCounts    map[string]int    // 各阶段重试次数统计
	TokenUsage     TokenUsageSummary // Token使用统计
	TokenBudget    TokenBudget       // Token预算配置
	StartedAt      time.Time         // 开始时间
	CompletedAt    *time.Time        // 完成时间（可为空）
}

// Room 是一个 Session 内部的 Agent 群聊协作空间。
type Room struct {
	ID        ID        // 房间唯一标识
	SessionID ID        // 关联的Session ID
	Topic     string    // 房间主题
	Status    string    // 房间状态
	Members   []ID      // 成员Agent ID列表
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
}

// Event 是驱动系统状态变化的机器协议。
type Event struct {
	ID            ID             // 事件唯一标识
	Type          EventType      // 事件类型
	RoomID        ID             // 关联的房间ID
	SessionID     ID             // 关联的Session ID
	CorrelationID ID             // 关联ID，用于追踪同一业务链路
	CausationID   ID             // 因果ID，指向导致本事件的事件
	SourceAgentID ID             // 源Agent ID，产生本事件的Agent
	TargetAgentID ID             // 目标Agent ID，指定处理的Agent（可选）
	Status        EventStatus    // 事件处理状态
	DispatchMode  DispatchMode   // 事件分发模式
	Priority      int            // 事件优先级（1-100）
	Payload       any            // 事件载荷数据
	Metadata      map[string]any // 事件元数据
	CreatedAt     time.Time      // 创建时间
	UpdatedAt     time.Time      // 更新时间
}

// Message 表示 Agent 在群聊房间中的自然语言发言。
type Message struct {
	ID          ID          // 消息唯一标识
	RoomID      ID          // 关联的房间ID
	SessionID   ID          // 关联的Session ID
	EventID     ID          // 关联的事件ID
	AgentID     ID          // 发送消息的Agent ID
	Role        MessageRole // 消息角色
	Content     string      // 消息内容
	ArtifactIDs []ID        // 引用的产物ID列表
	CreatedAt   time.Time   // 创建时间
}

// Artifact 是 Agent 产出的结构化中间结果。
type Artifact struct {
	ID        ID           // 产物唯一标识
	RoomID    ID           // 关联的房间ID
	SessionID ID           // 关联的Session ID
	EventID   ID           // 关联的事件ID
	AgentID   ID           // 产生产物的Agent ID
	Type      ArtifactType // 产物类型
	Data      any          // 产物数据内容
	CreatedAt time.Time    // 创建时间
}

// AgentRunStatus 表示一次 AgentRun 的执行状态。
type AgentRunStatus string

const (
	AgentRunPending   AgentRunStatus = "pending"
	AgentRunRunning   AgentRunStatus = "running"
	AgentRunSucceeded AgentRunStatus = "succeeded"
	AgentRunRejected  AgentRunStatus = "rejected"
	AgentRunFailed    AgentRunStatus = "failed"
	AgentRunCancelled AgentRunStatus = "cancelled"
)

// AgentRun 表示某个 Agent 对某个 Event 的一次处理记录。
type AgentRun struct {
	ID          ID                // Agent运行记录唯一标识
	SessionID   ID                // 关联的Session ID
	RoomID      ID                // 关联的房间ID
	EventID     ID                // 处理的事件ID
	AgentID     ID                // 执行的Agent ID
	Status      AgentRunStatus    // 运行状态
	TokenUsage  TokenUsageSummary // Token使用统计
	StartedAt   time.Time         // 开始时间
	CompletedAt *time.Time        // 完成时间（可为空）
	Error       string            // 错误信息（如果失败）
}

// StateSnapshot 是每个 Tick 后的系统状态快照，用于恢复、审计和 UI 展示。
type StateSnapshot struct {
	ID               ID                    // 快照唯一标识
	RoomID           ID                    // 关联的房间ID
	SessionID        ID                    // 关联的Session ID
	Tick             int                   // 当前Tick计数
	CurrentPhase     Phase                 // 当前阶段
	CurrentEventID   ID                    // 当前事件ID
	ActiveAgentID    ID                    // 当前活跃Agent ID
	EventStatus      map[ID]EventStatus    // 各事件状态映射
	AgentStatus      map[ID]AgentRunStatus // 各Agent运行状态映射
	ArtifactStatus   map[ID]string         // 各产物状态映射
	LatestGoalID     ID                    // 最新目标ID
	LatestPlanID     ID                    // 最新计划ID
	LatestResultID   ID                    // 最新结果ID
	LatestDeliveryID ID                    // 最新交付ID
	FailureCounts    map[string]int        // 各阶段失败次数统计
	LoopCounts       map[string]int        // 各阶段循环次数统计
	TokenUsage       TokenUsageSummary     // Token使用统计
	CreatedAt        time.Time             // 快照创建时间
}

// IsTerminal 判断快照是否已经处于终态。
func (s StateSnapshot) IsTerminal() bool {
	return s.CurrentPhase == PhaseCompleted || s.CurrentPhase == PhaseBlocked
}
