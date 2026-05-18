package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

// cortexStore CortexDB 向量存储实现
type cortexStore struct {
	db *cortexdb.DB
}

// NewCortexStore 创建 CortexDB 向量存储
//
// 参数:
//
//	dir: 存储目录
//	name: 存储名称（用于文件名）
//
// 返回:
//
//	VectorStore: 向量存储接口
//	error: 错误信息
func NewCortexStore(dir, name string) (VectorStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	dbPath := filepath.Join(dir, fmt.Sprintf("%s.db", name))

	cfg := cortexdb.DefaultConfig(dbPath)
	db, err := cortexdb.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	return &cortexStore{
		db: db,
	}, nil
}

func (v *cortexStore) Add(ctx context.Context, vector []float32, content string) (id string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("CortexDB添加向量失败，可能是历史向量维度与当前模型不一致: %v", r)
		}
	}()

	if len(vector) == 0 {
		return "", fmt.Errorf("向量不能为空")
	}

	return v.db.Quick().Add(ctx, vector, content)
}

func (v *cortexStore) Search(ctx context.Context, vector []float32, limit int) (items []SearchResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("CortexDB搜索向量失败，可能是历史向量维度与当前模型不一致: %v", r)
		}
	}()

	if len(vector) == 0 {
		return nil, fmt.Errorf("向量不能为空")
	}

	results, err := v.db.Quick().Search(ctx, vector, limit)
	if err != nil {
		return nil, err
	}

	var searchResults []SearchResult
	for _, r := range results {
		searchResults = append(searchResults, SearchResult{
			Content:   r.Content,
			Score:     r.Score,
			MatchType: MatchTypeVector,
			Timestamp: time.Now(), // CortexDB 目前不返回时间，使用当前时间
		})
	}

	return searchResults, nil
}

// SearchWithOptions 高级搜索（支持混合检索）
//
// 根据 opts.Mode 选择不同的检索策略：
// - SearchModeVector: 纯向量相似度搜索
// - SearchModeLexical: 纯文本关键词搜索（FTS5）
// - SearchModeHybrid: 混合搜索（向量+文本，结果合并去重）
func (v *cortexStore) SearchWithOptions(ctx context.Context, vector []float32, opts SearchOptions) ([]SearchResult, error) {
	switch opts.Mode {
	case SearchModeLexical:
		// 纯文本搜索
		return v.LexicalSearch(ctx, opts.Keywords, opts.Limit)

	case SearchModeHybrid:
		// 混合搜索：向量 + 文本
		return v.hybridSearch(ctx, vector, opts.Keywords, opts.Limit)

	case SearchModeVector:
		// 纯向量搜索
		return v.Search(ctx, vector, opts.Limit)

	default:
		// 默认纯向量搜索
		return v.Search(ctx, vector, opts.Limit)
	}
}

// LexicalSearch 纯文本搜索（使用 FTS5）
func (v *cortexStore) LexicalSearch(ctx context.Context, keywords []string, limit int) ([]SearchResult, error) {
	if len(keywords) == 0 {
		return []SearchResult{}, nil
	}

	// 构建 FTS5 查询语句
	// 使用 OR 连接多个关键词
	_ = strings.Join(keywords, " OR ")

	// 调用 CortexDB 的文本搜索
	// 注意：这里使用 db.Quick().Search 配合特殊标记实现
	// 或者使用更高级的 API
	// TODO: 使用 CortexDB 的 FTS5 功能进行文本搜索
	// 目前先使用向量搜索后过滤的方式
	results, err := v.db.Quick().Search(ctx, make([]float32, 1), limit*10)
	if err != nil {
		return nil, err
	}

	// 过滤包含关键词的结果
	var filteredResults []SearchResult
	for _, r := range results {
		contentLower := strings.ToLower(r.Content)
		match := false
		for _, kw := range keywords {
			if strings.Contains(contentLower, strings.ToLower(kw)) {
				match = true
				break
			}
		}
		if match {
			filteredResults = append(filteredResults, SearchResult{
				Content:   r.Content,
				Score:     r.Score,
				MatchType: MatchTypeLexical,
				Timestamp: time.Now(),
			})
			if len(filteredResults) >= limit {
				break
			}
		}
	}

	return filteredResults, nil
}

// hybridSearch 混合搜索（向量 + 文本）
func (v *cortexStore) hybridSearch(ctx context.Context, vector []float32, keywords []string, limit int) ([]SearchResult, error) {
	// 1. 执行向量搜索
	vectorResults, err := v.Search(ctx, vector, limit*2)
	if err != nil {
		return nil, err
	}

	// 2. 执行文本搜索（如果有关键词）
	var lexicalResults []SearchResult
	if len(keywords) > 0 {
		lexicalResults, err = v.LexicalSearch(ctx, keywords, limit*2)
		if err != nil {
			return nil, err
		}
	}

	// 3. 合并结果（去重 + 加权排序）
	return v.mergeResults(vectorResults, lexicalResults, limit), nil
}

// mergeResults 合并向量搜索结果和文本搜索结果
// 使用加权策略：向量分数 * 0.6 + 文本匹配 * 0.4
func (v *cortexStore) mergeResults(vectorResults, lexicalResults []SearchResult, limit int) []SearchResult {
	now := time.Now()
	// 使用 map 去重，key 为 content，value 为结果和匹配类型
	type resultInfo struct {
		score     float64
		matchType MatchType
		timestamp time.Time
	}
	resultMap := make(map[string]resultInfo)

	// 添加向量结果（权重 0.6）
	for _, r := range vectorResults {
		resultMap[r.Content] = resultInfo{
			score:     r.Score * 0.6,
			matchType: MatchTypeVector,
			timestamp: now,
		}
	}

	// 添加文本结果（权重 0.4），如果已存在则累加
	for _, r := range lexicalResults {
		if info, exists := resultMap[r.Content]; exists {
			// 同时匹配向量和文本，分数叠加
			resultMap[r.Content] = resultInfo{
				score:     info.score + r.Score*0.4 + 0.2, // 额外加分
				matchType: MatchTypeBoth,
				timestamp: now,
			}
		} else {
			resultMap[r.Content] = resultInfo{
				score:     r.Score * 0.4,
				matchType: MatchTypeLexical,
				timestamp: now,
			}
		}
	}

	// 转换为切片
	var merged []SearchResult
	for content, info := range resultMap {
		merged = append(merged, SearchResult{
			Content:   content,
			Score:     info.score,
			MatchType: info.matchType,
			Timestamp: info.timestamp,
		})
	}

	// 按分数降序排序
	sortResultsByScore(merged)

	// 限制数量
	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged
}

// sortResultsByScore 按分数降序排序
func sortResultsByScore(results []SearchResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

func (v *cortexStore) Close() error {
	if v.db != nil {
		return v.db.Close()
	}
	return nil
}
