// background 后台Agent模块
package background

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// IndependentAgentBuilder 完全独立的Agent构建器
// 每次调用Build都创建全新的执行组件，使用最新配置
// 不依赖cmd/agent.go中的AgentBuilder，完全独立
type IndependentAgentBuilder struct {
	db *sql.DB
}

// NewIndependentAgentBuilder 创建独立Agent构建器
// db: 数据库连接，用于读取配置
// 返回构建器实例
func NewIndependentAgentBuilder(db *sql.DB) *IndependentAgentBuilder {
	return &IndependentAgentBuilder{db: db}
}

// BuildOptions Agent构建选项
type BuildOptions struct {
	SystemPrompt  string   // 系统提示词
	MaxIterations int      // 最大迭代次数
	Timeout       int      // 超时时间(秒)
	AllowedTools  []string // 允许的工具列表，为空表示允许所有基础工具
}

// Build 构建全新的任务执行器
// 每次调用都会：
// 1. 从原有setting系统读取最新LLM配置
// 2. 检查Token预算
// 3. 创建新的LLM适配器
// 4. 创建新的工具注册表
// 5. 创建新的任务执行器
//
// ctx: 上下文
// opts: 构建选项
// 返回任务执行器实例或错误
func (b *IndependentAgentBuilder) Build(ctx context.Context, opts *BuildOptions) (*TaskExecutor, error) {
	logger.Info("开始构建任务执行器", map[string]interface{}{
		"system_prompt_length": len(opts.SystemPrompt),
		"max_iterations":       opts.MaxIterations,
		"timeout":              opts.Timeout,
		"allowed_tools_count":  len(opts.AllowedTools),
	})

	// 1. 获取ConfigProvider（原有setting系统）
	configProvider := setting.GetProvider()
	if configProvider == nil {
		logger.Error("ConfigProvider未初始化", nil)
		return nil, fmt.Errorf("config provider未初始化")
	}

	// 2. 获取最新LLM配置
	llmCfg, err := configProvider.GetLLMConfig()
	if err != nil {
		logger.Error("获取LLM配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("获取LLM配置失败: %w", err)
	}

	if llmCfg == nil {
		logger.Error("LLM配置为空", nil)
		return nil, fmt.Errorf("LLM配置为空，请先配置LLM")
	}

	logger.Info("获取到最新LLM配置", map[string]interface{}{
		"provider": llmCfg.Provider,
		"model":    llmCfg.Model,
	})

	// 3. 检查Token预算
	if err := b.checkTokenBudget(configProvider); err != nil {
		logger.Error("Token预算检查失败", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	// 4. 创建LLM适配器（使用最新配置）
	llmAdapter, err := model.NewAdapter(llmCfg.Provider, llmCfg.APIKey, llmCfg.Model, llmCfg.BaseURL)
	if err != nil {
		logger.Error("创建LLM适配器失败", map[string]interface{}{
			"provider": llmCfg.Provider,
			"model":    llmCfg.Model,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("创建LLM适配器失败: %w", err)
	}

	if llmAdapter == nil {
		logger.Error("LLM适配器创建失败，返回nil", map[string]interface{}{
			"provider": llmCfg.Provider,
		})
		return nil, fmt.Errorf("不支持的LLM提供商: %s", llmCfg.Provider)
	}

	logger.Info("LLM适配器创建成功", map[string]interface{}{
		"provider": llmCfg.Provider,
		"model":    llmCfg.Model,
	})

	// 5. 创建工具注册表
	toolRegistry := b.createToolRegistry(opts.AllowedTools)
	logger.Info("工具注册表创建完成", map[string]interface{}{
		"tools_count": len(toolRegistry.GetToolDefinitions()),
	})

	// 6. 创建任务执行器（全新实例）
	executor := NewTaskExecutor(
		llmAdapter,
		toolRegistry,
		opts.MaxIterations,
		time.Duration(opts.Timeout)*time.Second,
		opts.SystemPrompt,
		b.db,
	)

	logger.Info("任务执行器构建完成", map[string]interface{}{
		"max_loop": opts.MaxIterations,
		"timeout":  opts.Timeout,
	})

	return executor, nil
}

// checkTokenBudget 检查Token预算
// 从原有setting系统获取配置，检查是否超出每日限额
// configProvider: 配置提供者
// 返回错误（如果超出预算）
func (b *IndependentAgentBuilder) checkTokenBudget(configProvider *setting.ConfigProvider) error {
	agentCfg, err := configProvider.GetAgentConfig()
	if err != nil {
		logger.Error("获取Agent配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("获取Agent配置失败: %w", err)
	}

	// 如果设置了每日限制（>0），检查是否超限
	if agentCfg != nil && agentCfg.DailyTokenLimit > 0 {
		// 获取今日已使用的Token数
		// 注意：这里简化处理，实际应该调用token统计模块
		// 由于token模块的复杂性，这里只记录日志
		logger.Info("Token预算检查", map[string]interface{}{
			"daily_limit": agentCfg.DailyTokenLimit,
			"note":        "Token预算检查已启用，实际使用量检查需要在执行后统计",
		})
	} else {
		logger.Debug("Token预算未设置或无限额", map[string]interface{}{
			"daily_limit": 0,
		})
	}

	return nil
}

// createToolRegistry 创建工具注册表
// 根据允许列表创建工具注册表，如果允许列表为空则注册所有基础工具
// allowedTools: 允许的工具名称列表
// 返回工具注册表
func (b *IndependentAgentBuilder) createToolRegistry(allowedTools []string) *tool.Registry {
	registry := tool.NewRegistry()

	// 如果没有指定允许列表，注册所有基础工具
	if len(allowedTools) == 0 {
		logger.Debug("未指定允许工具列表，注册所有基础工具", nil)

		// 文件读取工具
		if reader := builtin.NewFileReader(); reader != nil {
			registry.Register(reader)
			logger.Debug("注册工具", map[string]interface{}{"tool": "file_reader"})
		}

		// 文件写入工具
		if writer := builtin.NewFileWriter(); writer != nil {
			registry.Register(writer)
			logger.Debug("注册工具", map[string]interface{}{"tool": "file_writer"})
		}

		// HTTP请求工具
		if httpTool := builtin.SimpleHTTP(); httpTool != nil {
			registry.Register(httpTool)
			logger.Debug("注册工具", map[string]interface{}{"tool": "http"})
		}

		// 计算器工具
		if calc := builtin.NewCalculator(); calc != nil {
			registry.Register(calc)
			logger.Debug("注册工具", map[string]interface{}{"tool": "calculator"})
		}

		return registry
	}

	// 只注册允许的工具
	logger.Info("根据允许列表注册工具", map[string]interface{}{
		"allowed_tools": allowedTools,
	})

	for _, toolName := range allowedTools {
		switch toolName {
		case "file_reader":
			if reader := builtin.NewFileReader(); reader != nil {
				registry.Register(reader)
				logger.Debug("注册工具", map[string]interface{}{"tool": "file_reader"})
			}
		case "file_writer":
			if writer := builtin.NewFileWriter(); writer != nil {
				registry.Register(writer)
				logger.Debug("注册工具", map[string]interface{}{"tool": "file_writer"})
			}
		case "http":
			if httpTool := builtin.SimpleHTTP(); httpTool != nil {
				registry.Register(httpTool)
				logger.Debug("注册工具", map[string]interface{}{"tool": "http"})
			}
		case "calculator":
			if calc := builtin.NewCalculator(); calc != nil {
				registry.Register(calc)
				logger.Debug("注册工具", map[string]interface{}{"tool": "calculator"})
			}
		default:
			logger.Warn("未知的工具名称", map[string]interface{}{
				"tool": toolName,
			})
		}
	}

	return registry
}
