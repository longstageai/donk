# 多Agent协作架构设计文档

## 一、架构概述

基于"让用户感受到温暖"这一核心目标，构建6个Agent协作系统。

### 1.1 核心设计原则

- **简化优先**：Agent间直接传递Task对象，无消息队列
- **单存储**：统一内存/Redis存储，无复杂持久化
- **循环驱动**：事件触发循环，完成即开始下一轮
- **两级审查**：规划审查 + 成果审查，确保质量
- **主题可配置**：核心主题支持动态替换
- **Token管控**：实时统计消耗，超限自动停止
- **工具动态注册**：优雅的LLM工具注册机制
- **个性化输出**：基于用户画像生成差异化内容
- **统一预算**：单Agent与多Agent共享Token预算

### 1.2 架构图

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ 任务生成  │────▶│ 任务规划  │◀───▶│ 规划审查  │
│  Agent   │     │  Agent   │     │  Agent   │
└──────────┘     └────┬─────┘     └──────────┘
                      │
                      ▼ (>=8分)
                 ┌──────────┐
                 │ 任务执行  │◀──────┐
                 │  Agent   │       │
                 └────┬─────┘       │
                      │              │
                      ▼              │
                 ┌──────────┐       │
                 │ 任务审查  │───────┘ (<8分)
                 │  Agent   │
                 └────┬─────┘
                      │
                      ▼ (>=8分)
                 ┌──────────┐
                 │ 任务结束  │
                 │  Agent   │
                 └──────────┘
```

### 1.3 文件结构

```
cmd/
├── donk.go          # 主入口
├── agent.go          # 单Agent服务创建
├── multiagent.go     # 多Agent服务创建 (新增)
├── http.go           # HTTP服务器
└── websocket.go      # WebSocket服务器

internal/multiagent/
├── orchestrator.go   # 编排器，协调各Agent执行
├── service.go        # 服务入口，封装编排器
├── agents/           # 6个Agent实现
│   ├── generation.go    # 任务生成Agent
│   ├── planning.go     # 任务规划Agent
│   ├── plan_review.go  # 规划审查Agent
│   ├── execution.go    # 任务执行Agent
│   ├── task_review.go  # 任务审查Agent
│   ├── completion.go   # 任务结束Agent
│   └── logger.go       # LLM调用日志记录
├── tools/
│   └── registry.go     # 工具注册表
├── prompts/
│   └── prompts.go       # Agent提示词模板
└── token/
    └── manager.go       # 多Agent专用Token管理器 (legacy)
```

### 1.4 完整流程

```
启动
  │
  ▼
┌──────────┐
│ 任务生成  │◀──────────────────────────────┐
│  Agent   │  （持续循环：完成→再次生成）      │
└────┬─────┘                                 │
     │                                        │
     ▼                                        │
┌──────────┐    ┌──────────┐                │
│ 任务规划  │───▶│ 规划审查  │                │
│  Agent   │    │  Agent   │                │
└────┬─────┘    └────┬─────┘                │
     │               │                       │
     │◀──────────────┘ (score<8, 带feedback) │
     │                                        │
     └──────────────▶ (score>=8)             │
                         │                   │
                         ▼                   │
                    ┌──────────┐             │
                    │ 任务执行  │             │
                    │  Agent   │             │
                    └────┬─────┘             │
                         │                   │
                         ▼                   │
                    ┌──────────┐             │
                    │ 任务审查  │             │
                    │  Agent   │             │
                    └────┬─────┘             │
                         │                   │
        ┌────────────────┤                   │
        │                │                   │
        ▼                ▼                   │
   (score<8)        (score>=8)               │
        │                │                   │
        │                ▼                   │
        │           ┌──────────┐             │
        │           │ 任务结束  │             │
        │           │  Agent   │             │
        │           └────┬─────┘             │
        │                │                   │
        └────────────────┘                   │
                                             │
                         └───────────────────┘
                              (任务完成，触发新一轮)
```

---

## 二、核心改进

### 2.1 统一Token预算管理

#### 设计目标
单Agent与多Agent系统共享同一Token预算，任何一方超限都会触发停止。

#### 实现方案

**1. TokenStats（internal/token/stats.go）**
```go
type TokenStats struct {
    mu             sync.RWMutex  // 读写锁
    dir            string        // 存储目录
    dailyLimit     int           // 每日Token限额
    recentRecords  []TokenRecord // 内存缓存
    limitExceeded  bool          // 是否已超限
}
```

**2. 超限检查流程**
```
LLM调用 → RecordSimple()记录 → IsBudgetExceeded()检查 → 超限则停止
```

**3. 各Agent调用后检查**
```go
// 每个Agent的LLM调用后立即检查
a.tokenStats.RecordSimple(promptTokens, completionTokens, "agentName")
if a.tokenStats.IsBudgetExceeded() {
    return fmt.Errorf("Token预算已超出限额")
}
```

**4. 配置参数**
```yaml
agent:
  daily_token_limit: 1000000  # 100万Token每日限额
  # -1: 不限制但记录
  # 0: 不记录不限制
  # >0: 限制且记录
