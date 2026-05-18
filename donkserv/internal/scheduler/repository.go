package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// TaskRepository 任务仓储接口定义
// 提供任务的持久化操作，包括创建、更新、删除、查询等功能
type TaskRepository interface {
	// Create 创建新任务
	Create(ctx context.Context, task *Task) error

	// Update 更新任务
	Update(ctx context.Context, task *Task) error

	// Delete 删除任务
	Delete(ctx context.Context, id string) error

	// GetByID 根据ID获取任务
	GetByID(ctx context.Context, id string) (*Task, error)

	// List 根据过滤条件查询任务列表
	List(ctx context.Context, filter TaskFilter) ([]*Task, int64, error)

	// ListPending 获取待执行的任务列表
	// before: 返回在此时间戳之前需要执行的任务
	ListPending(ctx context.Context, before int64) ([]*Task, error)

	// ListByStatus 根据状态获取任务列表
	ListByStatus(ctx context.Context, status TaskStatus) ([]*Task, error)

	// ClaimTask 认领任务（用于分布式场景，单机场景可直接更新状态）
	// 返回成功认领的任务，workerID 用于标识认领者
	ClaimTask(ctx context.Context, taskID, workerID string) (*Task, error)

	// UpdateResult 更新任务执行结果
	UpdateResult(ctx context.Context, taskID string, result *TaskResult) error
}

// TaskFilter 任务过滤条件
type TaskFilter struct {
	Status   TaskStatus   // 按状态过滤
	Executor ExecutorType // 按执行器类型过滤
	TaskType TaskType     // 按任务类型过滤
	Page     int          // 页码，从1开始
	Size     int          // 每页数量
}

// ErrTaskNotFound 任务不存在错误
var ErrTaskNotFound = errors.New("task not found")

// ErrTaskNotPending 任务非等待状态错误
var ErrTaskNotPending = errors.New("task is not in pending status")

// SQLiteTaskRepository SQLite 实现的任务仓储
// 使用 database/sql 作为驱动，实现任务的持久化存储
type SQLiteTaskRepository struct {
	db *sql.DB
}

// NewSQLiteTaskRepository 创建 SQLite 仓储实例
// db: sql.DB 数据库连接实例
func NewSQLiteTaskRepository(db *sql.DB) *SQLiteTaskRepository {
	return &SQLiteTaskRepository{db: db}
}

// Create 实现 TaskRepository 接口的创建方法
func (r *SQLiteTaskRepository) Create(ctx context.Context, task *Task) error {
	task.BeforeCreate()

	configJSON, _ := encodeConfig(task.Config)
	resultJSON, _ := encodeResult(task.Result)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO scheduled_tasks (
			id, name, task_type, executor, schedule, next_run_at, last_run_at,
			config, status, result, retries, max_retries, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Name, task.TaskType, task.Executor, task.Schedule,
		task.NextRunAt, task.LastRunAt, configJSON, task.Status, resultJSON,
		task.Retries, task.MaxRetries, task.CreatedBy, task.CreatedAt, task.UpdatedAt,
	)
	return err
}

// Update 实现 TaskRepository 接口的更新方法
func (r *SQLiteTaskRepository) Update(ctx context.Context, task *Task) error {
	task.BeforeUpdate()

	configJSON, _ := encodeConfig(task.Config)
	resultJSON, _ := encodeResult(task.Result)

	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_tasks SET
			name = ?, task_type = ?, executor = ?, schedule = ?,
			next_run_at = ?, last_run_at = ?, config = ?, status = ?,
			result = ?, retries = ?, max_retries = ?, updated_at = ?
		WHERE id = ?`,
		task.Name, task.TaskType, task.Executor, task.Schedule,
		task.NextRunAt, task.LastRunAt, configJSON, task.Status, resultJSON,
		task.Retries, task.MaxRetries, task.UpdatedAt, task.ID,
	)
	return err
}

// Delete 实现 TaskRepository 接口的删除方法
func (r *SQLiteTaskRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM scheduled_tasks WHERE id = ?", id)
	return err
}

// GetByID 实现 TaskRepository 接口的按ID查询方法
func (r *SQLiteTaskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	var task Task
	var configJSON, resultJSON sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, task_type, executor, schedule, next_run_at, last_run_at,
			config, status, result, retries, max_retries, created_by, created_at, updated_at
		FROM scheduled_tasks WHERE id = ?`, id).Scan(
		&task.ID, &task.Name, &task.TaskType, &task.Executor, &task.Schedule,
		&task.NextRunAt, &task.LastRunAt, &configJSON, &task.Status, &resultJSON,
		&task.Retries, &task.MaxRetries, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}

	task.Config = decodeConfig(configJSON)
	task.Result = decodeResult(resultJSON)
	return &task, nil
}

