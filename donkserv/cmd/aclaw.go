package main

import (
	"context"
	"fmt"
	"github.com/longstageai/donk/donk/internal/background"
	"github.com/longstageai/donk/donk/internal/creative"
	"github.com/longstageai/donk/donk/internal/creative/agent"
	"github.com/longstageai/donk/donk/internal/http"
	"github.com/longstageai/donk/donk/internal/knowledge"
	"github.com/longstageai/donk/donk/internal/multiagent"
	"github.com/longstageai/donk/donk/internal/scheduler"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/skilldiscovery"
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
	skillDiscoveryInit   *skilldiscovery.Initializer
	multiAgentService    *multiagent.Service
	httpServer           *http.Server
	engine               *gin.Engine
	wsServer             *websocket.Server
	scheduler            *scheduler.Scheduler
	bgManager            *background.Manager
	creativeRuntime      *creative.Runtime
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

	// 初始化技能发现模块
	//skillDiscoveryInit, err := skilldiscovery.Init(init.db.DB)
	//if err != nil {
	//	// 非致命错误，记录日志但继续启动
	//	fmt.Printf("初始化技能发现模块失败: %v\n", err)
	//} else {
	//	init.skillDiscoveryInit = skillDiscoveryInit
	//}

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

	// 给 skilldiscovery 设置 WebSocket Hub（如果已初始化）
	if init.skillDiscoveryInit != nil {
		init.skillDiscoveryInit.SetWebSocketHub(wsServer.Hub())
	}
	// 创建多Agent服务（此时 ConfigProvider 已可用）
	multiAgentService, err := NewMultiAgentSvc(init.app, init.db.DB, wsServer.Hub())
	if err != nil {
		return fmt.Errorf("初始化 MultiAgent 失败: %w", err)
	}
	init.multiAgentService = multiAgentService

	// 创建调度器服务（使用已创建的WebSocket服务器进行事件推送）
	sched, err := SetupScheduler(init.app, engine, init.db, wsServer)
	if err != nil {
		return fmt.Errorf("初始化调度器失败: %w", err)
	}
	init.scheduler = sched

	// 初始化 Creative 运行时
	if err := init.initCreativeRuntime(); err != nil {
		// 非致命错误，记录日志但继续启动
		fmt.Printf("初始化 Creative 运行时失败: %v\n", err)
	}

	// 注册 Creative 路由
	if init.creativeRuntime != nil {
		creativeHandler := creative.NewHandler(init.creativeRuntime)
		creativeHandler.RegisterRoutes(engine)
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

	// 注册所有 LLM Agents
	agent.RegisterLLMDefaultAgents(registry, llmClient)

	// 创建 Runtime
	runtime := creative.NewRuntime(registry)

	init.creativeRuntime = runtime
	fmt.Println("Creative 运行时初始化成功")
	go func() {
		s1, _ := init.creativeRuntime.StartSession(context.Background(), creative.Trigger{
			"TimerTriggered",
			"温暖小助手",
		})
		init.creativeRuntime.StartLoop(context.Background(), s1)
	}()
	return nil
}

// initBusinessServices 初始化业务服务
// 包括: Agent服务、后台Agent服务
func (init *AppInitializer) initBusinessServices() error {
	// 创建Agent服务（内部注册SSE聊天路由）
	_, err := NewAgentSvc(init.app, init.db.DB, init.scheduler, init.engine)
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

	// 注册MultiAgent服务
	init.app.RegisterTaskFunc("multiagent", func(ctx context.Context, application *appctx.Application) error {
		go func() {
			//init.multiAgentService.Start()
		}()
		<-ctx.Done()
		init.multiAgentService.Stop()
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

	// 注册技能发现服务优雅关闭
	if init.skillDiscoveryInit != nil {
		init.app.RegisterTaskFunc("skilldiscovery", func(ctx context.Context, application *appctx.Application) error {
			<-ctx.Done()
			init.skillDiscoveryInit.Stop()
			return nil
		}, 0)
	}

	// 注册后台Agent服务优雅关闭
	RegisterBackgroundShutdown(init.app, init.bgManager)
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
