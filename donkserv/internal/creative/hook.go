package creative

// AgentRunRequest 表示 Runtime 准备执行某个 AgentRun 的请求。
type AgentRunRequest struct {
	Session Session // 会话信息
	Room    Room    // 房间信息
	Event   Event   // 待处理事件
	Agent   Agent   // 执行Agent
	RunID   ID      // 运行记录ID
}

// HookPipeline 定义 Agent 输入输出处理管线。
type HookPipeline interface {
	BeforeInput(req *AgentRunRequest) error
	AfterInput(input *AgentInput) error
	BeforeOutput(output *AgentOutput) error
	AfterOutput(result *AgentRun) error
}

// NoopHookPipeline 是默认空 Hook 实现，保留扩展点但不改变数据。
type NoopHookPipeline struct{}

func (NoopHookPipeline) BeforeInput(req *AgentRunRequest) error { return nil }
func (NoopHookPipeline) AfterInput(input *AgentInput) error     { return nil }
func (NoopHookPipeline) BeforeOutput(output *AgentOutput) error { return nil }
func (NoopHookPipeline) AfterOutput(result *AgentRun) error     { return nil }

// CompositeHookPipeline 将多个 HookPipeline 串联为一个管线。
type CompositeHookPipeline struct {
	hooks []HookPipeline // Hook管道列表
}

// NewCompositeHookPipeline 创建组合 Hook 管线，nil hook 会被自动忽略。
func NewCompositeHookPipeline(hooks ...HookPipeline) HookPipeline {
	filtered := make([]HookPipeline, 0, len(hooks))
	for _, hook := range hooks {
		if hook != nil {
			filtered = append(filtered, hook)
		}
	}
	if len(filtered) == 0 {
		return NoopHookPipeline{}
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return &CompositeHookPipeline{hooks: filtered}
}

// BeforeInput 按注册顺序执行所有 Hook，任一 Hook 返回错误则中断后续执行。
func (p *CompositeHookPipeline) BeforeInput(req *AgentRunRequest) error {
	for _, hook := range p.hooks {
		if err := hook.BeforeInput(req); err != nil {
			return err
		}
	}
	return nil
}

// AfterInput 按注册顺序执行所有输入构建后 Hook。
func (p *CompositeHookPipeline) AfterInput(input *AgentInput) error {
	for _, hook := range p.hooks {
		if err := hook.AfterInput(input); err != nil {
			return err
		}
	}
	return nil
}

// BeforeOutput 按注册顺序执行所有输出提交前 Hook。
func (p *CompositeHookPipeline) BeforeOutput(output *AgentOutput) error {
	for _, hook := range p.hooks {
		if err := hook.BeforeOutput(output); err != nil {
			return err
		}
	}
	return nil
}

// AfterOutput 按注册顺序执行所有 AgentRun 完成后 Hook。
func (p *CompositeHookPipeline) AfterOutput(result *AgentRun) error {
	for _, hook := range p.hooks {
		if err := hook.AfterOutput(result); err != nil {
			return err
		}
	}
	return nil
}

// ContextBuilder 根据 ContextPolicy 组装 Agent 输入。
type ContextBuilder struct {
	store *Store // 数据存储
}

// NewContextBuilder 创建上下文构建器。
func NewContextBuilder(store *Store) *ContextBuilder {
	return &ContextBuilder{store: store}
}

// Build 为 AgentRun 构建上下文。第一阶段主要注入 Room、Session、消息、产物和快照。
func (b *ContextBuilder) Build(req AgentRunRequest, snapshot StateSnapshot) AgentInput {
	messages := b.store.ListMessages(req.Room.ID, 20)
	artifacts := b.store.ListArtifacts(req.Session.ID)
	return AgentInput{
		Event:     req.Event,
		Room:      req.Room,
		Session:   req.Session,
		Snapshot:  snapshot,
		Messages:  messages,
		Artifacts: artifacts,
	}
}
