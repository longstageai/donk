// agents 任务生成Agent
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/multiagent/prompts"
	multiagentToken "github.com/longstageai/donk/donk/internal/multiagent/token"
	"github.com/longstageai/donk/donk/internal/multiagent/tools"
	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// GenerationAgent 任务生成Agent
// 负责发现让用户感受到温暖的机会，生成任务
// 支持使用知识库搜索工具获取用户信息
type GenerationAgent struct {
	llm            types.LLMClient
	tokenManager   *multiagentToken.Manager
	tokenStats     *token.TokenStats
	name           string
	description    string
	agentLogger    *AgentLogger
	toolRegistry   *tools.Registry
	maxSteps       int
	historyStore   *memory.HistoryStore // 历史记录存储（依赖注入）
	profileStorage profile.Storage      // 用户画像存储（依赖注入）
}

// NewGenerationAgent 创建任务生成Agent（使用token.Manager）
func NewGenerationAgent(llm types.LLMClient, tokenManager *multiagentToken.Manager, log *logger.Logger) *GenerationAgent {
	return &GenerationAgent{
		llm:          llm,
		tokenManager: tokenManager,
		name:         "GenerationAgent",
		description:  "任务生成Agent - 发现温暖契机，生成任务",
		agentLogger:  NewAgentLogger(log),
		maxSteps:     3,
	}
}

// NewGenerationAgentWithStats 创建任务生成Agent（使用统一token.TokenStats）
func NewGenerationAgentWithStats(llm types.LLMClient, tokenStats *token.TokenStats, log *logger.Logger) *GenerationAgent {
	return &GenerationAgent{
		llm:         llm,
		tokenStats:  tokenStats,
		name:        "GenerationAgent",
		agentLogger: NewAgentLogger(log),
		maxSteps:    3,
	}
}

// SetHistoryStore 设置历史记录存储
// 通过依赖注入传入
func (a *GenerationAgent) SetHistoryStore(store *memory.HistoryStore) {
	a.historyStore = store
}

// SetProfileStorage 设置用户画像存储
// 通过依赖注入传入
func (a *GenerationAgent) SetProfileStorage(storage profile.Storage) {
	a.profileStorage = storage
}

// InitTools 初始化工具注册表
// 传入 setting 服务以创建 knowledge_search 工具
func (a *GenerationAgent) InitTools(settingSvc *setting.ConfigProvider) {
	a.toolRegistry = tools.NewRegistry()

	// 创建并注册 knowledge_search 工具
	if err := a.registerKnowledgeSearchTool(settingSvc); err != nil {
		logger.Debug("知识库搜索工具未启用", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// registerKnowledgeSearchTool 注册知识库搜索工具
func (a *GenerationAgent) registerKnowledgeSearchTool(settingSvc *setting.ConfigProvider) error {
	if settingSvc == nil {
		return fmt.Errorf("配置服务未设置")
	}

	// 获取 embedding 配置
	embedConfig, err := settingSvc.GetEmbeddingConfig()
	if err != nil {
		return fmt.Errorf("获取 embedding 配置失败: %w", err)
	}

	if embedConfig == nil || embedConfig.Provider == "" || embedConfig.Model == "" || embedConfig.APIKey == "" {
		return fmt.Errorf("embedding 配置不完整")
	}

	// 创建 embedder
	embedder, err := embedding.NewEmbedding(
		embedConfig.Provider,
		embedConfig.APIKey,
		embedConfig.Model,
		embedConfig.BaseURL,
	)
	if err != nil {
		return fmt.Errorf("创建 embedder 失败: %w", err)
	}

	// 设置知识库路径
	dataDir := filepath.Join(".", "data", "knowledge")
	dbPath := filepath.Join(dataDir, "meta.db")

	// 创建知识库搜索工具
	knowledgeTool := builtin.NewKnowledgeSearcher(dataDir, dbPath, embedder)

	// 定义工具参数
	parameters := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "搜索查询语句（支持语义理解）",
			},
			"keywords": map[string]interface{}{
				"type":        "string",
				"description": "关键词过滤（多个用逗号分隔）",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "返回结果数量限制，默认5，最大20",
				"default":     5,
			},
		},
		"required": []string{"query"},
	}

	// 注册工具
	err = a.toolRegistry.Register(
		knowledgeTool.Name(),
		knowledgeTool.Description(),
		parameters,
		func(params map[string]interface{}) (map[string]interface{}, error) {
			// 执行工具
			toolCtx := tool.NewContext(knowledgeTool.Name(), params)
			result, err := knowledgeTool.Execute(toolCtx)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"result": result.String()}, nil
		},
	)

	if err != nil {
		return fmt.Errorf("注册工具失败: %w", err)
	}

	logger.Info("知识库搜索工具注册成功", map[string]interface{}{
		"dataDir": dataDir,
		"dbPath":  dbPath,
	})

	return nil
}

