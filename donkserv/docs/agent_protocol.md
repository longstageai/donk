# Agent SSE 通信协议

`donkserv` 的主对话接口使用 HTTP SSE 返回流式 Agent 事件。桌面端通过一次 HTTP 请求提交用户输入，服务端持续推送推理、内容、工具调用和结束事件。

## 基础信息

```text
POST http://localhost:65434/api/v1/chat
Content-Type: application/json
Accept: text/event-stream
```

## 请求体

```json
{
  "content": "帮我总结今天要处理的任务",
  "file_path": "C:\\Users\\user\\Desktop\\report.pdf",
  "file_type": "pdf"
}
```

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `content` | string | 是 | 用户输入内容 |
| `file_path` | string | 否 | 附件路径。提供后会拼接进 Agent 输入 |
| `file_type` | string | 否 | 附件类型，支持 `pdf`、`docx`、`txt`、`md` |

当 `file_path` 为空时，Agent 直接接收 `content`。当存在附件时，后端会组装为：

```text
文件类型：{file_type}
文件路径：{file_path}
需求：{content}
```

## 响应格式

SSE 事件格式：

```text
event: <event_type>
data: <json>

```

后端会设置：

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

## 事件类型

| 事件 | 说明 |
| --- | --- |
| `user_input` | 已接收用户输入 |
| `reasoning_delta` | 推理内容增量 |
| `content_delta` | 回复内容增量 |
| `assistant` | 助手完整回复 |
| `tool_call` | Agent 发起工具调用 |
| `tool_result` | 工具执行结果 |
| `warning` | 警告信息，如 Token 预算提示 |
| `error` | 执行错误 |
| `stop` | 正常结束 |
| `canceled` | 用户取消或请求上下文取消 |
| `heartbeat` | 心跳事件，`data` 为 `ping` |

## 事件数据示例

### user_input

```text
event: user_input
data: {"type":"stream","event":"user_input","content":"你好"}
```

### reasoning_delta

```text
event: reasoning_delta
data: {"type":"stream","event":"reasoning_delta","reasoning_content":"用户希望了解系统能力..."}
```

### content_delta

```text
event: content_delta
data: {"type":"stream","event":"content_delta","content":"Donk 可以帮助你"}
```

### tool_call

```text
event: tool_call
data: {"type":"stream","event":"tool_call","tool_name":"knowledge_search","tool_input":"{\"query\":\"项目文档\"}"}
```

### tool_result

```text
event: tool_result
data: {"type":"stream","event":"tool_result","tool_name":"knowledge_search","tool_result":"..."}
```

### assistant

```text
event: assistant
data: {"type":"stream","event":"assistant","content":"完整回复内容"}
```

### stop

```text
event: stop
data: {"type":"stream","event":"stop","usage":{"prompt_tokens":100,"completion_tokens":80,"total_tokens":180}}
```

## 错误响应

请求参数错误会直接返回 HTTP JSON：

```json
{
  "error": "无效的请求参数: ..."
}
```

Agent 执行过程中的错误会以 SSE `error` 事件返回：

```text
event: error
data: {"type":"stream","event":"error","error":"LLM配置不存在"}
```

## 典型流程

```text
Client -> POST /api/v1/chat
Server -> user_input
Server -> reasoning_delta
Server -> content_delta
Server -> tool_call
Server -> tool_result
Server -> content_delta
Server -> assistant
Server -> stop
```
