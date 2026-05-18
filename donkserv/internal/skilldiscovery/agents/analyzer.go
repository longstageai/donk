// agents 技能自动发现 Agent 模块
// Analyzer Agent 负责分析对话历史，提取潜在技能需求
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// AnalyzerAgent 对话分析 Agent
// 负责从对话历史中识别用户的潜在技能需求
// 支持 ReAct 循环，可自主调用工具获取额外信息
type AnalyzerAgent struct {
	llm          model.LLM
	tools        *tool.Registry
	dataDir      string
	dbPath       string
	embedder     embedding.Embedder
	settingSvc   *setting.ConfigProvider
	systemPrompt string
	userPrompt   string
	maxSteps     int
}

// NewAnalyzerAgent 创建分析 Agent
// 参数:
//   - llm: LLM 接口
//   - settingSvc: 配置服务（用于创建 embedder，可为 nil）
//   - prompt: 分析 Prompt（可选，为空使用默认）
//
// 返回:
//   - *AnalyzerAgent: Agent 实例
func NewAnalyzerAgent(llm model.LLM, settingSvc *setting.ConfigProvider, prompt string) *AnalyzerAgent {
	agent := &AnalyzerAgent{
		llm:        llm,
		settingSvc: settingSvc,
		maxSteps:   3, // 默认最大 5 步 ReAct 循环
	}

	if prompt == "" {
		agent.systemPrompt = defaultAnalyzerSystemPrompt
		agent.userPrompt = defaultAnalyzerUserPrompt
	} else {
		agent.systemPrompt = prompt
		agent.userPrompt = ""
	}

	// 初始化工具注册表
	agent.initTools()

	return agent
}

