package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/longstageai/donk/donk/internal/websocket"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/robfig/cron/v3"
)

// Scheduler 定时任务调度器
// 负责管理任务的创建、调度、执行和状态跟踪
// 支持 cron 表达式、延迟任务和单次任务三种调度方式
type Scheduler struct {
	repo         TaskRepository    // 任务仓储层
	runRepo      TaskRunRepository // 任务执行记录仓储层
	factory      ExecutorFactory   // 执行器工厂
	eventBus     *EventBus         // 事件总线
	hub          *websocket.Hub
	cron         *cron.Cron              // Cron 调度器
	timers       map[string]*time.Timer  // 延迟任务定时器
	cronEntries  map[string]cron.EntryID // Cron 任务 entry ID 映射
	deletedTasks map[string]bool         // 已删除任务集合，防止竞态条件
	workers      int                     // 并发执行 worker 数量
	taskCh       chan *Task              // 任务队列
	wg           sync.WaitGroup          // 用于等待 worker 退出
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	running      bool
}

// SchedulerOption 调度器配置选项
type SchedulerOption func(*Scheduler)

// WithWorkers 设置并发 worker 数量
func WithWorkers(n int) SchedulerOption {
	return func(s *Scheduler) {
		s.workers = n
	}
}

// WithEventBus 设置事件总线
func WithEventBus(eb *EventBus) SchedulerOption {
	return func(s *Scheduler) {
		//s.eventBus = eb
	}
}

