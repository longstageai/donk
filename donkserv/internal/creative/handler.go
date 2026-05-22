package creative

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Handler 处理 Creative 相关的 HTTP 请求
type Handler struct {
	runtime *Runtime // Creative运行时实例
}

// NewHandler 创建 Creative Handler
func NewHandler(runtime *Runtime) *Handler {
	return &Handler{runtime: runtime}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(engine *gin.Engine) {
	group := engine.Group("/api/v1/creative")
	{
		group.POST("/sessions", h.CreateSession)
		group.GET("/sessions/:id", h.GetSession)
		group.POST("/sessions/:id/start", h.StartSession)
		group.POST("/sessions/:id/pause", h.PauseSession)
		group.POST("/sessions/:id/resume", h.ResumeSession)
		group.POST("/sessions/:id/stop", h.StopSession)
		group.POST("/sessions/:id/cancel", h.CancelSession)
		group.GET("/sessions/:id/snapshot", h.GetSnapshot)
	}
}

// CreateSessionRequest 创建 Session 请求
type CreateSessionRequest struct {
	TriggerType string `json:"trigger_type" binding:"required"` // 触发类型
	Payload     any    `json:"payload"`                         // 请求载荷数据
}

// CreateSessionResponse 创建 Session 响应
type CreateSessionResponse struct {
	SessionID string `json:"session_id"` // Session唯一标识
	Status    string `json:"status"`     // Session状态
}

// CreateSession 创建新的 Creative Session
func (h *Handler) CreateSession(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trigger := Trigger{
		Type:    TriggerType(req.TriggerType),
		Payload: req.Payload,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessionID, err := h.runtime.StartSession(ctx, trigger)
	if err != nil {
		logger.Error("创建 creative session 失败", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, CreateSessionResponse{
		SessionID: string(sessionID),
		Status:    "created",
	})
}

// GetSession 获取 Session 状态
func (h *Handler) GetSession(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	sessionID := ID(c.Param("id"))
	session, ok := h.runtime.Store().GetSession(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// StartSession 启动 Session 事件循环
func (h *Handler) StartSession(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	sessionID := ID(c.Param("id"))

	// 异步启动事件循环
	go func() {
		ctx := context.Background()
		if err := h.runtime.StartLoop(ctx, sessionID); err != nil {
			logger.Error("启动 creative session 失败", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// PauseSession 暂停 Session
func (h *Handler) PauseSession(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	sessionID := ID(c.Param("id"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.runtime.PauseLoop(ctx, sessionID, "user requested"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "paused"})
}

// ResumeSession 恢复 Session
func (h *Handler) ResumeSession(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	sessionID := ID(c.Param("id"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.runtime.ResumeLoop(ctx, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "resumed"})
}

// StopSession 停止 Session
func (h *Handler) StopSession(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	sessionID := ID(c.Param("id"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.runtime.StopLoop(ctx, sessionID, StopGraceful, "user requested"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// CancelSession 取消 Session
func (h *Handler) CancelSession(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	sessionID := ID(c.Param("id"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.runtime.CancelSession(ctx, sessionID, "user requested"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

// GetSnapshot 获取 Session 快照
func (h *Handler) GetSnapshot(c *gin.Context) {
	if h.runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runtime not initialized"})
		return
	}

	sessionID := ID(c.Param("id"))
	snapshot := h.runtime.BuildSnapshot(sessionID)

	c.JSON(http.StatusOK, snapshot)
}