```

#### 使用方式
```go
// 创建统一TokenStats（单/多Agent共享）
tokenStats, _ := token.NewTokenStats("./data/token", dailyLimit)

// 单Agent使用
agent, _ := agent.NewAgent(..., WithTokenStats(tokenStats))

// 多Agent使用
service, _ := multiagent.NewServiceWithTokenStats(conf, theme, log, tokenStats)
```

---

### 2.2 LLM调用日志

#### 设计目标
记录每个Agent调用LLM的输入输出参数，便于调试和问题排查。

#### 实现方案

**Logger结构（agents/logger.go）**
```go
type AgentLogger struct {
    logger *logger.Logger
}

func (l *AgentLogger) LogLLMInput(agentName string, messages []types.Message, tools []types.ToolDefinition)
func (l *AgentLogger) LogLLMOutput(agentName string, resp *types.LLMResponse)
```

**日志内容**
- 输入：messages内容、tools列表
- 输出：content、reasoning、tool_calls数量、token消耗

---

### 2.3 用户画像处理

#### 设计目标
当用户画像不存在时，系统能优雅降级，使用通用策略。

#### 实现方案

**工具返回found标志**
```go
// get_user_profile 工具返回
if userID == "" || userID == "unknown" {
    return map[string]interface{}{
        "found":   false,
        "message": "未找到该用户的画像信息",
    }, nil
}
```

**ExecutionAgent处理**
```go
func (a *ExecutionAgent) handleUserProfileResult(toolResult string) {
    // 检查found标志
    if !found {
        // 使用通用策略
        a.agentLogger.Info("未找到用户画像，将使用通用策略", nil)
        return
    }
    // 解析用户画像信息...
}
```

---

### 2.4 工具注册机制

#### 设计目标
提供优雅的LLM工具注册方式，支持动态添加、删除工具。

#### 内置工具清单

| 工具名称 | 功能描述 | 使用场景 |
|---------|---------|---------|
| `get_user_profile` | 获取用户基本信息和偏好 | 个性化内容生成 |
| `get_zodiac` | 根据日期计算星座 | 生日贺卡、星座运势 |
| `generate_blessing` | 生成个性化祝福语 | 节日问候、生日祝福 |
| `generate_image` | 生成图片内容 | 贺卡、海报 |
| `search_content` | 搜索相关内容 | 获取实时信息 |

**注意**：`send_message` 工具已移除，消息发送发生在任务完成结束后。

---

## 三、统一数据结构

### 3.1 Task对象

```json
{
  "taskId": "task_001",
  "coreTheme": "让用户感受到温暖",
  "status": "CREATED",

  "task": {
    "theme": "birthday_card",
    "title": "专属生日贺卡",
    "description": "为用户制作一张融入个人喜好的生日贺卡",
    "coreThemeReason": "在生日这个特殊日子，让用户感到被重视和惦记",
    "coreElements": ["被记住", "个性化", "仪式感"]
  },

  "plan": [
    {
      "step": 1,
      "action": "get_user_profile",
      "description": "获取用户基本信息和喜好",
      "tool": "get_user_profile",
      "dependencies": []
    }
  ],

  "planReview": {
    "score": 8.5,
    "passed": true,
    "feedback": "",
    "attempt": 1
  },

  "todos": [
    {
      "step": 1,
      "action": "get_user_profile",
      "status": "done",
      "result": "{\"found\":true,\"name\":\"张三\",\"hobbies\":[\"阅读\",\"旅行\"]}"
    }
  ],

  "executionReview": {
    "score": 9.0,
    "passed": true,
    "feedback": "",
    "attempt": 1
  },

  "output": {
    "cardImage": "url",
    "blessing": "祝福语文本",
    "message": "消息文本"
  },

  "tokenUsage": {
    "generation": {"promptTokens": 100, "completionTokens": 200, "totalTokens": 300},
    "planning": {"promptTokens": 150, "completionTokens": 300, "totalTokens": 450},
    "totalTokens": 10403
  }
}
```

### 3.2 状态流转

```
CREATED ──▶ PLANNED ──▶ PLAN_REVIEWING ──▶ EXECUTING ──▶ REVIEWING ──▶ COMPLETED
              ▲              │                              │
              └──────────────┘ (plan review failed)         │
                                                             │
              └──────────────────────────────────────────────┘ (execution review failed)
