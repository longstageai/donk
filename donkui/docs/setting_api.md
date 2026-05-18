# Setting 模块 API 接口文档

## 概述

Setting 模块提供配置管理功能，包括完整配置、LLM 配置、Embedding 配置和 Agent 配置的增删改查。所有配置存储在单一的 `config` 表中，通过不同前缀区分不同模块的配置。

## 基础信息

- 基础路径: `/api/v1`
- 数据格式: JSON
- 认证方式: Bearer Token (API Key)

## 数据库表结构

```sql
CREATE TABLE IF NOT EXISTS config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    llm_provider TEXT NOT NULL DEFAULT 'openai',
    llm_model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
    llm_api_key TEXT NOT NULL DEFAULT '',
    llm_base_url TEXT NOT NULL DEFAULT '',
    llm_temperature REAL NOT NULL DEFAULT 0.7,
    llm_max_tokens INTEGER NOT NULL DEFAULT 4096,
    embedding_provider TEXT NOT NULL DEFAULT 'openai',
    embedding_model TEXT NOT NULL DEFAULT 'text-embedding-3-small',
    embedding_api_key TEXT NOT NULL DEFAULT '',
    embedding_base_url TEXT NOT NULL DEFAULT '',
    embedding_dimension INTEGER NOT NULL DEFAULT 1536,
    agent_name TEXT NOT NULL DEFAULT 'donk',
    agent_max_loop INTEGER NOT NULL DEFAULT 10,
    agent_converge_after INTEGER NOT NULL DEFAULT 3,
    agent_timeout INTEGER NOT NULL DEFAULT 300,
    agent_daily_token_limit INTEGER NOT NULL DEFAULT -1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
```

**说明**: `config` 表有且只有一条数据，初始化时自动创建默认配置。

## 初始化

### 方式一：统一数据库（推荐）

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/longstageai/donk/donk/internal/setting"
    "github.com/longstageai/donk/donk/internal/sql"
)

// 创建单一数据库连接
db, err := sql.Open("./data/donk.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// 创建 Gin 引擎
engine := gin.Default()

// 初始化 Setting 模块（registerSchema=true 表示创建表结构）
engine, err = setting.Setup(db.DB, engine, true)
if err != nil {
    log.Fatal(err)
}

// 启动服务
engine.Run(":8080")
```

### 方式二：独立使用

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/longstageai/donk/donk/internal/setting"
)

// 创建 Gin 引擎
engine := gin.Default()

// 使用数据库文件路径初始化
engine, err := setting.SetupWithPath("./data/setting/setting.db", engine)
if err != nil {
    log.Fatal(err)
}

// 启动服务
engine.Run(":8080")
```

---

## 接口列表

### 1. 健康检查

| 项目 | 说明 |
|------|------|
| 接口 | `GET /health` |
| 认证 | 无需认证 |
| 响应 | `{"status": "ok"}` |

**示例请求**
```bash
curl -X GET http://localhost:8080/health
```

**响应**
```json
{
    "status": "ok"
}
```

---

### 2. 完整配置

#### 获取完整配置

| 项目 | 说明 |
|------|------|
| 接口 | `GET /api/v1/config` |
| 认证 | 需要 |
| 响应 | Config 对象 |

**示例请求**
```bash
curl -X GET http://localhost:8080/api/v1/config \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应**
```json
{
    "id": 1,
    "llm_provider": "openai",
    "llm_model": "gpt-4o-mini",
    "llm_api_key": "sk-xxx",
    "llm_base_url": "",
    "llm_temperature": 0.7,
    "llm_max_tokens": 4096,
    "embedding_provider": "openai",
    "embedding_model": "text-embedding-3-small",
    "embedding_api_key": "sk-xxx",
    "embedding_base_url": "",
    "embedding_dimension": 1536,
    "agent_name": "donk",
    "agent_max_loop": 10,
    "agent_converge_after": 3,
    "agent_timeout": 300,
    "agent_daily_token_limit": -1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
}
```

#### 更新完整配置（支持部分更新）

| 项目 | 说明 |
|------|------|
| 接口 | `PUT /api/v1/config` |
| 认证 | 需要 |
| 请求体 | ConfigUpdateRequest |

**说明**: 所有字段均为可选，未传字段保持数据库原有值。

**请求字段**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| llm_provider | string | 否 | LLM 提供商 (openai/deepseek/qwen/doubao) |
| llm_model | string | 否 | LLM 模型名称 |
| llm_api_key | string | 否 | LLM API 密钥 |
| llm_base_url | string | 否 | LLM API 基础 URL |
| llm_temperature | float64 | 否 | LLM 温度参数 (0-2) |
| llm_max_tokens | int | 否 | LLM 最大输出 token 数 |
| embedding_provider | string | 否 | Embedding 提供商 |
| embedding_model | string | 否 | Embedding 模型名称 |
| embedding_api_key | string | 否 | Embedding API 密钥 |
| embedding_base_url | string | 否 | Embedding API 基础 URL |
| embedding_dimension | int | 否 | Embedding 向量维度 |
| agent_name | string | 否 | Agent 名称 |
| agent_max_loop | int | 否 | Agent 最大循环次数 |
| agent_converge_after | int | 否 | Agent 连续无工具调用终止数 |
| agent_timeout | int | 否 | Agent 超时时间（秒） |
| agent_daily_token_limit | int | 否 | Agent 每日 Token 限额（-1 表示不限） |

**示例请求（只更新 LLM 模型）**
```bash
curl -X PUT http://localhost:8080/api/v1/config \
  -H "Authorization: Bearer sk-your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "llm_model": "deepseek-chat"
}'
```

**响应**
```json
{
    "message": "更新成功"
}
```

---

### 3. LLM 配置

#### 获取 LLM 配置

| 项目 | 说明 |
|------|------|
| 接口 | `GET /api/v1/config/llm` |
| 认证 | 需要 |
| 响应 | LLMConfigResponse 对象 |

**示例请求**
```bash
curl -X GET http://localhost:8080/api/v1/config/llm \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应**
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

