package msgbus

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// isNormalWebSocketClose 判断 WebSocket 关闭是否为正常关闭
func isNormalWebSocketClose(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "1005") ||
		strings.Contains(errStr, "1000") ||
		strings.Contains(errStr, "normal")
}

// Server WebSocket 服务器
// 使用 gorilla/websocket 库实现
type Server struct {
	addr               string
	hub                *Hub
	server             *http.Server
	isClosed           bool
	mu                 sync.RWMutex
	onClientMessage    func(clientID string, msg *ClientMessage)
	onClientConnect    func(clientID string)
	onClientDisconnect func(clientID string)
	onSubscribe        func(clientID, topic string)
	onUnsubscribe      func(clientID, topic string)
	onPublish          func(clientID, topic string, payload interface{})
}

// ServerOption 函数选项模式
type ServerOption func(*Server)

// WithOnClientMessage 设置客户端消息处理回调
func WithOnClientMessage(fn func(clientID string, msg *ClientMessage)) ServerOption {
	return func(s *Server) {
		s.onClientMessage = fn
	}
}

// WithOnClientConnect 设置客户端连接回调
func WithOnClientConnect(fn func(clientID string)) ServerOption {
	return func(s *Server) {
		s.onClientConnect = fn
	}
}

// WithOnDisconnect 设置客户端断开回调
func WithOnClientDisconnect(fn func(clientID string)) ServerOption {
	return func(s *Server) {
		s.onClientDisconnect = fn
	}
}

// WithOnSubscribe 设置订阅回调
func WithOnSubscribe(fn func(clientID, topic string)) ServerOption {
	return func(s *Server) {
		s.onSubscribe = fn
	}
}

// WithOnUnsubscribe 设置取消订阅回调
func WithOnUnsubscribe(fn func(clientID, topic string)) ServerOption {
	return func(s *Server) {
		s.onUnsubscribe = fn
	}
}

// WithOnPublish 设置发布回调
func WithOnPublish(fn func(clientID, topic string, payload interface{})) ServerOption {
	return func(s *Server) {
		s.onPublish = fn
	}
}

// NewServer 创建并初始化 WebSocket 服务器
func NewServer(addr string, opts ...ServerOption) *Server {
	s := &Server{
		addr: addr,
		hub:  NewHub(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start 启动 WebSocket 服务器
func (s *Server) Start() error {
	go s.hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	logger.Info("WebSocket 服务器已启动", map[string]interface{}{
		"addr": s.addr,
	})

	return s.server.ListenAndServe()
}

// Stop 停止 WebSocket 服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	s.isClosed = true
	s.mu.Unlock()

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger.Debug("handleWebSocket 被调用", nil)

	// 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket 升级失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.Debug("WebSocket 升级成功", nil)

	// 创建客户端
	clientID := generateClientID()
	client := newClient(clientID, conn, s.hub)

	logger.Debug("准备注册客户端", map[string]interface{}{
		"clientID": clientID,
	})

	// 注册客户端
	if err := s.hub.Register(client); err != nil {
		logger.Error("客户端注册失败", map[string]interface{}{
			"clientID": clientID,
			"error":    err.Error(),
		})
		client.Close()
		return
	}

	logger.Info("客户端已连接", map[string]interface{}{
		"clientID": clientID,
	})

	// 触发连接回调
	if s.onClientConnect != nil {
		s.onClientConnect(clientID)
	}

	// 启动读写循环
	go s.readLoop(client)
	go s.writeLoop(client)
}

// readLoop 读取客户端消息的循环
func (s *Server) readLoop(client *Client) {
	defer func() {
		s.hub.Unregister(client)
		if s.onClientDisconnect != nil {
			s.onClientDisconnect(client.ID)
		}
	}()

	for {
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			if isNormalWebSocketClose(err) {
				logger.Debug("客户端连接已关闭", map[string]interface{}{
					"clientID": client.ID,
				})
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("读取客户端消息失败", map[string]interface{}{
					"clientID": client.ID,
					"error":    err.Error(),
				})
			} else {
				logger.Debug("客户端连接已关闭", map[string]interface{}{
					"clientID": client.ID,
					"error":    err.Error(),
				})
			}
			return
		}

		// 解析客户端消息
		msg, err := ParseClientMessage(data)
		if err != nil {
			logger.Error("解析客户端消息失败", map[string]interface{}{
				"clientID": client.ID,
				"error":    err.Error(),
			})
			continue
		}

		// 处理消息
		s.handleClientMessage(client, msg)
	}
}

// handleClientMessage 处理客户端消息
func (s *Server) handleClientMessage(client *Client, msg *ClientMessage) {
	logger.Debug("处理客户端消息", map[string]interface{}{
		"clientID": client.ID,
		"type":     msg.Type,
		"topic":    msg.Topic,
	})

	switch msg.Type {
	case TypeSubscribe:
		client.AddTopic(msg.Topic)
		logger.Debug("客户端订阅主题", map[string]interface{}{
			"clientID": client.ID,
			"topic":    msg.Topic,
		})
		if s.onSubscribe != nil {
			s.onSubscribe(client.ID, msg.Topic)
		}

	case TypeUnsubscribe:
		client.RemoveTopic(msg.Topic)
		logger.Debug("客户端取消订阅主题", map[string]interface{}{
			"clientID": client.ID,
			"topic":    msg.Topic,
		})
		if s.onUnsubscribe != nil {
			s.onUnsubscribe(client.ID, msg.Topic)
		}

	case TypePublish:
		logger.Debug("客户端发布消息", map[string]interface{}{
			"clientID": client.ID,
			"topic":    msg.Topic,
			"payload":  msg.Payload,
		})
		if s.onClientMessage != nil {
			s.onClientMessage(client.ID, msg)
		}
		if s.onPublish != nil {
			s.onPublish(client.ID, msg.Topic, msg.Payload)
		}

	case TypePing:
		client.SendBlocking(PongJSON())

	default:
		logger.Debug("收到未知类型的客户端消息", map[string]interface{}{
			"clientID": client.ID,
			"type":     msg.Type,
		})
	}
}

// writeLoop 写入消息到客户端的循环
func (s *Server) writeLoop(client *Client) {
	defer func() {
		client.Close()
	}()

	for {
		select {
		case data, ok := <-client.send:
			if !ok {
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Error("发送消息失败", map[string]interface{}{
					"clientID": client.ID,
					"error":    err.Error(),
				})
				return
			}

		case <-s.hub.done:
			return
		}
	}
}
