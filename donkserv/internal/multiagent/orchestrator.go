// multiagent 多Agent协作编排器
package multiagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/multiagent/agents"
	multiagentToken "github.com/longstageai/donk/donk/internal/multiagent/token"
	"github.com/longstageai/donk/donk/internal/multiagent/tools"
	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Orchestrator 多Agent任务编排器
// 负责协调6个Agent的执行流程，管理任务状态流转
type Orchestrator struct {
	// 6个Agent实例
	generationAgent *agents.GenerationAgent
	planningAgent   *agents.PlanningAgent
	planReviewAgent *agents.PlanReviewAgent
	executionAgent  *agents.ExecutionAgent
	taskReviewAgent *agents.TaskReviewAgent
	completionAgent *agents.CompletionAgent

	// 基础设施
	llm          types.LLMClient
	tokenManager *multiagentToken.Manager
	toolRegistry *tools.Registry
	logger       *logger.Logger
	hub          *websocket.Hub // WebSocket Hub，用于任务完成通知

	// 统一Token统计（可选，与token.Manager二选一）
	tokenStats *token.TokenStats

	// 用户数据存储（用于个性化）
	historyStore   *memory.HistoryStore // 历史记录存储
	profileStorage profile.Storage      // 用户画像存储

	// 配置
	coreTheme string
	config    *OrchestratorConfig

	// 控制信号
	stopChan chan struct{}
	wg       sync.WaitGroup

	// 事件回调
	onTaskStart    func(ctx *types.TaskContext)
	onTaskEnd      func(ctx *types.TaskContext)
	onTaskError    func(ctx *types.TaskContext, err error)
	onStatusChange func(ctx *types.TaskContext, oldStatus, newStatus types.TaskStatus)
}

// OrchestratorConfig 编排器配置
type OrchestratorConfig struct {
	// 审查配置
	ReviewThreshold       float64 // 审查通过分数阈值(默认8.0)
	MaxPlanReviewAttempts int     // 最大规划审查尝试次数(默认3)
	MaxTaskReviewAttempts int     // 最大任务审查尝试次数(默认3)

	// 循环配置
	LoopInterval      time.Duration // 任务间隔(默认1分钟)
	AutoStartNextTask bool          // 是否自动开始下一个任务(默认true)
}

// DefaultOrchestratorConfig 默认配置
func DefaultOrchestratorConfig() *OrchestratorConfig {
	return &OrchestratorConfig{
		ReviewThreshold:       8.0,
		MaxPlanReviewAttempts: 3,
		MaxTaskReviewAttempts: 3,
		LoopInterval:          1 * time.Minute,
		AutoStartNextTask:     true,
	}
}

// OrchestratorOption 编排器配置选项
type OrchestratorOption func(*Orchestrator)

// WithCoreTheme 设置核心主题
func WithCoreTheme(theme string) OrchestratorOption {
	return func(o *Orchestrator) {
		o.coreTheme = theme
	}
}

// WithOrchestratorConfig 设置配置
func WithOrchestratorConfig(config *OrchestratorConfig) OrchestratorOption {
	return func(o *Orchestrator) {
		o.config = config
	}
}

// WithTokenManager 设置Token管理器
func WithTokenManager(tm *multiagentToken.Manager) OrchestratorOption {
	return func(o *Orchestrator) {
		o.tokenManager = tm
	}
}

// WithTokenStats 设置统一Token统计器
func WithTokenStats(ts *token.TokenStats) OrchestratorOption {
	return func(o *Orchestrator) {
		o.tokenStats = ts
	}
}

// WithToolRegistry 设置工具注册表
func WithToolRegistry(tr *tools.Registry) OrchestratorOption {
	return func(o *Orchestrator) {
		o.toolRegistry = tr
	}
}

// WithWebSocketHub 设置WebSocket Hub
func WithWebSocketHub(hub *websocket.Hub) OrchestratorOption {
	return func(o *Orchestrator) {
		o.hub = hub
	}
}

// WithLogger 设置日志记录器
func WithLogger(log *logger.Logger) OrchestratorOption {
	return func(o *Orchestrator) {
		o.logger = log
	}
}

// WithOnTaskStart 设置任务开始回调
func WithOnTaskStart(fn func(ctx *types.TaskContext)) OrchestratorOption {
	return func(o *Orchestrator) {
		o.onTaskStart = fn
	}
}

// WithOnTaskEnd 设置任务结束回调
func WithOnTaskEnd(fn func(ctx *types.TaskContext)) OrchestratorOption {
	return func(o *Orchestrator) {
		o.onTaskEnd = fn
	}
}

// WithOnTaskError 设置任务错误回调
func WithOnTaskError(fn func(ctx *types.TaskContext, err error)) OrchestratorOption {
	return func(o *Orchestrator) {
		o.onTaskError = fn
	}
}

// WithOnStatusChange 设置状态变更回调
func WithOnStatusChange(fn func(ctx *types.TaskContext, oldStatus, newStatus types.TaskStatus)) OrchestratorOption {
	return func(o *Orchestrator) {
		o.onStatusChange = fn
	}
}