#### 更新 LLM 配置

| 项目 | 说明 |
|------|------|
| 接口 | `PUT /api/v1/config/llm` |
| 认证 | 需要 |
| 请求体 | LLMConfigRequest |

**请求字段**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| provider | string | 是 | LLM 提供商 (openai/deepseek/qwen/doubao) |
| model | string | 是 | LLM 模型名称 |
| api_key | string | 否 | LLM API 密钥 |
| base_url | string | 否 | LLM API 基础 URL |
| temperature | float64 | 否 | LLM 温度参数 (0-2) |
| max_tokens | int | 否 | LLM 最大输出 token 数 |

**示例请求**
```bash
curl -X PUT http://localhost:8080/api/v1/config/llm \
  -H "Authorization: Bearer sk-your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "deepseek",
    "model": "deepseek-chat",
    "api_key": "sk-xxx",
    "base_url": "https://api.deepseek.com",
    "temperature": 0.7,
    "max_tokens": 4096
}'
```

**响应**
```json
{
    "message": "更新成功"
}
```

---

### 4. Embedding 配置

#### 获取 Embedding 配置

| 项目 | 说明 |
|------|------|
| 接口 | `GET /api/v1/config/embedding` |
| 认证 | 需要 |
| 响应 | EmbeddingConfigResponse 对象 |

**示例请求**
```bash
curl -X GET http://localhost:8080/api/v1/config/embedding \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应**
```json
{
    "provider": "openai",
    "model": "text-embedding-3-small",
    "api_key": "sk-xxx",
    "base_url": "",
    "dimension": 1536
}
```

#### 更新 Embedding 配置

| 项目 | 说明 |
|------|------|
| 接口 | `PUT /api/v1/config/embedding` |
| 认证 | 需要 |
| 请求体 | EmbeddingConfigRequest |

**请求字段**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| provider | string | 是 | Embedding 提供商 |
| model | string | 是 | Embedding 模型名称 |
| api_key | string | 否 | Embedding API 密钥 |
| base_url | string | 否 | Embedding API 基础 URL |
| dimension | int | 否 | Embedding 向量维度 |

**示例请求**
```bash
curl -X PUT http://localhost:8080/api/v1/config/embedding \
  -H "Authorization: Bearer sk-your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "model": "text-embedding-3-small",
    "api_key": "sk-xxx",
    "base_url": "",
    "dimension": 1536
}'
```

**响应**
```json
{
    "message": "更新成功"
}
```

---

### 5. Agent 配置

#### 获取 Agent 配置

| 项目 | 说明 |
|------|------|
| 接口 | `GET /api/v1/config/agent` |
| 认证 | 需要 |
| 响应 | AgentConfigResponse 对象 |

**示例请求**
```bash
curl -X GET http://localhost:8080/api/v1/config/agent \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应**
```json
{
    "name": "donk",
    "max_loop": 10,
    "converge_after": 3,
    "timeout": 300,
    "daily_token_limit": -1
}
```

