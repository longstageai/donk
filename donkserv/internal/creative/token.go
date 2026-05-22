package creative

import "time"

// BudgetScope 表示 Token 预算生效范围。
type BudgetScope string

const (
	BudgetGlobal   BudgetScope = "global"
	BudgetSession  BudgetScope = "session"
	BudgetRoom     BudgetScope = "room"
	BudgetAgent    BudgetScope = "agent"
	BudgetEvent    BudgetScope = "event"
	BudgetAgentRun BudgetScope = "agent_run"
)

// TokenLimitAction 表示 Token 达到阈值后 Runtime 应执行的策略动作。
type TokenLimitAction string

const (
	TokenActionContinue        TokenLimitAction = "continue"
	TokenActionCompactContext  TokenLimitAction = "compact_context"
	TokenActionDowngradeModel  TokenLimitAction = "downgrade_model"
	TokenActionSkipOptional    TokenLimitAction = "skip_optional"
	TokenActionPauseSession    TokenLimitAction = "pause_session"
	TokenActionBlockSession    TokenLimitAction = "block_session"
	TokenActionStopSession     TokenLimitAction = "stop_session"
	TokenActionRequireApproval TokenLimitAction = "require_approval"
)

// TokenUsage 记录一次模型调用产生的 Token 消耗。
type TokenUsage struct {
	ID               ID        // 记录唯一标识
	SessionID        ID        // 关联的Session ID
	RoomID           ID        // 关联的Room ID
	EventID          ID        // 关联的Event ID
	AgentRunID       ID        // 关联的AgentRun ID
	AgentID          ID        // Agent ID
	ModelProvider    string    // 模型提供商
	ModelName        string    // 模型名称
	PromptTokens     int       // 提示词Token数
	CompletionTokens int       // 完成内容Token数
	TotalTokens      int       // 总Token数
	CachedTokens     int       // 缓存Token数
	ReasoningTokens  int       // 推理Token数
	EstimatedCost    float64   // 预估费用
	Currency         string    // 货币单位
	CreatedAt        time.Time // 创建时间
}

// TokenBudget 定义不同范围内允许消耗的 Token 与费用上限。
type TokenBudget struct {
	ID                  ID               // 预算唯一标识
	Scope               BudgetScope      // 预算生效范围
	ScopeID             ID               // 范围ID
	MaxPromptTokens     int              // 最大提示词Token数
	MaxCompletionTokens int              // 最大完成内容Token数
	MaxTotalTokens      int              // 最大总Token数
	MaxEstimatedCost    float64          // 最大预估费用
	WarnThresholdRatio  float64          // 警告阈值比例（如0.7表示70%）
	StopThresholdRatio  float64          // 停止阈值比例（如1.0表示100%）
	OnWarn              TokenLimitAction // 达到警告阈值时的动作
	OnStop              TokenLimitAction // 达到停止阈值时的动作
	CreatedAt           time.Time        // 创建时间
	UpdatedAt           time.Time        // 更新时间
}

// TokenUsageSummary 汇总一个 Session、Event 或 AgentRun 的 Token 消耗。
type TokenUsageSummary struct {
	PromptTokens     int            // 提示词Token总数
	CompletionTokens int            // 完成内容Token总数
	TotalTokens      int            // 总Token数
	EstimatedCost    float64        // 预估总费用
	ByAgent          map[string]int // 各Agent的Token消耗
	ByEventType      map[string]int // 各事件类型的Token消耗
	ByModel          map[string]int // 各模型的Token消耗
	BudgetRatio      float64        // 预算使用比例
	IsNearLimit      bool           // 是否接近限制
	IsExceeded       bool           // 是否已超出限制
}

// Clone 复制 Token 汇总，避免快照中的 map 被外部修改。
func (s TokenUsageSummary) Clone() TokenUsageSummary {
	return TokenUsageSummary{
		PromptTokens:     s.PromptTokens,
		CompletionTokens: s.CompletionTokens,
		TotalTokens:      s.TotalTokens,
		EstimatedCost:    s.EstimatedCost,
		ByAgent:          cloneStringIntMap(s.ByAgent),
		ByEventType:      cloneStringIntMap(s.ByEventType),
		ByModel:          cloneStringIntMap(s.ByModel),
		BudgetRatio:      s.BudgetRatio,
		IsNearLimit:      s.IsNearLimit,
		IsExceeded:       s.IsExceeded,
	}
}

// TokenBudgetDecision 表示 Token 预算检查结果。
type TokenBudgetDecision struct {
	Allowed      bool              // 是否允许继续
	Action       TokenLimitAction  // 建议执行的动作
	Reason       string            // 决策原因
	CurrentUsage TokenUsageSummary // 当前使用量
	Budget       TokenBudget       // 预算配置
}

// TokenBudgetGuard 负责在 Agent 调用前后以及每个 Tick 前检查 Token 预算。
type TokenBudgetGuard interface {
	BeforeBuildInput(req AgentRunRequest) TokenBudgetDecision
	BeforeInvoke(input AgentInput) TokenBudgetDecision
	AfterInvoke(usage TokenUsage) TokenBudgetDecision
	BeforeNextTick(snapshot StateSnapshot) TokenBudgetDecision
}

