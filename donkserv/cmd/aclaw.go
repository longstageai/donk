package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/longstageai/donk/donk/internal/background"
	"github.com/longstageai/donk/donk/internal/config"
	"github.com/longstageai/donk/donk/internal/creative"
	"github.com/longstageai/donk/donk/internal/creative/agent"
	"github.com/longstageai/donk/donk/internal/db"
	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/http"
	"github.com/longstageai/donk/donk/internal/knowledge"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/scheduler"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/sql"
	"github.com/longstageai/donk/donk/internal/websocket"
	appctx "github.com/longstageai/donk/donk/pkg/context"

	"github.com/gin-gonic/gin"
)

// AppInitializer 应用程序初始化器
// 负责管理所有服务的初始化和依赖关系
type AppInitializer struct {
	app                  *appctx.Application
	db                   *sql.DB
	knowledgeInitializer *knowledge.Initializer
	httpServer           *http.Server
	engine               *gin.Engine
	wsServer             *websocket.Server
	scheduler            *scheduler.Scheduler
	bgManager            *background.Manager
	creativeRuntime      *creative.Runtime
	creativeHandler      *creative.Handler       // Creative HTTP 处理器
	historyStore         *memory.HistoryStore    // 历史记录存储（共享给Agent和Creative模块）
	profileMgr           *profile.ProfileManager // 用户画像管理器（共享给Agent和Creative模块）
}

// NewAppInitializer 创建应用程序初始化器
func NewAppInitializer() *AppInitializer {
	return &AppInitializer{}
}

// Init 执行完整的应用程序初始化
// 按照依赖顺序初始化各个服务
func (init *AppInitializer) Init() error {
	// 阶段1: 初始化基础服务
	if err := init.initBaseServices(); err != nil {
		return fmt.Errorf("初始化基础服务失败: %w", err)
	}

	// 阶段2: 初始化配置服务
	if err := init.initConfigServices(); err != nil {
		return fmt.Errorf("初始化配置服务失败: %w", err)
	}

	// 阶段3: 初始化核心服务
	if err := init.initCoreServices(); err != nil {
		return fmt.Errorf("初始化核心服务失败: %w", err)
	}

	// 阶段4: 初始化业务服务
	if err := init.initBusinessServices(); err != nil {
		return fmt.Errorf("初始化业务服务失败: %w", err)
	}

	// 阶段5: 注册优雅关闭钩子
	init.registerShutdownHooks()

	return nil
}

// initBaseServices 初始化基础服务
// 包括: 应用程序上下文、数据库连接
func (init *AppInitializer) initBaseServices() error {
	// 初始化应用程序上下文
	app, err := InitApp()
	if err != nil {
		return fmt.Errorf("初始化应用程序失败: %w", err)
	}
	init.app = app

	// 打开数据库连接
	db, err := OpenDB()
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}
	init.db = db

	return nil
}

// initConfigServices 初始化配置服务
// 包括: ConfigProvider、知识库模块、技能发现模块
func (init *AppInitializer) initConfigServices() error {
	// 初始化 ConfigProvider
	// 必须在创建其他服务之前完成，供其他模块使用
	if err := setting.InitConfigProvider(init.db.DB); err != nil {
		return fmt.Errorf("初始化配置提供者失败: %w", err)
	}

	// 初始化并启动知识库模块
	// 程序启动时自动启动定时器，默认每小时检查一次
	// 实际是否处理文档由数据库配置控制
	knowledgeInitializer, err := knowledge.InitAndStart(init.db.DB)
	if err != nil {
		// 非致命错误，记录日志但继续启动
		fmt.Printf("初始化知识库失败: %v\n", err)
	} else if knowledgeInitializer != nil {
		init.knowledgeInitializer = knowledgeInitializer
		// 将知识库控制器注入 setting 模块，供 HTTP API 使用
		setting.SetKnowledgeController(knowledgeInitializer)
	}

	return nil
}

// initCoreServices 初始化核心服务
// 包括: MultiAgent、HTTP服务器、WebSocket、调度器、Creative运行时
func (init *AppInitializer) initCoreServices() error {

	// 创建HTTP服务器并获取gin引擎
	httpServer, engine, err := NewHttp(init.app, init.db)
	if err != nil {
		return fmt.Errorf("初始化 HTTP 服务器失败: %w", err)
	}
	init.httpServer = httpServer
	init.engine = engine

	// 创建WebSocket事件推送服务
	wsServer, err := SetupWebSocket(init.app, engine)
	if err != nil {
		return fmt.Errorf("初始化 WebSocket 失败: %w", err)
	}
	init.wsServer = wsServer

	// 创建调度器服务（使用已创建的WebSocket服务器进行事件推送）
	sched, err := SetupScheduler(init.app, engine, init.db, wsServer)
	if err != nil {
		return fmt.Errorf("初始化调度器失败: %w", err)
	}
	init.scheduler = sched

	// 创建共享的历史记录存储（用于Agent和Creative模块共享）
	if err := init.initHistoryStore(); err != nil {
		fmt.Printf("初始化历史记录存储失败: %v\n", err)
		// 非致命错误，继续启动
	}

	// 创建共享的用户画像管理器（用于Agent和Creative模块共享）
	if err := init.initProfileManager(); err != nil {
		fmt.Printf("初始化用户画像管理器失败: %v\n", err)
		// 非致命错误，继续启动
	}

	// 初始化 Creative 运行时
	if err := init.initCreativeRuntime(); err != nil {
		// 非致命错误，记录日志但继续启动
		fmt.Printf("初始化 Creative 运行时失败: %v\n", err)
	}

	// 注册 Creative 路由
	if init.creativeRuntime != nil {
		init.creativeHandler = creative.NewHandler(init.creativeRuntime, init.db)
		init.creativeHandler.RegisterRoutes(engine)
		fmt.Println("Creative API 路由已注册: /api/v1/creative/*")
	}

	return nil
}

