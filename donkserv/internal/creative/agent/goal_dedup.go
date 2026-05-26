package agent

import (
	"context"
	"fmt"
	"github.com/longstageai/donk/donk/pkg/logger"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/creative"
	donksql "github.com/longstageai/donk/donk/internal/sql"
)

// CreativeDedupTool 创意去重工具
type CreativeDedupTool struct {
	db *donksql.DB
}

// NewCreativeDedupTool 创建创意去重工具
func NewCreativeDedupTool(db *donksql.DB) *CreativeDedupTool {
	return &CreativeDedupTool{
		db: db,
	}
}

// SetDB 设置数据库连接
func (t *CreativeDedupTool) SetDB(db *donksql.DB) {
	t.db = db
}

// GetRecentCreatives 查询最近N条创意记录（默认100条）
func (t *CreativeDedupTool) GetRecentCreatives(limit int) ([]donksql.CreativeRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	if t.db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	query := `
		SELECT id, title, description, content, source, status, created_at, updated_at
		FROM creatives
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := t.db.Query(query, limit)
	if err != nil {
		// 如果表不存在，返回空列表
		if isTableNotExist(err) {
			return []donksql.CreativeRecord{}, nil
		}
		return nil, fmt.Errorf("查询创意记录失败: %w", err)
	}
	defer rows.Close()

	var records []donksql.CreativeRecord
	for rows.Next() {
		var r donksql.CreativeRecord
		err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Content, &r.Source, &r.Status, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			continue
		}
		records = append(records, r)
	}

	return records, nil
}

// GetRecentCreativesText 获取历史创意文本格式（用于提示词）
func (t *CreativeDedupTool) GetRecentCreativesText(limit int) string {
	records, err := t.GetRecentCreatives(limit)
	if err != nil {
		return "暂无历史创意记录"
	}

	if len(records) == 0 {
		return "暂无历史创意记录"
	}

	var sb strings.Builder
	for i, r := range records {
		sb.WriteString(fmt.Sprintf("%d. 标题：%s\n   描述：%s\n\n", i+1, r.Title, r.Description))
	}
	return sb.String()
}

// InsertCreative 插入新的创意记录
func (t *CreativeDedupTool) InsertCreative(title, description, content, source string) (int64, error) {
	if t.db == nil {
		return 0, fmt.Errorf("数据库连接未初始化")
	}

	// 确保表存在
	if err := t.ensureCreativesTable(); err != nil {
		return 0, fmt.Errorf("创建创意表失败: %w", err)
	}

	query := `
		INSERT INTO creatives (title, description, content, source, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := t.db.Exec(query, title, description, content, source, "active", now, now)
	if err != nil {
		return 0, fmt.Errorf("插入创意记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取插入ID失败: %w", err)
	}

	return id, nil
}

// ensureCreativesTable 确保创意表存在
func (t *CreativeDedupTool) ensureCreativesTable() error {
	if t.db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	for _, schema := range donksql.TableSchemas {
		if strings.Contains(schema, "CREATE TABLE IF NOT EXISTS creatives") {
			_, err := t.db.Exec(schema)
			if err != nil {
				return err
			}
		}
		// 同时创建索引
		if strings.Contains(schema, "idx_creatives_") {
			_, err := t.db.Exec(schema)
			if err != nil {
				// 索引创建失败不影响主流程
				continue
			}
		}
	}
	return nil
}

// isTableNotExist 检查错误是否为表不存在
func isTableNotExist(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "no such table") ||
		strings.Contains(errStr, "doesn't exist") ||
		strings.Contains(errStr, "does not exist")
}

// GoalDedupAgentDeps 目标去重 Agent 的依赖配置
type GoalDedupAgentDeps struct {
	DB *donksql.DB // 数据库连接（必传）
}

// NewGoalDedupAgent 创建目标去重 Agent，负责检测候选目标是否与历史任务重复。
// deps 参数中的 DB 是必传的，用于创意去重和存储。
func NewGoalDedupAgent(llm CreativeLLMClient, deps *GoalDedupAgentDeps) creative.Agent {
	if deps == nil || deps.DB == nil {
		panic("NewGoalDedupAgent: deps.DB 是必传参数，不能为nil")
	}

	// 确保表存在
	tool := NewCreativeDedupTool(deps.DB)
	if err := tool.ensureCreativesTable(); err != nil {
		// 记录错误但不阻止创建Agent
		fmt.Printf("警告: 创建创意表失败: %v\n", err)
	}

	// 使用动态提示词构建
	promptBuilder := func(input creative.AgentInput) PromptSpec {
		return buildGoalDedupPrompt(deps.DB, input)
	}

	return NewLLMAgentWithDynamicPrompt("goal_dedup", "目标去重 Agent", creative.RoleGoalDedup, []creative.EventType{creative.EventGoalDedupRequested}, promptBuilder, llm, goalDedupOutputWithDB(deps.DB))
}

