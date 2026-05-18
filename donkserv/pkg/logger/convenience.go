package logger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// 全局默认日志记录器
var (
	defaultLogger *Logger   // 默认日志实例
	once          sync.Once // 初始化一次
)

// init 初始化默认日志记录器
// 默认配置：启用控制台输出，使用彩色日志
func init() {
	once.Do(func() {
		defaultLogger = New(WithWriter(NewConsoleWriter(ConsoleConfig{EnableColor: true})))
	})
}

// Default 获取默认日志记录器实例
// 返回: 全局默认Logger实例
func Default() *Logger {
	return defaultLogger
}

// SetDefault 设置全局默认日志记录器
// log: 新的默认Logger实例
func SetDefault(log *Logger) {
	defaultLogger = log
}

// InitDefault 使用指定选项初始化默认日志
// opts: 配置选项
// 返回: 初始化后的Logger实例
func InitDefault(opts ...Option) *Logger {
	defaultLogger = New(opts...)
	return defaultLogger
}

// InitWithConsole 使用控制台输出初始化日志
// level: 日志级别
// enableColor: 是否启用彩色输出
// 返回: 配置好的Logger实例
func InitWithConsole(level Level, enableColor bool) *Logger {
	consoleWriter := NewConsoleWriter(ConsoleConfig{
		EnableColor: enableColor,
	})

	defaultLogger = New(
		WithLevel(level),
		WithWriter(consoleWriter),
	)

	return defaultLogger
}

// InitWithFile 使用文件输出初始化日志
// dir: 日志目录
// level: 日志级别
// maxSize: 单个文件最大大小
// maxAge: 文件保留天数
// maxBackups: 备份文件数量
// 返回: 配置好的Logger实例
func InitWithFile(dir string, level Level, maxSize int64, maxAge int, maxBackups int) *Logger {
	fileWriter, err := NewFileWriter(FileConfig{
		Dir:           dir,
		Level:         level,
		MaxSize:       maxSize,
		MaxAge:        maxAge,
		MaxBackups:    maxBackups,
		RotateEnabled: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init file writer: %v\n", err)
		return nil
	}

	consoleWriter := NewConsoleWriter(ConsoleConfig{
		EnableColor: true,
	})

	defaultLogger = New(
		WithLevel(level),
		WithWriter(fileWriter),
		WithWriter(consoleWriter),
	)

	return defaultLogger
}

// Init 初始化完整的日志系统
// 同时配置文件输出（按级别分文件）和控制台输出
// dir: 日志目录
// level: 日志级别
// maxSize: 单个文件最大大小
// maxAge: 文件保留天数
// maxBackups: 备份文件数量
// 返回: 配置好的Logger实例
func Init(dir string, level Level, maxSize int64, maxAge int, maxBackups int) *Logger {
	writers := []Writer{}

	// 为每个日志级别创建独立的文件写入器
	levels := []Level{DEBUG, INFO, WARN, ERROR}
	for _, lv := range levels {
		fileWriter, err := NewFileWriter(FileConfig{
			Dir:           dir,
			Level:         lv,
			MaxSize:       maxSize,
			MaxAge:        maxAge,
			MaxBackups:    maxBackups,
			RotateEnabled: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create file writer for %s: %v\n", lv.String(), err)
			continue
		}
		writers = append(writers, fileWriter)
	}

	// 添加控制台写入器
	consoleWriter := NewConsoleWriter(ConsoleConfig{
		EnableColor: true,
		MinLevel:    level,
	})
	writers = append(writers, consoleWriter)

	// 创建Logger实例
	defaultLogger = New(
		WithLevel(level),
	)

	// 添加所有写入器
	for _, w := range writers {
		defaultLogger.writers = append(defaultLogger.writers, w)
	}

	return defaultLogger
}

// Debug 记录调试级别日志（全局）
// message: 日志消息
// fields: 附加字段
func Debug(message string, fields map[string]interface{}) {
	defaultLogger.Debug(message, fields)
}

// Info 记录信息级别日志（全局）
// message: 日志消息
// fields: 附加字段
func Info(message string, fields map[string]interface{}) {
	defaultLogger.Info(message, fields)
}

// Warn 记录警告级别日志（全局）
// message: 日志消息
// fields: 附加字段
func Warn(message string, fields map[string]interface{}) {
	defaultLogger.Warn(message, fields)
}

// Error 记录错误级别日志（全局）
// message: 日志消息
// fields: 附加字段
func Error(message string, fields map[string]interface{}) {
	defaultLogger.Error(message, fields)
}

// Fatal 记录致命错误日志（全局）
// message: 日志消息
// fields: 附加字段
func Fatal(message string, fields map[string]interface{}) {
	defaultLogger.Fatal(message, fields)
}

// Debugf 记录格式化调试日志
// format: 格式化字符串
// args: 参数列表
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debug(fmt.Sprintf(format, args...), nil)
}

// Infof 记录格式化信息日志
// format: 格式化字符串
// args: 参数列表
func Infof(format string, args ...interface{}) {
	defaultLogger.Info(fmt.Sprintf(format, args...), nil)
}

// Warnf 记录格式化警告日志
// format: 格式化字符串
// args: 参数列表
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warn(fmt.Sprintf(format, args...), nil)
}

