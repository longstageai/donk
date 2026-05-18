# Prompt 模块架构

## 概述

Prompt 模块负责提示词模板的管理，是 Agent 与 LLM 交互的核心输入来源。

## 目录结构

```
data/prompt/
└── SYSTEM.md    # 系统提示词
```

## 系统提示词内容

SYSTEM.md 包含以下核心部分：

### 1. Agent 角色定义
- AI 助手身份定位
- 核心能力描述

### 2. 工具使用指南
- 可用工具列表
- 何时使用 memory_search vs knowledge_search
- 工具调用格式

### 3. 记忆系统说明
- 短期记忆（当前会话）
- 长期记忆（持久化知识）
- 用户画像（个人偏好）

### 4. 输出格式规范
- 结构化输出要求
- JSON 格式定义

## 使用示例

```go
// 读取系统提示词
prompt, err := os.ReadFile("data/prompt/SYSTEM.md")

// 构建完整消息
messages := []schema.ChatMessage{
    {Role: "system", Content: prompt},
    {Role: "user", Content: userInput},
}

// 调用模型
response, err := model.Chat(ctx, &schema.ChatRequest{
    Messages: messages,
})
```

## 提示词设计原则

1. **清晰的角色定义**：明确 AI 的身份和能力
2. **工具使用指导**：何时用什么工具，避免滥用
3. **记忆系统说明**：帮助模型理解记忆层级
4. **输出格式规范**：确保结构化输出

## 与 Agent 的关系

```
┌─────────────────────────────────────────────────────┐
│                      Agent                           │
├─────────────────────────────────────────────────────┤
│  1. 读取 SYSTEM.md                                  │
│       │                                             │
│       ▼                                             │
│  2. 拼接用户画像 (profile)                           │
│       │                                             │
│       ▼                                             │
│  3. 拼接对话历史 (memory)                            │
│       │                                             │
│       ▼                                             │
│  4. 拼接当前输入                                     │
│       │                                             │
│       ▼                                             │
│  5. 发送给 LLM                                       │
└─────────────────────────────────────────────────────┘
```

## 扩展建议

后续可以考虑：

1. **模板变量**：支持占位符替换
   ```
   你好，{{.UserName}}，我是...
   ```

2. **场景分类**：不同场景使用不同提示词
   ```
   data/prompt/
   ├── SYSTEM.md
   ├── assistant.md
   ├── code.md
   └── creative.md
   ```

3. **版本管理**：提示词版本控制
