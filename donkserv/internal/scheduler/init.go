package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/config"

	_ "github.com/mattn/go-sqlite3"
)

// Config 调度器配置
type Config struct {
	DBPath     string // 数据库文件路径
	Workers    int    // 并发 worker 数量
	EnableLog  bool   // 启用日志订阅者
	WebhookURL string // Webhook URL（可选）
}

// DefaultConfig 返回默认配置
// 使用统一的数据路径
func DefaultConfig() *Config {
	paths := config.GetDataPaths()
	return &Config{
		DBPath:    paths.MainDB, // 使用统一的 SQLite 主库
		Workers:   5,
		EnableLog: true,
	}
}

// New 创建并初始化调度器
// 根据配置创建数据库、仓储、调度器等组件
func New(config *Config) (*Scheduler, *SQLiteTaskRepository, error) {
	// 打开数据库
	db, err := sql.Open("sqlite3", config.DBPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 创建仓储
	repo := NewSQLiteTaskRepository(db)

	// 创建执行器工厂
	factory := NewDefaultExecutorFactory()

	// 创建事件总线
	var eventBus *EventBus
	if config.EnableLog || config.WebhookURL != "" {
		eventBus = NewEventBus()
		if config.EnableLog {
			eventBus.Subscribe(NewLogSubscriber())
		}
		if config.WebhookURL != "" {
			eventBus.Subscribe(NewWebhookSubscriber(config.WebhookURL))
		}
	}

	// 创建调度器
	scheduler := NewScheduler(nil, repo, factory, nil,
		WithWorkers(config.Workers),
		WithEventBus(eventBus),
	)

	return scheduler, repo, nil
}

// StartWithContext 启动调度器并返回上下文
// ctx 用于控制调度器的生命周期
func StartWithContext(ctx context.Context, config *Config) (*Scheduler, *SQLiteTaskRepository, error) {
	scheduler, repo, err := New(config)
	if err != nil {
		return nil, nil, err
	}

	// 启动调度器
	if err := scheduler.Start(); err != nil {
		return nil, nil, err
	}

	// 等待上下文取消
	go func() {
		<-ctx.Done()
		scheduler.Stop()
		db := repo.db
		if db != nil {
			db.Close()
		}
	}()

	return scheduler, repo, nil
}

// Start 启动调度器（简化版本）
func Start(config *Config) (*Scheduler, error) {
	scheduler, _, err := StartWithContext(context.Background(), config)
	return scheduler, err
}

// StartWithExistingDB 使用已存在的数据库创建调度器
// db: 已打开的数据库连接
// workers: 并发 worker 数量
func StartWithExistingDB(db *sql.DB, workers int) (*Scheduler, error) {

	// 创建仓储
	repo := NewSQLiteTaskRepository(db)

	// 创建执行器工厂
	factory := NewDefaultExecutorFactory()

	// 创建事件总线（带日志）
	eventBus := NewEventBus()
	eventBus.Subscribe(NewLogSubscriber())

	// 创建调度器
	scheduler := NewScheduler(nil, repo, factory, nil,
		WithWorkers(workers),
		WithEventBus(eventBus),
	)

	// 启动调度器
	if err := scheduler.Start(); err != nil {
		return nil, fmt.Errorf("启动调度器失败: %w", err)
	}

	return scheduler, nil
}

// MustStart 启动调度器，失败时 panic
func MustStart(config *Config) *Scheduler {
	scheduler, err := Start(config)
	if err != nil {
		panic(fmt.Sprintf("启动调度器失败: %v", err))
	}
	return scheduler
}

// StopScheduler 停止调度器（兼容旧版本）
func StopScheduler(scheduler *Scheduler) {
	if scheduler != nil {
		scheduler.Stop()
	}
}
