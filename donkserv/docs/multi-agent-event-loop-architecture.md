# 多 Agent 事件循环协作架构设计

## 一、设计目标

本文档基于 `multi-agent-creative-architecture.md` 中的多 Agent 创意目标生成方案，重新设计一套以事件循环为核心的多 Agent 协作架构。

新的设计不采用固定责任链或硬编码流水线，而是将多个 Agent 组织在同一个协作空间中，以类似“群聊”的方式监听事件、认领事件、发表消息、产出结构化结果，并由全局事件循环推动整个任务生命周期。

核心目标：

```text
让每个 Agent 独立存在
让每次执行状态可追踪
让失败回路自然表达
让工具调用可控可审计
让上下文装配可配置
让系统可恢复、可观察、可扩展
```

核心思想：

```text
不是 Agent A 直接调用 Agent B，
而是 Agent A 处理事件后发布新事件，
Agent B 看到自己能处理的事件后主动认领。
```

---

## 二、总体架构定位

### 2.1 原始链路

原始多 Agent 任务链路可以概括为：

```text
目标创意 Agent
  ↓
目标去重 Agent
  ↓
目标价值评审 Agent
  ↓
目标可行性 Agent
  ↓
目标收敛 Agent
  ↓
任务规划 Agent
  ↓
规划审查 Agent
  ↓
任务执行 Agent
  ↓
成果审查 Agent
  ↓
任务交付 Agent
```

这条链路适合表达主流程，但不适合表达复杂回路，例如：

```text
目标去重失败 → 回到目标创意 Agent
目标价值评审失败 → 回到目标创意 Agent
目标可行性失败 → 回到目标创意 Agent
规划审查失败 → 回到任务规划 Agent
成果审查失败 → 回到任务执行 Agent
连续失败 → 升级到更上游 Agent
```

如果使用固定调用链，这些回路会不断增加条件分支，系统会逐渐变得复杂且难以追踪。

### 2.2 新架构定位

新架构将系统设计为：

```text
以 Session 管理一次任务生命周期
以 Room 承载 Agent 群聊协作空间
以 Event 驱动状态变化
以 Message 表达 Agent 发言
以 Artifact 保存结构化产物
以 Agent Claim 表示 Agent 对事件的认领
以 StateSnapshot 记录每一步状态
以 Runtime Event Loop 推动整体流程
```

整体形态：

```text
Trigger
  ↓
Create Session
  ↓
Create Room
  ↓
Publish Root Event
  ↓
Runtime Event Loop
  ↓
Agent Claim
  ↓
Agent Runtime
      ↓
      Build Context
      ↓
      Input Hooks
      ↓
      Agent Internal Loop
          ↓
          LLM Call
          ↓
          Tool Call
          ↓
          Tool Observation
          ↓
          Final Output
      ↓
      Output Hooks
      ↓
      Commit Messages / Artifacts / Events
  ↓
State Snapshot
  ↓
Next Event
  ↓
Session Completed / Failed / Blocked
```

---

## 三、核心设计原则

### 3.1 Agent 不直接调用 Agent

Agent 不应该知道下一个 Agent 一定是谁。

错误方式：

```text
PlanningAgent 内部直接调用 PlanReviewAgent
PlanReviewAgent 内部直接调用 ExecutionAgent
ExecutionAgent 失败后直接调用 PlanningAgent
```

正确方式：

```text
PlanningAgent 输出 PlanCreated + PlanReviewRequested
Runtime Event Loop 发现 PlanReviewRequested
PlanReviewAgent 认领 PlanReviewRequested
PlanReviewAgent 输出 PlanReviewPassed 或 PlanReviewRejected
Runtime Event Loop 根据新事件继续推进
```

### 3.2 Agent 只处理事件并产出结果

Agent 的职责是：

```text
读取输入
完成判断或生成
输出消息
输出结构化产物
建议发布新事件
```

Agent 不负责：

```text
直接改全局状态
直接控制主流程
直接调用其他 Agent
直接使用未授权工具
直接写入最终事件流
```

### 3.3 Runtime 统一提交状态

所有状态变化都应由 Runtime 统一提交。

Agent 只返回草稿：

```text
MessageDraft
ArtifactDraft
EventDraft
Decision
```

Runtime 负责：

```text
校验输出
解析结构
落库消息
落库产物
发布事件
生成快照
推进 Session 状态
```

### 3.4 上下文由 Runtime 装配

Agent 不应该自己到处查询数据库或拼接上下文。

每个 Agent 通过 `ContextPolicy` 声明自己需要什么上下文，Runtime 根据策略装配 `AgentInput`。

### 3.5 工具由 ToolRuntime 管理

Agent 不能直接获得任意工具。

工具访问必须经过：

```text
ToolRegistry
ToolPolicy
ToolRuntime
ToolCallTrace
```

高风险工具调用应事件化，进入全局事件流。

### 3.6 大小循环边界清晰

系统中存在两层循环：

```text
Runtime Event Loop：系统级大事件循环
Agent Internal Loop：单 Agent 内部执行循环
```

大事件循环负责跨 Agent、跨阶段推进；Agent 内部循环只负责完成一次事件处理。

---

## 四、核心概念模型

### 4.1 Session

Session 表示一次从触发开始，到最终交付、失败、阻塞或取消结束的多 Agent 协作生命周期。

它不是用户聊天 Session，也不是 HTTP Session。

它的含义是：

```text
一次完整的目标发现 + 计划生成 + 计划审查 + 执行 + 成果审查 + 交付过程
```

一个 Session 通常由以下触发源创建：

```text
SystemStarted
UserTriggered
ContextChanged
TimerTriggered
PreviousLoopCompleted
```

Session 结束于：

```text
DeliveryCompleted
SessionCompleted
SessionFailed
SessionCancelled
SessionBlocked
```

Session 主要保存生命周期状态摘要：

