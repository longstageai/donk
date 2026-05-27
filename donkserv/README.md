# Donk Server

`donkserv` 是 Donk 的 Go 后端服务，负责 Agent 运行、模型适配、工具调用、Skill 管理、知识库、任务调度、配置持久化和 WebSocket 通知。

## 快速启动

```powershell
go mod download
go run ./cmd
```

默认监听：

```text
http://localhost:65434
```

健康检查：

```powershell
curl http://localhost:65434/health
```

## 配置方式

运行配置主要由桌面端首次引导和设置页面写入本地 SQLite。后端配置服务负责持久化和读取：

- LLM 配置
- Embedding 配置
- Agent 配置
- 知识库开关
- Token 预算
- 睡眠管理状态

## 文档

后端文档位于 [docs/](docs/README.md)：

- Agent SSE 协议
- WebSocket 通知协议
- Setting / Skill / Scheduler API
- Knowledge 模块说明
- Background Agent 和 Creative 架构设计
- `design/` 模块架构说明

## 构建

```powershell
.\sh\build.bat
```

输出：

```text
sh\donk.exe
```

## 运行时数据

`data/` 目录保存本地数据库、历史记录、知识库索引、Skill、脚本运行时等数据。该目录通常不应提交到公开仓库。
