package conversation

import (
	"context"
	"time"
)

// IRetriever 检索器接口
type IRetriever interface {
	// Search 检索对话历史
	// ctx: 上下文
	// query: 查询文本
	// topK: 返回数量
	// timeFilter: 时间过滤条件（可选）
	Search(ctx context.Context, query string, topK int, timeFilter *TimeFilter) ([]SearchResult, error)
}

// Search 对话历史检索器
type Search struct {
	store *Store // 对话历史存储
}

// NewSearch 创建对话历史检索器
//
// 参数:
//
//	store: 对话历史存储
//
// 返回:
//
//	*Search: 检索器实例
func NewSearch(store *Store) *Search {
	return &Search{
		store: store,
	}
}

// Search 检索对话历史
// 支持语义搜索和时间范围过滤
//
// 参数:
//
//	ctx: 上下文
//	query: 查询文本
//	topK: 返回数量
//	timeFilter: 时间过滤条件（可选）
//
// 返回:
//
//	[]SearchResult: 搜索结果
//	error: 错误信息
func (s *Search) Search(ctx context.Context, query string, topK int, timeFilter *TimeFilter) ([]SearchResult, error) {
	return s.store.Search(ctx, query, topK, timeFilter)
}

// SearchParams 搜索参数
type SearchParams struct {
	Query     string     // 查询文本
	TopK      int        // 返回数量
	StartTime *time.Time // 开始时间
	EndTime   *time.Time // 结束时间
}

// ParseSearchParams 解析搜索参数
// 从 map 中解析出 SearchParams，支持以下参数：
// - query: string (必需)
// - top_k: float64 (可选，默认2)
// - start_time: string (可选，RFC3339 格式)
// - end_time: string (可选，RFC3339 格式)
func ParseSearchParams(params map[string]any) (*SearchParams, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, ErrInvalidParams
	}

	p := &SearchParams{
		Query: query,
		TopK:  2,
	}

	if topK, ok := params["top_k"].(float64); ok {
		p.TopK = int(topK)
	}

	if startTimeStr, ok := params["start_time"].(string); ok && startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			p.StartTime = &t
		}
	}

	if endTimeStr, ok := params["end_time"].(string); ok && endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			p.EndTime = &t
		}
	}

	return p, nil
}

// TimeFilter 时间过滤条件
type TimeFilter struct {
	StartTime *time.Time // 开始时间
	EndTime   *time.Time // 结束时间
}

// IsMatchByString 使用时间字符串检查是否匹配
//
// 参数:
//
//	endTimeStr: 结束时间字符串
//
// 返回:
//
//	bool: 是否匹配
func (t *TimeFilter) IsMatchByString(endTimeStr string) bool {
	if t == nil {
		return true
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		return true
	}

	// 检查是否在时间范围内
	if t.StartTime != nil && endTime.Before(*t.StartTime) {
		return false
	}
	if t.EndTime != nil && endTime.After(*t.EndTime) {
		return false
	}

	return true
}

// ErrInvalidParams 无效参数错误
var ErrInvalidParams = &SearchError{Code: "INVALID_PARAMS", Message: "无效的搜索参数"}

// SearchError 搜索错误
type SearchError struct {
	Code    string
	Message string
}

func (e *SearchError) Error() string {
	return e.Code + ": " + e.Message
}
