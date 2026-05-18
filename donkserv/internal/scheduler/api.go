package scheduler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIHandler API 处理器
// 处理任务管理的 REST API 请求
type APIHandler struct {
	scheduler *Scheduler
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(scheduler *Scheduler) *APIHandler {
	return &APIHandler{scheduler: scheduler}
}

// RegisterRoutes 注册路由
// 将任务管理的 API 路由注册到 gin 引擎
func (h *APIHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		tasks := api.Group("/tasks")
		{
			tasks.POST("", h.CreateTask)
			tasks.GET("", h.ListTasks)
			tasks.GET("/:id", h.GetTask)
			tasks.DELETE("/:id", h.DeleteTask)
			tasks.POST("/:id/cancel", h.CancelTask)
			tasks.POST("/:id/run", h.TriggerTask)
			tasks.GET("/:id/result", h.GetTaskResult)
			tasks.GET("/:id/runs", h.ListTaskRuns)
		}

		runs := api.Group("/runs")
		{
			runs.GET("", h.ListRuns)
			runs.GET("/:id", h.GetRun)
			runs.DELETE("/:id", h.DeleteRun)
		}
	}
}

// CreateTaskRequest 创建任务请求结构
type CreateTaskRequest struct {
	Name       string                 `json:"name" binding:"required"`      // 任务名称
	TaskType   string                 `json:"task_type" binding:"required"` // 任务类型 (cron/delay/once)
	Schedule   string                 `json:"schedule" binding:"required"`  // 调度表达式
	Executor   string                 `json:"executor" binding:"required"`  // 执行器类型 (script/api/agent)
	Config     map[string]interface{} `json:"config"`                       // 执行配置
	MaxRetries int                    `json:"max_retries"`                  // 最大重试次数
	CreatedBy  string                 `json:"created_by"`                   // 创建者
}

// CreateTask 创建任务
// POST /api/v1/tasks
func (h *APIHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "参数错误",
			"details": err.Error(),
		})
		return
	}

	// 验证任务类型
	taskType := TaskType(req.TaskType)
	if taskType != TaskTypeCron && taskType != TaskTypeDelay && taskType != TaskTypeOnce {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的任务类型",
		})
		return
	}

	// 验证执行器类型
	executor := ExecutorType(req.Executor)
	if executor != ExecutorScript && executor != ExecutorAPI && executor != ExecutorAgent {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的执行器类型",
		})
		return
	}

	// 创建任务
	task := &Task{
		ID:         uuid.New().String(),
		Name:       req.Name,
		TaskType:   taskType,
		Schedule:   req.Schedule,
		Executor:   executor,
		Config:     req.Config,
		MaxRetries: req.MaxRetries,
		Status:     TaskStatusPending,
		CreatedBy:  req.CreatedBy,
	}

	// 设置默认重试次数
	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}

	// 保存并调度任务
	if err := h.scheduler.CreateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "任务创建成功",
		"id":          task.ID,
		"name":        task.Name,
		"task_type":   task.TaskType,
		"schedule":    task.Schedule,
		"status":      task.Status,
		"next_run_at": task.NextRunAt,
		"created_at":  task.CreatedAt,
	})
}

// ListTasks 列出任务
// GET /api/v1/tasks
func (h *APIHandler) ListTasks(c *gin.Context) {
	// 解析查询参数
	filter := TaskFilter{
		Page: 1,
		Size: 20,
	}

	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		filter.Page = page
	}

	if size, err := strconv.Atoi(c.DefaultQuery("size", "20")); err == nil && size > 0 && size <= 100 {
		filter.Size = size
	}

	if status := c.Query("status"); status != "" {
		filter.Status = TaskStatus(status)
	}

	if executor := c.Query("executor"); executor != "" {
		filter.Executor = ExecutorType(executor)
	}

	// 查询任务列表
	tasks, total, err := h.scheduler.ListTasks(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 转换为响应格式
	items := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, gin.H{
			"id":          task.ID,
			"name":        task.Name,
			"task_type":   task.TaskType,
			"schedule":    task.Schedule,
			"executor":    task.Executor,
			"status":      task.Status,
			"next_run_at": task.NextRunAt,
			"last_run_at": task.LastRunAt,
			"created_at":  task.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  filter.Page,
		"size":  filter.Size,
	})
}

// GetTask 获取任务详情
// GET /api/v1/tasks/:id
func (h *APIHandler) GetTask(c *gin.Context) {
	id := c.Param("id")

	task, err := h.scheduler.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          task.ID,
		"name":        task.Name,
		"task_type":   task.TaskType,
		"schedule":    task.Schedule,
		"executor":    task.Executor,
		"config":      task.Config,
		"status":      task.Status,
		"result":      task.Result,
		"retries":     task.Retries,
		"max_retries": task.MaxRetries,
		"next_run_at": task.NextRunAt,
		"last_run_at": task.LastRunAt,
		"created_at":  task.CreatedAt,
		"updated_at":  task.UpdatedAt,
		"created_by":  task.CreatedBy,
	})
}