```go
type Session struct {
    ID             string
    TriggerType    TriggerType
    TriggerPayload  any

    Status         SessionStatus
    CurrentPhase   Phase

    RoomID         string
    RootEventID    string
    CurrentEventID string

    FinalGoalID    string
    PlanID         string
    ExecutionID    string
    DeliveryID     string

    Tick           int

    FailureCounts  map[string]int
    RetryCounts    map[string]int

    StartedAt      time.Time
    CompletedAt    *time.Time
}
```

Session 是系统恢复、追踪、审计和可视化的核心单位。

### 4.2 Room

Room 是一个 Session 内部的 Agent 群聊协作空间。

第一版建议：

```text
一个 Session 对应一个 Room
```

后续支持复杂子任务时，可以扩展为：

```text
一个 Session 对应一个主 Room
一个复杂 PlanStep 可以创建一个子 Room
```

Room 包含：

```text
参与 Agent
群聊消息
事件流
中间产物引用
状态
```

```go
type ConversationRoom struct {
    ID        string
    SessionID string
    Topic     string
    Status    RoomStatus
    Members   []AgentID
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 4.3 Event

Event 是系统状态变化的核心协议。

Agent 不直接调用 Agent，而是消费事件、产出事件。

```go
type Event struct {
    ID            string
    Type          EventType
    RoomID        string
    SessionID     string
    CorrelationID string
    CausationID   string

    SourceAgentID string
    TargetAgentID string

    Status        EventStatus
    DispatchMode  DispatchMode
    Priority      int

    Payload       any
    Metadata      EventMetadata

    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

关键字段说明：

| 字段 | 说明 |
|---|---|
| `ID` | 当前事件唯一 ID |
| `Type` | 事件类型 |
| `RoomID` | 所属协作房间 |
| `SessionID` | 所属任务生命周期 |
| `CorrelationID` | 整条任务链路关联 ID |
| `CausationID` | 当前事件由哪个事件触发 |
| `SourceAgentID` | 事件来源 Agent |
| `TargetAgentID` | 指定目标 Agent，可为空 |
| `Status` | 事件当前状态 |
| `DispatchMode` | 分发模式 |
| `Payload` | 事件负载 |
| `Metadata` | 重试、错误、标签、依赖等元数据 |

### 4.4 Message

Message 表示 Agent 在群聊房间中的发言。

Message 是给人、给模型、给上下文看的自然语言记录。

```go
type Message struct {
    ID          string
    RoomID      string
    SessionID   string
    EventID     string
    AgentID     string
    Role        MessageRole
    Content     string
    ArtifactIDs []string
    CreatedAt   time.Time
}
```

Message 与 Event 的区别：

```text
Message 是聊天表达
Event 是状态协议
Artifact 是结构化产物
```

### 4.5 Artifact

Artifact 是结构化中间产物。

包括：

```text
CandidateGoal
DedupReview
ValueReview
FeasibilityReview
FinalExecutableGoal
ExecutablePlan
PlanReview
ExecutionResult
ResultReview
FinalDelivery
```

Artifact 用于避免所有上下文都依赖自然语言消息。

### 4.6 AgentRun

AgentRun 表示某个 Agent 对某个 Event 的一次处理记录。

```go
type AgentRun struct {
    ID        string
    SessionID string
    RoomID    string
    EventID   string
    AgentID   string

    Status    AgentRunStatus

    InputID   string
    OutputID  string
    TraceID   string

    StartedAt   time.Time
    CompletedAt *time.Time
    Error        *AgentError
}
```

AgentRun 是调试和追踪 Agent 行为的核心对象。

### 4.7 StateSnapshot

StateSnapshot 表示每个 Tick 后系统状态快照。

它用于回答：

```text
当前处于哪个阶段？
当前哪个事件正在执行？
哪个 Agent 正在工作？
目标是否已确定？
计划是否已通过？
执行结果是否已审查？
失败过几次？
是否进入回退流程？
```

```go
type StateSnapshot struct {
    ID               string
    RoomID           string
    SessionID        string
    Tick             int

    CurrentPhase     Phase
    CurrentEventID   string
    ActiveAgentID    string

    EventStatus      map[string]EventStatus
    AgentStatus      map[string]AgentRuntimeStatus
    ArtifactStatus   map[string]ArtifactStatus

    LatestGoalID     string
    LatestPlanID     string
    LatestResultID   string
    LatestDeliveryID string

    FailureCounts    map[string]int
    LoopCounts       map[string]int

    CreatedAt        time.Time
}
```

---

## 五、状态模型

### 5.1 EventStatus

```go
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
```

状态说明：

| 状态 | 含义 |
|---|---|
| `pending` | 事件已进入房间，等待处理 |
| `claimed` | 某个 Agent 已认领 |
| `processing` | Agent 正在处理 |
| `succeeded` | 处理成功 |
| `rejected` | 业务否决 |
| `failed` | 技术失败 |
| `skipped` | 被策略跳过 |
| `expired` | 超时未处理 |

### 5.2 SessionStatus

```go
type SessionStatus string

const (
    SessionRunning   SessionStatus = "running"
    SessionBlocked   SessionStatus = "blocked"
    SessionCompleted SessionStatus = "completed"
    SessionFailed    SessionStatus = "failed"
    SessionCancelled SessionStatus = "cancelled"
)
```

### 5.3 Phase

```go
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
```

---

## 六、事件类型设计

### 6.1 触发类事件

```text
SystemStarted
UserTriggered
ContextChanged
TimerTriggered
PreviousLoopCompleted
GoalRequested
```

### 6.2 目标生成层事件

```text
CandidateGoalCreated
GoalDedupRequested
GoalDedupPassed
GoalDedupRejected
GoalValueReviewRequested
GoalValueReviewPassed
GoalValueReviewRejected
GoalFeasibilityRequested
GoalFeasibilityPassed
GoalFeasibilityRejected
GoalConvergenceRequested
FinalGoalCreated
GoalRegenerationRequested
GoalRefinementRequested
```

### 6.3 任务执行准备层事件

```text
PlanRequested
PlanCreated
PlanReviewRequested
PlanReviewPassed
PlanReviewRejected
PlanRevisionRequested
GoalFeasibilityRecheckRequested
```

### 6.4 执行交付层事件

```text
ExecutionRequested
ExecutionStepStarted
ExecutionStepCompleted
ExecutionCompleted
ExecutionRevisionRequested
ResultReviewRequested
ResultReviewPassed
ResultReviewRejected
DeliveryRequested
DeliveryCompleted
LoopCompleted
```

### 6.5 工具类事件

```text
ToolCallRequested
ToolCallStarted
ToolCallCompleted
ToolCallFailed
ToolApprovalRequested
ToolApprovalGranted
ToolApprovalRejected
```

### 6.6 Session 类事件

```text
SessionStarted
SessionCompleted
SessionFailed
SessionBlocked
SessionCancelled
```

---

## 七、事件分发与认领机制

### 7.1 DispatchMode

事件支持三种分发模式：

```go
type DispatchMode string

const (
    DispatchExclusive   DispatchMode = "exclusive"
    DispatchCompetitive DispatchMode = "competitive"
    DispatchBroadcast   DispatchMode = "broadcast"
)
```

| 模式 | 说明 | 示例 |
|---|---|---|
| `exclusive` | 只有一个指定 Agent 或角色处理 | `PlanReviewRequested` |
| `competitive` | 多个 Agent 可申请处理，调度器选择一个 | `UnexpectedFailure` |
| `broadcast` | 多个 Agent 都可以发表意见 | `CandidateGoalCreated` |

### 7.2 EventClaim

Agent 通过 Claim 机制认领事件。

```go
type EventClaim struct {
    ID         string
    EventID    string
    AgentID    string
    Confidence float64
    Priority   int
    Reason     string
    Status     ClaimStatus
    CreatedAt  time.Time
}
```

Claim 状态：

```text
proposed
accepted
rejected
expired
cancelled
```

### 7.3 认领流程

```text
Event Published
  ↓
Runtime 找到可用 Agent
  ↓
Agent.CanHandle(event, snapshot)
  ↓
生成 ClaimDecision
  ↓
Scheduler 选择 Claim
  ↓
Event 标记为 claimed
  ↓
AgentRuntime 执行 AgentRun
```

Agent 接口：

```go
type Agent interface {
    ID() AgentID
    Name() string
    Role() AgentRole

    CanHandle(ctx context.Context, event Event, room RoomSnapshot) ClaimDecision

    Handle(ctx context.Context, input AgentInput) AgentOutput
}
```

ClaimDecision：

```go
type ClaimDecision struct {
    CanClaim   bool
    Confidence float64
    Reason     string
    Priority   int
}
```

---

## 八、Agent 输入输出 Hook 设计

### 8.1 Hook 定位

Agent 的输入输出 Hook 是 Agent 执行过程中的中间件。

它不属于 Agent 的核心业务逻辑，而属于 AgentRuntime 的执行管线。

Hook 主要负责：

```text
输入前：
- 上下文装配
- Prompt 渲染
- 权限检查
- 工具注入
- Token 控制
- 安全过滤
- 输入脱敏

输出后：
- 输出解析
- Schema 校验
- Artifact 提取
- EventDraft 生成
- MessageDraft 生成
- 状态提交前校验
- 指标记录
- 审计记录
```

### 8.2 Agent 执行管线

```text
BeforeClaim
BeforeBuildInput
BuildInput
AfterBuildInput
BeforeInvoke
InvokeAgent
AfterInvoke
BeforeParseOutput
ParseOutput
AfterParseOutput
BeforeCommit
CommitOutput
AfterCommit
```

### 8.3 Hook 接口

可以采用接口形式：

```go
type AgentHook interface {
    BeforeBuildInput(ctx context.Context, req *AgentRunRequest) error
    AfterBuildInput(ctx context.Context, input *AgentInput) error

    BeforeInvoke(ctx context.Context, input *AgentInput) error
    AfterInvoke(ctx context.Context, raw *AgentRawOutput) error

    BeforeParseOutput(ctx context.Context, raw *AgentRawOutput) error
    AfterParseOutput(ctx context.Context, output *AgentOutput) error

    BeforeCommit(ctx context.Context, output *AgentOutput) error
    AfterCommit(ctx context.Context, result *AgentRunResult) error
}
```

也可以采用中间件形式：

```go
type AgentMiddleware func(next AgentHandler) AgentHandler
```

中间件方式更适合组合扩展。

### 8.4 Hook 分类

#### 输入装配 Hook

```text
EventContextHook
RoomMessageHook
ArtifactContextHook
MemoryContextHook
UserProfileHook
ToolContextHook
PromptTemplateHook
```

#### 调用前 Hook

```text
PermissionHook
ToolAccessHook
TokenBudgetHook
RateLimitHook
SafetyInputHook
DedupInputHook
```

#### 输出解析 Hook

```text
SchemaValidationHook
ArtifactExtractHook
DecisionParseHook
EventDraftHook
MessageDraftHook
SafetyOutputHook
```

#### 提交后 Hook

```text
PersistMessageHook
PersistArtifactHook
PublishEventHook
StateSnapshotHook
MetricsHook
AuditHook
```

---

## 九、Agent 上下文与工具配置

### 9.1 AgentProfile

每个 Agent 通过 AgentProfile 描述自己的身份、模型、上下文策略、工具策略、输出结构和运行限制。

```go
type AgentProfile struct {
    ID          AgentID
    Name        string
    Role        AgentRole
    Description string

    Model       ModelConfig
    Prompt      PromptConfig

    ContextPolicy ContextPolicy
    ToolPolicy    ToolPolicy
    OutputSchema  OutputSchema

    MaxTurns    int
    MaxRetries  int
    Timeout     time.Duration
}
```

### 9.2 ContextPolicy

ContextPolicy 定义某个 Agent 可以看到什么上下文。

```go
type ContextPolicy struct {
    IncludeRoomMessages    bool
    MessageWindow          int
    IncludeSessionState    bool
    IncludeUserProfile     bool
    IncludeHistoricalTasks bool
    IncludeArtifacts       []ArtifactType
    IncludeVectorMemory    bool
    VectorQueryTemplates   []string
    MaxContextTokens       int
}
```

### 9.3 不同 Agent 的上下文建议

#### 目标创意 Agent

```text
可见上下文：
- 核心主题
- 用户画像
- 最近聊天记录
- 历史已执行任务摘要
- 向量检索结果
- 当前时间
- 外部参考资料摘要

不需要：
- 具体执行工具
- 低层执行日志
```

#### 目标去重 Agent

```text
可见上下文：
- CandidateGoal
- 历史目标列表
- 相似度检索结果
- 已完成任务摘要

不需要：
- 用户完整聊天记录
- 执行工具
```

#### 目标价值评审 Agent

```text
可见上下文：
- CandidateGoal
- 当前核心主题
- 用户画像摘要
- 历史任务摘要
- 价值评审标准
```

#### 目标可行性 Agent

```text
可见上下文：
- CandidateGoal
- 可用工具能力摘要
- Agent 能力边界
- 权限约束
- 安全约束
```

#### 目标收敛 Agent

```text
可见上下文：
- CandidateGoal
- DedupReview
- ValueReview
- FeasibilityReview
- 目标输出 Schema
```

#### 任务规划 Agent

```text
可见上下文：
- FinalExecutableGoal
- 可用工具清单
- 工具能力说明
- 约束条件
- 历史类似计划
```

#### 规划审查 Agent

```text
可见上下文：
- FinalExecutableGoal
- ExecutablePlan
- 规划审查标准
- 工具能力边界
```

#### 任务执行 Agent

```text
可见上下文：
- ExecutablePlan
- 当前 PlanStep
- 上一步执行结果
- 可调用工具
- 工具调用结果
- 错误反馈
```

#### 成果审查 Agent

```text
可见上下文：
- FinalExecutableGoal
- ExecutablePlan
- ExecutionResult
- 交付标准
```

#### 任务交付 Agent

```text
可见上下文：
- FinalExecutableGoal
- ExecutablePlan
- ExecutionResult
- ResultReview
- 交付渠道
- 归档规则
```

### 9.4 ToolPolicy

ToolPolicy 定义 Agent 可使用哪些工具，以及工具的权限、限制、审批规则。

```go
type ToolPolicy struct {
    AllowedTools    []ToolName
    DeniedTools     []ToolName
    ToolScopes      map[ToolName]ToolScope
    RequireApproval []ToolName
    MaxCallsPerRun  int
    MaxCallsPerTool map[ToolName]int
    TimeoutPerTool  map[ToolName]time.Duration
}
```

工具描述：

```go
type ToolSpec struct {
    Name         ToolName
    Description  string
    InputSchema  Schema
    OutputSchema Schema
    RiskLevel    RiskLevel
}
```

### 9.5 工具调用模式

#### 模式 A：Agent 内部同步工具调用

适合轻量工具：

```text
向量检索
历史任务查询
上下文摘要
Schema 修复
```

执行形态：

```text
AgentRunStarted
  → ToolCallStarted
  → ToolCallCompleted
AgentRunCompleted
```

这类工具调用记录到 AgentRunTrace，不一定进入主事件流。

#### 模式 B：工具调用事件化

适合重工具或高风险工具：

```text
文件写入
命令执行
HTTP 请求
数据库修改
外部 API 调用
```

执行形态：

```text
Agent 输出 ToolCallRequested
Runtime 发布 ToolCallRequested Event
ToolRuntime 或 ToolAgent 认领
执行后发布 ToolCallCompleted
原 Agent 或下一 Agent 继续
```

---

## 十、大小循环关系

### 10.1 Runtime Event Loop

Runtime Event Loop 是系统级大事件循环。

它负责：

```text
1. 查找 pending 事件
2. 让 Agent 评估是否可认领
3. 调度选中的 Claim
4. 执行 AgentRun
5. 提交 Message / Artifact / Event
6. 生成 StateSnapshot
7. 判断 Session 是否结束
```

伪代码：

```go
func (r *Runtime) Run(ctx context.Context, sessionID string) error {
    for {
        snapshot := r.state.TakeSnapshot(sessionID)

        if snapshot.IsTerminal() {
            return nil
        }

        events := r.eventStore.ListPending(sessionID)

        if len(events) == 0 {
            r.eventBus.Publish(SystemIdleDetected)
            continue
        }

        for _, event := range events {
            claims := r.collectClaims(ctx, event, snapshot)
            selectedClaims := r.scheduler.Select(event, claims)

            for _, claim := range selectedClaims {
                r.processClaim(ctx, event, claim)
            }
        }
    }
}
```

### 10.2 Agent Internal Loop

Agent Internal Loop 是单 Agent 内部循环。

它负责：

```text
1. 接收 AgentInput
2. 进行一次或多次模型推理
3. 根据需要调用允许的工具
4. 汇总工具结果
5. 形成 AgentOutput
```

它可以是 ReAct 风格：

```text
思考 → 调工具 → 观察 → 再思考 → 再调工具 → 输出结论
```

但必须有明确边界：

```go
type AgentLoopConfig struct {
    MaxTurns     int
    MaxToolCalls int
    MaxDuration  time.Duration
    StopOnFinal  bool
}
```

### 10.3 两层循环的关系

```text
Runtime Event Loop
  └── AgentRun
        └── Agent Internal Loop
              ├── LLM Call
              ├── Tool Call
              ├── Tool Result
              └── Final AgentOutput
```

Agent 小循环只完成一个事件处理；大事件循环负责系统推进。

### 10.4 哪些动作进入主事件流

判断标准：

```text
这个动作是否需要其他 Agent 看到？
这个动作是否会改变全局阶段？
这个动作是否可能被其他 Agent 认领？
这个动作是否需要失败回路？
这个动作是否需要人工审批？
```

如果是，则进入 Event。

如果只是 Agent 内部推理细节，则进入 AgentRunTrace。

---

## 十一、Token 消耗记录与预算控制

### 11.1 设计目标

多 Agent 系统必须把 Token 当作一等资源管理。

Token 管理需要解决：

```text
每次 LLM 调用消耗了多少 Token
每个 Agent 消耗了多少 Token
每个 Event 消耗了多少 Token
每个 Session 总共消耗了多少 Token
是否接近预算上限
超过预算后是否继续、降级、暂停或终止
```

Token 消耗不应该只记录在日志里，而应该成为 Runtime 状态的一部分，并进入 `AgentRunTrace`、`StateSnapshot` 和 Session 汇总。

### 11.2 TokenUsage

每次模型调用都应该产生一条 TokenUsage 记录。

```go
type TokenUsage struct {
    ID               string
    SessionID        string
    RoomID           string
    EventID          string
    AgentRunID       string
    AgentID          string

    ModelProvider    string
    ModelName        string

    PromptTokens     int
    CompletionTokens int
    TotalTokens      int

    CachedTokens     int
    ReasoningTokens  int

    EstimatedCost    float64
    Currency         string

    CreatedAt        time.Time
}
```

TokenUsage 的关联维度：

```text
Session 维度：统计整轮任务成本
Room 维度：统计某个协作空间成本
Event 维度：统计某类事件成本
AgentRun 维度：统计某次 Agent 执行成本
Agent 维度：统计不同 Agent 成本
Model 维度：统计不同模型成本
```

### 11.3 TokenBudget

TokenBudget 用于定义预算上限。

```go
type TokenBudget struct {
    ID                  string
    Scope               BudgetScope
    ScopeID             string

    MaxPromptTokens      int
    MaxCompletionTokens  int
    MaxTotalTokens       int
    MaxEstimatedCost     float64

    WarnThresholdRatio   float64
    StopThresholdRatio   float64

    OnWarn               TokenLimitAction
    OnStop               TokenLimitAction

    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

BudgetScope：

```go
type BudgetScope string

const (
    BudgetGlobal   BudgetScope = "global"
    BudgetSession  BudgetScope = "session"
    BudgetRoom     BudgetScope = "room"
    BudgetAgent    BudgetScope = "agent"
    BudgetEvent    BudgetScope = "event"
    BudgetAgentRun BudgetScope = "agent_run"
)
```

不同层级可以同时设置预算：

```text
GlobalBudget：系统总预算
SessionBudget：单次任务生命周期预算
AgentBudget：单个 Agent 在一个 Session 内的预算
EventBudget：单类事件预算
AgentRunBudget：单次 AgentRun 预算
```

### 11.4 TokenLimitAction

当 Token 接近或超过上限时，不应该只有一种处理方式。

```go
type TokenLimitAction string

const (
    TokenActionContinue         TokenLimitAction = "continue"
    TokenActionCompactContext   TokenLimitAction = "compact_context"
    TokenActionDowngradeModel   TokenLimitAction = "downgrade_model"
    TokenActionSkipOptional     TokenLimitAction = "skip_optional"
    TokenActionPauseSession     TokenLimitAction = "pause_session"
    TokenActionBlockSession     TokenLimitAction = "block_session"
    TokenActionStopSession      TokenLimitAction = "stop_session"
    TokenActionRequireApproval  TokenLimitAction = "require_approval"
)
```

推荐策略：

```text
达到 70%：记录告警，压缩上下文
达到 85%：跳过非必要广播 Agent，降低模型规格
达到 95%：暂停 Session，等待继续授权或人工确认
达到 100%：终止循环，SessionBlocked 或 SessionFailed
```

### 11.5 TokenBudgetGuard

TokenBudgetGuard 应该集成在 Agent 输入输出 Hook 和 Runtime Event Loop 中。

```go
type TokenBudgetGuard interface {
    BeforeBuildInput(ctx context.Context, req AgentRunRequest) TokenBudgetDecision
    BeforeInvoke(ctx context.Context, input AgentInput) TokenBudgetDecision
    AfterInvoke(ctx context.Context, usage TokenUsage) TokenBudgetDecision
    BeforeNextTick(ctx context.Context, snapshot StateSnapshot) TokenBudgetDecision
}
```

TokenBudgetDecision：

```go
type TokenBudgetDecision struct {
    Allowed       bool
    Action        TokenLimitAction
    Reason        string
    CurrentUsage  TokenUsageSummary
    Budget        TokenBudget
}
```

### 11.6 TokenUsageSummary

StateSnapshot 中应包含 Token 汇总信息。

```go
type TokenUsageSummary struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
    EstimatedCost    float64

    ByAgent          map[string]int
    ByEventType      map[string]int
    ByModel          map[string]int

    BudgetRatio      float64
    IsNearLimit      bool
    IsExceeded       bool
}
```

StateSnapshot 建议增加：

```go
type StateSnapshot struct {
    TokenUsage TokenUsageSummary
}
```

Session 建议增加：

```go
type Session struct {
    TokenUsage TokenUsageSummary
    TokenBudget TokenBudget
}
```

AgentRun 建议增加：

```go
type AgentRun struct {
    TokenUsage TokenUsageSummary
}
```

### 11.7 Token 相关事件

Token 预算变化也可以进入事件流。

```text
TokenUsageRecorded
TokenBudgetWarning
TokenBudgetExceeded
TokenContextCompactionRequested
TokenModelDowngradeRequested
TokenApprovalRequested
TokenSessionPauseRequested
TokenSessionStopRequested
```

其中：

```text
TokenUsageRecorded：每次模型调用后记录
TokenBudgetWarning：接近预算阈值
TokenBudgetExceeded：超过预算阈值
TokenContextCompactionRequested：请求压缩上下文
TokenModelDowngradeRequested：请求降低模型规格
TokenApprovalRequested：请求人工确认是否继续消耗
TokenSessionPauseRequested：请求暂停 Session
TokenSessionStopRequested：请求停止 Session
```

### 11.8 Token 与上下文压缩

当 Token 接近上限时，优先不应该直接停止，而是逐步降级：

```text
1. 缩小 MessageWindow
2. 使用 Artifact 摘要替代完整消息
3. 对历史消息做摘要压缩
4. 去掉非当前阶段必要上下文
5. 跳过非必要广播 Agent
6. 降低模型规格
7. 暂停 Session 等待确认
8. 终止 Session
```

上下文压缩应该由 ContextBuilder 和 HookPipeline 完成，而不是由 Agent 自己决定。

---

## 十二、循环启动、暂停、停止与恢复

### 12.1 设计目标

事件循环必须支持显式生命周期控制。

需要支持：

```text
启动循环
暂停循环
恢复循环
优雅停止循环
立即停止循环
终止 Session
恢复未完成 Session
```

### 12.2 LoopStatus

```go
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
```

Session 与 Loop 的关系：

```text
SessionStatus 表示任务生命周期状态
LoopStatus 表示事件循环运行状态
```

例如：

```text
SessionStatus = running, LoopStatus = paused
表示任务未结束，但事件循环暂时暂停。
```

### 12.3 LoopControlCommand

循环控制建议通过命令或控制事件完成。

```go
type LoopControlCommand struct {
    ID        string
    SessionID string
    Type      LoopControlType
    Mode      StopMode
    Reason    string
    CreatedAt time.Time
}
```

```go
type LoopControlType string

