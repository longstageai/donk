# Agent 定时任务系统 - 设计方案

## 一、系统架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            Master Agent                                 │
│  - 任务编排与决策                                                        │
│  - 提供 LLM Tools (create/cancel/delete/list/get_result)               │
│  - 订阅任务事件                                                         │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ 任务管理
                                 ↓
┌────────────────────────────────┴────────────────────────────────────────┐
│                           Scheduler                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │
│  │ TaskStore    │  │ Cron/Timer   │  │ EventBus     │  │  Executor  │ │
│  │ (持久化)      │  │ (定时调度)    │  │ (消息通知)    │  │ (执行器)   │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘ │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ 任务分发
           ┌─────────────────────┼─────────────────────┐
           ↓                     ↓                     ↓
    ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
    │ Script       │     │ API           │     │ Agent        │
    │ Executor     │     │ Executor      │     │ Executor     │
    │ (脚本/命令)   │     │ (HTTP调用)    │     │ (LLM执行)    │
    └──────────────┘     └──────────────┘     └──────────────┘
```

## 二、核心组件

### 2.1 任务模型

```go
type Task struct {
    ID          string                 // 任务ID (UUID)
    Name        string                 // 任务名称

    // 调度配置
    TaskType    TaskType               // cron | delay | once
    Schedule    string                 // cron表达式 / 延迟时间 / 时间戳
    NextRunAt   int64                  // 下次执行时间
    LastRunAt   int64                  // 上次执行时间

    // 执行配置
    Executor    ExecutorType           // script | api | agent
    Config      TaskConfig             // 执行器配置

    // 状态与结果
    Status      TaskStatus             // pending | running | done | failed | cancelled
    Result      *TaskResult            // 执行结果
    Retries     int                    // 当前重试次数
    MaxRetries  int                    // 最大重试次数

    // 元数据
    CreatedBy   string                 // 创建者
    CreatedAt   int64
    UpdatedAt   int64
}

type TaskType string
const (
    TaskTypeCron  TaskType = "cron"
    TaskTypeDelay TaskType = "delay"
    TaskTypeOnce  TaskType = "once"
)

type ExecutorType string
const (
    ExecutorScript ExecutorType = "script"
    ExecutorAPI    ExecutorType = "api"
    ExecutorAgent  ExecutorType = "agent"
)

