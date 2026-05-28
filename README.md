# Donk

English | [简体中文](README_CH.md)

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![Flutter](https://img.shields.io/badge/Flutter-3.7+-02569B?logo=flutter&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows-lightgrey)

# 🐴 Nuclear Donkey (Donk) - Your Fully Automated Local AI Work Partner

**A bold experiment in progress: when AI has complete context and enough capability, how much value can it autonomously create for you?**

---

## 🤔 Why Are We Building Nuclear Donkey?

Have you ever had this experience?
- You download a bunch of AI tools, open them, and then stare at an empty input box
- You know AI is powerful, but you still do not know "what exactly can it help me do?"
- Every time, you rack your brain for prompts, only to feel unsatisfied with the result
- You watch other people multiply their productivity with AI, while you cannot even take the first step

**This is not your problem. It is the problem with today's AI products.**

They all push the hardest question, "how to use AI", onto ordinary users.

Nuclear Donkey was born to solve this problem. We believe: **the best AI is the AI whose presence you can barely feel.**

---

## ⚡ What Is Nuclear Donkey?

Nuclear Donkey is not just another chat window. It is a **fully automated AI work partner that quietly runs in the background on your computer**.

You only need to do three things:
1. Download and install it
2. Tell it who you are and what you care about
3. Wait for results

Then Nuclear Donkey will:
- 🧠 Automatically learn your work habits and knowledge system
- 🔍 Proactively discover where you may need help
- 🛠️ Decide which tools to call and which tasks to execute
- 📅 Arrange its own work schedule
- 📱 Notify you only when it truly needs you

It is like a tireless, high-powered donkey working out of sight, handling tedious, repetitive work that you may not even have realized could be automated.

---

## 🧪 This Is an Open-Source, Large-Scale Social Experiment

Nuclear Donkey is not only a software product. It is also a **scientific experiment about the boundaries of AI capability**.

We are testing a radical hypothesis:
> **When an AI Agent has enough local context, complete tool permissions, and unlimited runtime, can it autonomously evolve reusable capabilities and work scenarios that humans have never imagined?**

This experiment needs your participation. Every user who downloads and uses Nuclear Donkey, every task it completes autonomously, and every workflow it invents by itself helps answer this question.

We will consume massive amounts of tokens and record countless attempts and failures to explore one possibility: **can AI truly become an "extension" of humanity, instead of yet another tool that we must learn how to use?**

---

## ✨ What Nuclear Donkey Can Do (Now and in the Future)

- 📚 **Local knowledge base**: Automatically indexes documents on your computer without uploading them to the cloud
- 🧠 **Long-term memory**: Remembers every sentence you have said and every decision you have made
- 🛠️ **Tool calling**: Automatically uses tools such as the browser, file system, and terminal
- 📝 **Skill extensions**: Supports custom scripts for unlimited capability expansion
- ⏰ **Task scheduling**: Schedules tasks by itself and executes them at the right time
- 🔔 **Real-time notifications**: Disturbs you only when your decision is needed
- 🖥️ **Desktop interaction**: Deeply integrates with Windows and truly becomes part of your computer

---

## 🚀 Quick Start

1. Download the latest version from the [Releases](https://github.com/longstageai/donk/releases) page
2. Double-click to install it and complete the initial setup wizard
3. Tell Nuclear Donkey who you are and what you want it to help you with
4. Minimize the window and go do what actually matters
5. Wait for it to surprise you

---

## 🤝 Join Our Experiment

Nuclear Donkey is currently in an early experimental stage. It may make mistakes, do silly things, or even do nothing at all. But that is exactly what makes it an experiment.

If you are also curious about the future of AI, tired of "prompt engineering", and believe AI should work for us rather than the other way around:

- ⭐ Give us a Star so more people can see this experiment
- 🐛 Submit an Issue and tell us what interesting or foolish things it did
- 💡 Share your ideas and tell us what you hope it can do for you
- 🔧 Submit a PR to help it become smarter and more powerful

**Our goal is not to build a perfect product, but to jointly explore the infinite possibilities of AI.**

---

## 📢 Experimental Statement

> "Nuclear Donkey is an experimental project. It may produce unexpected results, consume large amounts of computing resources, or fail completely. But if we succeed, we will redefine the relationship between humans and AI."
>
> -- Nuclear Donkey Experiment Team

---

*Nuclear Donkey - Let AI work for you, not you for AI.*

## Table of Contents

- [Interface Preview](#interface-preview)
- [Use Cases](#use-cases)
- [Core Features](#core-features)
- [Current Capabilities](#current-capabilities)
- [Architecture Overview](#architecture-overview)
- [Technology Stack](#technology-stack)
- [Directory Structure](#directory-structure)
- [Quick Start](#quick-start-1)
- [Configuration](#configuration)
- [HTTP API](#http-api)
- [Skill Development](#skill-development)
- [Build](#build)
- [Development Notes](#development-notes)
- [Security Notes](#security-notes)

## Use Cases

| Scenario | What Donk Provides |
| --- | --- |
| Local personal AI assistant | Manages conversations, long-term memory, knowledge bases, and daily tasks |
| Agent tool platform | Registers capabilities such as files, commands, browser, HTTP, document parsing, and knowledge-base retrieval as tools |
| Skill runtime environment | Extends the Agent's vertical task capabilities through `SKILL.md`, scripts, and references |
| Automation task entry point | Creates one-off, delayed, or Cron-scheduled tasks with natural language |
| Desktop Agent experiment project | Separates frontend and backend, making it easy to replace models, tools, workflows, and UI |

## Core Features

- **Local first**: Configurations, sessions, tasks, knowledge bases, Skills, and runtime state are stored locally.
- **Desktop configuration**: Models, Embedding, Agent, knowledge base, and general switches are all configured through the frontend UI.
- **Streaming Agent chat**: Returns reasoning, content, tool calls, tool results, and other events through SSE.
- **Extensible tool system**: Built-in tools cover files, commands, HTTP, browser, document parsing, knowledge bases, tasks, and more.
- **Skill plugin system**: Extends Agent capabilities through `SKILL.md`, scripts, references, and dependency declarations.
- **Knowledge base and long-term memory**: Uses Embedding and vector storage to build local semantic retrieval.
- **Task scheduling and notifications**: Supports background tasks, run records, and real-time WebSocket notifications.

## Interface Preview

The main interface is centered on conversation. The left side provides functional navigation, the middle area displays the Agent's streaming replies, thinking state, and action buttons, and the bottom input area supports text input, attachments, mentions, and quick entry points.

![Donk main chat interface](docs/img/Snipaste_2026-05-27_14-32-01.png)

| Inspiration Plaza | Task Management |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-32-40.png" alt="Donk Inspiration Plaza" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-32-52.png" alt="Donk Task Management" width="420"> |
| Manage local Skills and quickly view the Agent's currently available capabilities. | View background tasks, run status, schedules, and enabled or disabled states. |

| Notifications | Settings Center |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-33-15.png" alt="Donk Notifications" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-33-53.png" alt="Donk Settings Center" width="420"> |
| Centrally displays task deliveries, system status, and background events. | Configure LLM, Embedding, Agent, knowledge base, and general switches. |

| WeChat Connection | About Page |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-33-33.png" alt="Donk WeChat Connection" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-34-07.png" alt="Donk About Page" width="420"> |
| View WeChat connection status and disconnect when needed. | Displays the app name, icon, and version information. |

## Current Capabilities

### Agent Chat

- Provides SSE streaming responses through `POST /api/v1/chat`.
- Supports user input confirmation, reasoning deltas, content deltas, complete replies, tool calls, tool results, warnings, and done events.
- Supports history loading, long-term memory retrieval, user profiles, and token statistics.
- The Agent can access built-in tools and Skill tools through the tool registry.

### Models and Embedding

- The LLM Provider adapter layer is located in `donkserv/internal/model`.
- The current code includes adapters for `openai`, `qwen`, `deepseek`, and `doubao`.
- The Embedding adapter layer is located in `donkserv/internal/embedding`.
- LLM and Embedding configurations are persisted by the configuration service and can be managed from the desktop settings page.

### Tool System

Built-in tools are located in `donkserv/internal/tool/builtin` and include, but are not limited to:

- File reading and writing
- Command execution
- HTTP requests
- Browser control
- Calculator
- PDF parsing
- Word parsing
- Knowledge-base search
- Long-term memory save and search
- Conversation history search
- Task management
- Skill invocation
- Skill creation
- Skill installation
- Python script execution
- Python dependency management

Tools are registered into `tool.Registry` and then passed to the Agent for decision-making and invocation.

### Skill Extensions

Donk supports loading Skills from a local directory:

```text
donkserv/data/skills
```

A Skill usually contains:

```text
skill-name/
├── SKILL.md
├── scripts/
├── references/
└── assets/
```

Skill capabilities include:

- Reading skill instructions and trigger semantics from `SKILL.md`.
- Loading scripts and references.
- Registering enabled Skills as Agent-callable capabilities.
- Querying, enabling, disabling, deleting, and rescanning through APIs.
- Automatically syncing changes through file watching.
- Supporting script runtimes and dependency declarations.

### Knowledge Base

The knowledge-base module is located in `donkserv/internal/knowledge`. It is responsible for local document scanning, indexing, and semantic retrieval. It combines Embedding with vector storage to turn local documents into searchable context for the Agent.

Supported capabilities include:

- Periodically scanning local directories.
- Configuring scan depth, batch size, file size limits, and scan intervals.
- Tiering knowledge files into hot and cold data.
- Providing semantic retrieval through the `knowledge_search` tool.
- Controlling startup, shutdown, and status queries through configuration APIs.

### Long-Term Memory and User Profile

Related modules:

```text
donkserv/internal/memory
donkserv/internal/profile
```

Capabilities include:

- Conversation history persistence.
- Long-term memory persistence and semantic search.
- User profile extraction, updates, and management.
- Shared history and profile context between the Agent and Creative workflow.

### Task Scheduling

The scheduler is located in `donkserv/internal/scheduler` and supports:

- One-off tasks
- Delayed tasks
- Cron recurring tasks
- Agent task executor
- Task run records
- Cancellation, triggering, deletion, and querying
- WebSocket event push

The Agent can create background tasks through task tools, enabling workflows such as "remind me later", "run once every day", and "organize materials on a schedule".

### Creative Workflow

The Creative module is located in:

```text
donkserv/internal/creative
```

It provides a more complex runtime than a normal single-turn Agent and is suitable for goal decomposition, idea generation, task planning, and staged execution. The current startup process registers default LLM Agents and connects to the scheduler, knowledge base, long-term memory, user profile, and WebSocket hooks.

### Desktop Client

The Flutter client is located in `donkui`. Main pages include:

- `home`: main chat interface
- `idea`: skill and idea-related interfaces
- `task`: tasks and run records
- `notification`: notification center
- `setting`: model, Embedding, Agent, knowledge-base, and app settings
- `onboarding`: first-time configuration guide

The client uses:

- `GetX` for state and dependency management
- `GoRouter` for routing
- `window_manager` for desktop window management
- `tray_manager` for tray interaction
- An SSE client to receive chat streams
- A WebSocket client to receive notification events

## Architecture Overview

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

Data flow for a typical chat request:

1. Flutter sends a user message through `POST /api/v1/chat`.
2. The Go backend creates the Agent request context.
3. The Agent reads configuration, history, long-term memory, user profile, and available tools.
4. The LLM generates streaming output or tool-call intent.
5. The tool registry executes tools such as files, commands, knowledge base, Skills, and tasks.
6. The backend continuously returns events through SSE.
7. If background tasks or notifications are generated, WebSocket pushes them to the desktop notification center.

## Technology Stack

| Layer | Technology |
| --- | --- |
| Backend language | Go |
| HTTP service | Gin |
| Real-time communication | HTTP SSE, gorilla/websocket |
| Database | SQLite |
| Vector storage | cortexdb |
| Scheduling | robfig/cron |
| Document parsing | PDF and Word parsing tools |
| Frontend | Flutter Windows |
| Frontend state | GetX |
| Frontend routing | GoRouter |
| Desktop capabilities | window_manager, tray_manager |
| Models | OpenAI, Qwen, DeepSeek, Doubao Provider Adapter |

## Directory Structure

<details>
<summary>Expand to view repository directories</summary>

```text
.
├── README.md
├── LICENSE
├── docs/
├── donkserv/
│   ├── cmd/
│   │   ├── aclaw.go           # Application assembly and startup flow
│   │   ├── agent.go           # Agent builder
│   │   ├── http.go            # HTTP service and base routes
│   │   ├── websocket.go       # WebSocket event service
│   │   ├── scheduler.go       # Scheduler assembly
│   │   ├── background.go      # Background Agent service
│   │   └── init.go            # Application and database initialization
│   ├── conf/                  # Built-in default resources and background service config
│   ├── internal/
│   │   ├── agent/             # Main Agent runtime logic
│   │   ├── background/        # Background Agent Runner
│   │   ├── config/            # Data directory and path configuration
│   │   ├── conversation/      # Conversation management
│   │   ├── creative/          # Creative runtime
│   │   ├── db/                # Database and vector database management
│   │   ├── embedding/         # Embedding Provider
│   │   ├── http/              # HTTP server, middleware, chat handler
│   │   ├── knowledge/         # Knowledge-base scanning, indexing, search
│   │   ├── memory/            # History and long-term memory
│   │   ├── model/             # LLM Provider
│   │   ├── profile/           # User profile
│   │   ├── prompt/            # System prompts and tool prompts
│   │   ├── scheduler/         # Scheduled tasks and run records
│   │   ├── setting/           # Configuration storage and API
│   │   ├── skill/             # Skill loading, parsing, registration, execution
│   │   ├── sql/               # SQLite table schema and connection
│   │   ├── token/             # Token statistics and budget
│   │   ├── tool/              # Tool interface, registry, and built-in tools
│   │   └── websocket/         # WebSocket Hub, Client, Message
│   ├── pkg/
│   │   ├── config/
│   │   ├── context/
│   │   ├── graceful/
│   │   ├── handler/
│   │   ├── ioc/
│   │   ├── logger/
│   │   ├── schema/
│   │   └── websocket/
│   ├── data/                  # Runtime data, Skills, knowledge base, history
│   ├── sh/                    # Backend build scripts
│   ├── go.mod
│   └── go.sum
└── donkui/
    ├── lib/
    │   ├── app/
    │   │   ├── conf/          # Frontend service address and other config
    │   │   ├── init/          # App initialization
    │   │   ├── layout/        # Desktop layout
    │   │   └── router/        # GoRouter routes
    │   ├── common/
    │   │   ├── client/        # SSE/WebSocket/HTTP clients
    │   │   ├── model/         # Frontend data models
    │   │   ├── service/       # Settings, tasks, Skills, notifications, and other services
    │   │   └── widget/        # Common components
    │   ├── l10n/              # Localization strings
    │   └── ui/
    │       ├── home/
    │       ├── idea/
    │       ├── notification/
    │       ├── onboarding/
    │       ├── setting/
    │       └── task/
    ├── assets/
    ├── docs/                  # Frontend-backend protocol docs
    ├── scripts/               # Windows installer scripts
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
- Available LLM API Key
- Optional: Embedding API Key, used for knowledge-base retrieval and long-term memory
- Optional: Inno Setup, used to build the Windows installer

### 1. Clone the Project

```powershell
git clone https://github.com/longstageai/dank.git
cd dank
```

### 2. Start the Backend

```powershell
cd donkserv
go mod download
go run ./cmd
```

The service listens by default at:

```text
http://localhost:65434
```

Health check:

```powershell
curl http://localhost:65434/health
```

### 3. Start the Desktop Client

Open another terminal:

```powershell
cd donkui
flutter pub get
flutter run -d windows
```

The frontend default connection addresses are defined in:

```text
donkui/lib/app/conf/config.dart
```

Default values:

```text
API: http://localhost:65434/api/v1
SSE: http://localhost:65434/api/v1/chat
WebSocket: ws://localhost:65434/ws/events
```

The backend auto-start call in `donkui/lib/app/init/app.dart` is currently commented out. During development, start `donkserv` manually first.

### 4. Complete Configuration in the Desktop Client

Donk's model, Embedding, Agent, knowledge-base, and general switches are all configured in the desktop client.

On first launch, enter the onboarding page and fill in:

- LLM Provider, model name, API Key, Base URL
- Embedding Provider, model name, API Key, Base URL
- Agent runtime parameters

Later, you can continue adjusting them in `Settings`:

- `LLM Settings`: switch model provider, model, key, and endpoint address.
- `Embedding Settings`: configure the vector model for knowledge-base retrieval and long-term memory.
- `Agent Settings`: adjust Agent loop count, timeout, history, and token budget.
- `General Settings`: switch language, security protection, automatic knowledge-base building, and sleep prevention.
- `Usage Statistics`: view token usage and budget status.

## Configuration

Donk's runtime configuration is persisted by the configuration service. The desktop client reads and writes these configurations through the settings page. For normal users, all configuration entry points are in the frontend UI.

### Configuration Entry Points

| Page | Configuration | Purpose |
| --- | --- | --- |
| First-time onboarding | Basic LLM and Embedding parameters | Makes a usable model available to the Agent on first launch |
| LLM Settings | Provider, model, API Key, Base URL | Controls the model used by chat and Agent reasoning |
| Embedding Settings | Provider, model, API Key, Base URL | Controls semantic retrieval for knowledge bases and long-term memory |
| Agent Settings | Loop count, convergence parameters, timeout, history, token budget | Controls Agent behavior boundaries |
| General Settings | Language, security protection, automatic knowledge-base building, sleep prevention | Controls app-level switches |
| Usage Statistics | Token usage and budget status | Views model call consumption |

### Model Configuration

LLM configuration determines which model Donk uses for chat and Agent reasoning. The current backend adapter layer supports:

```text
openai
qwen
deepseek
doubao
```

After you fill in the provider, model name, API Key, and optional Base URL in desktop `Settings -> LLM Settings`, the configuration is written to the local database and later requests use the latest configuration.

### Embedding Configuration

Embedding configuration is used for knowledge-base retrieval, long-term memory, and semantic search. If you only use basic chat, you can skip Embedding at first. If you need knowledge-base and memory capabilities, complete the provider, model name, API Key, and Base URL in `Settings -> Embedding Settings`.

### Agent Configuration

Agent configuration is managed in `Settings -> Agent Settings` and mainly controls:

- `max_loop` controls the maximum loop count for a single Agent task.
- `converge_after` controls convergence-related behavior.
- `timeout` controls Agent runtime timeout.
- `history_max_entries` controls how many history entries are loaded.
- `history_max_days` controls history retention duration.
- `daily_token_limit` controls the daily token budget.

### Knowledge-Base Configuration

Knowledge-base switches and automatic build strategies are managed in `Settings -> General Settings` and knowledge-base-related configuration pages. The knowledge base depends on Embedding configuration. After it is enabled, local documents are scanned and searchable indexes are built.

Main parameters include:

- `enabled` controls whether automatic knowledge-base building is enabled.
- `interval` is the scan interval in seconds.
- `batch_size` is the number of files processed per batch.
- `sleep_ms` is the interval between batches to avoid excessive resource usage.
- `max_depth` is the directory scan depth.
- `max_file_size` is the single-file size limit.
- `directories` uses the default directory strategy when empty.

### Local Persistence

After the desktop client submits configuration, the backend saves it into the local SQLite database. After the app restarts, it reads the latest configuration from the database, so daily use only requires operating the frontend pages.

## HTTP API

Main backend APIs:

<details>
<summary>Expand to view API list</summary>

| Capability | Method and Path | Description |
| --- | --- | --- |
| Health check | `GET /health` | Checks whether the service is running |
| Streaming chat | `POST /api/v1/chat` | Agent SSE conversation |
| Full configuration | `GET /api/v1/config` | Gets full configuration |
| Full configuration | `PUT /api/v1/config` | Updates full configuration |
| LLM configuration | `GET /api/v1/config/llm` | Gets LLM configuration |
| LLM configuration | `PUT /api/v1/config/llm` | Updates LLM configuration |
| Embedding configuration | `GET /api/v1/config/embedding` | Gets Embedding configuration |
| Embedding configuration | `PUT /api/v1/config/embedding` | Updates Embedding configuration |
| Agent configuration | `GET /api/v1/config/agent` | Gets Agent configuration |
| Agent configuration | `PUT /api/v1/config/agent` | Updates Agent configuration |
| Knowledge-base configuration | `GET /api/v1/config/knowledge` | Gets knowledge-base configuration |
| Knowledge-base configuration | `PUT /api/v1/config/knowledge` | Updates knowledge-base configuration |
| Knowledge-base status | `GET /api/v1/knowledge/status` | Queries knowledge-base status |
| Knowledge-base control | `POST /api/v1/knowledge/start` | Starts the knowledge base |
| Knowledge-base control | `POST /api/v1/knowledge/stop` | Stops the knowledge base |
| Sleep status | `GET /api/v1/system/sleep` | Queries system sleep-prevention status |
| Prevent sleep | `POST /api/v1/system/sleep/prevent` | Prevents system sleep |
| Allow sleep | `POST /api/v1/system/sleep/allow` | Allows system sleep |
| Skill list | `GET /api/v1/skills` | Gets Skill list |
| Skill scan | `POST /api/v1/skills/rescan` | Rescans Skills |
| Skill detail | `GET /api/v1/skills/:name` | Gets a specified Skill |
| Skill delete | `DELETE /api/v1/skills/:name` | Deletes a specified Skill |
| Skill enable | `POST /api/v1/skills/:name/enable` | Enables a Skill |
| Skill disable | `POST /api/v1/skills/:name/disable` | Disables a Skill |
| Skill instructions | `GET /api/v1/skills/:name/instructions` | Gets Skill instructions |
| Skill scripts | `GET /api/v1/skills/:name/scripts` | Gets script list |
| Skill script content | `GET /api/v1/skills/:name/scripts/:script` | Gets script content |
| Create task | `POST /api/v1/tasks` | Creates a scheduled task |
| Task list | `GET /api/v1/tasks` | Queries task list |
| Task detail | `GET /api/v1/tasks/:id` | Queries task detail |
| Delete task | `DELETE /api/v1/tasks/:id` | Deletes a task |
| Cancel task | `POST /api/v1/tasks/:id/cancel` | Cancels a task |
| Trigger task | `POST /api/v1/tasks/:id/run` | Manually triggers a task |
| Task result | `GET /api/v1/tasks/:id/result` | Queries task result |
| Task runs | `GET /api/v1/tasks/:id/runs` | Queries task run records |
| Run list | `GET /api/v1/runs` | Queries all run records |
| Run detail | `GET /api/v1/runs/:id` | Queries run record detail |
| Delete run | `DELETE /api/v1/runs/:id` | Deletes a run record |
| Token usage | `GET /api/v1/tokens/usage` | Queries token usage list |
| Token budget | `GET /api/v1/tokens/budget` | Queries token budget status |
| Creative status | `GET /api/v1/creative/status` | Queries Creative status |
| Creative start | `POST /api/v1/creative/start` | Starts the Creative workflow |
| Creative stop | `POST /api/v1/creative/stop` | Stops the Creative workflow |
| WebSocket events | `GET /ws/events` | Notification and task event push |
| WebSocket test | `POST /ws/test-push` | Manually pushes a test notification |

</details>

For more detailed protocols, see:

```text
donkui/docs/agent_protocol.md
donkui/docs/websocket_protocol.md
donkui/docs/skill_api.md
donkui/docs/setting_api.md
donkui/docs/scheduler-api.md
donkui/docs/knowledge-base.md
```

## SSE Chat API Example

Request:

```http
POST /api/v1/chat
Content-Type: application/json
Accept: text/event-stream
```

Request body:

```json
{
  "content": "Help me summarize the tasks I need to handle today"
}
```

The server returns an SSE event stream. Typical events include:

```text
event: user_input
data: {...}

event: reasoning_delta
data: {...}

event: content_delta
data: {...}

event: tool_call
data: {...}

event: tool_result
data: {...}

event: assistant
data: {...}

event: done
data: {...}
```

## Skill Development

### Basic Structure

```text
my-skill/
├── SKILL.md
├── scripts/
│   └── main.py
├── references/
│   └── guide.md
└── assets/
```

### `SKILL.md` Example

```markdown
---
name: my-skill
description: A Skill for handling a specific category of tasks. Use it when the user needs to execute this type of task.
version: 1.0.0
---

# My Skill

When the user requests this category of task, read these instructions and execute `scripts/main.py`.

## Input

- task: The problem the user wants to handle

## Output

Returns structured processing results.
```

### Loading Flow

1. Put Skill files into `donkserv/data/skills`.
2. The backend scans Skills on startup.
3. Skill status is synced to the database.
4. Enabled Skills are registered into the Skill Registry.
5. The Agent uses the `skill` tool to read instructions, execute scripts, or obtain references.
6. After Skill files change, the Watcher attempts to sync them.

## Local Data

Donk stores runtime data in the backend data directory, for example:

```text
donkserv/data/db/
donkserv/data/history/
donkserv/data/knowledge/
donkserv/data/skills/
donkserv/data/script_runtime/
```

These directories may contain:

- SQLite databases
- Conversation history
- User memory
- Knowledge-base indexes
- Vector data
- Local Skills
- Python runtime and dependencies

If the project is open-sourced or published to GitHub, avoid committing real runtime data, user files, databases, vector databases, and API keys.

## Build

### Backend Build

```powershell
cd donkserv
.\sh\build.bat
```

Output:

```text
donkserv/sh/donk.exe
```

You can also use Go directly:

```powershell
cd donkserv
go build -ldflags="-s -w" -o sh\donk.exe ./cmd/...
```

### Flutter Windows Build

```powershell
cd donkui
flutter pub get
flutter build windows --release
```

### Installer Build

The installer script is located at:

```text
donkui/scripts/build_installer.bat
```

Run:

```powershell
cd donkui
.\scripts\build_installer.bat
```

This script depends on Inno Setup and checks for:

```text
donkui/server/donk.exe
```

It is used to distribute the backend service together with the desktop client. Before publishing, build the backend first and place it where the installer script expects it.

## Development Notes

### Backend Development

Common commands:

```powershell
cd donkserv
go test ./...
go run ./cmd
```

Modules to focus on:

- Adding a model: modify `internal/model` and necessary configuration structures.
- Adding a tool: implement the tool in `internal/tool/builtin`, then register it with the tool registry.
- Adding an API: register routes in the corresponding module Handler.
- Adding knowledge-base capabilities: modify `internal/knowledge`.
- Adding Skill behavior: modify `internal/skill` or the built-in `SkillTool`.

### Frontend Development

Common commands:

```powershell
cd donkui
flutter pub get
flutter run -d windows
```

Modules to focus on:

- Service address: `lib/app/conf/config.dart`
- Routes: `lib/app/router/routes.dart`
- Main chat: `lib/ui/home`
- Settings page: `lib/ui/setting`
- Task page: `lib/ui/task`
- Notification page: `lib/ui/notification`
- Frontend-backend service wrappers: `lib/common/service`
- SSE and WebSocket clients: `lib/common/client`

## Runtime Notes

- The current desktop window default size and minimum size are both `960x720`.
- The Flutter client uses a single-instance service, so a repeated launch exits the new instance.
- Closing the window hides it to the tray by default and does not mean the backend has stopped.
- In development mode, the backend service must be started manually.
- The backend port is currently fixed at `65434`.
- If the knowledge base or long-term memory is unavailable, the usual cause is missing Embedding configuration or initialization failure.
- If chat does not respond, first check the LLM Provider, API Key, Base URL, and model name.
- If a Skill does not appear, check the `data/skills` directory, `SKILL.md` frontmatter, and enabled status.

## Security Notes

Donk has strong local execution capabilities, including command execution, file reading and writing, HTTP requests, browser control, and script execution. Pay attention to the following when using and extending it:

- Do not load Skills from untrusted sources.
- Do not write real API keys into public repositories.
- Do not commit runtime data such as `data/db`, `data/history`, and `data/knowledge`.
- Be cautious with command execution, file writing, and script execution tools.
- Before exposing the backend port to external networks, enable authentication and restrict access sources.

## Known Status

This repository is still in rapid development, and the code contains some historical traces:

- Some comments have encoding display issues.
- `donkserv/data` may contain local runtime data and example Skills.
- The frontend logic for automatically starting the backend process is currently commented out.
- Configuration entry points are based on the desktop first-time onboarding and settings center. After app restart, saved local configuration continues to be used.

These do not affect understanding the overall architecture, but before formal open sourcing, it is recommended to further clean runtime data, example keys, and encoding issues.

## License

This project uses the GPL-3.0 License. See [LICENSE](LICENSE) for details.
