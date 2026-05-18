// knowledge 知识库模块
package knowledge

import (
	"database/sql"
	"fmt"

	"github.com/longstageai/donk/donk/internal/config"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Initializer 知识库初始化器
// 负责初始化并启动知识库模块
type Initializer struct {
	runner    *Runner
	config    *Config
	running   bool
	lastError string
}

// NewInitializer 创建知识库初始化器
// db: 数据库连接
// autoStart: 是否自动启动定时器
// 返回初始化器实例或错误
func NewInitializer(db *sql.DB, autoStart bool) (*Initializer, error) {
	// 从主配置获取知识库配置
	cfg := loadConfigFromFile()

	// 获取数据目录
	paths := config.GetDataPaths()

	// 使用数据库连接创建运行器
	runner, err := NewRunnerWithDB(paths.Knowledge, db, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建知识库运行器失败: %w", err)
	}

	initializer := &Initializer{
		runner: runner,
		config: cfg,
	}

	// 如果设置了自动启动，立即启动定时器
	if autoStart {
		if err := initializer.Start(); err != nil {
			return nil, fmt.Errorf("启动知识库失败: %w", err)
		}
	}

	return initializer, nil
}

// Start 启动知识库模块
// 在后台启动定时构建任务
func (i *Initializer) Start() error {
	if i.runner == nil {
		err := fmt.Errorf("知识库运行器未初始化，无法启动")
		i.lastError = err.Error()
		return err
	}

	if i.running {
		return nil // 已在运行中
	}

	i.runner.Start()
	i.running = true
	i.lastError = ""
	logger.Info("知识库模块已启动", map[string]interface{}{
		"interval": i.config.Interval,
	})
	return nil
}

// Stop 停止知识库模块
func (i *Initializer) Stop() error {
	if i.runner == nil {
		return nil
	}

	if !i.running {
		return nil // 已停止
	}

	i.runner.Stop()
	i.running = false
	logger.Info("知识库模块已停止", nil)
	return nil
}

// IsRunning 返回知识库是否运行中
func (i *Initializer) IsRunning() bool {
	return i.running
}

// GetLastError 获取最后错误信息
func (i *Initializer) GetLastError() string {
	return i.lastError
}

// GetRunner 获取运行器实例
// 返回运行器实例
func (i *Initializer) GetRunner() *Runner {
	return i.runner
}

// loadConfigFromFile 从配置文件加载知识库配置
// 返回配置实例
func loadConfigFromFile() *Config {
	// 目前使用默认配置
	// 后续可以从配置文件或数据库加载
	logger.Debug("使用默认知识库配置", nil)
	return DefaultConfig()
}

// InitAndStart 初始化并启动知识库模块（便捷函数）
// db: 数据库连接
// 返回初始化器实例或错误
func InitAndStart(db *sql.DB) (*Initializer, error) {
	// 自动启动定时器
	return NewInitializer(db, true)
}
