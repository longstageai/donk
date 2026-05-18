package builtin

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/longstageai/donk/donk/internal/scheduler"
	"github.com/longstageai/donk/donk/internal/tool"
)

// TaskManager 调度任务管理工具
// 用于创建、删除、取消、查询定时任务
type TaskManager struct {
	scheduler *scheduler.Scheduler
}

// NewTaskManager 创建任务管理工具
func NewTaskManager(sched *scheduler.Scheduler) *TaskManager {
	return &TaskManager{
		scheduler: sched,
	}
}

// Name 返回工具名称
func (t *TaskManager) Name() string {
	return "task_manager"
}

// Description 返回工具描述
func (t *TaskManager) Description() string {
	return "调度任务管理工具，用于创建、删除、取消和查询定时任务"
}

// Version 返回版本
func (t *TaskManager) Version() string {
	return "1.0.0"
}

// Category 返回分类
func (t *TaskManager) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回参数定义
func (t *TaskManager) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"action": {
			Type:        "string",
			Description: "操作类型：create(创建任务)、delete(删除任务)、cancel(取消任务)、list(查询任务列表)、get(获取任务详情)、run_list(查看执行记录列表)、run_get(查看单条执行记录)、run_delete(删除执行记录)",
			Enum:        []interface{}{"create", "delete", "cancel", "list", "get", "run_list", "run_get", "run_delete"},
		},
		"task_id": {
			Type:        "string",
			Description: "任务ID，用于 delete/cancel/get/run_list 操作",
		},
		"run_id": {
			Type:        "string",
			Description: "执行记录ID，用于 run_get、run_delete 操作",
		},
		"run_limit": {
			Type:        "integer",
			Description: "返回记录数量限制，默认 20，用于 run_list 操作",
			Default:     20,
		},
		"run_offset": {
			Type:        "integer",
			Description: "记录列表偏移量，默认 0，用于 run_list 操作",
			Default:     0,
		},
		"run_status": {
			Type:        "string",
			Description: "执行记录状态过滤，用于 run_list 操作：running/done/failed",
		},
		"name": {
			Type:        "string",
			Description: "任务名称，用于 create 操作",
		},
		"task_type": {
			Type:        "string",
			Description: "任务类型：cron(定时循环)、delay(延迟执行)、once(单次执行)，用于 create 操作",
			Enum:        []interface{}{"cron", "delay", "once"},
		},
		"schedule": {
			Type:        "string",
			Description: "调度表达式：cron表达式(如 */5 * * * *)、延迟时间(如 10s、5m)、时间戳或RFC3339格式，用于 create 操作",
		},
		"executor": {
			Type:        "string",
			Description: "执行器类型：agent(Agent调用)，用于 create 操作",
			Enum:        []interface{}{"agent"},
		},
		"prompt": {
			Type:        "string",
			Description: "Agent 任务提示词，仅用于 agent 执行器，指定 Agent 需要执行的任务内容",
		},
		"timeout": {
			Type:        "integer",
			Description: "Agent 执行超时时间（秒），仅用于 agent 执行器，默认 300 秒",
			Default:     300,
		},
		"max_retries": {
			Type:        "integer",
			Description: "最大重试次数，用于 create 操作，默认 3",
			Default:     3,
		},
		"status": {
			Type:        "string",
			Description: "任务状态过滤，用于 list 操作：pending/running/done/failed/cancelled",
		},
		"limit": {
			Type:        "integer",
			Description: "返回任务数量限制，用于 list 操作，默认 20",
			Default:     20,
		},
		"offset": {
			Type:        "integer",
			Description: "任务列表偏移量，用于 list 操作，默认 0",
			Default:     0,
		},
	}
	schema.Required = []string{"action"}
	return schema
}

