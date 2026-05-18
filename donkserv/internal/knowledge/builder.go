// knowledge 知识库模块
package knowledge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Builder 知识库构建器
// 整合扫描、去重、索引等功能，提供完整的知识库构建流程
type Builder struct {
	config      *BuilderConfig // 配置
	metaStore   Store          // 元数据存储
	vectorStore *VectorStore   // 向量存储
	indexer     *Indexer       // 文档索引器
	scanner     *Scanner       // 文件扫描器
}

// BuilderConfig 构建器配置
type BuilderConfig struct {
	Enabled     bool     // 是否启用
	Interval    int      // 扫描间隔（秒）
	BatchSize   int      // 每批处理数量
	SleepMs     int      // 处理间隔（毫秒）
	MaxDepth    int      // 最大扫描深度
	MaxFileSize int64    // 最大文件大小（字节）
	HotDays     int      // 热数据天数
	WarmDays    int      // 温数据天数
	Directories []string // 扫描目录（空则使用默认）
}

// DefaultBuilderConfig 返回默认配置
func DefaultBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		Enabled:     true,
		Interval:    3600,             // 每小时扫描一次
		BatchSize:   50,               // 每批50个文件
		SleepMs:     100,              // 100ms间隔
		MaxDepth:    3,                // 3层深度
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		HotDays:     7,                // 7天热数据
		WarmDays:    30,               // 30天温数据
		Directories: nil,              // 使用默认目录
	}
}

// NewBuilder 创建知识库构建器
// dataDir: 数据目录
// embedder: 嵌入器
// config: 配置（nil则使用默认）
// 返回构建器实例或错误
func NewBuilder(dataDir string, embedder embedding.Embedder, config *BuilderConfig) (*Builder, error) {
	if config == nil {
		config = DefaultBuilderConfig()
	}

	// 创建数据目录
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 初始化元数据存储
	metaDBPath := filepath.Join(dataDir, "meta.db")
	metaStore, err := NewSQLiteStore(metaDBPath)
	if err != nil {
		return nil, fmt.Errorf("初始化元数据存储失败: %w", err)
	}

	// 检查向量维度是否变化
	currentDim := embedder.Dimension()
	storedDim, err := metaStore.GetVectorDimension()
	if err != nil {
		metaStore.Close()
		return nil, fmt.Errorf("获取存储的向量维度失败: %w", err)
	}

	// 如果维度发生变化，需要清空重建
	if storedDim != 0 && storedDim != currentDim {
		logger.Warn("向量维度发生变化，需要重建知识库", map[string]interface{}{
			"old_dimension": storedDim,
			"new_dimension": currentDim,
		})

		// 清空元数据
		if err := metaStore.ClearAllDocuments(); err != nil {
			metaStore.Close()
			return nil, fmt.Errorf("清空文档记录失败: %w", err)
		}

		// 更新维度记录
		if err := metaStore.SetVectorDimension(currentDim); err != nil {
			metaStore.Close()
			return nil, fmt.Errorf("更新向量维度记录失败: %w", err)
		}

		// 删除向量数据库文件（CortexDB）
		vectorDBPath := filepath.Join(dataDir, "vectors.db")
		if err := os.RemoveAll(vectorDBPath); err != nil {
			logger.Warn("删除旧向量数据库失败", map[string]interface{}{
				"path":  vectorDBPath,
				"error": err.Error(),
			})
		} else {
			logger.Info("已删除旧向量数据库", map[string]interface{}{
				"path": vectorDBPath,
			})
		}
	} else if storedDim == 0 {
		// 首次创建，记录维度
		if err := metaStore.SetVectorDimension(currentDim); err != nil {
			metaStore.Close()
			return nil, fmt.Errorf("设置向量维度记录失败: %w", err)
		}
		logger.Info("记录向量维度", map[string]interface{}{
			"dimension": currentDim,
		})
	}

	// 初始化向量存储
	vectorStore, err := NewVectorStore(dataDir)
	if err != nil {
		metaStore.Close()
		return nil, fmt.Errorf("初始化向量存储失败: %w", err)
	}

	// 初始化扫描器
	scanner := NewScanner(config.MaxDepth, config.MaxFileSize)

	// 初始化索引器
	indexer := NewIndexer(metaStore, vectorStore, embedder)

	builder := &Builder{
		config:      config,
		metaStore:   metaStore,
		vectorStore: vectorStore,
		indexer:     indexer,
		scanner:     scanner,
	}

	logger.Info("知识库构建器初始化成功", map[string]interface{}{
		"data_dir":  dataDir,
		"interval":  config.Interval,
		"dimension": currentDim,
	})

	return builder, nil
}

