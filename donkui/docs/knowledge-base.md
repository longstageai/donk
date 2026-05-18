# 知识库模块设计与使用文档

## 概述

知识库模块是一个独立的文档索引系统，用于自动扫描、处理和索引用户文档，构建可搜索的向量知识库。

## 核心特性

- **定时任务**：程序启动后自动启动定时器，默认每2-3小时执行一次
- **配置控制**：通过数据库配置控制是否处理文档（定时器始终运行）
- **文档支持**：支持 txt、md、pdf、docx 格式
- **智能处理**：包含去重、优先级队列、冷热数据分离
- **低资源占用**：单线程处理，可配置处理间隔

## 架构设计

### 模块结构

```
internal/knowledge/
├── config.go         # 配置结构定义
├── model.go          # 数据模型
├── scanner.go        # 文件扫描器（支持深度限制）
├── priority.go       # 优先级队列
├── store.go          # 存储接口
├── sqlite_store.go   # SQLite 元数据存储
├── vector_store.go   # 向量存储
├── indexer.go        # 文档索引器
├── builder.go        # 知识库构建器
├── runner.go         # 定时任务运行器
├── embedder.go       # Embedder 集成
├── initializer.go    # 初始化器
└── knowledge_test.go # 测试用例
```

### 数据流

```
文件扫描 → 去重检查 → 优先级排序 → 文档解析 → 向量嵌入 → 存储索引
    ↑                                                              ↓
    └──────────────────── 定时触发（2-3小时）──────────────────────┘
```

### 核心组件

#### 1. Scanner（扫描器）
- 扫描 Windows 用户目录（桌面、下载、文档）
- 支持目录深度限制（默认3层）
- 文件类型过滤（txt、md、pdf、docx）

#### 2. Priority Queue（优先级队列）
基于以下因素计算优先级：
- **时间分**：越新的文件优先级越高
- **热度分**：访问次数越多优先级越高
- **大小分**：小文件优先处理
- **扩展名分**：txt/md 优先于 pdf/docx

#### 3. Indexer（索引器）
- 文档内容提取和解析
- 内容去重（MD5哈希）
- 向量嵌入生成
- 元数据存储

#### 4. Runner（运行器）
- 定时任务调度
- 配置动态检查
- 资源控制（CPU占用限制）

## 配置说明

### 知识库配置（Config）

```go
type Config struct {
    Enabled     bool     // 是否启用（控制是否处理文档）
    Interval    int      // 扫描间隔（秒），默认3600（1小时）
    BatchSize   int      // 每批处理数量，默认50
    SleepMs     int      // 处理间隔（毫秒），默认100
    MaxDepth    int      // 最大扫描深度，默认3
    MaxFileSize int64    // 最大文件大小（字节），默认10MB
    HotDays     int      // 热数据天数，默认7
    WarmDays    int      // 温数据天数，默认30
    Directories []string // 扫描目录（空则使用默认）
}
```

### 默认配置

```go
func DefaultConfig() *Config {
    return &Config{
        Enabled:     true,              // 默认启用
        Interval:    3600,              // 每小时扫描一次
        BatchSize:   50,                // 每批50个文件
        SleepMs:     100,               // 100ms间隔
        MaxDepth:    3,                 // 3层深度
        MaxFileSize: 10 * 1024 * 1024,  // 10MB
        HotDays:     7,                 // 7天热数据
        WarmDays:    30,                // 30天温数据
        Directories: []string{},        // 使用默认目录
    }
}
```

## 使用方式

### 1. 程序集成

在 `main.go` 中添加：

```go
package main

import (
    "github.com/longstageai/donk/donk/internal/knowledge"
    "github.com/longstageai/donk/donk/internal/setting"
)

func main() {
    // 打开数据库连接
    db, err := sql.Open("sqlite3", "./data/donk.db")
    if err != nil {
        log.Fatal(err)
    }
    
    // 初始化并启动知识库模块
    // 程序启动时自动启动定时器
    knowledgeInitializer, err := knowledge.InitAndStart(db)
    if err != nil {
        log.Printf("初始化知识库失败: %v\n", err)
    } else if knowledgeInitializer != nil {
        // 注入到 setting 模块，供 HTTP API 使用
        setting.SetKnowledgeController(knowledgeInitializer)
    }
    
    // ... 其他初始化代码 ...
    
    // 注册优雅关闭
    if knowledgeInitializer != nil {
        app.RegisterTaskFunc("knowledge", func(ctx context.Context, app *appctx.Application) error {
            <-ctx.Done()
            knowledgeInitializer.Stop()
            return nil
        }, 0)
    }
}
```

### 2. HTTP API 控制

#### 获取知识库配置

```http
GET /api/v1/config/knowledge
```

响应：
```json
{
    "enabled": true
}
```

#### 更新知识库配置

```http
PUT /api/v1/config/knowledge
Content-Type: application/json

{
    "enabled": false
}
```

响应：
```json
{
    "message": "知识库配置更新成功",
    "enabled": false,
    "note": "配置仅控制是否处理文档，定时器始终运行"
}
```

#### 获取知识库状态

```http
GET /api/v1/knowledge/status
```

响应：
```json
{
    "enabled": true,
    "running": true,
    "last_error": ""
}
```

### 3. 手动触发构建

