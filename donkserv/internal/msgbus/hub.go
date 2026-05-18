package msgbus

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// Client WebSocket 客户端连接实例
// 负责与单个客户端的 WebSocket 通信，包括消息读写和连接管理
type Client struct {
	ID       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	mu       sync.RWMutex
	topics   map[string]bool
	isClosed bool
}

// newClient 创建新的客户端实例
func newClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:     id,
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		topics: make(map[string]bool),
	}
}

// AddTopic 添加订阅主题
func (c *Client) AddTopic(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topics[topic] = true
}

// RemoveTopic 移除订阅主题
func (c *Client) RemoveTopic(topic string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.topics, topic)
}

// Topics 获取该客户端订阅的所有主题
func (c *Client) Topics() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	topics := make([]string, 0, len(c.topics))
	for topic := range c.topics {
		topics = append(topics, topic)
	}
	return topics
}

// IsSubscribed 检查客户端是否订阅了指定主题
func (c *Client) IsSubscribed(topic string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.topics[topic]
}

// Send 发送消息到客户端
func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		logger.Warn("客户端发送缓冲区已满", map[string]interface{}{
			"clientID": c.ID,
		})
		return false
	}
}

// SendBlocking 阻塞发送消息到客户端
func (c *Client) SendBlocking(data []byte) bool {
	select {
	case c.send <- data:
		return true
	case <-c.hub.done:
		return false
	}
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return
	}
	c.isClosed = true

	c.topics = nil
	close(c.send)
	c.conn.Close()

	logger.Debug("客户端已关闭", map[string]interface{}{
		"clientID": c.ID,
	})
}

// Hub WebSocket 连接的中心管理器
type Hub struct {
	clients              map[string]*Client
	register             chan *Client
	unregister           chan *Client
	broadcast            chan []byte
	topicBroadcast       map[string]chan []byte
	done                 chan struct{}
	mu                   sync.RWMutex
	onClientConnected    func(clientID string)
	onClientDisconnected func(clientID string)
}

// NewHub 创建并初始化一个新的 Hub 实例
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[string]*Client),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan []byte, 256),
		topicBroadcast: make(map[string]chan []byte),
		done:           make(chan struct{}),
	}
}

// Run 是 Hub 的主事件循环
func (h *Hub) Run() {
	logger.Debug("WebSocket Hub 已启动", nil)

	for {
		select {
		case client := <-h.register:
			logger.Debug("Hub 收到注册信号", map[string]interface{}{
				"clientID": client.ID,
			})
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			logger.Info("客户端已连接", map[string]interface{}{
				"clientID":    client.ID,
				"clientCount": h.ClientCount(),
			})
			if h.onClientConnected != nil {
				h.onClientConnected(client.ID)
			}

		case client := <-h.unregister:
			logger.Debug("Hub 收到注销信号", map[string]interface{}{
				"clientID": client.ID,
			})
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				logger.Debug("准备调用 client.Close()", map[string]interface{}{
					"clientID": client.ID,
				})
				h.mu.Unlock()

				// 在锁外调用 Close，避免死锁
				client.Close()

				logger.Debug("client.Close() 完成", map[string]interface{}{
					"clientID": client.ID,
				})

				if h.onClientDisconnected != nil {
					h.onClientDisconnected(client.ID)
				}
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				client.Send(message)
			}
			h.mu.RUnlock()

		case <-h.done:
			logger.Info("WebSocket Hub 正在关闭", nil)
			return

		case <-time.After(30 * time.Second):
			logger.Debug("Hub 心跳检查", map[string]interface{}{
				"clientCount": h.ClientCount(),
			})
		}
	}
}

// Register 注册一个新的客户端连接到 Hub
// 添加超时保护，避免永久阻塞
func (h *Hub) Register(client *Client) error {
	select {
	case h.register <- client:
		return nil
	case <-time.After(5 * time.Second):
		return ErrRegisterTimeout
	case <-h.done:
		return ErrHubClosed
	}
}

// Unregister 从 Hub 中注销一个客户端连接
func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-time.After(5 * time.Second):
		logger.Warn("Unregister 超时", map[string]interface{}{
			"clientID": client.ID,
		})
	}
}

// Broadcast 向所有已连接的客户端广播消息
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

// ClientCount 返回当前已连接的客户端数量
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClient 获取指定 ID 的客户端
func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.clients[clientID]
	return client, ok
}

// SendToTopic 向订阅了指定主题的客户端发送消息
func (h *Hub) SendToTopic(topic string, msg *Message) int {
	data, err := msg.ToJSON()
	if err != nil {
		logger.Error("消息序列化失败", map[string]interface{}{
			"error": err.Error(),
		})
		return 0
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, client := range h.clients {
		if client.IsSubscribed(topic) {
			if client.Send(data) {
				count++
			}
		}
	}
	return count
}
