package http

import (
	"context"
	"net/http"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器结构体
// 封装Gin引擎、服务器配置和优雅关闭功能
type Server struct {
	engine           *gin.Engine
	addr             string
	httpServer       *http.Server
	shutdownCallback func() // 关闭时的回调函数（如关闭数据库连接）
}

// Option 服务器配置选项函数类型
type Option func(*Server)

// New 创建HTTP服务器实例
// 参数:
//   - opts: 配置选项
//
// 返回:
//   - *Server: HTTP服务器实例
func New(opts ...Option) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{
		engine: engine,
		addr:   ":8080",
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithAddress 设置服务器监听地址
// 示例: http.WithAddress(":8080")
func WithAddress(addr string) Option {
	return func(s *Server) {
		s.addr = addr
	}
}

// WithGinMode 设置Gin运行模式
// 示例: http.WithGinMode(gin.DebugMode)
func WithGinMode(mode string) Option {
	return func(s *Server) {
		gin.SetMode(mode)
	}
}

// WithShutdownCallback 设置关闭时的回调函数
// 用于在服务器关闭时执行清理操作（如关闭数据库连接）
func WithShutdownCallback(callback func()) Option {
	return func(s *Server) {
		s.shutdownCallback = callback
	}
}

// Engine 获取Gin引擎实例
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Addr 获取监听地址
func (s *Server) Addr() string {
	return s.addr
}

// Run 启动HTTP服务器
// 在goroutine中启动服务器并监听系统信号实现优雅关闭
func (s *Server) Run() error {
	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: s.engine,
	}
	logger.Infof("HTTP Server 启动成功，监听地址: %s", s.addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("HTTP Server 启动失败: %v", err)
		return err
	}
	return nil
}

// RunTLS 启动HTTPS服务器
// 参数:
//   - certFile: 证书文件路径
//   - keyFile: 私钥文件路径
func (s *Server) RunTLS(certFile, keyFile string) {
	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: s.engine,
	}

	go func() {
		logger.Infof("HTTPS Server 启动成功，监听地址: %s", s.addr)
		if err := s.httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTPS Server 启动失败: %v", err)
		}
	}()
}

// WaitForShutdown 等待服务器关闭
// 监听SIGINT和SIGTERM信号，执行优雅关闭：
// 1. 停止接收新请求
// 2. 等待现有请求处理完成（最多10秒）
// 3. 执行关闭回调
//
// 返回:
//   - error: 关闭过程中的错误信息
func (s *Server) WaitForShutdown() error {
	logger.Info("正在关闭 HTTP Server...", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Errorf("HTTP Server 关闭出错: %v", err)
	} else {
		logger.Info("HTTP Server 已正常关闭", nil)
	}

	if s.shutdownCallback != nil {
		logger.Info("执行关闭回调...", nil)
		s.shutdownCallback()
	}

	return nil
}

// GetEngine 获取Gin引擎实例（兼容旧版本）
func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}
