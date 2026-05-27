# Donk Server 文档

本文档目录维护 `donkserv` 后端服务的协议、API 和核心模块设计说明。当前后端默认监听：

```text
http://localhost:65434
```

## 文档索引

### 协议与 API

| 文档 | 内容 |
| --- | --- |
| [agent_protocol.md](agent_protocol.md) | `POST /api/v1/chat` SSE 流式对话协议 |
| [websocket_protocol.md](websocket_protocol.md) | `GET /ws/events` WebSocket 通知协议 |
| [setting_api.md](setting_api.md) | 配置、知识库、睡眠管理、Token 统计 API |
| [skill_api.md](skill_api.md) | Skill 列表、详情、启停、脚本查看 API |
| [scheduler-api.md](scheduler-api.md) | 任务调度和执行记录 API |
| [knowledge-base.md](knowledge-base.md) | 知识库模块设计与运行说明 |

### 架构设计

| 文档 | 内容 |
| --- | --- |
| [background-agent-design.md](background-agent-design.md) | 后台 Agent 服务设计 |
| [multi-agent-creative-architecture.md](multi-agent-creative-architecture.md) | Creative 多 Agent 目标生成设计 |
| [multi-agent-event-loop-architecture.md](multi-agent-event-loop-architecture.md) | Creative 事件循环协作设计 |
| [embedding_dimensions.md](embedding_dimensions.md) | Embedding 向量维度说明 |
| [design/](design/) | Agent、Tool、Memory、Knowledge、Prompt 等模块架构说明 |

## 维护原则

- 后端运行配置以桌面端首次引导和设置页面为主要入口。
- API 示例统一使用 `http://localhost:65434`。
- `donkserv/data` 下的数据库、历史记录、知识库和运行时文件属于本地数据，不应作为接口文档维护。
- 已删除旧 `internal/multiagent`、`internal/skilldiscovery` 对应文档，避免和当前代码结构冲突。
