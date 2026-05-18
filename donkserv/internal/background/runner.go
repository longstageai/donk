// background 后台Agent模块
package background

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/internal/websocket"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Runner 后台Agent运行器
// 负责按配置循环执行单个Agent任务
type Runner struct {
	id     string       // Runner ID（同Agent ID）
	config *AgentConfig // Agent配置

	// 依赖
	db      *sql.DB
	builder *IndependentAgentBuilder
	wsHub   *websocket.Hub

	// 控制
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex

	// 统计
	stats RunnerStats
}

// RunnerStats 运行统计
type RunnerStats struct {
	RunCount     int       // 总执行次数
	SuccessCount int       // 成功次数
	FailCount    int       // 失败次数
	LastRunAt    time.Time // 上次执行时间
	TotalTokens  int       // 累计Token消耗
}

// NewRunner 创建Runner
// config: Agent配置
// db: 数据库连接
// wsHub: WebSocket Hub
// 返回Runner实例
func NewRunner(config *AgentConfig, db *sql.DB, wsHub *websocket.Hub) *Runner {
	ctx, cancel := context.WithCancel(context.Background())

	runner := &Runner{
		id:      config.ID,
		config:  config,
		db:      db,
		builder: NewIndependentAgentBuilder(db),
		wsHub:   wsHub,
		ctx:     ctx,
		cancel:  cancel,
	}

	logger.Info("创建Runner实例", map[string]interface{}{
		"runner_id":   runner.id,
		"runner_name": runner.config.Name,
		"interval":    runner.config.Interval,
		"timeout":     runner.config.Timeout,
	})

	return runner
}

// Start 启动Runner
// 启动后台循环，按配置间隔执行任务
func (r *Runner) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		logger.Warn("Runner已在运行中", map[string]interface{}{
			"runner_id": r.id,
		})
		return
	}
	r.running = true
	r.mu.Unlock()

	r.wg.Add(1)
	go r.runLoop()

	logger.Info("Runner启动成功", map[string]interface{}{
		"runner_id":   r.id,
		"runner_name": r.config.Name,
		"interval":    r.config.Interval,
	})
}

// Stop 停止Runner
// 优雅停止，等待当前任务完成后退出
func (r *Runner) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		logger.Warn("Runner未在运行", map[string]interface{}{
			"runner_id": r.id,
		})
		return
	}
	r.running = false
	r.mu.Unlock()

	// 发送取消信号
	r.cancel()

	// 等待循环退出
	r.wg.Wait()

	logger.Info("Runner已停止", map[string]interface{}{
		"runner_id":   r.id,
		"runner_name": r.config.Name,
		"stats": map[string]interface{}{
			"run_count":     r.stats.RunCount,
			"success_count": r.stats.SuccessCount,
			"fail_count":    r.stats.FailCount,
		},
	})
}

// IsRunning 检查Runner是否正在运行
// 返回运行状态
func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// runLoop 执行循环
// 按配置间隔循环执行任务
func (r *Runner) runLoop() {
	defer r.wg.Done()

	logger.Info("Runner循环开始", map[string]interface{}{
		"runner_id": r.id,
	})

	// 首次立即执行一次
	logger.Info("执行首次任务", map[string]interface{}{
		"runner_id": r.id,
	})
	//r.executeOnce()

	// 创建定时器
	ticker := time.NewTicker(time.Duration(r.config.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			logger.Info("Runner收到停止信号，退出循环", map[string]interface{}{
				"runner_id": r.id,
			})
			return
		case <-ticker.C:
			logger.Debug("Runner定时触发", map[string]interface{}{
				"runner_id": r.id,
			})
			r.executeOnce()
		}
	}
}

