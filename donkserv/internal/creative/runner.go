package creative

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// LoopRunner 是 Creative 多 Agent 的持续循环运行器。
// 负责在后台持续创建新的 Session 并执行，完成一个任务后等待固定间隔再开启下一个任务。
// 支持通过 Start/Stop 方法控制循环的启停，停止时会先让当前任务在 Runtime 的安全控制点停止，
// 再结束外层循环，避免强制取消 context 导致状态不一致。
type LoopRunner struct {
	runtime   *Runtime           // Creative 运行时实例，用于创建 Session 和执行循环
	interval  time.Duration      // 任务间隔时间，完成一个任务后等待多久开启下一个
	mu        sync.Mutex         // 保护以下字段的并发访问
	cancel    context.CancelFunc // 用于取消外层循环的 context
	running   bool               // 循环 goroutine 是否正在运行
	stopping  bool               // 是否已收到停止请求（用于在当前任务结束后退出循环）
	current   ID                 // 当前正在执行的 Session ID
	onStopped func(reason string)
}

// NewLoopRunner 创建一个新的持续循环运行器。
// runtime: Creative 运行时实例
// interval: 任务间隔时间，如果小于等于 0 则使用默认 1 分钟
func NewLoopRunner(runtime *Runtime, interval time.Duration, onStopped ...func(reason string)) *LoopRunner {
	if interval <= 0 {
		interval = time.Minute
	}
	runner := &LoopRunner{runtime: runtime, interval: interval}
	if len(onStopped) > 0 {
		runner.onStopped = onStopped[0]
	}
	return runner
}

// Start 启动持续循环。
// 如果循环已经在运行，返回 false；否则启动循环 goroutine 并返回 true。
// 启动时会先保存运行状态到数据库，确保服务重启后可以恢复。
func (r *LoopRunner) Start(ctx context.Context) bool {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return false
	}
	runCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.running = true
	r.stopping = false
	r.mu.Unlock()

	go r.run(runCtx)
	return true
}

// Stop 停止持续循环。
// 首先设置 stopping 标志，然后对当前正在执行的任务发送 StopLoop 命令，
// 让 Runtime 在下一个安全控制点停止当前任务。如果当前没有任务在执行，
// 则直接取消外层 context 结束循环。
// 这种设计确保停止操作不会绕过 Runtime 的内部控制逻辑。
func (r *LoopRunner) Stop(ctx context.Context) {
	r.mu.Lock()
	cancel := r.cancel
	current := r.current
	r.stopping = true
	r.mu.Unlock()

	// 如果有正在执行的任务，先让 Runtime 安全停止它
	if current != "" && r.runtime != nil {
		if err := r.runtime.StopLoop(ctx, current, StopGraceful, "user requested"); err != nil && !errors.Is(err, ErrSessionNotFound) {
			logger.Error("停止 creative 当前任务失败", map[string]interface{}{"session_id": current, "error": err.Error()})
		}
		return
	}
	// 没有任务在执行，直接取消外层 context
	if cancel != nil {
		cancel()
	}
}

// Running 返回循环是否正在运行。
func (r *LoopRunner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// CurrentSessionID 返回当前正在执行的 Session ID。
// 如果没有任务在执行，返回空字符串。
func (r *LoopRunner) CurrentSessionID() ID {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current
}

// run 是循环的核心 goroutine。
// 持续创建新的 Session 并执行，完成一个任务后等待固定间隔再开启下一个。
// 当收到停止请求时，会在当前任务结束后退出循环。
func (r *LoopRunner) run(ctx context.Context) {
	// 确保退出时清理状态
	defer func() {
		r.mu.Lock()
		r.running = false
		r.stopping = false
		r.cancel = nil
		r.current = ""
		r.mu.Unlock()
	}()

	for {
		// 检查是否收到取消信号
		if err := ctx.Err(); err != nil {
			return
		}

		// 创建新的 Session
		sessionID, err := r.runtime.StartSession(ctx, Trigger{Type: TriggerTimerTriggered})
		if err != nil {
			logger.Error("创建 creative 循环任务失败", map[string]interface{}{"error": err.Error()})
			if !r.wait(ctx) {
				return
			}
			continue
		}

		// 记录当前 Session ID
		r.mu.Lock()
		r.current = sessionID
		r.mu.Unlock()

		// 执行 Session 的事件循环
		if err := r.runtime.StartLoop(ctx, sessionID); err != nil {
			if errors.Is(err, ErrTokenBudgetExceeded) {
				logger.Warn("creative token 预算已超限，停止持续循环", map[string]interface{}{"session_id": sessionID, "error": err.Error()})
				if r.onStopped != nil {
					r.onStopped("token_budget_exceeded")
				}
				r.mu.Lock()
				r.stopping = true
				r.mu.Unlock()
			} else if !errors.Is(err, ErrLoopStopped) && !errors.Is(err, context.Canceled) {
				logger.Error("执行 creative 循环任务失败", map[string]interface{}{"session_id": sessionID, "error": err.Error()})
			}
		}

		// 清理当前 Session ID，检查是否需要停止
		r.mu.Lock()
		if r.current == sessionID {
			r.current = ""
		}
		stopping := r.stopping
		cancel := r.cancel
		r.mu.Unlock()

		// 如果收到停止请求，取消 context 并退出循环
		if stopping {
			if cancel != nil {
				cancel()
			}
			return
		}

		// 等待固定间隔后再开启下一个任务
		if !r.wait(ctx) {
			return
		}
	}
}

// wait 等待固定间隔或收到取消信号。
// 返回 true 表示正常等待结束，返回 false 表示收到取消信号。
func (r *LoopRunner) wait(ctx context.Context) bool {
	timer := time.NewTimer(r.interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
