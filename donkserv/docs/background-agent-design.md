# 后台Agent系统设计文档

## 一、概述

后台Agent系统是一个独立的、持续运行的后台服务，支持根据配置的主题自动循环执行Agent任务，并通过WebSocket推送执行结果。

### 核心特性

- **配置文件驱动**：通过 `background.yaml` 配置，无需HTTP API管理
- **完全独立**：后台Agent与原有Agent系统隔离，互不影响
- **每次新建Agent**：每次执行任务都创建全新的Agent实例
- **动态获取配置**：每次创建Agent时读取原有config系统的最新配置
- **Token上限校验**：复用原有Token统计模块进行预算检查
- **WebSocket推送**：任务执行完成后主动推送结果

## 二、架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application                              │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   Background Service                     │    │
│  │                                                          │    │
│  │  ┌─────────────┐    ┌─────────────┐    ┌────────────┐   │    │
│  │  │  Manager    │───→│   Runner    │───→│  Execute   │   │    │
│  │  │  (管理多个)  │    │  (循环执行)  │    │  (单次任务) │   │    │
│  │  └──────┬──────┘    └─────────────┘    └────────────┘   │    │
│  │         │                                                │    │
│  │         └────────────────────────────────────────────┐   │    │
│  │                                                      │   │    │
│  │  ┌──────────────────────────────────────────────────┐│   │    │
│  │  │         IndependentAgentBuilder                   ││   │    │
│  │  │  (每次新建Agent，读取最新配置)                     ││   │    │
│  │  │                                                   ││   │    │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌───────────┐ ││   │    │
│  │  │  │ 读取LLM配置  │  │ 读取Token预算 │  │ 创建Agent  │ ││   │    │
│  │  │  │ (原有系统)   │  │ (原有系统)   │  │ (新实例)   │ ││   │    │
│  │  │  └─────────────┘  └─────────────┘  └───────────┘ ││   │    │
│  │  └──────────────────────────────────────────────────┘│   │    │
│  │                           │                          │   │    │
│  │                           ▼                          │   │    │
│  │                  ┌─────────────────┐                 │   │    │
│  │                  │  WebSocket Push │─────────────────┼───┘    │
│  │                  │  (推送结果)      │                 │        │
│  │                  └─────────────────┘                 │        │
│  └──────────────────────────────────────────────────────┘        │
│                              │                                   │
└──────────────────────────────┼───────────────────────────────────┘
                               │
┌──────────────────────────────┼───────────────────────────────────┐
│                              ▼                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐ │
│  │ background.yaml  │  │ 原有config系统    │  │ WebSocket Hub  │ │
│  │ (后台Agent配置)   │  │ (LLM/Token等)    │  │ (推送消息)     │ │
│  └──────────────────┘  └──────────────────┘  └────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## 三、配置文件

### 3.1 background.yaml

```yaml
# conf/background.yaml - 后台Agent配置文件

# 全局配置
global:
  enabled: true              # 总开关
  default_interval: 300      # 默认执行间隔(秒)

# 后台Agent列表
agents:
  - id: "code-reviewer"           # 唯一标识
    name: "代码审查助手"           # 显示名称
    enabled: true                 # 是否启用
    interval: 600                 # 执行间隔(秒)，覆盖全局配置
    timeout: 120                  # 单次任务超时(秒)
    max_iterations: 15            # 最大迭代次数
    system_prompt: |              # 系统提示词
      你是一个专业的代码审查助手。
      你的任务是自动检查代码仓库中的潜在问题。
      
      请重点关注：
      1. 代码规范问题
      2. 潜在的安全漏洞
      3. 性能优化建议
      4. 可维护性问题
      
      每次审查后，输出简洁的审查报告。
    
    # 允许使用的工具（空数组表示可以使用所有工具）
    allowed_tools:
      - file_reader
      - file_writer
      - http
    
    # 自定义变量（在prompt中可用）
    variables:
      target_path: "./src"
      exclude_patterns: ["*.test.js", "*.spec.go"]

  - id: "security-scanner"
    name: "安全扫描助手"
    enabled: true
    interval: 1800                # 30分钟
    timeout: 300
    max_iterations: 20
    system_prompt: |
      你是一个安全扫描助手，负责检测系统中的安全隐患。
      
      扫描范围：
      - 依赖漏洞检查
      - 敏感信息泄露
      - 不安全的配置
      
      发现高危问题时立即报告。

  - id: "doc-generator"
    name: "文档生成助手"
    enabled: false                # 禁用
    interval: 3600
    system_prompt: |
      你是一个文档生成助手，负责自动生成项目文档。
```