// executeOnce 执行单次任务
// 创建Agent、执行任务、推送结果、更新统计
func (r *Runner) executeOnce() {
	startTime := time.Now()

	logger.Info("开始执行任务", map[string]interface{}{
		"runner_id":   r.id,
		"runner_name": r.config.Name,
	})

	// 1. 构建任务执行器（全新实例，使用最新配置）
	logger.Debug("构建任务执行器", map[string]interface{}{
		"runner_id": r.id,
	})

	executor, err := r.builder.Build(r.ctx, &BuildOptions{
		SystemPrompt:  r.config.SystemPrompt,
		MaxIterations: r.config.MaxIterations,
		Timeout:       r.config.Timeout,
		AllowedTools:  r.config.AllowedTools,
	})
	if err != nil {
		logger.Error("任务执行器构建失败", map[string]interface{}{
			"runner_id": r.id,
			"error":     err.Error(),
		})
		r.pushError("executor_build_failed", err.Error(), startTime)
		r.updateStats(false, 0)
		return
	}

	logger.Info("任务执行器构建成功，开始执行任务", map[string]interface{}{
		"runner_id": r.id,
	})

	// 2. 执行任务（带超时）
	taskCtx, cancel := context.WithTimeout(r.ctx, time.Duration(r.config.Timeout)*time.Second)
	logger.Debug("执行任务", map[string]interface{}{
		"runner_id": r.id,
		"timeout":   r.config.Timeout,
	})

	// 使用任务执行器执行任务
	execResult := executor.Execute(taskCtx)
	cancel()

	// 3. 立即释放执行器（不保留引用，让GC回收）
	logger.Debug("释放任务执行器", map[string]interface{}{
		"runner_id": r.id,
	})
	executor = nil

	// 4. 计算执行耗时和结果
	duration := execResult.Duration
	success := execResult.Error == ""
	tokenUsage := execResult.TokenUsage.TotalTokens // 从执行结果获取实际Token消耗

	// 5. 更新统计（包含Token消耗）
	r.updateStats(success, tokenUsage)

	// 6. 推送结果（包含详细的执行信息）
	if !success {
		logger.Error("任务执行失败", map[string]interface{}{
			"runner_id":  r.id,
			"error":      execResult.Error,
			"duration":   duration.Milliseconds(),
			"iterations": execResult.Iterations,
			"tokens":     tokenUsage,
		})
		r.pushErrorWithDetails(execResult, startTime)
	} else {
		logger.Info("任务执行成功", map[string]interface{}{
			"runner_id":  r.id,
			"duration":   duration.Milliseconds(),
			"iterations": execResult.Iterations,
			"tokens":     tokenUsage,
		})
		r.pushSuccessWithDetails(execResult, startTime)
	}
}

// pushSuccess 推送成功结果
// output: 执行输出
// duration: 执行耗时
// tokens: Token消耗
// startTime: 开始时间
func (r *Runner) pushSuccess(output string, duration time.Duration, tokens int, startTime time.Time) {
	//msg := TaskCompleteMessage{
	//	Type:       MessageTypeTaskComplete,
	//	RunnerID:   r.id,
	//	RunnerName: r.config.Name,
	//	Status:     "success",
	//	Output:     output,
	//	Duration:   duration.Milliseconds(),
	//	Tokens:     tokens,
	//	Timestamp:  time.Now().Unix(),
	//}
	msg := websocket.NewNotification(MessageTypeTaskComplete, r.config.Name, output)
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化消息失败", map[string]interface{}{
			"runner_id": r.id,
			"error":     err.Error(),
		})
		return
	}

	// 广播消息
	if r.wsHub != nil {
		r.wsHub.BroadcastJSON(data)
		logger.Debug("成功推送任务完成消息", map[string]interface{}{
			"runner_id": r.id,
			"clients":   r.wsHub.ClientCount(),
		})
	} else {
		logger.Warn("WebSocket Hub未设置，无法推送消息", map[string]interface{}{
			"runner_id": r.id,
		})
	}
}

// pushError 推送错误
// errorType: 错误类型
// errorMsg: 错误信息
// startTime: 开始时间
func (r *Runner) pushError(errorType, errorMsg string, startTime time.Time) {
	duration := time.Since(startTime)

	msg := TaskCompleteMessage{
		Type:       MessageTypeTaskError,
		RunnerID:   r.id,
		RunnerName: r.config.Name,
		Status:     "failed",
		ErrorType:  errorType,
		Error:      errorMsg,
		Duration:   duration.Milliseconds(),
		Timestamp:  time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化错误消息失败", map[string]interface{}{
			"runner_id": r.id,
			"error":     err.Error(),
		})
		return
	}

	// 广播消息
	if r.wsHub != nil {
		r.wsHub.BroadcastJSON(data)
		logger.Debug("成功推送任务错误消息", map[string]interface{}{
			"runner_id":  r.id,
			"error_type": errorType,
			"clients":    r.wsHub.ClientCount(),
		})
	} else {
		logger.Warn("WebSocket Hub未设置，无法推送错误消息", map[string]interface{}{
			"runner_id": r.id,
		})
	}
}

