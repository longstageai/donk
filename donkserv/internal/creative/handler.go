package creative

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	donksql "github.com/longstageai/donk/donk/internal/sql"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Handler 处理 Creative 模块的 HTTP 请求。
// 负责提供启动和停止 Creative 多 Agent 循环的 API 接口，
// 并将运行状态持久化到数据库中，支持服务重启后自动恢复。
type Handler struct {
	runtime *Runtime    // Creative 运行时实例，用于创建 Session 和执行循环
	runner  *LoopRunner // 持续循环运行器，控制多 Agent 的循环启停
	db      *donksql.DB // 数据库连接，用于保存和恢复运行状态
}

// NewHandler 创建一个新的 Handler 实例。
// runtime: Creative 运行时实例
// db: 数据库连接，用于持久化运行状态
func NewHandler(runtime *Runtime, db *donksql.DB) *Handler {
	h := &Handler{runtime: runtime, db: db}
	h.runner = NewLoopRunner(runtime, 10*time.Minute, func(reason string) {
		if err := h.saveRuntimeState("stopped"); err != nil {
			logger.Error("保存 creative 自动停止状态失败", map[string]interface{}{"reason": reason, "error": err.Error()})
		}
	})
	return h
}

// Restore 从数据库恢复 Creative 的运行状态。
// 如果数据库中记录的状态为 "running"，则自动启动循环。
// 此方法应在应用启动时调用，确保服务重启后能够恢复之前的运行状态。
// 注意：数据库的默认数据初始化在 setting/initializer.go 中统一处理。
func (h *Handler) Restore() error {
	if h.runner == nil || h.db == nil {
		return nil
	}

	// 查询保存的状态
	var status string
	err := h.db.QueryRow("SELECT status FROM creative_runtime_state WHERE id = 1").Scan(&status)
	if err != nil {
		// 没有记录或查询失败，不启动循环
		return nil
	}

	// 如果状态为运行中，则启动循环
	if status == "running" {
		h.runner.Start(context.Background())
	}
	return nil
}

// RegisterRoutes 注册 Creative 模块的 HTTP 路由。
// 提供以下接口：
//
//	GET  /api/v1/creative/status - 获取 Creative 多 Agent 循环状态
//	POST /api/v1/creative/start  - 启动 Creative 多 Agent 循环
//	POST /api/v1/creative/stop   - 停止 Creative 多 Agent 循环
func (h *Handler) RegisterRoutes(engine *gin.Engine) {
	group := engine.Group("/api/v1/creative")
	{
		group.GET("/status", h.Status)
		group.POST("/start", h.Start)
		group.POST("/stop", h.Stop)
	}
}

// Status 处理获取 Creative 多 Agent 循环状态的请求。
// 返回当前循环是否正在运行，以及当前正在执行的 Session ID（如果有）。
func (h *Handler) Status(c *gin.Context) {
	if h.runner == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runner not initialized"})
		return
	}

	// 获取运行状态和当前 Session ID
	running := h.runner.Running()
	currentSessionID := h.runner.CurrentSessionID()

	// 从数据库读取保存的状态
	dbStatus := "unknown"
	if h.db != nil {
		if err := h.ensureRuntimeStateTable(); err == nil {
			var status string
			if err := h.db.QueryRow("SELECT status FROM creative_runtime_state WHERE id = 1").Scan(&status); err == nil {
				dbStatus = status
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"running":    running,
		"db_status":  dbStatus,
		"session_id": currentSessionID,
	})
}

// Start 处理启动 Creative 多 Agent 循环的请求。
// 首先将运行状态保存到数据库，然后启动循环运行器。
// 如果循环已经在运行，返回 "already_started" 状态。
func (h *Handler) Start(c *gin.Context) {
	if h.runner == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runner not initialized"})
		return
	}

	// 保存运行状态到数据库
	if err := h.saveRuntimeState("running"); err != nil {
		logger.Error("保存 creative 运行状态失败", map[string]interface{}{"status": "running", "error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 启动循环运行器
	started := h.runner.Start(context.Background())
	if !started {
		c.JSON(http.StatusOK, gin.H{"status": "already_started"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// Stop 处理停止 Creative 多 Agent 循环的请求。
// 首先将停止状态保存到数据库，然后优雅地停止循环运行器。
// 停止操作会等待当前任务完成后再结束循环。
func (h *Handler) Stop(c *gin.Context) {
	if h.runner == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "creative runner not initialized"})
		return
	}

	// 保存停止状态到数据库
	if err := h.saveRuntimeState("stopped"); err != nil {
		logger.Error("保存 creative 运行状态失败", map[string]interface{}{"status": "stopped", "error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 优雅地停止循环运行器，设置 10 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	h.runner.Stop(ctx)

	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// saveRuntimeState 保存 Creative 的运行状态到数据库。
// status: 运行状态，"running" 表示运行中，"stopped" 表示已停止
// 使用 UPSERT 语义，如果记录已存在则更新，不存在则插入。
func (h *Handler) saveRuntimeState(status string) error {
	if h.db == nil {
		return nil
	}
	// 确保数据库表存在
	if err := h.ensureRuntimeStateTable(); err != nil {
		return err
	}
	// 插入或更新状态记录，id 固定为 1，确保只有一条记录
	_, err := h.db.Exec(`
		INSERT INTO creative_runtime_state (id, status, updated_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET status = excluded.status, updated_at = excluded.updated_at`, status, time.Now())
	return err
}

// ensureRuntimeStateTable 确保 creative_runtime_state 表存在。
// 如果表不存在，则创建该表。
// 表结构：
//   - id: 主键，固定为 1，确保只有一条记录
//   - status: 运行状态
//   - updated_at: 最后更新时间
func (h *Handler) ensureRuntimeStateTable() error {
	if h.db == nil {
		return nil
	}
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS creative_runtime_state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			status TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		)`)
	return err
}
