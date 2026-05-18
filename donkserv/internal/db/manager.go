package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/longstageai/donk/donk/internal/config"
	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// StoreType 向量存储类型
type StoreType string

const (
	StoreTypeMemory       StoreType = "memory"       // 长期记忆
	StoreTypeConversation StoreType = "conversation" // 对话历史
)

// VectorDBManager 向量数据库管理器
// 统一管理多个向量存储实例
type VectorDBManager struct {
	mu        sync.RWMutex
	stores    map[StoreType]VectorStore // 存储实例映射
	paths     config.DataPaths          // 数据路径配置
	dimension int                       // 当前向量维度
}

// dimensionFileName 维度记录文件名
const dimensionFileName = "vector_dimension.txt"

// NewVectorDBManager 创建向量数据库管理器
// 使用统一的数据路径配置
//
// 返回:
//   - *VectorDBManager: 管理器实例
//   - error: 错误信息
func NewVectorDBManager() (*VectorDBManager, error) {
	paths := config.GetDataPaths()
	return NewVectorDBManagerWithPaths(paths)
}

// NewVectorDBManagerWithEmbedder 使用指定 embedder 创建向量数据库管理器
// 会自动检测维度变化并在必要时重建存储
//
// 参数:
//   - embedder: 向量嵌入器
//
// 返回:
//   - *VectorDBManager: 管理器实例
//   - error: 错误信息
func NewVectorDBManagerWithEmbedder(embedder embedding.Embedder) (*VectorDBManager, error) {
	paths := config.GetDataPaths()
	return NewVectorDBManagerWithPathsAndEmbedder(paths, embedder)
}

// NewVectorDBManagerWithPaths 使用指定路径创建向量数据库管理器（不带 embedder）
// 注意：此方式不会进行维度检测，建议使用 NewVectorDBManagerWithEmbedder
//
// 参数:
//   - paths: 数据路径配置
//
// 返回:
//   - *VectorDBManager: 管理器实例
//   - error: 错误信息
func NewVectorDBManagerWithPaths(paths config.DataPaths) (*VectorDBManager, error) {
	manager := &VectorDBManager{
		stores: make(map[StoreType]VectorStore),
		paths:  paths,
	}

	// 初始化默认存储
	if err := manager.InitStore(StoreTypeMemory, paths.MemoryVectorDB); err != nil {
		return nil, fmt.Errorf("初始化记忆存储失败: %w", err)
	}
	if err := manager.InitStore(StoreTypeConversation, paths.ConversationDB); err != nil {
		return nil, fmt.Errorf("初始化对话历史存储失败: %w", err)
	}

	logger.Info("VectorDBManager 初始化完成", map[string]interface{}{
		"stores": len(manager.stores),
	})

	return manager, nil
}

