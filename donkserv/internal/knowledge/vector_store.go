// knowledge 知识库模块
package knowledge

import (
	"context"
	"fmt"
	"github.com/longstageai/donk/donk/internal/db"
	"github.com/longstageai/donk/donk/pkg/logger"
	"os"
)

// VectorStore 知识库向量存储封装
// 基于CortexDB实现，独立存储知识库向量数据
type VectorStore struct {
	store db.VectorStore
}

// NewVectorStore 创建知识库向量存储
// dataDir: 数据目录（如：./data）
// 返回向量存储实例或错误
func NewVectorStore(dataDir string) (*VectorStore, error) {
	// 确保目录存在
	//dbDir := filepath.Join(dataDir, "knowledge")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建知识库目录失败: %w", err)
	}

	// 创建CortexStore实例
	store, err := db.NewCortexStore(dataDir, "vectors")
	if err != nil {
		return nil, fmt.Errorf("创建向量存储失败: %w", err)
	}

	logger.Info("知识库向量存储初始化成功", map[string]interface{}{
		"path": dataDir,
	})

	return &VectorStore{store: store}, nil
}

// Add 添加向量到知识库
// ctx: 上下文
// vector: 向量数据
// content: 文档内容
// 返回向量ID或错误
func (v *VectorStore) Add(ctx context.Context, vector []float32, content string) (string, error) {
	id, err := v.store.Add(ctx, vector, content)
	if err != nil {
		return "", fmt.Errorf("添加向量失败: %w", err)
	}

	logger.Debug("向量添加成功", map[string]interface{}{
		"vector_id":   id,
		"content_len": len(content),
	})

	return id, nil
}

// Search 向量相似度搜索
// ctx: 上下文
// vector: 查询向量
// limit: 返回结果数量
// 返回搜索结果或错误
func (v *VectorStore) Search(ctx context.Context, vector []float32, limit int) ([]db.SearchResult, error) {
	results, err := v.store.Search(ctx, vector, limit)
	if err != nil {
		return nil, fmt.Errorf("向量搜索失败: %w", err)
	}

	logger.Debug("向量搜索完成", map[string]interface{}{
		"results_count": len(results),
	})

	return results, nil
}

// SearchWithOptions 高级搜索
// ctx: 上下文
// vector: 查询向量
// opts: 搜索选项
// 返回搜索结果或错误
func (v *VectorStore) SearchWithOptions(ctx context.Context, vector []float32, opts db.SearchOptions) ([]db.SearchResult, error) {
	results, err := v.store.SearchWithOptions(ctx, vector, opts)
	if err != nil {
		return nil, fmt.Errorf("高级搜索失败: %w", err)
	}

	logger.Debug("高级搜索完成", map[string]interface{}{
		"results_count": len(results),
		"search_mode":   opts.Mode,
	})

	return results, nil
}

// LexicalSearch 文本关键词搜索
// ctx: 上下文
// keywords: 关键词列表
// limit: 返回结果数量
// 返回搜索结果或错误
func (v *VectorStore) LexicalSearch(ctx context.Context, keywords []string, limit int) ([]db.SearchResult, error) {
	results, err := v.store.LexicalSearch(ctx, keywords, limit)
	if err != nil {
		return nil, fmt.Errorf("文本搜索失败: %w", err)
	}

	logger.Debug("文本搜索完成", map[string]interface{}{
		"results_count": len(results),
		"keywords":      keywords,
	})

	return results, nil
}

// Close 关闭向量存储
// 返回错误
func (v *VectorStore) Close() error {
	if v.store != nil {
		return v.store.Close()
	}
	return nil
}