// DeleteTask 删除任务
// DELETE /api/v1/tasks/:id
func (h *APIHandler) DeleteTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduler.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "任务删除成功",
	})
}

// CancelTask 取消任务
// POST /api/v1/tasks/:id/cancel
func (h *APIHandler) CancelTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduler.CancelTask(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "任务已取消",
	})
}

// TriggerTask 手动触发任务
// POST /api/v1/tasks/:id/run
func (h *APIHandler) TriggerTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduler.TriggerTask(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "任务已触发执行",
	})
}

// GetTaskResult 获取任务执行结果
// GET /api/v1/tasks/:id/result
func (h *APIHandler) GetTaskResult(c *gin.Context) {
	id := c.Param("id")

	task, err := h.scheduler.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "任务不存在",
		})
		return
	}

	if task.Result == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "任务尚未执行",
		})
		return
	}

	c.JSON(http.StatusOK, task.Result)
}

// ListRuns 获取任务执行记录列表
// GET /api/v1/runs
func (h *APIHandler) ListRuns(c *gin.Context) {
	filter := TaskRunFilter{
		Page: 0,
		Size: 20,
	}

	if page, err := strconv.Atoi(c.DefaultQuery("page", "0")); err == nil && page >= 0 {
		filter.Page = page
	}

	if size, err := strconv.Atoi(c.DefaultQuery("size", "20")); err == nil && size > 0 && size <= 100 {
		filter.Size = size
	}

	if taskID := c.Query("task_id"); taskID != "" {
		filter.TaskID = taskID
	}

	if status := c.Query("status"); status != "" {
		filter.Status = TaskRunStatus(status)
	}

	runs, total, err := h.scheduler.ListRuns(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	items := make([]gin.H, 0, len(runs))
	for _, run := range runs {
		items = append(items, gin.H{
			"id":          run.ID,
			"task_id":     run.TaskID,
			"task_name":   run.TaskName,
			"executor":    run.Executor,
			"status":      run.Status,
			"start_time":  run.StartTime,
			"end_time":    run.EndTime,
			"duration":    run.Duration,
			"exit_code":   run.ExitCode,
			"retry_count": run.RetryCount,
			"created_at":  run.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  filter.Page,
		"size":  filter.Size,
	})
}

// GetRun 获取单条执行记录
// GET /api/v1/runs/:id
func (h *APIHandler) GetRun(c *gin.Context) {
	id := c.Param("id")

	run, err := h.scheduler.GetRun(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "执行记录不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          run.ID,
		"task_id":     run.TaskID,
		"task_name":   run.TaskName,
		"executor":    run.Executor,
		"input":       run.Input,
		"status":      run.Status,
		"start_time":  run.StartTime,
		"end_time":    run.EndTime,
		"duration":    run.Duration,
		"output":      run.Output,
		"error":       run.Error,
		"exit_code":   run.ExitCode,
		"retry_count": run.RetryCount,
		"created_at":  run.CreatedAt,
		"updated_at":  run.UpdatedAt,
	})
}

// DeleteRun 删除执行记录
// DELETE /api/v1/runs/:id
func (h *APIHandler) DeleteRun(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduler.DeleteRun(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "执行记录删除成功",
	})
}

// ListTaskRuns 获取指定任务的执行记录列表
// GET /api/v1/tasks/:id/runs
func (h *APIHandler) ListTaskRuns(c *gin.Context) {
	taskID := c.Param("id")

	// 解析分页参数
	page := 1
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && p > 0 {
		page = p
	}

	size := 20
	if s, err := strconv.Atoi(c.DefaultQuery("size", "20")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	// 构建过滤条件
	filter := TaskRunFilter{
		TaskID: taskID,
		Page:   page - 1, // 转换为从0开始的页码
		Size:   size,
	}

	// 可选：按状态过滤
	if status := c.Query("status"); status != "" {
		filter.Status = TaskRunStatus(status)
	}

	// 查询执行记录列表
	runs, total, err := h.scheduler.ListRuns(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 返回所有字段
	items := make([]gin.H, 0, len(runs))
	for _, run := range runs {
		items = append(items, gin.H{
			"id":          run.ID,
			"task_id":     run.TaskID,
			"task_name":   run.TaskName,
			"executor":    run.Executor,
			"input":       run.Input,
			"status":      run.Status,
			"start_time":  run.StartTime,
			"end_time":    run.EndTime,
			"duration":    run.Duration,
			"output":      run.Output,
			"error":       run.Error,
			"exit_code":   run.ExitCode,
			"retry_count": run.RetryCount,
			"created_at":  run.CreatedAt,
			"updated_at":  run.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
