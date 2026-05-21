package middleware

import (
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// LogLevel 日志级别
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// LogConfig 日志中间件配置
type LogConfig struct {
	LogLevel      LogLevel                                                       // 日志级别
	Prefix        string                                                         // 日志前缀
	PrintParams   bool                                                           // 是否打印参数
	PrintResult   bool                                                           // 是否打印结果
	PrintDuration bool                                                           // 是否打印执行时间
	Formatter     func(*tool.Context, time.Duration, *tool.Result, error) string // 自定义格式化函数
}

// DefaultLogConfig 默认日志配置
var DefaultLogConfig = LogConfig{
	LogLevel:      LevelInfo,
	Prefix:        "[TOOL]",
	PrintParams:   true,
	PrintResult:   true,
	PrintDuration: true,
}

// LogMiddleware 创建日志中间件
// 记录工具执行的请求、响应和时间
func LogMiddleware(config ...LogConfig) Middleware {
	cfg := DefaultLogConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next tool.Handler) tool.Handler {
		return func(ctx *tool.Context) (*tool.Result, error) {
			// 打印请求信息
			if cfg.LogLevel <= LevelDebug {
				logger.Info(fmt.Sprintf("%s [%s] 开始执行工具: %s", cfg.Prefix, time.Now().Format("2006-01-02 15:04:05"), ctx.ToolName), nil)

				if cfg.PrintParams && len(ctx.Params) > 0 {
					logger.Info(fmt.Sprintf("%s 参数: %+v", cfg.Prefix, ctx.Params), nil)
				}
			}

			// 记录开始时间
			startTime := time.Now()

			// 执行
			result, err := next(ctx)

			// 计算执行时间
			duration := time.Since(startTime)

			// 打印结果
			if cfg.LogLevel <= LevelInfo {
				if err != nil {
					logger.Error(fmt.Sprintf("%s [%s] 工具执行失败: %s, 错误: %v, 耗时: %v",
						cfg.Prefix, time.Now().Format("2006-01-02 15:04:05"), ctx.ToolName, err, duration), nil)
				} else if cfg.LogLevel <= LevelDebug || cfg.PrintDuration {
					status := "成功"
					if result != nil && !result.Success {
						status = "失败"
					}
					logger.Info(fmt.Sprintf("%s [%s] 工具执行完成: %s, 状态: %s, 耗时: %v",
						cfg.Prefix, time.Now().Format("2006-01-02 15:04:05"), ctx.ToolName, status, duration), nil)
				}

				if cfg.PrintResult && result != nil {
					if result.Success && result.Data != nil {
						logger.Info(fmt.Sprintf("%s 结果: %+v", cfg.Prefix, result.Data), nil)
					} else if !result.Success && result.Error != nil {
						logger.Warn(fmt.Sprintf("%s 错误: %s", cfg.Prefix, result.String()), nil)
					}
				}
			}

			return result, err
		}
	}
}

// StructuredLogMiddleware 结构化日志中间件
// 使用结构化格式记录日志
func StructuredLogMiddleware() Middleware {
	return func(next tool.Handler) tool.Handler {
		return func(ctx *tool.Context) (*tool.Result, error) {
			logger.Info(fmt.Sprintf("[TOOL] tool=%s request_id=%s started", ctx.ToolName, ctx.RequestID), nil)

			startTime := time.Now()
			result, err := next(ctx)
			duration := time.Since(startTime)

			if err != nil {
				logger.Error(fmt.Sprintf("[TOOL] tool=%s request_id=%s error=%v duration=%v",
					ctx.ToolName, ctx.RequestID, err, duration), nil)
			} else {
				logger.Info(fmt.Sprintf("[TOOL] tool=%s request_id=%s success=%v duration=%v",
					ctx.ToolName, ctx.RequestID, result.Success, duration), nil)
			}

			return result, err
		}
	}
}

// WithLogLevel 设置日志级别
func WithLogLevel(level LogLevel) func(*LogConfig) {
	return func(c *LogConfig) {
		c.LogLevel = level
	}
}

// WithPrefix 设置日志前缀
func WithPrefix(prefix string) func(*LogConfig) {
	return func(c *LogConfig) {
		c.Prefix = prefix
	}
}

// WithFormatter 设置自定义格式化函数
func WithFormatter(formatter func(*tool.Context, time.Duration, *tool.Result, error) string) func(*LogConfig) {
	return func(c *LogConfig) {
		c.Formatter = formatter
	}
}

// CustomLogMiddleware 创建自定义日志中间件
func CustomLogMiddleware(formatter func(*tool.Context, time.Duration, *tool.Result, error) string) Middleware {
	return func(next tool.Handler) tool.Handler {
		return func(ctx *tool.Context) (*tool.Result, error) {
			startTime := time.Now()
			result, err := next(ctx)
			duration := time.Since(startTime)

			if formatter != nil {
				logger.Info(formatter(ctx, duration, result, err), nil)
			}

			return result, err
		}
	}
}
