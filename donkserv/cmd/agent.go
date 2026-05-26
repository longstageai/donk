package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/longstageai/donk/donk/internal/agent"
	"github.com/longstageai/donk/donk/internal/config"
	"github.com/longstageai/donk/donk/internal/conversation"
	"github.com/longstageai/donk/donk/internal/db"
	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/http"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/prompt"
	"github.com/longstageai/donk/donk/internal/scheduler"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
	"github.com/longstageai/donk/donk/internal/tool/middleware"
	appctx "github.com/longstageai/donk/donk/pkg/context"
	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/gin-gonic/gin"
)

// AgentBuilder Agent构建器
// 负责按依赖顺序初始化Agent所需的所有组件
type AgentBuilder struct {
	app *appctx.Application
	db  *sql.DB

	llmAdapter      model.Adapter           // LLM模型适配器
	embedder        embedding.Embedder      // 向量嵌入模型
	vectorDBManager *db.VectorDBManager     // 向量数据库管理器
	longMemory      *memory.LongMemory      // 长期记忆
	historyStore    *memory.HistoryStore    // 历史记录存储
	profileMgr      *profile.ProfileManager // 用户画像管理器
	conversationMgr *conversation.Manager   // 对话历史管理器
	toolRegistry    *tool.Registry          // 工具注册表
	skillRegistry   *skill.SkillRegistry    // Skill注册表
	tokenStats      *token.TokenStats       // Token统计
	providerCache   *model.ProviderCache    // Provider缓存
	historyLoader   *agent.HistoryLoader    // 历史记录加载器
	systemPrompt    string                  // 系统提示词
	workspace       string                  // 工作空间目录
	scheduler       *scheduler.Scheduler    // 任务调度器
}

// NewAgentBuilder 创建Agent构建器实例
func NewAgentBuilder(app *appctx.Application, db *sql.DB, sched *scheduler.Scheduler) *AgentBuilder {
	return &AgentBuilder{app: app, db: db, scheduler: sched}
}

// WithHistoryStore 设置历史记录存储
// 用于外部传入已创建的 HistoryStore，实现共享
func (b *AgentBuilder) WithHistoryStore(store *memory.HistoryStore) *AgentBuilder {
	b.historyStore = store
	return b
}

// WithProfileManager 设置用户画像管理器
// 用于外部传入已创建的 ProfileManager，实现共享
func (b *AgentBuilder) WithProfileManager(profileMgr *profile.ProfileManager) *AgentBuilder {
	b.profileMgr = profileMgr
	return b
}

// Build 构建Agent实例
// 按依赖顺序依次初始化各个组件，返回配置好的Agent实例
func (b *AgentBuilder) Build() *agent.Agent {
	// 初始化路径
	if err := b.initPaths(); err != nil {
		return nil
	}

	// 初始化LLM模型
	if err := b.initLLM(); err != nil {
		return nil
	}

	// 初始化向量嵌入模型
	if err := b.initEmbedder(); err != nil {
		return nil
	}

	// 初始化向量数据库管理器
	if err := b.initVectorDBManager(); err != nil {
		return nil
	}

	// 初始化长期记忆
	if err := b.initMemory(); err != nil {
		return nil
	}

	// 初始化历史记录存储
	if err := b.initHistory(); err != nil {
		return nil
	}

	// 初始化Token统计（无错误返回，需要在initProfile之前）
	b.initTokenStats()

	// 初始化用户画像管理器
	if err := b.initProfile(); err != nil {
		return nil
	}

	// 初始化对话历史管理器
	if err := b.initConversation(); err != nil {
		return nil
	}
	// 初始化工具注册表（无错误返回）
	b.initTools()
	// 初始化Skill系统（需要在initTools之后，因为需要toolRegistry）
	b.initSkills(b.app)
	// 初始化任务调度器工具（需要在initSkills之后）
	b.initSchedulerTool()
	// 初始化历史记录加载器（无错误返回）
	b.initHistoryLoader()
	// 初始化系统提示词（无错误返回）
	b.initSystemPrompt()

	// 创建Agent实例
	return b.createAgent()
}

// initPaths 初始化工作空间路径
func (b *AgentBuilder) initPaths() error {
	execPath, _ := os.Getwd()
	b.workspace = filepath.Join(execPath, "data")
	return nil
}

