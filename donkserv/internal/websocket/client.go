package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

const (
	writeWait      = 10 * time.Second    // 写入超时时间
	pongWait       = 60 * time.Second    // 读取超时时间
	pingPeriod     = (pongWait * 9) / 10 // 心跳发送周期（54秒）
	maxMessageSize = 512 * 1024          // 最大消息大小（512KB）
)

// Client WebSocket 客户端连接实例
// 负责与单个客户端的 WebSocket 通信
type Client struct {
	hub      *Hub            // 所属的 Hub 管理器
	conn     *websocket.Conn // WebSocket 连接
	send     chan []byte     // 发送消息通道
	taskID   string          // 订阅的任务 ID（可选，空表示接收所有任务事件）
	done     chan struct{}   // 关闭信号，用于通知 WritePump 退出
	doneOnce sync.Once       // 确保 done 只关闭一次
}

// NewClient 创建新的客户端实例
// hub: 所属的 Hub 管理器
// conn: 已建立的 WebSocket 连接
// taskID: 订阅的任务 ID（可选）
func NewClient(hub *Hub, conn *websocket.Conn, taskID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		taskID: taskID,
		done:   make(chan struct{}),
	}
}

// closeDone 关闭 done 通道，通知 WritePump 退出
func (c *Client) closeDone() {
	c.doneOnce.Do(func() {
		close(c.done)
	})
}

// ReadPump 处理从客户端接收消息的循环
// 运行在独立的 goroutine 中，读取客户端发送的消息并处理
func (c *Client) ReadPump() {
	defer func() {
		c.closeDone() // 通知 WritePump 退出
		c.hub.Unregister(c)
		c.conn.Close()
		logger.Info("客户端 ReadPump 已退出", map[string]interface{}{})
	}()

	// 设置读取限制
	c.conn.SetReadLimit(maxMessageSize)
	// 设置读取超时
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// 设置 Pong 处理器，自动重置读超时
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// 读取客户端消息
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket 读取消息错误", map[string]interface{}{
					"error": err.Error(),
				})
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

// WritePump 处理向客户端发送消息的循环
// 运行在独立的 goroutine 中，从 send 通道读取消息并发送给客户端
// 同时负责定期发送心跳（Ping）消息
func (c *Client) WritePump() {
	// 创建心跳定时器
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.Unregister(c)
		c.conn.Close()
		logger.Info("客户端 WritePump 已退出", map[string]interface{}{})
	}()

	for {
		select {
		// 从发送通道读取消息
		case message, ok := <-c.send:
			// 设置写入超时
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道已关闭
				logger.Info("send 通道已关闭，WritePump 退出", nil)
				return
			}

			// 获取下一个写 writer
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logger.Error("获取 WebSocket writer 失败", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}
			w.Write(message)

			// 关闭 writer，完成消息发送
			if err := w.Close(); err != nil {
				logger.Error("关闭 WebSocket writer 失败", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}

		// 心跳周期到达
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("发送心跳失败，客户端可能已断开", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}

		// 收到退出信号
		case <-c.done:
			logger.Info("收到退出信号，WritePump 退出", nil)
			return
		}
	}
}

// handleMessage 处理从客户端接收到的消息
// message: 接收到的原始消息数据
func (c *Client) handleMessage(message []byte) {
	// 解析消息类型
	var msg struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		logger.Error("解析客户端消息失败", map[string]interface{}{
			"error": err.Error(),
		})
		c.sendError("无效的消息格式")
		return
	}

	// 根据消息类型处理
	switch msg.Type {
	case string(TypePing):
		// 客户端发送心跳请求，响应 pong
		c.sendPong()
	default:
		logger.Debug("收到未知类型的客户端消息", map[string]interface{}{
			"type": msg.Type,
		})
	}
}

// sendPong 发送心跳响应给客户端
func (c *Client) sendPong() {
	msg := &Message{
		Type:      TypePong,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化 pong 消息失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	c.send <- data
}

// sendError 发送错误消息给客户端
// errMsg: 错误信息
func (c *Client) sendError(errMsg string) {
	msg := &Message{
		Type:      TypeError,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化错误消息失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	c.send <- data
}
