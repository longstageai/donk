// background 后台Agent模块
// 提供配置文件驱动的后台Agent服务
package background

import (
	"fmt"
	"os"

	"github.com/longstageai/donk/donk/pkg/logger"
	"gopkg.in/yaml.v3"
)

// BackgroundConfig 后台Agent总配置
// 对应background.yaml文件结构
type BackgroundConfig struct {
	Global GlobalConfig  `yaml:"global"` // 全局配置
	Agents []AgentConfig `yaml:"agents"` // Agent配置列表
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	Enabled         bool `yaml:"enabled"`          // 总开关
	DefaultInterval int  `yaml:"default_interval"` // 默认执行间隔(秒)
}

// AgentConfig 单个后台Agent配置
type AgentConfig struct {
	ID            string            `yaml:"id"`             // 唯一标识
	Name          string            `yaml:"name"`           // 显示名称
	Enabled       bool              `yaml:"enabled"`        // 是否启用
	Interval      int               `yaml:"interval"`       // 执行间隔(秒)
	Timeout       int               `yaml:"timeout"`        // 单次任务超时(秒)
	MaxIterations int               `yaml:"max_iterations"` // 最大迭代次数
	SystemPrompt  string            `yaml:"system_prompt"`  // 系统提示词
	AllowedTools  []string          `yaml:"allowed_tools"`  // 允许的工具列表
	Variables     map[string]string `yaml:"variables"`      // 自定义变量
}

// LoadConfig 从文件加载配置
// path: 配置文件路径
// 返回解析后的配置对象或错误
func LoadConfig(path string) (*BackgroundConfig, error) {
	logger.Info("加载后台Agent配置文件", map[string]interface{}{
		"path": path,
	})

	// 读取文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("读取配置文件失败", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config BackgroundConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		logger.Error("解析配置文件失败", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	config.setDefaults()

	// 验证配置
	if err := config.validate(); err != nil {
		logger.Error("配置验证失败", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	logger.Info("配置文件加载成功", map[string]interface{}{
		"global_enabled":   config.Global.Enabled,
		"agent_count":      len(config.Agents),
		"default_interval": config.Global.DefaultInterval,
	})

	return &config, nil
}

// setDefaults 设置默认值
// 为未配置或配置不合理的字段设置默认值
func (c *BackgroundConfig) setDefaults() {
	// 设置全局默认间隔
	if c.Global.DefaultInterval <= 0 {
		c.Global.DefaultInterval = 300 // 默认5分钟
		logger.Debug("使用默认全局间隔", map[string]interface{}{
			"default_interval": c.Global.DefaultInterval,
		})
	}

	// 为每个Agent设置默认值
	for i := range c.Agents {
		// 默认启用
		if !c.Agents[i].Enabled && c.Agents[i].ID != "" {
			// 如果ID存在但Enabled未显式设置，yaml会解析为false
			// 这里不做处理，保持用户显式控制
		}

		// 默认间隔
		if c.Agents[i].Interval <= 0 {
			c.Agents[i].Interval = c.Global.DefaultInterval
			logger.Debug("Agent使用默认间隔", map[string]interface{}{
				"agent_id": c.Agents[i].ID,
				"interval": c.Agents[i].Interval,
			})
		}

		// 默认超时
		if c.Agents[i].Timeout <= 0 {
			c.Agents[i].Timeout = 60 // 默认1分钟
			logger.Debug("Agent使用默认超时", map[string]interface{}{
				"agent_id": c.Agents[i].ID,
				"timeout":  c.Agents[i].Timeout,
			})
		}

		// 默认最大迭代次数
		if c.Agents[i].MaxIterations <= 0 {
			c.Agents[i].MaxIterations = 10
			logger.Debug("Agent使用默认最大迭代次数", map[string]interface{}{
				"agent_id":       c.Agents[i].ID,
				"max_iterations": c.Agents[i].MaxIterations,
			})
		}
	}
}

// validate 验证配置
// 检查配置是否合法，返回验证错误
func (c *BackgroundConfig) validate() error {
	// 检查Agent ID唯一性
	ids := make(map[string]bool)
	for _, agent := range c.Agents {
		// 检查ID不能为空
		if agent.ID == "" {
			logger.Error("Agent配置错误", map[string]interface{}{
				"error": "agent id不能为空",
			})
			return fmt.Errorf("agent id不能为空")
		}

		// 检查ID是否重复
		if ids[agent.ID] {
			logger.Error("Agent配置错误", map[string]interface{}{
				"agent_id": agent.ID,
				"error":    "agent id重复",
			})
			return fmt.Errorf("agent id重复: %s", agent.ID)
		}
		ids[agent.ID] = true

		// 检查系统提示词不能为空
		if agent.SystemPrompt == "" {
			logger.Error("Agent配置错误", map[string]interface{}{
				"agent_id": agent.ID,
				"error":    "system_prompt不能为空",
			})
			return fmt.Errorf("agent %s 的system_prompt不能为空", agent.ID)
		}
	}

	return nil
}

// GetAgentConfig 获取指定Agent配置
// id: Agent ID
// 返回Agent配置和是否找到
func (c *BackgroundConfig) GetAgentConfig(id string) (*AgentConfig, bool) {
	for i := range c.Agents {
		if c.Agents[i].ID == id {
			return &c.Agents[i], true
		}
	}
	return nil, false
}

// GetEnabledAgents 获取所有启用的Agent配置
// 返回启用的Agent配置列表
func (c *BackgroundConfig) GetEnabledAgents() []AgentConfig {
	var enabled []AgentConfig
	for _, agent := range c.Agents {
		if agent.Enabled {
			enabled = append(enabled, agent)
		}
	}
	return enabled
}
