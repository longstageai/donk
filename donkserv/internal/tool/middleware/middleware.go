package middleware

import (
	"github.com/longstageai/donk/donk/internal/tool"
)

// Middleware 中间件类型
// 中间件是一个函数，接收一个Handler并返回一个新的Handler
type Middleware func(next tool.Handler) tool.Handler

// Chain 将多个中间件串联成链
// 执行顺序: middlewares[0] -> middlewares[1] -> ... -> handler
func Chain(middlewares ...Middleware) Middleware {
	return func(next tool.Handler) tool.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// Apply 应用中间件到Handler
func Apply(handler tool.Handler, middlewares ...Middleware) tool.Handler {
	return Chain(middlewares...)(handler)
}
