package websocket

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/config"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
	_ "github.com/mattn/go-sqlite3"
)

type NotificationSummaryAgent struct {
	tokenStats *token.TokenStats
	db         *sql.DB
}

type notificationSummaryResult struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func NewNotificationSummaryAgent() *NotificationSummaryAgent {
	db, err := sql.Open("sqlite3", config.GetDataPaths().MainDB+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		logger.Warn("通知总结Token统计数据库打开失败", map[string]interface{}{
			"error": err.Error(),
		})
		return &NotificationSummaryAgent{}
	}

	tokenStats, err := token.NewTokenStats(db)
	if err != nil {
		logger.Warn("通知总结Token统计初始化失败", map[string]interface{}{
			"error": err.Error(),
		})
		_ = db.Close()
		return &NotificationSummaryAgent{}
	}

	return &NotificationSummaryAgent{
		tokenStats: tokenStats,
		db:         db,
	}
}

func (a *NotificationSummaryAgent) Summarize(ctx context.Context, title, content string) (string, string, error) {
	if a.isTokenBudgetExceeded() {
		return title, content, fmt.Errorf("Token预算已超出限额")
	}

	provider := setting.GetProvider()
	if provider == nil {
		return title, content, fmt.Errorf("配置提供者未初始化")
	}

	llmConfig, err := provider.GetLLMConfig()
	if err != nil {
		return title, content, fmt.Errorf("获取LLM配置失败: %w", err)
	}
	if llmConfig == nil || llmConfig.Provider == "" || llmConfig.Model == "" {
		return title, content, fmt.Errorf("LLM配置不完整")
	}

	adapter, err := model.NewAdapter(llmConfig.Provider, llmConfig.APIKey, llmConfig.Model, llmConfig.BaseURL)
	if err != nil {
		return title, content, fmt.Errorf("创建LLM适配器失败: %w", err)
	}
	if adapter == nil {
		return title, content, fmt.Errorf("不支持的LLM提供商: %s", llmConfig.Provider)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := adapter.Chat(ctx, &schema.ChatRequest{
		Temperature: llmConfig.Temperature,
		MaxTokens:   llmConfig.MaxTokens,
		Messages: []schema.Message{
			{
				Role: "system",
				Content: `你是 Donk（核动驴），一只拟人化宠物助手，全天候为用户干活。
你的通知风格：短句、幽默、萌趣、带一点小脾气，有陪伴感，但不夸张。
你的任务是把输入的通知标题和内容改写成更适合推送给用户的短通知。
要求：
1. 只输出 JSON 对象，格式为 {"title":"...","content":"..."}。
2. title 不超过 20 个中文字符。
3. content 不超过 80 个中文字符。
4. 保留原通知的核心事实，不添加输入中不存在的信息。
5. 不要进行多轮确认，不要调用工具，不要输出解释。
6. 语气像提醒主人：简短、可爱、利落。`,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("标题：%s\n内容：%s", title, content),
			},
		},
	})
	if err != nil {
		return title, content, fmt.Errorf("LLM总结失败: %w", err)
	}
	if resp == nil {
		return title, content, fmt.Errorf("LLM响应为空")
	}
	if resp.Error != nil {
		return title, content, fmt.Errorf("LLM返回错误: %s", resp.Error.Message)
	}

	if err := a.recordTokenUsage(resp); err != nil {
		logger.Warn("通知总结Token消耗记录失败", map[string]interface{}{
			"error": err.Error(),
		})
	}
	if a.isTokenBudgetExceeded() {
		return title, content, fmt.Errorf("Token预算已超出限额")
	}

	summary, err := parseNotificationSummary(resp.Content)
	if err != nil {
		return title, content, err
	}
	if summary.Title == "" {
		summary.Title = title
	}
	if summary.Content == "" {
		summary.Content = content
	}

	return summary.Title, summary.Content, nil
}

func (a *NotificationSummaryAgent) isTokenBudgetExceeded() bool {
	if a == nil || a.tokenStats == nil {
		return false
	}
	ok, remaining := a.tokenStats.CheckBudget()
	if !ok {
		logger.Warn("通知总结Token预算已达到上限，使用原始通知", map[string]interface{}{
			"remaining": remaining,
		})
		return true
	}
	if a.tokenStats.IsBudgetExceeded() {
		logger.Warn("通知总结Token预算已超出，使用原始通知", nil)
		return true
	}
	return false
}

func (a *NotificationSummaryAgent) recordTokenUsage(resp *schema.ChatResponse) error {
	if a.tokenStats == nil || resp == nil {
		return nil
	}
	return a.tokenStats.RecordSimple(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "notificationSummary")
}

func (a *NotificationSummaryAgent) SummarizeJSON(ctx context.Context, data []byte) []byte {
	if a.isTokenBudgetExceeded() {
		return data
	}

	var msg NotificationMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return data
	}
	if msg.Title == "" && msg.Content == "" {
		return data
	}

	title, content, err := a.Summarize(ctx, msg.Title, msg.Content)
	if err != nil {
		logger.Warn("通知总结失败，使用原始通知", map[string]interface{}{
			"error": err.Error(),
			"type":  msg.Type,
		})
		return data
	}

	msg.Title = title
	msg.Content = content
	summaryData, err := json.Marshal(msg)
	if err != nil {
		logger.Warn("总结通知序列化失败，使用原始通知", map[string]interface{}{
			"error": err.Error(),
			"type":  msg.Type,
		})
		return data
	}

	return summaryData
}

func parseNotificationSummary(content string) (*notificationSummaryResult, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		content = content[start : end+1]
	}

	var result notificationSummaryResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("解析通知总结失败: %w", err)
	}

	return &result, nil
}