// initCreativeRuntime 初始化 Creative 多Agent运行时
func (init *AppInitializer) initCreativeRuntime() error {
	// 创建 Agent 注册表
	registry := creative.NewAgentRegistry()

	// 创建 LLM 客户端
	llmClient := agent.NewSettingModelLLMClient()

	// 创建目标创意 Agent 的依赖
	goalCreativeDeps := init.createGoalCreativeAgentDeps()

	// 创建任务执行 Agent 的依赖
	executionAgentDeps := init.createExecutionAgentDeps()

	// 注册所有 LLM Agents
	agent.RegisterLLMDefaultAgents(registry, llmClient, goalCreativeDeps, executionAgentDeps, init.db)

	// 创建 Runtime
	runtimeOptions := []creative.RuntimeOption{
		creative.WithTokenGuardAndDB(init.creativeTokenBudget(), init.db.DB),
	}

	// 如果 WebSocket 服务器可用，配置 WebSocket Hook
	if init.wsServer != nil {
		wsHook := creative.NewWebSocketHook(init.wsServer.Hub())
		runtimeOptions = append(runtimeOptions, creative.WithHookPipeline(wsHook))
	}
	runtime := creative.NewRuntime(registry, runtimeOptions...)

	init.creativeRuntime = runtime
	return nil
}

func (init *AppInitializer) creativeTokenBudget() creative.TokenBudget {
	budget := creative.TokenBudget{
		ID:                 creative.ID("creative_global_daily"),
		Scope:              creative.BudgetGlobal,
		ScopeID:            creative.ID("agent_daily_token_limit"),
		WarnThresholdRatio: 0.8,
		StopThresholdRatio: 1,
		OnWarn:             creative.TokenActionCompactContext,
		OnStop:             creative.TokenActionBlockSession,
	}

	provider := setting.GetProvider()
	if provider == nil {
		return budget
	}
	cfg, err := provider.GetAgentConfig()
	if err != nil || cfg == nil {
		return budget
	}
	if cfg.DailyTokenLimit > 0 {
		budget.MaxTotalTokens = cfg.DailyTokenLimit
	}
	return budget
}

// createGoalCreativeAgentDeps 创建目标创意 Agent 的依赖配置
func (init *AppInitializer) createGoalCreativeAgentDeps() *agent.GoalCreativeAgentDeps {
	deps := &agent.GoalCreativeAgentDeps{
		Scheduler: init.scheduler,
	}

	// 获取数据路径
	paths := config.GetDataPaths()
	deps.DataDir = paths.Knowledge
	deps.DBPath = paths.MainDB

	// 设置技能目录
	execPath, _ := os.Getwd()
	deps.SkillsDir = filepath.Join(execPath, "data", "skills")

	// 使用共享的历史记录存储
	if init.historyStore != nil {
		deps.HistoryStore = init.historyStore
	}

	// 尝试获取用户画像
	profile := init.getUserProfile()
	if profile != nil {
		deps.Profile = profile
	}

	// 尝试创建 embedder
	embedder, err := init.createEmbedder()
	if err != nil {
		fmt.Printf("创建 Embedder 失败，知识库搜索工具将不可用: %v\n", err)
	} else {
		deps.Embedder = embedder

		// 尝试创建长期记忆（需要 embedder）
		longMemory, err := init.createLongMemory(embedder)
		if err != nil {
			fmt.Printf("创建长期记忆失败，记忆搜索工具将不可用: %v\n", err)
		} else {
			deps.LongMemory = longMemory
		}
	}

	return deps
}

// createExecutionAgentDeps 创建任务执行 Agent 的依赖配置
func (init *AppInitializer) createExecutionAgentDeps() *agent.ExecutionAgentDeps {
	deps := &agent.ExecutionAgentDeps{
		Scheduler: init.scheduler,
	}

	// 获取执行路径作为工作目录
	execPath, _ := os.Getwd()
	deps.WorkDir = execPath

	// 设置技能目录
	deps.SkillsDir = filepath.Join(execPath, "data", "skills")

	// 使用共享的历史记录存储
	if init.historyStore != nil {
		deps.HistoryStore = init.historyStore
	}

	return deps
}

