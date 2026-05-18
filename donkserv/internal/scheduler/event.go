package scheduler

import (
	"bytes"
	"encoding/json"
	"github.com/longstageai/donk/donk/internal/websocket"
	"net/http"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// EventType 事件类型定义
type EventType string

const (
	EventTaskCreated   EventType = "task:created"   // 任务创建
	EventTaskStarted   EventType = "task:started"   // 任务开始执行
	EventTaskCompleted EventType = "task:completed" // 任务执行完成
	EventTaskFailed    EventType = "task:failed"    // 任务执行失败
	EventTaskCancelled EventType = "task:cancelled" // 任务被取消
)

// TaskEvent 任务事件结构
// 用于在系统内部传递任务状态变化
type TaskEvent struct {
	Type      EventType // 事件类型
	TaskID    string    // 任务ID
	Task      *Task     // 任务详情（可能为nil）
	Timestamp int64     // 事件时间戳
}

// Subscriber 事件订阅者接口
// 所有事件处理器需要实现此接口
type Subscriber interface {
	// OnEvent 处理事件
	OnEvent(event *TaskEvent)
}

// EventBus 事件总线
// 提供事件的发布和订阅功能，支持多种订阅者
type EventBus struct {
	subscribers []Subscriber
	mu          sync.RWMutex
}

// NewEventBus 创建事件总线实例
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make([]Subscriber, 0),
	}
}

// Subscribe 订阅事件
// 添加新的订阅者到事件总线
func (eb *EventBus) Subscribe(sub Subscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers = append(eb.subscribers, sub)
}

// Unsubscribe 取消订阅
// 从事件总线中移除指定的订阅者
func (eb *EventBus) Unsubscribe(sub Subscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for i, s := range eb.subscribers {
		if s == sub {
			eb.subscribers = append(eb.subscribers[:i], eb.subscribers[i+1:]...)
			return
		}
	}
}

// Publish 发布事件
// 将事件分发给所有订阅者，异步执行不影响主流程
func (eb *EventBus) Publish(event *TaskEvent) {
	eb.mu.RLock()
	subscribers := make([]Subscriber, len(eb.subscribers))
	copy(subscribers, eb.subscribers)
	eb.mu.RUnlock()

	for _, sub := range subscribers {
		go func(s Subscriber) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("事件处理 panic", map[string]interface{}{"panic": r})
				}
			}()
			s.OnEvent(event)
		}(sub)
	}
}

// LogSubscriber 日志订阅者
// 将事件记录到日志
type LogSubscriber struct{}

// NewLogSubscriber 创建日志订阅者
func NewLogSubscriber() *LogSubscriber {
	return &LogSubscriber{}
}

// OnEvent 实现 Subscriber 接口
func (s *LogSubscriber) OnEvent(event *TaskEvent) {
	logger.Info("事件已发布", map[string]interface{}{
		"event_type": event.Type,
		"task_id":    event.TaskID,
		"status":     getTaskStatus(event.Task),
	})
}

func getTaskStatus(task *Task) string {
	if task == nil {
		return "unknown"
	}
	return string(task.Status)
}

// Ensure LogSubscriber 实现 Subscriber 接口
var _ Subscriber = (*LogSubscriber)(nil)

// WebhookSubscriber Webhook 订阅者
// 将事件推送到指定的 HTTP URL
type WebhookSubscriber struct {
	URL     string
	client  *http.Client
	retries int
}

// NewWebhookSubscriber 创建 Webhook 订阅者
// url: Webhook 目标地址
func NewWebhookSubscriber(url string) *WebhookSubscriber {
	return &WebhookSubscriber{
		URL: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		retries: 3,
	}
}

