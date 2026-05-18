package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/db"
	"github.com/longstageai/donk/donk/internal/embedding"
)

// ConversationChunk 元数据
// 存储在向量的 content 字段中
type ChunkMetadata struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Content        string `json:"content"`
	Index          int    `json:"index"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	Date           string `json:"date"`
	YearMonth      string `json:"year_month"`
	MessageCount   int    `json:"message_count"`
}

// Store 对话历史向量存储
// 基于 CortexDB 实现对话历史的向量存储和检索
type Store struct {
	embedder  embedding.Embedder  // 向量嵌入模型
	manager   *db.VectorDBManager // 向量数据库管理器
	storeType db.StoreType        // 存储类型
	chunker   *TextSplitter       // 文本切片器
}

// NewStore 创建对话历史向量存储
//
// 参数:
//
//	embedder: 向量嵌入模型
//	manager: 向量数据库管理器
//	chunker: 文本切片器（可选，为nil使用默认）
//
// 返回:
//
//	*Store: 对话历史存储实例
//	error: 错误信息
func NewStore(embedder embedding.Embedder, manager *db.VectorDBManager, chunker *TextSplitter) (*Store, error) {
	if chunker == nil {
		chunker = NewTextSplitter(DefaultTextSplitterConfig)
	}

	ks := &Store{
		embedder:  embedder,
		manager:   manager,
		storeType: db.StoreTypeConversation,
		chunker:   chunker,
	}

	return ks, nil
}

// NewStoreWithDefault 创建对话历史向量存储（使用默认路径）
//
// 参数:
//
//	embedder: 向量嵌入模型
//	chunker: 文本切片器（可选，为nil使用默认）
//
// 返回:
//
//	*Store: 对话历史存储实例
//	error: 错误信息
func NewStoreWithDefault(embedder embedding.Embedder, chunker *TextSplitter) (*Store, error) {
	manager, err := db.NewVectorDBManagerWithEmbedder(embedder)
	if err != nil {
		return nil, err
	}

	return NewStore(embedder, manager, chunker)
}

// AddConversation 添加对话到存储
// 对话内容会被切片、向量化后存储到向量数据库
//
// 参数:
//
//	ctx: 上下文
//	conversationID: 对话ID
//	messages: 格式化后的消息列表
//	startTime: 对话开始时间
//	endTime: 对话结束时间
//
// 返回:
//
//	error: 错误信息
func (s *Store) AddConversation(ctx context.Context, conversationID string, messages string, startTime, endTime time.Time) error {
	chunks := s.chunker.Split(messages)
	if len(chunks) == 0 {
		return nil
	}

	for i, chunk := range chunks {
		emb, err := s.embedder.GetEmbedding(ctx, chunk)
		if err != nil {
			return fmt.Errorf("生成向量失败: %w", err)
		}
		vec := make([]float32, len(emb))
		for j, v := range emb {
			vec[j] = float32(v)
		}

		chunkID := fmt.Sprintf("%s_%d", conversationID, i)
		date := endTime.Format("2006-01-02")
		yearMonth := endTime.Format("2006-01")

		metadata := ChunkMetadata{
			ID:             chunkID,
			ConversationID: conversationID,
			Content:        chunk,
			Index:          i,
			StartTime:      startTime.Format(time.RFC3339),
			EndTime:        endTime.Format(time.RFC3339),
			Date:           date,
			YearMonth:      yearMonth,
			MessageCount:   0,
		}

		content, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("序列化元数据失败: %w", err)
		}

		_, err = s.manager.Add(ctx, s.storeType, vec, string(content))
		if err != nil {
			return fmt.Errorf("存储向量失败: %w", err)
		}
	}

	return nil
}

// SearchResult 搜索结果
type SearchResult struct {
	Content        string
	Score          float64
	ConversationID string
	Index          int
	StartTime      time.Time
	EndTime        time.Time
	Date           string
}

// Search 搜索对话历史
// 支持语义搜索和时间范围过滤
//
// 参数:
//
//	ctx: 上下文
//	query: 查询文本
//	topK: 返回数量
//	timeFilter: 时间过滤（可选）
//
// 返回:
//
//	[]SearchResult: 搜索结果列表
//	error: 错误信息
func (s *Store) Search(ctx context.Context, query string, topK int, timeFilter *TimeFilter) ([]SearchResult, error) {
	vector, err := s.embedder.GetEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	vec32 := make([]float32, len(vector))
	for i, v := range vector {
		vec32[i] = float32(v)
	}

	if topK == 0 {
		topK = 5
	}
	searchLimit := topK * 3

	// 使用纯向量搜索 + 分数排序
	searchOpts := db.SearchOptions{
		Mode:  db.SearchModeVector,
		Limit: searchLimit,
	}
	sortOpts := db.SortOptions{
		By:    db.SortByScore,
		Order: db.Descending,
	}

	results, err := s.manager.SearchWithSort(ctx, s.storeType, vec32, searchOpts, sortOpts)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	var searchResults []SearchResult
	for _, r := range results {
		var metadata ChunkMetadata
		err := json.Unmarshal([]byte(r.Content), &metadata)
		if err != nil {
			continue
		}

		if timeFilter != nil && !timeFilter.IsMatchByString(metadata.EndTime) {
			continue
		}

		startTime, _ := time.Parse(time.RFC3339, metadata.StartTime)
		endTime, _ := time.Parse(time.RFC3339, metadata.EndTime)

		result := SearchResult{
			Content:        metadata.Content,
			Score:          r.Score,
			ConversationID: metadata.ConversationID,
			Index:          metadata.Index,
			StartTime:      startTime,
			EndTime:        endTime,
			Date:           metadata.Date,
		}

		searchResults = append(searchResults, result)

		if len(searchResults) >= topK {
			break
		}
	}

	return searchResults, nil
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	// Manager 由上层统一管理，这里不需要关闭
	return nil
}
