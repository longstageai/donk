# Skill Auto-Discovery System 设计文档

## 1. 系统概述

技能自动发现系统是一个基于多 Agent 协作的自动化模块，用于从对话历史中分析用户需求，自动创建新的 Skill。

### 1.1 核心功能
- 周期性分析对话历史（每2小时执行一次）
- 自动提取潜在技能需求
- 检测并避免重复技能创建
- 无需求时自动生成创意技能
- 通过 WebSocket 推送创建通知

### 1.2 执行周期
- 使用 Scheduler 模块的 Cron 任务
- 配置: `0 */2 * * *` (每2小时执行一次)

---

## 2. 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                 Skill Auto-Discovery System                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Scheduler (每2小时触发)                      │   │
│  │              Cron: "0 */2 * * *"                         │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                    │
│                            ↓                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              DiscoveryExecutor (执行器)                   │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │   │
│  │  │Analyzer │→ │Planner  │→ │Creator  │→ │Notifier │    │   │
│  │  │ Agent   │  │ Agent   │  │ Agent   │  │         │    │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘    │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                    │
│         ┌──────────────────┼──────────────────┐                │
│         ↓                  ↓                  ↓                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ conversation │  │ skill_states │  │  websocket   │         │
│  │    Store     │  │    (DB)      │  │     Hub      │         │
│  │  (读取历史)   │  │ (查重/存储)   │  │   (推送通知)  │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 核心组件

### 3.1 Agent 职责

| Agent | 输入 | 输出 | 核心职责 |
|-------|------|------|---------|
| **Analyzer** | 对话历史列表 | 候选技能列表 | 识别用户需求，提取技能名称和描述 |
| **Creative** | 无/上下文 | 创意技能候选 | 无需求时生成有趣的技能创意 |
| **Planner** | 候选技能 | 技能规划 | 设计技能结构、指令、工具需求 |
| **Creator** | 技能规划 | Skill 文件 | 调用 skill.Builder 创建技能 |
| **Notifier** | 创建好的 Skill | WebSocket 消息 | 推送通知给前端 |

### 3.2 重复检测机制

```go
// 基于数据库的重复检测
1. 查询 skill_states 表获取所有现有技能
2. 名称精确匹配（不区分大小写）
3. 描述相似度检查（关键词重叠度 > 60%）
```

---

## 4. 执行流程

```
每2小时
  │
  ↓
┌────────────────────┐
│ 1. 获取对话历史     │◄── conversation.Store (最近2小时)
└─────────┬──────────┘
          ↓
┌────────────────────┐
│ 2. LLM 分析对话    │──► 提取潜在技能需求 [名称, 描述, 触发场景]
│    Analyzer Agent  │
└─────────┬──────────┘
          ↓
┌────────────────────┐     无需求时
│ 3. 有需求?         │────────────────┐
└─────────┬──────────┘                │
          │ 是                         │
          ↓                            ↓
┌────────────────────┐    ┌────────────────────┐
│ 4. 查重检查         │    │ 4'. 创意技能生成    │
│    skill_states表  │    │    Creative Agent  │
│    name+description│    └─────────┬──────────┘
└─────────┬──────────┘              │
     是/否│                         │
   ┌─────┴─────┐                    │
   ↓           ↓                    │
┌──────┐   ┌────────────────────┐   │
│ 跳过  │   │ 5. 规划技能结构     │◄──┘
│      │   │    Planner Agent   │
└──────┘   └─────────┬──────────┘
                     ↓
            ┌────────────────────┐
            │ 6. 创建技能文件     │──► skill.Builder
            │    Creator Agent   │    写入 ./data/skills/ 目录
            └─────────┬──────────┘
                      ↓
             ┌────────────────────┐
             │ 7. 推送 WebSocket   │──► websocket.Hub
             │    通知前端         │    skill_created 事件
             └────────────────────┘
```

---

## 5. 目录结构

```
internal/skilldiscovery/
├── types.go               # 类型定义
├── executor.go            # Scheduler Executor 实现
├── duplicate_checker.go   # 重复检测器
├── config.go              # 配置
├── initializer.go         # 模块初始化器
│
├── agents/                # Agent 实现
│   ├── analyzer.go        # 对话分析 Agent
│   ├── creative.go        # 创意生成 Agent
│   ├── planner.go         # 技能规划 Agent
│   ├── creator.go         # 技能创建 Agent
│   └── notifier.go        # 通知 Agent
│
└── prompts/               # Prompt 模板
    ├── analyzer.md
    ├── creative.md
    ├── planner.md

docs/
└── skilldiscovery_design.md  # 设计文档

./data/skills/             # 生成的技能存储目录
```

---

## 6. 集成方式

### 6.1 注册定时任务

```go
// 在系统初始化时调用
func RegisterDiscoveryJob(scheduler *scheduler.Scheduler, executor *skilldiscovery.Executor) error {
    task := &scheduler.Task{
        ID:        "skill-discovery",
        Name:      "技能自动发现",
        TaskType:  scheduler.TaskTypeCron,
        Schedule:  "0 */2 * * *",  // 每2小时
        Executor:  scheduler.ExecutorAgent,
        Config: scheduler.TaskConfig{
            "type": "skill_discovery",
        },
    }
    return scheduler.SaveTask(task)
}
```

### 6.2 Executor 工厂注册

```go
// 注册 DiscoveryExecutor 到 Scheduler
factory := &DiscoveryExecutorFactory{executor: executor}
scheduler.SetExecutorFactory(factory)
```

---

## 7. 配置项

```go
type Config struct {
    // 执行间隔（默认2小时）
    Interval time.Duration
    
    // 重复检测相似度阈值（默认0.6）
    SimilarityThreshold float64
    
    // 每次最大创建技能数（默认5）
    MaxSkillsPerRun int
    
    // 无需求时生成创意技能数量（默认1）
    CreativeSkillsCount int
    
    // 是否启用通知（默认true）
    EnableNotification bool
}
```

---

## 8. WebSocket 消息格式

```json
{
  "type": "skill_created",
  "data": {
    "name": "skill-name",
    "description": "技能描述",
    "created_at": "2024-01-01T12:00:00Z",
    "source": "auto_discovery"
  }
}
```

---

## 9. 日志规范

使用 `pkg/logger` 模块记录日志：

```go
logger.Info("技能自动发现任务开始", map[string]interface{}{
    "task_id": task.ID,
    "timestamp": time.Now(),
})

logger.Debug("分析对话历史", map[string]interface{}{
    "conversation_count": len(conversations),
})

logger.Info("创建新技能", map[string]interface{}{
    "skill_name": skill.Name(),
    "source": "analyzer", // 或 "creative"
})

logger.Warn("跳过重复技能", map[string]interface{}{
    "skill_name": candidate.Name,
    "reason": "名称已存在",
})
```
