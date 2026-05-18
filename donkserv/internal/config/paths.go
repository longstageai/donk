package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// DataPaths 数据文件路径配置
// 统一管理所有数据文件的存储位置
type DataPaths struct {
	DataDir        string // 数据根目录
	MainDB         string // SQLite 主数据库 (donk.db)
	MemoryVectorDB string // 长期记忆向量数据库 (memory.db)
	ConversationDB string // 对话历史向量数据库 (conversation.db)
	Knowledge      string // 知识库数据目录 (knowledge/)
}

// DefaultDataPaths 默认数据路径配置
var DefaultDataPaths = DataPaths{
	DataDir:        "./data",
	MainDB:         "./data/db/donk.db",
	MemoryVectorDB: "./data/db/memory.db",
	ConversationDB: "./data/db/conversation.db",
	Knowledge:      "./data/knowledge",
}

// InitDataDir 初始化数据目录
// 创建数据目录及其子目录（如果不存在）
//
// 返回:
//   - error: 错误信息
func InitDataDir() error {
	paths := DefaultDataPaths

	// 创建数据根目录
	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	return nil
}

// GetDataPaths 获取数据路径配置
// 支持通过环境变量覆盖默认路径
//
// 环境变量:
//   - donk_DATA_DIR: 数据根目录
//
// 返回:
//   - DataPaths: 数据路径配置
func GetDataPaths() DataPaths {
	paths := DefaultDataPaths

	// 允许通过环境变量覆盖
	if dataDir := os.Getenv("donk_DATA_DIR"); dataDir != "" {
		paths.DataDir = dataDir
		paths.MainDB = filepath.Join(dataDir, "db", "donk.db")
		paths.MemoryVectorDB = filepath.Join(dataDir, "db", "memory.db")
		paths.ConversationDB = filepath.Join(dataDir, "db", "conversation.db")
		paths.Knowledge = filepath.Join(dataDir, "knowledge")
	}

	return paths
}

// EnsureDir 确保目录存在
//
// 参数:
//   - path: 目录路径
//
// 返回:
//   - error: 错误信息
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("创建目录失败 %s: %w", path, err)
	}
	return nil
}
