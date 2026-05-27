# WebSocket 通知协议

`donkserv` 通过 WebSocket 向桌面端推送任务事件、系统通知和测试通知。当前 WebSocket 主要用于服务端事件推送，不承载主聊天协议；聊天流请参考 [agent_protocol.md](agent_protocol.md)。

## 连接地址

```text
ws://localhost:65434/ws/events
```

## 消息结构

### 任务事件

```json
{
  "type": "task.event",
  "action": "completed",
  "data": {
    "output": "执行结果"
  },
  "timestamp": 1779870000,
  "task_id": "task-id",
  "task_name": "任务名称",
  "status": "done",
  "error": ""
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `type` | string | 消息类型，任务事件为 `task.event` |
| `action` | string | 业务动作，如 `created`、`completed`、`failed` |
| `data` | any | 业务数据 |
| `timestamp` | integer | Unix 时间戳 |
| `task_id` | string | 关联任务 ID |
| `task_name` | string | 关联任务名称 |
| `status` | string | 当前状态 |
| `error` | string | 错误信息 |

### 系统通知

```json
{
  "type": "notification",
  "id": "uuid",
  "title": "任务完成",
  "content": "后台任务已执行完成",
  "level": "success"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `type` | string | 通知类型 |
| `id` | string | 通知 ID |
| `title` | string | 标题 |
| `content` | string | 内容 |
| `level` | string | 等级，如 `info`、`success`、`warning`、`error` |

## 测试推送

开发时可以手动触发一条测试通知：

```http
POST http://localhost:65434/ws/test-push
```

响应：

```json
{
  "status": "sent",
  "clientCount": 1,
  "title": "任务完成"
}
```

如果没有客户端连接：

```json
{
  "status": "no_clients",
  "clientCount": 0
}
```
