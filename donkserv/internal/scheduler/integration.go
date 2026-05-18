package scheduler

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/longstageai/donk/donk/internal/websocket"
)

// SetupWithDB 使用现有数据库创建并启动调度器
// 参数db是已打开的数据库连接，wsServer是可选的WebSocket服务器（用于事件推送）
// 如果wsServer为nil，则内部创建
// 返回调度器实例、WebSocket服务器实例和错误信息
func SetupWithDB(db *sql.DB, wsServer *websocket.Server) (*Scheduler, *websocket.Server, error) {

	// 如果没有提供WebSocket服务器，则创建一个
	if wsServer == nil {
		wsServer = websocket.NewServer()
	}

	// 创建执行器工厂
	factory := NewDefaultExecutorFactory()

	// 创建事件总线（带日志 + WebSocket）
	eventBus := NewEventBus()
	eventBus.Subscribe(NewLogSubscriber())
	eventBus.Subscribe(NewWebSocketSubscriber(wsServer.Hub()))

	// 使用现有的数据库连接创建调度器
	repo := NewSQLiteTaskRepository(db)
	runRepo := NewSQLiteTaskRunRepository(db)
	sched := NewScheduler(wsServer.Hub(), repo, factory, runRepo,
		WithWorkers(5),
		WithEventBus(eventBus),
	)

	// 启动调度器
	if err := sched.Start(); err != nil {
		return nil, nil, fmt.Errorf("启动调度器失败: %w", err)
	}

	return sched, wsServer, nil
}

// RegisterRoutes 注册调度器 API 路由
// ginEngine: gin 引擎实例
// scheduler: 调度器实例
func RegisterRoutes(ginEngine *gin.Engine, scheduler *Scheduler) {
	apiHandler := NewAPIHandler(scheduler)
	apiHandler.RegisterRoutes(ginEngine)
}
