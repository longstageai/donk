# Agent 通信协议

本文档定义了 Agent 与客户端之间的通信协议，支持 WebSocket 和 HTTP SSE 两种方式。

## 通信方式

| 方式 | 协议 | 适用场景 | 特点 |
|------|------|----------|------|
| HTTP SSE | Server-Sent Events | 实时对话 | 单向流式，客户端发送一次，服务端持续推送 |
| WebSocket | WebSocket | 实时对话 | 双向通信，支持更复杂的交互 |

---

## HTTP SSE 协议

### 接口端点

```
POST /api/v1/chat
Content-Type: application/json
```

### 客户端请求

#### 请求头

| 字段 | 值 | 说明 |
|------|-----|------|
| `Content-Type` | `application/json` | 请求体格式 |
| `Accept` | `text/event-stream` | 接受SSE流式响应 |

#### 请求体

```json
{
  "content": "你好，请帮我介绍一下你自己"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `content` | string | 是 | 用户输入的消息内容 |

### 服务端响应 (SSE)

SSE 格式：`event: <event_type>\ndata: <json>\n\n`

#### 1. 用户输入确认 (user_input)

```
event: user_input
data: {"type":"stream","event":"user_input","content":"你好，请帮我介绍一下你自己"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `user_input` |
| `content` | string | 用户输入的原始内容 |

#### 2. 思考过程 (reasoning_delta)

```
event: reasoning_delta
data: {"type":"stream","event":"reasoning_delta","reasoning_content":"用户想要了解我是什么..."}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `reasoning_delta` |
| `reasoning_content` | string | Agent 的思考过程增量 |

#### 3. 内容增量 (content_delta)

```
event: content_delta
data: {"type":"stream","event":"content_delta","content":"你好！我是"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `content_delta` |
| `content` | string | 文本内容增量 |

#### 4. 助手完整回复 (assistant)

```
event: assistant
data: {"type":"stream","event":"assistant","content":"你好！我是 donk..."}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `assistant` |
| `content` | string | 助手完整回复内容 |

#### 5. 工具调用 (tool_call)

```
event: tool_call
data: {"type":"stream","event":"tool_call","tool_name":"calculator","tool_input":"{\"expression\":\"10+20\"}"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `tool_call` |
| `tool_name` | string | 被调用的工具名称 |
| `tool_input` | string | 工具输入参数 (JSON字符串) |

#### 6. 工具执行结果 (tool_result)

```
event: tool_result
data: {"type":"stream","event":"tool_result","tool_name":"calculator","tool_result":"30"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `tool_result` |
| `tool_name` | string | 工具名称 |
| `tool_result` | string | 工具执行结果 |

#### 7. 警告 (warning)

```
event: warning
data: {"type":"stream","event":"warning","content":"Token预算即将耗尽"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `warning` |
| `content` | string | 警告信息 |

#### 8. 任务完成 (stop)

```
event: stop
data: {"type":"stream","event":"stop"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `stop` |

#### 9. 错误 (error)

```
event: error
data: {"type":"stream","event":"error","error":"Token预算已超出"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `error` |
| `error` | string | 错误信息 |

#### 10. 用户取消 (canceled)

```
event: canceled
data: {"type":"stream","event":"canceled"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `stream` |
| `event` | string | 固定为 `canceled` |

---

## WebSocket 协议

### 连接端点

```
ws://<host>/ws
```

### 客户端发送消息

#### 聊天消息

```json
{
  "type": "chat",
  "content": "你好，请帮我介绍一下你自己"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 消息类型，固定为 `chat` |
| `content` | string | 是 | 用户输入的内容 |

### 服务端推送消息

WebSocket 消息格式与 SSE 的 `data` 字段相同：

```json
{
  "type": "stream",
  "event": "user_input",
  "content": "你好，请帮我介绍一下你自己"
}
```

事件类型与 SSE 协议完全一致，参见上文。

---

## 事件类型汇总

| 事件类型 | 方向 | 触发时机 | 说明 |
|----------|------|----------|------|
| `user_input` | Server→Client | 收到用户输入 | 确认收到用户消息 |
| `reasoning_delta` | Server→Client | LLM思考中 | 推理模型的思考过程 |
| `content_delta` | Server→Client | LLM生成中 | 文本内容增量 |
| `assistant` | Server→Client | LLM生成完成 | 助手完整回复 |
| `tool_call` | Server→Client | 调用工具 | 工具名称和参数 |
| `tool_result` | Server→Client | 工具执行完成 | 工具执行结果 |
| `warning` | Server→Client | 预算不足等 | 警告信息 |
| `stop` | Server→Client | 任务正常结束 | 对话完成 |
| `error` | Server→Client | 发生错误 | 错误信息 |
| `canceled` | Server→Client | 用户取消 | 任务被取消 |

---

## 典型对话流程

### 简单对话（无工具调用）

```
Client → Server: POST /api/v1/chat {content:"你好"}
Server → Client: event: user_input
Server → Client: event: reasoning_delta (可选，推理模型)
Server → Client: event: content_delta (多次，流式输出)
Server → Client: event: assistant
Server → Client: event: stop
```

### 工具调用对话

```
Client → Server: POST /api/v1/chat {content:"计算10+20"}
Server → Client: event: user_input
Server → Client: event: reasoning_delta
Server → Client: event: tool_call {tool_name:"calculator", tool_input:"{...}"}
Server → Client: event: tool_result {tool_name:"calculator", tool_result:"30"}
Server → Client: event: content_delta
Server → Client: event: assistant
Server → Client: event: stop
```

### 多轮工具调用

```
Client → Server: POST /api/v1/chat {content:"复杂任务"}
Server → Client: event: user_input
Server → Client: event: tool_call (工具1)
Server → Client: event: tool_result (工具1结果)
Server → Client: event: tool_call (工具2)
Server → Client: event: tool_result (工具2结果)
...
Server → Client: event: assistant
Server → Client: event: stop
```

---

## 数据结构定义

### Go 结构定义

```go
// StreamEvent 流式事件结构
type StreamEvent struct {
    Type             StreamEventType  // 事件类型
    Content          string           // 文本内容
    ReasoningContent string           // 思考过程内容
    ToolName         string           // 工具名称
    ToolInput        string           // 工具输入参数
    ToolResult       string           // 工具执行结果
    Error            string           // 错误信息
    Usage            schema.UsageInfo // Token使用量统计
}

// StreamEventType 事件类型枚举
type StreamEventType string

const (
    EventUserInput      StreamEventType = "user_input"
    EventReasoningDelta StreamEventType = "reasoning_delta"
    EventContentDelta   StreamEventType = "content_delta"
    EventAssistant      StreamEventType = "assistant"
    EventToolCall       StreamEventType = "tool_call"
    EventToolResult     StreamEventType = "tool_result"
    EventWarning        StreamEventType = "warning"
    EventError          StreamEventType = "error"
    EventStop           StreamEventType = "stop"
    EventCanceled       StreamEventType = "canceled"
)
```

---

## 错误处理

### 客户端错误

| HTTP状态码 | 说明 |
|------------|------|
| 400 | 请求参数错误 |
| 500 | 服务器内部错误 |

### 服务端错误事件

当 Agent 执行过程中发生错误时，会发送 `error` 事件：

```
event: error
data: {"type":"stream","event":"error","error":"Token预算已超出"}
```

常见错误：
- `Token预算已超出` - Token使用量超过限制
- `达到最大循环次数` - ReAct循环超过最大次数
- `任务收敛超时` - 连续多轮无工具调用
- `任务被用户取消` - 用户主动取消任务

---

## 心跳机制 (SSE)

SSE 连接每 30 秒发送一次心跳：

```
event: heartbeat
data: {}
```

用于保持连接活跃，客户端可忽略此事件。
