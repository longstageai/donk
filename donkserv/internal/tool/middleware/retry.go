package middleware

import (
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries   int                                           // 最大重试次数
	InitialDelay time.Duration                                 // 初始延迟
	MaxDelay     time.Duration                                 // 最大延迟
	Multiplier   float64                                       // 延迟倍增因子
	shouldRetry  func(*tool.Context, *tool.Result, error) bool // 自定义重试判断
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	Multiplier:   2.0,
	shouldRetry: func(ctx *tool.Context, result *tool.Result, err error) bool {
		// 默认重试策略：仅在执行错误时重试
		if err != nil {
			return true
		}
		if result != nil && !result.Success {
			return true
		}
		return false
	},
}

// RetryMiddleware 创建重试中间件
// 当工具执行失败时自动重试
func RetryMiddleware(config ...RetryConfig) Middleware {
	cfg := DefaultRetryConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next tool.Handler) tool.Handler {
		return func(ctx *tool.Context) (*tool.Result, error) {
			var lastResult *tool.Result
			var lastErr error

			// 计算延迟
			delay := cfg.InitialDelay

			for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
				// 更新上下文中的重试次数
				ctx.IncrRetry()

				// 执行
				result, err := next(ctx)

				// 如果成功，直接返回
				if err == nil && result != nil && result.Success {
					return result, nil
				}

				// 记录错误
				lastResult = result
				lastErr = err

				// 检查是否应该重试
				if !cfg.shouldRetry(ctx, result, err) {
					return result, err
				}

				// 如果不是最后一次尝试，等待后重试
				if attempt < cfg.MaxRetries {
					select {
					case <-ctx.Values.Done():
						return lastResult, lastErr
					case <-time.After(delay):
						// 延迟倍增
						delay = time.Duration(float64(delay) * cfg.Multiplier)
						if delay > cfg.MaxDelay {
							delay = cfg.MaxDelay
						}
					}
				}
			}

			return lastResult, lastErr
		}
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(max int) func(*RetryConfig) {
	return func(c *RetryConfig) {
		c.MaxRetries = max
	}
}

// WithInitialDelay 设置初始延迟
func WithInitialDelay(delay time.Duration) func(*RetryConfig) {
	return func(c *RetryConfig) {
		c.InitialDelay = delay
	}
}

// WithRetryCondition 设置自定义重试条件
func WithRetryCondition(shouldRetry func(*tool.Context, *tool.Result, error) bool) func(*RetryConfig) {
	return func(c *RetryConfig) {
		c.shouldRetry = shouldRetry
	}
}

// FixedDelayRetry 创建固定延迟重试中间件
func FixedDelayRetry(maxRetries int, delay time.Duration) Middleware {
	return RetryMiddleware(RetryConfig{
		MaxRetries:   maxRetries,
		InitialDelay: delay,
		MaxDelay:     delay,
		Multiplier:   1.0,
	})
}

// ExponentialBackoffRetry 创建指数退避重试中间件
func ExponentialBackoffRetry(maxRetries int, initialDelay time.Duration) Middleware {
	return RetryMiddleware(RetryConfig{
		MaxRetries:   maxRetries,
		InitialDelay: initialDelay,
		Multiplier:   2.0,
	})
}