// List 实现 TaskRepository 接口的列表查询方法
// 支持分页和多种过滤条件
func (r *SQLiteTaskRepository) List(ctx context.Context, filter TaskFilter) ([]*Task, int64, error) {
	// 构建基础查询
	baseSQL := "FROM scheduled_tasks WHERE 1=1"
	args := []interface{}{}

	if filter.Status != "" {
		baseSQL += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Executor != "" {
		baseSQL += " AND executor = ?"
		args = append(args, filter.Executor)
	}
	if filter.TaskType != "" {
		baseSQL += " AND task_type = ?"
		args = append(args, filter.TaskType)
	}

	// 统计总数
	var total int64
	countSQL := "SELECT COUNT(*) " + baseSQL
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (filter.Page - 1) * filter.Size
	selectSQL := "SELECT id, name, task_type, executor, schedule, next_run_at, last_run_at, " +
		"config, status, result, retries, max_retries, created_by, created_at, updated_at " +
		baseSQL + " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	args = append(args, filter.Size, offset)

	rows, err := r.db.QueryContext(ctx, selectSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var task Task
		var configJSON, resultJSON sql.NullString

		if err := rows.Scan(
			&task.ID, &task.Name, &task.TaskType, &task.Executor, &task.Schedule,
			&task.NextRunAt, &task.LastRunAt, &configJSON, &task.Status, &resultJSON,
			&task.Retries, &task.MaxRetries, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		task.Config = decodeConfig(configJSON)
		task.Result = decodeResult(resultJSON)
		tasks = append(tasks, &task)
	}

	return tasks, total, rows.Err()
}

// ListPending 实现 TaskRepository 接口的待执行任务查询方法
// 返回在指定时间之前需要执行的所有 pending 任务
func (r *SQLiteTaskRepository) ListPending(ctx context.Context, before int64) ([]*Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, task_type, executor, schedule, next_run_at, last_run_at,
			config, status, result, retries, max_retries, created_by, created_at, updated_at
		FROM scheduled_tasks
		WHERE status = ? AND next_run_at <= ?
		ORDER BY next_run_at ASC`,
		TaskStatusPending, before,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var task Task
		var configJSON, resultJSON sql.NullString

		if err := rows.Scan(
			&task.ID, &task.Name, &task.TaskType, &task.Executor, &task.Schedule,
			&task.NextRunAt, &task.LastRunAt, &configJSON, &task.Status, &resultJSON,
			&task.Retries, &task.MaxRetries, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, err
		}

		task.Config = decodeConfig(configJSON)
		task.Result = decodeResult(resultJSON)
		tasks = append(tasks, &task)
	}

	return tasks, rows.Err()
}

// ListByStatus 实现 TaskRepository 接口的按状态查询方法
func (r *SQLiteTaskRepository) ListByStatus(ctx context.Context, status TaskStatus) ([]*Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, task_type, executor, schedule, next_run_at, last_run_at,
			config, status, result, retries, max_retries, created_by, created_at, updated_at
		FROM scheduled_tasks WHERE status = ?`,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var task Task
		var configJSON, resultJSON sql.NullString

		if err := rows.Scan(
			&task.ID, &task.Name, &task.TaskType, &task.Executor, &task.Schedule,
			&task.NextRunAt, &task.LastRunAt, &configJSON, &task.Status, &resultJSON,
			&task.Retries, &task.MaxRetries, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, err
		}

		task.Config = decodeConfig(configJSON)
		task.Result = decodeResult(resultJSON)
		tasks = append(tasks, &task)
	}

	return tasks, rows.Err()
}

// ClaimTask 实现 TaskRepository 接口的任务认领方法
// 原子性地将 pending 状态的任务设置为 running 状态
// 防止多个 worker 同时执行同一任务
func (r *SQLiteTaskRepository) ClaimTask(ctx context.Context, taskID, workerID string) (*Task, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_tasks
		SET status = ?, updated_at = ?
		WHERE id = ? AND status = ?`,
		TaskStatusRunning, time.Now().Unix(), taskID, TaskStatusPending,
	)
	if err != nil {
		return nil, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrTaskNotPending
	}

	return r.GetByID(ctx, taskID)
}

// UpdateResult 实现 TaskRepository 接口的结果更新方法
func (r *SQLiteTaskRepository) UpdateResult(ctx context.Context, taskID string, result *TaskResult) error {
	resultJSON, _ := encodeResult(result)

	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_tasks
		SET result = ?, updated_at = ?
		WHERE id = ?`,
		resultJSON, time.Now().Unix(), taskID,
	)
	return err
}