type TaskStatus string
const (
    TaskStatusPending   TaskStatus = "pending"
    TaskStatusRunning   TaskStatus = "running"
    TaskStatusDone      TaskStatus = "done"
    TaskStatusFailed    TaskStatus = "failed"
    TaskStatusCancelled TaskStatus = "cancelled"
)
```

### 2.2 执行器接口

```go
type Executor interface {
    Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

// Script Executor - 执行本地命令/脚本
type ScriptExecutor struct{}
func (e *ScriptExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error)

// API Executor - 调用 HTTP API
type APIExecutor struct{}
func (e *APIExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error)

// Agent Executor - 调用 LLM Agent
type AgentExecutor struct{}
func (e *AgentExecutor) Execute(ctx context.Context, task *Task) (*TaskResult, error)
```

## 三、定时调度

### 3.1 混合调度器

```go
type Scheduler struct {
    repo      TaskRepository           // 持久化层
    cron      *cron.Cron               // Cron 调度器
    timers    map[string]*time.Timer // 延迟任务
    executor  Executor                // 执行器
    eventBus  *EventBus               // 事件总线
    ctx       context.Context
    cancel    context.CancelFunc
}
```

### 3.2 调度策略

| 任务类型 | 调度方式 |
|----------|----------|
| cron | cron 库 (robfig/cron) |
| delay | time.AfterFunc |
| once | time.AfterFunc (执行后删除) |

### 3.3 cron 表达式

```
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日期 (1 - 31)
│ │ │ ┌───────────── 月份 (1 - 12)
│ │ │ │ ┌───────────── 星期 (0 - 6)
│ │ │ │ │
* * * * *

示例:
*/5 * * * *     每5分钟
0 2 * * *       每天凌晨2点
0 */2 * * *     每隔2小时
0 9-18 * * 1-5  工作日每天9点到18点
```

## 四、持久化

### 4.1 数据库表设计

```sql
CREATE TABLE scheduled_tasks (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    task_type   TEXT NOT NULL,
    executor    TEXT NOT NULL,
    schedule    TEXT NOT NULL,
    next_run_at INTEGER NOT NULL,
    last_run_at INTEGER,
    config      TEXT,
    status      TEXT DEFAULT 'pending',
    result      TEXT,
    retries     INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    created_by  TEXT,

    INDEX idx_status (status),
    INDEX idx_next_run (next_run_at)
);
```

### 4.2 重启恢复

```go
func (s *Scheduler) Recover() error {
    // 1. 恢复 pending 任务 → 重新加入调度
    // 2. 恢复 running 任务 → 判断是否重试
    // 3. 处理超时任务 → 重试执行
}
```

## 五、消息传递

### 5.1 事件类型

```go
type EventType string

const (
    EventTaskCreated   EventType = "task:created"
    EventTaskStarted   EventType = "task:started"
    EventTaskCompleted EventType = "task:completed"
    EventTaskFailed    EventType = "task:failed"
    EventTaskCancelled EventType = "task:cancelled"
)
```

### 5.2 事件订阅

```go
// 可订阅: Log / Webhook / WebSocket / LLM回调
type Subscriber interface {
    OnEvent(event *TaskEvent)
}
```

## 六、REST API

### 6.1 接口清单

```
任务 CRUD:
POST   /api/v1/tasks           创建任务
GET    /api/v1/tasks           列表查询
GET    /api/v1/tasks/:id       详情
DELETE /api/v1/tasks/:id       删除

任务操作:
POST   /api/v1/tasks/:id/cancel  取消
POST   /api/v1/tasks/:id/run     手动触发

结果查询:
GET    /api/v1/tasks/:id/result   执行结果
GET    /api/v1/tasks/:id/history  执行历史
```

### 6.2 Request/Response

#### 创建任务

```go
type CreateTaskRequest struct {
    Name       string                 `json:"name" binding:"required"`
    TaskType   string                 `json:"task_type" binding:"required"`  // cron | delay | once
    Schedule   string                 `json:"schedule" binding:"required"`  // cron表达式 | 延迟 | 时间戳
    Executor   string                 `json:"executor" binding:"required"`   // script | api | agent
    Config     map[string]interface{} `json:"config"`                        // 执行配置
    MaxRetries int                    `json:"max_retries"`                   // 默认3
}
```

#### 列表查询

```go
type ListTasksRequest struct {
    Status   string `form:"status"`    // pending | running | done | failed | cancelled
    Executor string `form:"executor"`  // script | api | agent
    Page     int    `form:"page"`      // 默认1
    Size     int    `form:"size"`      // 默认20，最大100
}
```

#### 执行结果

```go
type TaskResult struct {
    Output   string `json:"output"`
    Error    string `json:"error,omitempty"`
    ExitCode int    `json:"exit_code"`
    DoneAt   int64  `json:"done_at"`
    Duration int64  `json:"duration"`  // 毫秒
}
```

### 6.3 使用示例

```bash
# 创建 cron 任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "服务器状态检查",
    "task_type": "cron",
    "schedule": "*/5 * * * *",
    "executor": "agent",
    "config": {
      "prompt": "检查服务器状态，返回JSON格式"
    }
  }'

# 列表查询
curl "http://localhost:8080/api/v1/tasks?status=pending&page=1&size=10"

# 手动触发
curl -X POST "http://localhost:8080/api/v1/tasks/task-xxx/run"