// initLLM 初始化LLM模型适配器
// 从数据库 ConfigProvider 读取配置，确保使用最新配置
func (b *AgentBuilder) initLLM() error {
	provider := setting.GetProvider()
	if provider == nil {
		return fmt.Errorf("ConfigProvider 未初始化")
	}

	llmCfg, err := provider.GetLLMConfig()
	if err != nil {
		logger.Error("获取LLM配置失败", map[string]interface{}{"error": err.Error()})
		return err
	}
	if llmCfg == nil {
		return fmt.Errorf("LLM 配置不存在")
	}

	adapter, err := model.NewAdapter(llmCfg.Provider, llmCfg.APIKey, llmCfg.Model, llmCfg.BaseURL)
	if err != nil {
		logger.Error("创建LLM适配器失败", map[string]interface{}{"error": err.Error()})
		return err
	}
	b.llmAdapter = adapter
	return nil
}

// initEmbedder 初始化向量嵌入模型
// 从数据库 ConfigProvider 读取配置，确保使用最新配置
// 创建失败时记录警告但继续启动，不影响程序正常运行
func (b *AgentBuilder) initEmbedder() error {
	provider := setting.GetProvider()
	if provider == nil {
		logger.Warn("ConfigProvider 未初始化，Embedder 不可用", nil)
		return nil
	}

	embCfg, err := provider.GetEmbeddingConfig()
	if err != nil {
		logger.Warn("获取Embedding配置失败，Embedder 不可用", map[string]interface{}{"error": err.Error()})
		return nil
	}
	if embCfg == nil {
		logger.Warn("Embedding 配置不存在，Embedder 不可用", nil)
		return nil
	}

	emb, err := embedding.NewEmbedding(embCfg.Provider, embCfg.APIKey, embCfg.Model, embCfg.BaseURL)
	if err != nil {
		logger.Warn("创建Embedder失败，Embedder 不可用", map[string]interface{}{"error": err.Error()})
		return nil
	}
	b.embedder = emb
	logger.Info("Embedder 初始化成功", nil)
	return nil
}

// initVectorDBManager 初始化向量数据库管理器
// 如果 Embedder 未初始化，则跳过向量数据库管理器的初始化
func (b *AgentBuilder) initVectorDBManager() error {
	if b.embedder == nil {
		logger.Warn("Embedder 未初始化，跳过 VectorDBManager 初始化", nil)
		return nil
	}

	manager, err := db.NewVectorDBManagerWithEmbedder(b.embedder)
	if err != nil {
		logger.Warn("初始化VectorDBManager失败，向量数据库功能不可用", map[string]interface{}{"error": err.Error()})
		return nil
	}
	b.vectorDBManager = manager
	logger.Info("VectorDBManager 初始化成功", nil)
	return nil
}

// initMemory 初始化长期记忆
// 如果 VectorDBManager 未初始化，则跳过长期记忆初始化
func (b *AgentBuilder) initMemory() error {
	if b.vectorDBManager == nil {
		logger.Warn("VectorDBManager 未初始化，跳过长期记忆初始化", nil)
		return nil
	}
	lm, err := memory.NewLongMemory(b.embedder, b.vectorDBManager)
	if err != nil {
		logger.Warn("初始化长期记忆失败", map[string]interface{}{"error": err.Error()})
	}
	b.longMemory = lm
	return nil
}

// initHistory 初始化历史记录存储
// 从数据库 ConfigProvider 读取配置，确保使用最新配置
// 如果外部已通过 WithHistoryStore 传入 HistoryStore，则跳过创建
func (b *AgentBuilder) initHistory() error {
	// 如果外部已传入 HistoryStore，跳过创建
	if b.historyStore != nil {
		logger.Info("使用外部传入的 HistoryStore", nil)
		return nil
	}

	provider := setting.GetProvider()
	if provider == nil {
		return fmt.Errorf("ConfigProvider 未初始化")
	}

	agentCfg, err := provider.GetAgentConfig()
	if err != nil {
		logger.Warn("获取Agent配置失败", map[string]interface{}{"error": err.Error()})
		// 使用默认值继续
		hs, err := memory.NewHistoryStore(b.workspace, 100, 30)
		if err != nil {
			logger.Warn("初始化历史记录存储失败", map[string]interface{}{"error": err.Error()})
		}
		b.historyStore = hs
		return nil
	}

	// 使用配置值，如果为0则使用默认值
	maxEntries := agentCfg.HistoryMaxEntries
	if maxEntries <= 0 {
		maxEntries = 100
	}
	maxAgeDays := agentCfg.HistoryMaxDays
	if maxAgeDays <= 0 {
		maxAgeDays = 30
	}

	hs, err := memory.NewHistoryStore(b.workspace, maxEntries, maxAgeDays)
	if err != nil {
		logger.Warn("初始化历史记录存储失败", map[string]interface{}{"error": err.Error()})
	}
	b.historyStore = hs
	return nil
}

