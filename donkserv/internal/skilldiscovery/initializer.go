// skilldiscovery 技能自动发现模块
// 初始化器和定时执行管理
package skilldiscovery

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/longstageai/donk/donk/internal/memory"
	"os"
	"path/filepath"
	"time"

	"github.com/longstageai/donk/donk/internal/conversation"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/skilldiscovery/agents"
	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Initializer 模块初始化器
type Initializer struct {
	config       *Config
	stateRepo    *skill.StateRepository
	settingSvc   *setting.ConfigProvider
	convStore    *conversation.Store
	websocketHub *websocket.Hub
	skillsDir    string
	db           *sql.DB
	ticker       *time.Ticker
	stopCh       chan struct{}
	executor     *Executor
}

// InitializerOption 初始化器选项
type InitializerOption func(*Initializer)

// WithStateRepository 设置 Skill 状态仓库
func WithStateRepository(repo *skill.StateRepository) InitializerOption {
	return func(i *Initializer) {
		i.stateRepo = repo
	}
}

// WithSettingService 设置配置服务
func WithSettingService(svc *setting.ConfigProvider) InitializerOption {
	return func(i *Initializer) {
		i.settingSvc = svc
	}
}

// WithConversationStore 设置对话存储
func WithConversationStore(store *conversation.Store) InitializerOption {
	return func(i *Initializer) {
		i.convStore = store
	}
}

// WithWebSocketHub 设置 WebSocket Hub
func WithWebSocketHub(hub *websocket.Hub) InitializerOption {
	return func(i *Initializer) {
		i.websocketHub = hub
	}
}

// WithSkillsDirectory 设置技能目录
func WithSkillsDirectory(dir string) InitializerOption {
	return func(i *Initializer) {
		i.skillsDir = dir
	}
}

// WithDB 设置数据库连接
func WithDB(db *sql.DB) InitializerOption {
	return func(i *Initializer) {
		i.db = db
	}
}

// NewInitializer 创建初始化器
// 参数:
//   - config: 配置
//   - opts: 初始化选项
//
// 返回:
//   - *Initializer: 初始化器实例
func NewInitializer(config *Config, opts ...InitializerOption) *Initializer {
	if config == nil {
		config = DefaultConfig()
	}

	init := &Initializer{
		config:    config,
		skillsDir: config.SkillsDir,
		stopCh:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(init)
	}

	return init
}

// Initialize 初始化技能发现模块
// 创建所有必要的组件并启动定时执行
// 返回:
//   - error: 错误信息
func (i *Initializer) Initialize() error {
	logger.Info("初始化技能发现模块", map[string]interface{}{})

	// 验证必要依赖
	if i.settingSvc == nil {
		return fmt.Errorf("配置服务未设置")
	}

	if i.stateRepo == nil && i.db != nil {
		i.stateRepo = skill.NewStateRepository(i.db)
	}

	if i.stateRepo == nil {
		return fmt.Errorf("Skill 状态仓库未设置")
	}

	if i.skillsDir == "" {
		return fmt.Errorf("技能目录未设置")
	}

	// 确保技能目录存在
	if err := os.MkdirAll(i.skillsDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}
	logger.Debug("技能目录已就绪", map[string]interface{}{
		"skills_dir": i.skillsDir,
	})

	// 创建重复检查器
	checker := NewDuplicateChecker(i.stateRepo, i.config.SimilarityThreshold)
	logger.Debug("创建重复检查器完成", map[string]interface{}{
		"threshold": i.config.SimilarityThreshold,
	})

	// 创建 Notifier Agent
	var notifier *agents.NotifierAgent
	if i.websocketHub != nil && i.config.EnableNotification {
		notifier = agents.NewNotifierAgent(i.websocketHub)
		logger.Debug("创建 Notifier Agent 完成", map[string]interface{}{})
	}
	dir, _ := os.Getwd()
	workspace := filepath.Join(dir, "data")
	store, _ := memory.NewHistoryStore(workspace, 100, 30)
	// entries 包含最近7天最多20条记录，按时间升序排列
	// 创建执行器
	executorCfg := &ExecutorConfig{
		Config:       i.config,
		Checker:      checker,
		Notifier:     notifier,
		ConvStore:    i.convStore,
		SettingSvc:   i.settingSvc,
		SkillsDir:    i.skillsDir,
		DB:           i.db,
		HistoryStore: store,
	}

	i.executor = NewExecutor(executorCfg)
	logger.Info("技能发现模块初始化完成", map[string]interface{}{})

	// 启动定时执行
	i.Start()

	return nil
}

