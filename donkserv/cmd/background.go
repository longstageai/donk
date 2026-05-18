package main

import (
	"context"
	"database/sql"

	"github.com/longstageai/donk/donk/internal/background"
	"github.com/longstageai/donk/donk/internal/websocket"
	appctx "github.com/longstageai/donk/donk/pkg/context"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// SetupBackgroundService 创建并启动后台Agent服务
// 从配置文件加载配置，创建Manager并启动所有启用的Runner
//
// 参数:
//   - app: 应用程序上下文
//   - db: 数据库连接
//   - wsServer: WebSocket服务器
//
// 返回:
//   - *background.Manager: 后台Agent管理器实例
//   - error: 错误信息
//
// 注意:
//   - 如果配置文件不存在，会记录警告并返回nil（非致命错误）
//   - 如果global.enabled为false，服务不会启动
//   - 只有enabled为true的Agent才会被启动
func SetupBackgroundService(app *appctx.Application, db *sql.DB, wsServer *websocket.Server) (*background.Manager, error) {
	logger.Info("初始化后台Agent服务", nil)

	// 1. 加载配置文件
	config, err := background.LoadConfig("conf/background.yaml")
	if err != nil {
		// 配置文件不存在不是致命错误，记录警告并返回nil
		logger.Warn("后台Agent配置文件加载失败，服务将不启动", map[string]interface{}{
			"path":  "conf/background.yaml",
			"error": err.Error(),
		})
		return nil, nil
	}

	logger.Info("后台Agent配置文件加载成功", map[string]interface{}{
		"global_enabled":   config.Global.Enabled,
		"default_interval": config.Global.DefaultInterval,
		"agent_count":      len(config.Agents),
	})

	// 2. 检查总开关
	if !config.Global.Enabled {
		logger.Info("后台Agent服务已禁用（global.enabled=false），跳过初始化", nil)
		return nil, nil
	}

	// 3. 获取WebSocket Hub
	var wsHub *websocket.Hub
	if wsServer != nil {
		wsHub = wsServer.Hub()
		logger.Info("WebSocket Hub已关联", map[string]interface{}{
			"client_count": wsHub.ClientCount(),
		})
	} else {
		logger.Warn("WebSocket服务器未提供，后台Agent将无法推送消息", nil)
	}

	// 4. 创建管理器
	manager := background.NewManager(config, db, wsHub)

	// 5. 在独立goroutine中启动
	go func() {
		logger.Info("在后台goroutine中启动后台Agent服务", nil)
		manager.Start()
	}()

	logger.Info("后台Agent服务初始化完成", map[string]interface{}{
		"runner_count": len(config.GetEnabledAgents()),
	})

	return manager, nil
}

// RegisterBackgroundShutdown 注册后台Agent服务的优雅关闭
// 在应用程序关闭时停止所有Runner
//
// 参数:
//   - app: 应用程序上下文
//   - manager: 后台Agent管理器
func RegisterBackgroundShutdown(app *appctx.Application, manager *background.Manager) {
	if manager == nil {
		logger.Debug("后台Agent管理器为nil，跳过注册关闭钩子", nil)
		return
	}

	app.RegisterTaskFunc("background", func(ctx context.Context, application *appctx.Application) error {
		logger.Info("后台Agent服务收到关闭信号", nil)

		// 等待上下文取消信号
		<-ctx.Done()

		// 停止所有Runner
		logger.Info("正在停止所有后台Agent Runner", nil)
		manager.Stop()

		logger.Info("后台Agent服务已优雅关闭", nil)
		return nil
	}, 0)

	logger.Info("后台Agent服务关闭钩子已注册", nil)
}
