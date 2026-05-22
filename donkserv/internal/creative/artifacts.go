package creative

import "time"

// CandidateGoal 表示目标创意 Agent 生成的候选目标。
type CandidateGoal struct {
	ID             ID        // 候选目标唯一标识
	Title          string    // 目标标题
	Description    string    // 目标描述
	Motivation     string    // 动机说明
	ContextBasis   []string  // 上下文依据
	ExpectedOutput string    // 预期产出
	Boundaries     []string  // 边界条件
	Risks          []string  // 风险列表
	CreatedBy      ID        // 创建者Agent ID
	CreatedAt      time.Time // 创建时间
}

// FinalExecutableGoal 表示目标收敛 Agent 输出的最终可执行目标。
type FinalExecutableGoal struct {
	ID               ID        // 最终目标唯一标识
	Title            string    // 目标标题
	Description      string    // 目标描述
	WhyNow           string    // 为什么是现在（背景说明）
	ContextBasis     []string  // 上下文依据
	ExpectedDelivery string    // 预期交付物
	Constraints      []string  // 约束条件
	AcceptedReviews  []ID      // 已接受的审核ID列表
	CreatedAt        time.Time // 创建时间
}

// ExecutablePlan 表示任务规划 Agent 输出的可执行计划。
type ExecutablePlan struct {
	ID           ID         // 计划唯一标识
	GoalID       ID         // 关联的目标ID
	Steps        []PlanStep // 执行步骤列表
	Dependencies []string   // 依赖项列表
	Risks        []string   // 风险列表
	ToolNeeds    []string   // 所需工具列表
	CreatedAt    time.Time  // 创建时间
}

// PlanStep 表示计划中的单个执行步骤。
type PlanStep struct {
	ID          ID     // 步骤唯一标识
	Title       string // 步骤标题
	Description string // 步骤描述
	Tool        string // 使用的工具
	DependsOn   []ID   // 依赖的步骤ID列表
}

// ExecutionResult 表示任务执行 Agent 的执行结果。
type ExecutionResult struct {
	ID          ID           // 执行结果唯一标识
	PlanID      ID           // 关联的计划ID
	StepResults []StepResult // 各步骤执行结果
	Outputs     []ID         // 输出产物ID列表
	Errors      []string     // 错误信息列表
	CreatedAt   time.Time    // 创建时间
}

// StepResult 表示计划步骤执行结果。
type StepResult struct {
	StepID ID     // 步骤ID
	Status string // 执行状态
	Output string // 输出内容
	Error  string // 错误信息（如果有）
}

// FinalDelivery 表示任务交付 Agent 的最终交付结果。
type FinalDelivery struct {
	ID            ID        // 交付结果唯一标识
	GoalID        ID        // 关联的目标ID
	PlanID        ID        // 关联的计划ID
	ResultID      ID        // 关联的执行结果ID
	Summary       string    // 交付摘要
	Deliverables  []ID      // 可交付物ID列表
	ArchiveRecord string    // 归档记录
	CreatedAt     time.Time // 创建时间
}
