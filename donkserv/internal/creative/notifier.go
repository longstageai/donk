package creative

import (
	"context"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// RuntimeStreamEventType 表示 Runtime 对外通知的运行时事件类型。
type RuntimeStreamEventType string

const (
	RuntimeStreamSessionStarted   RuntimeStreamEventType = "session_started"
	RuntimeStreamSessionCompleted RuntimeStreamEventType = "session_completed"
	RuntimeStreamSessionBlocked   RuntimeStreamEventType = "session_blocked"
	RuntimeStreamSessionCancelled RuntimeStreamEventType = "session_cancelled"

	RuntimeStreamLoopStarted   RuntimeStreamEventType = "loop_started"
	RuntimeStreamLoopPaused    RuntimeStreamEventType = "loop_paused"
	RuntimeStreamLoopResumed   RuntimeStreamEventType = "loop_resumed"
	RuntimeStreamLoopStopped   RuntimeStreamEventType = "loop_stopped"
	RuntimeStreamTickStarted   RuntimeStreamEventType = "tick_started"
	RuntimeStreamTickCompleted RuntimeStreamEventType = "tick_completed"

	RuntimeStreamEventCreated    RuntimeStreamEventType = "event_created"
	RuntimeStreamEventClaimed    RuntimeStreamEventType = "event_claimed"
	RuntimeStreamEventProcessing RuntimeStreamEventType = "event_processing"
	RuntimeStreamEventSucceeded  RuntimeStreamEventType = "event_succeeded"
	RuntimeStreamEventRejected   RuntimeStreamEventType = "event_rejected"
	RuntimeStreamEventFailed     RuntimeStreamEventType = "event_failed"
	RuntimeStreamEventSkipped    RuntimeStreamEventType = "event_skipped"

	RuntimeStreamAgentRunStarted   RuntimeStreamEventType = "agent_run_started"
	RuntimeStreamAgentInputBuilt   RuntimeStreamEventType = "agent_input_built"
	RuntimeStreamAgentOutputReady  RuntimeStreamEventType = "agent_output_ready"
	RuntimeStreamAgentRunCompleted RuntimeStreamEventType = "agent_run_completed"
	RuntimeStreamAgentRunFailed    RuntimeStreamEventType = "agent_run_failed"

	RuntimeStreamMessageCreated  RuntimeStreamEventType = "message_created"
	RuntimeStreamArtifactCreated RuntimeStreamEventType = "artifact_created"
	RuntimeStreamSnapshotCreated RuntimeStreamEventType = "snapshot_created"

	RuntimeStreamTokenUpdated      RuntimeStreamEventType = "token_updated"
	RuntimeStreamTokenLimitReached RuntimeStreamEventType = "token_limit_reached"

	RuntimeStreamControlReceived RuntimeStreamEventType = "control_received"
	RuntimeStreamRuntimeError    RuntimeStreamEventType = "runtime_error"
)

// RuntimeEventSeverity 表示运行时通知的严重级别。
type RuntimeEventSeverity string

const (
	RuntimeSeverityDebug RuntimeEventSeverity = "debug"
	RuntimeSeverityInfo  RuntimeEventSeverity = "info"
	RuntimeSeverityWarn  RuntimeEventSeverity = "warn"
	RuntimeSeverityError RuntimeEventSeverity = "error"
)

// RuntimeErrorPayload 是推送给客户端的标准化错误信息。
type RuntimeErrorPayload struct {
	Code      string `json:"code"`             // 错误代码
	Message   string `json:"message"`          // 错误消息
	Retryable bool   `json:"retryable"`        // 是否可重试
	Reason    string `json:"reason,omitempty"` // 错误原因（可选）
}

// RuntimeStreamEvent 是 Runtime 对外广播的统一运行时事件。
type RuntimeStreamEvent struct {
	ID        ID                     `json:"id"`                 // 事件唯一标识
	Type      RuntimeStreamEventType `json:"type"`               // 事件类型
	SessionID ID                     `json:"session_id"`         // 关联的Session ID
	RoomID    ID                     `json:"room_id,omitempty"`  // 关联的Room ID（可选）
	EventID   ID                     `json:"event_id,omitempty"` // 关联的Event ID（可选）
	AgentID   ID                     `json:"agent_id,omitempty"` // 关联的Agent ID（可选）
	RunID     ID                     `json:"run_id,omitempty"`   // 关联的Run ID（可选）
	Tick      int                    `json:"tick"`               // 当前Tick计数
	Phase     Phase                  `json:"phase,omitempty"`    // 当前阶段（可选）
	Severity  RuntimeEventSeverity   `json:"severity"`           // 严重级别
	Payload   any                    `json:"payload,omitempty"`  // 事件载荷（可选）
	Error     *RuntimeErrorPayload   `json:"error,omitempty"`    // 错误信息（可选）
	CreatedAt time.Time              `json:"created_at"`         // 创建时间
}

// AgentInputView 是 AgentInput 的外部展示视图，避免直接暴露完整上下文内容。
type AgentInputView struct {
	EventID       ID             `json:"event_id"`       // 事件ID
	EventType     EventType      `json:"event_type"`     // 事件类型
	AgentID       ID             `json:"agent_id"`       // Agent ID
	RunID         ID             `json:"run_id"`         // 运行记录ID
	SessionID     ID             `json:"session_id"`     // Session ID
	MessageCount  int            `json:"message_count"`  // 消息数量
	ArtifactIDs   []ID           `json:"artifact_ids"`   // 产物ID列表
	ArtifactTypes []ArtifactType `json:"artifact_types"` // 产物类型列表
	ToolNames     []string       `json:"tool_names"`     // 可用工具名称列表
	SnapshotID    ID             `json:"snapshot_id"`    // 快照ID
}

// AgentOutputView 是 AgentOutput 的外部展示视图，用于展示输出概要。
type AgentOutputView struct {
	Status        AgentRunStatus `json:"status"`         // 运行状态
	Decision      AgentDecision  `json:"decision"`       // 决策结果
	MessageCount  int            `json:"message_count"`  // 消息数量
	ArtifactTypes []ArtifactType `json:"artifact_types"` // 产物类型列表
	EventTypes    []EventType    `json:"event_types"`    // 事件类型列表
	TokenUsage    TokenUsage     `json:"token_usage"`    // Token使用统计
	HasError      bool           `json:"has_error"`      // 是否有错误
}

// RuntimeNotifier 定义 Runtime 运行时通知出口。
type RuntimeNotifier interface {
	Notify(ctx context.Context, event RuntimeStreamEvent)
}

// NoopRuntimeNotifier 是默认空通知器，不影响 Runtime 正常执行。
type NoopRuntimeNotifier struct{}

func (NoopRuntimeNotifier) Notify(ctx context.Context, event RuntimeStreamEvent) {}

// CompositeRuntimeNotifier 将多个通知器组合为一个通知器。
type CompositeRuntimeNotifier struct {
	notifiers []RuntimeNotifier // 通知器列表
}

// NewCompositeRuntimeNotifier 创建组合通知器，nil 通知器会被自动忽略。
func NewCompositeRuntimeNotifier(notifiers ...RuntimeNotifier) RuntimeNotifier {
	filtered := make([]RuntimeNotifier, 0, len(notifiers))
	for _, notifier := range notifiers {
		if notifier != nil {
			filtered = append(filtered, notifier)
		}
	}
	if len(filtered) == 0 {
		return NoopRuntimeNotifier{}
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return &CompositeRuntimeNotifier{notifiers: filtered}
}

// Notify 按注册顺序广播事件，单个通知器异常不会影响后续通知器和 Runtime 主流程。
func (n *CompositeRuntimeNotifier) Notify(ctx context.Context, event RuntimeStreamEvent) {
	for _, notifier := range n.notifiers {
		safeNotifyRuntimeNotifier(ctx, notifier, event)
	}
}

func safeNotifyRuntimeNotifier(ctx context.Context, notifier RuntimeNotifier, event RuntimeStreamEvent) {
	if notifier == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Warn("Runtime 通知器执行异常，已忽略", map[string]interface{}{"event_id": event.ID, "event_type": event.Type, "panic": recovered})
		}
	}()
	notifier.Notify(ctx, event)
}

