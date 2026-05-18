// background 后台Agent模块
package background

import (
	"database/sql"
	"sync"

	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Manager 后台Agent管理器
// 负责管理所有Runner的生命周期
type Manager struct {
	config  *BackgroundConfig  // 后台Agent配置
	runners map[string]*Runner // Runner映射表，key为Runner ID
	db      *sql.DB            // 数据库连接
	wsHub   *websocket.Hub     // WebSocket Hub

	mu sync.RWMutex // 读写锁，保护runners映射表
}

// NewManager 创建管理器
// config: 后台Agent配置
// db: 数据库连接
// wsHub: WebSocket Hub
// 返回Manager实例
func NewManager(config *BackgroundConfig, db *sql.DB, wsHub *websocket.Hub) *Manager {
	manager := &Manager{
		config:  config,
		runners: make(map[string]*Runner),
		db:      db,
		wsHub:   wsHub,
	}

	logger.Info("创建BackgroundManager", map[string]interface{}{
		"global_enabled": config.Global.Enabled,
		"agent_count":    len(config.Agents),
	})

	return manager
}

// Start 启动所有启用的Agent
// 根据配置创建并启动所有启用的Runner
func (m *Manager) Start() {
	// 检查总开关
	if !m.config.Global.Enabled {
		logger.Info("后台Agent服务已禁用（global.enabled=false）", nil)
		return
	}

	logger.Info("开始启动后台Agent服务", map[string]interface{}{
		"total_agents": len(m.config.Agents),
	})

	// 统计启用的Agent数量
	enabledCount := 0

	// 遍历所有Agent配置
	for i := range m.config.Agents {
		agentConfig := &m.config.Agents[i]

		// 检查是否启用
		if !agentConfig.Enabled {
			logger.Info("Agent已跳过（禁用）", map[string]interface{}{
				"agent_id":   agentConfig.ID,
				"agent_name": agentConfig.Name,
			})
			continue
		}

		enabledCount++

		// 创建Runner
		logger.Info("创建Runner", map[string]interface{}{
			"agent_id":       agentConfig.ID,
			"agent_name":     agentConfig.Name,
			"interval":       agentConfig.Interval,
			"timeout":        agentConfig.Timeout,
			"max_iterations": agentConfig.MaxIterations,
		})

		runner := NewRunner(agentConfig, m.db, m.wsHub)

		// 保存到映射表
		m.mu.Lock()
		m.runners[agentConfig.ID] = runner
		m.mu.Unlock()

		// 启动Runner
		runner.Start()
	}

	logger.Info("后台Agent服务启动完成", map[string]interface{}{
		"total_agents":    len(m.config.Agents),
		"enabled_agents":  enabledCount,
		"started_runners": len(m.runners),
	})
}

// Stop 停止所有Runner
// 优雅停止所有正在运行的Runner
func (m *Manager) Stop() {
	logger.Info("开始停止后台Agent服务", map[string]interface{}{
		"runner_count": len(m.runners),
	})

	// 复制runners映射表，避免在遍历时修改
	m.mu.Lock()
	runners := make([]*Runner, 0, len(m.runners))
	for _, r := range m.runners {
		runners = append(runners, r)
	}
	// 清空映射表
	m.runners = make(map[string]*Runner)
	m.mu.Unlock()

	// 停止所有Runner
	for _, runner := range runners {
		logger.Info("停止Runner", map[string]interface{}{
			"runner_id":   runner.GetID(),
			"runner_name": runner.GetName(),
		})
		runner.Stop()
	}

	logger.Info("后台Agent服务已停止", map[string]interface{}{
		"stopped_runners": len(runners),
	})
}

// GetRunner 获取指定Runner
// id: Runner ID
// 返回Runner实例和是否找到
func (m *Manager) GetRunner(id string) (*Runner, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runner, ok := m.runners[id]
	return runner, ok
}

// GetRunnerStats 获取Runner统计
// id: Runner ID
// 返回Runner统计和是否找到
func (m *Manager) GetRunnerStats(id string) (RunnerStats, bool) {
	m.mu.RLock()
	runner, ok := m.runners[id]
	m.mu.RUnlock()

	if !ok {
		return RunnerStats{}, false
	}

	return runner.GetStats(), true
}

// GetAllStats 获取所有Runner统计
// 返回所有Runner的统计信息映射表
func (m *Manager) GetAllStats() map[string]RunnerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]RunnerStats)
	for id, runner := range m.runners {
		stats[id] = runner.GetStats()
	}

	return stats
}

// GetRunnerCount 获取Runner数量
// 返回当前管理的Runner数量
func (m *Manager) GetRunnerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.runners)
}

// GetRunningCount 获取运行中的Runner数量
// 返回正在运行的Runner数量
func (m *Manager) GetRunningCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, runner := range m.runners {
		if runner.IsRunning() {
			count++
		}
	}

	return count
}

// IsRunnerRunning 检查Runner是否正在运行
// id: Runner ID
// 返回是否正在运行
func (m *Manager) IsRunnerRunning(id string) bool {
	m.mu.RLock()
	runner, ok := m.runners[id]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	return runner.IsRunning()
}

// GetConfig 获取配置
// 返回后台Agent配置的副本
func (m *Manager) GetConfig() BackgroundConfig {
	return *m.config
}