// NewScheduler 创建调度器实例
// repo: 任务仓储层
// factory: 执行器工厂
// runRepo: 任务执行记录仓储层(可选，传nil时不会记录执行历史)
func NewScheduler(hub *websocket.Hub, repo TaskRepository, factory ExecutorFactory, runRepo TaskRunRepository, opts ...SchedulerOption) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Scheduler{
		hub:          hub,
		repo:         repo,
		runRepo:      runRepo,
		factory:      factory,
		cron:         cron.New(),
		timers:       make(map[string]*time.Timer),
		cronEntries:  make(map[string]cron.EntryID),
		deletedTasks: make(map[string]bool),
		workers:      5,
		taskCh:       make(chan *Task, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SetAgentFactory 设置 Agent 工厂函数
// 用于在调度器启动时注入 Agent 工厂，使 Agent 执行器可以创建 Agent 实例
func (s *Scheduler) SetAgentFactory(agentFactory func() interface{}) {
	if f, ok := s.factory.(*DefaultExecutorFactory); ok {
		f.SetAgentFactory(agentFactory)
	}
}

// Start 启动调度器
// 加载待执行任务、启动 cron 调度、启动 worker 池
func (s *Scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("调度器已启动")
	}
	s.running = true
	s.mu.Unlock()

	logger.Info("启动调度器", map[string]interface{}{"workers": s.workers})

	// 启动 cron 调度器
	s.cron.Start()
	logger.Info("Cron 调度器已启动", nil)

	// 启动 worker 池
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	logger.Info("已启动 Worker", map[string]interface{}{"count": s.workers})

	// 启动定时检查（作为 cron 的补充）
	go s.scheduleLoop()

	// 从数据库恢复待执行任务
	if err := s.Recover(); err != nil {
		logger.Error("任务恢复失败", map[string]interface{}{"error": err.Error()})
	}

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()

	// 停止 cron
	s.cron.Stop()

	// 停止所有延迟任务定时器
	for _, timer := range s.timers {
		timer.Stop()
	}
	s.timers = make(map[string]*time.Timer)

	// 等待 worker 结束
	close(s.taskCh)
	s.wg.Wait()

	s.running = false
	logger.Info("调度器已停止", nil)
}

// Recover 从数据库恢复未完成的任务
// 启动时调用，将 pending/running 状态的任务重新加入调度
func (s *Scheduler) Recover() error {
	logger.Info("开始恢复任务", nil)

	// 恢复 pending 任务
	pendingTasks, err := s.repo.ListByStatus(s.ctx, TaskStatusPending)
	if err != nil {
		return fmt.Errorf("查询 pending 任务失败: %w", err)
	}

	for _, task := range pendingTasks {
		if err := s.scheduleTask(task); err != nil {
			logger.Warn("恢复任务失败", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
		}
	}
	logger.Info("已恢复 pending 任务", map[string]interface{}{"count": len(pendingTasks)})

	// 恢复 running 任务（由于某种原因中断的任务）
	runningTasks, err := s.repo.ListByStatus(s.ctx, TaskStatusRunning)
	if err != nil {
		return fmt.Errorf("查询 running 任务失败: %w", err)
	}

	for _, task := range runningTasks {
		// 判断是否超时（超过 1 小时）
		if time.Now().Unix()-task.UpdatedAt > 3600 {
			// 超时任务，标记为失败或重试
			if task.Retries < task.MaxRetries {
				task.Status = TaskStatusPending
				task.Retries++
				if err := s.repo.Update(s.ctx, task); err != nil {
					logger.Warn("更新任务状态失败", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
				}
				if err := s.scheduleTask(task); err != nil {
					logger.Warn("恢复超时任务失败", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
				}
			} else {
				task.Status = TaskStatusFailed
				if err := s.repo.Update(s.ctx, task); err != nil {
					logger.Warn("更新任务状态失败", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
				}
			}
		} else {
			// 未超时，重新执行
			task.Status = TaskStatusPending
			if err := s.repo.Update(s.ctx, task); err != nil {
				logger.Warn("更新任务状态失败", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
			}
			if err := s.scheduleTask(task); err != nil {
				logger.Warn("恢复任务失败", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
			}
		}
	}
	logger.Info("已恢复 running 任务", map[string]interface{}{"count": len(runningTasks)})

	return nil
}

// scheduleLoop 定时检查任务（补充 cron）
// 每秒检查一次是否有需要执行的任务
func (s *Scheduler) scheduleLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkPendingTasks()
		}
	}
}

// checkPendingTasks 检查待执行任务
// 从数据库获取即将到期的任务并加入执行队列
func (s *Scheduler) checkPendingTasks() {
	now := time.Now().Unix()
	tasks, err := s.repo.ListPending(s.ctx, now)
	if err != nil {
		logger.Error("查询待执行任务失败", map[string]interface{}{"error": err.Error()})
		return
	}

	for _, task := range tasks {
		// 再次检查任务状态，防止重复调度
		currentTask, err := s.repo.GetByID(s.ctx, task.ID)
		if err != nil || currentTask.Status != TaskStatusPending {
			continue
		}

		// 直接发送到执行队列，不预更新状态
		// 由 worker 执行任务时更新状态
		s.enqueueTask(currentTask)
	}
}

// scheduleTask 将任务加入调度器
// 根据任务类型选择合适的调度方式
func (s *Scheduler) scheduleTask(task *Task) error {
	switch task.TaskType {
	case TaskTypeCron:
		return s.scheduleCronTask(task)
	case TaskTypeDelay, TaskTypeOnce:
		return s.scheduleTimerTask(task)
	default:
		return fmt.Errorf("未知的任务类型: %s", task.TaskType)
	}
}

// scheduleCronTask 添加 cron 任务到调度器
func (s *Scheduler) scheduleCronTask(task *Task) error {
	// 先验证 cron 表达式
	if err := ValidateCronExpression(task.Schedule); err != nil {
		logger.Warn("cron 表达式无效", map[string]interface{}{"schedule": task.Schedule, "error": err.Error()})
		return fmt.Errorf("无效的 cron 表达式: %v", err)
	}

	s.mu.Lock()
	if oldEntryID, ok := s.cronEntries[task.ID]; ok {
		s.cron.Remove(oldEntryID)
		delete(s.cronEntries, task.ID)
	}
	s.mu.Unlock()

	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.enqueueTask(task.Clone())
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.cronEntries[task.ID] = entryID
	s.mu.Unlock()

	return nil
}

// scheduleTimerTask 添加延迟/单次任务到调度器
func (s *Scheduler) scheduleTimerTask(task *Task) error {
	delay := time.Until(time.Unix(task.NextRunAt, 0))
	if delay <= 0 {
		delay = time.Second
	}

	timer := time.AfterFunc(delay, func() {
		s.enqueueTask(task.Clone())
	})

	s.mu.Lock()
	s.timers[task.ID] = timer
	s.mu.Unlock()

	return nil
}

// enqueueTask 将任务加入执行队列
func (s *Scheduler) enqueueTask(task *Task) {
	select {
	case s.taskCh <- task:
	default:
		logger.Warn("任务队列已满", map[string]interface{}{"task_id": task.ID})
	}
}

// worker 执行任务的工作协程
// 从任务队列中获取任务并执行
func (s *Scheduler) worker(id int) {
	defer s.wg.Done()
	logger.Info("Worker 已启动", map[string]interface{}{"worker_id": id})

	for {
		select {
		case <-s.ctx.Done():
			logger.Info("Worker 已退出", map[string]interface{}{"worker_id": id})
			return
		case task, ok := <-s.taskCh:
			if !ok {
				return
			}
			s.executeTask(task)
		}
	}
}

// executeTask 执行单个任务
func (s *Scheduler) executeTask(task *Task) {
	logger.Info("开始执行任务", map[string]interface{}{"task_id": task.ID, "task_name": task.Name})

	s.mu.Lock()
	if s.deletedTasks[task.ID] {
		s.mu.Unlock()
		logger.Info("任务已删除，跳过执行", map[string]interface{}{"task_id": task.ID})
		return
	}
	s.mu.Unlock()

	// 使用原子操作认领任务，防止多个 worker 同时执行同一任务
	workerID := fmt.Sprintf("worker-%d", time.Now().UnixNano())
	claimedTask, err := s.repo.ClaimTask(s.ctx, task.ID, workerID)
	if err != nil {
		if err == ErrTaskNotPending {
			logger.Info("任务已被其他worker认领或状态不是pending，跳过执行", map[string]interface{}{"task_id": task.ID})
		} else {
			logger.Error("认领任务失败，跳过执行", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
		}
		return
	}

	latestTask := claimedTask

	// 发布任务开始事件
	s.publishEvent(EventTaskStarted, latestTask)

	// 创建执行记录
	run := s.createTaskRun(latestTask)
	if run == nil {
		logger.Warn("创建执行记录失败，将继续执行任务但不记录", map[string]interface{}{"task_id": task.ID})
	}

	// 1. 创建执行器
	executor, err := s.factory.Create(latestTask.Executor)
	if err != nil {
		s.handleTaskFailure(latestTask, fmt.Sprintf("创建执行器失败: %v", err), run)
		return
	}

	// 2. 执行任务
	result, err := executor.Execute(s.ctx, latestTask)

	// 3. 处理执行结果
	s.handleTaskResult(latestTask, result, err, run)
}

// createTaskRun 创建任务执行记录
func (s *Scheduler) createTaskRun(task *Task) *TaskRun {
	if s.runRepo == nil {
		return nil
	}

	inputJSON, _ := encodeJSON(task.Config)
	run := &TaskRun{
		ID:         generateRunID(),
		TaskID:     task.ID,
		TaskName:   task.Name,
		Executor:   task.Executor,
		Input:      inputJSON,
		Status:     TaskRunStatusRunning,
		StartTime:  time.Now().Unix(),
		RetryCount: task.Retries,
	}

	if err := s.runRepo.Create(s.ctx, run); err != nil {
		logger.Error("创建执行记录失败", map[string]interface{}{"task_id": task.ID, "error": err.Error()})
		return nil
	}

	return run
}

// updateTaskRun 更新任务执行记录
func (s *Scheduler) updateTaskRun(run *TaskRun, result *TaskResult, execErr error) {
	if run == nil || s.runRepo == nil {
		return
	}

	run.EndTime = time.Now().Unix()
	run.Duration = run.CalculateDuration()

	if execErr != nil {
		run.Status = TaskRunStatusFailed
		run.Error = execErr.Error()
	} else if result != nil {
		run.Output = result.Output
		run.ExitCode = result.ExitCode
		if result.Error != "" {
			run.Error = result.Error
			run.Status = TaskRunStatusFailed
		} else {
			run.Status = TaskRunStatusDone
		}
	}

	if err := s.runRepo.Update(s.ctx, run); err != nil {
		logger.Error("更新执行记录失败", map[string]interface{}{"run_id": run.ID, "error": err.Error()})
	}
}

// handleTaskResult 处理任务执行结果
// 注意：此方法已获取锁后调用
func (s *Scheduler) handleTaskResult(task *Task, result *TaskResult, execErr error, run *TaskRun) {
	// 更新执行记录
	s.updateTaskRun(run, result, execErr)

	// 判断是否成功
	if execErr != nil {
		s.handleExecutionFailure(task, run)
		return
	}

	task.Result = result
	if result.Error != "" || result.ExitCode != 200 {
		s.handleExecutionFailure(task, run)
		return
	}

	if s.hub != nil {
		msg := websocket.NewNotification("cron_task", task.Name, task.Result.Output)
		data, _ := json.Marshal(msg)
		// 广播消息
		s.hub.BroadcastJSON(data)
	}
	// 执行成功
	s.handleExecutionSuccess(task, run)
}

// handleExecutionFailure 处理执行失败的情况
// 注意：此方法已获取锁后调用
func (s *Scheduler) handleExecutionFailure(task *Task, run *TaskRun) {
	// 检查是否可重试
	if task.Retries < task.MaxRetries {
		s.scheduleRetry(task, run)
		return
	}

	// 不可重试，标记为失败
	s.handleTaskFailure(task, fmt.Sprintf("执行失败: %s", task.Result.Error), run)
}

// handleExecutionSuccess 处理执行成功的情况
// 注意：此方法已获取锁后调用
func (s *Scheduler) handleExecutionSuccess(task *Task, run *TaskRun) {
	task.Status = TaskStatusDone

	// cron 任务需要重新调度
	if task.TaskType == TaskTypeCron {
		s.rescheduleCronTask(task, run)
		return
	}

	// 非 cron 任务，直接完成
	s.finishTask(task)
}

// scheduleRetry 安排重试
func (s *Scheduler) scheduleRetry(task *Task, run *TaskRun) {
	task.Status = TaskStatusPending
	task.Retries++

	// 计算退避时间
	delay := s.calculateBackoff(task.Retries)
	task.NextRunAt = time.Now().Add(delay).Unix()

	if err := s.repo.Update(s.ctx, task); err != nil {
		logger.Warn("更新任务状态失败", map[string]interface{}{"error": err.Error()})
	}

	if err := s.scheduleTask(task); err != nil {
		logger.Warn("重新调度任务失败", map[string]interface{}{"error": err.Error()})
		// 调度失败，标记为失败
		s.handleTaskFailure(task, fmt.Sprintf("调度失败: %v", err), run)
		return
	}

	logger.Info("任务已安排重试", map[string]interface{}{"task_id": task.ID, "retry": task.Retries, "max_retries": task.MaxRetries})
}

// rescheduleCronTask 重新调度 cron 任务
// 注意：cron 任务一旦添加就会永久运行，不需要重新调用 scheduleTask
// 此处只需要更新任务状态和下次执行时间到数据库
func (s *Scheduler) rescheduleCronTask(task *Task, run *TaskRun) {
	s.mu.Lock()
	if s.deletedTasks[task.ID] {
		s.mu.Unlock()
		logger.Info("任务已删除，跳过重新调度", map[string]interface{}{"task_id": task.ID})
		return
	}
	s.mu.Unlock()

	s.publishEvent(EventTaskCompleted, task)

	task.NextRunAt = task.CalculateNextRunAt()
	if task.NextRunAt == 0 {
		s.handleTaskFailure(task, "cron 表达式无效，无法计算下次执行时间", run)
		return
	}

	task.Status = TaskStatusPending
	task.LastRunAt = time.Now().Unix()

	if err := s.repo.Update(s.ctx, task); err != nil {
		logger.Warn("更新任务状态失败", map[string]interface{}{"error": err.Error()})
	}

	logger.Info("cron 任务已更新下次执行时间", map[string]interface{}{
		"task_id":       task.ID,
		"next_run_time": time.Unix(task.NextRunAt, 0).Format("2006-01-02 15:04:05"),
	})
}

// handleTaskFailure 处理任务失败
func (s *Scheduler) handleTaskFailure(task *Task, errMsg string, run *TaskRun) {
	logger.Error("任务执行失败", map[string]interface{}{"task_id": task.ID, "error": errMsg})
	task.Status = TaskStatusFailed
	task.Result = &TaskResult{
		Error:  errMsg,
		DoneAt: time.Now().Unix(),
	}

	// 更新执行记录为失败状态
	if run != nil {
		run.Status = TaskRunStatusFailed
		run.Error = errMsg
		run.EndTime = time.Now().Unix()
		if err := s.runRepo.Update(s.ctx, run); err != nil {
			logger.Error("更新执行记录失败", map[string]interface{}{"run_id": run.ID, "error": err.Error()})
		}
	}

	s.finishTask(task)
}

// finishTask 完成任务处理
func (s *Scheduler) finishTask(task *Task) {
	// 设置上次执行时间
	task.LastRunAt = time.Now().Unix()

	if err := s.repo.Update(s.ctx, task); err != nil {
		logger.Warn("更新任务状态失败", map[string]interface{}{"error": err.Error()})
	}

	// 发布完成事件
	if task.Status == TaskStatusDone {
		s.publishEvent(EventTaskCompleted, task)
	} else if task.Status == TaskStatusFailed {
		s.publishEvent(EventTaskFailed, task)
	}
}

// calculateBackoff 计算重试退避时间
// 使用指数退避策略
func (s *Scheduler) calculateBackoff(retries int) time.Duration {
	base := time.Minute
	max := 30 * time.Minute
	delay := base
	for i := 1; i < retries; i++ {
		delay *= 2
		if delay > max {
			delay = max
			break
		}
	}
	return delay
}

// publishEvent 发布任务事件
func (s *Scheduler) publishEvent(eventType EventType, task *Task) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.Publish(&TaskEvent{
		Type:      eventType,
		TaskID:    task.ID,
		Task:      task,
		Timestamp: time.Now().Unix(),
	})
}

// CreateTask 创建新任务
func (s *Scheduler) CreateTask(task *Task) error {
	// 计算下次执行时间
	task.NextRunAt = task.CalculateNextRunAt()
	if task.NextRunAt <= 0 {
		task.NextRunAt = time.Now().Unix()
	}

	// 保存到数据库
	if err := s.repo.Create(s.ctx, task); err != nil {
		return err
	}

	// 加入调度
	if err := s.scheduleTask(task); err != nil {
		return err
	}

	// 发布创建事件
	s.publishEvent(EventTaskCreated, task)

	return nil
}

// CancelTask 取消任务
func (s *Scheduler) CancelTask(taskID string) error {
	task, err := s.repo.GetByID(s.ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status != TaskStatusPending {
		return fmt.Errorf("任务状态不是 pending，无法取消")
	}

	s.mu.Lock()
	if timer, ok := s.timers[taskID]; ok {
		timer.Stop()
		delete(s.timers, taskID)
	}
	if entryID, ok := s.cronEntries[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.cronEntries, taskID)
	}
	s.mu.Unlock()

	task.Status = TaskStatusCancelled
	if err := s.repo.Update(s.ctx, task); err != nil {
		return err
	}

	s.publishEvent(EventTaskCancelled, task)

	return nil
}

// DeleteTask 删除任务
// 同时删除任务的所有运行记录
func (s *Scheduler) DeleteTask(taskID string) error {
	s.mu.Lock()
	s.deletedTasks[taskID] = true

	if timer, ok := s.timers[taskID]; ok {
		timer.Stop()
		delete(s.timers, taskID)
	}
	if entryID, ok := s.cronEntries[taskID]; ok {
		logger.Info("删除 cron 任务", map[string]interface{}{"task_id": taskID, "entry_id": entryID})
		s.cron.Remove(entryID)
		delete(s.cronEntries, taskID)
	} else {
		logger.Info("未找到 cron 任务", map[string]interface{}{"task_id": taskID})
	}
	s.mu.Unlock()

	s.CancelTask(taskID)

	// 先删除任务的运行记录
	if s.runRepo != nil {
		if err := s.runRepo.DeleteByTaskID(s.ctx, taskID); err != nil {
			logger.Warn("删除任务运行记录失败", map[string]interface{}{"task_id": taskID, "error": err.Error()})
			// 继续删除任务本身，不中断流程
		} else {
			logger.Info("删除任务运行记录成功", map[string]interface{}{"task_id": taskID})
		}
	}

	// 删除任务
	return s.repo.Delete(s.ctx, taskID)
}

// GetTask 获取任务
func (s *Scheduler) GetTask(taskID string) (*Task, error) {
	return s.repo.GetByID(s.ctx, taskID)
}

// ListTasks 列出任务
func (s *Scheduler) ListTasks(filter TaskFilter) ([]*Task, int64, error) {
	return s.repo.List(s.ctx, filter)
}

// TriggerTask 手动触发任务
func (s *Scheduler) TriggerTask(taskID string) error {
	task, err := s.repo.GetByID(s.ctx, taskID)
	if err != nil {
		return err
	}

	// 手动触发直接执行，设置为 running
	// 注意：不修改 NextRunAt，由调度器管理
	task.Status = TaskStatusRunning
	if err := s.repo.Update(s.ctx, task); err != nil {
		return err
	}

	// 直接执行任务（异步执行）
	go s.executeTaskDirect(task)
	return nil
}

// executeTaskDirect 直接执行任务（用于手动触发，跳过队列）
func (s *Scheduler) executeTaskDirect(task *Task) {
	logger.Info("手动触发执行任务", map[string]interface{}{"task_id": task.ID, "task_name": task.Name})

	// 发布任务开始事件
	s.publishEvent(EventTaskStarted, task)

	// 创建执行记录
	run := s.createTaskRun(task)
	if run == nil {
		logger.Warn("创建执行记录失败，将继续执行任务但不记录", map[string]interface{}{"task_id": task.ID})
	}

	// 1. 创建执行器
	executor, err := s.factory.Create(task.Executor)
	if err != nil {
		s.handleTaskFailure(task, fmt.Sprintf("创建执行器失败: %v", err), run)
		return
	}

	// 2. 执行任务
	result, err := executor.Execute(s.ctx, task)

	// 3. 处理执行结果
	s.handleTaskResult(task, result, err, run)
}

// generateRunID 生成执行记录ID
func generateRunID() string {
	return fmt.Sprintf("run_%s", uuid.New().String())
}

// ListRuns 获取任务执行记录列表
func (s *Scheduler) ListRuns(filter TaskRunFilter) ([]*TaskRun, int64, error) {
	if s.runRepo == nil {
		return nil, 0, fmt.Errorf("执行记录仓储未初始化")
	}
	return s.runRepo.List(s.ctx, filter)
}

// GetRun 获取单条执行记录
func (s *Scheduler) GetRun(runID string) (*TaskRun, error) {
	if s.runRepo == nil {
		return nil, fmt.Errorf("执行记录仓储未初始化")
	}
	return s.runRepo.GetByID(s.ctx, runID)
}

// DeleteRun 删除执行记录
func (s *Scheduler) DeleteRun(runID string) error {
	if s.runRepo == nil {
		return fmt.Errorf("执行记录仓储未初始化")
	}
	return s.runRepo.Delete(s.ctx, runID)
}
