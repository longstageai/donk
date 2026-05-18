package main

import (
	"github.com/longstageai/donk/donk/internal/scheduler"
	"github.com/longstageai/donk/donk/internal/sql"
	"github.com/longstageai/donk/donk/internal/websocket"
	appctx "github.com/longstageai/donk/donk/pkg/context"
	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/gin-gonic/gin"
)

// SetupScheduler 创建调度器服务并注册任务管理路由
// 参数app是应用程序上下文，engine是gin引擎实例，db是数据库连接，wsServer是WebSocket服务器
// 返回调度器实例和错误信息
// 自动注册的路由：
//   - GET/PUT /api/v1/tasks/* (调度任务管理)
func SetupScheduler(app *appctx.Application, engine *gin.Engine, db *sql.DB, wsServer *websocket.Server) (*scheduler.Scheduler, error) {
	// 创建调度器（使用外部传入的WebSocket服务器用于事件推送）
	sched, _, err := scheduler.SetupWithDB(db.DB, wsServer)
	if err != nil {
		logger.Error("调度器创建失败", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	// 注册调度器任务管理路由
	scheduler.RegisterRoutes(engine, sched)
	logger.Info("调度器路由已注册: GET/PUT /api/v1/tasks/*", nil)

	logger.Info("调度器服务初始化成功", nil)
	return sched, nil
}