// WithHistoryStore 设置历史记录存储
func WithHistoryStore(store *memory.HistoryStore) OrchestratorOption {
	return func(o *Orchestrator) {
		o.historyStore = store
	}
}

// WithProfileStorage 设置用户画像存储
func WithProfileStorage(storage profile.Storage) OrchestratorOption {
	return func(o *Orchestrator) {
		o.profileStorage = storage
	}
}

// NewOrchestrator 创建任务编排器
func NewOrchestrator(llm types.LLMClient, opts ...OrchestratorOption) *Orchestrator {
	o := &Orchestrator{
		llm:       llm,
		coreTheme: "让用户感受到温暖",
		config:    DefaultOrchestratorConfig(),
		stopChan:  make(chan struct{}),
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(o)
	}

	// 初始化基础设施
	if o.tokenManager == nil && o.tokenStats == nil {
		o.tokenManager = multiagentToken.NewManager()
	}
	if o.toolRegistry == nil {
		o.toolRegistry = tools.CreateStandardRegistry()
	}

	// 初始化Agent
	if o.tokenStats != nil {
		// 使用统一的TokenStats
		o.generationAgent = agents.NewGenerationAgentWithStats(llm, o.tokenStats, o.logger)
		o.planningAgent = agents.NewPlanningAgentWithStats(llm, o.tokenStats, o.logger)
		o.planReviewAgent = agents.NewPlanReviewAgentWithStats(llm, o.tokenStats, o.logger)
		o.executionAgent = agents.NewExecutionAgentWithStats(llm, o.tokenStats, o.toolRegistry, o.logger)
		o.taskReviewAgent = agents.NewTaskReviewAgentWithStats(llm, o.tokenStats, o.logger)
		o.completionAgent = agents.NewCompletionAgentWithStatsAndHub(llm, o.tokenStats, o.logger, o.hub)
	} else {
		// 使用multiagent的TokenManager
		o.generationAgent = agents.NewGenerationAgent(llm, o.tokenManager, o.logger)
		o.planningAgent = agents.NewPlanningAgent(llm, o.tokenManager, o.logger)
		o.planReviewAgent = agents.NewPlanReviewAgent(llm, o.tokenManager, o.logger)
		o.executionAgent = agents.NewExecutionAgent(llm, o.tokenManager, o.toolRegistry, o.logger)
		o.taskReviewAgent = agents.NewTaskReviewAgent(llm, o.tokenManager, o.logger)
		o.completionAgent = agents.NewCompletionAgentWithHub(llm, o.tokenManager, o.logger, o.hub)
	}

	// 初始化 GenerationAgent 的工具（知识库搜索）
	settingSvc := setting.GetProvider()
	if settingSvc != nil {
		o.generationAgent.InitTools(settingSvc)
	}

	// 注入用户数据存储到 GenerationAgent（用于个性化）
	if o.historyStore != nil {
		o.generationAgent.SetHistoryStore(o.historyStore)
		logger.Debug("注入历史记录存储到 GenerationAgent", map[string]interface{}{})
	}
	if o.profileStorage != nil {
		o.generationAgent.SetProfileStorage(o.profileStorage)
		logger.Debug("注入用户画像存储到 GenerationAgent", map[string]interface{}{})
	}

	return o
}

// Start 启动编排器
// 开始持续循环执行任务
func (o *Orchestrator) Start() {
	fmt.Println("=====================================")
	fmt.Println("  多Agent协作系统启动")
	fmt.Printf("  核心主题：%s\n", o.coreTheme)
	fmt.Println("=====================================")

	o.wg.Add(1)
	go o.taskLoop()
}

// Stop 停止编排器
func (o *Orchestrator) Stop() {
	fmt.Println("\n正在停止多Agent协作系统...")
	close(o.stopChan)
	o.wg.Wait()
	fmt.Println("多Agent协作系统已停止")
}

// RunOnce 执行一次任务周期
// 用于测试或手动触发
func (o *Orchestrator) RunOnce() (*types.TaskContext, error) {
	ctx := types.NewTaskContext(o.coreTheme)
	return ctx, o.executeTaskCycle(ctx)
}

// taskLoop 任务循环
// 持续执行：生成 -> 规划 -> 规划审查 -> 执行 -> 执行审查 -> 结束 -> 再次生成
func (o *Orchestrator) taskLoop() {
	defer o.wg.Done()

	for {
		select {
		case <-o.stopChan:
			return
		default:
			// 创建新的任务上下文
			ctx := types.NewTaskContext(o.coreTheme)

			// 重置Token统计
			if o.tokenManager != nil {
				o.tokenManager.ResetTaskUsage()
			}
			if o.tokenStats != nil {
				o.tokenStats.ResetLimitExceeded()
			}

			// 执行任务周期
			if err := o.executeTaskCycle(ctx); err != nil {
				fmt.Printf("任务执行出错: %v\n", err)
				if o.onTaskError != nil {
					o.onTaskError(ctx, err)
				}
			}

			// 打印Token统计报告
			if o.tokenManager != nil {
				fmt.Println(o.tokenManager.GenerateReport())
			}

			// 检查是否自动开始下一个任务
			if !o.config.AutoStartNextTask {
				return
			}

			// 任务完成后等待一段时间再开始下一个
			fmt.Println("\n=====================================")
			fmt.Println("  任务完成，准备开始下一个温暖任务...")
			fmt.Println("=====================================")

			select {
			case <-o.stopChan:
				return
			case <-time.After(o.config.LoopInterval):
				// 继续下一个任务
			}
		}
	}
}

