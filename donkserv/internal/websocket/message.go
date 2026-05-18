package websocket

import (
	"encoding/json"
	"time"
)

// MessageType 消息类型定义
// 用于区分不同业务类型的消息
type MessageType string

// 支持的消息类型常量
const (
	TypeTaskEvent MessageType = "task.event" // 任务事件消息
	TypeSystem    MessageType = "system"     // 系统通知消息
	TypePing      MessageType = "ping"       // 心跳检测请求
	TypePong      MessageType = "pong"       // 心跳检测响应
	TypeError     MessageType = "error"      // 错误消息
)

// Message WebSocket 消息的统一结构
// 包含消息类型、业务数据、时间戳等信息
type Message struct {
	Type      MessageType `json:"type"`                // 消息类型
	Action    string      `json:"action,omitempty"`    // 业务动作，如 created/completed/failed
	Data      interface{} `json:"data,omitempty"`      // 业务数据
	Timestamp int64       `json:"timestamp"`           // 时间戳（Unix 时间戳）
	TaskID    string      `json:"task_id,omitempty"`   // 关联任务 ID
	TaskName  string      `json:"task_name,omitempty"` // 关联任务名称
	Status    string      `json:"status,omitempty"`    // 状态
	Error     string      `json:"error,omitempty"`     // 错误信息
}

// NewMessage 创建指定类型的消息
func NewMessage(msgType MessageType) *Message {
	return &Message{
		Type:      msgType,
		Timestamp: time.Now().Unix(),
	}
}

// ToJSON 将消息序列化为 JSON 字节数组
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage 从 JSON 数据解析消息
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// TaskEventMessage 任务事件消息结构
// 用于承载任务状态变化时的详细信息
type TaskEventMessage struct {
	TaskID    string      `json:"task_id"`          // 任务 ID
	TaskName  string      `json:"task_name"`        // 任务名称
	Status    string      `json:"status"`           // 任务状态
	Result    interface{} `json:"result,omitempty"` // 执行结果
	Error     string      `json:"error,omitempty"`  // 错误信息
	Timestamp int64       `json:"timestamp"`        // 时间戳
}

// NewTaskEventMessage 创建任务事件消息
// eventType: 事件类型，如 created/completed/failed
// taskID: 任务 ID
// taskName: 任务名称
// status: 任务状态
// result: 执行结果
// errMsg: 错误信息
func NewTaskEventMessage(eventType, taskID, taskName, status string, result interface{}, errMsg string) *Message {
	return &Message{
		Type:      TypeTaskEvent,
		Action:    eventType,
		TaskID:    taskID,
		TaskName:  taskName,
		Status:    status,
		Data:      result,
		Error:     errMsg,
		Timestamp: time.Now().Unix(),
	}
}