#### 更新 Agent 配置

| 项目 | 说明 |
|------|------|
| 接口 | `PUT /api/v1/config/agent` |
| 认证 | 需要 |
| 请求体 | AgentConfigRequest |

**请求字段**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Agent 名称 |
| max_loop | int | 否 | 最大循环次数 |
| converge_after | int | 否 | 连续无工具调用终止数 |
| timeout | int | 否 | 超时时间（秒） |
| daily_token_limit | int | 否 | 每日 Token 限额（-1 表示不限） |

**示例请求**
```bash
curl -X PUT http://localhost:8080/api/v1/config/agent \
  -H "Authorization: Bearer sk-your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "donk",
    "max_loop": 10,
    "converge_after": 3,
    "timeout": 300,
    "daily_token_limit": -1
}'
```

**响应**
```json
{
    "message": "更新成功"
}
```

---

## 默认配置值

| 模块 | 字段 | 默认值 | 说明 |
|------|------|--------|------|
| LLM | provider | openai | LLM 提供商 |
| LLM | model | gpt-4o-mini | LLM 模型名称 |
| LLM | temperature | 0.7 | 温度参数 |
| LLM | max_tokens | 4096 | 最大 token 数 |
| Embedding | provider | openai | Embedding 提供商 |
| Embedding | model | text-embedding-3-small | Embedding 模型名称 |
| Embedding | dimension | 1536 | 向量维度 |
| Agent | name | donk | Agent 名称 |
| Agent | max_loop | 10 | 最大循环次数 |
| Agent | converge_after | 3 | 连续无工具调用终止数 |
| Agent | timeout | 300 | 超时时间（秒） |
| Agent | daily_token_limit | -1 | 每日 Token 限额（-1 表示不限） |

---

## 6. Token 统计

Token 统计模块提供 Token 使用记录的查询和预算状态查看功能。

### 数据库表结构

```sql
CREATE TABLE IF NOT EXISTS token_daily_usage (
    date TEXT PRIMARY KEY,                                 -- 日期(YYYYMMDD格式)
    total_tokens INTEGER NOT NULL DEFAULT 0,               -- 总Token数
    input_tokens INTEGER NOT NULL DEFAULT 0,               -- 输入Token数
    output_tokens INTEGER NOT NULL DEFAULT 0,              -- 输出Token数
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')) -- 更新时间
);
```

---

### 6.1 获取 Token 使用记录列表

| 项目 | 说明 |
|------|------|
| 接口 | `GET /api/v1/tokens/usage` |
| 认证 | 需要 |
| 功能 | 分页查询历史 Token 使用记录，按日期倒序排列 |

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，从1开始，默认1 |
| page_size | int | 否 | 每页条数，默认20，最大100 |

**响应字段**

| 字段 | 类型 | 说明 |
|------|------|------|
| items | array | Token 使用记录列表 |
| items[].date | string | 日期，格式 20060102 |
| items[].total_tokens | int | 总 Token 数 |
| items[].input_tokens | int | 输入 Token 数 |
| items[].output_tokens | int | 输出 Token 数 |
| items[].updated_at | string | 更新时间，ISO8601格式 |
| total | int | 总记录数 |
| page | int | 当前页码 |
| page_size | int | 每页条数 |