// GetName 获取Agent名称
func (a *GenerationAgent) GetName() string {
	return a.name
}

// GetDescription 获取Agent描述
func (a *GenerationAgent) GetDescription() string {
	return a.description
}

// Process 处理任务生成
// 支持 ReAct 循环，可使用工具获取额外信息
// 会结合用户画像和对话历史生成个性化任务
func (a *GenerationAgent) Process(ctx *types.TaskContext) error {
	// 加载用户画像和对话历史
	a.loadUserData(ctx)

	// 构建提示词
	config := prompts.NewConfig(ctx.CoreTheme)
	systemPrompt := prompts.GetGenerationAgentPrompt(config)

	// 构建用户内容（包含用户画像和对话历史）
	userContent := a.buildUserContent(ctx, config.CurrentTime)

	// 构建消息
	messages := []types.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	// 获取工具定义
	var toolDefs []types.ToolDefinition
	if a.toolRegistry != nil {
		toolDefs = a.getToolDefinitions()
	}

	// 打印LLM输入参数
	a.agentLogger.LogLLMInput("GenerationAgent", messages, toolDefs)

	// 使用 ReAct 循环调用LLM
	resp, err := a.processWithReAct(messages, toolDefs)
	if err != nil {
		return fmt.Errorf("LLM调用失败: %w", err)
	}

	// 打印LLM输出参数
	a.agentLogger.LogLLMOutput("GenerationAgent", resp)

	// 记录Token使用
	if a.tokenStats != nil {
		a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "generation")
		if a.tokenStats.IsBudgetExceeded() {
			return fmt.Errorf("Token预算已超出限额")
		}
	} else if a.tokenManager != nil {
		a.tokenManager.RecordUsage("generation", types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		})
	}

	// 解析响应
	var result struct {
		Insight      string          `json:"insight"`
		Task         *types.TaskInfo `json:"task"`
		CoreElements []string        `json:"coreElements"`
	}

	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// 尝试从内容中提取JSON
		content := extractJSON(resp.Content)
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return fmt.Errorf("解析任务生成结果失败: %w", err)
		}
	}

	// 填充任务上下文
	ctx.Task = result.Task
	ctx.Task.CoreElements = result.CoreElements
	ctx.UpdateStatus(types.StatusCreated)

	return nil
}

// getToolDefinitions 获取工具定义列表
func (a *GenerationAgent) getToolDefinitions() []types.ToolDefinition {
	if a.toolRegistry == nil {
		return nil
	}

	definitions := a.toolRegistry.GetAllDefinitions()
	result := make([]types.ToolDefinition, 0, len(definitions))

	for _, def := range definitions {
		result = append(result, types.ToolDefinition{
			Type: def.Type,
			Function: types.FunctionInfo{
				Name:        def.Function.Name,
				Description: def.Function.Description,
				Parameters:  def.Function.Parameters,
			},
		})
	}

	return result
}