// initTools 初始化工具注册表
// 注册知识库搜索工具
func (a *AnalyzerAgent) initTools() {
	a.tools = tool.NewRegistry()

	// 创建 embedder 并注册知识库搜索工具
	if err := a.createAndRegisterKnowledgeTool(); err != nil {
		logger.Debug("知识库搜索工具未启用", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// createAndRegisterKnowledgeTool 创建并注册知识库搜索工具
func (a *AnalyzerAgent) createAndRegisterKnowledgeTool() error {
	// 设置默认路径（使用 ./data/knowledge 与项目其他部分保持一致）
	a.dataDir = filepath.Join(".", "data", "knowledge")
	a.dbPath = filepath.Join(a.dataDir, "meta.db")

	// 创建 embedder
	embedder, err := a.createEmbedder()
	if err != nil {
		return fmt.Errorf("创建 embedder 失败: %w", err)
	}
	a.embedder = embedder

	// 注册知识库搜索工具
	knowledgeTool := builtin.NewKnowledgeSearcher(a.dataDir, a.dbPath, a.embedder)
	if err := a.tools.Register(knowledgeTool); err != nil {
		return fmt.Errorf("注册知识库搜索工具失败: %w", err)
	}

	logger.Info("知识库搜索工具注册成功", map[string]interface{}{
		"dataDir": a.dataDir,
		"dbPath":  a.dbPath,
	})

	return nil
}

// createEmbedder 从 setting 创建嵌入器
func (a *AnalyzerAgent) createEmbedder() (embedding.Embedder, error) {
	if a.settingSvc == nil {
		return nil, fmt.Errorf("配置服务未设置")
	}

	// 获取 embedding 配置
	embedConfig, err := a.settingSvc.GetEmbeddingConfig()
	if err != nil {
		return nil, fmt.Errorf("获取 embedding 配置失败: %w", err)
	}

	if embedConfig == nil || embedConfig.Provider == "" || embedConfig.Model == "" || embedConfig.APIKey == "" {
		return nil, fmt.Errorf("embedding 配置不完整")
	}

	// 创建 embedder
	embedder, err := embedding.NewEmbedding(
		embedConfig.Provider,
		embedConfig.APIKey,
		embedConfig.Model,
		embedConfig.BaseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 embedder 失败: %w", err)
	}

	logger.Info("创建 embedder 成功", map[string]interface{}{
		"provider": embedConfig.Provider,
		"model":    embedConfig.Model,
	})

	return embedder, nil
}

// NewAnalyzerAgentSimple 创建简化版 AnalyzerAgent（不使用工具）
// 参数:
//   - llm: LLM 接口
//   - prompt: 分析 Prompt（可选，为空使用默认）
//
// 返回:
//   - *AnalyzerAgent: Agent 实例
func NewAnalyzerAgentSimple(llm model.LLM, prompt string) *AnalyzerAgent {
	return NewAnalyzerAgent(llm, nil, prompt)
}

// SetMaxSteps 设置最大 ReAct 循环步数
func (a *AnalyzerAgent) SetMaxSteps(steps int) {
	a.maxSteps = steps
}

// Analyze 分析对话历史
// 参数:
//   - ctx: 上下文
//   - conversations: 对话历史列表
//
// 返回:
//   - []*SkillCandidate: 候选技能列表
//   - error: 错误信息
func (a *AnalyzerAgent) Analyze(ctx context.Context, conversations []Conversation) ([]*SkillCandidate, error) {
	candidates, _, err := a.AnalyzeWithUsage(ctx, conversations)
	return candidates, err
}

// AnalyzeWithUsage 分析对话历史并返回 Token 使用情况
// 支持 ReAct 循环：如果配置了工具，Agent 可以自主决定调用工具获取额外信息
// 参数:
//   - ctx: 上下文
//   - conversations: 对话历史列表
//
// 返回:
//   - []*SkillCandidate: 候选技能列表
//   - *schema.UsageInfo: Token 使用信息
//   - error: 错误信息
func (a *AnalyzerAgent) AnalyzeWithUsage(ctx context.Context, conversations []Conversation) ([]*SkillCandidate, *schema.UsageInfo, error) {
	logger.Info("开始分析对话历史", map[string]interface{}{
		"conversation_count": len(conversations),
		"tools_enabled":      a.tools != nil,
	})

	if len(conversations) == 0 {
		logger.Debug("对话历史为空，跳过分析", map[string]interface{}{})
		return []*SkillCandidate{}, nil, nil
	}

	// 如果没有配置工具，使用简单的单次调用模式
	if a.tools == nil || len(a.tools.List()) == 0 {
		return a.analyzeSimple(ctx, conversations)
	}

	// 使用 ReAct 循环模式
	return a.analyzeWithReAct(ctx, conversations)
}

// analyzeSimple 简单分析模式（无工具）
func (a *AnalyzerAgent) analyzeSimple(ctx context.Context, conversations []Conversation) ([]*SkillCandidate, *schema.UsageInfo, error) {
	// 构建对话文本
	conversationText := a.buildConversationText(conversations)

	// 构建用户提示词
	var userContent string
	if a.userPrompt != "" {
		userContent = fmt.Sprintf(a.userPrompt, conversationText)
	} else {
		userContent = conversationText
	}

	// 调用 LLM 分析
	req := &schema.ChatRequest{
		Messages: []schema.Message{
			{Role: "system", Content: a.systemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	response, err := a.llm.Chat(ctx, req)
	if err != nil {
		logger.Error("LLM 分析失败", map[string]interface{}{"error": err.Error()})
		return nil, nil, fmt.Errorf("LLM 分析失败: %w", err)
	}

	// 解析结果
	candidates, err := a.parseResponse(response.Content)
	if err != nil {
		return nil, nil, fmt.Errorf("解析分析结果失败: %w", err)
	}

	logger.Info("简单分析完成", map[string]interface{}{
		"candidate_count": len(candidates),
	})

	return candidates, &response.Usage, nil
}

// analyzeWithReAct 使用 ReAct 循环分析（支持工具调用）
func (a *AnalyzerAgent) analyzeWithReAct(ctx context.Context, conversations []Conversation) ([]*SkillCandidate, *schema.UsageInfo, error) {
	// 构建初始消息
	messages := a.buildInitialMessages(conversations)

	totalUsage := &schema.UsageInfo{}

	// ReAct 循环
	for step := 0; step < a.maxSteps; step++ {
		logger.Debug("ReAct 循环步骤", map[string]interface{}{
			"step":     step + 1,
			"maxSteps": a.maxSteps,
		})

		// 构建请求，包含可用工具
		req := &schema.ChatRequest{
			Messages: messages,
			Tools:    a.tools.GetToolDefinitions(),
		}

		// 调用 LLM
		response, err := a.llm.Chat(ctx, req)
		if err != nil {
			logger.Error("LLM 调用失败", map[string]interface{}{
				"step":  step,
				"error": err.Error(),
			})
			return nil, nil, fmt.Errorf("LLM 调用失败 (step %d): %w", step, err)
		}

		// 累加 Token 使用
		totalUsage.PromptTokens += response.Usage.PromptTokens
		totalUsage.CompletionTokens += response.Usage.CompletionTokens
		totalUsage.TotalTokens += response.Usage.TotalTokens

		// 检查是否需要调用工具
		if len(response.ToolCalls) > 0 {
			logger.Info("模型请求调用工具", map[string]interface{}{
				"step":       step,
				"tool_count": len(response.ToolCalls),
			})

			// 添加助手消息（包含工具调用）
			messages = append(messages, schema.Message{
				Role:      "assistant",
				Content:   response.Content,
				ToolCalls: response.ToolCalls,
			})

			// 执行每个工具调用
			for _, tc := range response.ToolCalls {
				result, err := a.executeTool(ctx, tc)
				if err != nil {
					logger.Error("工具执行失败", map[string]interface{}{
						"tool":  tc.Function.Name,
						"error": err.Error(),
					})
					result = fmt.Sprintf("工具执行错误: %s", err.Error())
				}

				// 添加工具结果消息
				messages = append(messages, schema.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}

			continue // 继续循环，让模型基于工具结果继续思考
		}

		// 模型直接返回结果，解析候选技能
		logger.Info("模型返回最终结果", map[string]interface{}{
			"step":            step,
			"response_length": len(response.Content),
		})

		candidates, err := a.parseResponse(response.Content)
		if err != nil {
			return nil, nil, fmt.Errorf("解析分析结果失败: %w", err)
		}

		logger.Info("ReAct 分析完成", map[string]interface{}{
			"steps":           step + 1,
			"candidate_count": len(candidates),
		})

		// 记录每个候选技能
		for _, c := range candidates {
			logger.Debug("发现候选技能", map[string]interface{}{
				"name":       c.Name,
				"confidence": c.Confidence,
			})
		}

		return candidates, totalUsage, nil
	}

	logger.Warn("达到最大 ReAct 循环步数，返回空结果", map[string]interface{}{
		"maxSteps": a.maxSteps,
	})
	return []*SkillCandidate{}, totalUsage, nil
}

// buildInitialMessages 构建初始消息列表
func (a *AnalyzerAgent) buildInitialMessages(conversations []Conversation) []schema.Message {
	// 构建对话文本
	conversationText := a.buildConversationText(conversations)

	// 构建用户提示词
	var userContent string
	if a.userPrompt != "" {
		userContent = fmt.Sprintf(a.userPrompt, conversationText)
	} else {
		userContent = conversationText
	}

	return []schema.Message{
		{Role: "system", Content: a.systemPrompt},
		{Role: "user", Content: userContent},
	}
}

// executeTool 执行单个工具调用
func (a *AnalyzerAgent) executeTool(ctx context.Context, tc schema.ToolCall) (string, error) {
	toolName := tc.Function.Name
	argsStr := tc.Function.Arguments

	logger.Info("执行工具", map[string]interface{}{
		"tool":      toolName,
		"arguments": argsStr,
	})

	// 解析参数
	var params map[string]any
	if err := json.Unmarshal([]byte(argsStr), &params); err != nil {
		return "", fmt.Errorf("解析工具参数失败: %w", err)
	}

	// 执行工具
	result, err := a.tools.Execute(toolName, params)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

// buildConversationText 构建对话文本
// 参数:
//   - conversations: 对话列表
//
// 返回:
//   - string: 格式化的对话文本
func (a *AnalyzerAgent) buildConversationText(conversations []Conversation) string {
	var sb strings.Builder

	// 分离用户画像和对话历史
	var profileConversations []Conversation
	var historyConversations []Conversation

	for _, conv := range conversations {
		if conv.ID == "profile_tags" {
			profileConversations = append(profileConversations, conv)
		} else {
			historyConversations = append(historyConversations, conv)
		}
	}

	// 先输出用户画像
	if len(profileConversations) > 0 {
		sb.WriteString("## 用户画像信息\n\n")
		for _, conv := range profileConversations {
			sb.WriteString(fmt.Sprintf("更新时间: %s\n", conv.Timestamp.Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("画像内容:\n%s\n\n", conv.Content))
		}
	}

	// 再输出对话历史
	if len(historyConversations) > 0 {
		sb.WriteString("## 对话历史记录\n\n")
		for i, conv := range historyConversations {
			sb.WriteString(fmt.Sprintf("=== 对话 %d ===\n", i+1))
			sb.WriteString(fmt.Sprintf("时间: %s\n", conv.Timestamp.Format("2006-01-02 15:04:05")))
			sb.WriteString(fmt.Sprintf("内容:\n%s\n\n", conv.Content))
		}
	}

	return sb.String()
}

// parseResponse 解析 LLM 响应
// 参数:
//   - response: LLM 响应文本
//
// 返回:
//   - []*SkillCandidate: 候选技能列表
//   - error: 错误信息
func (a *AnalyzerAgent) parseResponse(response string) ([]*SkillCandidate, error) {
	// 提取 JSON 部分
	jsonStr := a.extractJSON(response)
	if jsonStr == "" {
		logger.Warn("未能从响应中提取 JSON", map[string]interface{}{
			"response": response,
		})
		return []*SkillCandidate{}, nil
	}

	// 解析 JSON
	var result struct {
		Candidates []struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Trigger     string   `json:"trigger"`
			Confidence  float64  `json:"confidence"`
			Evidence    []string `json:"evidence"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 转换为 SkillCandidate
	candidates := make([]*SkillCandidate, 0, len(result.Candidates))
	for _, c := range result.Candidates {
		// 过滤低置信度的候选
		if c.Confidence < 0.5 {
			logger.Debug("跳过低置信度候选", map[string]interface{}{
				"name":       c.Name,
				"confidence": c.Confidence,
			})
			continue
		}

		candidates = append(candidates, &SkillCandidate{
			Name:        c.Name,
			Description: c.Description,
			Trigger:     c.Trigger,
			Confidence:  c.Confidence,
			Evidence:    c.Evidence,
		})
	}

	return candidates, nil
}

// extractJSON 从文本中提取 JSON
// 参数:
//   - text: 包含 JSON 的文本
//
// 返回:
//   - string: 提取的 JSON 字符串
func (a *AnalyzerAgent) extractJSON(text string) string {
	// 查找 JSON 开始标记
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		return ""
	}

	// 查找 JSON 结束标记（最后一个 }）
	endIdx := strings.LastIndex(text, "}")
	if endIdx == -1 || endIdx <= startIdx {
		return ""
	}

	return text[startIdx : endIdx+1]
}

// defaultAnalyzerSystemPrompt 默认分析系统提示词
const defaultAnalyzerSystemPrompt = `
#你是一个富有创造力的 AI 助手，专注于生成独特且实用的技能创意。

## 输入
- 用户最近对话（recent_dialogues）
- 用户人物画像（user_profile）
- 历史技能（past_skills）
- 向量数据库/联网资源（vector_db_query(query), fetch_online_resources(query)）

## 设计原则
1. 创新：可以是全新功能或对现有功能的创新组合
2. 实用：解决实际问题或提升效率，优先将与用户对话过程中完成的工作原封不动的总结为可执行的skill。
3. 可实现：技术上可落地，避免依赖不可控资源
4. 去重：避免生成与 past_skills 相同或高度相似的技能
5. 跨界：鼓励不同功能或类别的组合

## 创意方向
- 数据处理：格式化、分析、批量操作
- 内容创作：写作辅助、创意生成、内容结构化
- 效率工具：自动化重复任务、工作流优化
- 学习辅助：知识整理、记忆辅助、学习规划
- 生活助手：日程管理、习惯追踪、决策辅助

## 输出格式（JSON）
{
  "name": "skill-name",
  "description": "技能能做什么、解决什么问题、价值体现",
  "trigger": "触发场景，用户什么时候需要这个技能"
}

注意：
- name 使用英文小写和短横线
- description 具体说明技能价值
- trigger 指明使用场景
- 每次只生成一个独特创意
- 生成前先调用 past_skills 去重，必要时使用向量数据库检索相似技能





你是专业的需求分析师，擅长从用户对话和画像识别潜在技能需求。

## 输入
- 用户近期对话（recent_dialogues）
- 用户人物画像变更（user_profile_history）
- 历史技能（past_skills）

## 分析目标
1. 识别重复性操作
2. 发现自动化意愿和效率提升需求
3. 捕捉功能期望或未满足需求
4. 推断技能成长轨迹和兴趣偏好变化

## 方法提示
- 使用 past_skills 去重，避免生成已存在技能
- 对用户标签、偏好变化进行分析
- 可调用向量数据库 query_vector_db(user_id, query) 或联网 fetch_online_resources(query) 获取灵感或相关技能

## 输出格式（JSON）
{
  "candidates": [
    {
      "name": "skill-name",
      "description": "技能的详细描述",
      "trigger": "触发场景",
      "confidence": 0.85,
      "evidence": ["对话原文或标签变更"]
    }
  ]
}

注意：
- confidence 范围 0-1，低于 0.9 的候选过滤掉
- evidence 必须引用真实对话或画像变化
`

// defaultAnalyzerUserPrompt 默认分析用户提示词
const defaultAnalyzerUserPrompt = `## 输入内容

%s

请分析以上内容，输出 JSON 格式的分析结果。`