// Execute 执行任务管理操作
func (t *TaskManager) Execute(ctx *tool.Context) (*tool.Result, error) {
	action, ok := ctx.Params["action"].(string)
	if !ok || action == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "action 参数不能为空"), nil
	}

	switch action {
	case "create":
		return t.createTask(ctx)
	case "delete":
		return t.deleteTask(ctx)
	case "cancel":
		return t.cancelTask(ctx)
	case "list":
		return t.listTasks(ctx)
	case "get":
		return t.getTask(ctx)
	case "run_list":
		return t.listRuns(ctx)
	case "run_get":
		return t.getRun(ctx)
	case "run_delete":
		return t.deleteRun(ctx)
	default:
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("不支持的操作类型: %s", action)), nil
	}
}

// createTask 创建新任务
func (t *TaskManager) createTask(ctx *tool.Context) (*tool.Result, error) {
	name, _ := ctx.Params["name"].(string)
	if name == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "name 参数不能为空"), nil
	}

	taskTypeStr, _ := ctx.Params["task_type"].(string)
	if taskTypeStr == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "task_type 参数不能为空"), nil
	}

	schedule, _ := ctx.Params["schedule"].(string)
	if schedule == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "schedule 参数不能为空"), nil
	}

	executorStr, _ := ctx.Params["executor"].(string)
	if executorStr == "" {
		executorStr = string(scheduler.ExecutorAgent)
	}
	if executorStr != string(scheduler.ExecutorAgent) {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "executor 仅支持 agent"), nil
	}

	taskType := scheduler.TaskType(taskTypeStr)
	executor := scheduler.ExecutorAgent

	config := scheduler.TaskConfig{}

	prompt, _ := ctx.Params["prompt"].(string)
	if prompt == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "agent 执行器需要 prompt 参数"), nil
	}
	config["prompt"] = prompt
	if timeout, ok := ctx.Params["timeout"].(float64); ok {
		config["timeout"] = int(timeout)
	}

	// 兼容旧版本的 config 参数
	if cfg, ok := ctx.Params["config"].(map[string]any); ok {
		for k, v := range cfg {
			if _, exists := config[k]; !exists {
				config[k] = v
			}
		}
	}

	maxRetries := 3
	if mr, ok := ctx.Params["max_retries"].(float64); ok {
		maxRetries = int(mr)
	}

	task := &scheduler.Task{
		ID:         generateTaskID(),
		Name:       name,
		TaskType:   taskType,
		Schedule:   schedule,
		Executor:   executor,
		Config:     config,
		Status:     scheduler.TaskStatusPending,
		MaxRetries: maxRetries,
	}

	if err := t.scheduler.CreateTask(task); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("创建任务失败: %v", err)), nil
	}

	return tool.NewResult(map[string]any{
		"success": true,
		"task_id": task.ID,
		"message": fmt.Sprintf("任务 %s 创建成功", name),
		"task":    task,
	}), nil
}

// deleteTask 删除任务
func (t *TaskManager) deleteTask(ctx *tool.Context) (*tool.Result, error) {
	taskID, ok := ctx.Params["task_id"].(string)
	if !ok || taskID == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "task_id 参数不能为空"), nil
	}

	if err := t.scheduler.DeleteTask(taskID); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("删除任务失败: %v", err)), nil
	}

	return tool.NewResult(map[string]any{
		"success": true,
		"task_id": taskID,
		"message": fmt.Sprintf("任务 %s 删除成功", taskID),
	}), nil
}

// cancelTask 取消任务
func (t *TaskManager) cancelTask(ctx *tool.Context) (*tool.Result, error) {
	taskID, ok := ctx.Params["task_id"].(string)
	if !ok || taskID == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "task_id 参数不能为空"), nil
	}

	if err := t.scheduler.CancelTask(taskID); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("取消任务失败: %v", err)), nil
	}

	return tool.NewResult(map[string]any{
		"success": true,
		"task_id": taskID,
		"message": fmt.Sprintf("任务 %s 取消成功", taskID),
	}), nil
}