// processWithReAct 使用 ReAct 循环处理
func (a *GenerationAgent) processWithReAct(messages []types.Message, toolDefs []types.ToolDefinition) (*types.LLMResponse, error) {
	// 如果没有工具，直接调用一次
	if len(toolDefs) == 0 {
		return a.llm.Chat(messages, nil)
	}

	// ReAct 循环
	for step := 0; step < a.maxSteps; step++ {
		logger.Debug("GenerationAgent ReAct 循环", map[string]interface{}{
			"step":     step + 1,
			"maxSteps": a.maxSteps,
		})

		// 调用LLM
		resp, err := a.llm.Chat(messages, toolDefs)
		if err != nil {
			return nil, fmt.Errorf("LLM调用失败 (step %d): %w", step, err)
		}

		// 检查是否需要调用工具
		if len(resp.ToolCalls) > 0 {
			logger.Info("模型请求调用工具", map[string]interface{}{
				"step":       step,
				"tool_count": len(resp.ToolCalls),
			})

			// 添加助手消息（包含工具调用）
			messages = append(messages, types.Message{
				Role:      "assistant",
				Content:   resp.Content,
				ToolCalls: resp.ToolCalls,
			})

			// 执行每个工具调用
			for _, tc := range resp.ToolCalls {
				result, err := a.executeTool(tc)
				if err != nil {
					logger.Error("工具执行失败", map[string]interface{}{
						"tool":  tc.Function.Name,
						"error": err.Error(),
					})
					result = fmt.Sprintf("工具执行错误: %s", err.Error())
				}

				// 添加工具结果消息
				messages = append(messages, types.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}

			continue // 继续循环
		}

		// 模型直接返回结果
		logger.Info("模型返回最终结果", map[string]interface{}{
			"step":            step,
			"response_length": len(resp.Content),
		})

		return resp, nil
	}

	logger.Warn("达到最大 ReAct 循环步数", map[string]interface{}{
		"maxSteps": a.maxSteps,
	})

	// 返回最后一次响应
	return a.llm.Chat(messages, toolDefs)
}

// executeTool 执行单个工具调用
func (a *GenerationAgent) executeTool(tc types.ToolCall) (string, error) {
	toolName := tc.Function.Name

	logger.Info("执行工具", map[string]interface{}{
		"tool":      toolName,
		"arguments": tc.Function.Arguments,
	})

	// 解析参数
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
		return "", fmt.Errorf("解析工具参数失败: %w", err)
	}

	// 获取工具
	tool, err := a.toolRegistry.Get(toolName)
	if err != nil {
		return "", err
	}

	// 执行工具
	result, err := tool.Handler(params)
	if err != nil {
		return "", err
	}

	// 返回结果
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// buildUserContent 构建用户内容
// 包含用户画像和对话历史信息
func (a *GenerationAgent) buildUserContent(ctx *types.TaskContext, currentTime string) string {
	logger.Debug("构建用户内容", map[string]interface{}{
		"userProfileIsNil":  ctx.UserProfile == nil,
		"conversationCount": len(ctx.Conversations),
	})

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("请生成一个任务，核心主题：%s，当前时间：%s\n\n", ctx.CoreTheme, currentTime))

	// 添加用户画像信息
	if ctx.UserProfile != nil {
		sb.WriteString(a.buildUserProfileText(ctx.UserProfile))
		logger.Debug("已添加用户画像到提示词", map[string]interface{}{})
	} else {
		logger.Debug("用户画像为nil，跳过添加", map[string]interface{}{})
	}

	// 添加对话历史
	if len(ctx.Conversations) > 0 {
		sb.WriteString(a.buildConversationText(ctx.Conversations))
		logger.Debug("已添加对话历史到提示词", map[string]interface{}{
			"count": len(ctx.Conversations),
		})
	} else {
		logger.Debug("对话历史为空，跳过添加", map[string]interface{}{})
	}

	return sb.String()
}