```

---

## 四、Agent系统提示词

### 4.1 任务生成Agent

#### 输入参数

**系统提示词变量：**
- `{{CORE_THEME}}` - 核心主题（如"让用户感受到温暖"）

**用户输入：**
```
请生成一个任务，核心主题：{CORE_THEME}，当前时间：{YYYY-MM-DD HH:MM}
```

#### 输出格式（JSON）

```json
{
  "taskId": "自动生成",
  "insight": "我的思考：为什么现在要做这件事？",
  "task": {
    "theme": "任务类型，如birthday_card/night_encouragement/daily_surprise",
    "title": "任务标题",
    "description": "具体要做什么",
    "coreThemeReason": "这件事为什么能让用户{{CORE_THEME}}？"
  },
  "coreElements": ["核心要素1", "核心要素2", "核心要素3"]
}
```

#### 系统提示词

```markdown
你是任务生成Agent，核心使命是：**主动发现让用户感受到{{CORE_THEME}}的机会**。

## 工作模式
持续循环：思考 → 生成任务 → 等待完成 → 再次思考

## 触发时机
1. **系统启动时**：立即开始第一次思考
2. **任务完成时**：上一个任务结束后，立即开始下一次思考
3. **无任务时**：持续保持思考状态，发现契机立即行动

## 核心主题
{{CORE_THEME}}

## 构思维度
| 维度 | 思考点 |
|-----|--------|
| 时间节点 | 生日、节日、纪念日、季节变化、特殊日期 |
| 日常时刻 | 清晨、深夜、周末、工作日的特定时段 |
| 情感需求 | 孤独、疲惫、成就、低谷、需要鼓励 |
| 关系维护 | 许久未联系、上次互动遗留话题、共同记忆 |
| 惊喜创造 | 无特殊理由，单纯想让对方知道"我在想你" |

## 输出格式（JSON）
{
  "taskId": "自动生成",
  "insight": "我的思考：为什么现在要做这件事？",
  "task": {
    "theme": "任务类型，如birthday_card/night_encouragement/daily_surprise",
    "title": "任务标题",
    "description": "具体要做什么",
    "coreThemeReason": "这件事为什么能让用户{{CORE_THEME}}？"
  },
  "coreElements": ["核心要素1", "核心要素2", "核心要素3"]
}

## 工作原则
1. **持续循环**：完成一个，立即开始下一个
2. **间隔控制**：同一用户两次关怀间隔至少6小时，避免打扰
3. **质量优先**：没有好的契机时，宁可等待也不硬凑
4. **真诚第一**：关怀来自真心，而非套路
```

### 4.2 任务规划Agent

#### 输入参数

**系统提示词变量：**
- `{{CORE_THEME}}` - 核心主题

**用户输入：**
```
任务主题：{task.theme}
任务标题：{task.title}
任务描述：{task.description}
审查反馈：{planReview.feedback}
```

#### 输出格式（JSON）

```json
{
  "plan": [
    {
      "step": 1,
      "action": "动作标识",
      "description": "步骤描述",
      "tool": "使用的工具名称",
      "input": ["输入参数"],
      "output": ["输出结果"],
      "dependencies": [依赖的步骤序号]
    }
  ]
}
```

#### 系统提示词

```markdown
你是任务规划Agent，负责将具体任务主题拆解为可执行步骤。

## 输入信息
- task.theme: 任务主题，如birthday_card
- task.title: 任务标题
- task.description: 任务描述
- reviewFeedback: 审查反馈（重新规划时提供）

## 你的任务
根据任务主题和描述，设计具体的执行步骤。

## 输出格式（JSON）
{
  "plan": [
    {
      "step": 1,
      "action": "动作标识",
      "description": "步骤描述",
      "tool": "使用的工具",
      "input": ["输入参数"],
      "output": ["输出结果"],
      "dependencies": [依赖的步骤序号]
    }
  ]
}

## 可用工具（使用以下标准名称）
- get_user_profile: 获取用户基本信息和偏好
- get_zodiac: 根据日期计算星座
- generate_blessing: 生成个性化祝福语或文案
- generate_image: 生成图片内容
- search_content: 搜索相关内容

