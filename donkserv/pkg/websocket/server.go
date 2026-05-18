package websocket

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// upgrader 用于将 HTTP 连接升级为 WebSocket 连接
// 它配置了读写缓冲区大小和跨域检查策略
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // 读取缓冲区大小
	WriteBufferSize: 1024, // 写入缓冲区大小
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源的跨域请求（生产环境应更严格）
	},
}

// TokenValidator 是 token 校验接口
// 用于自定义 token 校验逻辑
type TokenValidator interface {
	// ValidateToken 校验 token 是否有效
	// 返回 true 表示 token 有效，连接可以建立
	// 返回 false 表示 token 无效，连接将被拒绝
	ValidateToken(token string) bool
}

// DefaultTokenValidator 是默认的 token 校验器
// 当未设置自定义校验器时使用
type DefaultTokenValidator struct{}

// ValidateToken 默认的 token 校验逻辑（始终返回 true）
func (v DefaultTokenValidator) ValidateToken(token string) bool {
	return true
}

// Server 是 WebSocket 服务器的核心结构
// 整合了 Hub 和 MessageRouter，提供完整的 WebSocket 服务能力
type Server struct {
	hub            *Hub           // 连接管理中心，管理所有客户端连接
	addr           string         // 服务器监听地址
	readTimeout    time.Duration  // HTTP 读取超时时间
	writeTimeout   time.Duration  // HTTP 写入超时时间
	httpServer     *http.Server   // HTTP 服务器实例
	tokenValidator TokenValidator // token 校验器
}

// NewServer 创建一个新的 WebSocket 服务器实例
// addr 格式为 ":8080" 或 "localhost:8080"
func NewServer(addr string) *Server {
	return &Server{
		hub:            NewHub(),         // 初始化 Hub
		addr:           addr,             // 监听地址
		readTimeout:    60 * time.Second, // 读取超时
		writeTimeout:   10 * time.Second, // 写入超时
		tokenValidator: DefaultTokenValidator{},
	}
}

// SetTokenValidator 设置自定义的 token 校验器
// 如果不设置，则使用默认的校验器（token 非空即通过）
func (s *Server) SetTokenValidator(validator TokenValidator) {
	s.tokenValidator = validator
}

// Start 启动 WebSocket 服务器
// 该方法会阻塞当前 goroutine，支持优雅退出（Ctrl+C 或 kill 信号）
func (s *Server) Start() error {
	// 启动 Hub 的主循环
	go s.hub.Run()

	// 创建 HTTP 服务器
	s.httpServer = &http.Server{
		Addr:         s.addr,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}
	// 注册 HTTP 处理器
	http.HandleFunc("/ws", s.handleWebSocket)  // WebSocket 连接端点
	http.HandleFunc("/health", s.handleHealth) // 健康检查端点

	// 启动服务器并监听
	logger.Info(fmt.Sprintf("WebSocket 服务器启动，监听地址: %s", s.addr), nil)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(fmt.Sprintf("服务器启动失败: %v", err), nil)
		return err
	}
	return nil
}

// waitForShutdown 监听系统信号并执行优雅退出
// 支持 SIGINT (Ctrl+C) 和 SIGTERM (kill 信号)
func (s *Server) waitForShutdown() {
	// 创建一个通道接收系统信号
	sigChan := make(chan os.Signal, 1)

	// 监听中断信号和终止信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	sig := <-sigChan
	logger.Info(fmt.Sprintf("收到退出信号: %v，开始优雅关闭...", sig), nil)

	// 执行优雅退出
	s.Shutdown()
}

// Shutdown 执行优雅退出流程
// 关闭 HTTP 服务器和所有 WebSocket 连接
func (s *Server) Shutdown() {
	// 设置超时上下文，5秒内必须完成退出
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭 HTTP 服务器（停止接受新连接）
	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Error(fmt.Sprintf("HTTP 服务器关闭错误: %v", err), nil)
	}

	// 关闭 Hub（关闭所有客户端连接）
	s.hub.Shutdown()
	logger.Info("WebSocket 服务器已完全关闭", nil)
}

// handleWebSocket 处理 WebSocket 连接请求
// 从 HTTP 升级到 WebSocket 协议，并创建 Client 实例
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 从 Header 中获取 token
	token := r.Header.Get("Authorization")
	if token == "" {
		// 尝试从 Query 参数获取 token（作为备选）
		token = r.URL.Query().Get("token")
	}

	// 校验 token
	if !s.tokenValidator.ValidateToken(token) {
		logger.Info(fmt.Sprintf("Token 校验失败: %s", token), nil)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized: invalid token"))
		return
	}
	// 将 HTTP 连接升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("WebSocket 升级失败: %v", err), nil)
		return
	}
	logger.Info(fmt.Sprintf("新的 WebSocket 连接来自: %s", r.RemoteAddr), nil)
	// 创建新的客户端实例
	ctx, cancel := context.WithCancel(context.Background())
	clientID := uuid.New().String()
	client := NewClient(s.hub, conn, ctx, clientID)
	client.CancelFunc = cancel

	// 向 Hub 注册客户端
	s.hub.Register(client)

	// 启动客户端的读写 goroutine
	go client.WritePump() // 处理写入和心跳
	go client.ReadPump()  // 处理读取和路由
}

// GetHub 获取 WebSocket 服务器的 Hub 实例
// 用于访问客户端管理功能
func (s *Server) GetHub() *Hub {
	return s.hub
}

// GetRouter 获取消息路由器
// 用于注册业务处理器
func (s *Server) GetRouter() *MessageRouter {
	return s.hub.router
}

// handleHealth 健康检查处理器
// 返回 200 OK
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