// listTasks 查询任务列表
func (t *TaskManager) listTasks(ctx *tool.Context) (*tool.Result, error) {
	filter := scheduler.TaskFilter{}

	if status, ok := ctx.Params["status"].(string); ok && status != "" {
		filter.Status = scheduler.TaskStatus(status)
	}

	limit := 20
	if l, ok := ctx.Params["limit"].(float64); ok {
		limit = int(l)
	}

	offset := 0
	if o, ok := ctx.Params["offset"].(float64); ok {
		offset = int(o)
	}
	filter.Size = limit
	filter.Page = offset

	tasks, total, err := t.scheduler.ListTasks(filter)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("查询任务列表失败: %v", err)), nil
	}

	taskList := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		taskList = append(taskList, map[string]any{
			"id":          task.ID,
			"name":        task.Name,
			"task_type":   task.TaskType,
			"schedule":    task.Schedule,
			"executor":    task.Executor,
			"status":      task.Status,
			"next_run_at": task.NextRunAt,
			"last_run_at": task.LastRunAt,
			"max_retries": task.MaxRetries,
			"retries":     task.Retries,
		})
	}

	return tool.NewResult(map[string]any{
		"success": true,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"tasks":   taskList,
	}), nil
}

// listRuns 查询执行记录列表
func (t *TaskManager) listRuns(ctx *tool.Context) (*tool.Result, error) {
	filter := scheduler.TaskRunFilter{}

	if taskID, ok := ctx.Params["task_id"].(string); ok && taskID != "" {
		filter.TaskID = taskID
	}

	if status, ok := ctx.Params["run_status"].(string); ok && status != "" {
		filter.Status = scheduler.TaskRunStatus(status)
	}

	limit := 20
	if l, ok := ctx.Params["run_limit"].(float64); ok {
		limit = int(l)
	}

	offset := 0
	if o, ok := ctx.Params["run_offset"].(float64); ok {
		offset = int(o)
	}
	filter.Size = limit
	filter.Page = offset

	runs, total, err := t.scheduler.ListRuns(filter)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("查询执行记录失败: %v", err)), nil
	}

	runList := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		runList = append(runList, map[string]any{
			//"id":          run.ID,
			//"task_id":     run.TaskID,
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

	return tool.NewResult(map[string]any{
		"success": true,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"runs":    runList,
	}), nil
}

// getRun 获取单条执行记录详情
func (t *TaskManager) getRun(ctx *tool.Context) (*tool.Result, error) {
	runID, ok := ctx.Params["run_id"].(string)
	if !ok || runID == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "run_id 参数不能为空"), nil
	}

	run, err := t.scheduler.GetRun(runID)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("获取执行记录失败: %v", err)), nil
	}

	return tool.NewResult(map[string]any{
		"success": true,
		"run": map[string]any{
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
		},
	}), nil
}

// deleteRun 删除执行记录
func (t *TaskManager) deleteRun(ctx *tool.Context) (*tool.Result, error) {
	runID, ok := ctx.Params["run_id"].(string)
	if !ok || runID == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "run_id 参数不能为空"), nil
	}

	if err := t.scheduler.DeleteRun(runID); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("删除执行记录失败: %v", err)), nil
	}

	return tool.NewResult(map[string]any{
		"success": true,
		"run_id":  runID,
		"message": fmt.Sprintf("执行记录 %s 删除成功", runID),
	}), nil
}

// getTask 获取任务详情
func (t *TaskManager) getTask(ctx *tool.Context) (*tool.Result, error) {
	taskID, ok := ctx.Params["task_id"].(string)
	if !ok || taskID == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "task_id 参数不能为空"), nil
	}

	task, err := t.scheduler.GetTask(taskID)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("获取任务详情失败: %v", err)), nil
	}

	return tool.NewResult(map[string]any{
		"success": true,
		"task": map[string]any{
			"id":          task.ID,
			"name":        task.Name,
			"task_type":   task.TaskType,
			"schedule":    task.Schedule,
			"executor":    task.Executor,
			"config":      task.Config,
			"status":      task.Status,
			"result":      task.Result,
			"next_run_at": task.NextRunAt,
			"last_run_at": task.LastRunAt,
			"max_retries": task.MaxRetries,
			"retries":     task.Retries,
			"created_at":  task.CreatedAt,
			"updated_at":  task.UpdatedAt,
		},
	}), nil
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return "task_" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + randomString(8)
}

// randomString 生成长度为 n 的随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}
