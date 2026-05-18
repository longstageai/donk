package websocket

import (
	"sync"
	"time"
)

// MessageType 定义了消息类型的枚举
// 用于区分不同业务类型的消息，实现按类型分发
type MessageType string

// 支持的消息类型常量
const (
	TypeChat         MessageType = "chat"         // 聊天消息
	TypeNotification MessageType = "notification" // 通知消息
	TypeCommand      MessageType = "command"      // 命令消息
	TypePing         MessageType = "ping"         // 心跳检测请求
	TypePong         MessageType = "pong"         // 心跳检测响应
	TypeClose        MessageType = "close"        // 关闭连接消息
)

// Message 是 WebSocket 消息的统一结构
// 包含消息内容、来源客户端、时间戳等信息
type Message struct {
	Type      MessageType `json:"type"`      // 消息类型
	Content   []byte      `json:"content"`   // 消息内容（原始字节）
	Client    *Client     `json:"-"`         // 来源客户端（不序列化，用于回复）
	Timestamp int64       `json:"timestamp"` // 时间戳（Unix时间戳）
}

// NewMessage 创建一个新的消息
func NewMessage(msgType MessageType, content []byte, client *Client) *Message {
	return &Message{
		Type:      msgType,
		Content:   content,
		Client:    client,
		Timestamp: time.Now().Unix(),
	}
}

// BusinessHandler 是业务处理器的接口
// 业务模块实现此接口来处理特定类型的消息
type BusinessHandler interface {
	// GetMessageType 返回该处理器感兴趣的消息类型
	GetMessageType() MessageType
	// Handle 处理接收到的消息
	Handle(msg *Message)
}

// HandlerFunc 是业务处理函数的类型
// 实现了 BusinessHandler 接口，方便使用函数作为处理器
type HandlerFunc func(msg *Message)

func (f HandlerFunc) GetMessageType() MessageType {
	return TypeChat
}

func (f HandlerFunc) Handle(msg *Message) {
	f(msg)
}

// MessageRouter 消息路由器
// 管理多个业务处理器，按消息类型分发消息到对应的处理器
type MessageRouter struct {
	handlers map[MessageType][]BusinessHandler // 按消息类型存储处理器列表
	mu       sync.RWMutex                      // 保护 handlers map
}

// NewMessageRouter 创建一个新的消息路由器
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers: make(map[MessageType][]BusinessHandler),
	}
}

// Register 注册一个业务处理器
// 同一类型的消息可以注册多个处理器，都会收到该类型的消息
func (mr *MessageRouter) Register(handler BusinessHandler) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	msgType := handler.GetMessageType()
	mr.handlers[msgType] = append(mr.handlers[msgType], handler)
}

// Unregister 注销指定类型的处理器
func (mr *MessageRouter) Unregister(msgType MessageType) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	delete(mr.handlers, msgType)
}

// Route 将消息路由到对应的处理器
// 消息会被传递给所有注册了该类型消息的处理器
func (mr *MessageRouter) Route(msg *Message) {
	mr.mu.RLock()
	handlers, ok := mr.handlers[msg.Type]
	mr.mu.RUnlock()

	if !ok || len(handlers) == 0 {
		return
	}

	for _, handler := range handlers {
		go handler.Handle(msg)
	}
}