const (
    LoopCommandStart  LoopControlType = "start"
    LoopCommandPause  LoopControlType = "pause"
    LoopCommandResume LoopControlType = "resume"
    LoopCommandStop   LoopControlType = "stop"
    LoopCommandCancel LoopControlType = "cancel"
)
```

### 12.4 StopMode

停止循环需要区分优雅停止和立即停止。

```go
type StopMode string

const (
    StopGraceful         StopMode = "graceful"
    StopAfterCurrentTick StopMode = "after_current_tick"
    StopImmediate        StopMode = "immediate"
)
```

含义：

| 模式 | 说明 |
|---|---|
| `graceful` | 不再领取新事件，等待当前 AgentRun 和必要提交完成 |
| `after_current_tick` | 当前 Tick 完成后停止 |
| `immediate` | 尽快取消当前上下文，标记运行中 AgentRun 为 cancelled |

### 12.5 RuntimeController

建议增加 RuntimeController 作为外部控制入口。

```go
type RuntimeController interface {
    StartSession(ctx context.Context, trigger Trigger) (SessionID, error)
    StartLoop(ctx context.Context, sessionID SessionID) error
    PauseLoop(ctx context.Context, sessionID SessionID, reason string) error
    ResumeLoop(ctx context.Context, sessionID SessionID) error
    StopLoop(ctx context.Context, sessionID SessionID, mode StopMode, reason string) error
    CancelSession(ctx context.Context, sessionID SessionID, reason string) error
}
```

### 12.6 启动流程

```text
StartSession
  ↓
