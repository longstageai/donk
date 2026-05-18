package token

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/internal/setting"
)

// TokenStats Token消耗统计器
// 按日期存储Token使用记录到数据库，支持预算检查
type TokenStats struct {
	mu            sync.RWMutex
	db            *sql.DB
	limitExceeded bool // 是否已超出限额
}

// DailyUsage 每日Token消耗记录
type DailyUsage struct {
	Date         string    `json:"date" db:"date"`                   // 日期，格式 20260416
	TotalTokens  int       `json:"total_tokens" db:"total_tokens"`   // 总Token数
	InputTokens  int       `json:"input_tokens" db:"input_tokens"`   // 输入Token数
	OutputTokens int       `json:"output_tokens" db:"output_tokens"` // 输出Token数
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`       // 更新时间
}

// NewTokenStats 创建Token统计器
// db: 数据库连接
func NewTokenStats(db *sql.DB) (*TokenStats, error) {
	stats := &TokenStats{
		db:            db,
		limitExceeded: false,
	}
	return stats, nil
}

// Record 记录一次Token消耗
// input: 输入Token数
// output: 输出Token数
func (s *TokenStats) Record(input, output int) error {
	return s.RecordSimple(input, output)
}

// RecordSimple 记录一次Token消耗（简化版本，保持向后兼容）
// input: 输入Token数
// output: 输出Token数
func (s *TokenStats) RecordSimple(input, output int, model ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	date := time.Now().Format("20060102")
	total := input + output

	// 使用 UPSERT 语法更新或插入记录
	query := `
		INSERT INTO token_daily_usage (date, total_tokens, input_tokens, output_tokens, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(date) DO UPDATE SET
			total_tokens = total_tokens + ?,
			input_tokens = input_tokens + ?,
			output_tokens = output_tokens + ?,
			updated_at = datetime('now')`

	_, err := s.db.Exec(query, date, total, input, output, total, input, output)
	if err != nil {
		return fmt.Errorf("记录Token消耗失败: %w", err)
	}

	// 检查是否超限
	s.checkLimit()

	return nil
}

// checkLimit 检查是否超出每日限制
func (s *TokenStats) checkLimit() {
	provider := setting.GetProvider()
	if provider == nil {
		return
	}

	cfg, err := provider.GetConfig()
	if err != nil || cfg == nil {
		return
	}

	// -1 表示不限制
	if cfg.AgentDailyTokenLimit <= 0 {
		s.limitExceeded = false
		return
	}

	usage := s.getTodayUsageUnsafe()
	if usage >= cfg.AgentDailyTokenLimit {
		s.limitExceeded = true
	}
}

// getTodayUsageUnsafe 获取今日累计Token消耗（无锁版本）
func (s *TokenStats) getTodayUsageUnsafe() int {
	date := time.Now().Format("20060102")
	var total int
	err := s.db.QueryRow(
		"SELECT total_tokens FROM token_daily_usage WHERE date = ?",
		date,
	).Scan(&total)
	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		return 0
	}
	return total
}

// GetTodayUsage 获取今日累计Token消耗
func (s *TokenStats) GetTodayUsage() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getTodayUsageUnsafe()
}

// GetDailyUsage 获取指定日期的Token消耗
// date: 日期格式 20060102
func (s *TokenStats) GetDailyUsage(date string) (*DailyUsage, error) {
	var usage DailyUsage
	err := s.db.QueryRow(
		"SELECT date, total_tokens, input_tokens, output_tokens, updated_at FROM token_daily_usage WHERE date = ?",
		date,
	).Scan(&usage.Date, &usage.TotalTokens, &usage.InputTokens, &usage.OutputTokens, &usage.UpdatedAt)
	if err == sql.ErrNoRows {
		return &DailyUsage{
			Date:         date,
			TotalTokens:  0,
			InputTokens:  0,
			OutputTokens: 0,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// GetUsageRange 获取日期范围内的Token消耗
// startDate, endDate: 日期格式 20060102
func (s *TokenStats) GetUsageRange(startDate, endDate string) ([]*DailyUsage, error) {
	rows, err := s.db.Query(
		"SELECT date, total_tokens, input_tokens, output_tokens, updated_at FROM token_daily_usage WHERE date >= ? AND date <= ? ORDER BY date",
		startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []*DailyUsage
	for rows.Next() {
		var usage DailyUsage
		if err := rows.Scan(&usage.Date, &usage.TotalTokens, &usage.InputTokens, &usage.OutputTokens, &usage.UpdatedAt); err != nil {
			return nil, err
		}
		usages = append(usages, &usage)
	}
	return usages, rows.Err()
}

// GetUsageList 获取Token使用记录列表（分页、倒序）
// page: 页码，从1开始
// pageSize: 每页条数
// 返回: 记录列表和总条数
func (s *TokenStats) GetUsageList(page, pageSize int) ([]*DailyUsage, int, error) {
	// 获取总条数
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM token_daily_usage").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 查询分页数据，按日期倒序
	rows, err := s.db.Query(
		"SELECT date, total_tokens, input_tokens, output_tokens, updated_at FROM token_daily_usage ORDER BY date DESC LIMIT ? OFFSET ?",
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var usages []*DailyUsage
	for rows.Next() {
		var usage DailyUsage
		if err := rows.Scan(&usage.Date, &usage.TotalTokens, &usage.InputTokens, &usage.OutputTokens, &usage.UpdatedAt); err != nil {
			return nil, 0, err
		}
		usages = append(usages, &usage)
	}
	return usages, total, rows.Err()
}

// CheckBudget 检查Token预算是否充足
// 返回值：是否还有剩余预算，剩余Token数量
// 当 dailyLimit <= 0 时表示不限制，返回 true, -1
func (s *TokenStats) CheckBudget() (bool, int) {
	provider := setting.GetProvider()
	if provider == nil {
		return true, -1
	}

	cfg, err := provider.GetConfig()
	if err != nil || cfg == nil {
		return true, -1
	}

	// -1 或 0 表示不限制
	if cfg.AgentDailyTokenLimit <= 0 {
		return true, -1
	}

	usage := s.GetTodayUsage()
	remaining := cfg.AgentDailyTokenLimit - usage
	return remaining > 0, remaining
}

// IsBudgetExceeded 检查预算是否已超出
// 当 dailyLimit <= 0 时返回 false（不限制）
func (s *TokenStats) IsBudgetExceeded() bool {
	provider := setting.GetProvider()
	if provider == nil {
		return false
	}

	cfg, err := provider.GetConfig()
	if err != nil || cfg == nil {
		return false
	}

	if cfg.AgentDailyTokenLimit <= 0 {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.limitExceeded
}

// ResetLimitExceeded 重置超限标志（新一天开始时调用）
func (s *TokenStats) ResetLimitExceeded() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limitExceeded = false
}

// GetRemainingBudget 获取剩余可用预算
// 返回 -1 表示不限制
func (s *TokenStats) GetRemainingBudget() int {
	_, remaining := s.CheckBudget()
	return remaining
}

// GetDailyLimit 获取每日限额
// 从 setting 模块读取
func (s *TokenStats) GetDailyLimit() int {
	provider := setting.GetProvider()
	if provider == nil {
		return -1
	}

	cfg, err := provider.GetConfig()
	if err != nil || cfg == nil {
		return -1
	}

	return cfg.AgentDailyTokenLimit
}

// GetTodayUsageDetail 获取今日详细的Token消耗
func (s *TokenStats) GetTodayUsageDetail() (*DailyUsage, error) {
	date := time.Now().Format("20060102")
	return s.GetDailyUsage(date)
}
