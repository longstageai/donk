package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/db"
	"github.com/longstageai/donk/donk/internal/embedding"
)

// LongMemory 长期记忆存储
// 使用 CortexDB 向量数据库实现持久化存储
// 支持向量搜索和关键词过滤
type LongMemory struct {
	embedder  embedding.Embedder  // 向量嵌入器
	manager   *db.VectorDBManager // 向量数据库管理器
	storeType db.StoreType        // 存储类型
}

// NewLongMemory 创建长期记忆存储
//
// 参数:
//
//	embedder: 向量嵌入器实例
//	manager: 向量数据库管理器
//
// 返回:
//
//	*LongMemory: 长期记忆存储实例
//	error: 错误信息
func NewLongMemory(embedder embedding.Embedder, manager *db.VectorDBManager) (*LongMemory, error) {
	m := &LongMemory{
		embedder:  embedder,
		manager:   manager,
		storeType: db.StoreTypeMemory,
	}

	return m, nil
}

// NewLongMemoryWithDefault 创建长期记忆存储（使用默认路径）
//
// 参数:
//
//	embedder: 向量嵌入器实例
//
// 返回:
//
//	*LongMemory: 长期记忆存储实例
//	error: 错误信息
func NewLongMemoryWithDefault(embedder embedding.Embedder) (*LongMemory, error) {
	manager, err := db.NewVectorDBManager()
	if err != nil {
		return nil, err
	}

	return NewLongMemory(embedder, manager)
}

// Save 保存记忆条目
// 自动生成向量嵌入并持久化存储到 CortexDB
//
// 参数:
//
//	entry: 记忆条目
//
// 返回:
//
//	error: 错误信息
func (m *LongMemory) Save(entry *MemoryEntry) error {
	ctx := context.Background()

	entry.Type = MemoryTypeLong
	entry.Timestamp = time.Now()

	vector, err := m.embedder.GetEmbedding(ctx, entry.Content)
	if err != nil {
		return fmt.Errorf("生成向量失败: %w", err)
	}

	vec32 := make([]float32, len(vector))
	for i, v := range vector {
		vec32[i] = float32(v)
	}

	storeContent := fmt.Sprintf("%s|||%s|||%s|||%s",
		entry.Key,
		entry.Content,
		entry.Summary,
		strings.Join(entry.Tags, ","))

	_, err = m.manager.Add(ctx, m.storeType, vec32, storeContent)
	if err != nil {
		return fmt.Errorf("保存记忆失败: %w", err)
	}

	return nil
}

// Search 搜索记忆
// 支持语义搜索（向量相似度）和关键词过滤
//
// 参数:
//
//	req: 搜索请求，包含查询文本、关键词、标签等
//
// 返回:
//
//	*SearchResult: 搜索结果
//	error: 错误信息
func (m *LongMemory) Search(req SearchRequest) (*SearchResult, error) {
	ctx := context.Background()

	query := req.Context
	if query == "" && len(req.Keywords) > 0 {
		query = strings.Join(req.Keywords, " ")
	}

	vector, err := m.embedder.GetEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	vec32 := make([]float32, len(vector))
	for i, v := range vector {
		vec32[i] = float32(v)
	}

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}
	searchLimit := limit * 3

	// 使用混合搜索 + 混合排序（分数+时间）
	searchOpts := db.SearchOptions{
		Mode:     db.SearchModeHybrid,
		Keywords: req.Keywords,
		Limit:    searchLimit,
	}
	sortOpts := db.SortOptions{
		By:       db.SortByHybrid,
		Order:    db.Descending,
		Priority: db.MatchTypeBoth,
	}

	results, err := m.manager.SearchWithSort(ctx, m.storeType, vec32, searchOpts, sortOpts)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	var entries []MemoryEntry
	for _, r := range results {
		content := r.Content

		parts := strings.SplitN(content, "|||", 4)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		actualContent := parts[1]

		var tags []string
		if len(parts) > 3 && parts[3] != "" {
			tags = strings.Split(parts[3], ",")
		}

		entry := MemoryEntry{
			Key:     key,
			Content: actualContent,
			Summary: parts[2],
			Tags:    tags,
			Type:    MemoryTypeLong,
			Score:   r.Score,
		}

		entries = append(entries, entry)

		if len(entries) >= limit {
			break
		}
	}

	return &SearchResult{
		Total:   len(entries),
		Entries: entries,
	}, nil
}

// Get 根据 key 获取记忆条目
//
// 参数:
//
//	key: 记忆条目唯一标识
//
// 返回:
//
//	*MemoryEntry: 记忆条目（不存在时返回 nil）
//	error: 错误信息
func (m *LongMemory) Get(key string) (*MemoryEntry, error) {
	ctx := context.Background()
	results, err := m.manager.Search(ctx, m.storeType, make([]float32, m.embedder.Dimension()), 1000)
	if err != nil {
		return nil, err
	}

	for _, r := range results {
		parts := strings.SplitN(r.Content, "|||", 4)
		if len(parts) > 0 && parts[0] == key {
			entry := MemoryEntry{
				Key:     parts[0],
				Content: parts[1],
				Summary: parts[2],
			}
			if len(parts) > 3 {
				entry.Tags = strings.Split(parts[3], ",")
			}
			return &entry, nil
		}
	}

	return nil, nil
}

// Delete 删除记忆条目
//
// 参数:
//
//	key: 记忆条目唯一标识
//
// 返回:
//
//	error: 错误信息
func (m *LongMemory) Delete(key string) error {
	return fmt.Errorf("当前版本不支持删除操作")
}

// Count 获取记忆条目数量
//
// 返回:
//
//	int: 记忆条目数量
func (m *LongMemory) Count() int {
	ctx := context.Background()
	results, _ := m.manager.Search(ctx, m.storeType, make([]float32, m.embedder.Dimension()), 10000)
	return len(results)
}

// Close 关闭数据库连接
//
// 返回:
//
//	error: 错误信息
func (m *LongMemory) Close() error {
	// Manager 由上层统一管理，这里不需要关闭
	return nil
}