Create Session
  ↓
Create Room
  ↓
Publish SessionStarted
  ↓
Publish Root Event，例如 GoalRequested
  ↓
StartLoop
  ↓
LoopStatus = running
```

### 12.7 暂停流程

暂停适用于：

```text
Token 接近上限，需要人工确认
等待外部资源
等待用户补充信息
等待高风险工具审批
系统维护
```

流程：

```text
PauseLoop
  ↓
LoopStatus = pausing
  ↓
不再领取新的 pending 事件
  ↓
当前 AgentRun 到安全点后停止
  ↓
LoopStatus = paused
  ↓
SessionStatus 保持 running 或 blocked
```

### 12.8 恢复流程

```text
ResumeLoop
  ↓
检查 SessionStatus 是否允许恢复
  ↓
检查 TokenBudget / ToolApproval / PendingEvent
  ↓
LoopStatus = running
  ↓
继续处理 pending 事件
```

### 12.9 停止流程

优雅停止：

```text
StopLoop(graceful)
  ↓
LoopStatus = stopping
  ↓
停止领取新事件
  ↓
等待当前 AgentRun 完成
  ↓
提交已产生的 Message / Artifact / Event
  ↓
生成最终 StateSnapshot
  ↓
LoopStatus = stopped
```

立即停止：

```text
StopLoop(immediate)
  ↓