// Errorf 记录格式化错误日志
// format: 格式化字符串
// args: 参数列表
func Errorf(format string, args ...interface{}) {
	defaultLogger.Error(fmt.Sprintf(format, args...), nil)
}

// Fatalf 记录格式化致命错误日志
// format: 格式化字符串
// args: 参数列表
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatal(fmt.Sprintf(format, args...), nil)
}

// With 创建带上下文的日志辅助对象
// fields: 日志附加字段
// 返回: 可链式调用的logHelper
func With(fields map[string]interface{}) *logHelper {
	return &logHelper{fields: fields}
}

// logHelper 带上下文的日志辅助对象
type logHelper struct {
	fields map[string]interface{} // 日志附加字段
}

// Debug 记录调试日志
func (h *logHelper) Debug(msg string) {
	defaultLogger.Debug(msg, h.fields)
}

// Info 记录信息日志
func (h *logHelper) Info(msg string) {
	defaultLogger.Info(msg, h.fields)
}

// Warn 记录警告日志
func (h *logHelper) Warn(msg string) {
	defaultLogger.Warn(msg, h.fields)
}

// Error 记录错误日志
func (h *logHelper) Error(msg string) {
	defaultLogger.Error(msg, h.fields)
}

// Fatal 记录致命错误日志
func (h *logHelper) Fatal(msg string) {
	defaultLogger.Fatal(msg, h.fields)
}

// Debugf 记录格式化调试日志
func (h *logHelper) Debugf(format string, args ...interface{}) {
	defaultLogger.Debug(fmt.Sprintf(format, args...), h.fields)
}

// Infof 记录格式化信息日志
func (h *logHelper) Infof(format string, args ...interface{}) {
	defaultLogger.Info(fmt.Sprintf(format, args...), h.fields)
}

// Warnf 记录格式化警告日志
func (h *logHelper) Warnf(format string, args ...interface{}) {
	defaultLogger.Warn(fmt.Sprintf(format, args...), h.fields)
}

// Errorf 记录格式化错误日志
func (h *logHelper) Errorf(format string, args ...interface{}) {
	defaultLogger.Error(fmt.Sprintf(format, args...), h.fields)
}

// Fatalf 记录格式化致命错误日志
func (h *logHelper) Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatal(fmt.Sprintf(format, args...), h.fields)
}

// SetLevel 设置全局日志级别
// level: 新的日志级别
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// GetLevel 获取全局日志级别
// 返回: 当前日志级别
func GetLevel() Level {
	return defaultLogger.GetLevel()
}

// Close 关闭全局日志记录器
func Close() {
	defaultLogger.Close()
}

// FilterWriter 带过滤功能的写入器装饰器
type FilterWriter struct {
	mu      sync.Mutex
	writer  Writer
	filters []Filter
}

// NewFilterWriter 创建带过滤功能的写入器
// writer: 底层写入器
// filters: 过滤规则列表
// 返回: 装饰后的写入器
func NewFilterWriter(writer Writer, filters ...Filter) *FilterWriter {
	return &FilterWriter{
		writer:  writer,
		filters: filters,
	}
}

// Write 写入日志前先应用过滤规则
func (fw *FilterWriter) Write(record *Record) error {
	// 依次应用所有过滤器
	for _, filter := range fw.filters {
		if !filter.ShouldLog(record) {
			return nil
		}
	}
	return fw.writer.Write(record)
}

// Close 关闭底层写入器
func (fw *FilterWriter) Close() error {
	return fw.writer.Close()
}

// MultiWriter 多写入器
// 同时向多个写入器写入日志
type MultiWriter struct {
	mu      sync.Mutex
	writers []Writer
}

// NewMultiWriter 创建多写入器
// writers: 写入器列表
// 返回: 多写入器实例
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write 向所有写入器写入日志
func (mw *MultiWriter) Write(record *Record) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	for _, writer := range mw.writers {
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

// Close 关闭所有写入器
func (mw *MultiWriter) Close() error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	for _, writer := range mw.writers {
		if err := writer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	File     string // 匹配的文件路径
	Line     int    // 匹配的行号
	Content  string // 匹配的内容
	MatchNum int    // 匹配的次数
}

// SearchInDir 在目录中搜索日志
// dir: 要搜索的目录
// keyword: 搜索关键字
// 返回: 搜索结果列表和可能的错误
func SearchInDir(dir string, keyword string) ([]SearchResult, error) {
	var results []SearchResult

	// 遍历目录
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 只处理.log文件
		if !strings.HasSuffix(info.Name(), ".log") {
			return nil
		}

		// 打开文件并搜索
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if strings.Contains(scanner.Text(), keyword) {
				results = append(results, SearchResult{
					File:     path,
					Line:     lineNum,
					Content:  scanner.Text(),
					MatchNum: strings.Count(scanner.Text(), keyword),
				})
			}
		}

		return nil
	})

	return results, err
}

// GetCallerInfo 获取调用者信息
// skip: 跳过的栈帧数
// 返回: 文件路径、行号、函数名
func GetCallerInfo(skip int) (file string, line int, function string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0, ""
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return file, line, ""
	}
	return file, line, fn.Name()
}

// Recovery panic恢复函数
// 可用于defer中捕获panic并记录日志
func Recovery() {
	if r := recover(); r != nil {
		Error("Panic recovered", map[string]interface{}{
			"panic": r,
		})
	}
}