// Ensure SQLiteTaskRepository 实现 TaskRepository 接口
var _ TaskRepository = (*SQLiteTaskRepository)(nil)

// encodeConfig 将 TaskConfig 编码为 JSON 字符串
func encodeConfig(config TaskConfig) (string, error) {
	if config == nil {
		return "", nil
	}
	return encodeJSON(config)
}

// decodeConfig 将 JSON 字符串解码为 TaskConfig
func decodeConfig(nullString sql.NullString) TaskConfig {
	if !nullString.Valid {
		return nil
	}
	var config TaskConfig
	_ = decodeJSON(nullString.String, &config)
	return config
}

// encodeResult 将 TaskResult 编码为 JSON 字符串
func encodeResult(result *TaskResult) (string, error) {
	if result == nil {
		return "", nil
	}
	return encodeJSON(result)
}

// decodeResult 将 JSON 字符串解码为 TaskResult
func decodeResult(nullString sql.NullString) *TaskResult {
	if !nullString.Valid {
		return nil
	}
	var result TaskResult
	if err := decodeJSON(nullString.String, &result); err != nil {
		return nil
	}
	return &result
}

// encodeJSON 通用 JSON 编码函数
func encodeJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	return string(data), err
}

// decodeJSON 通用 JSON 解码函数
func decodeJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// TaskRunRepository 任务执行记录仓储接口定义
// 提供任务执行记录的持久化操作
type TaskRunRepository interface {
	// Create 创建执行记录
	Create(ctx context.Context, run *TaskRun) error

	// Update 更新执行记录
	Update(ctx context.Context, run *TaskRun) error

	// Delete 删除执行记录
	Delete(ctx context.Context, id string) error

	// DeleteByTaskID 根据任务ID删除所有执行记录
	DeleteByTaskID(ctx context.Context, taskID string) error

	// GetByID 根据ID获取执行记录
	GetByID(ctx context.Context, id string) (*TaskRun, error)

	// ListByTaskID 根据任务ID获取执行记录列表
	ListByTaskID(ctx context.Context, taskID string, limit, offset int) ([]*TaskRun, int64, error)

	// List 获取执行记录列表(支持分页和状态过滤)
	List(ctx context.Context, filter TaskRunFilter) ([]*TaskRun, int64, error)
}

// TaskRunFilter 执行记录过滤条件
type TaskRunFilter struct {
	TaskID string        // 按任务ID过滤
	Status TaskRunStatus // 按执行状态过滤
	Page   int           // 页码，从0开始
	Size   int           // 每页数量
}

// ErrTaskRunNotFound 执行记录不存在错误
var ErrTaskRunNotFound = errors.New("task run not found")

// SQLiteTaskRunRepository SQLite 实现的任务执行记录仓储
type SQLiteTaskRunRepository struct {
	db *sql.DB
}

// NewSQLiteTaskRunRepository 创建执行记录仓储实例
func NewSQLiteTaskRunRepository(db *sql.DB) *SQLiteTaskRunRepository {
	return &SQLiteTaskRunRepository{db: db}
}

// Create 实现 TaskRunRepository 接口的创建方法
func (r *SQLiteTaskRunRepository) Create(ctx context.Context, run *TaskRun) error {
	run.BeforeCreate()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO task_runs (
			id, task_id, task_name, executor, input, status,
			start_time, end_time, duration, output, error, exit_code,
			retry_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.TaskID, run.TaskName, run.Executor, run.Input, run.Status,
		run.StartTime, run.EndTime, run.Duration, run.Output, run.Error, run.ExitCode,
		run.RetryCount, run.CreatedAt, run.UpdatedAt,
	)
	return err
}

