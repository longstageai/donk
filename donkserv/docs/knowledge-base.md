# 知识库模块

知识库模块负责扫描本地文档、解析内容、生成 Embedding，并构建可供 Agent 搜索的本地语义索引。

## 模块位置

```text
donkserv/internal/knowledge
```

## 核心能力

- 扫描本地目录中的 `txt`、`md`、`pdf`、`docx` 文件。
- 通过文件大小、修改时间、访问热度等因素进行优先级排序。
- 使用 Embedding 模型生成向量。
- 使用 SQLite 保存元数据。
- 使用向量存储保存语义索引。
- 通过 `knowledge_search` 工具供 Agent 检索。
- 通过设置 API 控制是否处理文档。

## 数据流

```text
目录扫描
  ↓
文件过滤
  ↓
优先级排序
  ↓
内容解析
  ↓
Embedding 生成
  ↓
元数据与向量索引写入
  ↓
Agent 工具检索
```

## 关键组件

| 文件 | 说明 |
| --- | --- |
| `config.go` | 知识库配置结构 |
| `scanner.go` | 文件扫描 |
| `priority.go` | 优先级计算 |
| `indexer.go` | 文档索引 |
| `embedder.go` | Embedding 集成 |
| `sqlite_store.go` | SQLite 元数据存储 |
| `vector_store.go` | 向量存储 |
| `runner.go` | 定时运行逻辑 |
| `initializer.go` | 启动初始化 |

## 配置入口

普通用户通过桌面端设置页面配置知识库。前端最终调用：

```text
GET  /api/v1/config/knowledge
PUT  /api/v1/config/knowledge
GET  /api/v1/knowledge/status
POST /api/v1/knowledge/start
POST /api/v1/knowledge/stop
```

当前 `enabled` 控制是否处理文档。定时器常驻运行，每次执行前读取配置。

## Embedding 依赖

知识库需要可用的 Embedding 配置。若 Embedding 未配置或初始化失败：

- 知识库索引不会正常构建。
- `knowledge_search` 工具不可用或返回空结果。
- 基础聊天仍可使用。

## 数据目录

运行时数据位于：

```text
donkserv/data/knowledge
```

该目录可能包含向量数据和元数据，不应提交到公开仓库。

## Agent 使用方式

Agent 通过内置工具 `knowledge_search` 使用知识库。典型输入包括：

```json
{
  "query": "项目配置说明",
  "keywords": "配置,模型",
  "limit": 5,
  "file_type": ".md"
}
```

实际参数以 `internal/tool/builtin/knowledge_search.go` 中工具 Schema 为准。
