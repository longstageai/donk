# Profile 模块架构

## 概述

Profile 模块负责用户画像管理，从对话历史中提取和维护用户特征、偏好和背景信息。

## 目录结构

```
internal/profile/
├── storage.go    # 画像存储
├── manager.go    # 画像管理器
├── loader.go     # 画像加载
├── analyzer.go   # 画像分析
├── buffer.go     # 消息缓冲区
└── message.go    # 消息定义
```

## 核心组件

### 1. ProfileManager（画像管理器）

```go
type ProfileManager struct {
    storage  Storage
    analyzer *Analyzer
    buffer   *Buffer
}
```

### 2. Storage（存储接口）

```go
type Storage interface {
    Save(ctx context.Context, profile *UserProfile) error
    Load(ctx context.Context, userID string) (*UserProfile, error)
    Delete(ctx context.Context, userID string) error
}
```

### 3. Analyzer（分析器）

从对话中提取用户特征。

```go
type Analyzer struct {
    model model.Adapter
}

type UserProfile struct {
    UserID      string
    Name        string
    Background  string
    Preferences []Preference
    Facts       []Fact
    UpdatedAt   time.Time
}
```

### 4. Buffer（消息缓冲区）

累积对话消息，批量分析。

```go
type Buffer struct {
    messages []Message
    maxSize  int
}
```

## 画像内容

```go
type Preference struct {
    Key      string
    Value    string
    Category string
}

type Fact struct {
    Statement string
    Source    string
    Confidence float64
}
```

## 使用示例

```go
// 创建画像管理器
pm, err := profile.NewManager(cfg, modelAdapter)

// 加载用户画像
userProfile, err := pm.LoadProfile(ctx, "user123")

// 添加对话消息
pm.AddMessage(ctx, "user123", profile.Message{
    Role:    "user",
    Content: "我喜欢蓝色",
})

// 分析更新画像
err := pm.Analyze(ctx, "user123")

// 保存画像
err := pm.SaveProfile(ctx, userProfile)

// 获取画像摘要
summary := pm.GetSummary(userProfile)
```

## 核心流程

```
┌─────────────────────────────────────────────────────┐
│                    用户画像更新流程                  │
├─────────────────────────────────────────────────────┤
│  1. 新对话消息                                      │
│       │                                             │
│       ▼                                             │
│  2. 添加到 Buffer                                   │
│       │                                             │
│       ▼                                             │
│  3. 判断是否达到分析阈值                             │
│       │                                             │
│       ├── 未达到 → 等待更多消息                      │
│       │                                             │
│       ▼                                             │
│  4. 调用 Analyzer 分析                               │
│       │                                             │
│       ▼                                             │
│  5. 提取 Preferences 和 Facts                        │
│       │                                             │
│       ▼                                             │
│  6. 合并到用户画像                                   │
│       │                                             │
│       ▼                                             │
│  7. 持久化存储                                      │
└─────────────────────────────────────────────────────┘
```

## 与其他模块的关系

```
┌─────────────┐
│   agent     │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│   memory    │────▶│   profile   │
│  (对话历史)  │     │  (用户画像)  │
└─────────────┘     └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Storage    │
                    │  (持久化)   │
                    └─────────────┘
```

- agent 调用 profile 模块获取用户背景信息
- memory 提供对话历史作为分析来源
- Storage 负责画像的持久化

## 设计特点

1. **增量更新**：从对话中逐步提取和更新用户信息
2. **批量分析**：使用 Buffer 累积消息，批量处理
3. **置信度**： Facts 支持置信度评估
4. **分类存储**： Preferences 按类别组织
