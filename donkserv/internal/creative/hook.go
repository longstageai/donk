package creative

import (
	"encoding/json"
	"time"

	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

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
	BeforeOutput(req *AgentRunRequest, output *AgentOutput) error
	AfterOutput(result *AgentRun) error
}

// NoopHookPipeline 是默认空 Hook 实现，保留扩展点但不改变数据。
type NoopHookPipeline struct{}

func (NoopHookPipeline) BeforeInput(req *AgentRunRequest) error                       { return nil }
func (NoopHookPipeline) AfterInput(input *AgentInput) error                           { return nil }
func (NoopHookPipeline) BeforeOutput(req *AgentRunRequest, output *AgentOutput) error { return nil }
func (NoopHookPipeline) AfterOutput(result *AgentRun) error                           { return nil }

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
func (p *CompositeHookPipeline) BeforeOutput(req *AgentRunRequest, output *AgentOutput) error {
	for _, hook := range p.hooks {
		if err := hook.BeforeOutput(req, output); err != nil {
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

type WebSocketAgentMessage struct {
	Type      string         `json:"type"`       // 消息类型
	Event     string         `json:"event"`      // 事件类型
	SessionID ID             `json:"session_id"` // 会话ID
	RoomID    ID             `json:"room_id"`    // 房间ID
	EventID   ID             `json:"event_id"`   // 事件ID
	AgentID   ID             `json:"agent_id"`   // Agent ID
	RunID     ID             `json:"run_id"`     // 运行记录ID
	Status    AgentRunStatus `json:"status"`     // 运行状态
	Role      MessageRole    `json:"role"`       // 消息角色
	Content   string         `json:"content"`    // 消息内容
	Timestamp int64          `json:"timestamp"`  // 时间戳
}

type WebSocketHook struct {
	hub *websocket.Hub
}

func NewWebSocketHook(hub *websocket.Hub) *WebSocketHook {
	return &WebSocketHook{hub: hub}
}

func (h *WebSocketHook) BeforeInput(req *AgentRunRequest) error { return nil }
func (h *WebSocketHook) AfterInput(input *AgentInput) error     { return nil }

func (h *WebSocketHook) BeforeOutput(req *AgentRunRequest, output *AgentOutput) error {
	if h == nil || h.hub == nil || req == nil || output == nil {
		return nil
	}
	for _, draft := range output.Messages {
		if draft.Content == "" {
			continue
		}
		message := WebSocketAgentMessage{Type: "stream", Event: "content_delta", SessionID: req.Session.ID, RoomID: req.Room.ID, EventID: req.Event.ID, AgentID: req.Agent.ID(), RunID: req.RunID, Status: output.Status, Role: draft.Role, Content: draft.Content, Timestamp: time.Now().Unix()}
		data, err := json.Marshal(message)
		if err != nil {
			logger.Error("creative agent 输出消息序列化失败", map[string]interface{}{"error": err.Error()})
			continue
		}
		h.hub.BroadcastJson(data)
	}
	return nil
}

func (h *WebSocketHook) AfterOutput(result *AgentRun) error { return nil }

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
