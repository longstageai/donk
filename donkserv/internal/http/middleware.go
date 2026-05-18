package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware API Key认证中间件
// API Key 直接在代码中写死
type AuthMiddleware struct {
	validKeys map[string]bool // 有效的API Key集合
}

// NewAuthMiddleware 创建认证中间件
// 参数:
//   - keys: 有效的API Key列表
//
// 返回:
//   - *AuthMiddleware: 认证中间件实例
func NewAuthMiddleware(keys ...string) *AuthMiddleware {
	m := &AuthMiddleware{
		validKeys: make(map[string]bool),
	}

	for _, key := range keys {
		if key != "" {
			m.validKeys[key] = true
		}
	}

	return m
}

// GinHandler 返回Gin中间件处理函数
// 用于 Gin 框架的 middleware
func (m *AuthMiddleware) GinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization Header
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 Authorization header"})
			return
		}

		// 解析 Bearer token
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的 Authorization 格式"})
			return
		}

		token := parts[1]

		// 验证API Key
		if !m.validKeys[token] {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的 API Key"})
			return
		}

		// 认证通过，继续处理请求
		c.Next()
	}
}

// Handler 返回http.Handler中间件处理函数
// 用于标准库 http.Server
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取 Authorization Header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "缺少 Authorization header", http.StatusUnauthorized)
			return
		}

		// 解析 Bearer token
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "无效的 Authorization 格式", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// 验证API Key
		if !m.validKeys[token] {
			http.Error(w, "无效的 API Key", http.StatusUnauthorized)
			return
		}

		// 认证通过，继续处理请求
		next.ServeHTTP(w, r)
	})
}

// IsEnabled 检查认证是否已启用
func (m *AuthMiddleware) IsEnabled() bool {
	return len(m.validKeys) > 0
}