// initProviderCache 初始化Provider缓存
func (b *AgentBuilder) initProviderCache() {
	b.providerCache = model.NewProviderCache()
	logger.Info("Provider缓存初始化成功", nil)
}

// initProfile 初始化用户画像管理器
// 如果外部已通过 WithProfileManager 传入 ProfileManager，则跳过创建
func (b *AgentBuilder) initProfile() error {
	// 如果外部已传入 ProfileManager，跳过创建
	if b.profileMgr != nil {
		logger.Info("使用外部传入的 ProfileManager", nil)
		return nil
	}

	// 创建数据库存储（表结构由 sql/setting.go 统一管理）
	storage := profile.NewDBStorage(b.db)

	// 确保Provider缓存已初始化
	if b.providerCache == nil {
		b.initProviderCache()
	}

	// 创建画像管理器（使用默认用户ID，传入ProviderCache和TokenStats）
	b.profileMgr = profile.NewProfileManager("default", b.providerCache, storage, b.tokenStats)
	if err := b.profileMgr.Start(); err != nil {
		return fmt.Errorf("启动画像管理器失败: %w", err)
	}

	return nil
}

// initConversation 初始化对话历史管理器
// 如果 VectorDBManager 未初始化，则跳过对话历史管理器初始化
func (b *AgentBuilder) initConversation() error {
	if b.vectorDBManager == nil {
		logger.Warn("VectorDBManager 未初始化，跳过对话历史管理器初始化", nil)
		return nil
	}
	cs, err := conversation.NewStore(b.embedder, b.vectorDBManager, nil)
	if err != nil {
		logger.Warn("初始化对话历史失败", map[string]interface{}{"error": err.Error()})
		return nil
	}
	search := conversation.NewSearch(cs)
	b.conversationMgr = conversation.NewManager(conversation.DefaultConfig, cs, search)
	b.conversationMgr.Start()
	return nil
}

// initTools 初始化工具注册表
// 注册内置工具和中间件
func (b *AgentBuilder) initTools() {
	registry := tool.NewRegistry()

	// 注册内置工具
	registry.Register(builtin.NewHTTP())
	registry.Register(builtin.NewCalculator())
	registry.Register(builtin.NewFileReader())
	registry.Register(builtin.NewFileWriter())
	registry.Register(builtin.NewPDFParser())
	registry.Register(builtin.NewWordParser())
	registry.Register(builtin.NewOfficialSkillsSearch())
	registry.Register(builtin.NewSkillInstaller(filepath.Join(b.workspace, "skills")))
	registry.Register(builtin.NewSkillCreator(filepath.Join(b.workspace, "skills")))
	registry.Register(builtin.NewCommandExecutor())
	// 注册脚本执行工具和 Python 依赖管理工具，使用 Donk 预置 runtime，并将 run 数据放到 workspace 下。
	scriptRuntimeDir := filepath.Join(b.workspace, "script_runtime")
	registry.Register(builtin.NewScriptRunner(builtin.WithScriptRunnerBaseDir(scriptRuntimeDir)))
	registry.Register(builtin.NewPythonDependencyManager(builtin.WithPythonDependencyManagerBaseDir(scriptRuntimeDir)))
	//registry.Register(builtin.NewBrowserController())
	registry.Register(builtin.NewMemorySaver(b.longMemory))
	registry.Register(builtin.NewMemorySearcher(b.longMemory))
	registry.Register(builtin.NewConversationSearchTool(b.conversationMgr))

	// 注册知识库搜索工具
	if b.embedder != nil && b.db != nil {
		paths := config.GetDataPaths()
		// 使用 MainDB 作为数据库文件路径
		knowledgeSearcher := builtin.NewKnowledgeSearcher(paths.Knowledge, paths.MainDB, b.embedder)
		registry.Register(knowledgeSearcher)
		logger.Info("知识库搜索工具已注册", nil)
	} else {
		logger.Warn("Embedder 未初始化，知识库搜索工具不可用", nil)
	}

	// 注册中间件
	registry.Use(tool.Middleware(middleware.LogMiddleware()))
	registry.Use(tool.Middleware(middleware.TimeoutMiddleware()))
	registry.Use(tool.Middleware(middleware.RetryMiddleware()))

	b.toolRegistry = registry
}