// buildUserProfileText 构建用户画像文本
func (a *GenerationAgent) buildUserProfileText(profile *types.UserProfile) string {
	if profile == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 用户画像信息\n\n")

	if profile.Name != "" {
		sb.WriteString(fmt.Sprintf("- 姓名：%s\n", profile.Name))
	}
	if profile.Gender != "" {
		sb.WriteString(fmt.Sprintf("- 性别：%s\n", profile.Gender))
	}
	if profile.Age > 0 {
		sb.WriteString(fmt.Sprintf("- 年龄：%d\n", profile.Age))
	}
	if profile.Occupation != "" {
		sb.WriteString(fmt.Sprintf("- 职业：%s\n", profile.Occupation))
	}
	if len(profile.Hobbies) > 0 {
		sb.WriteString(fmt.Sprintf("- 兴趣爱好：%s\n", strings.Join(profile.Hobbies, ", ")))
	}
	if len(profile.Preferences) > 0 {
		sb.WriteString("- 偏好设置：\n")
		for key, value := range profile.Preferences {
			sb.WriteString(fmt.Sprintf("  - %s：%s\n", key, value))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// buildConversationText 构建对话历史文本
func (a *GenerationAgent) buildConversationText(conversations []*types.Conversation) string {
	if len(conversations) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 对话历史记录\n\n")

	for i, conv := range conversations {
		sb.WriteString(fmt.Sprintf("=== 对话 %d ===\n", i+1))
		sb.WriteString(fmt.Sprintf("时间：%s\n", conv.Timestamp.Format("2006-01-02 15:04:05")))
		if conv.Role != "" {
			sb.WriteString(fmt.Sprintf("角色：%s\n", conv.Role))
		}
		sb.WriteString(fmt.Sprintf("内容：\n%s\n\n", conv.Content))
	}

	return sb.String()
}

// loadUserData 加载用户画像和对话历史
func (a *GenerationAgent) loadUserData(ctx *types.TaskContext) {
	logger.Debug("开始加载用户数据", map[string]interface{}{
		"hasProfileStorage": a.profileStorage != nil,
		"hasHistoryStore":   a.historyStore != nil,
	})

	// 加载用户画像
	if err := a.loadUserProfile(ctx); err != nil {
		logger.Debug("加载用户画像失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 加载对话历史
	if err := a.loadConversations(ctx); err != nil {
		logger.Debug("加载对话历史失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Debug("用户数据加载完成", map[string]interface{}{
		"userProfileIsNil":  ctx.UserProfile == nil,
		"conversationCount": len(ctx.Conversations),
	})
}

// loadUserProfile 加载用户画像
// 使用依赖注入的 profileStorage，如果没有则跳过
// 参考 executor.go 的 getRecentConversations 实现
func (a *GenerationAgent) loadUserProfile(ctx *types.TaskContext) error {
	// 如果没有配置用户画像存储，跳过加载
	if a.profileStorage == nil {
		logger.Debug("用户画像存储未配置，跳过加载", map[string]interface{}{})
		ctx.UserProfile = nil
		return nil
	}

	// 加载用户画像（使用默认用户ID）
	userProfile, err := a.profileStorage.Load(context.Background(), "default")
	if err != nil {
		logger.Error("加载用户画像失败", map[string]interface{}{
			"error": err.Error(),
		})
		ctx.UserProfile = nil
		return nil
	}

	// 使用 ToPrompt 转换为文本格式
	profilePrompt := userProfile.ToPrompt()
	if profilePrompt == "" {
		logger.Debug("用户画像为空", map[string]interface{}{})
		ctx.UserProfile = nil
		return nil
	}

	// 转换为 multiagent 的 UserProfile
	ctx.UserProfile = &types.UserProfile{
		UserID:      userProfile.UserID,
		Name:        userProfile.Preferences["name"],
		Gender:      userProfile.Preferences["gender"],
		Occupation:  userProfile.Preferences["occupation"],
		Hobbies:     make([]string, 0),
		Preferences: userProfile.Preferences,
		RawContent:  profilePrompt, // 保存 ToPrompt 生成的文本
	}

	// 从标签中提取兴趣爱好
	for name, tag := range userProfile.Tags {
		if tag.Type == "interest" {
			ctx.UserProfile.Hobbies = append(ctx.UserProfile.Hobbies, name)
		}
	}

	logger.Info("用户画像加载成功", map[string]interface{}{
		"userId":     userProfile.UserID,
		"hobbies":    len(ctx.UserProfile.Hobbies),
		"tag_count":  len(userProfile.Tags),
		"pref_count": len(userProfile.Preferences),
	})

	return nil
}

// loadConversations 加载对话历史
// 使用依赖注入的 historyStore，如果没有则跳过
func (a *GenerationAgent) loadConversations(ctx *types.TaskContext) error {
	// 如果没有配置历史记录存储，跳过加载
	if a.historyStore == nil {
		logger.Debug("历史记录存储未配置，跳过加载", map[string]interface{}{})
		ctx.Conversations = make([]*types.Conversation, 0)
		return nil
	}

	// 使用 LoadRecent7Days 加载最近7天的历史记录
	entries, err := a.historyStore.LoadRecent7Days()
	if err != nil {
		logger.Error("加载历史记录失败", map[string]interface{}{
			"error": err.Error(),
		})
		ctx.Conversations = make([]*types.Conversation, 0)
		return nil
	}

	// 转换为 Conversation 列表
	ctx.Conversations = make([]*types.Conversation, 0, len(entries))
	for _, entry := range entries {
		ctx.Conversations = append(ctx.Conversations, &types.Conversation{
			ID:        entry.Key,
			Content:   entry.Content,
			Timestamp: entry.Timestamp,
			Role:      string(entry.Role),
		})
	}

	logger.Info("对话历史加载成功", map[string]interface{}{
		"count": len(ctx.Conversations),
	})

	return nil
}