### 3.2 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| global.enabled | bool | 否 | 总开关，默认true |
| global.default_interval | int | 否 | 默认执行间隔(秒)，默认300 |
| agents | array | 是 | Agent配置列表 |
| agents[].id | string | 是 | 唯一标识 |
| agents[].name | string | 是 | 显示名称 |
| agents[].enabled | bool | 否 | 是否启用，默认true |
| agents[].interval | int | 否 | 执行间隔(秒)，默认使用global.default_interval |
| agents[].timeout | int | 否 | 单次任务超时(秒)，默认60 |
| agents[].max_iterations | int | 否 | 最大迭代次数，默认10 |
| agents[].system_prompt | string | 是 | 系统提示词 |
| agents[].allowed_tools | array | 否 | 允许使用的工具列表，空数组表示允许所有 |
| agents[].variables | map | 否 | 自定义变量 |

## 四、核心组件

### 4.1 IndependentAgentBuilder（独立Agent构建器）

完全独立的Agent构建器，每次调用Build都创建全新的Agent实例。

**职责**：
- 从原有setting系统读取最新LLM配置
- 检查Token预算
- 创建新的LLM适配器
- 创建新的工具注册表
- 创建新的Agent实例

**关键特性**：
- 不依赖 `cmd/agent.go` 中的AgentBuilder
- 每次调用都重新初始化所有组件
- 使用 `agent.New()` 创建实例
- 支持通过allowed_tools限制可用工具

### 4.2 BackgroundRunner（运行器）

负责按配置循环执行单个Agent任务。

**执行流程**：
1. 检查启用状态
2. 调用IndependentAgentBuilder.Build()创建Agent（使用最新配置）
3. 执行Agent.Run()运行任务
4. 立即释放Agent实例（置nil）
5. 更新执行统计
6. 通过WebSocket推送结果
7. 等待interval后循环

### 4.3 BackgroundManager（管理器）

管理所有Runner的生命周期。

**职责**：
- 加载background.yaml配置
- 为每个启用的Agent创建Runner
- 启动所有Runner
- 优雅关闭时停止所有Runner
- 提供统计信息查询

## 五、配置读取关系

```
┌─────────────────────────────────────────────────────────┐
│                    后台Agent执行流程                      │
└─────────────────────────────────────────────────────────┘

background.yaml 配置
        │
        ▼
┌─────────────────┐
│ 读取Agent配置    │──────→ SystemPrompt
│ (prompt/interval)│       Timeout
└─────────────────┘       MaxIterations
        │                 AllowedTools
        ▼
┌─────────────────┐
│ 调用原有Config   │
│  (setting模块)   │
└─────────────────┘
        │
        ├──→ LLM配置 (Provider/Model/APIKey)
        ├──→ Token预算 (DailyTokenLimit)
        └──→ Agent配置 (MaxLoop等)
        │
        ▼
┌─────────────────┐
│ 创建新Agent实例  │
│ (Independent    │
│  AgentBuilder)  │
└─────────────────┘
        │
        ▼
    执行任务
```

## 六、WebSocket消息格式

### 6.1 任务完成消息（成功）

```json
{
  "type": "background_task_complete",
  "runner_id": "code-reviewer",
  "runner_name": "代码审查助手",
  "status": "success",
  "output": "审查结果...",
  "duration": 5230,
  "tokens": 1250,
  "timestamp": 1704067200
}
```

### 6.2 任务错误消息（失败）

```json
{
  "type": "background_task_error",
  "runner_id": "code-reviewer",
  "runner_name": "代码审查助手",
  "status": "failed",
  "error_type": "agent_build_failed",
  "error": "获取LLM配置失败",
  "duration": 150,
  "timestamp": 1704067200
}
```

### 6.3 消息字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 消息类型：background_task_complete / background_task_error |
| runner_id | string | Runner唯一标识 |
| runner_name | string | Runner显示名称 |
| status | string | 状态：success / failed |
| output | string | 执行输出（成功时） |
| error_type | string | 错误类型（失败时） |
| error | string | 错误信息（失败时） |
| duration | int64 | 执行耗时（毫秒） |
| tokens | int | Token消耗 |
| timestamp | int64 | 时间戳（秒） |