// initTokenStats 初始化Token统计
// 从数据库 ConfigProvider 读取配置，确保使用最新配置
func (b *AgentBuilder) initTokenStats() {
	if b.db == nil {
		logger.Warn("数据库连接未初始化，Token统计不可用", nil)
		return
	}

	stats, err := token.NewTokenStats(b.db)
	if err != nil {
		logger.Warn("初始化Token统计失败", map[string]interface{}{"error": err.Error()})
		return
	}
	b.tokenStats = stats
	logger.Info("Token统计初始化成功", nil)
}

// initSkills 初始化Skill系统
// 从数据库加载启用的Skill并注册到SkillRegistry
func (b *AgentBuilder) initSkills(app *appctx.Application) error {
	skillDir := filepath.Join(b.workspace, "skills")
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		logger.Info("Skill目录不存在，跳过初始化", map[string]interface{}{"dir": skillDir})
		return nil
	}

	loader := skill.NewSkillLoader(skillDir)
	registry := app.SkillRegistry

	// 从数据库获取启用的Skill列表
	stateRepo := skill.NewStateRepository(b.db)
	enabledStates, err := stateRepo.GetEnabled()
	if err != nil {
		logger.Warn("从数据库获取启用的Skill失败", map[string]interface{}{"error": err.Error()})
		// 数据库查询失败时，回退到加载所有Skill
		if err := registry.LoadAndRegister(); err != nil {
			logger.Warn("加载Skill失败", map[string]interface{}{"error": err.Error()})
			return err
		}
	} else if len(enabledStates) == 0 {
		// 数据库中没有记录时，加载文件系统中的所有Skill并自动启用
		logger.Info("数据库中无Skill记录，从文件系统加载所有Skill", nil)
		allSkills, err := loader.Load()
		if err != nil {
			logger.Warn("从文件系统加载Skill失败", map[string]interface{}{"error": err.Error()})
			return err
		}
		for _, s := range allSkills {
			if err := registry.Register(s); err != nil {
				logger.Warn("注册Skill失败", map[string]interface{}{"skill": s.Name(), "error": err.Error()})
				continue
			}
			// 将Skill状态写入数据库，默认启用
			if err := stateRepo.Save(s.Name(), s.Description(), true); err != nil {
				logger.Warn("保存Skill状态到数据库失败", map[string]interface{}{"skill": s.Name(), "error": err.Error()})
			}
		}
		logger.Info("从文件系统加载并启用所有Skill", map[string]interface{}{"count": len(registry.List())})
	} else {
		// 只加载数据库中启用的Skill
		for _, state := range enabledStates {
			skillPath := filepath.Join(skillDir, state.Name)
			s, err := loader.LoadFromDir(skillPath)
			if err != nil {
				logger.Warn("加载Skill失败", map[string]interface{}{"skill": state.Name, "error": err.Error()})
				continue
			}
			if err := registry.Register(s); err != nil {
				logger.Warn("注册Skill失败", map[string]interface{}{"skill": state.Name, "error": err.Error()})
			}
		}
		logger.Info("从数据库加载启用的Skill", map[string]interface{}{"count": len(registry.List())})
	}

	b.skillRegistry = registry
	logger.Info("Skill系统初始化成功", map[string]interface{}{"count": len(registry.List())})

	// 注册Skill工具到工具注册表
	if b.toolRegistry != nil {
		skillTool := builtin.NewSkillTool(registry, nil, b.workspace)
		b.toolRegistry.Register(skillTool)
		logger.Info("Skill工具已注册到工具注册表", nil)
	}

	return nil
}

// initSchedulerTool 初始化任务调度器工具
// 将调度器注册为 Agent 工具
func (b *AgentBuilder) initSchedulerTool() {
	if b.scheduler == nil {
		logger.Debug("调度器未初始化，跳过TaskManager工具注册", nil)
		return
	}

	if b.toolRegistry != nil {
		taskManager := builtin.NewTaskManager(b.scheduler)
		if err := b.toolRegistry.Register(taskManager); err != nil {
			logger.Warn("TaskManager工具注册失败", map[string]interface{}{"error": err.Error()})
			return
		}
		logger.Info("TaskManager工具已注册到工具注册表", nil)
	}
}

