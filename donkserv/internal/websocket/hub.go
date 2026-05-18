package websocket

import (
	"fmt"
	"sync"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// Hub 是 WebSocket 连接的中心管理器
// 负责维护所有活跃客户端连接，并处理消息的广播分发
type Hub struct {
	clients    map[*Client]bool // 已连接的客户端集合
	register   chan *Client     // 客户端注册通道
	unregister chan *Client     // 客户端注销通道
	broadcast  chan []byte      // 广播消息通道
	exit       chan struct{}    // 退出信号通道
	mu         sync.RWMutex     // 保护 clients map 的读写锁
}

// NewHub 创建并初始化一个新的 Hub 实例
// 返回初始化好的 Hub 管理器
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		exit:       make(chan struct{}),
	}
}

// Run 是 Hub 的主事件循环，运行在一个独立的 goroutine 中
// 通过 select 语句监听：注册、注销、广播、退出
func (h *Hub) Run() {
	logger.Info("WebSocket Hub 已启动", map[string]interface{}{
		"addr": "ws://0.0.0.0:8081/ws/tasks",
	})

	for {
		select {
		// 处理新客户端注册
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			clientPtr := fmt.Sprintf("%p", client)
			h.mu.Unlock()
			logger.Info("客户端已连接", map[string]interface{}{
				"clientCount": clientCount,
				"clientPtr":   clientPtr,
			})

		// 处理客户端注销
		case client := <-h.unregister:
			h.mu.Lock()
			clientPtr := fmt.Sprintf("%p", client)
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			clientCount := len(h.clients)
			h.mu.Unlock()
			logger.Info("客户端已断开", map[string]interface{}{
				"clientCount": clientCount,
				"clientPtr":   clientPtr,
			})

		// 处理广播消息
		case message := <-h.broadcast:
			// 收集需要清理的慢客户端
			var slowClients []*Client
			sentCount := 0
			h.mu.RLock()
			clientCount := len(h.clients)
			for client := range h.clients {
				select {
				case client.send <- message:
					sentCount++
				default:
					// 发送缓冲区已满，标记为慢客户端
					slowClients = append(slowClients, client)
				}
			}
			h.mu.RUnlock()

			logger.Info("广播消息", map[string]interface{}{
				"clientCount": clientCount,
				"sentCount":   sentCount,
				"slowCount":   len(slowClients),
			})

			// 清理慢客户端（在锁外处理）
			if len(slowClients) > 0 {
				h.mu.Lock()
				for _, client := range slowClients {
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						close(client.send)
					}
				}
				clientCount := len(h.clients)
				h.mu.Unlock()
				logger.Warn("客户端消息缓冲区满，已断开", map[string]interface{}{
					"clientCount": clientCount,
					"slowCount":   len(slowClients),
				})
			}

		// 处理退出信号
		case <-h.exit:
			logger.Info("WebSocket Hub 正在关闭", nil)
			return
		}
	}
}

// Register 注册一个新的客户端连接到 Hub
// client: 要注册的客户端实例
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 从 Hub 中注销一个客户端连接
// client: 要注销的客户端实例
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast 向所有已连接的客户端广播消息
// msg: 要广播的消息结构体
// 返回: 序列化失败时返回错误
func (h *Hub) Broadcast(msg *Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		logger.Error("消息序列化失败", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	h.broadcast <- data
	return nil
}

// BroadcastJSON 直接广播原始 JSON 数据（非阻塞）
// data: JSON 格式的字节数据
func (h *Hub) BroadcastJSON(data []byte) {
	select {
	case h.broadcast <- data:
		// 成功发送
	default:
		// 通道已满，丢弃消息
		logger.Warn("广播通道已满，消息被丢弃", map[string]interface{}{
			"channelSize": len(h.broadcast),
		})
	}
}

// ClientCount 返回当前已连接的客户端数量
// 返回: 当前客户端数量
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Shutdown 优雅关闭 Hub
// 关闭退出通道，通知 Run 方法退出
func (h *Hub) Shutdown() {
	logger.Info("正在关闭 WebSocket Hub...", nil)
	close(h.exit)
}