取消 Runtime context
  ↓
运行中的 AgentRun 标记为 cancelled 或 failed
  ↓
未处理事件保持 pending 或标记 skipped
  ↓
生成停止快照
  ↓
LoopStatus = stopped
```

### 12.10 启停相关事件

```text
LoopStartRequested
LoopStarted
LoopPauseRequested
LoopPaused
LoopResumeRequested
LoopResumed
LoopStopRequested
LoopStopped
LoopFailed
SessionCancelRequested
SessionCancelled
```

这些事件可以进入 Event Stream，便于审计和 UI 展示。

### 12.11 Runtime Event Loop 中的控制点

大事件循环每个 Tick 都应该检查控制命令和预算状态。

```go
func (r *Runtime) Run(ctx context.Context, sessionID string) error {
    for {
        if r.control.ShouldStop(sessionID) {
            return r.stopLoop(ctx, sessionID)
        }

        if r.control.ShouldPause(sessionID) {
            return r.pauseLoop(ctx, sessionID)
        }

        snapshot := r.state.TakeSnapshot(sessionID)

        decision := r.tokenGuard.BeforeNextTick(ctx, snapshot)
        if !decision.Allowed {
            return r.applyTokenDecision(ctx, sessionID, decision)
        }

        if snapshot.IsTerminal() {
            return nil
        }

        events := r.eventStore.ListPending(sessionID)

        for _, event := range events {
            if r.control.ShouldStop(sessionID) {
                return r.stopLoop(ctx, sessionID)
            }

            r.processEvent(ctx, sessionID, event)
        }
    }
}
```

这样可以保证循环不是失控运行，而是可启动、可暂停、可恢复、可停止。

---

## 十三、10 个 Agent 的事件订阅与输出

### 13.1 目标创意 Agent

订阅事件：

```text
GoalRequested
GoalRegenerationRequested
GoalRefinementRequested
```

输出事件：

```text
CandidateGoalCreated
GoalDedupRequested
```

输出产物：

```text
CandidateGoal[]
```

### 13.2 目标去重 Agent

订阅事件：

```text
GoalDedupRequested
CandidateGoalCreated
```

输出事件：

```text
GoalDedupPassed
GoalDedupRejected
GoalRefinementRequested
```

输出产物：

```text
DedupReview
```

### 13.3 目标价值评审 Agent

订阅事件：

```text
GoalValueReviewRequested
GoalDedupPassed
```

输出事件：

```text
GoalValueReviewPassed
GoalValueReviewRejected
GoalRefinementRequested
```

输出产物：

```text
ValueReview
```

### 13.4 目标可行性 Agent

订阅事件：

```text
GoalFeasibilityRequested
GoalValueReviewPassed
GoalFeasibilityRecheckRequested
```

输出事件：

```text
GoalFeasibilityPassed
GoalFeasibilityRejected
GoalRefinementRequested
```

输出产物：

```text
FeasibilityReview
```

### 13.5 目标收敛 Agent

订阅事件：

```text
GoalFeasibilityPassed
GoalConvergenceRequested
```

输出事件：

```text
FinalGoalCreated
PlanRequested
```

输出产物：

```text
FinalExecutableGoal
```

### 13.6 任务规划 Agent

订阅事件：

```text
PlanRequested
PlanRevisionRequested
```

输出事件：

```text
PlanCreated
PlanReviewRequested
```

输出产物：

```text
ExecutablePlan
```

### 13.7 规划审查 Agent

订阅事件：

```text
PlanReviewRequested
PlanCreated
```

输出事件：

```text
PlanReviewPassed
PlanReviewRejected
PlanRevisionRequested
GoalFeasibilityRecheckRequested
ExecutionRequested
```

输出产物：

```text
PlanReview
```

### 13.8 任务执行 Agent

订阅事件：

```text
ExecutionRequested
ExecutionRevisionRequested
```

输出事件：

```text
ExecutionStepStarted
ExecutionStepCompleted
ExecutionCompleted
ResultReviewRequested
```

输出产物：

```text
ExecutionResult
```

### 13.9 成果审查 Agent

订阅事件：

```text
ResultReviewRequested
ExecutionCompleted
```

输出事件：

```text
ResultReviewPassed
ResultReviewRejected
ExecutionRevisionRequested
PlanRevisionRequested
DeliveryRequested
```

输出产物：

```text
ResultReview
```

### 13.10 任务交付 Agent

订阅事件：

```text
DeliveryRequested
ResultReviewPassed
```

输出事件：

```text
DeliveryCompleted
LoopCompleted
SessionCompleted
```

输出产物：

```text
FinalDelivery
```

---

## 十四、核心数据结构

### 14.1 CandidateGoal

```go
type CandidateGoal struct {
    ID             string
    Title          string
    Description    string
    Motivation     string
    ContextBasis   []string
    ExpectedOutput string
    Boundaries     []string
    Risks          []string
    CreatedBy      AgentID
    CreatedAt      time.Time
}
```

### 14.2 FinalExecutableGoal

```go
type FinalExecutableGoal struct {
    ID               string
    Title            string
    Description      string
    WhyNow           string
    ContextBasis     []string
    Scope            GoalScope
    ExpectedDelivery DeliverySpec
    Constraints      []Constraint
    AcceptedReviews  []ReviewRef
    CreatedAt        time.Time
}
```

### 14.3 ExecutablePlan

```go
type ExecutablePlan struct {
    ID           string
    GoalID       string
    Steps        []PlanStep
    Dependencies []PlanDependency
    Risks        []Risk
    ToolNeeds    []ToolNeed
    CreatedAt    time.Time
}
```

### 14.4 ExecutionResult

```go
type ExecutionResult struct {
    ID          string
    PlanID      string
    StepResults []StepResult
    Outputs     []ArtifactRef
    Errors      []ExecutionError
    CreatedAt   time.Time
}
```

### 14.5 FinalDelivery

```go
type FinalDelivery struct {
    ID            string
    GoalID        string
    PlanID        string
    ResultID      string
    Summary       string
    Deliverables  []ArtifactRef
    ArchiveRecord ArchiveRecord
    NextTrigger   *TriggerSpec
    CreatedAt     time.Time
}
```

---

## 十五、失败回路设计

### 15.1 目标生成层失败回路

```text
GoalDedupRejected
  → GoalRegenerationRequested
  → GoalCreativeAgent 重新认领