// initHistoryLoader 初始化历史记录加载器
// 从数据库 ConfigProvider 读取配置，确保使用最新配置
func (b *AgentBuilder) initHistoryLoader() {
	provider := setting.GetProvider()
	if provider == nil {
		logger.Warn("ConfigProvider 未初始化，跳过HistoryLoader初始化", nil)
		return
	}

	agentCfg, err := provider.GetAgentConfig()
	if err != nil {
		logger.Warn("获取Agent配置失败，跳过HistoryLoader初始化", map[string]interface{}{"error": err.Error()})
		return
	}

	if agentCfg.DailyTokenLimit > 0 {
		b.historyLoader = agent.NewHistoryLoader(
			agentCfg.HistoryMaxEntries,
			agentCfg.HistoryMaxDays)
	}
}

// initSystemPrompt 初始化系统提示词
func (b *AgentBuilder) initSystemPrompt() {
	loader := prompt.NewFileLoader(b.workspace)
	component := prompt.NewSystemComponent(loader)
	content, err := component.Content(nil)
	if err != nil {
		logger.Warn("加载系统提示词失败", map[string]interface{}{
			"error":     err.Error(),
			"workspace": b.workspace,
		})
		return
	}
	if content == "" {
		logger.Warn("系统提示词为空", map[string]interface{}{
			"workspace": b.workspace,
		})
		return
	}
	b.systemPrompt = content
	logger.Info("系统提示词加载成功", map[string]interface{}{
		"length": len(content),
	})
}

// createAgent 创建Agent实例
// 使用构建器中初始化的所有组件创建Agent
// 从数据库 ConfigProvider 读取配置，确保使用最新配置
func (b *AgentBuilder) createAgent() *agent.Agent {
	// 默认配置值
	maxLoop := 10
	convergeAfter := 3
	timeout := 300

	// 尝试从数据库读取配置
	provider := setting.GetProvider()
	if provider == nil {
		logger.Error("ConfigProvider 未初始化，使用默认配置创建Agent", nil)
	} else {
		agentCfg, err := provider.GetAgentConfig()
		if err != nil {
			logger.Error("获取Agent配置失败，使用默认配置", map[string]interface{}{"error": err.Error()})
		} else if agentCfg != nil {
			maxLoop = agentCfg.MaxLoop
			convergeAfter = agentCfg.ConvergeAfter
			timeout = agentCfg.Timeout
		}
	}

	// 构建Agent选项
	options := []agent.Option{
		agent.WithMaxLoop(maxLoop),
		agent.WithConvergeAfter(convergeAfter),
		agent.WithTimeout(time.Duration(timeout) * time.Second),
		agent.WithSystemPrompt(b.systemPrompt),
		agent.WithWorkspace(b.workspace),
		agent.WithProfileManager(b.profileMgr),
		agent.WithConversationManager(b.conversationMgr),
		agent.WithTokenStats(b.tokenStats),
		agent.WithHistoryLoader(b.historyLoader),
	}

	// 注册Skill系统到Agent（用于系统提示词中加载技能列表）
	if b.skillRegistry != nil {
		options = append(options, agent.WithSkillRegistry(b.skillRegistry, b.workspace))
	}

	// 设置Skill数据库连接和目录，用于每次对话时动态加载启用的技能
	skillDir := filepath.Join(b.workspace, "skills")
	options = append(options, agent.WithSkillDB(b.db, skillDir))

	return agent.New(
		b.llmAdapter,
		b.toolRegistry,
		b.longMemory,
		b.historyStore,
		options...,
	)
}

// NewAgent 创建Agent实例的入口函数
// 封装了AgentBuilder的构建过程
func NewAgent(app *appctx.Application, db *sql.DB, sched *scheduler.Scheduler, historyStore *memory.HistoryStore, profileMgr *profile.ProfileManager) (*agent.Agent, error) {
	builder := NewAgentBuilder(app, db, sched)
	// 如果传入了 HistoryStore，使用它
	if historyStore != nil {
		builder.WithHistoryStore(historyStore)
	}
	// 如果传入了 ProfileManager，使用它
	if profileMgr != nil {
		builder.WithProfileManager(profileMgr)
	}
	agentInstance := builder.Build()
	if agentInstance == nil {
		return nil, fmt.Errorf("Agent初始化失败")
	}
	return agentInstance, nil
}

