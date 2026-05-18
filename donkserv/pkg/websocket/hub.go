package websocket

import (
	"sync"
)

// Hub 是 WebSocket 连接的中心管理器
// 负责维护所有活跃客户端连接，并处理消息的路由分发
type Hub struct {
	// clients 存储所有已连接的客户端，key为客户端指针
	clients map[*Client]bool

	// register 是客户端注册的通道
	// 当新客户端连接时，通过此通道通知 Hub
	register chan *Client

	// unregister 是客户端注销的通道
	// 当客户端断开连接时，通过此通道通知 Hub 进行清理
	unregister chan *Client

	// broadcast 是广播消息的通道
	// 所有需要发送给所有客户端的消息都通过此通道
	broadcast chan []byte

	// exit 是退出信号通道
	// 当需要关闭 Hub 时，发送信号到此通道
	exit chan struct{}

	// router 是消息路由器
	// 负责将消息按类型分发到对应的业务处理器
	router *MessageRouter

	// mu 用于保护 clients map 的并发访问
	mu sync.RWMutex
}

// NewHub 创建并初始化一个新的 Hub 实例
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		exit:       make(chan struct{}),
		router:     NewMessageRouter(),
	}
}

// Run 是 Hub 的主循环，运行在一个独立的 goroutine 中
// 通过 select 语句监听：注册、注销、广播、退出
func (h *Hub) Run() {
	for {
		select {
		// 处理新客户端注册
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		// 处理客户端注销
		case client := <-h.unregister:
			h.mu.Lock()
			// 检查客户端是否仍在列表中（防止重复注销）
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client) // 从管理列表中移除
				close(client.send)        // 关闭客户端的发送通道
			}
			h.mu.Unlock()

		// 处理广播消息
		case message := <-h.broadcast:
			h.mu.RLock()
			// 遍历所有客户端，发送消息
			for client := range h.clients {
				select {
				case client.send <- message:
					// 消息发送成功
				default:
					// 发送缓冲区已满，说明客户端处理速度过慢
					// 释放读锁，转换为写锁进行清理
					h.mu.RUnlock()
					h.mu.Lock()
					close(client.send)        // 关闭慢客户端的发送通道
					delete(h.clients, client) // 从管理列表中移除
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()

		// 处理退出信号
		case <-h.exit:
			// 收到退出信号，关闭所有客户端连接
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// Register 向 Hub 注册一个新的客户端
// 该方法是非阻塞的，将客户端放入注册通道
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 从 Hub 注销一个客户端
// 该方法是非阻塞的，将客户端放入注销通道
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast 向所有已连接的客户端广播消息
// 消息会被放入广播通道，由 Run 方法负责发送给每个客户端
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// ClientCount 返回当前已连接的客户端数量
// 该方法使用读锁，适合并发调用
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Shutdown 优雅关闭 Hub
// 会关闭所有客户端连接并停止 Hub 的主循环
func (h *Hub) Shutdown() {
	close(h.exit)
}

// GetRouter 返回消息路由器
// 业务模块可以通过此方法注册自己的消息处理器
func (h *Hub) GetRouter() *MessageRouter {
	return h.router
}

// RouteMessage 将消息路由到对应的业务处理器
// 由 Client 的 ReadPump 调用，消息会被分发到注册了该类型的处理器
func (h *Hub) RouteMessage(msg *Message) {
	h.router.Route(msg)
}
