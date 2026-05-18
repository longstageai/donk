# Agent 与 WebSocket 通信协议

本文档定义了 Agent 与 WebSocket 之间的 JSON 通信格式。

## 客户端发送 (Client → Server)

### 聊天消息

```json
{
  "type": "chat",
  "content": "你好，请帮我介绍一下你自己"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 消息类型，目前固定为 `chat` |
| `content` | string | 是 | 用户输入的内容 |

---

## 服务端返回 (Server → Client)

### 1. 用户输入事件

```json
{
  "type": "stream",
  "event": "user_input",
  "content": "你好，请帮我介绍一下你自己"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `user_input` |
| `content` | string | 用户发送的原始输入 |

---

### 2. 思考过程 (Reasoning Delta)

```json
{
  "type": "stream",
  "event": "reasoning_delta",
  "reasoning_content": "用户想要了解我是什么，我应该先介绍我的身份..."
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `reasoning_delta` |
| `reasoning_content` | string | Agent 的思考过程增量 |

---

### 3. 内容增量 (Content Delta)

```json
{
  "type": "stream",
  "event": "content_delta",
  "content": "你好！我是"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `content_delta` |
| `content` | string | 文本内容增量 |

---

### 4. 助手完整回复

```json
{
  "type": "stream",
  "event": "assistant",
  "content": "你好！我是 donk，一个基于大语言模型的 AI Agent..."
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `assistant` |
| `content` | string | 助手完整回复内容 |

---

### 5. 工具调用

```json
{
  "type": "stream",
  "event": "tool_call",
  "tool_name": "calculator",
  "tool_input": "{\"expression\": \"10 + 20\"}"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `tool_call` |
| `tool_name` | string | 被调用的工具名称 |
| `tool_input` | string | 工具输入参数 (JSON字符串) |

---

### 6. 工具执行结果

```json
{
  "type": "stream",
  "event": "tool_result",
  "tool_name": "calculator",
  "tool_result": "30"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `tool_result` |
| `tool_name` | string | 工具名称 |
| `tool_result` | string | 工具执行结果 |

---

### 7. 任务正常停止

```json
{
  "type": "stream",
  "event": "stop"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `stop` |

---

### 8. 错误

```json
{
  "type": "stream",
  "event": "error",
  "error": "Token预算已超出"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `error` |
| `error` | string | 错误信息 |

---

### 9. 用户取消

```json
{
  "type": "stream",
  "event": "canceled"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | 固定为 `canceled` |

---

## Event 事件类型汇总

| 事件类型 | 方向 | 说明 |
|----------|------|------|
| `user_input` | Server→Client | 用户输入事件 |
| `reasoning_delta` | Server→Client | 思考过程增量 |
| `content_delta` | Server→Client | 内容增量 |
| `assistant` | Server→Client | 助手完整回复 |
| `tool_call` | Server→Client | 工具调用 |
| `tool_result` | Server→Client | 工具执行结果 |
| `stop` | Server→Client | 任务正常停止 |
| `error` | Server→Client | 错误 |
| `canceled` | Server→Client | 用户取消 |

---

## 通信流程

```
客户端                           服务端
  │                                │
  │──── {"type":"chat",          │
  │      "content":"..."} ───────►│
  │                                │
  │◄─── {"type":"stream",        │
  │      "event":"user_input"} ────│
  │                                │
  │◄─── {"type":"stream",        │
  │      "event":"reasoning_      │
  │       delta"} ────────────────│
  │                                │
  │◄─── {"type":"stream",        │
  │      "event":"content_delta"} │ (多次)
  │                                │
  │◄─── {"type":"stream",        │
  │      "event":"tool_call",     │
  │      "tool_name":"..."} ──────│
  │                                │
  │◄─── {"type":"stream",        │
  │      "event":"tool_result",   │
  │      "tool_result":"..."} ────│
  │                                │
  │◄─── {"type":"stream",        │
  │      "event":"stop"} ─────────│
  │                                │
```
