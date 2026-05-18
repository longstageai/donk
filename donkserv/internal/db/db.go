package db

import (
	"context"
	"math"
	"sort"
	"time"
)

// SearchMode 搜索模式
type SearchMode int

const (
	SearchModeVector  SearchMode = iota // 纯向量搜索
	SearchModeLexical                   // 纯文本搜索（FTS5）
	SearchModeHybrid                    // 混合搜索（向量+文本）
)

// SearchOptions 搜索选项
type SearchOptions struct {
	Mode     SearchMode // 搜索模式
	Keywords []string   // 关键词（用于Lexical和Hybrid模式）
	Limit    int        // 返回数量
}

// MatchType 匹配类型
type MatchType int

const (
	MatchTypeVector  MatchType = iota // 仅向量匹配
	MatchTypeLexical                  // 仅文本匹配
	MatchTypeBoth                     // 同时匹配（加权）
)

// SortBy 排序字段
type SortBy int

const (
	SortByScore  SortBy = iota // 按分数
	SortByTime                 // 按时间
	SortByHybrid               // 混合（分数+时间衰减）
)

// SortOrder 排序方向
type SortOrder int

const (
	Descending SortOrder = iota // 降序
	Ascending                   // 升序
)

// SortOptions 排序选项
type SortOptions struct {
	By       SortBy    // 排序字段
	Order    SortOrder // 排序方向
	Priority MatchType // 优先类型（Both优先等）
}

// DefaultSortOptions 返回默认排序选项
func DefaultSortOptions(storeType StoreType) SortOptions {
	switch storeType {
	case StoreTypeMemory:
		// 长期记忆：混合排序（重要且新的优先）
		return SortOptions{
			By:       SortByHybrid,
			Order:    Descending,
			Priority: MatchTypeBoth,
		}
	case StoreTypeConversation:
		// 对话历史：分数优先（内容准确性优先）
		return SortOptions{
			By:       SortByScore,
			Order:    Descending,
			Priority: MatchTypeVector,
		}
	default:
		return SortOptions{
			By:    SortByScore,
			Order: Descending,
		}
	}
}

// VectorStore 向量存储接口
// 定义向量存储和检索的统一接口，便于切换不同的向量数据库实现
type VectorStore interface {
	// Add 添加向量到存储
	//
	// 参数:
	//   - ctx: 上下文
	//   - vector: 向量 ([]float32)
	//   - content: 内容（任意文本，用于检索）
	//
	// 返回:
	//   - id: 向量ID
	//   - error: 错误信息
	Add(ctx context.Context, vector []float32, content string) (string, error)

	// Search 向量相似度搜索
	//
	// 参数:
	//   - ctx: 上下文
	//   - vector: 查询向量
	//   - limit: 返回结果数量
	//
	// 返回:
	//   - []SearchResult: 搜索结果
	//   - error: 错误信息
	Search(ctx context.Context, vector []float32, limit int) ([]SearchResult, error)

	// SearchWithOptions 高级搜索（支持混合检索）
	//
	// 参数:
	//   - ctx: 上下文
	//   - vector: 查询向量
	//   - opts: 搜索选项
	//
	// 返回:
	//   - []SearchResult: 搜索结果
	//   - error: 错误信息
	SearchWithOptions(ctx context.Context, vector []float32, opts SearchOptions) ([]SearchResult, error)

	// LexicalSearch 纯文本搜索（关键词匹配）
	//
	// 参数:
	//   - ctx: 上下文
	//   - keywords: 关键词列表
	//   - limit: 返回结果数量
	//
	// 返回:
	//   - []SearchResult: 搜索结果
	//   - error: 错误信息
	LexicalSearch(ctx context.Context, keywords []string, limit int) ([]SearchResult, error)

	// Close 关闭数据库连接
	Close() error
}

// SearchResult 向量搜索结果
type SearchResult struct {
	Content   string    // 存储的内容
	Score     float64   // 相似度分数
	MatchType MatchType // 匹配类型
	Timestamp time.Time // 时间戳（用于时间排序）
}

// SortResults 对搜索结果进行排序
//
// 参数:
//   - results: 搜索结果切片
//   - opts: 排序选项
func SortResults(results []SearchResult, opts SortOptions) {
	switch opts.By {
	case SortByScore:
		sortByScore(results, opts.Order)
	case SortByTime:
		sortByTime(results, opts.Order)
	case SortByHybrid:
		sortByHybrid(results, opts.Order)
	}

	// 匹配类型优先级调整
	if opts.Priority == MatchTypeBoth {
		prioritizeBothMatches(results)
	}
}

// sortByScore 按分数排序
func sortByScore(results []SearchResult, order SortOrder) {
	sort.Slice(results, func(i, j int) bool {
		if order == Descending {
			return results[i].Score > results[j].Score
		}
		return results[i].Score < results[j].Score
	})
}

// sortByTime 按时间排序
func sortByTime(results []SearchResult, order SortOrder) {
	sort.Slice(results, func(i, j int) bool {
		if order == Descending {
			return results[i].Timestamp.After(results[j].Timestamp)
		}
		return results[i].Timestamp.Before(results[j].Timestamp)
	})
}

// sortByHybrid 混合排序（分数 + 时间衰减）
func sortByHybrid(results []SearchResult, order SortOrder) {
	now := time.Now()
	sort.Slice(results, func(i, j int) bool {
		// 时间衰减因子：越旧衰减越多
		scoreI := results[i].Score * timeDecay(results[i].Timestamp, now)
		scoreJ := results[j].Score * timeDecay(results[j].Timestamp, now)

		if order == Descending {
			return scoreI > scoreJ
		}
		return scoreI < scoreJ
	})
}

// timeDecay 时间衰减函数（指数衰减）
// 半衰期：30天
func timeDecay(t, now time.Time) float64 {
	if t.IsZero() {
		return 1.0 // 无时间信息不衰减
	}
	days := now.Sub(t).Hours() / 24
	return math.Exp(-days / 30)
}

// prioritizeBothMatches 提升同时匹配向量和文本的结果
func prioritizeBothMatches(results []SearchResult) {
	// 将 MatchTypeBoth 的结果排在前面（保持相对顺序）
	stableSortByMatchType(results)
}

// stableSortByMatchType 稳定排序，Both 类型优先
func stableSortByMatchType(results []SearchResult) {
	type indexedResult struct {
		index  int
		result SearchResult
		isBoth bool
	}

	indexed := make([]indexedResult, len(results))
	for i, r := range results {
		indexed[i] = indexedResult{
			index:  i,
			result: r,
			isBoth: r.MatchType == MatchTypeBoth,
		}
	}

	sort.SliceStable(indexed, func(i, j int) bool {
		// Both 类型排在前面
		if indexed[i].isBoth && !indexed[j].isBoth {
			return true
		}
		if !indexed[i].isBoth && indexed[j].isBoth {
			return false
		}
		// 相同类型保持原顺序（稳定排序）
		return indexed[i].index < indexed[j].index
	})

	// 写回原切片
	for i, idx := range indexed {
		results[i] = idx.result
	}
}
