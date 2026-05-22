package creative

import "context"

// ClaimStatus 表示 Agent 对事件认领申请的状态。
type ClaimStatus string

const (
	ClaimProposed  ClaimStatus = "proposed"
	ClaimAccepted  ClaimStatus = "accepted"
	ClaimRejected  ClaimStatus = "rejected"
	ClaimExpired   ClaimStatus = "expired"
	ClaimCancelled ClaimStatus = "cancelled"
)

// ClaimDecision 表示 Agent 对某个事件是否可处理的判断。
type ClaimDecision struct {
	CanClaim   bool
	Confidence float64
	Reason     string
	Priority   int
}

// EventClaim 表示 Agent 对事件的认领记录。
type EventClaim struct {
	ID         ID          // 认领记录唯一标识
	EventID    ID          // 被认领的事件ID
	AgentID    ID          // 认领的Agent ID
	Confidence float64     // 置信度，表示Agent处理该事件的自信程度（0-1）
	Priority   int         // 优先级，数值越高优先级越高
	Reason     string      // 认领原因说明
	Status     ClaimStatus // 认领状态
}

// AgentInput 是 Runtime 装配后传递给 Agent 的完整输入。
type AgentInput struct {
	Event     Event         // 当前待处理的事件
	Room      Room          // 房间信息，包含会话上下文
	Session   Session       // 会话状态信息
	Snapshot  StateSnapshot // 系统状态快照
	Messages  []Message     // 历史消息列表
	Artifacts []Artifact    // 相关产物数据
	Tools     []ToolSpec    // 可用的工具列表
}

// MessageDraft 是 Agent 输出的群聊消息草稿。
type MessageDraft struct {
	Role    MessageRole // 消息角色，如用户、Agent、系统等
	Content string      // 消息内容文本
}

// ArtifactDraft 是 Agent 输出的结构化产物草稿。
type ArtifactDraft struct {
	Type ArtifactType // 产物类型，如候选目标、执行计划等
	Data any          // 产物数据内容
}

// EventDraft 是 Agent 建议发布的新事件草稿。
type EventDraft struct {
	Type         EventType      // 事件类型，定义事件的业务含义
	TargetAgent  ID             // 目标Agent ID，指定由哪个Agent处理（可选）
	DispatchMode DispatchMode   // 分发模式：独占(Exclusive)或广播(Broadcast)
	Priority     int            // 事件优先级，数值越高优先级越高
	Payload      any            // 事件载荷数据
	Metadata     map[string]any // 事件元数据，用于传递额外信息
}

// AgentDecision 表示 Agent 对业务的判断结果。
type AgentDecision string

const (
	DecisionSucceeded AgentDecision = "succeeded"
	DecisionRejected  AgentDecision = "rejected"
	DecisionFailed    AgentDecision = "failed"
)

// AgentOutput 是 Agent 一次事件处理的输出，由 Runtime 统一校验、提交和发布。
type AgentOutput struct {
	Status     AgentRunStatus  // Agent运行状态：成功、拒绝或失败
	Messages   []MessageDraft  // 输出的群聊消息列表
	Artifacts  []ArtifactDraft // 输出的结构化产物列表
	Events     []EventDraft    // 建议触发的后续事件列表
	Decision   AgentDecision   // Agent对业务的判断结果：成功、拒绝或失败
	TokenUsage TokenUsage      // Token使用情况统计
	Error      error           // 处理过程中的错误信息
}

// Agent 是所有创意协作 Agent 必须实现的统一接口。
type Agent interface {
	ID() ID
	Name() string
	Role() AgentRole
	CanHandle(ctx context.Context, event Event, room Room) ClaimDecision
	Handle(ctx context.Context, input AgentInput) AgentOutput
}

// AgentProfile 描述 Agent 的身份、上下文策略、工具策略和运行限制。
type AgentProfile struct {
	ID            ID            // Agent唯一标识
	Name          string        // Agent显示名称
	Role          AgentRole     // Agent角色类型
	Description   string        // Agent功能描述
	ContextPolicy ContextPolicy // 上下文策略配置
	ToolPolicy    ToolPolicy    // 工具使用策略配置
	MaxTurns      int           // 最大对话轮数限制
	MaxRetries    int           // 最大重试次数
}

