package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/scheduler"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
)

// GoalCreativeAgentDeps 目标创意 Agent 的依赖配置
type GoalCreativeAgentDeps struct {
	Embedder     embedding.Embedder   // 向量嵌入模型（用于 knowledge_search）
	LongMemory   *memory.LongMemory   // 长期记忆（用于 memory_search）
	HistoryStore *memory.HistoryStore // 历史记录存储（用于获取最近对话）
	Profile      *profile.UserProfile // 用户画像（用于个性化目标生成）
	Scheduler    *scheduler.Scheduler // 任务调度器（用于 task_manager）
	SkillsDir    string               // 技能目录（用于 skill_creator）
	DataDir      string               // 数据目录（用于 knowledge_search）
	DBPath       string               // 数据库路径（用于 knowledge_search）
}

// NewGoalCreativeAgent 创建目标创意 Agent，负责生成候选目标并推进去重检查。
// 该 Agent 配置了 http、knowledge_search、memory_search、official_skills_search、skill_creator、task_manager 工具。
// deps 参数为可选，如果为 nil 或部分字段为空，则对应工具将使用零值初始化（可能无法正常工作）。
func NewGoalCreativeAgent(llm CreativeLLMClient, deps *GoalCreativeAgentDeps) creative.Agent {
	// 创建工具注册表并注册目标创意 Agent 所需的工具
	tools := tool.NewRegistry()

	// 注册 HTTP 工具
	tools.Register(builtin.NewHTTP())

	// 注册知识库搜索工具
	if deps != nil && deps.Embedder != nil {
		dataDir := deps.DataDir
		if dataDir == "" {
			dataDir = "./data/knowledge"
		}
		dbPath := deps.DBPath
		if dbPath == "" {
			dbPath = "./data/db/donk.db"
		}
		tools.Register(builtin.NewKnowledgeSearcher(dataDir, dbPath, deps.Embedder))
	}

	// 注册记忆搜索工具
	if deps != nil && deps.LongMemory != nil {
		tools.Register(builtin.NewMemorySearcher(deps.LongMemory))
	}

	// 注册任务管理工具
	if deps != nil && deps.Scheduler != nil {
		tools.Register(builtin.NewTaskManager(deps.Scheduler))
	}

	// 使用动态提示词构建器，每次对话时重新构建系统提示词
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		systemPrompt := buildGoalCreativePrompt(deps)
		return prompt(systemPrompt)
	}

	// 构建动态系统提示词，包含用户画像和对话历史
	//systemPrompt := buildGoalCreativePrompt(deps)
	//return NewLLMAgent("goal_creative", "目标创意 Agent", creative.RoleGoalCreative, []creative.EventType{creative.EventGoalRequested, creative.EventGoalRegenerationRequested, creative.EventGoalRefinementRequested}, promptSpec(systemPrompt), llm, goalCreativeOutput, WithTools(tools), WithHistoryStore(getHistoryStore(deps)), WithProfile(getProfile(deps)))
	return NewLLMAgentWithDynamicPrompt("goal_creative", "目标创意 Agent", creative.RoleGoalCreative, []creative.EventType{creative.EventGoalRequested, creative.EventGoalRegenerationRequested, creative.EventGoalRefinementRequested}, promptBuilder, llm, goalCreativeOutput, WithTools(tools), WithHistoryStore(getHistoryStore(deps)), WithProfile(getProfile(deps)))
}

// getHistoryStore 安全获取 HistoryStore
func getHistoryStore(deps *GoalCreativeAgentDeps) *memory.HistoryStore {
	if deps == nil {
		return nil
	}
	return deps.HistoryStore
}

// getProfile 安全获取 UserProfile
func getProfile(deps *GoalCreativeAgentDeps) *profile.UserProfile {
	if deps == nil {
		return nil
	}
	return deps.Profile
}

