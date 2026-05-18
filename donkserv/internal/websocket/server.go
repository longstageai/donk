package websocket

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// upgrader WebSocket 升级配置
// 用于将 HTTP 连接升级为 WebSocket 连接
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // 读取缓冲区大小
	WriteBufferSize: 1024, // 写入缓冲区大小
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源的跨域请求（生产环境应限制）
	},
}

// Server WebSocket 服务器
// 提供 HTTP 到 WebSocket 的升级能力
type Server struct {
	hub *Hub // 连接的 Hub 管理器
}

// NewServer 创建并初始化 WebSocket 服务器
// 返回带有运行中 Hub 的服务器实例
func NewServer() *Server {
	hub := NewHub()
	// 在独立 goroutine 中启动 Hub
	go hub.Run()
	return &Server{hub: hub}
}

// Hub 返回服务器关联的 Hub 管理器
// 返回: Hub 实例
func (s *Server) Hub() *Hub {
	return s.hub
}

// HandleWebSocket 处理 WebSocket 连接请求
// 这是 Gin 的处理函数，用于处理客户端的 WebSocket 升级请求
// c: Gin 上下文，包含 HTTP 请求和响应
func (s *Server) HandleWebSocket(c *gin.Context) {
	// 将 HTTP 连接升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket 连接升级失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 获取可选的任务 ID 查询参数，用于过滤特定任务的事件
	taskID := c.Query("task_id")
	if taskID != "" {
		logger.Info("客户端订阅特定任务", map[string]interface{}{
			"taskID": taskID,
		})
	}

	// 创建客户端实例
	client := NewClient(s.hub, conn, taskID)
	// 注册到 Hub
	s.hub.Register(client)

	// 启动读写泵
	go client.WritePump()
	go client.ReadPump()

	// 等待注册完成（给 Hub 处理时间）
	time.Sleep(10 * time.Millisecond)

	logger.Info("WebSocket 客户端已连接", map[string]interface{}{
		"taskID":    taskID,
		"clientNum": s.hub.ClientCount(),
		"clientPtr": fmt.Sprintf("%p", client),
	})
}
