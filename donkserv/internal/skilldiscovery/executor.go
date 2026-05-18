// skilldiscovery 技能自动发现模块
// Executor 执行技能发现任务
package skilldiscovery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/longstageai/donk/donk/internal/conversation"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/skilldiscovery/agents"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Executor 技能发现执行器
// 用于执行定时技能发现任务
type Executor struct {
	config         *Config
	checker        *DuplicateChecker
	notifier       *agents.NotifierAgent
	convStore      *conversation.Store
	profileStorage profile.Storage
	settingSvc     *setting.ConfigProvider
	skillsDir      string
	tokenStats     *token.TokenStats
	userID         string
	historyStore   *memory.HistoryStore // 历史记录存储
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	Config         *Config
	Checker        *DuplicateChecker
	Notifier       *agents.NotifierAgent
	ConvStore      *conversation.Store
	ProfileStorage profile.Storage
	SettingSvc     *setting.ConfigProvider
	SkillsDir      string
	DB             *sql.DB
	UserID         string
	HistoryStore   *memory.HistoryStore // 历史记录存储
}

// NewExecutor 创建技能发现执行器
// 参数:
//   - cfg: 执行器配置
//
// 返回:
//   - *Executor: 执行器实例
func NewExecutor(cfg *ExecutorConfig) *Executor {
	if cfg.Config == nil {
		cfg.Config = DefaultConfig()
	}

	executor := &Executor{
		config:         cfg.Config,
		checker:        cfg.Checker,
		notifier:       cfg.Notifier,
		convStore:      cfg.ConvStore,
		profileStorage: cfg.ProfileStorage,
		settingSvc:     cfg.SettingSvc,
		skillsDir:      cfg.SkillsDir,
		userID:         cfg.UserID,
		historyStore:   cfg.HistoryStore,
	}

	// 设置默认 userID
	if executor.userID == "" {
		executor.userID = "default"
		logger.Debug("UserID 使用默认值 'default'", map[string]interface{}{})
	}

	// 如果没有提供 profileStorage 但有 DB，创建默认的 profileStorage
	if executor.profileStorage == nil && cfg.DB != nil {
		executor.profileStorage = profile.NewDBStorage(cfg.DB)
		logger.Debug("ProfileStorage 初始化成功", map[string]interface{}{})
	}

	// 初始化 Token 统计器
	if cfg.DB != nil {
		if stats, err := token.NewTokenStats(cfg.DB); err == nil {
			executor.tokenStats = stats
			logger.Debug("Token 统计器初始化成功", map[string]interface{}{})
		} else {
			logger.Error("Token 统计器初始化失败", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return executor
}

// Execute 执行技能发现任务
// 参数:
//   - ctx: 上下文
//   - task: 发现任务
//
// 返回:
//   - *TaskResult: 任务执行结果
//   - error: 错误信息
func (e *Executor) Execute(ctx context.Context, task *DiscoveryTask) (*TaskResult, error) {
	startTime := time.Now()
	taskID := uuid.New().String()

	logger.Info("开始执行技能发现任务", map[string]interface{}{
		"task_id":   taskID,
		"task_name": task.Name,
	})

	// 检查 Token 预算是否已超出
	if e.tokenStats != nil && e.tokenStats.IsBudgetExceeded() {
		logger.Warn("Token 预算已超出，跳过技能发现任务", map[string]interface{}{
			"task_id": taskID,
		})
		return e.buildErrorResult(startTime, fmt.Errorf("Token 预算已超出，请明日再试")), nil
	}

	// 检查剩余预算
	if e.tokenStats != nil {
		hasBudget, remaining := e.tokenStats.CheckBudget()
		if !hasBudget {
			logger.Warn("Token 预算不足，跳过技能发现任务", map[string]interface{}{
				"task_id":   taskID,
				"remaining": remaining,
			})
			return e.buildErrorResult(startTime, fmt.Errorf("Token 预算不足，剩余: %d", remaining)), nil
		}
		logger.Debug("Token 预算检查通过", map[string]interface{}{
			"task_id":   taskID,
			"remaining": remaining,
		})
	}

	// 每次执行时从 setting 获取最新配置创建 LLM
	llm, err := e.createLLMFromSetting()
	if err != nil {
		logger.Error("创建 LLM 失败", map[string]interface{}{
			"task_id": taskID,
			"error":   err.Error(),
		})
		return e.buildErrorResult(startTime, err), nil
	}

	logger.Debug("LLM 创建成功", map[string]interface{}{
		"task_id":  taskID,
		"llm_name": llm.Name(),
	})

	// 获取最近的对话
	conversations, err := e.getRecentConversations(ctx)
	if err != nil {
		logger.Error("获取对话历史失败", map[string]interface{}{
			"task_id": taskID,
			"error":   err.Error(),
		})
		return e.buildErrorResult(startTime, err), nil
	}

	logger.Info("获取对话历史完成", map[string]interface{}{
		"task_id":            taskID,
		"conversation_count": len(conversations),
	})

	// 创建 Analyzer Agent（使用最新配置）
	// 传入 setting 服务，analyzer 内部会自动创建 embedder 和注册工具
	analyzer := agents.NewAnalyzerAgent(llm, e.settingSvc, "")

	// 分析对话
	candidates, err := e.analyzeConversations(ctx, analyzer, conversations)
	if err != nil {
		logger.Error("分析对话失败", map[string]interface{}{
			"task_id": taskID,
			"error":   err.Error(),
		})
		return e.buildErrorResult(startTime, err), nil
	}

	// 如果没有发现需求，生成创意技能
	source := "analyzer"
	if len(candidates) == 0 {
		logger.Info("未发现技能需求，生成创意技能", map[string]interface{}{
			"task_id": taskID,
		})
		// 创建 Creative Agent（使用最新配置）
		creativeAgent := agents.NewCreativeAgent(llm, "")
		candidates = creativeAgent.Generate(ctx, e.config.CreativeSkillsCount)
		source = "creative"
	}

	logger.Info("技能候选列表", map[string]interface{}{
		"task_id":         taskID,
		"candidate_count": len(candidates),
		"source":          source,
	})

	// 创建 Planner 和 Creator Agents（使用最新配置）
	planner := agents.NewPlannerAgent(llm, "")
	creator := agents.NewCreatorAgent(e.skillsDir)

	// 处理候选技能
	result := e.processCandidates(ctx, taskID, candidates, planner, creator)

	// 发送完成通知
	if e.config.EnableNotification && e.notifier != nil {
		e.notifier.NotifyDiscoveryCompleted(
			taskID,
			len(result.CreatedSkills),
			len(result.SkippedSkills),
			result.CreatedSkills,
		)
	}

	// 构建任务结果
	return e.buildSuccessResult(startTime, result), nil
}

// createLLMFromSetting 从 setting 获取配置创建 LLM
// 每次执行时调用，确保使用最新的配置
// 返回:
//   - model.LLM: LLM 实例
//   - error: 错误信息
func (e *Executor) createLLMFromSetting() (model.LLM, error) {
	if e.settingSvc == nil {
		return nil, fmt.Errorf("配置服务未设置")
	}

	// 获取 LLM 配置
	llmConfig, err := e.settingSvc.GetLLMConfig()
	if err != nil {
		return nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
	}

	if llmConfig == nil {
		return nil, fmt.Errorf("LLM 配置为空")
	}

	logger.Debug("从 setting 获取到 LLM 配置", map[string]interface{}{
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

// getRecentConversations 获取最近的对话或用户画像历史
// 同时返回用户画像和最近7天的对话历史
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - []agents.Conversation: 对话列表
//   - error: 错误信息
func (e *Executor) getRecentConversations(ctx context.Context) ([]agents.Conversation, error) {
	var conversations []agents.Conversation

	// 1. 获取用户画像
	if e.profileStorage != nil && e.userID != "" {
		logger.Debug("获取用户画像标签", map[string]interface{}{
			"user_id": e.userID,
		})

		profile, err := e.profileStorage.Load(ctx, e.userID)
		if err != nil {
			logger.Error("获取用户画像失败", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			profilePrompt := profile.ToPrompt()
			if profilePrompt != "" {
				// 转换用户画像为 Conversation
				conversations = append(conversations, agents.Conversation{
					ID:        "profile_tags",
					Content:   profilePrompt,
					Timestamp: profile.UpdatedAt,
				})

				logger.Info("获取用户画像标签完成", map[string]interface{}{
					"tag_count":  len(profile.Tags),
					"pref_count": len(profile.Preferences),
				})
			}
		}
	}

	// 2. 获取最近7天的对话历史
	historyConversations, err := e.getConversationsFromHistoryStore(ctx)
	if err != nil {
		logger.Error("获取历史对话失败", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		conversations = append(conversations, historyConversations...)
	}

	return conversations, nil
}

// getConversationsFromHistoryStore 从历史记录存储获取最近7天的对话
// 使用 LoadRecent7Days 方法，最多返回20条
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - []agents.Conversation: 对话列表
//   - error: 错误信息
func (e *Executor) getConversationsFromHistoryStore(ctx context.Context) ([]agents.Conversation, error) {
	if e.historyStore == nil {
		logger.Debug("历史记录存储未配置，返回空列表", map[string]interface{}{})
		return []agents.Conversation{}, nil
	}

	logger.Debug("获取最近7天对话历史", map[string]interface{}{})

	// 使用 LoadRecent7Days 获取最近7天的历史记录
	entries, err := e.historyStore.LoadRecent7Days()
	if err != nil {
		return nil, err
	}

	// 转换为 Conversation 列表
	conversations := make([]agents.Conversation, 0, len(entries))
	for _, entry := range entries {
		conversations = append(conversations, agents.Conversation{
			ID:        entry.Key,
			Content:   fmt.Sprintf("[%s] %s", entry.Role, entry.Content),
			Timestamp: entry.Timestamp,
		})
	}

	logger.Info("获取历史对话完成", map[string]interface{}{
		"count": len(conversations),
	})

	return conversations, nil
}

// getConversationsFromStore 从对话存储获取最近的对话
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - []agents.Conversation: 对话列表
//   - error: 错误信息
func (e *Executor) getConversationsFromStore(ctx context.Context) ([]agents.Conversation, error) {
	// 计算回溯时间
	since := time.Now().Add(-e.config.ConversationLookback)

	logger.Debug("获取对话历史", map[string]interface{}{
		"since":    since.Format("2006-01-02 15:04:05"),
		"lookback": e.config.ConversationLookback.String(),
	})

	// 如果没有配置对话存储，返回空列表
	if e.convStore == nil {
		logger.Debug("对话存储未配置，返回空列表", map[string]interface{}{})
		return []agents.Conversation{}, nil
	}

	// 使用 Search 方法获取最近的对话
	// 使用空字符串作为查询词，配合时间过滤器获取所有对话
	timeFilter := &conversation.TimeFilter{
		StartTime: &since,
		EndTime:   nil, // 不限制结束时间
	}

	results, err := e.convStore.Search(ctx, "", 100, timeFilter)
	if err != nil {
		logger.Error("搜索对话历史失败", map[string]interface{}{
			"error": err.Error(),
		})
		return []agents.Conversation{}, nil
	}

	// 转换搜索结果为 Conversation 列表
	conversations := make([]agents.Conversation, 0, len(results))
	for _, result := range results {
		conversations = append(conversations, agents.Conversation{
			ID:        result.ConversationID,
			Content:   result.Content,
			Timestamp: result.StartTime,
		})
	}

	logger.Info("获取对话历史完成", map[string]interface{}{
		"count": len(conversations),
	})

	return conversations, nil
}

// analyzeConversations 分析对话
// 参数:
//   - ctx: 上下文
//   - analyzer: 分析 Agent
//   - conversations: 对话列表
//
// 返回:
//   - []*SkillCandidate: 候选技能列表
//   - error: 错误信息
func (e *Executor) analyzeConversations(
	ctx context.Context,
	analyzer *agents.AnalyzerAgent,
	conversations []agents.Conversation,
) ([]*SkillCandidate, error) {
	if analyzer == nil {
		logger.Warn("Analyzer Agent 未初始化", map[string]interface{}{})
		return []*SkillCandidate{}, nil
	}

	candidates, usage, err := analyzer.AnalyzeWithUsage(ctx, conversations)
	if err != nil {
		return nil, err
	}

	// 记录 Token 使用
	if e.tokenStats != nil && usage != nil {
		if err := e.tokenStats.Record(usage.PromptTokens, usage.CompletionTokens); err != nil {
			logger.Error("记录 Token 使用失败", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			logger.Debug("记录 Token 使用成功", map[string]interface{}{
				"prompt_tokens":     usage.PromptTokens,
				"completion_tokens": usage.CompletionTokens,
			})
		}
	}

	return candidates, nil
}

// processCandidates 处理候选技能
// 参数:
//   - ctx: 上下文
//   - taskID: 任务ID
//   - candidates: 候选技能列表
//   - planner: 规划 Agent
//   - creator: 创建 Agent
//
// 返回:
//   - *DiscoveryResult: 发现结果
func (e *Executor) processCandidates(
	ctx context.Context,
	taskID string,
	candidates []*SkillCandidate,
	planner *agents.PlannerAgent,
	creator *agents.CreatorAgent,
) *DiscoveryResult {
	result := &DiscoveryResult{
		TaskID:        taskID,
		StartTime:     time.Now(),
		CreatedSkills: []string{},
		SkippedSkills: []SkippedSkillInfo{},
		Errors:        []string{},
	}

	// 限制处理数量
	maxCount := e.config.MaxSkillsPerRun
	if len(candidates) > maxCount {
		logger.Info("候选技能数量超过限制，截断处理", map[string]interface{}{
			"task_id":       taskID,
			"total":         len(candidates),
			"process_count": maxCount,
		})
		candidates = candidates[:maxCount]
	}

	// 处理每个候选
	for _, candidate := range candidates {
		logger.Info("处理候选技能", map[string]interface{}{
			"task_id":        taskID,
			"candidate_name": candidate.Name,
		})

		// 1. 检查重复
		if e.checker != nil {
			dupResult, err := e.checker.CheckDuplicate(ctx, candidate)
			if err != nil {
				logger.Error("重复检查失败", map[string]interface{}{
					"task_id":        taskID,
					"candidate_name": candidate.Name,
					"error":          err.Error(),
				})
				result.Errors = append(result.Errors,
					fmt.Sprintf("检查 %s 重复失败: %v", candidate.Name, err))
				continue
			}

			if dupResult.IsDuplicate {
				logger.Info("跳过重复技能", map[string]interface{}{
					"task_id":        taskID,
					"candidate_name": candidate.Name,
					"reason":         dupResult.Reason,
				})
				result.SkippedSkills = append(result.SkippedSkills, SkippedSkillInfo{
					Name:   candidate.Name,
					Reason: dupResult.Reason,
				})
				continue
			}
		}

		// 2. 规划技能
		plan, usage, err := planner.PlanWithUsage(ctx, candidate)
		if err != nil {
			logger.Error("技能规划失败", map[string]interface{}{
				"task_id":        taskID,
				"candidate_name": candidate.Name,
				"error":          err.Error(),
			})
			result.Errors = append(result.Errors,
				fmt.Sprintf("规划 %s 失败: %v", candidate.Name, err))
			continue
		}

		// 记录 Planner Token 使用
		if e.tokenStats != nil && usage != nil {
			if err := e.tokenStats.Record(usage.PromptTokens, usage.CompletionTokens); err != nil {
				logger.Error("记录 Planner Token 使用失败", map[string]interface{}{
					"error": err.Error(),
				})
			} else {
				logger.Debug("记录 Planner Token 使用成功", map[string]interface{}{
					"prompt_tokens":     usage.PromptTokens,
					"completion_tokens": usage.CompletionTokens,
				})
			}
		}

		// 3. 创建技能
		s, err := creator.Create(ctx, plan)
		if err != nil {
			logger.Error("技能创建失败", map[string]interface{}{
				"task_id":        taskID,
				"candidate_name": candidate.Name,
				"error":          err.Error(),
			})
			result.Errors = append(result.Errors,
				fmt.Sprintf("创建 %s 失败: %v", candidate.Name, err))
			continue
		}

		// 4. 记录成功
		logger.Info("技能创建成功", map[string]interface{}{
			"task_id":    taskID,
			"skill_name": s.Name(),
		})
		result.CreatedSkills = append(result.CreatedSkills, s.Name())

		// 5. 推送通知
		if e.config.EnableNotification && e.notifier != nil {
			source := "analyzer"
			if candidate.Evidence != nil && len(candidate.Evidence) > 0 &&
				candidate.Evidence[0] == "创意生成" {
				source = "creative"
			}
			e.notifier.NotifySkillCreated(s, source)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).Milliseconds()

	logger.Info("技能发现任务处理完成", map[string]interface{}{
		"task_id":       taskID,
		"created_count": len(result.CreatedSkills),
		"skipped_count": len(result.SkippedSkills),
		"error_count":   len(result.Errors),
		"duration_ms":   result.Duration,
	})

	return result
}

// buildSuccessResult 构建成功结果
// 参数:
//   - startTime: 开始时间
//   - result: 发现结果
//
// 返回:
//   - *TaskResult: 任务结果
func (e *Executor) buildSuccessResult(startTime time.Time, result *DiscoveryResult) *TaskResult {
	output := map[string]interface{}{
		"task_id":        result.TaskID,
		"created_skills": result.CreatedSkills,
		"skipped_skills": result.SkippedSkills,
		"created_count":  len(result.CreatedSkills),
		"skipped_count":  len(result.SkippedSkills),
		"duration_ms":    result.Duration,
	}

	if len(result.Errors) > 0 {
		output["errors"] = result.Errors
	}

	outputJSON, _ := json.Marshal(output)

	return &TaskResult{
		Output:   string(outputJSON),
		ExitCode: 0,
		Duration: time.Since(startTime).Milliseconds(),
		DoneAt:   time.Now().Unix(),
	}
}

// UpdateNotifier 更新 Notifier Agent
// 用于在初始化后设置 WebSocket Hub 以启用通知功能
// 参数:
//   - notifier: NotifierAgent 实例
func (e *Executor) UpdateNotifier(notifier *agents.NotifierAgent) {
	e.notifier = notifier
}

// buildErrorResult 构建错误结果
// 参数:
//   - startTime: 开始时间
//   - err: 错误信息
//
// 返回:
//   - *TaskResult: 任务结果
func (e *Executor) buildErrorResult(startTime time.Time, err error) *TaskResult {
	return &TaskResult{
		Output:   "",
		Error:    err.Error(),
		ExitCode: 1,
		Duration: time.Since(startTime).Milliseconds(),
		DoneAt:   time.Now().Unix(),
	}
}