// buildGoalCreativePrompt 构建目标创意 Agent 的动态系统提示词
// 包含基础提示词、用户画像信息和最近对话历史
func buildGoalCreativePrompt(deps *GoalCreativeAgentDeps) string {
	var parts []string

	// 1. 基础提示词
	parts = append(parts, goalCreativePrompt)

	// 2. 用户画像信息
	if deps != nil && deps.Profile != nil {
		profileInfo := formatProfileInfo(deps.Profile)
		if profileInfo != "" {
			parts = append(parts, "\n# 当前用户画像信息\n", profileInfo)
		}
	}

	// 3. 最近对话历史
	if deps != nil && deps.HistoryStore != nil {
		historyInfo := formatRecentHistory(deps.HistoryStore)
		if historyInfo != "" {
			parts = append(parts, "\n# 最近对话记录\n", historyInfo)
		}
	}

	return strings.Join(parts, "\n")
}

func prompt(systemPrompt string) PromptSpec {
	return PromptSpec{SystemPrompt: systemPrompt, OutputFormat: "请严格按系统提示词中的输出格式回答。评审类 Agent 必须明确写出 判定结果：通过 或 判定结果：不通过，并给出原因。"}
}

// formatProfileInfo 格式化用户画像信息
func formatProfileInfo(p *profile.UserProfile) string {
	if p == nil {
		return "暂无用户画像信息"
	}

	var parts []string

	// 用户ID
	if p.UserID != "" {
		parts = append(parts, fmt.Sprintf("- 用户ID: %s", p.UserID))
	}

	// 标签（只显示高置信度的）
	if len(p.Tags) > 0 {
		var highConfidenceTags []string
		for _, tag := range p.Tags {
			if tag.Confidence > 0.7 {
				highConfidenceTags = append(highConfidenceTags, fmt.Sprintf("%s (%s, 置信度: %.2f)", tag.Name, tag.Type, tag.Confidence))
			}
		}
		if len(highConfidenceTags) > 0 {
			parts = append(parts, "- 特征标签:")
			for _, tag := range highConfidenceTags {
				parts = append(parts, fmt.Sprintf("  - %s", tag))
			}
		}
	}

	// 偏好设置
	if len(p.Preferences) > 0 {
		parts = append(parts, "- 偏好设置:")
		for k, v := range p.Preferences {
			parts = append(parts, fmt.Sprintf("  - %s: %s", k, v))
		}
	}

	if len(parts) == 0 {
		return "用户画像信息为空"
	}

	return strings.Join(parts, "\n")
}

// formatRecentHistory 格式化最近对话历史
func formatRecentHistory(store *memory.HistoryStore) string {
	if store == nil {
		return "暂无对话历史"
	}

	// 获取最近10条对话记录
	entries, err := store.GetRecent(500)
	if err != nil {
		return fmt.Sprintf("获取对话记录失败: %v", err)
	}
	if len(entries) == 0 {
		return "暂无最近对话记录"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("以下是最近 %d 条对话记录（从新到旧）:\n", len(entries)))

	for i, entry := range entries {
		parts = append(parts, fmt.Sprintf("\n[%d] %s - %s:", i+1, entry.Role, entry.Timestamp.Format("2006-01-02 15:04:05")))
		parts = append(parts, entry.Content)
	}

	return strings.Join(parts, "\n")
}

func goalCreativeOutput(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
	goal := creative.CandidateGoal{ID: creative.NextID("candidate_goal"), Title: firstNonEmpty(extractJSONField(content, "title"), "LLM 生成的候选目标"), Description: firstNonEmpty(extractJSONField(content, "description"), content), Motivation: extractJSONField(content, "value"), ContextBasis: []string{extractJSONField(content, "context_basis")}, ExpectedOutput: firstNonEmpty(extractJSONField(content, "expected_output"), "标准化候选目标"), CreatedBy: "goal_creative", CreatedAt: time.Now()}
	return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}, Artifacts: []creative.ArtifactDraft{{Type: creative.ArtifactCandidateGoal, Data: goal}}, Events: []creative.EventDraft{{Type: creative.EventGoalDedupRequested, DispatchMode: creative.DispatchExclusive, Priority: 90}}}
}
