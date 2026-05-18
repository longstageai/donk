package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/longstageai/donk/donk/pkg/logger"
	"time"

	"github.com/gorilla/websocket"
)

// 连接读写超时和缓冲区大小配置
const (
	writeWait      = 60 * time.Second        // 写入操作的超时时间
	maxMessageSize = 512 * 1024              // 单条消息的最大字节数
	pingPeriod     = (readDeadline * 9) / 10 // Ping 消息发送周期（为 readDeadline 的 90%）
	readDeadline   = 3 * 60 * time.Second    // 读取操作的超时时间（也是 Pong 响应超时）
	sendBufferSize = 512 * 2                 // 发送缓冲区的容量
)

// Client 代表一个 WebSocket 客户端连接
// 每个 Client 都有自己的读写通道，与 Hub 的通信通过 channels 进行
type Client struct {
	ID         string
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	Context    context.Context
	CancelFunc context.CancelFunc
}

// NewClient 创建一个新的 Client 实例
// 参数 hub 是客户端所属的 Hub，conn 是已建立的 WebSocket 连接
func NewClient(hub *Hub, conn *websocket.Conn, ctx context.Context, clientID string) *Client {
	return &Client{
		ID:         clientID,
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, sendBufferSize),
		Context:    ctx,
		CancelFunc: nil,
	}
}

// ReadPump 负责从 WebSocket 连接读取消息
// 该方法运行在独立的 goroutine 中，读取到的消息会路由到业务处理器
func (c *Client) ReadPump() {
	// 函数返回时确保客户端已从 Hub 注销且连接已关闭
	defer func() {
		c.hub.Unregister(c) // 通知 Hub 移除此客户端
		c.conn.Close()      // 关闭 WebSocket 连接
	}()

	// 设置单条消息的最大字节数限制，防止恶意客户端发送超大消息
	c.conn.SetReadLimit(maxMessageSize)
	// 设置读取超时时间，超过此时间未收到消息则连接被关闭
	c.conn.SetReadDeadline(time.Now().Add(readDeadline))

	// 设置 Pong 处理器：当收到 Ping 消息时，自动回复 Pong
	// 同时重置读取超时计时器
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	// 持续读取消息循环
	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			// 检查是否为意外的连接关闭错误
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error(fmt.Sprintf("客户端异常断开: %v", err), nil)

			}
			break // 发生错误时退出循环
		}

		// 处理客户端发来的 Ping 消息
		if messageType == websocket.PingMessage {
			continue
		}

		// 处理普通文本消息
		if messageType == websocket.TextMessage {
			c.routeMessage(message)
		}
	}
}

// routeMessage 解析消息并路由到对应的业务处理器
func (c *Client) routeMessage(rawMessage []byte) {
	logger.Debug(fmt.Sprintf("收到客户端消息: %s", string(rawMessage)), nil)
	// 首先尝试解析消息类型
	var msgType struct {
		Type MessageType `json:"type"`
	}

	if err := json.Unmarshal(rawMessage, &msgType); err != nil {
		// 解析失败，默认作为聊天消息处理
		msgType.Type = TypeChat
	}

	// 创建消息结构
	msg := NewMessage(msgType.Type, rawMessage, c)

	// 路由到对应的业务处理器
	c.hub.RouteMessage(msg)
}

// WritePump 负责向 WebSocket 连接写入消息
// 该方法运行在独立的 goroutine 中，同时处理心跳检测
func (c *Client) WritePump() {
	// 创建定时器，定期发送 Ping 消息进行心跳检测
	ticker := time.NewTicker(pingPeriod)

	// 函数返回时停止定时器并关闭连接
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	// 持续处理发送队列和心跳循环
	for {
		select {
		// 处理待发送的消息
		case message, ok := <-c.send:
			// 设置写入超时时间
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// 通道已关闭（正常关闭），发送 Close 消息
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 获取下一个可用的 Writer
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			// 写入消息
			w.Write(message)
			// 关闭 Writer，完成消息发送
			if err := w.Close(); err != nil {
				return
			}

		// 处理心跳定时器（Ping 消息）
		case <-ticker.C:
			// 设置写入超时时间
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			// 发送 Ping 消息，客户端应回复 Pong
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Info(fmt.Sprintf("发送 Ping 失败，客户端可能已断开: %v", err), nil)
				return // 发送失败说明连接已断开
			}
		}
	}
}

// Send 用于向客户端发送消息
// 该方法是非阻塞的，如果发送缓冲区满则直接丢弃（防止 goroutine 阻塞）
func (c *Client) Send(message []byte) {
	select {
	case c.send <- message:
	default:
		// 缓冲区满，消息被丢弃
		// 这种背压策略可以防止慢客户端影响整体性能
	}
}