```

```text
GoalValueReviewRejected
  → GoalRefinementRequested
  → GoalCreativeAgent 重新认领
```

```text
GoalFeasibilityRejected
  → GoalRefinementRequested
  → GoalCreativeAgent 重新认领
```

### 15.2 任务执行准备层失败回路

```text
PlanReviewRejected
  → PlanRevisionRequested
  → PlanningAgent 重新认领
```

连续失败升级：

```text
PlanReviewRejected 超过阈值
  → GoalFeasibilityRecheckRequested
  → GoalFeasibilityAgent 重新检查目标边界
```

### 15.3 执行交付层失败回路

```text
ResultReviewRejected
  → ExecutionRevisionRequested
  → ExecutionAgent 修正执行结果
```

连续失败升级：

```text
ResultReviewRejected 超过阈值
  → PlanRevisionRequested
  → PlanningAgent 重新规划执行步骤
```

### 15.4 LoopGuard

事件循环必须有循环保护。

```go
type LoopGuard struct {
    MaxTotalTicks         int
    MaxSameEventRetries   int
    MaxPhaseRetries       int
    MaxAgentFailures      int
    MaxGoalRegenerations  int
    MaxPlanRevisions      int
    MaxExecutionRevisions int
}
```

示例策略：

```text
目标连续重新生成 3 次仍失败 → SessionBlocked
规划连续审查失败 3 次 → 回到可行性检查
执行连续修正失败 3 次 → 回到任务规划
总 Tick 超过 100 → SessionFailed
单 Agent 连续失败 5 次 → SessionBlocked
```

---

## 十六、推荐代码目录结构

```text
internal/
  agent/
    agent.go
    registry.go
    profile.go
    role.go

  runtime/
    runtime.go
    event_loop.go
    agent_runtime.go
    scheduler.go
    dispatcher.go
    snapshot.go

  room/
    room.go
    message.go
    conversation.go

  event/
    event.go
    event_type.go
    event_status.go
    claim.go
    bus.go
    store.go

  artifact/
    artifact.go
    goal.go
    plan.go
    execution.go
    delivery.go
    store.go

  state/
    session.go
    phase.go
    snapshot.go
    transition.go

  agents/
    goal_creative.go
    goal_dedup.go
    goal_value_review.go
    goal_feasibility.go
    goal_convergence.go
    planning.go
    plan_review.go
    execution.go
    result_review.go
    delivery.go

  hook/
    pipeline.go
    context.go
    safety.go
    schema.go
    artifact.go
    event.go
    audit.go

  context/
    builder.go
    policy.go
    memory.go
    prompt.go

  tool/
    registry.go
    runtime.go
    policy.go
    trace.go

  policy/
    claim_policy.go
    retry_policy.go
    escalation_policy.go
    loop_guard.go

  llm/
    client.go
    prompt.go
    structured_output.go