## 七、目录结构

```
conf/
├── config.yaml           # 原有：LLM/Token/Embedding配置
└── background.yaml       # 新增：后台Agent配置

internal/
├── background/           # 新增：后台Agent模块
│   ├── config.go         # 配置结构定义
│   ├── builder.go        # IndependentAgentBuilder
│   ├── runner.go         # BackgroundRunner
│   ├── manager.go        # BackgroundManager
│   └── message.go        # WebSocket消息
├── agent/                # 原有：Agent核心（不变）
├── setting/              # 原有：配置管理（复用）
├── token/                # 原有：Token统计（复用）
└── websocket/            # 原有：WebSocket（复用）

cmd/
├── donk.go              # 修改：集成后台Agent启动
└── background.go         # 新增：后台Agent初始化
```

## 八、集成方式

### 8.1 初始化（cmd/background.go）

```go
func SetupBackgroundService(app *appctx.Application, db *sql.DB, wsServer *websocket.Server) (*background.Manager, error) {
    // 1. 加载配置文件
    config, err := background.LoadConfig("conf/background.yaml")
    if err != nil {
        logger.Warn("后台Agent配置文件加载失败，服务将不启动", map[string]interface{}{
            "error": err.Error(),
        })
        return nil, nil
    }
    
    // 2. 创建管理器
    manager := background.NewManager(config, db.DB, wsServer.Hub())
    
    // 3. 启动
    go manager.Start()
    
    return manager, nil
}
```

### 8.2 主程序集成（cmd/aclaw.go）

```go
func main() {
    // ... 原有初始化 ...
    
    // 创建WebSocket服务器
    wsServer, err := SetupWebSocket(app, engine)
    // ...
    
    // 创建并启动后台Agent服务
    bgManager, err := SetupBackgroundService(app, db, wsServer)
    // ...
    
    // 注册优雅关闭
    if bgManager != nil {
        app.RegisterTaskFunc("background", func(ctx context.Context, application *appctx.Application) error {
            <-ctx.Done()
            bgManager.Stop()
            return nil
        }, 0)
    }
    
    // ... 启动应用 ...
}
```

## 九、与原有系统的关系

| 组件 | 后台Agent使用方式 | 是否影响原有系统 |
|------|------------------|-----------------|
| `internal/agent.Agent` | 直接调用 `agent.New()` | ❌ 不影响 |
| `internal/agent.Option` | 直接使用 | ❌ 不影响 |
| `cmd/agent.go` AgentBuilder | **不使用** | ❌ 完全不依赖 |
| `internal/setting` | 调用GetProvider()获取配置 | ⚠️ 只读配置 |
| `internal/token` | 调用统计接口检查预算 | ⚠️ 只读/写入统计 |
| `internal/websocket` | 调用Hub.BroadcastJSON() | ⚠️ 只调用发送接口 |
| `internal/model` | 独立创建适配器 | ❌ 独立实例 |

## 十、关键设计决策

### 10.1 为什么每次新建Agent？

1. **配置动态更新**：每次执行前读取最新配置
2. **Token预算检查**：每次执行前检查预算，避免超限
3. **资源隔离**：避免长时间运行导致的内存泄漏
4. **状态隔离**：每次执行都是干净的状态

### 10.2 为什么不提供HTTP API？

1. **简化设计**：配置文件足够表达配置意图
2. **减少攻击面**：无需暴露管理接口
3. **配置即代码**：便于版本控制和代码审查
4. **重启生效**：修改配置后重启服务即可

### 10.3 为什么分离background.yaml？

1. **关注点分离**：后台Agent配置与系统配置分离
2. **独立演进**：后台Agent配置可以独立修改
3. **可选组件**：没有配置文件时服务不启动，不影响主功能

## 十一、扩展建议

1. **热重载**：支持配置文件修改后自动重载（无需重启）
2. **执行历史**：记录每次执行的历史，支持查询
3. **告警机制**：连续失败时发送告警通知
4. **动态调整**：支持通过信号量动态调整执行间隔
5. **任务依赖**：支持配置Agent之间的依赖关系

---

**设计日期**: 2026-05-07  
**版本**: v1.0