// Update 实现 TaskRunRepository 接口的更新方法
func (r *SQLiteTaskRunRepository) Update(ctx context.Context, run *TaskRun) error {
	run.BeforeUpdate()

	run.Duration = run.CalculateDuration()

	_, err := r.db.ExecContext(ctx, `
		UPDATE task_runs SET
			status = ?, start_time = ?, end_time = ?, duration = ?,
			output = ?, error = ?, exit_code = ?, retry_count = ?,
			updated_at = ?
		WHERE id = ?`,
		run.Status, run.StartTime, run.EndTime, run.Duration,
		run.Output, run.Error, run.ExitCode, run.RetryCount,
		run.UpdatedAt, run.ID,
	)
	return err
}

// Delete 实现 TaskRunRepository 接口的删除方法
func (r *SQLiteTaskRunRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM task_runs WHERE id = ?", id)
	return err
}

// DeleteByTaskID 实现根据任务ID删除所有执行记录
func (r *SQLiteTaskRunRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM task_runs WHERE task_id = ?", taskID)
	return err
}

// GetByID 实现 TaskRunRepository 接口的ID查询方法
func (r *SQLiteTaskRunRepository) GetByID(ctx context.Context, id string) (*TaskRun, error) {
	var run TaskRun
	var inputJSON sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, task_id, task_name, executor, input, status,
			start_time, end_time, duration, output, error, exit_code,
			retry_count, created_at, updated_at
		FROM task_runs WHERE id = ?`, id,
	).Scan(
		&run.ID, &run.TaskID, &run.TaskName, &run.Executor, &inputJSON, &run.Status,
		&run.StartTime, &run.EndTime, &run.Duration, &run.Output, &run.Error, &run.ExitCode,
		&run.RetryCount, &run.CreatedAt, &run.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTaskRunNotFound
	}
	if err != nil {
		return nil, err
	}

	run.Input = inputJSON.String
	return &run, nil
}

// ListByTaskID 实现 TaskRunRepository 接口的任务ID查询方法
func (r *SQLiteTaskRunRepository) ListByTaskID(ctx context.Context, taskID string, limit, offset int) ([]*TaskRun, int64, error) {
	if limit <= 0 {
		limit = 20
	}

	// 获取总数
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_runs WHERE task_id = ?", taskID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, task_name, executor, input, status,
			start_time, end_time, duration, output, error, exit_code,
			retry_count, created_at, updated_at
		FROM task_runs WHERE task_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, taskID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []*TaskRun
	for rows.Next() {
		var run TaskRun
		var inputJSON sql.NullString
		err := rows.Scan(
			&run.ID, &run.TaskID, &run.TaskName, &run.Executor, &inputJSON, &run.Status,
			&run.StartTime, &run.EndTime, &run.Duration, &run.Output, &run.Error, &run.ExitCode,
			&run.RetryCount, &run.CreatedAt, &run.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		run.Input = inputJSON.String
		runs = append(runs, &run)
	}

	return runs, total, rows.Err()
}

// List 实现 TaskRunRepository 接口的分页查询方法
func (r *SQLiteTaskRunRepository) List(ctx context.Context, filter TaskRunFilter) ([]*TaskRun, int64, error) {
	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	offset := filter.Page * filter.Size

	// 构建 WHERE 条件
	where := "1=1"
	args := []interface{}{}
	if filter.TaskID != "" {
		where += " AND task_id = ?"
		args = append(args, filter.TaskID)
	}
	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, filter.Status)
	}

	// 获取总数
	var total int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_runs WHERE "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, task_name, executor, input, status,
			start_time, end_time, duration, output, error, exit_code,
			retry_count, created_at, updated_at
		FROM task_runs WHERE `+where+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, append(args, filter.Size, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []*TaskRun
	for rows.Next() {
		var run TaskRun
		var inputJSON sql.NullString
		err := rows.Scan(
			&run.ID, &run.TaskID, &run.TaskName, &run.Executor, &inputJSON, &run.Status,
			&run.StartTime, &run.EndTime, &run.Duration, &run.Output, &run.Error, &run.ExitCode,
			&run.RetryCount, &run.CreatedAt, &run.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		run.Input = inputJSON.String
		runs = append(runs, &run)
	}

	return runs, total, rows.Err()
}

// Ensure SQLiteTaskRunRepository 实现 TaskRunRepository 接口
var _ TaskRunRepository = (*SQLiteTaskRunRepository)(nil)
