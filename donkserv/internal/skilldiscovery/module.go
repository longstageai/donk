// skilldiscovery 技能自动发现模块
// 模块入口，提供简化的初始化接口
package skilldiscovery

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/longstageai/donk/donk/internal/conversation"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Init 初始化技能发现模块（简化接口）
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - *Initializer: 初始化器实例
//   - error: 错误信息
func Init(db *sql.DB) (*Initializer, error) {

	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为空")
	}

	logger.Info("初始化技能发现模块", map[string]interface{}{})

	// 获取 setting Provider
	provider := setting.GetProvider()
	if provider == nil {
		return nil, fmt.Errorf("配置提供者未初始化")
	}

	// 创建技能状态仓库
	stateRepo := skill.NewStateRepository(db)

	// 获取技能目录
	// 使用默认路径：./data/skills
	skillsDir := filepath.Join("data", "skills")

	// 创建配置
	config := DefaultConfig()

	// 创建初始化器
	initializer := NewInitializer(
		config,
		WithDB(db),
		WithStateRepository(stateRepo),
		WithSettingService(provider),
		WithSkillsDirectory(skillsDir),
	)

	// 初始化
	if err := initializer.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化技能发现模块失败: %w", err)
	}

	logger.Info("技能发现模块初始化完成", map[string]interface{}{})
	return initializer, nil
}

// InitWithOptions 使用自定义选项初始化技能发现模块
// 参数:
//   - db: 数据库连接
//   - opts: 初始化选项
//
// 返回:
//   - *Initializer: 初始化器实例
//   - error: 错误信息
func InitWithOptions(db *sql.DB, opts ...InitializerOption) (*Initializer, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为空")
	}

	logger.Info("初始化技能发现模块（自定义选项）", map[string]interface{}{})

	// 创建配置
	config := DefaultConfig()

	// 创建初始化器
	initializer := NewInitializer(config, opts...)

	// 如果没有设置 setting 服务，尝试获取默认 Provider
	if initializer.settingSvc == nil {
		if provider := setting.GetProvider(); provider != nil {
			initializer.settingSvc = provider
		}
	}

	// 如果没有设置状态仓库，创建默认的
	if initializer.stateRepo == nil {
		initializer.stateRepo = skill.NewStateRepository(db)
	}

	// 初始化
	if err := initializer.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化技能发现模块失败: %w", err)
	}

	logger.Info("技能发现模块初始化完成", map[string]interface{}{})
	return initializer, nil
}

// InitWithWebSocket 使用 WebSocket 通知初始化技能发现模块
// 参数:
//   - db: 数据库连接
//   - wsHub: WebSocket Hub
//
// 返回:
//   - *Initializer: 初始化器实例
//   - error: 错误信息
func InitWithWebSocket(db *sql.DB, wsHub *websocket.Hub) (*Initializer, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为空")
	}

	logger.Info("初始化技能发现模块（含 WebSocket）", map[string]interface{}{})

	// 获取 setting Provider
	provider := setting.GetProvider()
	if provider == nil {
		return nil, fmt.Errorf("配置提供者未初始化")
	}

	// 创建技能状态仓库
	stateRepo := skill.NewStateRepository(db)

	// 获取技能目录
	skillsDir := filepath.Join("data", "skills")

	// 创建配置
	config := DefaultConfig()

	// 创建初始化器
	initializer := NewInitializer(
		config,
		WithDB(db),
		WithStateRepository(stateRepo),
		WithSettingService(provider),
		WithSkillsDirectory(skillsDir),
		WithWebSocketHub(wsHub),
	)

	// 初始化
	if err := initializer.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化技能发现模块失败: %w", err)
	}

	logger.Info("技能发现模块初始化完成", map[string]interface{}{})
	return initializer, nil
}

// InitWithConversation 使用对话存储初始化技能发现模块
// 参数:
//   - db: 数据库连接
//   - convStore: 对话存储
//
// 返回:
//   - *Initializer: 初始化器实例
//   - error: 错误信息
func InitWithConversation(db *sql.DB, convStore *conversation.Store) (*Initializer, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为空")
	}

	logger.Info("初始化技能发现模块（含对话存储）", map[string]interface{}{})

	// 获取 setting Provider
	provider := setting.GetProvider()
	if provider == nil {
		return nil, fmt.Errorf("配置提供者未初始化")
	}

	// 创建技能状态仓库
	stateRepo := skill.NewStateRepository(db)

	// 获取技能目录
	skillsDir := filepath.Join("data", "skills")

	// 创建配置
	config := DefaultConfig()

	// 创建初始化器
	initializer := NewInitializer(
		config,
		WithDB(db),
		WithStateRepository(stateRepo),
		WithSettingService(provider),
		WithSkillsDirectory(skillsDir),
		WithConversationStore(convStore),
	)

	// 初始化
	if err := initializer.Initialize(); err != nil {
		return nil, fmt.Errorf("初始化技能发现模块失败: %w", err)
	}

	logger.Info("技能发现模块初始化完成", map[string]interface{}{})
	return initializer, nil
}