如需立即执行一次知识库构建（不等待定时器）：

```go
// 获取 Runner 实例
runner := knowledgeInitializer.GetRunner()

// 手动触发一次构建
// 注意：需要自行实现手动触发接口
```

## 数据库表结构

### config 表

```sql
CREATE TABLE IF NOT EXISTS config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- ... 其他配置字段 ...
    knowledge_enabled INTEGER NOT NULL DEFAULT 1,  -- 知识库是否启用
    -- ... 时间戳字段 ...
);
```

### kb_documents 表（知识库文档）

```sql
CREATE TABLE IF NOT EXISTS kb_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content_hash TEXT UNIQUE,        -- 内容哈希（去重用）
    content TEXT,                    -- 文档内容
    vector_id TEXT,                  -- 向量ID
    file_path TEXT UNIQUE,           -- 文件路径
    file_size INTEGER,               -- 文件大小
    modified_time TIMESTAMP,         -- 修改时间
    created_at TIMESTAMP,            -- 创建时间
    updated_at TIMESTAMP,            -- 更新时间
    access_count INTEGER DEFAULT 0,  -- 访问次数
    last_access_time TIMESTAMP,      -- 最后访问时间
    status TEXT DEFAULT 'pending',   -- 状态
    error_msg TEXT                   -- 错误信息
);
```

## 工作流程

### 1. 程序启动流程

```
1. 打开数据库连接
2. 调用 knowledge.InitAndStart(db)
   - 创建 Runner 实例
   - 启动定时器（立即执行一次，然后按间隔执行）
3. 注入 KnowledgeController 到 setting 模块
4. 注册优雅关闭钩子
```

### 2. 定时任务执行流程

```
定时触发
    ↓
检查数据库配置 knowledge_enabled
    ↓
    ├── false → 记录日志，跳过本次执行
    └── true  → 继续执行
                    ↓
            扫描文件目录
                    ↓
            筛选新文件（基于哈希去重）
                    ↓
            构建优先级队列
                    ↓
            按优先级处理文档
                    ↓
            解析内容 → 生成向量 → 存储索引
                    ↓
            更新元数据
```

### 3. 配置变更流程

```
用户调用 PUT /api/v1/config/knowledge
    ↓
更新数据库配置
    ↓
定时任务下次执行时读取新配置
    ↓
根据配置决定是否处理文档
```

## 性能优化

### 1. 去重机制
- 使用 MD5 哈希计算文件内容指纹
- 已处理过的文件不会重复处理
- 修改过的文件会重新处理（基于修改时间）

### 2. 优先级队列
- 新文件优先处理
- 小文件优先处理（快速完成）
- 文本文件优先于二进制文件

### 3. 资源控制
- 单线程处理，避免CPU占用过高
- 可配置处理间隔（SleepMs）
- 批量处理，减少数据库操作

### 4. 冷热数据分离
- **热数据**：最近7天访问的文档
- **温数据**：最近30天访问的文档
- **冷数据**：超过30天未访问的文档

## 错误处理

### 常见错误

1. **Embedder 配置错误**
   - 检查 setting 模块中的 embedding 配置
   - 确保 provider、model、api_key 已正确配置

2. **文件解析错误**
   - 不支持的文件格式会被跳过
   - 损坏的文件会记录错误并跳过

3. **数据库错误**
   - 自动重试机制
   - 错误日志记录

### 日志级别

- **INFO**：正常流程日志
- **WARN**：警告信息（如配置未启用）
- **ERROR**：错误信息（如处理失败）
- **DEBUG**：调试信息（如扫描文件列表）

## 扩展开发

### 添加新的文档格式支持

1. 在 `indexer.go` 中添加解析函数：

```go
func (idx *Indexer) readXxxFile(filePath string) (string, error) {
    // 实现文档解析逻辑
    return content, nil
}
```

2. 在 `readFileContent` 函数中添加文件类型判断：

```go
case ".xxx":
    return idx.readXxxFile(filePath)
```

3. 在 `scanner.go` 中添加文件扩展名支持：

```go
extensions: map[string]bool{
    // ... 其他扩展名 ...
    ".xxx": true,
}
```

### 自定义扫描目录

```go
config := &knowledge.Config{
    Directories: []string{
        "C:\\Users\\Username\\Documents",
        "C:\\Custom\\Path",
    },
}
runner, err := knowledge.NewRunnerWithDB(dataDir, db, config)
```

## 测试

运行知识库模块测试：

```bash
go test ./internal/knowledge/... -v
```

测试覆盖：
- 文件扫描
- 优先级队列
- 数据库存储
- 文档解析（txt、md、pdf、docx）
- 配置验证

## 注意事项

1. **定时器始终运行**：程序启动后定时器会自动启动，无法通过 API 停止
2. **配置控制处理**：只有 `knowledge_enabled = true` 时才会实际处理文档
3. **首次启动**：首次启动时会立即执行一次扫描，然后按间隔执行
4. **资源占用**：处理大量文档时可能占用较多 CPU 和内存，建议调整 `SleepMs` 和 `BatchSize`
5. **向量存储**：文档向量存储在 `./data/knowledge/vectors` 目录

## 相关文档

- [Setting 模块文档](./setting_api.md)
- [Embedding 模块文档](./embedding.md)
- [Vector Store 文档](./vector_store.md)