// executeTaskCycle 执行一个完整的任务周期
// 包含6个Agent的完整协作流程
func (o *Orchestrator) executeTaskCycle(ctx *types.TaskContext) error {
	// 触发任务开始回调
	if o.onTaskStart != nil {
		o.onTaskStart(ctx)
	}

	// 1. 任务生成
	fmt.Println("\n>>> 阶段1: 任务生成")
	if err := o.generationAgent.Process(ctx); err != nil {
		return fmt.Errorf("任务生成失败: %w", err)
	}
	fmt.Printf("生成任务: %s - %s\n", ctx.Task.Title, ctx.Task.Description)

	// 2. 任务规划
	fmt.Println("\n>>> 阶段2: 任务规划")
	if err := o.planningAgent.Process(ctx); err != nil {
		return fmt.Errorf("任务规划失败: %w", err)
	}
	fmt.Printf("规划步骤数: %d\n", len(ctx.Plan))

	// 3. 规划审查
	fmt.Println("\n>>> 阶段3: 规划审查")
	planReviewPassed := false
	for attempt := 1; attempt <= o.config.MaxPlanReviewAttempts; attempt++ {
		fmt.Printf("审查尝试 %d/%d...\n", attempt, o.config.MaxPlanReviewAttempts)
		if err := o.planReviewAgent.Process(ctx); err != nil {
			return fmt.Errorf("规划审查失败: %w", err)
		}
		if ctx.PlanReview.Passed {
			planReviewPassed = true
			fmt.Printf("规划审查通过，评分: %.1f\n", ctx.PlanReview.Score)
			break
		}
		fmt.Printf("规划审查未通过，评分: %.1f，反馈: %s\n", ctx.PlanReview.Score, ctx.PlanReview.Feedback)
	}
	if !planReviewPassed {
		fmt.Println("规划审查多次未通过，使用当前规划继续执行")
	}

	// 4. 任务执行
	fmt.Println("\n>>> 阶段4: 任务执行")
	if err := o.executionAgent.Process(ctx); err != nil {
		return fmt.Errorf("任务执行失败: %w", err)
	}
	fmt.Printf("执行任务步骤数: %d\n", len(ctx.Todos))

	// 5. 任务审查
	fmt.Println("\n>>> 阶段5: 任务审查")
	taskReviewPassed := false
	for attempt := 1; attempt <= o.config.MaxTaskReviewAttempts; attempt++ {
		fmt.Printf("审查尝试 %d/%d...\n", attempt, o.config.MaxTaskReviewAttempts)
		if err := o.taskReviewAgent.Process(ctx); err != nil {
			return fmt.Errorf("任务审查失败: %w", err)
		}
		if ctx.ExecutionReview.Passed {
			taskReviewPassed = true
			fmt.Printf("任务审查通过，评分: %.1f\n", ctx.ExecutionReview.Score)
			break
		}
		fmt.Printf("任务审查未通过，评分: %.1f，反馈: %s\n", ctx.ExecutionReview.Score, ctx.ExecutionReview.Feedback)
	}
	if !taskReviewPassed {
		fmt.Println("任务审查多次未通过，使用当前结果继续")
	}

	// 6. 任务结束
	fmt.Println("\n>>> 阶段6: 任务结束")
	if err := o.completionAgent.Process(ctx); err != nil {
		return fmt.Errorf("任务结束处理失败: %w", err)
	}
	fmt.Printf("任务完成: %s\n", ctx.Task.Title)
	fmt.Printf("祝福语: %s\n", ctx.Output.Blessing)

	// 触发任务结束回调
	if o.onTaskEnd != nil {
		o.onTaskEnd(ctx)
	}

	return nil
}

// GetTokenManager 获取Token管理器
func (o *Orchestrator) GetTokenManager() *multiagentToken.Manager {
	return o.tokenManager
}

// GetToolRegistry 获取工具注册表
func (o *Orchestrator) GetToolRegistry() *tools.Registry {
	return o.toolRegistry
}

// ExecuteTaskWithContext 使用指定的上下文执行任务
func (o *Orchestrator) ExecuteTaskWithContext(ctx context.Context) (*types.TaskContext, error) {
	taskCtx := types.NewTaskContext(o.coreTheme)
	return taskCtx, o.executeTaskCycle(taskCtx)
}

// RunWithContext 使用指定的上下文运行一次任务（别名方法）
func (o *Orchestrator) RunWithContext(ctx context.Context) (*types.TaskContext, error) {
	return o.ExecuteTaskWithContext(ctx)
}