// NewVectorDBManagerWithPathsAndEmbedder 使用指定路径和 embedder 创建管理器
// 会自动检测维度变化并在必要时重建存储
//
// 参数:
//   - paths: 数据路径配置
//   - embedder: 向量嵌入器
//
// 返回:
//   - *VectorDBManager: 管理器实例
//   - error: 错误信息
func NewVectorDBManagerWithPathsAndEmbedder(paths config.DataPaths, embedder embedding.Embedder) (*VectorDBManager, error) {
	// 确保数据目录存在
	if err := config.InitDataDir(); err != nil {
		return nil, err
	}

	currentDim := embedder.Dimension()

	// 检查维度是否变化
	dimFilePath := filepath.Join(paths.DataDir, dimensionFileName)
	storedDim, err := readDimensionFile(dimFilePath)
	if err != nil {
		// 文件不存在或读取失败，视为首次创建
		storedDim = 0
	}

	manager := &VectorDBManager{
		stores:    make(map[StoreType]VectorStore),
		paths:     paths,
		dimension: currentDim,
	}

	// 如果维度发生变化，需要删除旧存储
	if storedDim != 0 && storedDim != currentDim {
		logger.Warn("向量维度发生变化，需要重建向量存储", map[string]interface{}{
			"old_dimension": storedDim,
			"new_dimension": currentDim,
		})

		// 删除旧的存储文件
		if err := manager.removeStoreFiles(); err != nil {
			logger.Warn("删除旧存储文件失败", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// 记录新维度
	if err := writeDimensionFile(dimFilePath, currentDim); err != nil {
		logger.Warn("记录向量维度失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 初始化默认存储
	if err := manager.InitStore(StoreTypeMemory, paths.MemoryVectorDB); err != nil {
		return nil, fmt.Errorf("初始化记忆存储失败: %w", err)
	}
	if err := manager.InitStore(StoreTypeConversation, paths.ConversationDB); err != nil {
		return nil, fmt.Errorf("初始化对话历史存储失败: %w", err)
	}

	logger.Info("VectorDBManager 初始化完成", map[string]interface{}{
		"stores":    len(manager.stores),
		"dimension": currentDim,
	})

	return manager, nil
}

// readDimensionFile 读取维度记录文件
func readDimensionFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var dim int
	_, err = fmt.Sscanf(string(data), "%d", &dim)
	return dim, err
}

// writeDimensionFile 写入维度记录文件
func writeDimensionFile(path string, dimension int) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", dimension)), 0644)
}

// removeStoreFiles 删除所有存储文件
func (m *VectorDBManager) removeStoreFiles() error {
	files := []string{
		m.paths.MemoryVectorDB,
		m.paths.ConversationDB,
	}
	for _, file := range files {
		patterns := []string{file, file + "-*", file + ".*"}
		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				logger.Warn("匹配存储文件失败", map[string]interface{}{
					"pattern": pattern,
					"error":   err.Error(),
				})
				continue
			}
			for _, match := range matches {
				if err := os.RemoveAll(match); err != nil {
					logger.Warn("删除存储文件失败", map[string]interface{}{
						"path":  match,
						"error": err.Error(),
					})
				} else {
					logger.Info("已删除旧存储文件", map[string]interface{}{
						"path": match,
					})
				}
			}
		}
	}
	return nil
}

// InitStore 初始化指定类型的存储
//
// 参数:
//   - storeType: 存储类型
//   - dbPath: 数据库文件路径
//
// 返回:
//   - error: 错误信息
func (m *VectorDBManager) InitStore(storeType StoreType, dbPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.stores[storeType]; exists {
		return fmt.Errorf("存储类型 %s 已存在", storeType)
	}

	dbDir := filepath.Dir(dbPath)
	name := filepath.Base(dbPath)
	// 去掉 .db 后缀
	name = name[:len(name)-len(filepath.Ext(name))]

	store, err := NewCortexStore(dbDir, name)
	if err != nil {
		return err
	}

	m.stores[storeType] = store
	logger.Info("向量存储初始化成功", map[string]interface{}{
		"type": storeType,
		"path": dbPath,
	})

	return nil
}

func (m *VectorDBManager) validateVectorDimension(vector []float32) error {
	if len(vector) == 0 {
		return fmt.Errorf("向量不能为空")
	}
	if m.dimension > 0 && len(vector) != m.dimension {
		return fmt.Errorf("向量维度不匹配: 当前向量维度=%d, 数据库维度=%d", len(vector), m.dimension)
	}
	return nil
}

// GetStore 获取指定类型的存储实例
//
// 参数:
//   - storeType: 存储类型
//
// 返回:
//   - VectorStore: 存储实例
//   - error: 错误信息
func (m *VectorDBManager) GetStore(storeType StoreType) (VectorStore, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store, exists := m.stores[storeType]
	if !exists {
		return nil, fmt.Errorf("存储类型 %s 不存在", storeType)
	}

	return store, nil
}

// Add 向指定存储添加向量
//
// 参数:
//   - ctx: 上下文
//   - storeType: 存储类型
//   - vector: 向量
//   - content: 内容
//
// 返回:
//   - string: ID
//   - error: 错误信息
func (m *VectorDBManager) Add(ctx context.Context, storeType StoreType, vector []float32, content string) (string, error) {
	if err := m.validateVectorDimension(vector); err != nil {
		return "", err
	}

	store, err := m.GetStore(storeType)
	if err != nil {
		return "", err
	}
	return store.Add(ctx, vector, content)
}

// Search 在指定存储中搜索
//
// 参数:
//   - ctx: 上下文
//   - storeType: 存储类型
//   - vector: 查询向量
//   - limit: 返回数量
//
// 返回:
//   - []SearchResult: 搜索结果
//   - error: 错误信息
func (m *VectorDBManager) Search(ctx context.Context, storeType StoreType, vector []float32, limit int) ([]SearchResult, error) {
	if err := m.validateVectorDimension(vector); err != nil {
		return nil, err
	}

	store, err := m.GetStore(storeType)
	if err != nil {
		return nil, err
	}
	return store.Search(ctx, vector, limit)
}

// SearchWithOptions 高级搜索（支持混合检索）
//
// 参数:
//   - ctx: 上下文
//   - storeType: 存储类型
//   - vector: 查询向量
//   - opts: 搜索选项
//
// 返回:
//   - []SearchResult: 搜索结果
//   - error: 错误信息
func (m *VectorDBManager) SearchWithOptions(ctx context.Context, storeType StoreType, vector []float32, opts SearchOptions) ([]SearchResult, error) {
	if opts.Mode != SearchModeLexical {
		if err := m.validateVectorDimension(vector); err != nil {
			return nil, err
		}
	}

	store, err := m.GetStore(storeType)
	if err != nil {
		return nil, err
	}
	return store.SearchWithOptions(ctx, vector, opts)
}

// LexicalSearch 纯文本搜索
//
// 参数:
//   - ctx: 上下文
//   - storeType: 存储类型
//   - keywords: 关键词列表
//   - limit: 返回数量
//
// 返回:
//   - []SearchResult: 搜索结果
//   - error: 错误信息
func (m *VectorDBManager) LexicalSearch(ctx context.Context, storeType StoreType, keywords []string, limit int) ([]SearchResult, error) {
	store, err := m.GetStore(storeType)
	if err != nil {
		return nil, err
	}
	return store.LexicalSearch(ctx, keywords, limit)
}

// MultiSearchResult 多存储搜索结果
type MultiSearchResult struct {
	StoreType StoreType      // 来源存储类型
	Results   []SearchResult // 搜索结果
}

// SearchAll 在所有存储中搜索（并行）
//
// 参数:
//   - ctx: 上下文
//   - vector: 查询向量
//   - limitPerStore: 每个存储返回的数量
//
// 返回:
//   - []MultiSearchResult: 按存储分组的结果
//   - error: 错误信息
func (m *VectorDBManager) SearchAll(ctx context.Context, vector []float32, limitPerStore int) ([]MultiSearchResult, error) {
	m.mu.RLock()
	storeTypes := make([]StoreType, 0, len(m.stores))
	for t := range m.stores {
		storeTypes = append(storeTypes, t)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	resultChan := make(chan MultiSearchResult, len(storeTypes))
	errChan := make(chan error, len(storeTypes))

	// 并行搜索所有存储
	for _, storeType := range storeTypes {
		wg.Add(1)
		go func(st StoreType) {
			defer wg.Done()

			results, err := m.Search(ctx, st, vector, limitPerStore)
			if err != nil {
				errChan <- fmt.Errorf("搜索 %s 失败: %w", st, err)
				return
			}

			resultChan <- MultiSearchResult{
				StoreType: st,
				Results:   results,
			}
		}(storeType)
	}

	// 等待所有搜索完成
	go func() {
		wg.Wait()
		close(resultChan)
		close(errChan)
	}()

	// 收集结果
	var multiResults []MultiSearchResult
	for result := range resultChan {
		multiResults = append(multiResults, result)
	}

	// 检查错误
	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	return multiResults, nil
}

// SearchAllMerged 在所有存储中搜索并合并排序
//
// 参数:
//   - ctx: 上下文
//   - vector: 查询向量
//   - totalLimit: 总共返回的数量
//
// 返回:
//   - []SearchResult: 合并后的搜索结果（按相似度排序）
//   - error: 错误信息
func (m *VectorDBManager) SearchAllMerged(ctx context.Context, vector []float32, totalLimit int) ([]SearchResult, error) {
	multiResults, err := m.SearchAll(ctx, vector, totalLimit)
	if err != nil {
		return nil, err
	}

	// 合并所有结果
	var allResults []SearchResult
	for _, mr := range multiResults {
		// 添加来源标记到内容
		for i := range mr.Results {
			mr.Results[i].Content = fmt.Sprintf("[%s] %s", mr.StoreType, mr.Results[i].Content)
		}
		allResults = append(allResults, mr.Results...)
	}

	// 按相似度排序
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// 限制总数
	if len(allResults) > totalLimit {
		allResults = allResults[:totalLimit]
	}

	return allResults, nil
}

// SmartSearch 智能搜索（根据存储类型自动选择最佳搜索策略）
//
// 参数:
//   - ctx: 上下文
//   - storeType: 存储类型
//   - vector: 查询向量
//   - query: 查询文本
//   - keywords: 关键词列表
//   - limit: 返回数量
//
// 返回:
//   - []SearchResult: 搜索结果
//   - error: 错误信息
func (m *VectorDBManager) SmartSearch(ctx context.Context, storeType StoreType, vector []float32, query string, keywords []string, limit int) ([]SearchResult, error) {
	var opts SearchOptions

	switch storeType {
	case StoreTypeMemory:
		// 长期记忆：优先混合搜索
		opts = SearchOptions{
			Mode:     SearchModeHybrid,
			Keywords: keywords,
			Limit:    limit * 2,
		}
	case StoreTypeConversation:
		// 对话历史：根据是否有关键词选择
		if len(keywords) > 0 {
			opts = SearchOptions{
				Mode:     SearchModeHybrid,
				Keywords: keywords,
				Limit:    limit * 2,
			}
		} else {
			opts = SearchOptions{
				Mode:  SearchModeVector,
				Limit: limit * 2,
			}
		}
	default:
		opts = SearchOptions{
			Mode:  SearchModeVector,
			Limit: limit * 2,
		}
	}

	results, err := m.SearchWithOptions(ctx, storeType, vector, opts)
	if err != nil {
		return nil, err
	}

	// 应用默认排序
	sortOpts := DefaultSortOptions(storeType)
	SortResults(results, sortOpts)

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// SearchWithSort 带排序的搜索
//
// 参数:
//   - ctx: 上下文
//   - storeType: 存储类型
//   - vector: 查询向量
//   - opts: 搜索选项
//   - sortOpts: 排序选项
//
// 返回:
//   - []SearchResult: 搜索结果
//   - error: 错误信息
func (m *VectorDBManager) SearchWithSort(ctx context.Context, storeType StoreType, vector []float32, opts SearchOptions, sortOpts SortOptions) ([]SearchResult, error) {
	results, err := m.SearchWithOptions(ctx, storeType, vector, opts)
	if err != nil {
		return nil, err
	}

	// 应用排序
	SortResults(results, sortOpts)

	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// SmartSearchWithSort 智能搜索并排序
//
// 参数:
//   - ctx: 上下文
//   - storeType: 存储类型
//   - vector: 查询向量
//   - query: 查询文本
//   - keywords: 关键词列表
//   - limit: 返回数量
//   - sortOpts: 排序选项（可选，传nil使用默认）
//
// 返回:
//   - []SearchResult: 搜索结果
//   - error: 错误信息
func (m *VectorDBManager) SmartSearchWithSort(ctx context.Context, storeType StoreType, vector []float32, query string, keywords []string, limit int, sortOpts *SortOptions) ([]SearchResult, error) {
	// 先搜索
	results, err := m.SmartSearch(ctx, storeType, vector, query, keywords, limit*2)
	if err != nil {
		return nil, err
	}

	// 使用指定排序或默认排序
	var opts SortOptions
	if sortOpts != nil {
		opts = *sortOpts
	} else {
		opts = DefaultSortOptions(storeType)
	}

	SortResults(results, opts)

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Close 关闭所有存储
func (m *VectorDBManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for storeType, store := range m.stores {
		if err := store.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 %s 失败: %w", storeType, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭向量存储时发生错误: %v", errs)
	}

	return nil
}

// Stats 获取存储统计信息
func (m *VectorDBManager) Stats() map[StoreType]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[StoreType]bool)
	for t := range m.stores {
		stats[t] = true
	}
	return stats
}
