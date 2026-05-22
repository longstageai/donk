package creative

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"
)

var (
	ErrSessionNotFound = errors.New("creative session not found")
	ErrRoomNotFound    = errors.New("creative room not found")
	ErrLoopPaused      = errors.New("creative loop paused")
	ErrLoopStopped     = errors.New("creative loop stopped")
)

// StopMode 表示停止事件循环的方式。
type StopMode string

const (
	StopGraceful         StopMode = "graceful"
	StopAfterCurrentTick StopMode = "after_current_tick"
	StopImmediate        StopMode = "immediate"
)

// LoopControlType 表示循环控制命令类型。
type LoopControlType string

const (
	LoopCommandStart  LoopControlType = "start"
	LoopCommandPause  LoopControlType = "pause"
	LoopCommandResume LoopControlType = "resume"
	LoopCommandStop   LoopControlType = "stop"
	LoopCommandCancel LoopControlType = "cancel"
)

// LoopControlCommand 表示外部对事件循环发出的控制命令。
type LoopControlCommand struct {
	ID        ID              // 命令唯一标识
	SessionID ID              // 目标Session ID
	Type      LoopControlType // 命令类型：启动/暂停/恢复/停止/取消
	Mode      StopMode        // 停止模式（当Type为停止时有效）
	Reason    string          // 命令原因说明
	CreatedAt time.Time       // 命令创建时间
}

// RuntimeConfig 定义事件循环运行边界。
type RuntimeConfig struct {
	MaxTotalTicks         int // 最大总Tick数
	MaxSameEventRetries   int // 同一事件最大重试次数
	MaxPhaseRetries       int // 单个阶段最大重试次数
	MaxAgentFailures      int // Agent最大失败次数
	MaxGoalRegenerations  int // 目标最大重新生成次数
	MaxPlanRevisions      int // 计划最大修订次数
	MaxExecutionRevisions int // 执行最大修订次数
}

// DefaultRuntimeConfig 返回默认循环保护配置。
func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		MaxTotalTicks:         100,
		MaxSameEventRetries:   3,
		MaxPhaseRetries:       10,
		MaxAgentFailures:      5,
		MaxGoalRegenerations:  3,
		MaxPlanRevisions:      3,
		MaxExecutionRevisions: 3,
	}
}

// Runtime 是事件循环驱动的多 Agent 协作运行时。
type Runtime struct {
	store          *Store                    // 数据存储
	registry       *AgentRegistry            // Agent注册表
	contextBuilder *ContextBuilder           // 上下文构建器
	hooks          HookPipeline              // Hook管道
	notifier       RuntimeNotifier           // 运行时通知器
	tokenGuard     *SimpleTokenBudgetGuard   // Token预算守卫
	config         RuntimeConfig             // 运行时配置
	controlMu      sync.RWMutex              // 控制命令锁
	commands       map[ID]LoopControlCommand // 控制命令映射
}

