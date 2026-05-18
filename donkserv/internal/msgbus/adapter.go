package msgbus

import (
	"errors"
	"sync"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// ErrClientNotFound 客户端未找到错误
var ErrClientNotFound = errors.New("client not found")

// ErrInvalidMessage 无效消息错误
var ErrInvalidMessage = errors.New("invalid message")

// MessageHandler 业务消息处理函数类型
// 当 WebSocket 客户端发送业务消息时调用
type MessageHandler func(clientID string, topic string, payload interface{}) error

// ConnectHandler 客户端连接处理函数类型
type ConnectHandler func(clientID string)

// DisconnectHandler 客户端断开处理函数类型
type DisconnectHandler func(clientID string)

// Adapter WebSocket 适配器
// 桥接消息总线（Bus）和 WebSocket 服务器（Server）
// 负责管理客户端订阅、消息路由和分发
type Adapter struct {
	// 消息总线
	bus *Bus

	// WebSocket 服务器
	server *Server

	// 客户端到订阅者的映射
	// 用于在客户端断开时自动取消订阅
	clientSubscriptions map[string][]*Subscriber

	// 同步保护
	mu sync.RWMutex

	// 业务消息处理函数
	onMessage MessageHandler

	// 客户端连接处理函数
	onConnect ConnectHandler

	// 客户端断开处理函数
	onDisconnect DisconnectHandler
}

// AdapterOption 函数选项模式，用于配置 Adapter
type AdapterOption func(*Adapter)

// WithMessageHandler 设置业务消息处理函数
// handler: 消息处理函数
// 返回: 配置函数
func WithMessageHandler(handler MessageHandler) AdapterOption {
	return func(a *Adapter) {
		a.onMessage = handler
	}
}

// WithConnectHandler 设置客户端连接处理函数
// handler: 连接处理函数
// 返回: 配置函数
func WithConnectHandler(handler ConnectHandler) AdapterOption {
	return func(a *Adapter) {
		a.onConnect = handler
	}
}

// WithDisconnectHandler 设置客户端断开处理函数
// handler: 断开处理函数
// 返回: 配置函数
func WithDisconnectHandler(handler DisconnectHandler) AdapterOption {
	return func(a *Adapter) {
		a.onDisconnect = handler
	}
}

// NewAdapter 创建并初始化 WebSocket 适配器
// bus: 消息总线实例
// server: WebSocket 服务器实例
// opts: 配置选项
// 返回: 适配器实例
func NewAdapter(bus *Bus, server *Server, opts ...AdapterOption) *Adapter {
	adapter := &Adapter{
		bus:                 bus,
		server:              server,
		clientSubscriptions: make(map[string][]*Subscriber),
	}

	// 应用选项
	for _, opt := range opts {
		opt(adapter)
	}

	// 设置 Hub 的回调
	server.hub.onClientConnected = adapter.handleClientConnected
	server.hub.onClientDisconnected = adapter.handleClientDisconnected

	// 设置服务器的消息处理回调
	server.onClientMessage = adapter.handleClientMessage
	server.onSubscribe = adapter.handleSubscribe
	server.onUnsubscribe = adapter.handleUnsubscribe
	server.onPublish = adapter.handlePublish

	logger.Debug("WebSocket 适配器已创建", nil)

	return adapter
}

// handleClientConnected 处理客户端连接事件
func (a *Adapter) handleClientConnected(clientID string) {
	logger.Debug("WebSocket 适配器：客户端已连接", map[string]interface{}{
		"clientID": clientID,
	})

	if a.onConnect != nil {
		a.onConnect(clientID)
	}
}

// handleClientDisconnected 处理客户端断开事件
// 自动清理该客户端的所有订阅
func (a *Adapter) handleClientDisconnected(clientID string) {
	logger.Debug("WebSocket 适配器：准备清理客户端订阅", map[string]interface{}{
		"clientID": clientID,
	})

	// 清理订阅
	a.mu.Lock()
	subs := a.clientSubscriptions[clientID]
	delete(a.clientSubscriptions, clientID)
	a.mu.Unlock()

	// 取消所有订阅
	for _, sub := range subs {
		a.bus.Unsubscribe(sub)
	}

	if a.onDisconnect != nil {
		a.onDisconnect(clientID)
	}
}

// handleClientMessage 处理客户端消息
// 当客户端发送消息时，触发业务消息处理
func (a *Adapter) handleClientMessage(clientID string, msg *ClientMessage) {
	if a.onMessage != nil {
		a.onMessage(clientID, msg.Topic, msg.Payload)
	}
}

func (a *Adapter) handleSubscribe(clientID, topic string) {
	if err := a.Subscribe(clientID, topic); err != nil {
		logger.Error("订阅失败", map[string]interface{}{
			"clientID": clientID,
			"topic":    topic,
			"error":    err.Error(),
		})
	}
}

func (a *Adapter) handleUnsubscribe(clientID, topic string) {
	if err := a.Unsubscribe(clientID, topic); err != nil {
		logger.Error("取消订阅失败", map[string]interface{}{
			"clientID": clientID,
			"topic":    topic,
			"error":    err.Error(),
		})
	}
}

func (a *Adapter) handlePublish(clientID, topic string, payload interface{}) {
	if err := a.bus.Publish(topic, payload, clientID); err != nil {
		logger.Error("发布消息失败", map[string]interface{}{
			"clientID": clientID,
			"topic":    topic,
			"error":    err.Error(),
		})
	}
}

// Subscribe 客户端订阅主题
// clientID: 客户端ID
// topic: 主题名称
// 返回: 错误信息
func (a *Adapter) Subscribe(clientID string, topic string) error {
	// 获取客户端
	client, ok := a.server.hub.GetClient(clientID)
	if !ok {
		return ErrClientNotFound
	}

	// 检查是否已经订阅过该主题
	a.mu.RLock()
	for _, sub := range a.clientSubscriptions[clientID] {
		for _, t := range sub.Topics {
			if t == topic {
				a.mu.RUnlock()
				logger.Debug("客户端已订阅过该主题，跳过", map[string]interface{}{
					"clientID": clientID,
					"topic":    topic,
				})
				return nil
			}
		}
	}
	a.mu.RUnlock()

	// 添加主题到客户端
	client.AddTopic(topic)

	// 创建消息处理器
	handler := func(msg *Message) error {
		// 如果是当前客户端自己发布的消息，不发送给自己，避免消息循环
		if msg.Sender == clientID {
			return nil
		}
		// 通过 Hub 发送到特定主题的订阅者
		a.server.hub.SendToTopic(topic, msg)
		return nil
	}

	// 订阅到 Bus
	sub, err := a.bus.Subscribe(topic, handler)
	if err != nil {
		return err
	}

	// 记录订阅关系
	a.mu.Lock()
	a.clientSubscriptions[clientID] = append(a.clientSubscriptions[clientID], sub)
	a.mu.Unlock()

	logger.Debug("客户端已订阅主题", map[string]interface{}{
		"clientID": clientID,
		"topic":    topic,
	})

	return nil
}

// Unsubscribe 客户端取消订阅主题
// clientID: 客户端ID
// topic: 主题名称
// 返回: 错误信息
func (a *Adapter) Unsubscribe(clientID string, topic string) error {
	// 获取客户端
	client, ok := a.server.hub.GetClient(clientID)
	if !ok {
		return ErrClientNotFound
	}

	// 移除主题
	client.RemoveTopic(topic)

	logger.Debug("客户端已取消订阅主题", map[string]interface{}{
		"clientID": clientID,
		"topic":    topic,
	})

	return nil
}

// Publish 发布消息
// topic: 主题名称
// payload: 消息负载
// sender: 发送者标识
// 返回: 错误信息
func (a *Adapter) Publish(topic string, payload interface{}, sender string) error {
	return a.bus.Publish(topic, payload, sender)
}

// SendToClient 向指定客户端发送消息
// clientID: 客户端ID
// msg: 消息实例
// 返回: 是否发送成功
func (a *Adapter) SendToClient(clientID string, msg *Message) bool {
	client, ok := a.server.hub.GetClient(clientID)
	if !ok {
		return false
	}
	data, err := msg.ToJSON()
	if err != nil {
		return false
	}
	return client.Send(data)
}

// BroadcastToTopic 向订阅指定主题的所有客户端广播消息
// topic: 主题名称
// msg: 消息实例
// 返回: 发送成功的客户端数量
func (a *Adapter) BroadcastToTopic(topic string, msg *Message) int {
	return a.server.hub.SendToTopic(topic, msg)
}

// GetClientSubscriptions 获取客户端的订阅列表
// clientID: 客户端ID
// 返回: 订阅的主题列表
func (a *Adapter) GetClientSubscriptions(clientID string) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	client, ok := a.server.hub.GetClient(clientID)
	if !ok {
		return nil
	}

	return client.Topics()
}

// ClientCount 获取当前连接的客户端数量
// 返回: 客户端数量
func (a *Adapter) ClientCount() int {
	return a.server.hub.ClientCount()
}

// Bus 返回关联的消息总线
// 返回: Bus 实例
func (a *Adapter) Bus() *Bus {
	return a.bus
}

// Server 返回关联的 WebSocket 服务器
// 返回: Server 实例
func (a *Adapter) Server() *Server {
	return a.server
}