# 获取结果
curl "http://localhost:8080/api/v1/tasks/task-xxx/result"
```

## 七、LLM 工具

### 7.1 Master Agent 提供的工具

```go
var LLMtools = []Tool{
    {
        Name: "create_task",
        Description: "创建一个定时任务",
        Params: map[string]Param{
            "name":      {"type": "string", "description": "任务名称"},
            "task_type": {"type": "string", "enum": ["cron", "delay", "once"]},
            "schedule":  {"type": "string", "description": "cron表达式 或 延迟时间"},
            "executor":  {"type": "string", "enum": ["script", "api", "agent"]},
            "config":    {"type": "object", "description": "执行配置"},
            "max_retries": {"type": "int", "description": "最大重试次数"},
        }
    },
    {
        Name: "cancel_task",
        Description: "取消一个等待执行的定时任务",
    },
    {
        Name: "delete_task",
        Description: "删除一个任务及其所有记录",
    },
    {
        Name: "list_tasks",
        Description: "查看所有定时任务及其状态",
    },
    {
        Name: "get_task_result",
        Description: "获取任务执行结果",
    },
}
```

### 7.2 LLM 使用示例

```
用户: "每5分钟检查一次服务器状态"
LLM:  create_task({
    name: "服务器状态检查",
    task_type: "cron",
    schedule: "*/5 * * * *",
    executor: "agent",
    config: {prompt: "检查服务器状态，返回JSON格式"}
})
→ 返回: {task_id: "xxx", next_run: "14:05:00"}

用户: "任务执行得怎么样"
LLM:  get_task_result({task_id: "xxx"})
→ 返回: {status: "done", result: {output: "所有服务器正常"}}
```

## 八、数据流

```
┌────────────────────────────────────────────────────────────────────────┐
│                              创建任务                                    │
│  Master LLM / REST API → create_task() → Scheduler → DB + 调度器      │
└────────────────────────────────┬───────────────────────────────────────┘
                                 ↓
┌────────────────────────────────────────────────────────────────────────┐
│                              定时触发                                    │
│  Cron/Timer 触发 → 更新 status=running → 分发给 Executor              │
└────────────────────────────────┬───────────────────────────────────────┘
                                 ↓
┌────────────────────────────────────────────────────────────────────────┐
│                              执行任务                                    │
│  Executor 执行 → 成功/失败 → 更新 status + result → 发送事件          │
└────────────────────────────────┬───────────────────────────────────────┘
                                 ↓
┌────────────────────────────────────────────────────────────────────────┐
│                              事件通知                                     │
│  EventBus → Log / Webhook / WebSocket / LLM回调                        │
└────────────────────────────────┬───────────────────────────────────────┘
                                 ↓
┌────────────────────────────────────────────────────────────────────────┐
│                              重试调度                                    │
│  失败 → 可重试? → pending → 重新加入调度                                │
└────────────────────────────────┬───────────────────────────────────────┘
                                 ↓
┌────────────────────────────────────────────────────────────────────────┐
│                              重启恢复                                    │
│  启动 → Recover() → 加载 pending/running → 重新加入调度                 │
└────────────────────────────────────────────────────────────────────────┘
```

## 九、部署

```
┌─────────────────────────────────────────────┐
│              单机部署                         │
│  ┌─────────────────────────────────────┐    │
│  │  SQLite + Scheduler + Executor      │    │
│  │  REST API Server                    │    │
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

## 十、技术选型

| 模块 | 技术选择 | 理由 |
|------|----------|------|
| **语言** | Go | 高并发、简单、部署方便 |
| **数据库** | SQLite | 单机轻量、部署简单、无需额外服务 |
| **Cron** | robfig/cron | 稳定、简单、功能完整 |
| **HTTP** | Gin | 高性能、易用 |
| **ORM** | GORM / XORM | 简化数据库操作 |

## 十一、设计原则

1. **可扩展性**: 执行器通过接口实现，便于扩展新的执行类型
2. **可靠性**: 任务持久化存储，支持重启恢复
3. **解耦**: 调度、执行、通知分离，通过事件总线通信
4. **简洁**: 不过度设计，按需实现

## 十二、Agent 执行器详解

### 12.1 轻量级 Agent

定时任务 Agent 采用轻量级实现，不包含主 Agent 的复杂功能：

| 组件 | 定时任务 Agent | 主 Agent |
|------|---------------|----------|
| LLM 模型 | ✅ | ✅ |
| HTTP 工具 | ✅ | ✅ |
| Calculator 工具 | ✅ | ✅ |
| 中间件 (Log/Timeout/Retry) | ✅ | ✅ |
| 长期记忆 (向量数据库) | ❌ | ✅ |
| 历史记录存储 | ❌ | ✅ |
| 知识库管理 | ❌ | ✅ |
| Token 预算控制 | ❌ | ✅ |
| 技能系统 | ❌ | ✅ |

