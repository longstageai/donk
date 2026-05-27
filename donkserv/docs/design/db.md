# DB 模块架构

## 概述

DB 模块提供向量数据库的统一接口抽象，便于切换不同的向量数据库实现。当前默认使用 CortexDB。

## 目录结构

```
internal/db/
├── db.go       # VectorStore 接口定义
└── cortex.go   # CortexDB 实现
```

## 核心接口

```go
// VectorStore 向量存储接口
type VectorStore interface {
    Add(ctx context.Context, vector []float32, content string) (string, error)
    Search(ctx context.Context, vector []float32, limit int) ([]SearchResult, error)
    Close() error
}

// SearchResult 搜索结果
type SearchResult struct {
    ID       string
    Content  string
    Score    float32
}
```

## 设计模式

### 1. 接口抽象
- 定义 `VectorStore` 接口，统一向量存储操作
- 支持添加向量(`Add`)和相似度搜索(`Search`)

### 2. 依赖注入
- 上层模块(memory, knowledge)通过接口依赖 vector store
- 运行时注入具体实现，便于切换数据库

## 使用示例

```go
// 创建向量存储
store, err := db.NewCortexStore("./data/cortex", "memory")

// 添加向量
id, err := store.Add(ctx, vector, "用户对话内容")

// 相似度搜索
results, err := store.Search(ctx, queryVector, 5)

// 关闭连接
store.Close()
```

## 扩展新的向量数据库

1. 在 `db.go` 中定义新数据库的配置结构
2. 新增实现文件 `xxx.go`，实现 `VectorStore` 接口
3. 添加工厂函数 `NewXxxStore()`

```go
// db/xxx.go
type xxxStore struct { ... }

func NewXxxStore(config XxxConfig) (VectorStore, error) { ... }
func (s *xxxStore) Add(...) (string, error) { ... }
func (s *xxxStore) Search(...) ([]SearchResult, error) { ... }
func (s *xxxStore) Close() error { ... }
```

## 与其他模块的关系

```
┌─────────────┐
│   agent     │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│  knowledge  │────▶│     db      │
└─────────────┘     │  VectorStore │
       │            └──────┬──────┘
       │                   │
       ▼                   ▼
┌─────────────┐     ┌─────────────┐
│   memory    │────▶│   CortexDB   │
└─────────────┘     └─────────────┘
```

- knowledge 和 memory 模块依赖 VectorStore 接口
- db 模块提供具体实现(CortexDB)
- 便于未来切换到 Milvus、Qdrant、Weaviate 等
