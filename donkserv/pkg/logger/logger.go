package logger

import (
	"fmt"
	"sync"
	"time"
)

// Level 定义日志级别
type Level int

const (
	DEBUG Level = iota // 调试级别
	INFO               // 信息级别
	WARN               // 警告级别
	ERROR              // 错误级别
	FATAL              // 致命错误级别
)

// String 返回日志级别的字符串表示
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Record 表示一条日志记录
type Record struct {
	Timestamp time.Time              // 日志时间戳
	Level     Level                  // 日志级别
	Message   string                 // 日志消息内容
	Fields    map[string]interface{} // 附加字段信息
}

// Formatter 接口定义日志格式化器
type Formatter interface {
	Format(record *Record) string
}

// DefaultFormatter 默认的日志格式化器
type DefaultFormatter struct{}

// Format 格式化日志记录为字符串
// 输出格式: [时间戳] [级别] 消息 附加字段
func (f *DefaultFormatter) Format(record *Record) string {
	timestamp := record.Timestamp.Format("2006-01-02 15:04:05")
	level := record.Level.String()

	// 如果有附加字段，追加到消息末尾
	if len(record.Fields) > 0 {
		fieldStr := ""
		for k, v := range record.Fields {
			fieldStr += fmt.Sprintf(" %s=%v", k, v)
		}
		return fmt.Sprintf("[%s] [%s] %s%s", timestamp, level, record.Message, fieldStr)
	}

	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, record.Message)
}

// Writer 接口定义日志写入器
type Writer interface {
	Write(record *Record) error // 写入日志记录
	Close() error               // 关闭写入器
}

// Logger 日志记录器结构体
type Logger struct {
	mu        sync.RWMutex   // 读写锁，保证线程安全
	level     Level          // 当前日志级别
	writers   []Writer       // 日志写入器列表
	formatter Formatter      // 日志格式化器
	async     bool           // 是否启用异步模式
	msgChan   chan *Record   // 异步消息通道
	closeChan chan struct{}  // 关闭信号通道
	wg        sync.WaitGroup // 等待组，用于等待异步写入完成
}

// Option 函数选项模式，用于配置Logger
type Option func(*Logger)

// WithLevel 设置日志级别
// level: 日志级别
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithWriter 添加日志写入器
// writer: 日志写入器实现
func WithWriter(writer Writer) Option {
	return func(l *Logger) {
		l.writers = append(l.writers, writer)
	}
}

// WithFormatter 设置日志格式化器
// formatter: 日志格式化器实现
func WithFormatter(formatter Formatter) Option {
	return func(l *Logger) {
		l.formatter = formatter
	}
}

// WithAsync 启用异步日志模式
// 异步模式可以提高性能，但可能在程序异常退出时丢失少量日志
func WithAsync() Option {
	return func(l *Logger) {
		l.async = true
	}
}

// New 创建新的日志记录器
// opts: 可选的配置函数
// 返回: 配置好的Logger实例
func New(opts ...Option) *Logger {
	logger := &Logger{
		level:     INFO,                // 默认级别为INFO
		formatter: &DefaultFormatter{}, // 默认使用默认格式化器
		async:     false,               // 默认同步模式
		msgChan:   make(chan *Record, 1024),
		closeChan: make(chan struct{}),
	}

	// 应用所有配置选项
	for _, opt := range opts {
		opt(logger)
	}

	// 如果启用异步模式，启动异步写入协程
	if logger.async {
		logger.startAsyncWriter()
	}

	return logger
}

// startAsyncWriter 启动异步日志写入协程
// 从消息通道读取日志记录并写入所有写入器
func (l *Logger) startAsyncWriter() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		for {
			select {
			case record := <-l.msgChan:
				l.writeToWriters(record)
			case <-l.closeChan:
				// 处理通道中剩余的消息
				for {
					select {
					case record := <-l.msgChan:
						l.writeToWriters(record)
					default:
						return
					}
				}
			}
		}
	}()
}

// writeToWriters 将日志写入所有已配置的写入器
// 同时处理格式化
func (l *Logger) writeToWriters(record *Record) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 格式化日志消息
	formatted := l.formatter.Format(record)
	record.Message = formatted

	// 写入所有写入器
	for _, writer := range l.writers {
		if err := writer.Write(record); err != nil {
			fmt.Printf("Logger: write error: %v\n", err)
		}
	}
}

// log 内部日志记录方法
// 根据配置的日志级别决定是否记录
func (l *Logger) log(level Level, message string, fields map[string]interface{}) {
	// 如果日志级别低于配置的级别，则不记录
	if level < l.level {
		return
	}

	record := &Record{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	if l.async {
		// 异步模式：发送到消息通道
		select {
		case l.msgChan <- record:
		default:
			// 通道满时丢弃消息并打印警告
			fmt.Printf("Logger: message channel is full, dropping message: %s\n", message)
		}
	} else {
		// 同步模式：直接写入
		l.writeToWriters(record)
	}
}

// Debug 记录调试级别日志
// message: 日志消息
// fields: 附加字段，可为nil
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log(DEBUG, message, fields)
}

// Info 记录信息级别日志
// message: 日志消息
// fields: 附加字段，可为nil
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log(INFO, message, fields)
}

// Warn 记录警告级别日志
// message: 日志消息
// fields: 附加字段，可为nil
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log(WARN, message, fields)
}

// Error 记录错误级别日志
// message: 日志消息
// fields: 附加字段，可为nil
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log(ERROR, message, fields)
}

// Fatal 记录致命错误级别日志
// message: 日志消息
// fields: 附加字段，可为nil
func (l *Logger) Fatal(message string, fields map[string]interface{}) {
	l.log(FATAL, message, fields)
}

// SetLevel 动态设置日志级别
// level: 新的日志级别
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel 获取当前日志级别
// 返回: 当前日志级别
func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// Close 关闭日志记录器
// 会等待异步写入完成后关闭所有写入器
func (l *Logger) Close() {
	if l.async {
		close(l.closeChan)
		l.wg.Wait()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, writer := range l.writers {
		writer.Close()
	}
}