// OnEvent 实现 Subscriber 接口
func (s *WebhookSubscriber) OnEvent(event *TaskEvent) {
	// 只推送完成和失败事件
	if event.Type != EventTaskCompleted && event.Type != EventTaskFailed {
		return
	}

	// 序列化为 JSON
	data, err := json.Marshal(event)
	if err != nil {
		logger.Error("序列化事件失败", map[string]interface{}{"error": err.Error()})
		return
	}

	// 发送 POST 请求
	for i := 0; i < s.retries; i++ {
		resp, err := s.client.Post(s.URL, "application/json", bytes.NewReader(data))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 300 {
				logger.Info("Webhook 推送成功", map[string]interface{}{"url": s.URL})
				return
			}
		}
		// 等待后重试
		if i < s.retries-1 {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	logger.Warn("Webhook 推送失败", map[string]interface{}{"url": s.URL})
}

// Ensure WebhookSubscriber 实现 Subscriber 接口
var _ Subscriber = (*WebhookSubscriber)(nil)

// TaskEventHandler 任务事件处理器
// 用于处理特定任务的事件
type TaskEventHandler struct {
	TaskID  string
	Handler func(*TaskEvent)
}

// OnEvent 实现 Subscriber 接口
func (h *TaskEventHandler) OnEvent(event *TaskEvent) {
	if event.TaskID == h.TaskID && h.Handler != nil {
		h.Handler(event)
	}
}

// Ensure TaskEventHandler 实现 Subscriber 接口
var _ Subscriber = (*TaskEventHandler)(nil)

// EventBusOption 事件总线配置选项
type EventBusOption func(*EventBus)

// WithLogSubscriber 添加日志订阅者
func WithLogSubscriber() EventBusOption {
	return func(eb *EventBus) {
		eb.Subscribe(NewLogSubscriber())
	}
}

// WithWebhookSubscriber 添加 Webhook 订阅者
func WithWebhookSubscriber(url string) EventBusOption {
	return func(eb *EventBus) {
		eb.Subscribe(NewWebhookSubscriber(url))
	}
}

// WithWebSocketSubscriber 添加 WebSocket 订阅者
func WithWebSocketSubscriber(hub *websocket.Hub) EventBusOption {
	return func(eb *EventBus) {
		eb.Subscribe(NewWebSocketSubscriber(hub))
	}
}

// NewEventBusWithOptions 创建配置好的事件总线
func NewEventBusWithOptions(opts ...EventBusOption) *EventBus {
	eb := NewEventBus()
	for _, opt := range opts {
		opt(eb)
	}
	return eb
}

// WebSocketSubscriber WebSocket 事件订阅者
// 实现 Subscriber 接口，用于将任务事件通过 WebSocket 推送给客户端
type WebSocketSubscriber struct {
	hub *websocket.Hub // WebSocket Hub 管理器
}

// NewWebSocketSubscriber 创建 WebSocket 订阅者实例
// hub: WebSocket Hub 管理器，用于广播消息
func NewWebSocketSubscriber(hub *websocket.Hub) *WebSocketSubscriber {
	return &WebSocketSubscriber{hub: hub}
}

// OnEvent 处理任务事件
// 实现 Subscriber 接口，当任务状态发生变化时，将事件广播给所有 WebSocket 客户端
// event: 任务事件结构体
func (s *WebSocketSubscriber) OnEvent(event *TaskEvent) {
	// 根据事件类型转换为 action 字符串
	var action string
	switch event.Type {
	case EventTaskCreated:
		action = "created"
	case EventTaskStarted:
		action = "started"
	case EventTaskCompleted:
		action = "completed"
	case EventTaskFailed:
		action = "failed"
	case EventTaskCancelled:
		action = "cancelled"
	default:
		action = string(event.Type)
	}

	// 提取任务执行结果
	var result interface{}
	if event.Task != nil && event.Task.Result != nil {
		result = map[string]interface{}{
			"output":   event.Task.Result.Output,
			"error":    event.Task.Result.Error,
			"exitCode": event.Task.Result.ExitCode,
			"duration": event.Task.Result.Duration,
		}
	}

	// 创建 WebSocket 消息并广播
	msg := websocket.NewTaskEventMessage(
		action,
		event.Task.ID,
		event.Task.Name,
		string(event.Task.Status),
		result,
		"",
	)

	if err := s.hub.Broadcast(msg); err != nil {
		logger.Error("WebSocket 消息广播失败", map[string]interface{}{
			"taskID": event.Task.ID,
			"action": action,
			"error":  err.Error(),
		})
	}
}
