# Donk

English | [简体中文](README.md)

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![Flutter](https://img.shields.io/badge/Flutter-3.7+-02569B?logo=flutter&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows-lightgrey)

Donk is a local-first AI Agent desktop application built with a Go backend and a Flutter Windows client. It combines LLM chat, tool calling, Skill extensions, local knowledge base retrieval, long-term memory, task scheduling, real-time notifications, and desktop interaction into a runnable and extensible personal workflow platform.

Donk is not just another chat window. It is designed as a local Agent runtime: users talk to the Agent from the desktop client, while the Agent can search local knowledge, call tools, execute Skill scripts, save memories, create scheduled tasks, and push background events back to the client in real time.

## Table of Contents

- [Screenshots](#screenshots)
- [Use Cases](#use-cases)
- [Highlights](#highlights)
- [Capabilities](#capabilities)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [HTTP API](#http-api)
- [Skill Development](#skill-development)
- [Build](#build)
- [Development Notes](#development-notes)
- [Security Notes](#security-notes)

## Use Cases

| Use case | What Donk provides |
| --- | --- |
| Local personal AI assistant | Conversations, long-term memory, knowledge base, and daily task management |
| Agent tool platform | File, command, browser, HTTP, document parsing, knowledge search, and task tools |
| Skill runtime | Extend Agent behavior with `SKILL.md`, scripts, references, and assets |
| Automation entry point | Create one-time, delayed, or Cron tasks with natural language |
| Desktop Agent lab | Separate frontend/backend architecture for replacing models, tools, workflows, and UI |

## Highlights

- **Local-first**: configuration, conversations, tasks, knowledge data, Skills, and runtime state are stored locally.
- **Configured in the desktop UI**: LLM, Embedding, Agent, knowledge base, and general switches are configured from frontend pages.
- **Streaming Agent chat**: SSE events for reasoning, content, tool calls, tool results, warnings, and completion.
- **Extensible tool system**: built-in tools for files, commands, HTTP, browser control, document parsing, knowledge search, tasks, and more.
- **Skill-based extension**: extend Agent capabilities with `SKILL.md`, scripts, references, assets, and dependency metadata.
- **Knowledge and memory**: local semantic retrieval powered by Embeddings and vector storage.
- **Scheduling and notifications**: background tasks, run history, and real-time WebSocket notifications.

## Screenshots

The main interface centers around chat. Navigation is on the left, streamed Agent responses appear in the main area, and the input box supports text, attachments, mentions, and quick actions.

![Donk chat interface](docs/img/Snipaste_2026-05-27_14-32-01.png)

| Skill Gallery | Task Management |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-32-40.png" alt="Donk Skill Gallery" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-32-52.png" alt="Donk Task Management" width="420"> |
| Browse and manage local Skills available to the Agent. | View background tasks, status, schedules, and enable/disable state. |

| Notifications | Settings |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-33-15.png" alt="Donk Notifications" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-33-53.png" alt="Donk Settings" width="420"> |
| Centralized task delivery, system status, and background events. | Configure LLM, Embedding, Agent, knowledge base, and general switches. |

| WeChat Connection | About |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-33-33.png" alt="Donk WeChat Connection" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-34-07.png" alt="Donk About Page" width="420"> |
| View WeChat connection status and disconnect when needed. | Application name, icon, and version information. |

## Capabilities

### Agent Chat

- `POST /api/v1/chat` provides SSE streaming responses.
- Supports user input confirmation, reasoning deltas, content deltas, full assistant messages, tool calls, tool results, warnings, and completion events.
- Loads chat history, long-term memory, user profile, and token statistics.
- Uses the tool registry to access built-in tools and Skill tools.

### Models and Embeddings

- LLM provider adapters live in `donkserv/internal/model`.
- Current providers include `openai`, `qwen`, `deepseek`, and `doubao`.
- Embedding adapters live in `donkserv/internal/embedding`.
- LLM and Embedding settings are persisted by the configuration service and managed from the desktop settings pages.

### Tool System

Built-in tools live in `donkserv/internal/tool/builtin`, including:

- File read/write
- Command execution
- HTTP requests
- Browser control
- Calculator
- PDF parsing
- Word parsing
- Knowledge search
- Long-term memory save/search
- Conversation search
- Task management
- Skill invocation
- Skill creation
- Skill installation
- Python script execution
- Python dependency management

Tools are registered into `tool.Registry` and exposed to the Agent runtime.

### Skill Extensions

Donk loads local Skills from:

```text
donkserv/data/skills
```

A typical Skill looks like:

```text
skill-name/
├── SKILL.md
├── scripts/
├── references/
└── assets/
```

Skill features:

- Parse instructions and trigger semantics from `SKILL.md`.
- Load scripts and reference files.
- Register enabled Skills as Agent capabilities.
- Manage Skills through APIs: list, enable, disable, delete, and rescan.
- Watch filesystem changes and sync updates.
- Support script runtime and dependency metadata.

### Knowledge Base

The knowledge module lives in `donkserv/internal/knowledge`. It scans local documents, builds indexes, and provides semantic retrieval with Embeddings and vector storage.

It supports:

- Scheduled local directory scanning.
- Configurable scan depth, batch size, file size limit, and interval.
- Hot/warm knowledge data handling.
- Semantic retrieval through the `knowledge_search` tool.
- Start, stop, and status control through configuration APIs.

### Long-Term Memory and User Profile

Related modules:

```text
donkserv/internal/memory
donkserv/internal/profile
```

Capabilities:

- Conversation history storage.
- Long-term memory save and semantic search.
- User profile extraction, update, and management.
- Shared history and profile context for Agent and Creative workflows.

### Task Scheduling

The scheduler module lives in `donkserv/internal/scheduler` and supports:

- One-time tasks
- Delayed tasks
- Cron tasks
- Agent task executor
- Run history
- Cancel, trigger, delete, and query operations
- WebSocket event push

Agents can create background tasks through the task tool, enabling workflows such as reminders, daily jobs, and periodic information processing.

### Creative Workflow

The Creative module lives in:

```text
donkserv/internal/creative
```

It provides a more complex runtime than a regular single-turn Agent, suitable for goal decomposition, creative generation, task planning, and staged execution. It can connect to the scheduler, knowledge base, long-term memory, user profile, and WebSocket hooks.

### Desktop Client

The Flutter client lives in `donkui`. Main pages include:

- `home`: main chat UI
- `idea`: Skill and idea-related UI
- `task`: tasks and run history
- `notification`: notification center
- `setting`: model, Embedding, Agent, knowledge base, and application settings
- `onboarding`: first-run setup flow

The client uses:

- `GetX` for state and dependency management
- `GoRouter` for routing
- `window_manager` for desktop window control
- `tray_manager` for tray integration
- SSE client for chat streams
- WebSocket client for notification events

## Architecture

```text
┌──────────────────────────────────────────────┐
│                  Flutter UI                  │
│  Chat / Settings / Skills / Tasks / Notify   │
└───────────────┬─────────────────────┬────────┘
                │ HTTP/SSE            │ WebSocket
                ▼                     ▼
┌──────────────────────────────────────────────┐
│                Go Backend (Gin)              │
│  Chat API / Config API / Skill API / Tasks   │
└───────────────┬─────────────────────┬────────┘
                │                     │
                ▼                     ▼
┌──────────────────────────┐   ┌──────────────────────────┐
│        Agent Runtime     │   │      Scheduler/Notify     │
│ LLM / Tools / Memory     │   │ Cron / Runs / WS Events   │
└───────────────┬──────────┘   └──────────────┬───────────┘
                │                             │
                ▼                             ▼
┌──────────────────────────────────────────────┐
│        Local Data and Extension Layer        │
│ SQLite / Skills / Knowledge / Vector Store   │
└──────────────────────────────────────────────┘
```

Typical chat flow:

1. Flutter sends a user message to `POST /api/v1/chat`.
2. The Go backend creates an Agent request context.
3. The Agent loads configuration, history, long-term memory, user profile, and available tools.
4. The LLM produces streaming output or tool call intents.
5. The tool registry executes file, command, knowledge, Skill, task, and other tools.
6. The backend streams SSE events back to the client.
7. Background tasks and notifications are pushed to the desktop notification center through WebSocket.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Backend | Go |
| HTTP service | Gin |
| Realtime | HTTP SSE, gorilla/websocket |
| Database | SQLite |
| Vector storage | cortexdb |
| Scheduling | robfig/cron |
| Document parsing | PDF and Word parsing tools |
| Frontend | Flutter Windows |
| Frontend state | GetX |
| Frontend routing | GoRouter |
| Desktop integration | window_manager, tray_manager |
| Models | OpenAI, Qwen, DeepSeek, Doubao provider adapters |

## Project Structure

<details>
<summary>Expand project tree</summary>

```text
.
├── README.md
├── README_EN.md
├── LICENSE
├── docs/
├── donkserv/
│   ├── cmd/                  # Application assembly and startup
│   ├── conf/                 # Built-in defaults and background service config
│   ├── internal/             # Core backend modules
│   ├── pkg/                  # Shared packages
│   ├── data/                 # Runtime data, Skills, knowledge, history
│   ├── sh/                   # Backend build scripts
│   ├── go.mod
│   └── go.sum
└── donkui/
    ├── lib/
    │   ├── app/              # App initialization, routing, layout, config
    │   ├── common/           # Clients, services, models, widgets
    │   ├── l10n/             # Localization
    │   └── ui/               # Home, settings, tasks, notifications, onboarding
    ├── assets/
    ├── docs/                 # Frontend/backend protocol docs
    ├── scripts/              # Windows installer scripts
    ├── windows/
    ├── pubspec.yaml
    └── pubspec.lock
```

</details>

## Quick Start

### Requirements

- Windows 10/11
- Go 1.26 or later
- Flutter SDK 3.7 or later
- Windows desktop development environment
- SQLite CGO build environment
- A valid LLM API Key
- Optional: Embedding API Key for knowledge search and long-term memory
- Optional: Inno Setup for building a Windows installer

### 1. Clone

```powershell
git clone https://github.com/longstageai/dank.git
cd dank
```

### 2. Start Backend

```powershell
cd donkserv
go mod download
go run ./cmd
```

Default backend URL:

```text
http://localhost:65434
```

Health check:

```powershell
curl http://localhost:65434/health
```

### 3. Start Desktop Client

Open another terminal:

```powershell
cd donkui
flutter pub get
flutter run -d windows
```

Default client endpoints are defined in:

```text
donkui/lib/app/conf/config.dart
```

Default values:

```text
API: http://localhost:65434/api/v1
SSE: http://localhost:65434/api/v1/chat
WebSocket: ws://localhost:65434/ws/events
```

The auto-start backend process call in `donkui/lib/app/init/app.dart` is currently commented out, so during development the backend should be started manually first.

### 4. Configure in Desktop UI

Donk configuration is done from the desktop UI.

On first launch, complete the onboarding flow:

- LLM Provider, model name, API Key, Base URL
- Embedding Provider, model name, API Key, Base URL
- Agent runtime settings

Later, adjust settings from `Settings`:

- `LLM Settings`: provider, model, key, and endpoint.
- `Embedding Settings`: vector model for knowledge retrieval and long-term memory.
- `Agent Settings`: loop count, timeout, history, and token budget.
- `General Settings`: language, security protection, knowledge auto-build, and sleep prevention.
- `Usage Statistics`: token usage and budget state.

## Configuration

Runtime configuration is persisted by the configuration service and edited through desktop settings pages.

| Page | Configuration | Purpose |
| --- | --- | --- |
| Onboarding | Basic LLM and Embedding parameters | Make the Agent usable on first launch |
| LLM Settings | Provider, model, API Key, Base URL | Controls chat and Agent reasoning model |
| Embedding Settings | Provider, model, API Key, Base URL | Controls knowledge search and semantic memory |
| Agent Settings | Loop count, convergence, timeout, history, token budget | Controls Agent behavior boundaries |
| General Settings | Language, security protection, knowledge auto-build, sleep prevention | Application-level switches |
| Usage Statistics | Token usage and budget state | Model usage visibility |

The desktop client writes configuration to the local SQLite database through the backend. After restart, Donk continues using the saved local configuration.

## HTTP API

<details>
<summary>Expand API list</summary>

| Capability | Method and path | Description |
| --- | --- | --- |
| Health | `GET /health` | Check whether the service is running |
| Chat | `POST /api/v1/chat` | Agent SSE chat |
| Config | `GET /api/v1/config` | Get full configuration |
| Config | `PUT /api/v1/config` | Update full configuration |
| LLM Config | `GET /api/v1/config/llm` | Get LLM configuration |
| LLM Config | `PUT /api/v1/config/llm` | Update LLM configuration |
| Embedding Config | `GET /api/v1/config/embedding` | Get Embedding configuration |
| Embedding Config | `PUT /api/v1/config/embedding` | Update Embedding configuration |
| Agent Config | `GET /api/v1/config/agent` | Get Agent configuration |
| Agent Config | `PUT /api/v1/config/agent` | Update Agent configuration |
| Knowledge Config | `GET /api/v1/config/knowledge` | Get knowledge configuration |
| Knowledge Config | `PUT /api/v1/config/knowledge` | Update knowledge configuration |
| Knowledge Status | `GET /api/v1/knowledge/status` | Query knowledge status |
| Knowledge Control | `POST /api/v1/knowledge/start` | Start knowledge service |
| Knowledge Control | `POST /api/v1/knowledge/stop` | Stop knowledge service |
| Skill List | `GET /api/v1/skills` | List Skills |
| Skill Rescan | `POST /api/v1/skills/rescan` | Rescan Skills |
| Skill Detail | `GET /api/v1/skills/:name` | Get one Skill |
| Skill Delete | `DELETE /api/v1/skills/:name` | Delete one Skill |
| Skill Enable | `POST /api/v1/skills/:name/enable` | Enable Skill |
| Skill Disable | `POST /api/v1/skills/:name/disable` | Disable Skill |
| Task Create | `POST /api/v1/tasks` | Create scheduled task |
| Task List | `GET /api/v1/tasks` | List tasks |
| Task Detail | `GET /api/v1/tasks/:id` | Get task detail |
| Task Delete | `DELETE /api/v1/tasks/:id` | Delete task |
| Task Cancel | `POST /api/v1/tasks/:id/cancel` | Cancel task |
| Task Trigger | `POST /api/v1/tasks/:id/run` | Trigger task manually |
| Runs | `GET /api/v1/runs` | List task runs |
| Token Usage | `GET /api/v1/tokens/usage` | Query token usage |
| Token Budget | `GET /api/v1/tokens/budget` | Query token budget |
| Creative Status | `GET /api/v1/creative/status` | Query Creative status |
| Creative Start | `POST /api/v1/creative/start` | Start Creative workflow |
| Creative Stop | `POST /api/v1/creative/stop` | Stop Creative workflow |
| WebSocket Events | `GET /ws/events` | Notification and task event stream |
| WebSocket Test | `POST /ws/test-push` | Push test notification |

</details>

Detailed protocol docs:

```text
donkui/docs/agent_protocol.md
donkui/docs/websocket_protocol.md
donkui/docs/skill_api.md
donkui/docs/setting_api.md
donkui/docs/scheduler-api.md
donkui/docs/knowledge-base.md
```

## Skill Development

Basic structure:

```text
my-skill/
├── SKILL.md
├── scripts/
│   └── main.py
├── references/
│   └── guide.md
└── assets/
```

Minimal `SKILL.md`:

```markdown
---
name: my-skill
description: Use this Skill when the user needs to handle a specific task category.
version: 1.0.0
---

# My Skill

When the user asks for this task, read these instructions and run `scripts/main.py`.
```

Load flow:

1. Put Skill files in `donkserv/data/skills`.
2. Backend scans Skills on startup.
3. Skill state is synced to the database.
4. Enabled Skills are registered into the Skill Registry.
5. The Agent reads instructions, executes scripts, or loads references through the `skill` tool.
6. File changes are picked up by the watcher.

## Local Data

Donk stores runtime data under the backend data directory, for example:

```text
donkserv/data/db/
donkserv/data/history/
donkserv/data/knowledge/
donkserv/data/skills/
donkserv/data/script_runtime/
```

These directories may contain SQLite databases, history, memory, knowledge indexes, vector data, local Skills, Python runtimes, and dependencies. Avoid committing real runtime data, user files, databases, vector stores, and API keys to a public repository.

## Build

### Backend

```powershell
cd donkserv
.\sh\build.bat
```

Output:

```text
donkserv/sh/donk.exe
```

Or build directly:

```powershell
cd donkserv
go build -ldflags="-s -w" -o sh\donk.exe ./cmd/...
```

### Flutter Windows

```powershell
cd donkui
flutter pub get
flutter build windows --release
```

### Installer

Installer script:

```text
donkui/scripts/build_installer.bat
```

Run:

```powershell
cd donkui
.\scripts\build_installer.bat
```

The script depends on Inno Setup and checks for:

```text
donkui/server/donk.exe
```

Build the backend and place it there before packaging the desktop application.

## Development Notes

Backend:

```powershell
cd donkserv
go test ./...
go run ./cmd
```

Frontend:

```powershell
cd donkui
flutter pub get
flutter run -d windows
```

Important paths:

- Backend tools: `donkserv/internal/tool/builtin`
- Model adapters: `donkserv/internal/model`
- Knowledge module: `donkserv/internal/knowledge`
- Skill module: `donkserv/internal/skill`
- Frontend service config: `donkui/lib/app/conf/config.dart`
- Frontend routes: `donkui/lib/app/router/routes.dart`
- Chat UI: `donkui/lib/ui/home`
- Settings UI: `donkui/lib/ui/setting`

## Runtime Notes

- The desktop window defaults to `960x720`.
- The Flutter client uses single-instance mode.
- Closing the window hides it to tray instead of stopping the backend.
- In development mode, start the backend manually first.
- Backend port is currently `65434`.
- If knowledge or long-term memory is unavailable, check Embedding settings.
- If chat does not respond, check LLM provider, API key, Base URL, and model name.
- If Skills do not appear, check `data/skills`, `SKILL.md` frontmatter, and enabled state.

## Security Notes

Donk has powerful local execution capabilities, including command execution, file writing, HTTP requests, browser control, and script execution. Use it carefully:

- Do not load untrusted Skills.
- Do not commit real API keys to public repositories.
- Do not commit `data/db`, `data/history`, `data/knowledge`, or other runtime data.
- Treat command execution, file writing, and script execution tools with caution.
- If exposing the backend beyond localhost, enable authentication and restrict access.

## Current Status

This repository is under active development:

- Some historical comments may have encoding display issues.
- `donkserv/data` may contain local runtime data and example Skills.
- Automatic backend startup from the frontend is currently commented out.
- Runtime configuration is handled through onboarding and settings pages.

## License

This project is licensed under GPL-3.0. See [LICENSE](LICENSE).
