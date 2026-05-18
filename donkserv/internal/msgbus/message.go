package msgbus

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// 错误定义
var (
	ErrHubClosed       = errors.New("hub is closed")
	ErrRegisterTimeout = errors.New("register timeout")
)

// MessageType 消息类型定义
// 用于区分不同业务类型的消息
type MessageType string

// 支持的消息类型常量
const (
	TypeSubscribe   MessageType = "subscribe"   // 订阅主题
	TypeUnsubscribe MessageType = "unsubscribe" // 取消订阅
	TypePublish     MessageType = "publish"     // 发布消息
	TypePing        MessageType = "ping"        // 心跳检测请求
	TypePong        MessageType = "pong"        // 心跳检测响应
	TypeSystem      MessageType = "system"      // 系统通知
	TypeError       MessageType = "error"       // 错误消息
	TypeMessage     MessageType = "message"     // 业务消息
)

// Message WebSocket 消息的统一结构
// 包含消息类型、主题、负载、时间戳等信息
type Message struct {
	ID        string      `json:"id"`        // 消息唯一标识
	Type      MessageType `json:"type"`      // 消息类型
	Topic     string      `json:"topic"`     // 消息主题
	Payload   interface{} `json:"payload"`   // 消息负载
	Timestamp int64       `json:"timestamp"` // 时间戳（Unix 毫秒）
	Sender    string      `json:"sender"`    // 发送者标识
}

// NewMessage 创建指定类型的新消息
// msgType: 消息类型
// 返回: 初始化好的消息实例
func NewMessage(msgType MessageType) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewMessageWithTopic 创建带有主题的消息
// msgType: 消息类型
// topic: 消息主题
// 返回: 初始化好的消息实例
func NewMessageWithTopic(msgType MessageType, topic string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      msgType,
		Topic:     topic,
		Timestamp: time.Now().UnixMilli(),
	}
}

// SetPayload 设置消息负载
// payload: 消息负载数据
// 返回: 消息本身，支持链式调用
func (m *Message) SetPayload(payload interface{}) *Message {
	m.Payload = payload
	return m
}

// SetSender 设置消息发送者
// sender: 发送者标识
// 返回: 消息本身，支持链式调用
func (m *Message) SetSender(sender string) *Message {
	m.Sender = sender
	return m
}

// ToJSON 将消息序列化为 JSON 字节数组
// 返回: 序列化后的字节数组和错误信息
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// ParseMessage 从 JSON 数据解析消息
// data: JSON 格式的字节数据
// 返回: 解析后的消息实例和错误信息
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// ClientMessage 客户端发送给服务端的控制消息
// 用于订阅、取消订阅、发布等操作
type ClientMessage struct {
	Type    MessageType `json:"type"`    // 消息类型
	Topic   string      `json:"topic"`   // 主题
	Payload interface{} `json:"payload"` // 负载数据
}

// ParseClientMessage 解析客户端控制消息
// data: JSON 格式的字节数据
// 返回: 解析后的客户端消息和错误信息
func ParseClientMessage(data []byte) (*ClientMessage, error) {
	var msg ClientMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// SystemMessage 创建系统消息
// content: 系统消息内容
// 返回: 配置好的消息实例
func SystemMessage(content string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      TypeSystem,
		Payload:   content,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ErrorMessage 创建错误消息
// errMsg: 错误信息
// 返回: 配置好的错误消息实例
func ErrorMessage(errMsg string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      TypeError,
		Payload:   errMsg,
		Timestamp: time.Now().UnixMilli(),
	}
}

// PongMessage 创建心跳响应消息
// 返回: 心跳响应消息实例
func PongMessage() *Message {
	return &Message{
		ID:        uuid.New().String(),
		Type:      TypePong,
		Timestamp: time.Now().UnixMilli(),
	}
}

// PongJSON 创建心跳响应消息的 JSON 字节数组
func PongJSON() []byte {
	msg := PongMessage()
	data, _ := json.Marshal(msg)
	return data
}

// generateClientID 生成客户端唯一标识符
func generateClientID() string {
	return uuid.New().String()
}
