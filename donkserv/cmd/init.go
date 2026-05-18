package main

import (
	"github.com/longstageai/donk/donk/internal/config"
	"github.com/longstageai/donk/donk/internal/sql"
	appctx "github.com/longstageai/donk/donk/pkg/context"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// InitApp 应用初始化
// 创建应用程序上下文并初始化
func InitApp() (*appctx.Application, error) {
	// 初始化数据目录
	if err := config.InitDataDir(); err != nil {
		return nil, err
	}

	app := appctx.NewBuilder().
		WithAppName("donk").
		WithVersion("1.0.0").
		//WithConfigPath("./conf/config.yaml").
		WithLoggerLevel(logger.INFO).
		WithConsoleLogger(true).
		WithFileLogger(false).
		Build()

	if err := app.Initialize(); err != nil {
		return nil, err
	}

	return app, nil
}

// OpenDB 打开数据库连接
// 返回数据库连接实例
func OpenDB() (*sql.DB, error) {
	paths := config.GetDataPaths()
	db, err := sql.Open(paths.MainDB)
	if err != nil {
		logger.Error("数据库连接失败", map[string]interface{}{"error": err.Error()})
		return nil, err
	}
	logger.Info("数据库连接成功", map[string]interface{}{"path": paths.MainDB})
	return db, nil
}
