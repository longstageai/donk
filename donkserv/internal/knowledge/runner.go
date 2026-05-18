// knowledge 知识库模块
package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Runner 知识库构建任务运行器
// 定时执行知识库构建任务
type Runner struct {
	builder  *Builder
	config   *Config
	interval time.Duration
	stopCh   chan struct{}
	running  bool
	db       *sql.DB
	dataDir  string
	embedder embedding.Embedder
}

// NewRunner 创建知识库构建运行器
// dataDir: 数据目录
// embedder: 嵌入器
// config: 配置（nil则使用默认）
// 返回运行器实例或错误
func NewRunner(dataDir string, embedder embedding.Embedder, config *Config) (*Runner, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建构建器
	builder, err := NewBuilder(dataDir, embedder, config.ToBuilderConfig())
	if err != nil {
		return nil, fmt.Errorf("创建构建器失败: %w", err)
	}

	runner := &Runner{
		builder:  builder,
		config:   config,
		interval: time.Duration(config.Interval) * time.Second,
		stopCh:   make(chan struct{}),
		dataDir:  dataDir,
		embedder: embedder,
	}

	logger.Info("知识库运行器初始化成功", map[string]interface{}{
		"interval": config.Interval,
		"data_dir": dataDir,
	})

	return runner, nil
}

// NewRunnerWithDB 使用数据库连接创建知识库运行器
// dataDir: 数据目录
// db: 数据库连接
// config: 配置（nil则使用默认）
// 返回运行器实例或错误
func NewRunnerWithDB(dataDir string, db *sql.DB, config *Config) (*Runner, error) {
	// 从 setting 创建 embedder
	embedder, err := CreateEmbedderFromSetting(db)
	if err != nil {
		return nil, fmt.Errorf("创建 embedder 失败: %w", err)
	}

	// 创建运行器
	runner, err := NewRunner(dataDir, embedder, config)
	if err != nil {
		return nil, err
	}

	// 保存数据库连接
	runner.db = db

	return runner, nil
}

// Start 启动定时构建任务
// 在后台goroutine中运行
func (r *Runner) Start() {
	if r.running {
		logger.Warn("知识库运行器已在运行", nil)
		return
	}

	r.running = true

	// 立即执行一次
	go r.runOnce()

	// 启动定时器
	go r.runLoop()

	logger.Info("知识库运行器已启动", map[string]interface{}{
		"interval": r.interval.String(),
	})
}

// runLoop 定时循环
func (r *Runner) runLoop() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.runOnce()
		case <-r.stopCh:
			logger.Info("知识库运行器循环已停止", nil)
			return
		}
	}
}

// runOnce 执行一次构建
func (r *Runner) runOnce() {
	logger.Debug("定时任务触发，检查知识库配置", nil)

	// 检查数据库中的配置，决定是否处理文档
	if r.db != nil {
		enabled, err := r.isKnowledgeEnabled()
		if err != nil {
			logger.Error("获取知识库配置失败", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		if !enabled {
			logger.Debug("知识库已禁用，跳过本次构建", nil)
			return
		}
	}

	logger.Info("开始定时知识库构建", map[string]interface{}{
		"time": time.Now().Format("2006-01-02 15:04:05"),
	})

	// 每次执行时从数据库获取最新的 embedder 配置
	// 因为数据库中的配置可能会变化
	embedder, err := r.createEmbedder()
	if err != nil {
		logger.Error("创建 embedder 失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 使用新的 embedder 创建构建器
	builder, err := NewBuilder(r.dataDir, embedder, r.config.ToBuilderConfig())
	if err != nil {
		logger.Error("创建构建器失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer builder.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := builder.Build(ctx)
	if err != nil {
		logger.Error("知识库构建失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.Info("知识库构建完成", map[string]interface{}{
		"scanned":   result.ScannedCount,
		"new_files": result.NewFilesCount,
		"indexed":   result.IndexedCount,
		"failed":    result.FailedCount,
		"duration":  result.Duration.String(),
	})
}

// createEmbedder 创建 embedder
// 从数据库获取最新的 embedding 配置
func (r *Runner) createEmbedder() (embedding.Embedder, error) {
	if r.db != nil {
		// 从数据库获取最新配置
		return CreateEmbedderFromSetting(r.db)
	}
	// 如果没有数据库连接，使用保存的 embedder
	return r.embedder, nil
}

// isKnowledgeEnabled 从数据库检查知识库是否启用
// 返回是否启用或错误
func (r *Runner) isKnowledgeEnabled() (bool, error) {
	storage := setting.NewStorage(r.db)
	service := setting.NewService(storage)

	cfg, err := service.GetKnowledgeConfig()
	if err != nil {
		return false, err
	}
	if cfg == nil {
		return false, nil
	}

	return cfg.Enabled, nil
}

// Stop 停止定时任务
func (r *Runner) Stop() {
	if !r.running {
		return
	}

	r.running = false
	close(r.stopCh)

	if r.builder != nil {
		if err := r.builder.Close(); err != nil {
			logger.Error("关闭构建器失败", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	logger.Info("知识库运行器已停止", nil)
}

// IsRunning 检查是否正在运行
func (r *Runner) IsRunning() bool {
	return r.running
}

// GetStats 获取知识库统计
func (r *Runner) GetStats() (*Stats, error) {
	if r.builder == nil {
		return nil, fmt.Errorf("构建器未初始化")
	}
	return r.builder.GetStats()
}

// ManualBuild 手动触发构建
// 返回构建结果或错误
func (r *Runner) ManualBuild() (*BuildResult, error) {
	if r.builder == nil {
		return nil, fmt.Errorf("构建器未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	return r.builder.Build(ctx)
}
