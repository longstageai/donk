package main

import (
	"path/filepath"

	"github.com/longstageai/donk/donk/internal/config"
	"github.com/longstageai/donk/donk/internal/http"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/sql"
	"github.com/longstageai/donk/donk/internal/token"
	appctx "github.com/longstageai/donk/donk/pkg/context"
	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/gin-gonic/gin"
)

// NewHttp 创建HTTP服务器并注册基础路由
// 参数app是应用程序上下文，db是数据库连接
// 返回HTTP服务器实例、gin引擎和错误信息
// 自动注册的路由：
//   - GET /health (健康检查)
//   - GET/PUT /api/v1/config/* (配置管理)
func NewHttp(app *appctx.Application, db *sql.DB) (*http.Server, *gin.Engine, error) {
	httpServer := http.New(
		http.WithAddress("0.0.0.0:65434"),
		http.WithShutdownCallback(func() {
			if err := db.Close(); err != nil {
				logger.Error("数据库关闭失败", map[string]interface{}{"error": err.Error()})
			}
			logger.Info("数据库连接已关闭", nil)
		}),
	)

	// 从配置读取认证密钥
	cfg := app.Config()
	var authMiddleware *http.AuthMiddleware
	if cfg.GetBool("auth.enabled") {
		keys := cfg.GetSlice("auth.api_keys")
		strKeys := make([]string, 0, len(keys))
		for _, k := range keys {
			if s, ok := k.(string); ok && s != "" {
				strKeys = append(strKeys, s)
			}
		}
		if len(strKeys) > 0 {
			authMiddleware = http.NewAuthMiddleware(strKeys...)
			logger.Info("API认证已启用", map[string]interface{}{"key_count": len(strKeys)})
		}
	}
	if authMiddleware == nil {
		authMiddleware = http.NewAuthMiddleware()
		logger.Warn("API认证未配置或未启用", nil)
	}

	engine := httpServer.Engine()

	// 注册配置管理路由（GET/PUT /api/v1/config/*）
	// 注意：表结构已在 InitConfigProvider 中创建，这里不再重复创建
	setting.Setup(db.DB, engine, false, authMiddleware.GinHandler())
	logger.Info("配置管理路由已注册: GET/PUT /api/v1/config/*", nil)

	// 注册 Skill 管理路由
	if err := setupSkillRoutes(app, db, engine); err != nil {
		logger.Warn("Skill 管理路由注册失败", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Info("Skill 管理路由已注册: /api/v1/skills/*", nil)
	}

	// 注册 Token 统计路由
	if _, err := token.Setup(db.DB, engine); err != nil {
		logger.Warn("Token 统计路由注册失败", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Info("Token 统计路由已注册: /api/v1/tokens/*", nil)
	}

	return httpServer, engine, nil
}

// setupSkillRoutes 设置 Skill 管理路由
// 参数:
//   - db: 数据库连接
//   - engine: Gin 引擎
//
// 返回:
//   - error: 设置错误
func setupSkillRoutes(app *appctx.Application, db *sql.DB, engine *gin.Engine) error {
	paths := config.GetDataPaths()
	skillDir := filepath.Join(paths.DataDir, "skills")

	// 创建 Loader
	loader := skill.NewSkillLoader(skillDir)

	// 创建状态 Repository
	stateRepo := skill.NewStateRepository(db.DB)

	// 同步文件系统到数据库
	if err := stateRepo.SyncFromLoader(loader); err != nil {
		return err
	}

	// 创建 Registry，只加载启用的 Skill
	//registry := skill.NewSkillRegistry()
	registry := skill.NewSkillRegistryWithLoader(loader)
	app.SetSkillRegistry(registry)

	skills, _ := loader.Load()
	for _, s := range skills {
		if state, _ := stateRepo.Get(s.Name()); state == nil || state.Enabled {
			_ = registry.Register(s)
		}
	}

	// 创建 Service
	svc := skill.NewService(stateRepo, loader, registry)

	// 创建 Handler 并注册路由
	handler := skill.NewHandler(svc)
	handler.RegisterRoutes(engine)

	// 启动文件监听（默认开启）
	watcher, err := skill.NewWatcher(skillDir, loader, stateRepo, registry)
	if err != nil {
		logger.Warn("Skill 文件监听创建失败", map[string]interface{}{"error": err.Error()})
	} else {
		if err := watcher.Start(); err != nil {
			logger.Warn("Skill 文件监听启动失败", map[string]interface{}{"error": err.Error()})
		} else {
			logger.Info("Skill 文件监听已启动", map[string]interface{}{"dir": skillDir})
		}
	}

	return nil
}