// initHistoryStore 初始化共享的历史记录存储
func (init *AppInitializer) initHistoryStore() error {
	paths := config.GetDataPaths()
	hs, err := memory.NewHistoryStore(paths.DataDir, 1000, 30)
	if err != nil {
		return err
	}
	init.historyStore = hs
	fmt.Println("历史记录存储初始化成功")
	return nil
}

// initProfileManager 初始化用户画像管理器
func (init *AppInitializer) initProfileManager() error {
	// 创建数据库存储
	storage := profile.NewDBStorage(init.db.DB)

	// 创建Provider缓存
	providerCache := model.NewProviderCache()

	// 创建画像管理器（使用默认用户ID）
	init.profileMgr = profile.NewProfileManager("default", providerCache, storage, nil)
	if err := init.profileMgr.Start(); err != nil {
		return fmt.Errorf("启动画像管理器失败: %w", err)
	}

	fmt.Println("用户画像管理器初始化成功")
	return nil
}

// getUserProfile 获取用户画像
func (init *AppInitializer) getUserProfile() *profile.UserProfile {
	if init.profileMgr == nil {
		return nil
	}
	return init.profileMgr.GetProfile()
}

// createEmbedder 创建向量嵌入模型
func (init *AppInitializer) createEmbedder() (embedding.Embedder, error) {
	provider := setting.GetProvider()
	if provider == nil {
		return nil, fmt.Errorf("ConfigProvider 未初始化")
	}

	embCfg, err := provider.GetEmbeddingConfig()
	if err != nil {
		return nil, fmt.Errorf("获取 Embedding 配置失败: %w", err)
	}
	if embCfg == nil {
		return nil, fmt.Errorf("Embedding 配置不存在")
	}

	return embedding.NewEmbedding(embCfg.Provider, embCfg.APIKey, embCfg.Model, embCfg.BaseURL)
}

// createLongMemory 创建长期记忆
func (init *AppInitializer) createLongMemory(embedder embedding.Embedder) (*memory.LongMemory, error) {
	// 创建向量数据库管理器
	manager, err := db.NewVectorDBManagerWithEmbedder(embedder)
	if err != nil {
		return nil, fmt.Errorf("创建 VectorDBManager 失败: %w", err)
	}

	// 创建长期记忆
	return memory.NewLongMemory(embedder, manager)
}

// initBusinessServices 初始化业务服务
// 包括: Agent服务、后台Agent服务
func (init *AppInitializer) initBusinessServices() error {
	// 创建Agent服务（内部注册SSE聊天路由）
	// 传入共享的 HistoryStore 和 ProfileManager，实现与 Creative 模块的数据共享
	_, err := NewAgentSvc(init.app, init.db.DB, init.scheduler, init.engine, init.historyStore, init.profileMgr)
	if err != nil {
		return fmt.Errorf("初始化 Agent 服务失败: %w", err)
	}

	// 给调度器注入 Agent 工厂函数
	// 使 Agent 执行器可以创建轻量级 Agent 实例
	if init.scheduler != nil {
		init.scheduler.SetAgentFactory(func() interface{} {
			agentInstance, _ := NewTaskAgent()
			return agentInstance
		})
	}

	// 创建并启动后台Agent服务
	// 从配置文件加载配置，自动启动所有启用的后台Agent
	bgManager, err := SetupBackgroundService(init.app, init.db.DB, init.wsServer)
	if err != nil {
		// 非致命错误，记录日志但继续启动
		fmt.Printf("初始化后台Agent服务失败: %v\n", err)
	}
	init.bgManager = bgManager

	return nil
}

// registerShutdownHooks 注册优雅关闭钩子
func (init *AppInitializer) registerShutdownHooks() {
	// 注册HTTP服务
	init.app.RegisterTaskFunc("http", func(ctx context.Context, application *appctx.Application) error {
		go func() error {
			if err := init.httpServer.Run(); err != nil {
				return err
			}
			return nil
		}()
		<-ctx.Done()
		init.httpServer.WaitForShutdown()
		return nil
	}, 0)

	// 注册知识库服务优雅关闭
	if init.knowledgeInitializer != nil {
		init.app.RegisterTaskFunc("knowledge", func(ctx context.Context, application *appctx.Application) error {
			<-ctx.Done()
			init.knowledgeInitializer.Stop()
			return nil
		}, 0)
	}

	// 注册后台Agent服务优雅关闭
	RegisterBackgroundShutdown(init.app, init.bgManager)

	// 注册 Creative 服务启动和优雅关闭
	if init.creativeHandler != nil {
		init.app.RegisterTaskFunc("creative", func(ctx context.Context, application *appctx.Application) error {
			// 服务启动时恢复运行状态
			if err := init.creativeHandler.Restore(); err != nil {
				fmt.Printf("恢复 Creative 运行状态失败: %v\n", err)
			}
			<-ctx.Done()
			return nil
		}, 0)
	}
}

// Run 启动应用程序
func (init *AppInitializer) Run() {
	init.app.Run()
}

func main() {
	// 创建并执行初始化器
	initializer := NewAppInitializer()
	if err := initializer.Init(); err != nil {
		fmt.Printf("应用程序初始化失败: %v\n", err)
		return
	}

	// 启动应用程序
	initializer.Run()
}
