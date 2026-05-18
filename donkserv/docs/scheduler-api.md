# 定时任务调度器 HTTP API 文档

## 概述

调度器提供完整的 REST API 接口，用于管理定时任务和查看执行记录。

**基础路径**: `/api/v1`

---

## 任务管理接口

### 创建任务

**请求**
```http
POST /api/v1/tasks
Content-Type: application/json
```

**请求体**
```json
{
  "name": "任务名称",                    // 必填，任务名称
  "task_type": "cron",                  // 必填，任务类型: cron/delay/once
  "schedule": "*/5 * * * *",            // 必填，调度表达式
  "executor": "script",                 // 必填，执行器类型: script/api/agent
  "config": {},                         // 可选，执行配置
  "max_retries": 3,                     // 可选，最大重试次数，默认3
  "created_by": "user123"               // 可选，创建者标识
}
```

**响应示例**
```json
{
  "message": "任务创建成功",
  "id": "uuid-string",
  "name": "任务名称",
  "task_type": "cron",
  "schedule": "*/5 * * * *",
  "status": "pending",
  "next_run_at": 1713000000,
  "created_at": 1712999900
}
```

**任务类型说明**
| 类型 | 说明 |
|------|------|
| `cron` | 定时循环任务，支持 cron 表达式 |
| `delay` | 延迟执行任务，支持时间戳或延迟表达式 |
| `once` | 单次执行任务 |

**执行器类型说明**
| 类型 | 说明 |
|------|------|
| `script` | 脚本执行器 |
| `api` | API 调用执行器 |
| `agent` | Agent 执行器 |

---

### 任务列表

**请求**
```http
GET /api/v1/tasks?page=1&size=20&status=running&executor=script
```

**查询参数**
| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `size` | int | 每页数量，默认 20，最大 100 |
| `status` | string | 按状态筛选 |
| `executor` | string | 按执行器类型筛选 |

**响应示例**
```json
{
  "items": [
    {
      "id": "uuid-string",
      "name": "任务名称",
      "task_type": "cron",
      "schedule": "*/5 * * * *",
      "executor": "script",
      "status": "pending",
      "next_run_at": 1713000000,
      "last_run_at": 1712999000,
      "created_at": 1712999900
    }
  ],
  "total": 100,
  "page": 1,
  "size": 20
}
```

---

### 获取任务详情

**请求**
```http
GET /api/v1/tasks/:id
```

**响应示例**
```json
{
  "id": "uuid-string",
  "name": "任务名称",
  "task_type": "cron",
  "schedule": "*/5 * * * *",
  "executor": "script",
  "config": {},
  "status": "pending",
  "result": {},
  "retries": 0,
  "max_retries": 3,
  "next_run_at": 1713000000,
  "last_run_at": 1712999000,
  "created_at": 1712999900,
  "updated_at": 1712999900,
  "created_by": "user123"
}
```

---

### 删除任务

**请求**
```http
DELETE /api/v1/tasks/:id
```

**响应示例**
```json
{
  "message": "任务删除成功"
}
```

---

### 取消任务

**请求**
```http
POST /api/v1/tasks/:id/cancel
```

**响应示例**
```json
{
  "message": "任务已取消"
}
```

---

### 手动触发任务

**请求**
```http
POST /api/v1/tasks/:id/run
```

**响应示例**
```json
{
  "message": "任务已触发执行"
}
```

---

### 获取任务执行结果

**请求**
```http
GET /api/v1/tasks/:id/result
```

**响应示例（已执行）**
```json
{
  "output": "执行输出",
  "exit_code": 0
}
```

**响应示例（未执行）**
```json
{
  "message": "任务尚未执行"
}
```

---

### 获取任务执行记录列表

**请求**
```http
GET /api/v1/tasks/:id/runs?page=1&size=20&status=done
```

**路径参数**
| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 任务ID |

**查询参数**
| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `size` | int | 每页数量，默认 20，最大 100 |
| `status` | string | 按执行状态筛选（running/done/failed） |

**响应示例**
```json
{
  "items": [
    {
      "id": "run-uuid",
      "task_id": "task-uuid",
      "task_name": "任务名称",
      "executor": "script",
      "input": "{}",
      "status": "done",
      "start_time": 1712999000,
      "end_time": 1712999010,
      "duration": 1000,
      "output": "执行输出",
      "error": "",
      "exit_code": 0,
      "retry_count": 0,
      "created_at": 1712999000,
      "updated_at": 1712999010
    }
  ],
  "total": 50,
  "page": 1,
  "size": 20
}
```

---

## 执行记录接口

### 执行记录列表

**请求**
```http
GET /api/v1/runs?page=0&size=20&task_id=xxx&status=completed
```

**查询参数**
| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 0 |
| `size` | int | 每页数量，默认 20，最大 100 |
| `task_id` | string | 按任务ID筛选 |
| `status` | string | 按执行状态筛选 |

**响应示例**
```json
{
  "items": [
    {
      "id": "run-uuid",
      "task_id": "task-uuid",
      "task_name": "任务名称",
      "executor": "script",
      "status": "completed",
      "start_time": 1712999000,
      "end_time": 1712999010,
      "duration": 1000,
      "exit_code": 0,
      "retry_count": 0,
      "created_at": 1712999000
    }
  ],
  "total": 50,
  "page": 0,
  "size": 20
}
```

---

### 获取执行记录详情

**请求**
```http
GET /api/v1/runs/:id
```

**响应示例**
```json
{
  "id": "run-uuid",
  "task_id": "task-uuid",
  "task_name": "任务名称",
  "executor": "script",
  "input": {},
  "status": "completed",
  "start_time": 1712999000,
  "end_time": 1712999010,
  "duration": 1000,
  "output": "执行输出",
  "error": "",
  "exit_code": 0,
  "retry_count": 0,
  "created_at": 1712999000,
  "updated_at": 1712999010
}
```

---

### 删除执行记录

**请求**
```http
DELETE /api/v1/runs/:id
```

**响应示例**
```json
{
  "message": "执行记录删除成功"
}
```

---

## 状态说明

### 任务状态
| 状态 | 说明 |
|------|------|
| `pending` | 待执行 |
| `running` | 执行中 |
| `paused` | 暂停 |
| `completed` | 已完成 |
| `failed` | 失败 |
| `cancelled` | 已取消 |

### 执行记录状态
| 状态 | 说明 |
|------|------|
| `running` | 运行中 |
| `completed` | 成功完成 |
| `failed` | 失败 |
| `cancelled` | 已取消 |

---

## 错误响应

所有接口的错误响应格式：

```json
{
  "error": "错误描述",
  "details": "详细信息（可选）"
}
```

常见 HTTP 状态码：
| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |