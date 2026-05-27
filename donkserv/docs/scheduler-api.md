# 任务调度 API

调度器提供任务管理和执行记录 API。任务支持 `cron`、`delay`、`once` 三种类型，执行器支持 `script`、`api`、`agent`。

## 基础地址

```text
http://localhost:65434/api/v1
```

## 任务接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/tasks` | 创建任务 |
| `GET` | `/api/v1/tasks` | 查询任务列表 |
| `GET` | `/api/v1/tasks/:id` | 查询任务详情 |
| `DELETE` | `/api/v1/tasks/:id` | 删除任务 |
| `POST` | `/api/v1/tasks/:id/cancel` | 取消任务 |
| `POST` | `/api/v1/tasks/:id/run` | 手动触发任务 |
| `GET` | `/api/v1/tasks/:id/result` | 获取任务结果 |
| `GET` | `/api/v1/tasks/:id/runs` | 获取任务执行记录 |

## 执行记录接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/runs` | 查询执行记录列表 |
| `GET` | `/api/v1/runs/:id` | 查询执行记录详情 |
| `DELETE` | `/api/v1/runs/:id` | 删除执行记录 |

## 创建任务

```http
POST /api/v1/tasks
Content-Type: application/json
```

```json
{
  "name": "每日总结",
  "task_type": "cron",
  "schedule": "0 8 * * *",
  "executor": "agent",
  "config": {
    "prompt": "总结昨天的任务和今天的计划",
    "timeout": 300
  },
  "max_retries": 3,
  "created_by": "user"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 任务名称 |
| `task_type` | string | 是 | `cron`、`delay`、`once` |
| `schedule` | string | 是 | Cron 表达式、延迟时间或 RFC3339/Unix 时间戳 |
| `executor` | string | 是 | `script`、`api`、`agent` |
| `config` | object | 否 | 执行器配置 |
| `max_retries` | integer | 否 | 最大重试次数，默认 3 |
| `created_by` | string | 否 | 创建者 |

响应：

```json
{
  "message": "任务创建成功",
  "id": "uuid",
  "name": "每日总结",
  "task_type": "cron",
  "schedule": "0 8 * * *",
  "status": "pending",
  "next_run_at": 1779870000,
  "created_at": 1779860000
}
```

## 查询任务列表

```http
GET /api/v1/tasks?page=1&size=20&status=pending&executor=agent
```

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` | integer | 页码，默认 1 |
| `size` | integer | 每页数量，默认 20，最大 100 |
| `status` | string | 任务状态过滤 |
| `executor` | string | 执行器过滤 |

响应：

```json
{
  "items": [
    {
      "id": "uuid",
      "name": "每日总结",
      "task_type": "cron",
      "schedule": "0 8 * * *",
      "executor": "agent",
      "status": "pending",
      "next_run_at": 1779870000,
      "last_run_at": 0,
      "created_at": 1779860000
    }
  ],
  "total": 1,
  "page": 1,
  "size": 20
}
```

## 状态说明

### 任务状态

| 状态 | 说明 |
| --- | --- |
| `pending` | 等待执行 |
| `running` | 执行中 |
| `done` | 执行完成 |
| `failed` | 执行失败 |
| `cancelled` | 已取消 |

### 执行记录状态

执行记录状态与任务执行过程一致，常见为 `running`、`done`、`failed`。

## WebSocket 事件

任务状态变化会通过 `GET /ws/events` 推送。消息类型为：

```json
{
  "type": "task.event",
  "action": "completed",
  "task_id": "uuid",
  "task_name": "每日总结",
  "status": "done",
  "data": {},
  "timestamp": 1779870000
}
```