### 12.2 实现原理

```go
// AgentExecutor 通过工厂函数获取 Agent 实例
type AgentExecutor struct {
    agentFactory func() interface{}
}

// 执行流程：
// 1. 从任务配置获取 prompt
// 2. 调用 agentFactory() 创建轻量级 Agent 实例
// 3. 调用 agent.Run(ctx, prompt) 同步执行
// 4. 返回执行结果
```

### 12.3 注入方式

```go
// 在应用启动时注入 Agent 工厂
sched.SetAgentFactory(func() interface{} {
    agentInstance, _ := NewTaskAgent(app)
    return agentInstance
})
```

### 12.4 创建任务示例

通过 REST API 创建 Agent 任务：

```bash
curl -X POST http://localhost:8081/api/v1/scheduler/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "每日数据报告",
    "task_type": "cron",
    "schedule": "0 8 * * *",
    "executor": "agent",
    "prompt": "请分析过去24小时的用户行为数据，生成一份简洁的数据报告",
    "timeout": 600
  }'
```

通过 Master Agent 创建：

```
用户: "创建一个每天早上8点执行的数据分析任务"
LLM调用 task_manager 工具:
{
    "action": "create",
    "name": "每日数据报告",
    "task_type": "cron",
    "schedule": "0 8 * * *",
    "executor": "agent",
    "prompt": "请分析过去24小时的用户行为数据，生成一份简洁的数据报告",
    "timeout": 600
}
```

### 12.5 任务参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 任务名称 |
| task_type | string | 是 | 任务类型：cron/delay/once |
| schedule | string | 是 | 调度表达式 |
| executor | string | 是 | 执行器类型：script/api/agent |
| prompt | string | 是* | Agent 任务提示词（agent执行器必填）|
| timeout | int | 否 | Agent 执行超时时间（秒），默认300 |
| command | string | 是* | 脚本命令（script执行器必填）|
| api_url | string | 是* | API 地址（api执行器必填）|
| api_method | string | 否 | API 请求方法，默认 GET |
| api_headers | object | 否 | API 请求头 |
| api_body | object | 否 | API 请求体 |
| max_retries | int | 否 | 最大重试次数，默认3 |

## 十三、执行记录

### 13.1 表结构

```sql
CREATE TABLE task_runs (
    id          TEXT PRIMARY KEY,
    task_id     TEXT NOT NULL,
    status      TEXT NOT NULL,
    input       TEXT,
    output      TEXT,
    error       TEXT,
    exit_code   INTEGER,
    duration    INTEGER,
    started_at  INTEGER NOT NULL,
    finished_at INTEGER,
    
    INDEX idx_task_id (task_id),
    INDEX idx_started_at (started_at)
);
```

### 13.2 特性

- **历史保留**: 每次执行都会记录，不覆盖
- **输入保存**: 保存任务的 prompt/command/api_url 等输入参数
- **输出保存**: 保存 Agent 的回复或脚本的输出
- **状态追踪**: running/done/failed 三种状态

### 13.3 执行记录 API

#### 接口清单

```
GET    /api/v1/runs              列表查询（支持分页、筛选）
GET    /api/v1/runs/:id          详情查询
DELETE /api/v1/runs/:id          删除记录
```

#### 列表查询

```bash
# 查询所有执行记录（分页）
curl "http://localhost:8081/api/v1/runs?page=0&size=20"

# 按任务ID筛选
curl "http://localhost:8081/api/v1/runs?task_id=task_xxx"

# 按状态筛选
curl "http://localhost:8081/api/v1/runs?status=done"

# 组合筛选
curl "http://localhost:8081/api/v1/runs?task_id=task_xxx&status=failed&size=50"
```

**Query 参数说明：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，从 0 开始，默认 0 |
| size | int | 每页数量，默认 20，最大 100 |
| task_id | string | 按任务 ID 筛选 |
| status | string | 按状态筛选：running/done/failed |