// Build 执行知识库构建
// ctx: 上下文
// 返回构建结果或错误
func (b *Builder) Build(ctx context.Context) (*BuildResult, error) {
	startTime := time.Now()

	logger.Info("开始知识库构建", map[string]interface{}{
		"start_time": startTime.Format("2006-01-02 15:04:05"),
	})

	result := &BuildResult{
		StartTime: startTime,
	}

	// 1. 扫描目录
	directories := b.config.Directories
	if len(directories) == 0 {
		directories = GetDefaultDirectories()
	}

	scanResults, err := b.scanner.ScanDirectories(directories)
	if err != nil {
		return nil, fmt.Errorf("扫描目录失败: %w", err)
	}

	result.ScannedCount = len(scanResults)

	// 更新扫描时间
	if err := b.metaStore.UpdateScanTime(); err != nil {
		logger.Warn("更新扫描时间失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 2. 筛选新文件
	newFiles, err := FilterNewFiles(scanResults, b.metaStore)
	if err != nil {
		return nil, fmt.Errorf("筛选新文件失败: %w", err)
	}

	result.NewFilesCount = len(newFiles)

	if len(newFiles) == 0 {
		logger.Info("没有新文件需要索引", nil)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		return result, nil
	}

	// 3. 添加到元数据存储（标记为待处理）
	for _, file := range newFiles {
		// 先读取内容计算哈希
		hash, content, err := b.previewFile(file)
		if err != nil {
			logger.Warn("预览文件失败", map[string]interface{}{
				"path":  file.FilePath,
				"error": err.Error(),
			})
			continue
		}

		doc := ScanResultToDocument(file, content, hash)
		if err := b.metaStore.AddDocument(doc); err != nil {
			logger.Warn("添加文档记录失败", map[string]interface{}{
				"path":  file.FilePath,
				"error": err.Error(),
			})
		}
	}

	// 4. 获取待处理文档
	pendingDocs, err := b.metaStore.GetPendingDocuments(b.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("获取待处理文档失败: %w", err)
	}

	// 5. 转换为ScanResult并排序
	var pendingResults []*ScanResult
	for _, doc := range pendingDocs {
		pendingResults = append(pendingResults, &ScanResult{
			FilePath:     doc.FilePath,
			FileSize:     doc.FileSize,
			ModifiedTime: doc.ModifiedTime,
			Extension:    filepath.Ext(doc.FilePath),
		})
	}

	// 6. 构建优先级队列
	queue, err := BuildPriorityQueue(pendingResults, b.metaStore)
	if err != nil {
		return nil, fmt.Errorf("构建优先级队列失败: %w", err)
	}

	// 7. 按优先级索引文档
	successCount, failCount := 0, 0
	for !queue.IsEmpty() {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			logger.Warn("构建任务被取消", nil)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(startTime)
			result.IndexedCount = successCount
			result.FailedCount = failCount + queue.Len()
			return result, ctx.Err()
		default:
		}

		doc, _ := queue.Pop()

		// 索引文档
		if err := b.indexer.IndexDocument(ctx, doc); err != nil {
			logger.Error("索引文档失败", map[string]interface{}{
				"path":  doc.FilePath,
				"error": err.Error(),
			})
			failCount++
		} else {
			successCount++
		}

		// 控制CPU
		if b.config.SleepMs > 0 {
			time.Sleep(time.Duration(b.config.SleepMs) * time.Millisecond)
		}
	}

	// 更新索引时间
	if err := b.metaStore.(*SQLiteStore).UpdateIndexTime(); err != nil {
		logger.Warn("更新索引时间失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 获取最终统计
	stats, err := b.metaStore.GetStats()
	if err != nil {
		logger.Warn("获取统计信息失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	result.IndexedCount = successCount
	result.FailedCount = failCount
	result.Stats = stats

	logger.Info("知识库构建完成", map[string]interface{}{
		"scanned":   result.ScannedCount,
		"new_files": result.NewFilesCount,
		"indexed":   result.IndexedCount,
		"failed":    result.FailedCount,
		"duration":  result.Duration.String(),
	})

	return result, nil
}

// previewFile 预览文件（读取前部分计算哈希）
// result: 扫描结果
// 返回哈希、内容、错误
func (b *Builder) previewFile(result *ScanResult) (string, string, error) {
	// 简化实现：直接读取完整内容
	// 实际项目中可以只读取前N字节用于快速预览
	content, err := os.ReadFile(result.FilePath)
	if err != nil {
		return "", "", err
	}

	text := string(content)
	hash := calculateHash(text)

	return hash, truncateString(text, 1000), nil
}

// GetStats 获取知识库统计
// 返回统计信息或错误
func (b *Builder) GetStats() (*Stats, error) {
	return b.metaStore.GetStats()
}

// Close 关闭构建器
// 返回错误
func (b *Builder) Close() error {
	var errs []error

	if b.metaStore != nil {
		if err := b.metaStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if b.vectorStore != nil {
		if err := b.vectorStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭构建器失败: %v", errs)
	}

	return nil
}

// BuildResult 构建结果
type BuildResult struct {
	StartTime     time.Time     // 开始时间
	EndTime       time.Time     // 结束时间
	Duration      time.Duration // 耗时
	ScannedCount  int           // 扫描文件数
	NewFilesCount int           // 新文件数
	IndexedCount  int           // 成功索引数
	FailedCount   int           // 失败数
	Stats         *Stats        // 统计信息
}

// IsSuccess 检查构建是否成功
func (r *BuildResult) IsSuccess() bool {
	return r.FailedCount == 0
}