**示例请求**
```bash
curl -X GET "http://localhost:8080/api/v1/tokens/usage?page=1&page_size=10" \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应示例**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "items": [
            {
                "date": "20260422",
                "total_tokens": 15000,
                "input_tokens": 5000,
                "output_tokens": 10000,
                "updated_at": "2026-04-22T15:30:00Z"
            },
            {
                "date": "20260421",
                "total_tokens": 12000,
                "input_tokens": 4000,
                "output_tokens": 8000,
                "updated_at": "2026-04-21T18:20:00Z"
            }
        ],
        "total": 50,
        "page": 1,
        "page_size": 10
    }
}
```

---

### 6.2 获取今日 Token 预算状态

| 项目 | 说明 |
|------|------|
| 接口 | `GET /api/v1/tokens/budget` |
| 认证 | 需要 |
| 功能 | 获取今日 Token 使用量、剩余额度、是否超限等状态 |

**响应字段**

| 字段 | 类型 | 说明 |
|------|------|------|
| date | string | 当前日期，格式 20060102 |
| limit | int | 每日限额，-1 表示无限制 |
| used | int | 今日已使用 Token 数 |
| remaining | int | 剩余可用 Token 数，-1 表示无限制 |
| usage_percent | float64 | 使用百分比（limit > 0 时计算） |
| is_limited | bool | 是否设置了限额（limit > 0） |
| is_exceeded | bool | 是否已超出限额 |

**示例请求**
```bash
curl -X GET http://localhost:8080/api/v1/tokens/budget \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应示例（有限额时）**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "date": "20260422",
        "limit": 100000,
        "used": 15000,
        "remaining": 85000,
        "usage_percent": 15.0,
        "is_limited": true,
        "is_exceeded": false
    }
}
```

**响应示例（无限制时）**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "date": "20260422",
        "limit": -1,
        "used": 15000,
        "remaining": -1,
        "usage_percent": 0,
        "is_limited": false,
        "is_exceeded": false
    }
}
```

---

## 7. 系统睡眠管理（仅 Windows）

系统睡眠管理模块提供通过 API 控制 Windows 系统进入睡眠状态的功能。使用 Windows API `SetThreadExecutionState` 实现，程序退出后自动恢复。

---

### 7.1 获取睡眠管理状态

| 项目 | 说明 |
|------|------|
| 接口 | `GET /api/v1/system/sleep` |
| 认证 | 需要 |
| 功能 | 获取当前睡眠管理状态 |

**响应字段**

| 字段 | 类型 | 说明 |
|------|------|------|
| is_active | bool | 是否正在阻止睡眠 |
| keep_display | bool | 是否保持显示器开启 |

**示例请求**
```bash
curl -X GET http://localhost:8080/api/v1/system/sleep \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应示例**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "is_active": true,
        "keep_display": true
    }
}
```

---

### 7.2 阻止系统睡眠

| 项目 | 说明 |
|------|------|
| 接口 | `POST /api/v1/system/sleep/prevent` |
| 认证 | 需要 |
| 功能 | 阻止系统进入睡眠状态 |

**请求字段**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keep_display | bool | 否 | 是否保持显示器开启，默认 false |

**示例请求**
```bash
curl -X POST http://localhost:8080/api/v1/system/sleep/prevent \
  -H "Authorization: Bearer sk-your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "keep_display": true
}'
```

**响应示例**
```json
{
    "code": 0,
    "message": "已阻止系统睡眠",
    "data": {
        "is_active": true,
        "keep_display": true
    }
}
```

---

### 7.3 允许系统睡眠

| 项目 | 说明 |
|------|------|
| 接口 | `POST /api/v1/system/sleep/allow` |
| 认证 | 需要 |
| 功能 | 恢复系统默认睡眠行为 |

**示例请求**
```bash
curl -X POST http://localhost:8080/api/v1/system/sleep/allow \
  -H "Authorization: Bearer sk-your-secret-key"
```

**响应示例**
```json
{
    "code": 0,
    "message": "已恢复系统睡眠",
    "data": {
        "is_active": false,
        "keep_display": false
    }
}
```

**注意事项**
- 仅 Windows 系统有效，其他系统调用会返回错误
- 阻止睡眠状态在程序退出后自动恢复
- 长时间阻止睡眠会增加系统能耗

---

## 错误码

| HTTP 状态码 | 说明 |
|------------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（缺少或无效的认证 Token） |
| 404 | 配置不存在 |
| 500 | 服务器内部错误 |