// TaskAgentBuilder 轻量级任务Agent构建器
// 用于定时任务执行，避免数据库锁定问题
// 仅初始化 LLM 模型和基础工具，不包含知识库、长期记忆等功能
type TaskAgentBuilder struct {
	llmAdapter model.Adapter
	tools      *tool.Registry
}

// NewTaskAgent 创建轻量级任务Agent实例
// 使用 TaskAgentBuilder 构建，保持与主 AgentBuilder 一致的构建模式
func NewTaskAgent() (*agent.Agent, error) {
	builder := &TaskAgentBuilder{}

	// 初始化LLM
	if err := builder.initLLM(); err != nil {
		return nil, err
	}

	// 初始化工具
	builder.initTools()

	execPath, _ := os.Getwd()
	workspace := filepath.Join(execPath, "data")

	// 创建轻量级 Agent
	return agent.New(builder.llmAdapter, builder.tools, nil, nil, agent.WithWorkspace(workspace)), nil
}

// initLLM 初始化LLM模型适配器
func (b *TaskAgentBuilder) initLLM() error {
	provider := setting.GetProvider()
	if provider == nil {
		return fmt.Errorf("ConfigProvider 未初始化")
	}

	llmCfg, err := provider.GetLLMConfig()
	if err != nil {
		return fmt.Errorf("获取 LLM 配置失败: %v", err)
	}
	if llmCfg == nil {
		return fmt.Errorf("LLM 配置不存在")
	}

	adapter, err := model.NewAdapter(llmCfg.Provider, llmCfg.APIKey, llmCfg.Model, llmCfg.BaseURL)
	if err != nil {
		return fmt.Errorf("创建LLM适配器失败: %v", err)
	}

	b.llmAdapter = adapter
	return nil
}

// initTools 初始化工具注册表
func (b *TaskAgentBuilder) initTools() {
	registry := tool.NewRegistry()

	// 注册轻量级内置工具
	registry.Register(builtin.NewHTTP())
	registry.Register(builtin.NewCalculator())
	registry.Register(builtin.NewFileReader())
	registry.Register(builtin.NewFileWriter())
	registry.Register(builtin.NewCommandExecutor())
	execPath, _ := os.Getwd()
	// 注册脚本执行工具和 Python 依赖管理工具，轻量级 Agent 使用当前工作目录下的 data/script_runtime。
	scriptRuntimeDir := filepath.Join(execPath, "data", "script_runtime")
	registry.Register(builtin.NewScriptRunner(builtin.WithScriptRunnerBaseDir(scriptRuntimeDir)))
	registry.Register(builtin.NewPythonDependencyManager(builtin.WithPythonDependencyManagerBaseDir(scriptRuntimeDir)))
	//registry.Register(builtin.NewBrowserController())

	// 注册中间件
	registry.Use(tool.Middleware(middleware.LogMiddleware()))
	registry.Use(tool.Middleware(middleware.TimeoutMiddleware()))
	registry.Use(tool.Middleware(middleware.RetryMiddleware()))

	b.tools = registry
}

// NewAgentSvc 创建AgentService实例的入口函数
// 参数app是应用程序上下文，db是数据库连接，sched是调度器，engine是gin引擎实例用于注册路由
// 内部创建Agent实例，注册SSE聊天路由，并返回AgentService
// 如果传入 historyStore 和 profileMgr，则会共享使用
func NewAgentSvc(app *appctx.Application, db *sql.DB, sched *scheduler.Scheduler, engine *gin.Engine, historyStore *memory.HistoryStore, profileMgr *profile.ProfileManager) (*agent.AgentService, error) {
	agentInstance, err := NewAgent(app, db, sched, historyStore, profileMgr)
	if err != nil {
		return nil, err
	}

	agentSvc := agent.NewAgentService(agentInstance, app.Config())

	// 创建LLM Provider缓存（启动时预创建所有Provider）
	providerCache := model.NewProviderCache()
	logger.Info("LLM Provider缓存已初始化", map[string]interface{}{
		"providers": providerCache.List(),
	})

	// 注册Agent SSE聊天路由
	chatHandler := http.NewChatHandler(agentSvc, providerCache)
	chatHandler.RegisterRoutes(engine)
	logger.Info("Agent SSE聊天路由已注册: POST /api/v1/chat", nil)

	return agentSvc, nil
}