// SimpleTokenBudgetGuard 是内存版 Token 预算守卫，用于第一阶段实现与单元测试。
type SimpleTokenBudgetGuard struct {
	budget TokenBudget       // 预算配置
	usage  TokenUsageSummary // 当前使用量
}

// NewSimpleTokenBudgetGuard 创建 Token 预算守卫。
func NewSimpleTokenBudgetGuard(budget TokenBudget) *SimpleTokenBudgetGuard {
	if budget.WarnThresholdRatio <= 0 {
		budget.WarnThresholdRatio = 0.7
	}
	if budget.StopThresholdRatio <= 0 {
		budget.StopThresholdRatio = 1
	}
	if budget.OnWarn == "" {
		budget.OnWarn = TokenActionCompactContext
	}
	if budget.OnStop == "" {
		budget.OnStop = TokenActionBlockSession
	}
	return &SimpleTokenBudgetGuard{
		budget: budget,
		usage: TokenUsageSummary{
			ByAgent:     map[string]int{},
			ByEventType: map[string]int{},
			ByModel:     map[string]int{},
		},
	}
}

// BeforeBuildInput 在构建 Agent 输入前检查预算，避免已经超限后继续拼接大上下文。
func (g *SimpleTokenBudgetGuard) BeforeBuildInput(req AgentRunRequest) TokenBudgetDecision {
	return g.decision("构建 Agent 输入前检查 Token 预算")
}

// BeforeInvoke 在模型调用前检查预算，避免不必要的 LLM 调用。
func (g *SimpleTokenBudgetGuard) BeforeInvoke(input AgentInput) TokenBudgetDecision {
	return g.decision("模型调用前检查 Token 预算")
}

// AfterInvoke 记录本次模型调用的 Token 消耗，并返回最新预算决策。
func (g *SimpleTokenBudgetGuard) AfterInvoke(usage TokenUsage) TokenBudgetDecision {
	g.Record(usage)
	return g.decision("模型调用后检查 Token 预算")
}

// BeforeNextTick 在每个全局事件循环 Tick 前检查预算。
func (g *SimpleTokenBudgetGuard) BeforeNextTick(snapshot StateSnapshot) TokenBudgetDecision {
	return g.decision("进入下一 Tick 前检查 Token 预算")
}

// Record 累加一次 Token 消耗。
func (g *SimpleTokenBudgetGuard) Record(usage TokenUsage) {
	if g == nil {
		return
	}
	g.usage.PromptTokens += usage.PromptTokens
	g.usage.CompletionTokens += usage.CompletionTokens
	g.usage.TotalTokens += usage.TotalTokens
	g.usage.EstimatedCost += usage.EstimatedCost
	if usage.AgentID != "" {
		g.usage.ByAgent[string(usage.AgentID)] += usage.TotalTokens
	}
	if usage.ModelName != "" {
		g.usage.ByModel[usage.ModelName] += usage.TotalTokens
	}
	g.refreshRatio()
}

// Summary 返回当前 Token 消耗摘要。
func (g *SimpleTokenBudgetGuard) Summary() TokenUsageSummary {
	if g == nil {
		return TokenUsageSummary{}
	}
	return g.usage.Clone()
}

// Budget 返回当前预算配置。
func (g *SimpleTokenBudgetGuard) Budget() TokenBudget {
	if g == nil {
		return TokenBudget{}
	}
	return g.budget
}

// decision 根据当前消耗和预算阈值计算下一步动作。
func (g *SimpleTokenBudgetGuard) decision(reason string) TokenBudgetDecision {
	if g == nil || g.budget.MaxTotalTokens <= 0 {
		return TokenBudgetDecision{Allowed: true, Action: TokenActionContinue, Reason: reason}
	}
	g.refreshRatio()
	current := g.usage.Clone()
	if current.IsExceeded {
		return TokenBudgetDecision{Allowed: false, Action: g.budget.OnStop, Reason: reason, CurrentUsage: current, Budget: g.budget}
	}
	if current.IsNearLimit {
		return TokenBudgetDecision{Allowed: true, Action: g.budget.OnWarn, Reason: reason, CurrentUsage: current, Budget: g.budget}
	}
	return TokenBudgetDecision{Allowed: true, Action: TokenActionContinue, Reason: reason, CurrentUsage: current, Budget: g.budget}
}

// refreshRatio 刷新预算比例和阈值标记。
func (g *SimpleTokenBudgetGuard) refreshRatio() {
	if g.budget.MaxTotalTokens <= 0 {
		return
	}
	g.usage.BudgetRatio = float64(g.usage.TotalTokens) / float64(g.budget.MaxTotalTokens)
	g.usage.IsNearLimit = g.usage.BudgetRatio >= g.budget.WarnThresholdRatio
	g.usage.IsExceeded = g.usage.BudgetRatio >= g.budget.StopThresholdRatio
}

func cloneStringIntMap(src map[string]int) map[string]int {
	if src == nil {
		return map[string]int{}
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
