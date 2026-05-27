# 设置与系统 API

Setting 模块负责 LLM、Embedding、Agent、知识库、睡眠管理和 Token 统计的 API。普通用户通过桌面端设置页面配置这些内容，HTTP API 主要供前端和调试使用。

## 基础地址

```text
http://localhost:65434/api/v1
```

## 配置 API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/config` | 获取完整配置 |
| `PUT` | `/api/v1/config` | 部分更新完整配置 |
| `GET` | `/api/v1/config/llm` | 获取 LLM 配置 |
| `PUT` | `/api/v1/config/llm` | 更新 LLM 配置 |
| `GET` | `/api/v1/config/embedding` | 获取 Embedding 配置 |
| `PUT` | `/api/v1/config/embedding` | 更新 Embedding 配置 |
| `GET` | `/api/v1/config/agent` | 获取 Agent 配置 |
| `PUT` | `/api/v1/config/agent` | 更新 Agent 配置 |
| `GET` | `/api/v1/config/knowledge` | 获取知识库配置和运行状态 |
| `PUT` | `/api/v1/config/knowledge` | 更新知识库启用状态 |

## LLM 配置

```http
PUT /api/v1/config/llm
Content-Type: application/json
```

```json
{
  "provider": "openai",
  "model": "gpt-4o-mini",
  "api_key": "sk-xxx",
  "base_url": "",
  "temperature": 0.7,
  "max_tokens": 4096
}
```

支持的 Provider 由 `internal/model` 适配层决定，当前包括 `openai`、`qwen`、`deepseek`、`doubao`。

## Embedding 配置

```http
PUT /api/v1/config/embedding
Content-Type: application/json
```

```json
{
  "provider": "openai",
  "model": "text-embedding-3-small",
  "api_key": "sk-xxx",
  "base_url": "",
  "dimension": 1536
}
```

Embedding 用于知识库检索和长期记忆。未配置时，基础聊天仍可使用，但语义检索能力不可用。

## Agent 配置

```http
PUT /api/v1/config/agent
Content-Type: application/json
```

```json
{
  "name": "donk",
  "max_loop": 10,
  "converge_after": 3,
  "timeout": 300,
  "daily_token_limit": -1,
  "history_max_entries": 100,
  "history_max_days": 30
}
```

## 知识库配置与控制

```http
GET /api/v1/config/knowledge
PUT /api/v1/config/knowledge
GET /api/v1/knowledge/status
POST /api/v1/knowledge/start
POST /api/v1/knowledge/stop
```

`PUT /api/v1/config/knowledge` 示例：

```json
{
  "enabled": true
}
```

说明：

- `enabled` 控制是否处理文档。
- 知识库定时器始终运行，每次执行前读取配置。
- `start` 和 `stop` 接口保留为控制接口，当前主要返回运行状态和操作提示。

## 睡眠管理

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/system/sleep` | 获取睡眠管理状态 |
| `POST` | `/api/v1/system/sleep/prevent` | 阻止系统睡眠 |
| `POST` | `/api/v1/system/sleep/allow` | 恢复系统睡眠 |

阻止睡眠请求：

```json
{
  "keep_display": true
}
```

## Token 统计

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/tokens/usage` | 查询 Token 使用记录 |
| `GET` | `/api/v1/tokens/budget` | 查询今日 Token 预算 |

## 健康检查

```http
GET /health
```

响应：

```json
{
  "status": "ok"
}
```

## 持久化说明

配置存储在本地 SQLite 数据库中。桌面端设置页面调用这些 API 更新配置，应用重启后继续使用保存的本地配置。
