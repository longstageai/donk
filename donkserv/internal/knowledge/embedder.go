// knowledge 知识库模块
package knowledge

import (
	"database/sql"
	"fmt"

	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// CreateEmbedderFromSetting 从 setting 模块创建嵌入器
// db: 数据库连接
// 从数据库获取 embedding 配置并创建 embedder
// 返回嵌入器实例或错误
func CreateEmbedderFromSetting(db *sql.DB) (embedding.Embedder, error) {
	// 创建 setting 存储
	storage := setting.NewStorage(db)

	// 创建 setting 服务
	service := setting.NewService(storage)

	// 获取 embedding 配置
	embedConfig, err := service.GetEmbeddingConfig()
	if err != nil {
		return nil, fmt.Errorf("获取 embedding 配置失败: %w", err)
	}

	// 验证配置
	if embedConfig.Provider == "" {
		return nil, fmt.Errorf("embedding provider 未配置")
	}
	if embedConfig.Model == "" {
		return nil, fmt.Errorf("embedding model 未配置")
	}
	if embedConfig.APIKey == "" {
		return nil, fmt.Errorf("embedding api_key 未配置")
	}

	// 创建 embedder
	embedder, err := embedding.NewEmbedding(
		embedConfig.Provider,
		embedConfig.APIKey,
		embedConfig.Model,
		embedConfig.BaseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 embedder 失败: %w", err)
	}

	logger.Info("从 setting 模块创建 embedder 成功", map[string]interface{}{
		"provider": embedConfig.Provider,
		"model":    embedConfig.Model,
	})

	return embedder, nil
}

// NewBuilderWithDB 使用数据库连接创建知识库构建器
// 自动从 setting 模块获取 embedding 配置
// dataDir: 数据目录
// db: 数据库连接
// config: 构建器配置（nil则使用默认）
// 返回构建器实例或错误
func NewBuilderWithDB(dataDir string, db *sql.DB, config *BuilderConfig) (*Builder, error) {
	// 从 setting 创建 embedder
	embedder, err := CreateEmbedderFromSetting(db)
	if err != nil {
		return nil, fmt.Errorf("创建 embedder 失败: %w", err)
	}

	// 创建构建器
	return NewBuilder(dataDir, embedder, config)
}
