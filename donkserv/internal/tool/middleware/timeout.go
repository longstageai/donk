package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	DefaultTimeout time.Duration            // 默认超时时间
	PerToolTimeout map[string]time.Duration // 工具特定超时
}

// DefaultTimeoutConfig 默认超时配置
var DefaultTimeoutConfig = TimeoutConfig{
	DefaultTimeout: 180 * time.Second,
	PerToolTimeout: make(map[string]time.Duration),
}

// TimeoutMiddleware 创建超时中间件
// 限制工具执行的最大时间
func TimeoutMiddleware(config ...TimeoutConfig) Middleware {
	cfg := DefaultTimeoutConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next tool.Handler) tool.Handler {
		return func(ctx *tool.Context) (*tool.Result, error) {
			// 确定超时时间
			timeout := cfg.DefaultTimeout
			if toolTimeout, ok := cfg.PerToolTimeout[ctx.ToolName]; ok {
				timeout = toolTimeout
			}
			if ctx.Timeout > 0 && ctx.Timeout < timeout {
				timeout = ctx.Timeout
			}

			// 创建带超时的上下文
			timeoutCtx, cancel := context.WithTimeout(ctx.Values, timeout)
			defer cancel()

			// 创建新的执行上下文
			execCtx := tool.NewContext(ctx.ToolName, ctx.Params)
			execCtx.RequestID = ctx.RequestID
			execCtx.Metadata = ctx.Metadata
			execCtx.Timeout = timeout
			execCtx.Values = timeoutCtx

			// 执行并等待完成或超时
			resultChan := make(chan *tool.Result, 1)
			errChan := make(chan error, 1)

			go func() {
				result, err := next(execCtx)
				resultChan <- result
				errChan <- err
			}()

			select {
			case <-timeoutCtx.Done():
				// 上下文取消或超时
				if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
					return tool.NewErrorResultWithMsg(
						tool.ErrCodeTimeout,
						"工具执行超时",
						map[string]any{"timeout": timeout.String()},
					), nil
				}
				return tool.NewErrorResultWithMsg(
					tool.ErrCodeCancelled,
					"工具执行被取消",
				), nil
			case result := <-resultChan:
				return result, <-errChan
			}
		}
	}
}

// PerToolTimeout 为特定工具设置超时
func PerToolTimeout(toolName string, timeout time.Duration) func(*TimeoutConfig) {
	return func(c *TimeoutConfig) {
		if c.PerToolTimeout == nil {
			c.PerToolTimeout = make(map[string]time.Duration)
		}
		c.PerToolTimeout[toolName] = timeout
	}
}

// WithDefaultTimeout 设置默认超时时间
func WithDefaultTimeout(timeout time.Duration) func(*TimeoutConfig) {
	return func(c *TimeoutConfig) {
		c.DefaultTimeout = timeout
	}
}