// NewRuntime 创建创意多 Agent Runtime。
func NewRuntime(registry *AgentRegistry, opts ...RuntimeOption) *Runtime {
	store := NewStore()
	if registry == nil {
		registry = NewAgentRegistry()
	}
	r := &Runtime{
		store:          store,
		registry:       registry,
		contextBuilder: NewContextBuilder(store),
		hooks:          NoopHookPipeline{},
		notifier:       NoopRuntimeNotifier{},
		config:         DefaultRuntimeConfig(),
		commands:       map[ID]LoopControlCommand{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RuntimeOption 用于配置 Runtime。
type RuntimeOption func(*Runtime)

func WithStore(store *Store) RuntimeOption {
	return func(r *Runtime) {
		if store != nil {
			r.store = store
			r.contextBuilder = NewContextBuilder(store)
		}
	}
}

func WithHookPipeline(hooks HookPipeline) RuntimeOption {
	return func(r *Runtime) {
		if hooks != nil {
			r.hooks = hooks
		}
	}
}

// WithHookPipelines 配置多个 HookPipeline，并按传入顺序串联执行。
func WithHookPipelines(hooks ...HookPipeline) RuntimeOption {
	return func(r *Runtime) {
		r.hooks = NewCompositeHookPipeline(hooks...)
	}
}

func WithRuntimeNotifier(notifier RuntimeNotifier) RuntimeOption {
	return func(r *Runtime) {
		if notifier != nil {
			r.notifier = notifier
		}
	}
}

// WithRuntimeNotifiers 配置多个运行时通知器，并按传入顺序广播。
func WithRuntimeNotifiers(notifiers ...RuntimeNotifier) RuntimeOption {
	return func(r *Runtime) {
		r.notifier = NewCompositeRuntimeNotifier(notifiers...)
	}
}

func WithTokenGuard(guard *SimpleTokenBudgetGuard) RuntimeOption {
	return func(r *Runtime) { r.tokenGuard = guard }
}

func WithRuntimeConfig(config RuntimeConfig) RuntimeOption {
	return func(r *Runtime) { r.config = config }
}

// Store 返回底层存储，便于上层查询状态和测试断言。
func (r *Runtime) Store() *Store { return r.store }

// StartSession 创建 Session、Room 并发布根事件。
func (r *Runtime) StartSession(ctx context.Context, trigger Trigger) (ID, error) {
	now := time.Now()
	sessionID := nextID("session")
	roomID := nextID("room")
	rootEventID := nextID("event")

	session := Session{
		ID:             sessionID,
		TriggerType:    trigger.Type,
		TriggerPayload: trigger.Payload,
		Status:         SessionRunning,
		LoopStatus:     LoopCreated,
		CurrentPhase:   PhaseGoalGeneration,
		RoomID:         roomID,
		RootEventID:    rootEventID,
		CurrentEventID: rootEventID,
		FailureCounts:  map[string]int{},
		RetryCounts:    map[string]int{},
		StartedAt:      now,
	}
	if r.tokenGuard != nil {
		session.TokenBudget = r.tokenGuard.Budget()
	}
	room := Room{ID: roomID, SessionID: sessionID, Topic: "多 Agent 创意目标生成", Status: "running", CreatedAt: now, UpdatedAt: now}
	rootEvent := Event{ID: rootEventID, Type: EventGoalRequested, RoomID: roomID, SessionID: sessionID, CorrelationID: rootEventID, Status: EventPending, DispatchMode: DispatchExclusive, Priority: 100, Payload: trigger.Payload, CreatedAt: now, UpdatedAt: now}

	r.store.SaveSession(session)
	r.store.SaveRoom(room)
	r.store.SaveEvent(rootEvent)
	r.notify(ctx, RuntimeStreamSessionStarted, session, RuntimeSeverityInfo, map[string]any{"trigger_type": trigger.Type})
	r.notify(ctx, RuntimeStreamEventCreated, session, RuntimeSeverityInfo, rootEvent)
	r.publishSystemEvent(sessionID, roomID, EventSessionStarted, "Session 已启动", rootEventID)
	r.logInfo("创意多 Agent Session 已创建", map[string]interface{}{"session_id": sessionID, "room_id": roomID})
	return sessionID, nil
}

// StartLoop 启动或继续指定 Session 的事件循环。
func (r *Runtime) StartLoop(ctx context.Context, sessionID ID) error {
	if !r.store.UpdateSession(sessionID, func(session *Session) { session.LoopStatus = LoopRunning }) {
		return ErrSessionNotFound
	}
	if session, ok := r.store.GetSession(sessionID); ok {
		r.notify(ctx, RuntimeStreamLoopStarted, session, RuntimeSeverityInfo, nil)
	}
	r.logInfo("创意多 Agent 事件循环启动", map[string]interface{}{"session_id": sessionID})
	return r.RunSession(ctx, sessionID)
}

// RunSession 执行全局事件循环，直到完成、阻塞、暂停或停止。
func (r *Runtime) RunSession(ctx context.Context, sessionID ID) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.applyControlIfNeeded(sessionID); err != nil {
			return err
		}
		session, ok := r.store.GetSession(sessionID)
		if !ok {
			return ErrSessionNotFound
		}
		if session.Status == SessionCompleted || session.Status == SessionFailed || session.Status == SessionBlocked || session.Status == SessionCancelled {
			return nil
		}
		if r.config.MaxTotalTicks > 0 && session.Tick >= r.config.MaxTotalTicks {
			r.blockSession(sessionID, "超过最大 Tick 限制")
			return nil
		}

		r.notify(ctx, RuntimeStreamTickStarted, session, RuntimeSeverityDebug, map[string]any{"tick": session.Tick})
		snapshot := r.TakeSnapshot(sessionID)
		r.notifySnapshot(ctx, snapshot)
		if r.tokenGuard != nil {
			decision := r.tokenGuard.BeforeNextTick(snapshot)
			if !decision.Allowed {
				r.notifyTokenDecision(ctx, session, decision)
				r.applyTokenDecision(sessionID, decision)
				return nil
			}
			if decision.Action != TokenActionContinue {
				r.notifyTokenDecision(ctx, session, decision)
				r.publishTokenDecisionEvent(session, decision)
			}
		}

		events := r.store.ListPendingEvents(sessionID)
		if len(events) == 0 {
			// 没有待处理事件时，如果 Session 未完成，说明流程进入空闲阻塞态，防止空转。
			r.blockSession(sessionID, "没有待处理事件")
			return nil
		}

		for _, event := range events {
			if err := r.applyControlIfNeeded(sessionID); err != nil {
				return err
			}
			if err := r.processEvent(ctx, sessionID, event); err != nil {
				return err
			}
		}
		if latest, ok := r.store.GetSession(sessionID); ok {
			r.notify(ctx, RuntimeStreamTickCompleted, latest, RuntimeSeverityDebug, map[string]any{"tick": latest.Tick})
		}
	}
}

// PauseLoop 请求暂停循环。当前实现会在下一个安全控制点暂停。
func (r *Runtime) PauseLoop(ctx context.Context, sessionID ID, reason string) error {
	cmd := LoopControlCommand{ID: nextID("cmd"), SessionID: sessionID, Type: LoopCommandPause, Reason: reason, CreatedAt: time.Now()}
	r.setCommand(sessionID, cmd)
	if session, ok := r.store.GetSession(sessionID); ok {
		r.notify(ctx, RuntimeStreamControlReceived, session, RuntimeSeverityInfo, cmd)
	}
	return nil
}

// ResumeLoop 恢复暂停状态，并继续处理 pending 事件。
func (r *Runtime) ResumeLoop(ctx context.Context, sessionID ID) error {
	if !r.store.UpdateSession(sessionID, func(session *Session) { session.LoopStatus = LoopRunning }) {
		return ErrSessionNotFound
	}
	r.clearCommand(sessionID)
	if session, ok := r.store.GetSession(sessionID); ok {
		r.notify(ctx, RuntimeStreamLoopResumed, session, RuntimeSeverityInfo, nil)
	}
	r.publishSystemEventForSession(sessionID, EventLoopResumed, "事件循环已恢复")
	return r.RunSession(ctx, sessionID)
}

// StopLoop 请求停止循环。
func (r *Runtime) StopLoop(ctx context.Context, sessionID ID, mode StopMode, reason string) error {
	cmd := LoopControlCommand{ID: nextID("cmd"), SessionID: sessionID, Type: LoopCommandStop, Mode: mode, Reason: reason, CreatedAt: time.Now()}
	r.setCommand(sessionID, cmd)
	if session, ok := r.store.GetSession(sessionID); ok {
		r.notify(ctx, RuntimeStreamControlReceived, session, RuntimeSeverityInfo, cmd)
	}
	return nil
}

// CancelSession 取消整个 Session。
func (r *Runtime) CancelSession(ctx context.Context, sessionID ID, reason string) error {
	if !r.store.UpdateSession(sessionID, func(session *Session) {
		session.Status = SessionCancelled
		session.LoopStatus = LoopStopped
		now := time.Now()
		session.CompletedAt = &now
	}) {
		return ErrSessionNotFound
	}
	if session, ok := r.store.GetSession(sessionID); ok {
		r.notify(ctx, RuntimeStreamSessionCancelled, session, RuntimeSeverityWarn, map[string]any{"reason": reason})
	}
	r.publishSystemEventForSession(sessionID, EventSessionCancelled, reason)
	return nil
}

// processEvent 完成一次事件的认领、AgentRun 执行、输出提交和状态推进。
func (r *Runtime) processEvent(ctx context.Context, sessionID ID, event Event) error {
	session, ok := r.store.GetSession(sessionID)
	if !ok {
		return ErrSessionNotFound
	}
	room, ok := r.store.GetRoom(event.RoomID)
	if !ok {
		return ErrRoomNotFound
	}
	r.notify(ctx, RuntimeStreamEventProcessing, session, RuntimeSeverityDebug, event)
	agent, claim := r.selectAgent(ctx, event, room)
	if agent == nil {
		r.store.UpdateEvent(event.ID, func(e *Event) { e.Status = EventSkipped })
		r.notifyEventStatus(ctx, sessionID, event.ID, RuntimeStreamEventSkipped, RuntimeSeverityWarn)
		r.incrementFailure(sessionID, "no_agent")
		return nil
	}

	runID := nextID("run")
	run := AgentRun{ID: runID, SessionID: sessionID, RoomID: event.RoomID, EventID: event.ID, AgentID: agent.ID(), Status: AgentRunRunning, StartedAt: time.Now()}
	r.store.SaveAgentRun(run)
	r.store.UpdateEvent(event.ID, func(e *Event) {
		e.Status = EventClaimed
		e.TargetAgentID = agent.ID()
	})
	r.notify(ctx, RuntimeStreamEventClaimed, session, RuntimeSeverityInfo, claim, withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	r.notify(ctx, RuntimeStreamAgentRunStarted, session, RuntimeSeverityInfo, run, withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	r.logInfo("事件已被 Agent 认领", map[string]interface{}{"event_id": event.ID, "event_type": event.Type, "agent_id": agent.ID(), "confidence": claim.Confidence})

	req := AgentRunRequest{Session: session, Room: room, Event: event, Agent: agent, RunID: runID}
	if err := r.hooks.BeforeInput(&req); err != nil {
		r.failAgentRunAndEvent(ctx, session, runID, event.ID, agent.ID(), err)
		r.notifyRuntimeError(ctx, session, event.ID, "before_input_hook_failed", "Agent 输入前置 Hook 执行失败", err, true)
		return err
	}
	input := r.contextBuilder.Build(req, r.BuildSnapshot(sessionID))
	if err := r.hooks.AfterInput(&input); err != nil {
		r.failAgentRunAndEvent(ctx, session, runID, event.ID, agent.ID(), err)
		r.notifyRuntimeError(ctx, session, event.ID, "after_input_hook_failed", "Agent 输入后置 Hook 执行失败", err, true)
		return err
	}
	r.notify(ctx, RuntimeStreamAgentInputBuilt, session, RuntimeSeverityDebug, buildAgentInputView(input, agent.ID(), runID), withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	if r.tokenGuard != nil {
		decision := r.tokenGuard.BeforeInvoke(input)
		if !decision.Allowed {
			r.notifyTokenDecision(ctx, session, decision)
			r.applyTokenDecision(sessionID, decision)
			return nil
		}
	}

	r.store.UpdateEvent(event.ID, func(e *Event) { e.Status = EventProcessing })
	r.notify(ctx, RuntimeStreamEventProcessing, session, RuntimeSeverityInfo, event, withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	output := agent.Handle(ctx, input)
	if err := r.hooks.BeforeOutput(&output); err != nil {
		r.failAgentRunAndEvent(ctx, session, runID, event.ID, agent.ID(), err)
		r.notifyRuntimeError(ctx, session, event.ID, "before_output_hook_failed", "Agent 输出前置 Hook 执行失败", err, true)
		return err
	}
	r.notify(ctx, RuntimeStreamAgentOutputReady, session, RuntimeSeverityDebug, buildAgentOutputView(output), withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	if output.TokenUsage.TotalTokens > 0 && r.tokenGuard != nil {
		output.TokenUsage.SessionID = sessionID
		output.TokenUsage.RoomID = event.RoomID
		output.TokenUsage.EventID = event.ID
		output.TokenUsage.AgentRunID = runID
		output.TokenUsage.AgentID = agent.ID()
		decision := r.tokenGuard.AfterInvoke(output.TokenUsage)
		r.recordTokenUsage(ctx, sessionID, event.Type, output.TokenUsage)
		if decision.Action != TokenActionContinue {
			r.notifyTokenDecision(ctx, session, decision)
			r.publishTokenDecisionEvent(session, decision)
		}
	}

	r.commitOutput(ctx, sessionID, event, agent, runID, output)
	completedAt := time.Now()
	r.store.UpdateAgentRun(runID, func(ar *AgentRun) {
		ar.Status = output.Status
		ar.TokenUsage = output.TokenUsageSummary()
		ar.CompletedAt = &completedAt
		if output.Error != nil {
			ar.Error = output.Error.Error()
		}
	})
	finalRun, _ := r.latestRun(runID)
	if err := r.hooks.AfterOutput(&finalRun); err != nil {
		r.failAgentRunAndEvent(ctx, session, runID, event.ID, agent.ID(), err)
		r.notifyRuntimeError(ctx, session, event.ID, "after_output_hook_failed", "Agent 输出后置 Hook 执行失败", err, true)
		return err
	}
	if finalRun.Status == AgentRunFailed {
		r.notify(ctx, RuntimeStreamAgentRunFailed, session, RuntimeSeverityError, finalRun, withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	} else {
		r.notify(ctx, RuntimeStreamAgentRunCompleted, session, RuntimeSeverityInfo, finalRun, withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	}
	snapshot := r.TakeSnapshot(sessionID)
	r.notifySnapshot(ctx, snapshot)
	return nil
}

// selectAgent 根据 Agent 的 CanHandle 结果选择最合适的认领者。
func (r *Runtime) selectAgent(ctx context.Context, event Event, room Room) (Agent, EventClaim) {
	var selected Agent
	best := EventClaim{Status: ClaimRejected}
	for _, agent := range r.registry.List() {
		decision := agent.CanHandle(ctx, event, room)
		if !decision.CanClaim {
			continue
		}
		claim := EventClaim{ID: nextID("claim"), EventID: event.ID, AgentID: agent.ID(), Confidence: decision.Confidence, Priority: decision.Priority, Reason: decision.Reason, Status: ClaimProposed}
		if selected == nil || claim.Priority > best.Priority || (claim.Priority == best.Priority && claim.Confidence > best.Confidence) {
			selected = agent
			best = claim
		}
	}
	if selected != nil {
		best.Status = ClaimAccepted
	}
	return selected, best
}

// commitOutput 将 Agent 输出统一提交为消息、产物和后续事件。
func (r *Runtime) commitOutput(ctx context.Context, sessionID ID, event Event, agent Agent, runID ID, output AgentOutput) {
	status := EventSucceeded
	if output.Status == AgentRunRejected {
		status = EventRejected
	}
	if output.Status == AgentRunFailed || output.Error != nil {
		status = EventFailed
		r.incrementFailure(sessionID, string(agent.Role()))
	}
	r.store.UpdateEvent(event.ID, func(e *Event) { e.Status = status })
	switch status {
	case EventSucceeded:
		r.notifyEventStatus(ctx, sessionID, event.ID, RuntimeStreamEventSucceeded, RuntimeSeverityInfo)
	case EventRejected:
		r.notifyEventStatus(ctx, sessionID, event.ID, RuntimeStreamEventRejected, RuntimeSeverityWarn)
	case EventFailed:
		r.notifyEventStatus(ctx, sessionID, event.ID, RuntimeStreamEventFailed, RuntimeSeverityError)
	}
	artifactIDs := make([]ID, 0, len(output.Artifacts))
	for _, draft := range output.Artifacts {
		artifact := Artifact{ID: nextID("artifact"), RoomID: event.RoomID, SessionID: sessionID, EventID: event.ID, AgentID: agent.ID(), Type: draft.Type, Data: draft.Data, CreatedAt: time.Now()}
		r.store.SaveArtifact(artifact)
		artifactIDs = append(artifactIDs, artifact.ID)
		r.updateSessionArtifactRef(sessionID, artifact)
		r.notify(ctx, RuntimeStreamArtifactCreated, Session{ID: sessionID, RoomID: event.RoomID}, RuntimeSeverityInfo, artifact, withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	}
	for _, draft := range output.Messages {
		message := Message{ID: nextID("message"), RoomID: event.RoomID, SessionID: sessionID, EventID: event.ID, AgentID: agent.ID(), Role: draft.Role, Content: draft.Content, ArtifactIDs: artifactIDs, CreatedAt: time.Now()}
		r.store.SaveMessage(message)
		r.notify(ctx, RuntimeStreamMessageCreated, Session{ID: sessionID, RoomID: event.RoomID}, RuntimeSeverityInfo, message, withEventID(event.ID), withAgentID(agent.ID()), withRunID(runID))
	}
	for _, draft := range output.Events {
		newEvent := Event{ID: nextID("event"), Type: draft.Type, RoomID: event.RoomID, SessionID: sessionID, CorrelationID: event.CorrelationID, CausationID: event.ID, SourceAgentID: agent.ID(), TargetAgentID: draft.TargetAgent, Status: EventPending, DispatchMode: draft.DispatchMode, Priority: draft.Priority, Payload: draft.Payload, Metadata: draft.Metadata, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if newEvent.DispatchMode == "" {
			newEvent.DispatchMode = DispatchExclusive
		}
		r.store.SaveEvent(newEvent)
		r.notify(ctx, RuntimeStreamEventCreated, Session{ID: sessionID, RoomID: event.RoomID}, RuntimeSeverityInfo, newEvent, withEventID(newEvent.ID), withAgentID(agent.ID()), withRunID(runID))
		r.applyPhaseByEvent(sessionID, newEvent.Type)
	}
	if output.Status == AgentRunSucceeded {
		r.applyPhaseByEvent(sessionID, event.Type)
	}
}

// BuildSnapshot 构建当前 Session 状态快照，但不保存快照，也不推进 Tick。
func (r *Runtime) BuildSnapshot(sessionID ID) StateSnapshot {
	session, _ := r.store.GetSession(sessionID)
	eventStatus := map[ID]EventStatus{}
	agentStatus := map[ID]AgentRunStatus{}
	artifactStatus := map[ID]string{}
	for _, event := range r.store.ListEvents(sessionID) {
		eventStatus[event.ID] = event.Status
	}
	for _, run := range r.store.ListAgentRuns(sessionID) {
		agentStatus[run.ID] = run.Status
	}
	for _, artifact := range r.store.ListArtifacts(sessionID) {
		artifactStatus[artifact.ID] = "created"
	}
	usage := session.TokenUsage.Clone()
	if r.tokenGuard != nil {
		usage = r.tokenGuard.Summary()
	}
	return StateSnapshot{ID: nextID("snapshot"), RoomID: session.RoomID, SessionID: sessionID, Tick: session.Tick, CurrentPhase: session.CurrentPhase, CurrentEventID: session.CurrentEventID, EventStatus: eventStatus, AgentStatus: agentStatus, ArtifactStatus: artifactStatus, LatestGoalID: session.FinalGoalID, LatestPlanID: session.PlanID, LatestResultID: session.ExecutionID, LatestDeliveryID: session.DeliveryID, FailureCounts: cloneStringIntMap(session.FailureCounts), LoopCounts: cloneStringIntMap(session.RetryCounts), TokenUsage: usage, CreatedAt: time.Now()}
}

// TakeSnapshot 创建并保存当前 Session 状态快照，同时推进一次全局 Tick。
func (r *Runtime) TakeSnapshot(sessionID ID) StateSnapshot {
	snapshot := r.BuildSnapshot(sessionID)
	r.store.SaveSnapshot(snapshot)
	r.store.UpdateSession(sessionID, func(s *Session) {
		s.Tick++
		s.TokenUsage = snapshot.TokenUsage
	})
	return snapshot
}

func (r *Runtime) applyPhaseByEvent(sessionID ID, eventType EventType) {
	completed := false
	r.store.UpdateSession(sessionID, func(session *Session) {
		switch eventType {
		case EventGoalRequested, EventCandidateGoalCreated, EventGoalRegenerationRequested, EventGoalRefinementRequested:
			session.CurrentPhase = PhaseGoalGeneration
		case EventGoalDedupRequested, EventGoalValueReviewRequested, EventGoalFeasibilityRequested:
			session.CurrentPhase = PhaseGoalReview
		case EventGoalConvergenceRequested, EventFinalGoalCreated:
			session.CurrentPhase = PhaseGoalConvergence
		case EventPlanRequested, EventPlanCreated, EventPlanRevisionRequested:
			session.CurrentPhase = PhasePlanning
		case EventPlanReviewRequested, EventPlanReviewPassed, EventPlanReviewRejected:
			session.CurrentPhase = PhasePlanReview
		case EventExecutionRequested, EventExecutionStepStarted, EventExecutionStepCompleted, EventExecutionCompleted, EventExecutionRevisionRequested:
			session.CurrentPhase = PhaseExecution
		case EventResultReviewRequested, EventResultReviewPassed, EventResultReviewRejected:
			session.CurrentPhase = PhaseResultReview
		case EventDeliveryRequested:
			session.CurrentPhase = PhaseDelivery
		case EventDeliveryCompleted, EventLoopCompleted, EventSessionCompleted:
			wasCompleted := session.Status == SessionCompleted
			now := time.Now()
			session.Status = SessionCompleted
			session.LoopStatus = LoopStopped
			session.CurrentPhase = PhaseCompleted
			session.CompletedAt = &now
			completed = !wasCompleted
		case EventSessionBlocked:
			session.Status = SessionBlocked
			session.CurrentPhase = PhaseBlocked
		}
	})
	if completed {
		if session, ok := r.store.GetSession(sessionID); ok {
			r.notify(context.Background(), RuntimeStreamSessionCompleted, session, RuntimeSeverityInfo, map[string]any{"event_type": eventType})
			r.notify(context.Background(), RuntimeStreamLoopStopped, session, RuntimeSeverityInfo, map[string]any{"event_type": eventType})
		}
	}
}

func (r *Runtime) updateSessionArtifactRef(sessionID ID, artifact Artifact) {
	r.store.UpdateSession(sessionID, func(session *Session) {
		switch artifact.Type {
		case ArtifactFinalExecutableGoal:
			session.FinalGoalID = artifact.ID
		case ArtifactExecutablePlan:
			session.PlanID = artifact.ID
		case ArtifactExecutionResult:
			session.ExecutionID = artifact.ID
		case ArtifactFinalDelivery:
			session.DeliveryID = artifact.ID
		}
	})
}

func (r *Runtime) failAgentRunAndEvent(ctx context.Context, session Session, runID ID, eventID ID, agentID ID, err error) {
	completedAt := time.Now()
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	r.store.UpdateAgentRun(runID, func(run *AgentRun) {
		run.Status = AgentRunFailed
		run.Error = errorMessage
		run.CompletedAt = &completedAt
	})
	r.store.UpdateEvent(eventID, func(event *Event) { event.Status = EventFailed })
	if failedRun, ok := r.latestRun(runID); ok {
		r.notify(ctx, RuntimeStreamAgentRunFailed, session, RuntimeSeverityError, failedRun, withEventID(eventID), withAgentID(agentID), withRunID(runID))
	}
	r.notifyEventStatus(ctx, session.ID, eventID, RuntimeStreamEventFailed, RuntimeSeverityError)
}

func (r *Runtime) applyTokenDecision(sessionID ID, decision TokenBudgetDecision) {
	switch decision.Action {
	case TokenActionPauseSession, TokenActionRequireApproval:
		r.store.UpdateSession(sessionID, func(session *Session) { session.LoopStatus = LoopPaused })
		r.publishSystemEventForSession(sessionID, EventTokenSessionPauseRequested, decision.Reason)
	case TokenActionStopSession, TokenActionBlockSession:
		r.blockSession(sessionID, decision.Reason)
	default:
		// 压缩上下文、降级模型、跳过可选 Agent 等动作在第一阶段仅记录事件，保留后续扩展点。
		r.publishSystemEventForSession(sessionID, EventTokenBudgetWarning, decision.Reason)
	}
}

func (r *Runtime) publishTokenDecisionEvent(session Session, decision TokenBudgetDecision) {
	eventType := EventTokenBudgetWarning
	if decision.CurrentUsage.IsExceeded {
		eventType = EventTokenBudgetExceeded
	}
	r.publishSystemEvent(session.ID, session.RoomID, eventType, decision.Reason, session.CurrentEventID)
}

func (r *Runtime) recordTokenUsage(ctx context.Context, sessionID ID, eventType EventType, usage TokenUsage) {
	r.store.UpdateSession(sessionID, func(session *Session) {
		session.TokenUsage.PromptTokens += usage.PromptTokens
		session.TokenUsage.CompletionTokens += usage.CompletionTokens
		session.TokenUsage.TotalTokens += usage.TotalTokens
		session.TokenUsage.EstimatedCost += usage.EstimatedCost
		if session.TokenUsage.ByAgent == nil {
			session.TokenUsage.ByAgent = map[string]int{}
		}
		if session.TokenUsage.ByEventType == nil {
			session.TokenUsage.ByEventType = map[string]int{}
		}
		if session.TokenUsage.ByModel == nil {
			session.TokenUsage.ByModel = map[string]int{}
		}
		session.TokenUsage.ByAgent[string(usage.AgentID)] += usage.TotalTokens
		session.TokenUsage.ByEventType[string(eventType)] += usage.TotalTokens
		if usage.ModelName != "" {
			session.TokenUsage.ByModel[usage.ModelName] += usage.TotalTokens
		}
	})
	if session, ok := r.store.GetSession(sessionID); ok {
		r.notify(ctx, RuntimeStreamTokenUpdated, session, RuntimeSeverityInfo, usage)
	}
	r.publishSystemEventForSession(sessionID, EventTokenUsageRecorded, "Token 消耗已记录")
}

func (r *Runtime) blockSession(sessionID ID, reason string) {
	r.store.UpdateSession(sessionID, func(session *Session) {
		session.Status = SessionBlocked
		session.LoopStatus = LoopStopped
		session.CurrentPhase = PhaseBlocked
		now := time.Now()
		session.CompletedAt = &now
	})
	if session, ok := r.store.GetSession(sessionID); ok {
		r.notify(context.Background(), RuntimeStreamSessionBlocked, session, RuntimeSeverityError, map[string]any{"reason": reason})
		r.notify(context.Background(), RuntimeStreamLoopStopped, session, RuntimeSeverityWarn, map[string]any{"reason": reason})
	}
	r.publishSystemEventForSession(sessionID, EventSessionBlocked, reason)
	r.logWarn("创意多 Agent Session 已阻塞", map[string]interface{}{"session_id": sessionID, "reason": reason})
}

func (r *Runtime) incrementFailure(sessionID ID, key string) {
	r.store.UpdateSession(sessionID, func(session *Session) {
		if session.FailureCounts == nil {
			session.FailureCounts = map[string]int{}
		}
		session.FailureCounts[key]++
	})
}

func (r *Runtime) setCommand(sessionID ID, cmd LoopControlCommand) {
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	r.commands[sessionID] = cmd
}

func (r *Runtime) clearCommand(sessionID ID) {
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	delete(r.commands, sessionID)
}

func (r *Runtime) applyControlIfNeeded(sessionID ID) error {
	r.controlMu.RLock()
	cmd, ok := r.commands[sessionID]
	r.controlMu.RUnlock()
	if !ok {
		return nil
	}
	switch cmd.Type {
	case LoopCommandPause:
		r.store.UpdateSession(sessionID, func(session *Session) { session.LoopStatus = LoopPaused })
		if session, ok := r.store.GetSession(sessionID); ok {
			r.notify(context.Background(), RuntimeStreamLoopPaused, session, RuntimeSeverityInfo, map[string]any{"reason": cmd.Reason})
		}
		r.publishSystemEventForSession(sessionID, EventLoopPaused, cmd.Reason)
		return ErrLoopPaused
	case LoopCommandStop:
		r.store.UpdateSession(sessionID, func(session *Session) { session.LoopStatus = LoopStopped })
		if session, ok := r.store.GetSession(sessionID); ok {
			r.notify(context.Background(), RuntimeStreamLoopStopped, session, RuntimeSeverityWarn, map[string]any{"reason": cmd.Reason, "mode": cmd.Mode})
		}
		r.publishSystemEventForSession(sessionID, EventLoopStopped, cmd.Reason)
		return ErrLoopStopped
	case LoopCommandCancel:
		_ = r.CancelSession(context.Background(), sessionID, cmd.Reason)
		return ErrLoopStopped
	}
	return nil
}

func (r *Runtime) publishSystemEventForSession(sessionID ID, eventType EventType, reason string) {
	session, ok := r.store.GetSession(sessionID)
	if !ok {
		return
	}
	r.publishSystemEvent(session.ID, session.RoomID, eventType, reason, session.CurrentEventID)
}

func (r *Runtime) publishSystemEvent(sessionID, roomID ID, eventType EventType, reason string, causeID ID) {
	event := Event{ID: nextID("event"), Type: eventType, RoomID: roomID, SessionID: sessionID, CausationID: causeID, Status: EventSucceeded, DispatchMode: DispatchExclusive, Priority: 0, Payload: map[string]any{"reason": reason}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	r.store.SaveEvent(event)
	r.notify(context.Background(), RuntimeStreamEventCreated, Session{ID: sessionID, RoomID: roomID}, RuntimeSeverityDebug, event, withEventID(event.ID))
}

func (r *Runtime) latestRun(runID ID) (AgentRun, bool) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	run, ok := r.store.runs[runID]
	return run, ok
}

func (o AgentOutput) TokenUsageSummary() TokenUsageSummary {
	return TokenUsageSummary{PromptTokens: o.TokenUsage.PromptTokens, CompletionTokens: o.TokenUsage.CompletionTokens, TotalTokens: o.TokenUsage.TotalTokens, EstimatedCost: o.TokenUsage.EstimatedCost, ByAgent: map[string]int{}, ByEventType: map[string]int{}, ByModel: map[string]int{}}
}

type runtimeNotifyOption func(*RuntimeStreamEvent)

func withEventID(eventID ID) runtimeNotifyOption {
	return func(event *RuntimeStreamEvent) { event.EventID = eventID }
}

func withAgentID(agentID ID) runtimeNotifyOption {
	return func(event *RuntimeStreamEvent) { event.AgentID = agentID }
}

func withRunID(runID ID) runtimeNotifyOption {
	return func(event *RuntimeStreamEvent) { event.RunID = runID }
}

func (r *Runtime) notify(ctx context.Context, eventType RuntimeStreamEventType, session Session, severity RuntimeEventSeverity, payload any, opts ...runtimeNotifyOption) {
	if r.notifier == nil {
		return
	}
	event := RuntimeStreamEvent{ID: nextID("stream"), Type: eventType, SessionID: session.ID, RoomID: session.RoomID, Tick: session.Tick, Phase: session.CurrentPhase, Severity: severity, Payload: payload, CreatedAt: time.Now()}
	for _, opt := range opts {
		opt(&event)
	}
	safeNotifyRuntimeNotifier(ctx, r.notifier, event)
}

func (r *Runtime) notifySnapshot(ctx context.Context, snapshot StateSnapshot) {
	r.notify(ctx, RuntimeStreamSnapshotCreated, Session{ID: snapshot.SessionID, RoomID: snapshot.RoomID, Tick: snapshot.Tick, CurrentPhase: snapshot.CurrentPhase}, RuntimeSeverityDebug, snapshot)
}

func (r *Runtime) notifyEventStatus(ctx context.Context, sessionID ID, eventID ID, eventType RuntimeStreamEventType, severity RuntimeEventSeverity) {
	session, ok := r.store.GetSession(sessionID)
	if !ok {
		return
	}
	event, ok := r.store.GetEvent(eventID)
	if !ok {
		return
	}
	r.notify(ctx, eventType, session, severity, event, withEventID(event.ID), withAgentID(event.TargetAgentID))
}

func (r *Runtime) notifyTokenDecision(ctx context.Context, session Session, decision TokenBudgetDecision) {
	eventType := RuntimeStreamTokenUpdated
	severity := RuntimeSeverityWarn
	if !decision.Allowed || decision.CurrentUsage.IsExceeded {
		eventType = RuntimeStreamTokenLimitReached
		severity = RuntimeSeverityError
	}
	r.notify(ctx, eventType, session, severity, decision)
}

func (r *Runtime) notifyRuntimeError(ctx context.Context, session Session, eventID ID, code string, message string, err error, retryable bool) {
	if r.notifier == nil {
		return
	}
	payload := RuntimeErrorPayload{Code: code, Message: message, Retryable: retryable}
	if err != nil {
		payload.Reason = err.Error()
	}
	streamEvent := RuntimeStreamEvent{ID: nextID("stream"), Type: RuntimeStreamRuntimeError, SessionID: session.ID, RoomID: session.RoomID, EventID: eventID, Tick: session.Tick, Phase: session.CurrentPhase, Severity: RuntimeSeverityError, Error: &payload, CreatedAt: time.Now()}
	safeNotifyRuntimeNotifier(ctx, r.notifier, streamEvent)
}

func (r *Runtime) logInfo(message string, fields map[string]interface{}) {
	logger.Info(message, fields)
}

func (r *Runtime) logWarn(message string, fields map[string]interface{}) {
	logger.Warn(message, fields)
}

func (r *Runtime) String() string {
	return fmt.Sprintf("creative.Runtime{agents:%d}", len(r.registry.List()))
}
