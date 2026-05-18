# Memory 模块架构

## 概述

Memory 模块负责 AI Agent 的记忆管理，包括短期会话记忆、长期记忆和会话历史。

## 目录结构

```
internal/memory/
├── session.go   # 短期会话记忆
├── long.go      # 长期记忆（向量存储）
├── history.go   # 会话历史记录
└── entry.go     # 记忆条目定义
```

## 核心组件

### 1. SessionMemory（短期会话记忆）

管理当前会话的上下文信息，随会话结束而清空。

```go
type SessionMemory struct {
    messages   []Message       // 当前会话消息
    entities  map[string]any  // 实体信息
    context   string          // 上下文摘要
}
```

### 2. LongMemory（长期记忆）

持久化的向量记忆，支持语义检索。

```go
type LongMemory struct {
    embedder  embedding.Embedder  // 向量化器
    store     db.VectorStore      // 向量数据库
}
```

### 3. HistoryStore（会话历史）

管理历史会话记录，支持查询和统计。

```go
type HistoryStore struct {
    db        *bolt.DB
    sessions  map[string]*Session
}
```

### 4. Entry（记忆条目）

记忆的基本单元。

```go
type Entry struct {
    ID        string
    Type      EntryType  // message, fact, preference
    Content   string
    Timestamp time.Time
    Metadata  map[string]any
}
```

## 记忆类型

| 类型 | 描述 | 持久化 |
|-----|------|-------|
| SessionMemory | 当前会话上下文 | 否 |
| LongMemory | 长期知识/偏好 | 是(向量) |
| HistoryStore | 历史会话记录 | 是(文本) |

## 使用示例

```go
// 创建会话记忆
session := memory.NewSessionMemory()
session.AddMessage(role, content)

// 添加到长期记忆
err := longMem.Add(ctx, "用户偏好深海蓝")

// 搜索长期记忆
results, err := longMem.Search(ctx, "用户喜欢什么颜色", 5)

// 添加到历史记录
history.AddSession(sessionID, messages)

// 查询历史
history.GetSession(sessionID)
```

## 与其他模块的关系

```
┌─────────────┐
│   agent     │
└──────┬──────┘
       │
       ├──────────────────┬──────────────────┐
       ▼                  ▼                  ▼
┌─────────────┐   ┌─────────────┐    ┌─────────────┐
│  Session    │   │    Long     │    │   History   │
│  Memory     │   │   Memory    │    │   Store     │
└─────────────┘   └──────┬──────┘    └─────────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │  VectorStore│
                  │   (db)      │
                  └─────────────┘
```

## 设计特点

1. **分层记忆**：短期/长期/历史，三层分离
2. **向量检索**：长期记忆支持语义相似度搜索
3. **接口抽象**：依赖 embedding 和 db 模块，便于扩展