// buildGoalDedupPrompt 构建动态提示词，包含历史创意和新创意
func buildGoalDedupPrompt(db *donksql.DB, input creative.AgentInput) PromptSpec {
	tool := NewCreativeDedupTool(db)

	// 获取历史100条创意（文本格式）
	historyCreatives := tool.GetRecentCreativesText(20)

	// 获取新创意信息（获取最新的 CandidateGoal，即列表中最后一个）
	var title, description, content string
	for i := len(input.Artifacts) - 1; i >= 0; i-- {
		artifact := input.Artifacts[i]
		if artifact.Type == creative.ArtifactCandidateGoal {
			if goalData, ok := artifact.Data.(creative.CandidateGoal); ok {
				title = goalData.Title
				description = goalData.Description
				content = goalData.ExpectedOutput
				break
			}
		}
	}

	// 系统提示词：包含角色定义和历史创意
	systemPrompt := fmt.Sprintf(goalDedupPromptTemplate, historyCreatives, title, description, content)

	// 用户提示词：Agent 自己管理，只保留必要运行时信息
	userPrompt := fmt.Sprintf(`请根据系统提示词中的信息完成去重检测。

当前事件类型：%s
当前阶段：%s

输出要求：
请严格按系统提示词中的输出格式回答。必须明确写出 判定结果：通过 或 判定结果：不通过，并给出原因。`, input.Event.Type, input.Session.CurrentPhase)

	return PromptSpec{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		OutputFormat: "请严格按系统提示词中的输出格式回答。必须明确写出 判定结果：通过 或 判定结果：不通过，并给出原因。",
	}
}

// goalDedupOutputWithDB 带数据库功能的目标去重输出处理
func goalDedupOutputWithDB(db *donksql.DB) func(context.Context, creative.AgentInput, string, creative.TokenUsage) creative.AgentOutput {
	return func(ctx context.Context, input creative.AgentInput, content string, usage creative.TokenUsage) creative.AgentOutput {
		// 解析Agent的判定结果
		decision := parseAgentDecision(content)

		// 提取新创意信息（获取最新的 CandidateGoal，即列表中最后一个）
		var goalData *creative.CandidateGoal
		for i := len(input.Artifacts) - 1; i >= 0; i-- {
			artifact := input.Artifacts[i]
			if artifact.Type == creative.ArtifactCandidateGoal {
				if gd, ok := artifact.Data.(creative.CandidateGoal); ok {
					goalData = &gd
					break
				}
			}
		}

		// 根据判定结果处理
		if decision == creative.DecisionSucceeded && goalData != nil {
			// 判定通过，保存创意到数据库
			tool := NewCreativeDedupTool(db)
			id, err := tool.InsertCreative(
				goalData.Title,
				goalData.Description,
				goalData.ExpectedOutput,
				string(goalData.CreatedBy),
			)
			if err != nil {
				fmt.Printf("创意插入数据库失败: %v\n", err)
			} else {
				logger.Info(fmt.Sprintf("创意已插入数据库，ID: %d\n", id), nil)
			}
			// 发送通过事件
			return creative.AgentOutput{
				Status:     creative.AgentRunSucceeded,
				Decision:   creative.DecisionSucceeded,
				TokenUsage: usage,
				Messages: []creative.MessageDraft{
					{Role: creative.MessageRoleAgent, Content: content},
				},
				Artifacts: []creative.ArtifactDraft{
					{Type: creative.ArtifactDedupReview, Data: map[string]any{"decision": "passed", "content": content}},
				},
				Events: []creative.EventDraft{
					{Type: creative.EventGoalDedupPassed, DispatchMode: creative.DispatchExclusive, Priority: 80},
					{Type: creative.EventGoalValueReviewRequested, DispatchMode: creative.DispatchExclusive, Priority: 80},
				},
			}
		}

		// 判定未通过，不保存创意，发送未通过事件
		return creative.AgentOutput{
			Status:     creative.AgentRunRejected,
			Decision:   creative.DecisionRejected,
			TokenUsage: usage,
			Messages: []creative.MessageDraft{
				{Role: creative.MessageRoleAgent, Content: content},
			},
			Artifacts: []creative.ArtifactDraft{
				{Type: creative.ArtifactDedupReview, Data: map[string]any{"decision": "rejected", "content": content}},
			},
			Events: []creative.EventDraft{
				{Type: creative.EventGoalDedupRejected, DispatchMode: creative.DispatchExclusive, Priority: 80},
				{Type: creative.EventGoalRegenerationRequested, DispatchMode: creative.DispatchExclusive, Priority: 80},
			},
		}
	}
}