// ContextPolicy 定义 Agent 可见的上下文范围。
type ContextPolicy struct {
	IncludeRoomMessages    bool           // 是否包含房间消息
	MessageWindow          int            // 消息窗口大小，限制可见历史消息数量
	IncludeSessionState    bool           // 是否包含会话状态
	IncludeUserProfile     bool           // 是否包含用户画像
	IncludeHistoricalTasks bool           // 是否包含历史任务
	IncludeArtifacts       []ArtifactType // 包含的产物类型列表
	IncludeVectorMemory    bool           // 是否包含向量记忆
	VectorQueryTemplates   []string       // 向量查询模板
	MaxContextTokens       int            // 最大上下文Token数限制
}

// ToolPolicy 定义 Agent 可用工具与调用限制。
type ToolPolicy struct {
	AllowedTools    []string // 允许使用的工具列表
	DeniedTools     []string // 禁止使用的工具列表
	RequireApproval []string // 需要审批的工具列表
	MaxCallsPerRun  int      // 单次运行最大工具调用次数
}

// ToolSpec 描述工具名称、能力和风险等级。
type ToolSpec struct {
	Name        string // 工具名称
	Description string // 工具功能描述
	RiskLevel   string // 风险等级：low/medium/high
}

// AgentRegistry 保存所有可参与协作的 Agent。
type AgentRegistry struct {
	agents map[ID]Agent
}

// NewAgentRegistry 创建 Agent 注册表。
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{agents: map[ID]Agent{}}
}

// Register 注册一个 Agent。
func (r *AgentRegistry) Register(agent Agent) {
	if r == nil || agent == nil {
		return
	}
	r.agents[agent.ID()] = agent
}

// Get 根据 ID 获取 Agent。
func (r *AgentRegistry) Get(id ID) Agent {
	if r == nil {
		return nil
	}
	return r.agents[id]
}

// List 返回所有 Agent。
func (r *AgentRegistry) List() []Agent {
	if r == nil {
		return nil
	}
	list := make([]Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		list = append(list, agent)
	}
	return list
}

// StaticAgent 是第一阶段可配置的轻量 Agent 实现，适合单元测试和流程打通。
type StaticAgent struct {
	id         ID                                            // Agent唯一标识
	name       string                                        // Agent显示名称
	role       AgentRole                                     // Agent角色类型
	handles    map[EventType]bool                            // 可处理的事件类型映射表
	outputFunc func(context.Context, AgentInput) AgentOutput // 输出处理函数
}

// NewStaticAgent 创建静态 Agent。
func NewStaticAgent(id ID, name string, role AgentRole, handles []EventType, outputFunc func(context.Context, AgentInput) AgentOutput) *StaticAgent {
	m := make(map[EventType]bool, len(handles))
	for _, eventType := range handles {
		m[eventType] = true
	}
	return &StaticAgent{id: id, name: name, role: role, handles: m, outputFunc: outputFunc}
}

func (a *StaticAgent) ID() ID          { return a.id }
func (a *StaticAgent) Name() string    { return a.name }
func (a *StaticAgent) Role() AgentRole { return a.role }

// CanHandle 根据静态订阅表判断当前 Agent 是否可以认领事件。
func (a *StaticAgent) CanHandle(ctx context.Context, event Event, room Room) ClaimDecision {
	if a.handles[event.Type] {
		return ClaimDecision{CanClaim: true, Confidence: 1, Reason: "事件类型匹配", Priority: event.Priority}
	}
	return ClaimDecision{CanClaim: false, Reason: "事件类型不匹配"}
}

// Handle 执行静态 Agent 的输出函数。
func (a *StaticAgent) Handle(ctx context.Context, input AgentInput) AgentOutput {
	if a.outputFunc == nil {
		return AgentOutput{Status: AgentRunFailed, Decision: DecisionFailed}
	}
	return a.outputFunc(ctx, input)
}