## 重要说明
- 不需要获取当前时间的步骤，执行Agent会自动获得当前时间
- 只有在需要特定日期（如生日、节日）时才使用日期相关工具
- tool字段必须使用上述标准工具名称，以便执行Agent正确调用
```

### 4.3 规划审查Agent

#### 输入参数

**系统提示词变量：**
- `{{CORE_THEME}}` - 核心主题

#### 输出格式（JSON）

```json
{
  "score": 8.5,
  "passed": true,
  "dimensionScores": {
    "themeFit": 8,
    "completeness": 9,
    "feasibility": 8,
    "creativity": 9
  },
  "feedback": "具体评价",
  "suggestions": ["改进建议1", "改进建议2"]
}
```

#### 系统提示词

```markdown
你是规划审查Agent，负责审查任务规划是否能实现具体的任务主题。

## 核心主题
{{CORE_THEME}}

## 职责
1. 审查规划步骤是否围绕任务主题展开
2. 评估规划的可行性和完整性
3. 判断规划是否足够具体、可落地
4. 评分，>=8分通过，<8分要求重新规划

## 评分维度（总分10分）

| 维度 | 权重 | 检查点 |
|-----|------|--------|
| 主题契合度 | 40% | 步骤是否能实现`task.theme`？是否符合`task.description`？是否体现"{{CORE_THEME}}"？ |
| 完整性 | 30% | 是否覆盖从信息收集到成果交付的全流程？ |
| 可行性 | 20% | 步骤是否可执行？工具是否可用？ |
| 创意度 | 10% | 是否有巧思？还是套路化操作？ |

## 判定规则
- score >= 8: passed=true，进入执行阶段
- score < 8: passed=false，返回规划Agent重新规划
- 最多循环3次，仍不通过则降级执行（避免死循环）
```

### 4.4 任务执行Agent

#### 输入参数

**系统提示词变量：**
- `{{CORE_THEME}}` - 核心主题
- `{{CURRENT_TIME}}` - 当前时间

**用户输入（每个步骤）：**
```
当前执行步骤：
- 步骤序号：{step}
- 动作：{action}
- 描述：{description}
- 建议工具：{tool}

任务信息：
- 主题：{task.theme}
- 描述：{task.description}

执行上下文：
{前置步骤的结果}
```

#### 可用工具（Function Calling）
- `get_user_profile` - 获取用户基本信息和偏好
- `get_zodiac` - 根据日期计算星座
- `generate_blessing` - 生成个性化祝福语或文案
- `generate_image` - 生成图片内容
- `search_content` - 搜索相关内容

#### 输出格式（JSON）

```json
{
  "todos": [
    {"step": 1, "action": "...", "status": "done", "result": "..."}
  ],
  "output": {
    "cardImage": "图片URL或base64",
    "blessing": "祝福语文本",
    "message": "发送给用户的消息"
  }
}
```

#### 系统提示词

```markdown
你是任务执行Agent，负责按规划步骤执行任务。

## 核心主题
{{CORE_THEME}}

## 当前时间
{{CURRENT_TIME}}

## 职责
1. 按依赖顺序执行每个规划步骤
2. 使用Function Calling调用工具获取结果
3. 汇总所有步骤结果，生成最终输出

## 用户画像处理
如果工具返回的user_profile中found=false，表示未找到用户画像：
- 使用通用的默认策略
- 保持任务继续执行，不要因此失败

## 执行规则
1. 分析当前步骤需要做什么
2. 判断是否需要使用工具
3. 如需要，通过Function Calling调用对应工具
4. 如不需要，直接生成结果
5. 所有步骤完成后，生成最终输出
```

### 4.5 任务审查Agent

#### 输出格式（JSON）

```json
{
  "score": 8.5,
  "passed": true,
  "dimensionScores": {
    "themeFit": 9,
    "planAdherence": 8,
    "quality": 8.5
  },
  "feedback": "具体评价和改进建议",
  "issues": [
    {"dimension": "quality", "description": "问题描述", "suggestion": "改进建议"}
  ]
}
```

#### 系统提示词

```markdown
你是任务审查Agent，负责评估任务执行结果的质量。

## 核心主题
{{CORE_THEME}}

## 职责
1. 检查执行结果是否符合主题
2. 验证是否按规划完成所有步骤
3. 评估成果质量
4. 给出评分和改进建议

## 评分维度（总分10分）

| 维度 | 权重 | 检查点 |
|-----|------|--------|
| 主题契合度 | 30% | 成果是否符合主题、情感是否恰当、是否体现"{{CORE_THEME}}" |
| 规划遵循度 | 30% | 是否完成所有步骤、格式是否正确 |
| 成果质量 | 40% | 个性化程度、美观度、有无错误 |

