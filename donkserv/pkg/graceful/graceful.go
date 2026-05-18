package graceful

import (
	"context"
	"fmt"
	"github.com/longstageai/donk/donk/configs"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// AppContext 应用上下文接口
// 用于在任务中获取应用上下文，避免循环导入
type AppContext interface {
	ConfigBean() *configs.Conf
	AppName() string
	Version() string
	Env() string
}

// Task 任务函数类型定义
// 参数 context 用于接收取消信号，实现优雅退出
// 参数 app 为 Application 实例，可用于获取配置、日志等
type Task func(ctx context.Context, app AppContext) error

// TaskConfig 任务配置结构体
type TaskConfig struct {
	Name    string        // 任务名称，用于标识和日志输出
	Handler Task          // 任务处理函数
	Timeout time.Duration // 任务超时时间，0表示无超时限制
}

// TaskResult 任务执行结果结构体
type TaskResult struct {
	Name      string    // 任务名称
	Error     error     // 任务执行错误
	StartTime time.Time // 任务开始时间
	EndTime   time.Time // 任务结束时间
}

// Runner 优雅退出运行器，管理多个并发任务的启动、运行和退出
type Runner struct {
	tasks           []TaskConfig       // 已注册的任务列表
	shutdownSignals []os.Signal        // 需要捕获的退出信号列表
	results         []TaskResult       // 任务执行结果列表
	mu              sync.Mutex         // 保护 results 的互斥锁
	ctx             context.Context    // 主上下文，用于传递取消信号
	cancel          context.CancelFunc // 取消函数，用于触发所有任务停止
	started         atomic.Bool        // 标记是否已启动，防止重复启动
	wg              sync.WaitGroup     // 用于等待所有任务完成
	sigChan         chan os.Signal     // 信号通道
	app             AppContext         // Application 实例，传递给任务
}

// Option 运行器配置选项函数类型
type Option func(*Runner)

// WithShutdownSignals 配置自定义退出信号
// 默认捕获 os.Interrupt (Ctrl+C) 和 syscall.SIGTERM
// 示例: graceful.WithShutdownSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
func WithShutdownSignals(signals ...os.Signal) Option {
	return func(r *Runner) {
		r.shutdownSignals = signals
	}
}

// WithApp 设置 Application 实例
// 用于在任务中获取应用上下文
func WithApp(app AppContext) Option {
	return func(r *Runner) {
		r.app = app
	}
}

// New 创建新的运行器实例
// 可传入配置选项自定义行为
// 示例:
//
//	runner := graceful.New() // 使用默认配置
//	runner := graceful.New(graceful.WithShutdownSignals(syscall.SIGTERM)) // 自定义信号
func New(opts ...Option) *Runner {
	r := &Runner{
		shutdownSignals: []os.Signal{os.Interrupt, syscall.SIGTERM},
		results:         make([]TaskResult, 0),
	}
	for _, opt := range opts {
		opt(r)
	}
	if len(r.shutdownSignals) == 0 {
		r.shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
	}
	return r
}

// Register 注册一个带超时时间的任务
// 参数:
//   - name: 任务名称，用于标识和日志输出
//   - handler: 任务处理函数，需要监听 context.Done() 以支持优雅退出
//   - timeout: 任务超时时间，设置后任务超过该时间会自动停止
//
// 示例:
//
//	runner.Register("database-sync", func(ctx context.Context) error {
//	    return syncDatabase(ctx)
//	}, 30 * time.Second)
func (r *Runner) Register(name string, handler Task, timeout time.Duration) *Runner {
	r.tasks = append(r.tasks, TaskConfig{
		Name:    name,
		Handler: handler,
		Timeout: timeout,
	})
	return r
}

// RegisterWithOpt 使用选项模式注册任务（超时时间可选）
// 参数:
//   - name: 任务名称
//   - handler: 任务处理函数
//   - opts: 可选配置，目前支持 graceful.WithTimeout()
//
// 不设置超时示例:
//
//	runner.RegisterWithOpt("long-running-task", handler)
//
// 设置超时示例:
//
//	runner.RegisterWithOpt("timeout-task", handler, graceful.WithTimeout(5 * time.Second))
func (r *Runner) RegisterWithOpt(name string, handler Task, opts ...TaskOption) *Runner {
	var timeout time.Duration
	for _, opt := range opts {
		timeout = opt(timeout)
	}
	return r.Register(name, handler, timeout)
}

// TaskOption 任务配置选项函数类型
type TaskOption func(time.Duration) time.Duration

// WithTimeout 设置任务超时时间选项
// 与 RegisterWithOpt 配合使用
// 示例:
//
//	runner.RegisterWithOpt("my-task", handler, graceful.WithTimeout(10 * time.Second))
func WithTimeout(timeout time.Duration) TaskOption {
	return func(t time.Duration) time.Duration {
		return timeout
	}
}

// Run 启动所有已注册的任务并等待执行完成
// 该方法会阻塞直到:
//  1. 所有任务执行完成（成功或失败）
//  2. 收到退出信号（SIGINT/SIGTERM），触发优雅退出
//  3. 发生错误（如果有任务返回错误）
//
// 内部逻辑:
//   - 首先检查是否已启动，防止重复调用
//   - 创建上下文和取消函数，用于传递取消信号给所有任务
//   - 设置信号通道，捕获系统退出信号（Ctrl+C、TERM等）
//   - 如果有注册的任务，在后台goroutine中运行任务
//   - 主循环监控任务完成信号和退出信号:
//   - 如果任务完成且有错误，立即返回错误
//   - 如果收到退出信号，触发取消并等待所有任务完成
//   - 如果没有任务，直接等待退出信号
//
// 返回值:
//   - nil: 所有任务成功完成或收到退出信号后正常退出
//   - error: 任务执行过程中的错误（不会触发优雅退出）
func (r *Runner) Run() error {
	// 检查是否已启动，防止重复调用
	if r.started.Swap(true) {
		return fmt.Errorf("runner already started")
	}

	// 创建主上下文和取消函数，用于传递取消信号
	r.ctx, r.cancel = context.WithCancel(context.Background())
	// 确保函数返回时清理上下文
	defer r.cancel()

	// 创建信号通道并注册退出信号
	// 默认捕获 os.Interrupt (Ctrl+C) 和 syscall.SIGTERM
	r.sigChan = make(chan os.Signal, 1)
	signal.Notify(r.sigChan, r.shutdownSignals...)
	// 函数返回时停止监听信号
	defer signal.Stop(r.sigChan)

	// 创建错误通道，用于接收任务执行结果
	errChan := make(chan error, 1)
	// 检查是否有注册的任务
	hasTasks := len(r.tasks) > 0

	// 如果有任务，在后台goroutine中执行
	if hasTasks {
		go func() {
			errChan <- r.runTasks()
		}()
	}

	// 主循环：监控任务完成和退出信号
	for {
		if hasTasks {
			// 有任务时的处理逻辑
			select {
			// 任务完成信号
			case err := <-errChan:
				// 如果任务执行出错，立即返回错误
				// 这不会触发优雅退出流程
				if err != nil {
					return err
				}
				// 任务成功完成，标记为无任务状态
				hasTasks = false

			// 退出信号（Ctrl+C / SIGTERM）
			case <-r.sigChan:
				//fmt.Println("\nReceived shutdown signal, initiating graceful shutdown...")
				// 触发取消，通知所有任务停止
				r.cancel()
				// 等待任务完成（可能返回错误但不处理）
				<-errChan
				// 等待所有goroutine完成
				r.wg.Wait()
				// 正常退出
				return nil
			}
		} else {
			// 无任务时的处理逻辑
			select {
			// 退出信号（直接返回，不等待任务）
			case <-r.sigChan:
				//fmt.Println("\nReceived shutdown signal, initiating graceful shutdown...")
				// 触发取消（虽然没有任务，但保持一致性）
				r.cancel()
				return nil
			}
		}
	}
}

// runTasks 内部方法：使用 errgroup 执行所有任务
func (r *Runner) runTasks() error {
	if len(r.tasks) == 0 {
		return nil
	}

	g, ctx := errgroup.WithContext(r.ctx)

	for _, task := range r.tasks {
		task := task
		r.wg.Add(1)

		g.Go(func() error {
			defer r.wg.Done()
			return r.runTask(ctx, task)
		})
	}

	err := g.Wait()
	r.collectResults()
	return err
}

// runTask 内部方法：执行单个任务并记录结果
func (r *Runner) runTask(ctx context.Context, task TaskConfig) error {
	startTime := time.Now()

	taskCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	err := task.Handler(taskCtx, r.app)

	endTime := time.Now()

	r.mu.Lock()
	r.results = append(r.results, TaskResult{
		Name:      task.Name,
		Error:     err,
		StartTime: startTime,
		EndTime:   endTime,
	})
	r.mu.Unlock()

	return err
}

// collectResults 内部方法：收集并打印所有任务的执行结果
func (r *Runner) collectResults() {
	fmt.Println("\n=== Task Execution Results ===")
	hasError := false
	for _, result := range r.results {
		status := "SUCCESS"
		if result.Error != nil {
			status = "FAILED"
			hasError = true
		}
		duration := result.EndTime.Sub(result.StartTime)
		fmt.Printf("Task: %-20s Status: %-10s Duration: %v\n", result.Name, status, duration)
		if result.Error != nil {
			fmt.Printf("  Error: %v\n", result.Error)
		}
	}
	fmt.Println("===============================")

	if hasError {
		fmt.Println("Some tasks failed during execution")
	}
}

// GetResults 获取所有任务的执行结果
// 返回值是一个 TaskResult 切片，包含每个任务的名称、错误信息和执行时间
// 线程安全，可以随时调用
func (r *Runner) GetResults() []TaskResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	results := make([]TaskResult, len(r.results))
	copy(results, r.results)
	return results
}

// Wait 等待所有任务完成
// 主要用于在 Run() 返回后确保所有后台任务已完全停止
func (r *Runner) Wait() {
	r.wg.Wait()
}

func (r *Runner) Context() context.Context {
	return r.ctx
}
