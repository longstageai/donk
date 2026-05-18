# 模型厂商配置汇总

本文档汇总了 donk 项目支持的所有 LLM 和 Embedding 模型厂商的配置信息，包括默认模型、Base URL 等。

---

## 一、LLM 大语言模型厂商

### 1. OpenAI

| 配置项 | 值 |
|--------|-----|
| **Provider ID** | `openai` |
| **默认 Base URL** | `https://api.openai.com` |
| **默认模型** | `gpt-4o-mini` |
| **API 认证方式** | Bearer Token |
| **API 端点** | `/v1/chat/completions` |

**支持的模型示例：**
- `gpt-4o-mini` - 轻量级模型，性价比高
- `gpt-4o` - 多模态旗舰模型
- `gpt-4-turbo` - 高性能模型
- `gpt-3.5-turbo` - 经济型模型

---

### 2. DeepSeek

| 配置项 | 值 |
|--------|-----|
| **Provider ID** | `deepseek` |
| **默认 Base URL** | `https://api.deepseek.com` |
| **默认模型** | `deepseek-chat` |
| **API 认证方式** | Bearer Token |
| **API 端点** | `/v1/chat/completions` |

**支持的模型示例：**
- `deepseek-chat` - 通用对话模型
- `deepseek-reasoner` - 推理模型（支持 reasoning_content）

**特点：**
- 支持思维链输出（reasoning_content）
- 国产大模型，中文表现优秀

---

### 3. Qwen (通义千问)

| 配置项 | 值 |
|--------|-----|
| **Provider ID** | `qwen` |
| **默认 Base URL** | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| **默认模型** | `qwen-turbo` |
| **API 认证方式** | Bearer Token |
| **API 端点** | `/chat/completions` |

**支持的模型示例：**
- `qwen-turbo` - 快速响应模型
- `qwen-plus` - 增强版模型
- `qwen-max` - 最强性能模型
- `qwen-coder-plus` - 代码专用模型

**特点：**
- 阿里云百炼平台提供
- OpenAI 兼容模式接口
- 支持 reasoning_content 输出

---

### 4. Doubao (豆包)

| 配置项 | 值 |
|--------|-----|
| **Provider ID** | `doubao` |
| **默认 Base URL** | `https://ark.cn-beijing.volces.com` |
| **默认模型** | `doubao-seed-1-8-251228` |
| **API 认证方式** | Bearer Token |
| **API 端点** | `/api/v3/chat/completions` |
| **超时时间** | 5 分钟 |

**支持的模型示例：**
- `doubao-seed-1-8-251228` - Seed 系列模型
- `doubao-pro-32k` - 专业版 32K 上下文
- `doubao-lite-4k` - 轻量版 4K 上下文

**特点：**
- 字节跳动火山引擎提供
- 支持超长上下文（最高 256K）

---

## 二、Embedding 向量模型厂商

### 1. OpenAI

| 配置项 | 值 |
|--------|-----|
| **Provider ID** | `openai` |
| **默认 Base URL** | `https://api.openai.com` |
| **默认模型** | `text-embedding-3-small` |
| **默认维度** | 1536 |
| **API 认证方式** | Bearer Token |

**支持的模型：**

| 模型名称 | 默认维度 | 可配置维度 | 说明 |
|----------|----------|------------|------|
| `text-embedding-3-small` | 1536 | ❌ 固定 | 小型模型，性能与成本平衡 |
| `text-embedding-3-large` | 3072 | ❌ 固定 | 大型模型，精度更高 |
| `text-embedding-ada-002` | 1536 | ❌ 固定 | 旧版模型，已逐渐被 v3 替代 |

**最大输入长度：** 8192 tokens

---

### 2. Qwen (通义千问)

| 配置项 | 值 |
|--------|-----|
| **Provider ID** | `qwen` |
| **默认 Base URL** | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| **默认模型** | `text-embedding-v3` |
| **默认维度** | 1024 |
| **API 认证方式** | Bearer Token |

**支持的模型：**

| 模型名称 | 默认维度 | 可配置维度 | 说明 |
|----------|----------|------------|------|
| `text-embedding-v4` | 1024 | ✅ 64/128/256/512/768/1024/1536/2048 | 最新版本，维度灵活 |
| `text-embedding-v3` | 1024 | ✅ 64/128/256/512/768/1024 | 推荐版本，性价比高 |
| `text-embedding-v2` | 1536 | ❌ 固定 | 旧版模型 |
| `text-embedding-v1` | 1536 | ❌ 固定 | 初代模型 |

**最大输入长度：** 8192 tokens

---

### 3. Doubao (豆包)

| 配置项 | 值 |
|--------|-----|
| **Provider ID** | `doubao` |
| **默认 Base URL** | `https://ark.cn-beijing.volces.com/api/v3` |
| **默认模型** | `doubao-embedding-vision-250328` |
| **默认维度** | 2048 |
| **API 认证方式** | Bearer Token |

**支持的模型：**