// updateStats 更新统计
// success: 是否成功
// tokens: Token消耗
func (r *Runner) updateStats(success bool, tokens int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stats.RunCount++
	r.stats.LastRunAt = time.Now()
	r.stats.TotalTokens += tokens

	if success {
		r.stats.SuccessCount++
	} else {
		r.stats.FailCount++
	}

	logger.Debug("更新Runner统计", map[string]interface{}{
		"runner_id":     r.id,
		"run_count":     r.stats.RunCount,
		"success_count": r.stats.SuccessCount,
		"fail_count":    r.stats.FailCount,
		"total_tokens":  r.stats.TotalTokens,
	})
}

// pushSuccessWithDetails 推送成功结果（包含详细信息）
// execResult: 执行结果
// startTime: 开始时间
func (r *Runner) pushSuccessWithDetails(execResult *ExecutionResult, startTime time.Time) {
	//msg := TaskCompleteMessage{
	//	Type:             MessageTypeTaskComplete,
	//	RunnerID:         r.id,
	//	RunnerName:       r.config.Name,
	//	Status:           "success",
	//	Output:           execResult.Output,
	//	Duration:         execResult.Duration.Milliseconds(),
	//	Tokens:           execResult.TokenUsage.TotalTokens,
	//	Iterations:       execResult.Iterations,
	//	PromptTokens:     execResult.TokenUsage.PromptTokens,
	//	CompletionTokens: execResult.TokenUsage.CompletionTokens,
	//	TotalTokens:      execResult.TokenUsage.TotalTokens,
	//	Timestamp:        time.Now().Unix(),
	//}

	msg := websocket.NewNotification(MessageTypeTaskComplete, r.config.Name, execResult.Output)

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化消息失败", map[string]interface{}{
			"runner_id": r.id,
			"error":     err.Error(),
		})
		return
	}

	// 广播消息
	if r.wsHub != nil {
		r.wsHub.BroadcastJSON(data)
		logger.Debug("成功推送任务完成消息", map[string]interface{}{
			"runner_id":     r.id,
			"clients":       r.wsHub.ClientCount(),
			"iterations":    execResult.Iterations,
			"prompt_tokens": execResult.TokenUsage.PromptTokens,
			"output_tokens": execResult.TokenUsage.CompletionTokens,
			"total_tokens":  execResult.TokenUsage.TotalTokens,
		})
	} else {
		logger.Warn("WebSocket Hub未设置，无法推送消息", map[string]interface{}{
			"runner_id": r.id,
		})
	}
}

// pushErrorWithDetails 推送错误（包含详细信息）
// execResult: 执行结果
// startTime: 开始时间
func (r *Runner) pushErrorWithDetails(execResult *ExecutionResult, startTime time.Time) {
	msg := TaskCompleteMessage{
		Type:             MessageTypeTaskError,
		RunnerID:         r.id,
		RunnerName:       r.config.Name,
		Status:           "failed",
		Error:            execResult.Error,
		Duration:         execResult.Duration.Milliseconds(),
		Tokens:           execResult.TokenUsage.TotalTokens,
		Iterations:       execResult.Iterations,
		PromptTokens:     execResult.TokenUsage.PromptTokens,
		CompletionTokens: execResult.TokenUsage.CompletionTokens,
		TotalTokens:      execResult.TokenUsage.TotalTokens,
		Timestamp:        time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("序列化错误消息失败", map[string]interface{}{
			"runner_id": r.id,
			"error":     err.Error(),
		})
		return
	}

	// 广播消息
	if r.wsHub != nil {
		r.wsHub.BroadcastJSON(data)
		logger.Debug("成功推送任务错误消息", map[string]interface{}{
			"runner_id":    r.id,
			"error":        execResult.Error,
			"clients":      r.wsHub.ClientCount(),
			"iterations":   execResult.Iterations,
			"total_tokens": execResult.TokenUsage.TotalTokens,
		})
	} else {
		logger.Warn("WebSocket Hub未设置，无法推送错误消息", map[string]interface{}{
			"runner_id": r.id,
		})
	}
}

// GetStats 获取统计
// 返回Runner统计信息的副本
func (r *Runner) GetStats() RunnerStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

// GetID 获取Runner ID
// 返回Runner唯一标识
func (r *Runner) GetID() string {
	return r.id
}

// GetName 获取Runner名称
// 返回Runner显示名称
func (r *Runner) GetName() string {
	return r.config.Name
}

// GetConfig 获取Runner配置
// 返回Runner配置的副本
func (r *Runner) GetConfig() AgentConfig {
	return *r.config
}