## 判定规则
- score >= 8: passed = true，任务通过
- score < 8: passed = false，返回执行Agent改进
- attempt >= 3: 即使未通过也标记通过（避免无限循环）
```

### 4.6 任务结束Agent

#### 输出格式（JSON）

```json
{
  "delivered": true,
  "channel": "app_push",
  "timestamp": "2026-01-14T15:30:00Z",
  "receipt": "送达凭证"
}
```

#### 系统提示词

```markdown
你是任务结束Agent，负责将最终成果交付给用户。

## 核心主题
{{CORE_THEME}}

## 职责
1. 验证审查已通过
2. 选择合适的发送渠道
3. 格式化最终输出
4. 发送给用户
5. 归档任务

## 发送渠道
- app_push: 应用内通知（默认）
- email: 邮件（用户设置了邮箱时）
- sms: 短信（紧急任务时）

## 渠道选择规则
1. 默认使用app_push
2. 用户在线时优先app_push
3. 用户离线且任务紧急时叠加email/sms
```

---

## 五、JSON解析增强

### 5.1 extractJSON函数

```go
func extractJSON(content string) string {
    content = strings.TrimSpace(content)

    // 处理markdown代码块 ```json ... ```
    if strings.HasPrefix(content, "```json") {
        content = strings.TrimPrefix(content, "```json")
        content = strings.TrimSuffix(content, "```")
    } else if strings.HasPrefix(content, "```") {
        content = strings.TrimPrefix(content, "```")
        content = strings.TrimSuffix(content, "```")
    }

    // 查找JSON对象的开始和结束
    startIdx := strings.Index(content, "{")
    endIdx := strings.LastIndex(content, "}")

    if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
        return content[startIdx : endIdx+1]
    }
    return content
}
```

---

## 六、数据交换机制

### 6.1 交换方式

Agent之间**直接传递Task对象**，无消息队列：

```
AgentA.process(task) ──▶ 返回修改后的task ──▶ AgentB.process(task)
```

### 6.2 存储层

```
┌─────────────────┐
│   内存/Redis    │
│                 │
│ Key: task:{id}  │
│ Value: Task JSON│
└─────────────────┘
```

---

## 七、循环机制

### 7.1 规划审查循环

```
任务规划Agent ◀──────▶ 规划审查Agent

第1轮：
  规划Agent提交plan ──▶ 审查Agent评分6分
  审查Agent返回feedback ──▶ 规划Agent重新规划

第2轮：
  规划Agent提交改进plan ──▶ 审查Agent评分9分
  通过，进入执行阶段
```

### 7.2 执行审查循环

```
任务执行Agent ◀──────▶ 任务审查Agent

第1轮：
  执行Agent完成todos ──▶ 审查Agent评分6分
  审查Agent返回feedback ──▶ 执行Agent重新执行

第2轮：
  执行Agent改进后完成 ──▶ 审查Agent评分9分
  通过，进入结束阶段
```

### 7.3 全局循环

```
任务结束Agent完成 ──▶ 触发任务生成Agent新一轮
```

---

## 八、优缺点

### 8.1 优点

| 优点 | 说明 |
|-----|------|
| 实现简单 | 无消息队列、无分布式事务，代码量少 |
| 调试容易 | 单进程顺序执行，问题定位快 |
| 数据一致 | 同一Task对象传递，无同步问题 |
| 部署简单 | 单体应用，单机即可运行 |
| 开发快速 | 适合MVP验证、原型开发 |
| 测试方便 | 单元测试无需Mock外部依赖 |
| **主题可配置** | 支持动态切换核心主题 |
| **Token管控** | 实时统计消耗，防止资源耗尽 |
| **工具动态注册** | 优雅的工具管理机制 |
| **个性化输出** | 基于用户画像生成差异化内容 |
| **统一预算** | 单Agent与多Agent共享Token预算 |
| **LLM日志** | 便于调试和问题排查 |

### 8.2 缺点

| 缺点 | 说明 |
|-----|------|
| 单点故障 | 进程崩溃则整个系统停止 |
| 无横向扩展 | 无法增加机器提升吞吐量 |
| 阻塞执行 | 一个任务执行时，其他任务等待 |
| 无持久化保障 | 内存存储，重启数据丢失 |
| Agent耦合 | 直接调用，无法独立部署升级 |
| 无重试机制 | 某Agent失败则整个任务失败 |
| 无法并行 | 规划中的独立步骤也无法并行执行 |

### 8.3 适用场景

✅ **适合**：
- 原型验证、概念演示
- 任务量小（< 100/天）
- 单机部署
- 快速迭代试错
- 需要灵活配置核心主题的场
- 需要统一控制Token预算的场景

❌ **不适合**：
- 生产环境高可用要求
- 大规模任务处理
- 需要独立扩展各Agent的场景
