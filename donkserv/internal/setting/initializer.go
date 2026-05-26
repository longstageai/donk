package setting

import (
	"database/sql"
	"fmt"
	"strings"

	sqlmodule "github.com/longstageai/donk/donk/internal/sql"

	"github.com/gin-gonic/gin"
)

// Initializer 配置初始化器
// 负责在首次启动时创建默认配置
type Initializer struct {
	service *Service
}

// NewInitializer 创建初始化器实例
func NewInitializer(service *Service) *Initializer {
	return &Initializer{service: service}
}

// InitConfigProvider 初始化全局配置提供者
// 在应用启动时提前调用，确保其他模块可以使用 GetProvider() 获取配置
//
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - error: 错误信息
func InitConfigProvider(db *sql.DB) error {
	// 注册数据库表结构
	if err := initSchema(db); err != nil {
		return err
	}

	storage := NewStorage(db)
	InitProvider(storage)
	service := NewService(storage)
	initializer := NewInitializer(service)
	if err := initializer.Init(); err != nil {
		return err
	}

	// 初始化睡眠管理器并恢复之前的状态
	InitSleepManager(db)

	// 初始化 Creative 运行状态（如果不存在则插入默认记录）
	if err := initCreativeRuntimeState(db); err != nil {
		return err
	}

	return nil
}