| 模型名称 | 默认维度 | 可配置维度 | 说明 |
|----------|----------|------------|------|
| `doubao-embedding-vision-250328` | 2048 | ✅ 1024/2048 | 多模态模型，支持图文 |
| `doubao-embedding-vision-250615` | 2048 | ✅ 1024/2048 | 最新多模态版本，支持 128k 上下文 |
| `doubao-embedding-large-240915` | 2048 | ✅ 512/1024/2048/4096 | 大型文本模型 |
| `doubao-embedding-text-240715` | 2048 | ✅ 512/1024/2048 | 最高支持 2560 维 |
| `doubao-embedding-text-240515` | 2048 | ✅ 512/1024 | 早期文本模型 |

---

## 三、数据库默认配置

### Config 表默认值

```sql
-- LLM 配置默认值
llm_provider TEXT NOT NULL DEFAULT 'openai'
llm_model TEXT NOT NULL DEFAULT 'gpt-4o-mini'
llm_base_url TEXT NOT NULL DEFAULT ''
llm_temperature REAL NOT NULL DEFAULT 0.7
llm_max_tokens INTEGER NOT NULL DEFAULT 4096

-- Embedding 配置默认值
embedding_provider TEXT NOT NULL DEFAULT 'openai'
embedding_model TEXT NOT NULL DEFAULT 'text-embedding-3-small'
embedding_base_url TEXT NOT NULL DEFAULT ''
embedding_dimension INTEGER NOT NULL DEFAULT 1536
```

---

## 四、配置文件示例

### YAML 配置 (conf/config.yaml)

```yaml
llm:
  provider: doubao
  model: doubao-seed-1-8-251228
  api_key: your-api-key
  # base_url: 可选，留空使用默认值

embedding:
  provider: doubao
  model: ep-20260325174412-vz86z
  api_key: your-api-key
  # base_url: 可选，留空使用默认值
```

---

## 五、维度兼容性说明

### 不同厂商默认维度对比

| 厂商 | 默认维度 | 维度特点 |
|------|----------|----------|
| **OpenAI** | 1536 | 固定维度 |
| **Qwen** | 1024 | v3/v4 支持灵活配置 |
| **Doubao** | 2048 | 支持灵活配置 |

### 推荐的统一维度

为了在不同厂商间获得更好的兼容性，建议统一使用以下维度：

| 推荐维度 | 适用场景 | 支持的厂商/模型 |
|---------|---------|----------------|
| **1024** | 通用场景，存储节省 | Qwen v3/v4、Doubao (可配置) |
| **1536** | 与 OpenAI 兼容 | OpenAI 全系列、Qwen v1/v2 |
| **2048** | 高精度需求 | Doubao 全系列、Qwen v4 |

### 切换厂商注意事项

1. **同一厂商同一模型**: ✅ 完全兼容，维度一致
2. **同一厂商不同模型**: ⚠️ 需检查维度是否变化
3. **不同厂商间切换**: ❌ 不兼容，必须重建向量库

---

## 六、快速参考表

### LLM 厂商配置速查

| 厂商 | Provider ID | 默认 Base URL | 默认模型 |
|------|-------------|---------------|----------|
| OpenAI | `openai` | `https://api.openai.com` | `gpt-4o-mini` |
| DeepSeek | `deepseek` | `https://api.deepseek.com` | `deepseek-chat` |
| Qwen | `qwen` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `qwen-turbo` |
| Doubao | `doubao` | `https://ark.cn-beijing.volces.com` | `doubao-seed-1-8-251228` |

### Embedding 厂商配置速查

| 厂商 | Provider ID | 默认 Base URL | 默认模型 | 默认维度 |
|------|-------------|---------------|----------|----------|
| OpenAI | `openai` | `https://api.openai.com` | `text-embedding-3-small` | 1536 |
| Qwen | `qwen` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `text-embedding-v3` | 1024 |
| Doubao | `doubao` | `https://ark.cn-beijing.volces.com/api/v3` | `doubao-embedding-vision-250328` | 2048 |

---

## 七、相关代码文件

| 功能 | 文件路径 |
|------|----------|
| LLM 接口定义 | `internal/model/llm.go` |
| LLM 适配器工厂 | `internal/model/adapter.go` |
| OpenAI LLM 实现 | `internal/model/openai.go` |
| DeepSeek LLM 实现 | `internal/model/deepseek.go` |
| Qwen LLM 实现 | `internal/model/qwen.go` |
| Doubao LLM 实现 | `internal/model/doubao.go` |
| Embedding 接口定义 | `internal/embedding/embedder.go` |
| OpenAI Embedding 实现 | `internal/embedding/openai.go` |
| Qwen Embedding 实现 | `internal/embedding/qwen.go` |
| Doubao Embedding 实现 | `internal/embedding/doubao.go` |
| 配置模型定义 | `internal/setting/model.go` |
| 数据库表结构 | `internal/sql/tables.go` |

---

*文档生成时间: 2026-05-14*