**响应示例：**

```json
{
  "items": [
    {
      "id": "run_xxx",
      "task_id": "task_xxx",
      "task_name": "每日数据报告",
      "executor": "agent",
      "status": "done",
      "start_time": 1712000000,
      "end_time": 1712000060,
      "duration": 60,
      "exit_code": 200,
      "retry_count": 0,
      "created_at": 1712000000
    }
  ],
  "total": 100,
  "page": 0,
  "size": 20
}
```

#### 详情查询

```bash
# 获取单条执行记录详情
curl "http://localhost:8081/api/v1/runs/run_xxx"
```

**响应示例：**

```json
{
  "id": "run_xxx",
  "task_id": "task_xxx",
  "task_name": "每日数据报告",
  "executor": "agent",
  "input": "请分析过去24小时的用户行为数据...",
  "status": "done",
  "start_time": 1712000000,
  "end_time": 1712000060,
  "duration": 60,
  "output": "数据分析报告：\n1. 新增用户：100人\n2. 活跃用户：500人\n...",
  "error": "",
  "exit_code": 200,
  "retry_count": 0,
  "created_at": 1712000000,
  "updated_at": 1712000060
}
```

#### 删除记录

```bash
# 删除单条执行记录
curl -X DELETE "http://localhost:8081/api/v1/runs/run_xxx"
```

### 13.4 Master Agent 工具

Master Agent 可以通过 task_manager 工具管理执行记录：

```json
{
  "action": "run_list",
  "task_id": "task_xxx",
  "run_limit": 20,
  "run_offset": 0,
  "run_status": "done"
}
```

```json
{
  "action": "run_get",
  "run_id": "run_xxx"
}
```

```json
{
  "action": "run_delete",
  "run_id": "run_xxx"
}
```

## 14. WebSocket 事件通道

### 14.1 概述

WebSocket 作为项目的统一实时消息通道，负责将任务事件实时推送给客户端。定位为"**后台任务 + 系统通知通道**"，不处理 Agent 会话消息。

### 14.2 消息类型

| 类型 | 来源 | 说明 |
|------|------|------|
| `task.event` | Scheduler | 任务创建、开始、完成、失败、取消 |
| `system` | System | 系统通知、告警 |

### 14.3 消息格式

```json
{
  "type": "task.event",
  "action": "completed",
  "task_id": "task_xxx",
  "task_name": "每日数据报告",
  "status": "done",
  "data": {
    "output": "...",
    "error": "",
    "exitCode": 200,
    "duration": 60
  },
  "timestamp": 1712000000
}
```

### 14.4 事件类型

| 事件名 | 说明 |
|--------|------|
| task:created | 任务创建 |
| task:started | 任务开始执行 |
| task:completed | 任务执行完成 |
| task:failed | 任务执行失败 |
| task:cancelled | 任务被取消 |

### 14.5 连接方式

```javascript
// 连接 WebSocket
const ws = new WebSocket('ws://localhost:8081/ws/tasks');

// 接收消息
ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type === 'task.event') {
        console.log('任务事件:', msg.action, msg.task_id);
    }
};

// 心跳
setInterval(() => {
    ws.send(JSON.stringify({type: 'ping'}));
}, 30000);
```

### 14.6 消息示例

**任务完成事件：**
```json
{
  "type": "task.event",
  "action": "completed",
  "task_id": "task_xxx",
  "task_name": "每日数据报告",
  "status": "done",
  "data": {
    "output": "报告已生成",
    "exitCode": 200,
    "duration": 60
  },
  "timestamp": 1712000060
}
```

**任务失败事件：**
```json
{
  "type": "task.event",
  "action": "failed",
  "task_id": "task_xxx",
  "task_name": "每日数据报告",
  "status": "failed",
  "data": {
    "error": "Agent 执行失败: timeout"
  },
  "timestamp": 1712000100
}
```

### 14.7 代码结构

```
internal/websocket/
├── message.go   # 统一消息格式
├── hub.go       # 连接管理
├── client.go    # 客户端处理
└── server.go    # HTTP Upgrader
```
