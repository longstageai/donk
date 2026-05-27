# Tool 模块架构

## 概述

Tool 模块是 Agent 的工具系统，提供工具定义、注册、执行和中间件扩展能力。

## 目录结构

```
internal/tool/
├── tool.go        # 核心: Tool, BaseTool, Schema, ToolError
├── context.go    # 执行上下文: Context, Status, Chunk
├── registry.go   # 工具注册表
├── result.go     # 执行结果
├── builder/      # 工具构建器
├── builtin/      # 内置工具
│   ├── calculator.go
│   ├── file_reader.go
│   ├── file_writer.go
│   ├── http.go
│   ├── knowledge_search.go
│   ├── memory_save.go
│   └── memory_search.go
└── middleware/   # 中间件
    ├── log.go
    ├── middleware.go
    ├── retry.go
    └── timeout.go
```

## 核心组件

### 1. Tool 接口

所有工具必须实现的接口。

```go
type Tool interface {
    Name() string
    Description() string
    Version() string
    Category() string
    Parameters() *Schema
    Execute(ctx *Context) (*Result, error)
}
```

### 2. BaseTool

工具的基础实现，提供默认方法。

```go
type BaseTool struct {
    name, description, version, category string
    parameters *Schema
    handler Handler
    timeout  time.Duration
}
```

### 3. Schema 参数定义

工具参数的 JSON Schema 定义。

```go
type Schema struct {
    Type       string
    Properties map[string]*Property
    Required   []string
}

type Property struct {
    Type, Description string
    Default interface{}
    Enum    []interface{}
    Format  string
    Minimum, Maximum *float64
    MinLength, MaxLength *int
}
```

### 4. Context 执行上下文

工具执行时的运行时环境。

```go
type Context struct {
    RequestID  string
    ToolName   string
    Params     map[string]any
    Metadata   map[string]any
    Timeout    time.Duration
    Status     Status
    OutputChan chan *Chunk
}
```

### 5. Registry 工具注册表

管理和调用工具的注册中心。

```go
type Registry struct {
    tools   map[string]Tool
    mutex   sync.RWMutex
}
```

### 6. Middleware 中间件

扩展工具执行行为。

```go
type Middleware func(next Handler) Handler
```

内置中间件：
- **log**: 日志记录
- **retry**: 自动重试
- **timeout**: 超时控制

## 工具分类

| 分类 | 说明 |
|-----|------|
| search | 搜索类工具 |
| data | 数据处理类工具 |
| utility | 通用工具类 |
| file | 文件操作类 |
| network | 网络请求类 |
| compute | 计算类工具 |

## 使用示例

```go
// 创建工具
myTool := tool.NewBaseTool("my_tool", "工具描述")
    .SetCategory(tool.CategoryUtility)
    .SetHandler(func(ctx *tool.Context) (*tool.Result, error) {
        // 处理逻辑
        return tool.NewResult("success", result), nil
    })

// 注册工具
registry := tool.NewRegistry()
registry.Register(myTool)

// 调用工具
result, err := registry.Execute(ctx, "my_tool", params)
```

## 内置工具

| 工具 | 功能 |
|-----|------|
| calculator | 数学计算 |
| file_reader | 读取文件 |
| file_writer | 写入文件 |
| http | HTTP 请求 |
| knowledge_search | 知识库搜索 |
| memory_save | 保存记忆 |
| memory_search | 搜索记忆 |

## 设计特点

1. **接口驱动**：Tool 接口定义清晰，易于扩展
2. **中间件系统**：支持日志、重试、超时等横切关注点
3. **构建器模式**：BaseTool 提供流式 API
4. **统一错误处理**：ToolError 结构化错误