func buildAgentInputView(input AgentInput, agentID ID, runID ID) AgentInputView {
	artifactIDs := make([]ID, 0, len(input.Artifacts))
	artifactTypes := make([]ArtifactType, 0, len(input.Artifacts))
	for _, artifact := range input.Artifacts {
		artifactIDs = append(artifactIDs, artifact.ID)
		artifactTypes = append(artifactTypes, artifact.Type)
	}
	toolNames := make([]string, 0, len(input.Tools))
	for _, tool := range input.Tools {
		toolNames = append(toolNames, tool.Name)
	}
	return AgentInputView{EventID: input.Event.ID, EventType: input.Event.Type, AgentID: agentID, RunID: runID, SessionID: input.Session.ID, MessageCount: len(input.Messages), ArtifactIDs: artifactIDs, ArtifactTypes: artifactTypes, ToolNames: toolNames, SnapshotID: input.Snapshot.ID}
}

func buildAgentOutputView(output AgentOutput) AgentOutputView {
	artifactTypes := make([]ArtifactType, 0, len(output.Artifacts))
	for _, artifact := range output.Artifacts {
		artifactTypes = append(artifactTypes, artifact.Type)
	}
	eventTypes := make([]EventType, 0, len(output.Events))
	for _, event := range output.Events {
		eventTypes = append(eventTypes, event.Type)
	}
	return AgentOutputView{Status: output.Status, Decision: output.Decision, MessageCount: len(output.Messages), ArtifactTypes: artifactTypes, EventTypes: eventTypes, TokenUsage: output.TokenUsage, HasError: output.Error != nil}
}