// initCreativeRuntimeState 初始化 Creative 运行状态表
// 如果表中不存在记录，则插入默认记录（状态为 running）
//
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - error: 错误信息
func initCreativeRuntimeState(db *sql.DB) error {
	var status string
	err := db.QueryRow("SELECT status FROM creative_runtime_state WHERE id = 1").Scan(&status)
	if err == sql.ErrNoRows {
		// 没有记录，插入默认记录
		_, err = db.Exec("INSERT INTO creative_runtime_state (id, status, updated_at) VALUES (1, 'running', datetime('now'))")
		if err != nil {
			return fmt.Errorf("初始化 creative 运行状态失败: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("查询 creative 运行状态失败: %w", err)
	}
	return nil
}

// initSchema 初始化数据库表结构并执行兼容旧版本的字段迁移
// CREATE TABLE IF NOT EXISTS 不会修改已存在的表，因此这里额外补齐新增配置字段
func initSchema(db *sql.DB) error {
	for _, s := range sqlmodule.TableSchemas {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}

	migrations := []string{
		`ALTER TABLE config ADD COLUMN agent_history_max_entries INTEGER NOT NULL DEFAULT 100`,
		`ALTER TABLE config ADD COLUMN agent_history_max_days INTEGER NOT NULL DEFAULT 30`,
		`ALTER TABLE config ADD COLUMN knowledge_enabled INTEGER NOT NULL DEFAULT 1`,
	}
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil && !isDuplicateColumnError(err) {
			return err
		}
	}
	return nil
}

// isDuplicateColumnError 判断 ALTER TABLE 添加字段时是否因为字段已存在而失败
// 旧库重复启动时会再次执行迁移，字段已存在属于可忽略的幂等场景
func isDuplicateColumnError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// InitSleepManager 初始化睡眠管理器
// 从数据库恢复之前的睡眠阻止状态
// 如果数据库中没有记录，则默认开启阻止睡眠
//
// 参数:
//   - db: 数据库连接
func InitSleepManager(db *sql.DB) {
	sm := GetSleepManager(db)

	// 先尝试从数据库加载状态
	err := sm.LoadStateFromDB()
	if err == nil {
		// 成功加载，说明数据库中有记录，直接返回
		return
	}

	// 数据库中没有记录或加载失败，使用默认值：开启阻止睡眠
	// 插入默认状态到数据库
	db.Exec(
		`INSERT INTO system_state (key, value, updated_at) VALUES (?, ?, datetime('now'))`,
		SleepPreventedKey, "true",
	)
	db.Exec(
		`INSERT INTO system_state (key, value, updated_at) VALUES (?, ?, datetime('now'))`,
		SleepKeepDisplayKey, "true",
	)

	// 应用默认状态：阻止睡眠
	sm.Prevent(true)
}

// Init 初始化配置
// 如果配置已存在则跳过，不存在则创建默认配置
func (i *Initializer) Init() error {
	existing, err := i.service.GetConfig()
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	// 创建默认配置
	defaultConfig := &ConfigRequest{
		// LLM 默认配置
		LLMProvider:    "openai",      // LLM 提供商
		LLMModel:       "gpt-4o-mini", // LLM 模型名称
		LLMAPISKey:     "",            // LLM API 密钥（需用户填写）
		LLMBaseURL:     "",            // LLM API 地址（需用户填写）
		LLMTemperature: 0.7,           // LLM 温度参数
		LLMMaxTokens:   4096,          // LLM 最大 token 数

		// Embedding 默认配置
		EmbeddingProvider:  "openai",                 // Embedding 提供商
		EmbeddingModel:     "text-embedding-3-small", // Embedding 模型名称
		EmbeddingAPISKey:   "",                       // Embedding API 密钥（需用户填写）
		EmbeddingBaseURL:   "",                       // Embedding API 地址（需用户填写）
		EmbeddingDimension: 1536,                     // Embedding 向量维度

		// Agent 默认配置
		AgentName:              "donk", // Agent 名称
		AgentMaxLoop:           10,         // Agent 最大循环次数
		AgentConvergeAfter:     3,          // Agent 连续无工具调用终止数
		AgentTimeout:           300,        // Agent 超时时间（秒）
		AgentDailyTokenLimit:   -1,         // Agent 每日 Token 限额（-1 表示不限）
		AgentHistoryMaxEntries: 100,        // Agent 历史记录最大条目数
		AgentHistoryMaxDays:    30,         // Agent 历史记录保留天数

		// 知识库默认配置
		KnowledgeEnabled: true, // 知识库默认启用
	}
	return i.service.UpdateConfig(defaultConfig)
}

// Setup 初始化 Setting 模块
// 参数:
//   - db: 数据库连接
//   - engine: Gin 引擎实例
//   - registerSchema: 是否注册表结构
//   - authMiddleware: 认证中间件（可选）
func Setup(db *sql.DB, engine *gin.Engine, registerSchema bool, authMiddleware ...gin.HandlerFunc) (*gin.Engine, error) {
	// 注册数据库表结构
	if registerSchema {
		if err := initSchema(db); err != nil {
			return nil, err
		}
	}

	// 创建存储层和服务层
	storage := NewStorage(db)
	service := NewService(storage)

	// 初始化默认配置
	initializer := NewInitializer(service)
	if err := initializer.Init(); err != nil {
		return nil, err
	}

	// 初始化全局配置提供者（幂等：如果已初始化则跳过）
	if GetProvider() == nil {
		InitProvider(storage)
	}

	// 注册 HTTP 路由
	handler := NewHandler(service)
	registerRoutes(engine, handler, authMiddleware...)

	return engine, nil
}

// SetupWithPath 使用数据库文件路径初始化 Setting 模块
// 参数:
//   - dbPath: 数据库文件路径
//   - engine: Gin 引擎实例
//   - authMiddleware: 认证中间件（可选）
func SetupWithPath(dbPath string, engine *gin.Engine, authMiddleware ...gin.HandlerFunc) (*gin.Engine, error) {
	db, err := sqlmodule.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return Setup(db.DB, engine, true, authMiddleware...)
}

// registerRoutes 注册 API 路由
// 参数:
//   - engine: Gin 引擎实例
//   - handler: HTTP 处理器
//   - authMiddleware: 认证中间件（可选）
func registerRoutes(engine *gin.Engine, handler *Handler, authMiddleware ...gin.HandlerFunc) {
	// 健康检查（无需认证）
	engine.GET("/health", HealthCheck)

	// API 路由组
	api := engine.Group("/api/v1")

	// 可选的认证中间件
	if len(authMiddleware) > 0 && authMiddleware[0] != nil {
		//api.Use(authMiddleware[0])
	}

	// 完整配置接口（需放在具体配置路由之前，避免被匹配到 /config/llm 等路由）
	api.GET("/config", handler.GetConfig)
	api.PUT("/config", handler.UpdateConfig)

	// LLM 配置接口
	api.GET("/config/llm", handler.GetLLMConfig)
	api.PUT("/config/llm", handler.UpdateLLMConfig)

	// Embedding 配置接口
	api.GET("/config/embedding", handler.GetEmbeddingConfig)
	api.PUT("/config/embedding", handler.UpdateEmbeddingConfig)

	// Agent 配置接口
	api.GET("/config/agent", handler.GetAgentConfig)
	api.PUT("/config/agent", handler.UpdateAgentConfig)

	// 系统睡眠管理接口（仅 Windows 有效）
	api.GET("/system/sleep", handler.GetSleepStatus)
	api.POST("/system/sleep/prevent", handler.PreventSleepHandler)
	api.POST("/system/sleep/allow", handler.AllowSleepHandler)

	// 知识库配置接口
	api.GET("/config/knowledge", handler.GetKnowledgeConfig)
	api.PUT("/config/knowledge", handler.UpdateKnowledgeConfig)

	// 知识库控制接口
	api.GET("/knowledge/status", handler.GetKnowledgeStatus)
	api.POST("/knowledge/start", handler.StartKnowledge)
	api.POST("/knowledge/stop", handler.StopKnowledge)
}
