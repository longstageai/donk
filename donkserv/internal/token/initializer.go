package token

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

// Setup 初始化Token统计模块并注册API路由
// 参数:
//   - db: 数据库连接
//   - engine: Gin引擎实例
//
// 返回:
//   - *TokenStats: Token统计实例
//   - error: 错误信息
func Setup(db *sql.DB, engine *gin.Engine) (*TokenStats, error) {
	// 创建Token统计实例
	stats, err := NewTokenStats(db)
	if err != nil {
		return nil, err
	}

	// 注册API路由
	handler := NewHandler(stats)
	handler.RegisterRoutes(engine)

	return stats, nil
}
