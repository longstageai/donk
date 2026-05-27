# Knowledge 模块架构

## 概述

Knowledge 模块负责知识库的构建和管理，提供文档上传、分块、向量化存储和语义搜索功能。

## 目录结构

```
internal/knowledge/
├── store.go       # 知识存储实现
├── search.go      # 知识检索
├── manager.go     # 知识库管理器
└── chunker.go    # 文本分块
```

## 核心组件

### 1. KnowledgeStore（知识存储）

负责文档的向量化和持久化存储。

```go
type KnowledgeStore struct {
    embedder embedding.Embedder  // 向量化器
    store    db.VectorStore     // 向量数据库
}
```

### 2. KnowledgeSearch（知识检索）

负责接收检索参数并执行语义搜索。

```go
type KnowledgeSearch struct {
    store    *KnowledgeStore
    embedder embedding.Embedder
}

type SearchParams struct {
    Query      string
    TopK       int
    TimeFilter *TimeFilter
}
```

### 3. KnowledgeManager（知识库管理器）

负责知识库的完整生命周期管理。

```go
type KnowledgeManager struct {
    config        Config
    store         *KnowledgeStore
    search        *KnowledgeSearch
}
```

### 4. TextSplitter（文本分块）

将长文档拆分成适合向量化的文本块。

```go
type TextSplitter struct {
    chunkSize    int
    chunkOverlap int
}
```

## 核心流程

### 添加文档

```
用户上传文档 → 文本分块 → 向量化 → 存储到向量数据库
```

### 搜索知识

```
用户查询 → 向量化查询 → 向量数据库搜索 → 返回结果
```

## 使用示例

```go
// 创建知识库管理器
manager, err := knowledge.NewManager(cfg, embedder, store)

// 添加文档
err := manager.AddDocument(ctx, "文档标题", "文档内容...")

// 搜索知识
results, err := manager.Search(ctx, "用户问题", 5, nil)
```

## 与其他模块的关系

| 模块 | 关系 |
|-----|------|
| embedding | 提供文本向量化能力 |
| db | 提供向量存储和搜索能力 |
| tool/builtin | 通过 knowledge_search 工具对外提供服务 |
| agent | 通过 KnowledgeManager 调用知识搜索 |

## 设计特点

1. **参数直接传递**：搜索参数由 Agent 直接传递，无需 LLM 语义理解
2. **时间过滤支持**：支持按时间范围过滤搜索结果
3. **分块策略可配置**：支持自定义分块大小和重叠长度
