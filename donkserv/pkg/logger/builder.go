package logger

import "fmt"

// Filter 接口定义日志过滤器
// 用于过滤日志记录
type Filter interface {
	ShouldLog(record *Record) bool
}

// LevelFilter 按级别过滤日志
type LevelFilter struct {
	MinLevel Level // 最小级别
	MaxLevel Level // 最大级别
}

// ShouldLog 判断记录是否应该被记录
// 只有级别在MinLevel和MaxLevel之间时才记录
func (f *LevelFilter) ShouldLog(record *Record) bool {
	return record.Level >= f.MinLevel && record.Level <= f.MaxLevel
}

// KeywordFilter 按关键字过滤日志
type KeywordFilter struct {
	Keywords []string // 关键字列表
	Exclude  bool     // 是否为排除模式
}

// ShouldLog 判断记录是否应该被记录
// Exclude为true时，排除包含关键字的日志
// Exclude为false时，只记录包含关键字的日志
func (f *KeywordFilter) ShouldLog(record *Record) bool {
	if f.Exclude {
		// 排除模式：包含关键字的不记录
		for _, keyword := range f.Keywords {
			if contains(record.Message, keyword) {
				return false
			}
		}
		return true
	}

	// 包含模式：只有包含关键字的才记录
	for _, keyword := range f.Keywords {
		if contains(record.Message, keyword) {
			return true
		}
	}
	return false
}

// contains 检查字符串s是否包含子串substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

// containsHelper 辅助函数，检查字符串包含关系
func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// LoggerBuilder 日志构建器
// 使用链式调用方式构建Logger
type LoggerBuilder struct {
	logger *Logger
}

// NewLogger 创建新的日志构建器
// 返回: 可配置的LoggerBuilder实例
func NewLogger() *LoggerBuilder {
	return &LoggerBuilder{
		logger: New(),
	}
}

// SetLevel 设置日志级别
// level: 日志级别
// 返回: Builder本身，支持链式调用
func (b *LoggerBuilder) SetLevel(level Level) *LoggerBuilder {
	b.logger.SetLevel(level)
	return b
}

// AddConsoleWriter 添加控制台写入器
// enableColor: 是否启用彩色输出
// 返回: Builder本身，支持链式调用
func (b *LoggerBuilder) AddConsoleWriter(enableColor bool) *LoggerBuilder {
	writer := NewConsoleWriter(ConsoleConfig{
		EnableColor: enableColor,
	})
	b.logger.writers = append(b.logger.writers, writer)
	return b
}

// AddFileWriter 添加文件写入器
// dir: 日志目录
// level: 日志级别
// maxSize: 单个文件最大大小（字节）
// maxAge: 文件保留天数
// maxBackups: 保留的备份文件数量
// 返回: Builder本身，支持链式调用
func (b *LoggerBuilder) AddFileWriter(dir string, level Level, maxSize int64, maxAge int, maxBackups int) *LoggerBuilder {
	writer, err := NewFileWriter(FileConfig{
		Dir:           dir,
		Level:         level,
		MaxSize:       maxSize,
		MaxAge:        maxAge,
		MaxBackups:    maxBackups,
		RotateEnabled: true,
	})
	if err != nil {
		fmt.Printf("Failed to create file writer: %v\n", err)
		return b
	}
	b.logger.writers = append(b.logger.writers, writer)
	return b
}

// EnableAsync 启用异步日志模式
// 返回: Builder本身，支持链式调用
func (b *LoggerBuilder) EnableAsync() *LoggerBuilder {
	b.logger.async = true
	b.logger.msgChan = make(chan *Record, 1024)
	b.logger.closeChan = make(chan struct{})
	b.logger.startAsyncWriter()
	return b
}

// SetFormatter 设置日志格式化器
// formatter: 格式化器实现
// 返回: Builder本身，支持链式调用
func (b *LoggerBuilder) SetFormatter(formatter Formatter) *LoggerBuilder {
	b.logger.formatter = formatter
	return b
}

// Build 完成构建，返回Logger实例
// 返回: 配置完成的Logger实例
func (b *LoggerBuilder) Build() *Logger {
	return b.logger
}
