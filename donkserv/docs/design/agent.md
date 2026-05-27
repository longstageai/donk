# Agent 模块架构

## 概述

Agent 模块是 AI Agent 的核心实现，整合模型、工具和记忆系统，实现 ReAct 模式的自主任务执行。

## 目录结构

```
internal/agent/
├── agent.go     # Agent 核心结构定义
├── run.go       # Agent 创建和运行
├── options.go   # 配置选项
├── context.go   # Agent 上下文
├── tool.go      # 工具调用相关
└── step.go      # 执行步骤
```

## 核心组件

### 1. Agent 核心结构

```go
type Agent struct {
    model           model.Adapter           // 模型适配器
    tools           *tool.Registry         // 工具注册表
    workingMemory   *memory.SessionMemory // 短期会话记忆
    longMemory      *memory.LongMemory    // 长期记忆
    historyStore    *memory.HistoryStore  // 历史记录
    profileManager  *profile.ProfileManager // 用户画像
    knowledgeManager *knowledge.KnowledgeManager // 知识库
    maxLoop         int                   // 最大循环次数
    timeout         time.Duration         // 超时时间
    onStream        StreamCallback        // 流式回调
}
```

### 2. StreamCallback 流式回调

```go
type StreamCallback func(chunk *schema.StreamChunk)
```

### 3. 执行选项

通过函数式选项模式配置 Agent：

```go
type Option func(*Agent)

// 常用选项
WithMaxLoop(n int)           // 最大循环次数
WithTimeout(d time.Duration) // 超时时间
WithStreamCallback(fn StreamCallback) // 流式回调
```

## ReAct 执行模式

Agent 采用 ReAct(Reasoning + Acting) 模式：

```
1. 用户输入
2. 思考 (Reason): 模型分析问题
3. 行动 (Act): 调用工具获取信息
4. 观察 (Observe): 获取工具返回结果
5. 重复步骤 2-4 直到完成任务
6. 返回最终回答
```

## 使用示例

```go
// 创建 Agent
agent := agent.New(
    modelAdapter,
    toolRegistry,
    longMemory,
    historyStore,
    agent.WithMaxLoop(10),
    agent.WithTimeout(5 * time.Minute),
    agent.WithStreamCallback(func(chunk *schema.StreamChunk) {
        // 处理流式输出
    }),
)

// 执行任务
response, err := agent.Run(ctx, "用户问题")
```

## 与其他模块的关系

```
┌─────────────────────────────────────────────────────┐
│                      Agent                           │
├─────────────────────────────────────────────────────┤
│  model    │  tool   │  memory  │ knowledge │ profile│
└─────┬─────┴────┬────┴────┬─────┴────┬──────┴────┬────┘
      │          │         │          │           │
      ▼          ▼         ▼          ▼           ▼
┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
│  LLM    │ │ 工具    │ │ 记忆    │ │ 知识库  │ │ 用户画像│
└─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘
```

## 核心流程

```
Run(ctx, prompt)
    │
    ▼
┌─────────────────────────────┐
│  构建系统消息 (System Prompt) │
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│   ReAct 循环 (maxLoop次)     │
│  ┌─────────────────────────┐ │
│  │ 1. 模型推理 + 决定工具   │ │
│  │ 2. 执行工具             │ │
│  │ 3. 获取结果             │ │
│  │ 4. 判断是否完成         │ │
│  └─────────────────────────┘ │
└─────────────────────────────┘
    │
    ▼
返回最终响应
```

## 设计特点

1. **依赖注入**：通过参数注入各模块依赖，便于测试
2. **选项模式**：函数式选项灵活配置
3. **ReAct 模式**：_reasoning + action_ 循环执行
4. **流式支持**：支持流式输出回调
5. **记忆整合**：整合短期、长期记忆和知识库