// Start 启动定时执行
// 每2小时执行一次技能发现任务
func (i *Initializer) Start() {
	if i.ticker != nil {
		logger.Warn("技能发现定时任务已在运行", map[string]interface{}{})
		return
	}

	interval := i.config.Interval
	if interval <= 0 {
		interval = 2 * time.Hour
	}

	logger.Info("启动技能发现定时任务", map[string]interface{}{
		"interval": interval.String(),
	})

	// 立即执行一次
	go i.executeDiscovery()

	// 创建定时器
	i.ticker = time.NewTicker(interval)

	// 启动定时执行循环
	go i.runLoop()
}

// runLoop 定时执行循环
func (i *Initializer) runLoop() {
	for {
		select {
		case <-i.ticker.C:
			i.executeDiscovery()
		case <-i.stopCh:
			return
		}
	}
}

// executeDiscovery 执行技能发现任务
func (i *Initializer) executeDiscovery() {
	logger.Info("开始执行技能发现任务", map[string]interface{}{})

	ctx := context.Background()

	// 创建任务对象（用于兼容 Executor 接口）
	task := &DiscoveryTask{
		ID:        fmt.Sprintf("discovery-%d", time.Now().Unix()),
		Name:      "技能自动发现",
		StartTime: time.Now(),
	}

	// 执行任务
	result, err := i.executor.Execute(ctx, task)
	if err != nil {
		logger.Error("技能发现任务执行失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if result.Error != "" {
		logger.Error("技能发现任务执行出错", map[string]interface{}{
			"error": result.Error,
		})
		return
	}

	logger.Info("技能发现任务执行完成", map[string]interface{}{
		"duration_ms": result.Duration,
		"output":      result.Output,
	})
}

// Stop 停止定时执行
func (i *Initializer) Stop() {
	logger.Info("停止技能发现定时任务", map[string]interface{}{})

	if i.ticker != nil {
		i.ticker.Stop()
		i.ticker = nil
	}

	close(i.stopCh)
}

// SetWebSocketHub 设置 WebSocket Hub
// 在 initCoreServices 之后调用，用于启用通知功能
// 参数:
//   - hub: WebSocket Hub 实例
func (i *Initializer) SetWebSocketHub(hub *websocket.Hub) {
	i.websocketHub = hub

	// 如果已启用通知且 executor 已创建，重新创建 notifier
	if i.config.EnableNotification && i.executor != nil {
		// 注意：executor 的 notifier 是在创建时设置的
		// 这里需要更新 executor 的 notifier
		// 但 executor 结构中没有暴露 notifier 字段，所以需要添加一个方法来更新
		i.executor.UpdateNotifier(agents.NewNotifierAgent(hub))
		logger.Info("Notifier Agent 已更新", map[string]interface{}{})
	}
}

// createLLMFromConfig 根据配置创建 LLM 客户端
// 每次执行时调用，获取最新的配置
// 参数:
//   - settingSvc: 配置服务
//
// 返回:
//   - model.LLM: LLM 实例
//   - error: 错误信息
func createLLMFromConfig(settingSvc *setting.Service) (model.LLM, error) {
	// 获取 LLM 配置
	llmConfig, err := settingSvc.GetLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
	}

	if llmConfig == nil {
		return nil, fmt.Errorf("LLM 配置为空")
	}

	logger.Debug("获取到 LLM 配置", map[string]interface{}{
		"provider": llmConfig.Provider,
		"model":    llmConfig.Model,
	})

	// 使用 model 模块创建 LLM
	llm, err := model.NewAdapter(
		llmConfig.Provider,
		llmConfig.APIKey,
		llmConfig.Model,
		llmConfig.BaseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 适配器失败: %w", err)
	}

	if llm == nil {
		return nil, fmt.Errorf("不支持的 LLM 提供商: %s", llmConfig.Provider)
	}

	return llm, nil
}

// NewStateRepositoryFromDB 从数据库连接创建状态仓库
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - *skill.StateRepository: 状态仓库实例
func NewStateRepositoryFromDB(db *sql.DB) *skill.StateRepository {
	return skill.NewStateRepository(db)
}
