// multiagent 多Agent系统服务入口
// 使用项目原有的 configs 和 model 包
package multiagent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/configs"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/multiagent/tools"
	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// Service 多Agent系统服务
type Service struct {
	orchestrator *Orchestrator
	config       *configs.Conf
	llmClient    types.LLMClient
	logger       *logger.Logger
	tokenStats   *token.TokenStats
	hub          *websocket.Hub // WebSocket Hub，用于任务完成通知
}

// llmAdapter 内部LLM适配器
// 将model.LLM接口适配为types.LLMClient接口
// 支持动态获取最新配置，每次调用都重新创建LLM实例
type llmAdapter struct {
	getConfig func() (provider, model, apiKey, baseURL string, err error)
}

// Chat 发送聊天请求
// 每次调用都获取最新配置并创建LLM实例
func (a *llmAdapter) Chat(messages []types.Message, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	// 获取最新配置
	provider, modelName, apiKey, baseURL, err := a.getConfig()
	if err != nil {
		return nil, fmt.Errorf("获取LLM配置失败: %w", err)
	}

	// 创建新的LLM实例（使用最新配置）
	llm, err := model.NewAdapter(provider, apiKey, modelName, baseURL)
	if err != nil {
		return nil, fmt.Errorf("创建LLM适配器失败: %w", err)
	}
	if llm == nil {
		return nil, fmt.Errorf("不支持的LLM提供商: %s", provider)
	}

	// 转换消息格式
	schemaMessages := make([]schema.Message, len(messages))
	for i, msg := range messages {
		schemaMessages[i] = schema.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCalls:  convertToolCallsToSchema(msg.ToolCalls),
			ToolCallID: msg.ToolCallID,
		}
	}

	// 转换工具定义
	var schemaTools []schema.ToolDefinition
	if len(tools) > 0 {
		schemaTools = make([]schema.ToolDefinition, len(tools))
		for i, tool := range tools {
			schemaTools[i] = schema.ToolDefinition{
				Type: tool.Type,
				Function: schema.FunctionProperty{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}

	// 调用项目model的Chat方法
	req := &schema.ChatRequest{
		Messages: schemaMessages,
		Tools:    schemaTools,
	}

	resp, err := llm.Chat(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 转换响应格式
	return &types.LLMResponse{
		Content:   resp.Content,
		Reasoning: "",
		ToolCalls: convertToolCallsFromSchema(resp.ToolCalls),
		Usage: types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// ChatStream 流式聊天请求
// 每次调用都获取最新配置并创建LLM实例
func (a *llmAdapter) ChatStream(messages []types.Message, tools []types.ToolDefinition, callback types.StreamCallback) error {
	// 获取最新配置
	provider, apiKey, modelName, baseURL, err := a.getConfig()
	if err != nil {
		return fmt.Errorf("获取LLM配置失败: %w", err)
	}

	// 创建新的LLM实例（使用最新配置）
	llm, err := model.NewAdapter(provider, apiKey, modelName, baseURL)
	if err != nil {
		return fmt.Errorf("创建LLM适配器失败: %w", err)
	}
	if llm == nil {
		return fmt.Errorf("不支持的LLM提供商: %s", provider)
	}

	// 转换消息格式
	schemaMessages := make([]schema.Message, len(messages))
	for i, msg := range messages {
		schemaMessages[i] = schema.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCalls:  convertToolCallsToSchema(msg.ToolCalls),
			ToolCallID: msg.ToolCallID,
		}
	}

	// 转换工具定义
	var schemaTools []schema.ToolDefinition
	if len(tools) > 0 {
		schemaTools = make([]schema.ToolDefinition, len(tools))
		for i, tool := range tools {
			schemaTools[i] = schema.ToolDefinition{
				Type: tool.Type,
				Function: schema.FunctionProperty{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}

	// 调用项目model的StreamChat方法
	req := &schema.ChatRequest{
		Messages: schemaMessages,
		Tools:    schemaTools,
	}

	streamResp, err := llm.StreamChat(context.Background(), req)
	if err != nil {
		return fmt.Errorf("LLM流式调用失败: %w", err)
	}

	// 读取流式响应
	go func() {
		for chunk := range streamResp.Chunks {
			if callback != nil {
				callback(&types.StreamChunk{
					Content:   chunk.Choices[0].Delta.Content,
					Reasoning: "",
					Done:      chunk.Choices[0].FinishReason != "",
				})
			}
		}
	}()

	// 等待完成
	<-streamResp.Done

	return streamResp.Error
}

// convertToolCallsToSchema 将types.ToolCall转换为schema.ToolCall
func convertToolCallsToSchema(toolCalls []types.ToolCall) []schema.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]schema.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = schema.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: schema.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

// convertToolCallsFromSchema 将schema.ToolCall转换为types.ToolCall
func convertToolCallsFromSchema(toolCalls []schema.ToolCall) []types.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]types.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = types.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

// NewService 创建多Agent服务
// 使用项目原有的 configs.Conf 配置
func NewService(conf *configs.Conf, coreTheme string, log *logger.Logger) (*Service, error) {
	return NewServiceWithTokenStatsAndHub(conf, coreTheme, log, nil, nil, nil, "")
}

// NewServiceWithHub 创建多Agent服务（带WebSocket Hub）
func NewServiceWithHub(conf *configs.Conf, coreTheme string, log *logger.Logger, hub *websocket.Hub) (*Service, error) {
	return NewServiceWithTokenStatsAndHub(conf, coreTheme, log, nil, hub, nil, "")
}

// NewServiceWithTokenStats 创建多Agent服务（带Token统计）
// 如果 tokenStats 不为 nil，则使用传入的共享 TokenStats，实现单/多Agent共享预算
// LLM配置每次调用时动态获取，支持运行时配置变更
func NewServiceWithTokenStats(conf *configs.Conf, coreTheme string, log *logger.Logger, tokenStats *token.TokenStats, hub *websocket.Hub, db *sql.DB, dataDir string) (*Service, error) {
	return NewServiceWithTokenStatsAndHub(conf, coreTheme, log, tokenStats, hub, db, dataDir)
}

// NewServiceWithTokenStatsAndHub 创建多Agent服务（带Token统计和WebSocket Hub）
// 如果 tokenStats 不为 nil，则使用传入的共享 TokenStats，实现单/多Agent共享预算
// 如果 hub 不为 nil，则使用传入的 WebSocket Hub 发送任务完成通知
// 如果 db 不为 nil，则创建 profileStorage 用于个性化
// 如果 dataDir 不为空，则创建 historyStore 用于个性化
// LLM配置每次调用时动态获取，支持运行时配置变更
func NewServiceWithTokenStatsAndHub(conf *configs.Conf, coreTheme string, log *logger.Logger, tokenStats *token.TokenStats, hub *websocket.Hub, db *sql.DB, dataDir string) (*Service, error) {
	if conf == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 创建动态配置获取函数
	getLLMConfig := func() (string, string, string, string, error) {
		// 优先从 setting 模块获取最新配置
		provider := setting.GetProvider()
		if provider != nil {
			return provider.LLMProvider()
		}
		// 降级到配置文件
		return conf.Llm.Provider, conf.Llm.Model, conf.Llm.APIKey, conf.Llm.BaseURL, nil
	}

	// 使用内部适配器将 model.LLM 转换为 types.LLMClient
	// 适配器支持动态获取最新配置
	llmClient := &llmAdapter{getConfig: getLLMConfig}

	// 创建编排器配置
	orchConfig := &OrchestratorConfig{
		ReviewThreshold:       8.0,
		MaxPlanReviewAttempts: 3,
		MaxTaskReviewAttempts: 3,
		LoopInterval:          2 * time.Hour,
		AutoStartNextTask:     true,
	}

	// 创建用户数据存储（用于个性化）
	var historyStore *memory.HistoryStore
	var profileStorage profile.Storage

	// 创建历史记录存储（使用 dataDir）
	if dataDir != "" {
		hs, err := memory.NewHistoryStore(dataDir, 100, 30)
		if err != nil {
			log.Warn("创建历史记录存储失败", map[string]interface{}{"error": err.Error()})
		} else {
			historyStore = hs
			log.Debug("HistoryStore 创建成功", map[string]interface{}{})
		}
	}

	// 创建用户画像存储（使用 db）
	if db != nil {
		profileStorage = profile.NewDBStorage(db)
		log.Debug("ProfileStorage 创建成功", map[string]interface{}{})
	}

	// 创建编排器
	orchestrator := NewOrchestrator(
		llmClient,
		WithCoreTheme(coreTheme),
		WithOrchestratorConfig(orchConfig),
		WithLogger(log),
		WithTokenStats(tokenStats),
		WithWebSocketHub(hub),
		WithHistoryStore(historyStore),
		WithProfileStorage(profileStorage),
	)

	return &Service{
		orchestrator: orchestrator,
		config:       conf,
		llmClient:    llmClient,
		logger:       log,
		tokenStats:   tokenStats,
		hub:          hub,
	}, nil
}

// Start 启动服务
func (s *Service) Start() {
	s.orchestrator.Start()
}

// Stop 停止服务
func (s *Service) Stop() {
	s.orchestrator.Stop()
}

// RunOnce 执行一次任务
func (s *Service) RunOnce() (*types.TaskContext, error) {
	return s.orchestrator.RunOnce()
}

// RunWithContext 使用指定context运行一次任务
func (s *Service) RunWithContext(ctx context.Context) (*types.TaskContext, error) {
	return s.orchestrator.RunWithContext(ctx)
}

// GetOrchestrator 获取编排器
func (s *Service) GetOrchestrator() *Orchestrator {
	return s.orchestrator
}

// GetConfig 获取配置
func (s *Service) GetConfig() *configs.Conf {
	return s.config
}

// GetHub 获取 WebSocket Hub
func (s *Service) GetHub() *websocket.Hub {
	return s.hub
}

// RegisterTool 注册工具
func (s *Service) RegisterTool(name, description string, parameters map[string]interface{}, handler tools.Handler) error {
	return s.orchestrator.GetToolRegistry().Register(name, description, parameters, handler)
}

// SetOnTaskStart 设置任务开始回调
func (s *Service) SetOnTaskStart(fn func(ctx *types.TaskContext)) {
	// 重新创建编排器以应用回调
	// 注意：这需要在Start之前调用
}

// SetOnTaskEnd 设置任务结束回调
func (s *Service) SetOnTaskEnd(fn func(ctx *types.TaskContext)) {
	// 重新创建编排器以应用回调
}

// SetOnTaskError 设置任务错误回调
func (s *Service) SetOnTaskError(fn func(ctx *types.TaskContext, err error)) {
	// 重新创建编排器以应用回调
}

// SetOnStatusChange 设置状态变更回调
func (s *Service) SetOnStatusChange(fn func(ctx *types.TaskContext, oldStatus, newStatus types.TaskStatus)) {
	// 重新创建编排器以应用回调
}

// GetTokenUsage 获取Token使用统计
func (s *Service) GetTokenUsage() *types.TaskTokenUsage {
	usage := s.orchestrator.GetTokenManager().GetTaskUsage()
	return usage
}

// GetTokenReport 获取Token使用报告
func (s *Service) GetTokenReport() string {
	if s.tokenStats != nil {
		usage := s.tokenStats.GetTodayUsage()
		limit := s.tokenStats.GetDailyLimit()
		remaining := s.tokenStats.GetRemainingBudget()
		return fmt.Sprintf(`
========================================
Token消耗统计报告
========================================
今日累计使用: %d tokens
每日限额: %d tokens
剩余可用: %d tokens
========================================
`, usage, limit, remaining)
	}
	return s.orchestrator.GetTokenManager().GenerateReport()
}

// IsTokenBudgetExceeded 检查Token预算是否已超出
func (s *Service) IsTokenBudgetExceeded() bool {
	if s.tokenStats != nil {
		return s.tokenStats.IsBudgetExceeded()
	}
	return s.orchestrator.GetTokenManager().IsLimitExceeded()
}
