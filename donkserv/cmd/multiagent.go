// multiagent 多Agent服务创建入口
package main

import (
	"database/sql"
	"fmt"
	"github.com/longstageai/donk/donk/internal/websocket"
	"os"
	"path/filepath"

	"github.com/longstageai/donk/donk/configs"
	"github.com/longstageai/donk/donk/internal/multiagent"
	"github.com/longstageai/donk/donk/internal/token"
	appctx "github.com/longstageai/donk/donk/pkg/context"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// NewMultiAgentSvc 创建多Agent服务实例
// 使用与单Agent共享的 TokenStats 实现统一预算管理
func NewMultiAgentSvc(app *appctx.Application, db *sql.DB, hub *websocket.Hub) (*multiagent.Service, error) {
	conf := &configs.Conf{
		Server: configs.ServerConfig{
			Host: "localhost",
			Port: 0,
		},
		Llm: configs.Model{
			Provider: app.ConfigBean().Llm.Provider,
			APIKey:   app.ConfigBean().Llm.APIKey,
			Model:    app.ConfigBean().Llm.Model,
			BaseURL:  app.ConfigBean().Llm.BaseURL,
		},
		Embedding: configs.Model{
			Provider: app.ConfigBean().Embedding.Provider,
			APIKey:   app.ConfigBean().Embedding.APIKey,
			Model:    app.ConfigBean().Embedding.Model,
			BaseURL:  app.ConfigBean().Embedding.BaseURL,
		},
		Agent: configs.AgentConfig{
			Name:              "MultiAgent",
			MaxLoop:           app.ConfigBean().Agent.MaxLoop,
			ConvergeAfter:     app.ConfigBean().Agent.ConvergeAfter,
			Timeout:           app.ConfigBean().Agent.Timeout,
			HistoryMaxEntries: app.ConfigBean().Agent.HistoryMaxEntries,
			HistoryMaxDays:    app.ConfigBean().Agent.HistoryMaxDays,
			DailyTokenLimit:   app.ConfigBean().Agent.DailyTokenLimit,
		},
	}

	log := logger.New(logger.WithLevel(logger.INFO), logger.WithWriter(logger.NewConsoleWriter(logger.ConsoleConfig{EnableColor: true})))

	var tokenStats *token.TokenStats
	var err error
	if db != nil {
		tokenStats, err = token.NewTokenStats(db)
		if err != nil {
			fmt.Printf("创建Token统计器失败，多Agent将使用独立预算: %v\n", err)
			tokenStats = nil
		}
	} else {
		fmt.Printf("数据库连接未提供，多Agent将不使用Token统计\n")
	}

	// 获取数据目录（使用绝对路径，与单Agent保持一致）
	execPath, _ := os.Getwd()
	dataDir := filepath.Join(execPath, "data")

	return multiagent.NewServiceWithTokenStats(conf, "让用户感受到温暖", log, tokenStats, hub, db, dataDir)
}