```

---

## 十七、最小可落地版本

第一阶段不建议直接接入复杂 LLM 和真实工具。

建议先实现最小闭环：

```text
1. Session
2. Room
3. Event
4. Message
5. Artifact
6. AgentRun
7. StateSnapshot
8. Agent interface
9. AgentRegistry
10. Runtime Event Loop
11. Claim 机制
12. HookPipeline 空实现
13. 10 个 Agent 的 Mock 实现
```

先跑通成功路径：

```text
GoalRequested
→ CandidateGoalCreated
→ GoalDedupPassed
→ GoalValueReviewPassed
→ GoalFeasibilityPassed
→ FinalGoalCreated
→ PlanCreated
→ PlanReviewPassed
→ ExecutionCompleted
→ ResultReviewPassed
→ DeliveryCompleted
→ SessionCompleted
```

第二阶段加入失败路径：

```text
GoalDedupRejected → GoalRegenerationRequested
PlanReviewRejected → PlanRevisionRequested
ResultReviewRejected → ExecutionRevisionRequested
```

第三阶段加入：

```text
真实 LLM
真实工具
ToolRuntime
ContextBuilder
AgentRunTrace
持久化
UI 可视化
人工审批
```

---

## 十八、完整执行示例

### Tick 1

系统创建 Session 和 Room，发布：

```text
SessionStarted
GoalRequested
```

状态：

```text
phase = goal_generation
current_event = GoalRequested
status = pending
```

### Tick 2

目标创意 Agent 认领：

```text
GoalRequested
```

输出：

```text
Message: 我将基于当前上下文生成候选目标。
Artifact: CandidateGoal
Event: CandidateGoalCreated
Event: GoalDedupRequested
```

状态：

```text
GoalRequested = succeeded
latest_candidate_goal = candidate_goal_001
```

### Tick 3

目标去重 Agent 认领：

```text
GoalDedupRequested
```

通过时输出：

```text
GoalDedupPassed
GoalValueReviewRequested
```

失败时输出：

```text
GoalDedupRejected
GoalRegenerationRequested
```

### Tick 4

目标价值评审 Agent 认领：

```text
GoalValueReviewRequested
```

输出：

```text
GoalValueReviewPassed
GoalFeasibilityRequested
```

### Tick 5

目标可行性 Agent 认领：

```text
GoalFeasibilityRequested
```

输出：

```text
GoalFeasibilityPassed
GoalConvergenceRequested
```

### Tick 6

目标收敛 Agent 认领：

```text
GoalConvergenceRequested
```

输出：

```text
FinalGoalCreated
PlanRequested
```

状态：

```text
phase = planning
final_goal_id = final_goal_001
```

### Tick 7

任务规划 Agent 认领：

```text
PlanRequested
```

输出：

```text
PlanCreated
PlanReviewRequested
```

### Tick 8

规划审查 Agent 认领：

```text
PlanReviewRequested
```

通过时输出：

```text
PlanReviewPassed
ExecutionRequested
```

不通过时输出：

```text
PlanReviewRejected
PlanRevisionRequested
```

### Tick 9

任务执行 Agent 认领：

```text
ExecutionRequested
```

输出：

```text
ExecutionCompleted
ResultReviewRequested
```

### Tick 10

成果审查 Agent 认领：

```text
ResultReviewRequested
```

通过时输出：

```text
ResultReviewPassed
DeliveryRequested
```

不通过时输出：

```text
ResultReviewRejected
ExecutionRevisionRequested
```

### Tick 11

任务交付 Agent 认领：

```text
DeliveryRequested
```

输出：

```text
FinalDelivery
DeliveryCompleted
LoopCompleted
SessionCompleted
```

最终状态：

```text
phase = completed
session_status = completed
```

---

## 十九、架构总结

本设计将多 Agent 系统从固定责任链重构为事件循环驱动的群聊式协作系统。

核心抽象是：

```text
Session：一次任务生命周期
Room：Agent 群聊协作空间
Event：系统状态变化协议
Message：Agent 群聊发言
Artifact：结构化中间产物
AgentRun：一次 Agent 处理记录
StateSnapshot：每个 Tick 的状态快照
HookPipeline：Agent 输入输出中间件
ContextPolicy：上下文可见性策略
ToolPolicy：工具权限策略
Runtime Event Loop：全局事件循环
Agent Internal Loop：单 Agent 内部推理与工具循环
```

最终系统不再关心“谁直接调用谁”，而是关心：

```text
发生了什么事件
哪些 Agent 可以认领
谁最终认领了事件
Agent 产出了什么消息
Agent 产出了什么结构化结果
Agent 建议发布什么新事件
Runtime 如何提交状态
当前 Session 处于什么阶段
失败是否需要回退或升级
```

这种设计可以同时满足：

```text
多 Agent 独立协作
状态可追踪
失败可回退
上下文可配置
工具可管控
执行可恢复
过程可观察
系统可扩展
```
