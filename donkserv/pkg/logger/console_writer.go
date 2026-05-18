package logger

import (
	"fmt"
	"os"
	"sync"
)

// 控制台颜色常量
const (
	ColorReset  = "\033[0m"  // 重置颜色
	ColorRed    = "\033[31m" // 红色 - 错误/致命错误
	ColorGreen  = "\033[32m" // 绿色 - 信息
	ColorYellow = "\033[33m" // 黄色 - 警告
	ColorBlue   = "\033[34m" // 蓝色 - 调试
	ColorCyan   = "\033[36m" // 青色
	ColorGray   = "\033[90m" // 灰色
)

// ConsoleConfig 控制台写入器配置结构体
type ConsoleConfig struct {
	EnableColor bool             // 是否启用彩色输出
	UseColors   map[Level]string // 各日志级别对应的颜色
	MinLevel    Level            // 最小日志级别，低于此级别的日志不输出
}

// consoleWriter 控制台日志写入器
type consoleWriter struct {
	mu     sync.Mutex    // 互斥锁，保证线程安全
	config ConsoleConfig // 配置信息
}

// NewConsoleWriter 创建控制台写入器
// config: 控制台写入器配置
// 返回: 配置好的控制台写入器实例
func NewConsoleWriter(config ConsoleConfig) *consoleWriter {
	// 设置默认颜色映射
	if config.UseColors == nil {
		config.UseColors = map[Level]string{
			DEBUG: ColorBlue,   // 调试 - 蓝色
			INFO:  ColorGreen,  // 信息 - 绿色
			WARN:  ColorYellow, // 警告 - 黄色
			ERROR: ColorRed,    // 错误 - 红色
			FATAL: ColorRed,    // 致命错误 - 红色
		}
	}

	// 设置默认最小级别
	if config.MinLevel == 0 {
		config.MinLevel = DEBUG
	}

	return &consoleWriter{
		config: config,
	}
}

// Write 写入日志记录到控制台
// 根据配置的MinLevel过滤日志，并输出彩色日志
func (w *consoleWriter) Write(record *Record) error {
	// 级别过滤：只输出大于等于最小级别的日志
	if record.Level < w.config.MinLevel {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// 根据配置决定是否使用彩色输出
	if w.config.EnableColor {
		// 获取对应级别的颜色
		color, ok := w.config.UseColors[record.Level]
		if !ok {
			color = ColorReset
		}
		// 输出彩色日志
		fmt.Fprintf(os.Stdout, "%s%s%s\n", color, record.Message, ColorReset)
	} else {
		// 输出普通日志
		fmt.Fprintln(os.Stdout, record.Message)
	}

	return nil
}

// Close 关闭控制台写入器
// 控制台写入器不需要关闭操作，直接返回nil
func (w *consoleWriter) Close() error {
	return nil
}
