# Donk

English | [简体中文](README_CH.md)

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![Flutter](https://img.shields.io/badge/Flutter-3.7+-02569B?logo=flutter&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows-lightgrey)

# Donk - Your Fully Automated Local AI Work Partner

**An ongoing bold experiment: when AI has complete context and enough capability, how much value can it create for you autonomously?**

---

## Why Are We Building Donk?

Have you ever had these experiences?

- You downloaded a pile of AI tools, opened them, and stared at an empty input box.
- You know AI is powerful, but you still do not know what it can actually do for you.
- Every time, you have to rack your brain for prompts, and the result is still unsatisfying.
- You watch others multiply their productivity with AI, while you cannot even get past the first step.

**This is not your problem. It is the problem with current AI products.**

They all push the hardest question, "how to use AI", onto ordinary users.

Donk exists to solve this. We believe that **the best AI is the AI you can barely feel.**

---

## What Is Donk?

Donk is not another chat window. It is a **fully automated AI work partner that quietly runs in the background on your computer**.

You only need to do three things:

1. Download and install it.
2. Tell it who you are and what you care about.
3. Wait for results.

Then Donk will:

- Automatically learn your work habits and knowledge system.
- Proactively discover where you may need help.
- Decide which tools to call and which tasks to execute.
- Arrange its own work schedule.
- Notify you only when it truly needs you.

It is like a tireless, highly powered donkey working out of sight, handling tedious, repetitive work and tasks you may not even have realized could be automated.

---

## An Open-Source Social Experiment at Scale

Donk is not just a software product. It is also a **scientific experiment about the boundaries of AI capability**.

We are testing a radical hypothesis:

> **When an AI Agent has enough local context, full tool permissions, and unlimited runtime, can it autonomously evolve reusable capabilities and work scenarios that humans have never imagined?**

This experiment needs your participation. Every user who downloads and uses Donk, every task it completes autonomously, and every workflow it invents helps answer this question.

We will spend a huge number of tokens and record countless attempts and failures to explore one possibility: **can AI truly become an extension of humans, instead of another tool that humans must learn how to use?**

---

## What Donk Can Do Now and Later

- **Local knowledge base**: automatically index documents on your computer without uploading them to the cloud.
- **Long-term memory**: remember what you said and every decision you made.
- **Tool calling**: automatically use the browser, file system, terminal, and other tools.
- **Skill extensions**: support custom scripts so capabilities can keep expanding.
- **Task scheduling**: schedule tasks and automatically execute them at the right time.
- **Real-time notifications**: interrupt you only when your decision is needed.
- **Desktop integration**: deeply integrate with Windows and become part of your computer.

---

## Quick Start

1. Download the latest version from the [Releases](https://github.com/your-username/donk/releases) page.
2. Double-click the installer and follow the wizard to complete initial setup.
3. Tell Donk who you are and what you want it to help with.
4. Minimize the window and return to the work that matters.
5. Wait for it to surprise you.

---

## Join the Experiment

Donk is currently in an early experimental stage. It may make mistakes, do silly things, or even do nothing at all. That is exactly what makes the experiment meaningful.

If you are curious about the future of AI, tired of "prompt engineering", and believe AI should work for us rather than the other way around:

- Give us a star so more people can see this experiment.
- Submit issues and tell us what interesting or foolish things it did.
- Share your ideas and tell us what you want it to do for you.
- Submit PRs to help it become smarter and more capable.

**Our goal is not to build a perfect product, but to explore the unlimited possibilities of AI together.**

---

## Experimental Statement

> "Donk is an experimental project. It may produce unexpected results, consume a large amount of compute resources, or fail completely. But if we succeed, we will redefine the relationship between humans and AI."
>
> -- The Donk Experiment Team

---

*Donk - Let AI work for you, instead of you working for AI.*

## Table of Contents

- [Screenshots](#screenshots)
- [Use Cases](#use-cases)
- [Core Features](#core-features)
- [Current Capabilities](#current-capabilities)
- [Architecture Overview](#architecture-overview)
- [Tech Stack](#tech-stack)
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
| Local personal AI assistant | Conversation management, long-term memory, knowledge base, and daily task handling |
| Agent tool platform | Registers file, command, browser, HTTP, document parsing, and knowledge search capabilities as tools |
| Skill runtime | Extends vertical task capabilities through `SKILL.md`, scripts, and reference materials |
| Automation task entry point | Creates one-time, delayed, or Cron-scheduled tasks with natural language |
| Desktop Agent experiment | Frontend/backend separation for replacing models, tools, workflows, and UI |

## Core Features

- **Local-first**: configuration, sessions, tasks, knowledge base, skills, and runtime state are stored locally.
- **Desktop configuration**: models, embeddings, agents, knowledge base, and general switches are configured in the frontend.
- **Streaming Agent chat**: returns reasoning, content, tool calls, tool results, and other events over SSE.
- **Extensible tool system**: includes built-in tools for files, commands, HTTP, browser control, document parsing, knowledge base, tasks, and more.
- **Pluggable Skills**: extends Agent capabilities with `SKILL.md`, scripts, reference materials, and dependency declarations.
- **Knowledge base and long-term memory**: builds local semantic retrieval using embeddings and vector storage.
- **Task scheduling and notifications**: supports background tasks, run records, and WebSocket real-time notifications.

## Screenshots

The main interface centers on conversation. The left side provides navigation, the middle displays the Agent's streaming replies, thinking state, and action buttons, and the bottom input area supports text input, attachments, mentions, and shortcuts.

![Donk main chat interface](docs/img/Snipaste_2026-05-27_14-32-01.png)

| Idea Plaza | Task Management |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-32-40.png" alt="Donk Idea Plaza" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-32-52.png" alt="Donk Task Management" width="420"> |
| Manage local Skills and quickly view the Agent's available capabilities. | View background tasks, run status, schedule times, and enabled state. |

| Notifications | Settings |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-33-15.png" alt="Donk Notifications" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-33-53.png" alt="Donk Settings" width="420"> |
| Centralized display for task deliveries, system status, and background events. | Configure LLM, embedding, Agent, knowledge base, and general switches. |

| WeChat Connection | About |
| --- | --- |
| <img src="docs/img/Snipaste_2026-05-27_14-33-33.png" alt="Donk WeChat Connection" width="420"> | <img src="docs/img/Snipaste_2026-05-27_14-34-07.png" alt="Donk About Page" width="420"> |
| View WeChat connection status and disconnect when needed. | Display application name, icon, and version information. |

## Current Capabilities

### Agent Chat

- Provides SSE streaming responses through `POST /api/v1/chat`.
- Supports user input confirmation, reasoning deltas, content deltas, complete responses, tool calls, tool results, warnings, and done events.
- Supports loading history, retrieving long-term memory, user profiles, and token statistics.
- The Agent can access built-in tools and Skill tools through the tool registry.

### Models and Embeddings

- The LLM provider adapter layer is in `donkserv/internal/model`.
- Current code includes adapters for `openai`, `qwen`, `deepseek`, and `doubao`.
- The embedding adapter layer is in `donkserv/internal/embedding`.
- LLM and embedding configuration is persisted by the configuration service and can be managed from the desktop settings page.

### Tool System

Built-in tools live in `donkserv/internal/tool/builtin` and include, but are not limited to:

- File reading and writing
- Command execution
- HTTP requests
- Browser control
- Calculator
- PDF parsing
- Word parsing
- Knowledge base search
- Long-term memory save and search
- Conversation history search
- Task management
- Skill invocation
- Skill creation
- Skill installation
- Python script execution
- Python dependency management

Tools are registered into `tool.Registry` and then made available for Agent decision-making.

### Skill Extensions

Donk supports loading Skills from a local directory:

```text
donkserv/data/skills
```

A typical Skill contains:

```text
skill-name/
├── SKILL.md
├── scripts/
├── references/
└── assets/
```

Skill capabilities include:

- Reading capability descriptions and trigger semantics from `SKILL.md`.
- Loading scripts and reference materials.
- Registering enabled Skills as Agent-callable capabilities.
- Querying, enabling, disabling, deleting, and rescanning through APIs.
- Automatically syncing changes through file watching.
- Supporting script runtimes and dependency declarations.

### Knowledge Base

The knowledge base module is in `donkserv/internal/knowledge`. It scans, indexes, and semantically retrieves local documents. It combines embeddings and vector storage to turn local documents into searchable Agent context.

Supported capabilities include:

- Scheduled scanning of local directories.
- Configurable scan depth, batch size, maximum file size, and scan interval.
- Hot/cold data tiering for knowledge files.
- Semantic retrieval through the `knowledge_search` tool.
- Start, stop, and status query through configuration APIs.

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
- Shared history and profile context between the Agent and Creative workflows.

### Task Scheduling

The scheduler is in `donkserv/internal/scheduler` and supports:

- One-time tasks
- Delayed tasks
- Cron recurring tasks
- Agent task executor
- Task run records
- Cancel, trigger, delete, and query operations
- WebSocket event push

The Agent can create background tasks through task tools for workflows such as "remind me later", "run this every day", and "organize these resources on a schedule".

### Creative Workflow

The Creative module is in:

```text
donkserv/internal/creative
```

It provides a more complex runtime than a normal single-turn Agent, suitable for goal decomposition, idea generation, task planning, and staged execution. The current startup process registers default LLM Agents and connects the scheduler, knowledge base, long-term memory, user profile, and WebSocket hooks.

### Desktop Client

The Flutter client is in `donkui`. Main pages include:

- `home`: main chat interface
- `idea`: Skill and idea-related interfaces
- `task`: tasks and run records
- `notification`: notification center
- `setting`: model, embedding, Agent, knowledge base, and app settings
- `onboarding`: first-run configuration guide

The client uses:

- `GetX` for state and dependency management
- `GoRouter` for routing
- `window_manager` for desktop window management
- `tray_manager` for tray interaction
- SSE client for chat streams
- WebSocket client for notification events

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

A typical chat request data flow:

1. Flutter sends a user message through `POST /api/v1/chat`.
2. The Go backend creates the Agent request context.
3. The Agent reads configuration, history, long-term memory, user profile, and available tools.
4. The LLM produces streaming output or tool-call intent.
5. The tool registry executes tools such as files, commands, knowledge base, Skills, and tasks.
6. The backend continuously returns events over SSE.
7. If background tasks or notifications are produced, WebSocket pushes them to the desktop notification center.

## Tech Stack

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
| Desktop capability | window_manager, tray_manager |
| Models | OpenAI, Qwen, DeepSeek, Doubao provider adapters |

## Directory Structure

<details>
<summary>Expand repository structure</summary>

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
│   │   ├── knowledge/         # Knowledge base scan, index, search
│   │   ├── memory/            # History and long-term memory
│   │   ├── model/             # LLM Provider
│   │   ├── profile/           # User profile
│   │   ├── prompt/            # System prompts and tool prompts
│   │   ├── scheduler/         # Scheduled tasks and run records
│   │   ├── setting/           # Configuration storage and API
│   │   ├── skill/             # Skill loading, parsing, registration, execution
│   │   ├── sql/               # SQLite table schema and connection
│   │   ├── token/             # Token statistics and budget
│   │   ├── tool/              # Tool interfaces, registry, and built-in tools
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
│   ├── data/                  # Runtime data, skills, knowledge base, history
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
    │   │   ├── service/       # Settings, tasks, skills, notifications, and other services
    │   │   └── widget/        # Shared widgets
    │   ├── l10n/              # Localization strings
    │   └── ui/
    │       ├── home/
    │       ├── idea/
    │       ├── notification/
    │       ├── onboarding/
    │       ├── setting/
    │       └── task/
    ├── assets/
    ├── docs/                  # Frontend/backend protocol docs
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
- Available LLM API key
- Optional: embedding API key for knowledge base retrieval and long-term memory
- Optional: Inno Setup for building the Windows installer

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

The service listens on:

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

The automatic backend process startup call in `donkui/lib/app/init/app.dart` is currently commented out. During development, start `donkserv` manually first.

### 4. Complete Configuration in the Desktop Client

Donk's model, embedding, Agent, knowledge base, and general switches are configured in the desktop client.

On first launch, the onboarding page will guide you through:

- LLM provider, model name, API key, and Base URL
- Embedding provider, model name, API key, and Base URL
- Agent runtime parameters

You can continue adjusting them later in `Settings`:

- `LLM Settings`: switch model provider, model, key, and endpoint.
- `Embedding Settings`: configure vector models for knowledge base retrieval and long-term memory.
- `Agent Settings`: adjust Agent loop count, timeout, history, and token budget.
- `General Settings`: switch language, security guard, knowledge base auto-build, and sleep prevention.
- `Usage Statistics`: view token usage and budget status.

## Configuration

Donk runtime configuration is persisted by the configuration service. The desktop client reads and writes it through the settings pages. For normal users, all configuration entry points are in the frontend.

### Configuration Entry Points

| Page | Configuration | Purpose |
| --- | --- | --- |
| Onboarding | Basic LLM and embedding parameters | Gives the Agent usable models on first startup |
| LLM Settings | Provider, model, API key, Base URL | Controls the model used for chat and Agent reasoning |
| Embedding Settings | Provider, model, API key, Base URL | Controls semantic retrieval for knowledge base and long-term memory |
| Agent Settings | Loop count, convergence parameters, timeout, history, token budget | Controls Agent behavior boundaries |
| General Settings | Language, security guard, knowledge base auto-build, sleep prevention | Controls application-level switches |
| Usage Statistics | Token usage and budget status | Shows model call consumption |

### Model Configuration

LLM configuration determines which model Donk uses for chat and Agent reasoning. The current backend adapter layer supports:

```text
openai
qwen
deepseek
doubao
```

After you fill in the provider, model name, API key, and optional Base URL in `Settings -> LLM Settings`, the configuration is written to the local database and subsequent requests use the latest configuration.

### Embedding Configuration

Embedding configuration is used for knowledge base retrieval, long-term memory, and semantic search. If you only use basic chat, you can skip embedding configuration at first. If you need knowledge base and memory features, complete the provider, model name, API key, and Base URL in `Settings -> Embedding Settings`.

### Agent Configuration

Agent configuration is managed in `Settings -> Agent Settings` and mainly controls:

- `max_loop`: maximum loop count for a single Agent task.
- `converge_after`: behavior related to convergence detection.
- `timeout`: Agent runtime timeout.
- `history_max_entries`: number of history entries to load.
- `history_max_days`: history retention period.
- `daily_token_limit`: daily token budget.

### Knowledge Base Configuration

Knowledge base switches and auto-build strategy are managed in `Settings -> General Settings` and related knowledge base configuration pages. The knowledge base depends on embedding configuration. After it is enabled, Donk scans local documents and builds searchable indexes.

Main parameters include:

- `enabled`: whether to enable knowledge base auto-build.
- `interval`: scan interval in seconds.
- `batch_size`: number of files processed per batch.
- `sleep_ms`: delay between batches to avoid excessive resource usage.
- `max_depth`: directory scan depth.
- `max_file_size`: maximum size for a single file.
- `directories`: uses the default directory strategy when empty.

### Local Persistence

After the desktop client submits configuration, the backend saves it to the local SQLite database. On restart, the app reads the latest configuration from the database, so day-to-day use only requires operating the frontend pages.

## HTTP API

Main backend APIs:

<details>
<summary>Expand API list</summary>

| Capability | Method and Path | Description |
| --- | --- | --- |
| Health check | `GET /health` | Checks whether the service is running |
| Streaming chat | `POST /api/v1/chat` | Agent SSE conversation |
| Full configuration | `GET /api/v1/config` | Gets full configuration |
| Full configuration | `PUT /api/v1/config` | Updates full configuration |
| LLM configuration | `GET /api/v1/config/llm` | Gets LLM configuration |
| LLM configuration | `PUT /api/v1/config/llm` | Updates LLM configuration |
| Embedding configuration | `GET /api/v1/config/embedding` | Gets embedding configuration |
| Embedding configuration | `PUT /api/v1/config/embedding` | Updates embedding configuration |
| Agent configuration | `GET /api/v1/config/agent` | Gets Agent configuration |
| Agent configuration | `PUT /api/v1/config/agent` | Updates Agent configuration |
| Knowledge configuration | `GET /api/v1/config/knowledge` | Gets knowledge base configuration |
| Knowledge configuration | `PUT /api/v1/config/knowledge` | Updates knowledge base configuration |
| Knowledge status | `GET /api/v1/knowledge/status` | Queries knowledge base status |
| Knowledge control | `POST /api/v1/knowledge/start` | Starts the knowledge base |
| Knowledge control | `POST /api/v1/knowledge/stop` | Stops the knowledge base |
| Sleep status | `GET /api/v1/system/sleep` | Queries system sleep prevention status |
| Prevent sleep | `POST /api/v1/system/sleep/prevent` | Prevents system sleep |
| Allow sleep | `POST /api/v1/system/sleep/allow` | Allows system sleep |
| Skill list | `GET /api/v1/skills` | Gets Skill list |
| Skill rescan | `POST /api/v1/skills/rescan` | Rescans Skills |
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
| Creative start | `POST /api/v1/creative/start` | Starts Creative workflow |
| Creative stop | `POST /api/v1/creative/stop` | Stops Creative workflow |
| WebSocket events | `GET /ws/events` | Pushes notification and task events |
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
description: A Skill for handling a specific category of tasks. Use it when the user needs this kind of task.
version: 1.0.0
---

# My Skill

When the user requests this category of task, read these instructions and execute `scripts/main.py`.

## Input

- task: the problem the user wants to handle

## Output

Returns structured processing results.
```

### Loading Flow

1. Put Skill files into `donkserv/data/skills`.
2. The backend scans Skills on startup.
3. Skill state is synced to the database.
4. Enabled Skills are registered into the Skill Registry.
5. The Agent uses the `skill` tool to read instructions, execute scripts, or fetch reference materials.
6. After Skill files change, the watcher attempts to sync them.

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
- Knowledge base indexes
- Vector data
- Local Skills
- Python runtime and dependencies

If the project is open-sourced or published to GitHub, avoid committing real runtime data, user files, databases, vector stores, and API keys.

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

It is used to distribute the backend service together with the desktop client. Before release, build the backend first and place it where the installer script expects it.

## Development Notes

### Backend Development

Common commands:

```powershell
cd donkserv
go test ./...
go run ./cmd
```

Key modules:

- Add a model: modify `internal/model` and required configuration structures.
- Add a tool: implement it in `internal/tool/builtin`, then register it in the tool registry.
- Add an API: register routes in the corresponding module handler.
- Add knowledge base capability: modify `internal/knowledge`.
- Add Skill behavior: modify `internal/skill` or the built-in `SkillTool`.

### Frontend Development

Common commands:

```powershell
cd donkui
flutter pub get
flutter run -d windows
```

Key modules:

- Service address: `lib/app/conf/config.dart`
- Routes: `lib/app/router/routes.dart`
- Main chat: `lib/ui/home`
- Settings page: `lib/ui/setting`
- Task page: `lib/ui/task`
- Notification page: `lib/ui/notification`
- Frontend/backend service wrappers: `lib/common/service`
- SSE and WebSocket clients: `lib/common/client`

## Runtime Notes

- The current desktop window default size and minimum size are both `960x720`.
- The Flutter client uses a single-instance service. When launched repeatedly, the new instance exits.
- Closing the window hides it to the tray by default; this is not the same as stopping the backend.
- In development mode, the backend service must be started manually.
- The backend port is currently fixed at `65434`.
- If the knowledge base or long-term memory is unavailable, the usual cause is missing embedding configuration or initialization failure.
- If chat does not respond, first check LLM provider, API key, Base URL, and model name.
- If a Skill does not appear, check the `data/skills` directory, `SKILL.md` frontmatter, and enabled state.

## Security Notes

Donk has powerful local execution capabilities, including command execution, file read/write, HTTP requests, browser control, and script execution. Be careful when using and extending it:

- Do not load Skills from untrusted sources.
- Do not commit real API keys to public repositories.
- Do not commit runtime data such as `data/db`, `data/history`, and `data/knowledge`.
- Be cautious with command execution, file writing, and script execution tools.
- Before exposing the backend port to external networks, enable authentication and restrict access sources.

## Known Status

This repository is still under rapid development and contains some historical traces:

- Some comments have encoding display issues.
- `donkserv/data` may contain local runtime data and example Skills.
- The frontend logic for automatically starting the backend process is currently commented out.
- Configuration entry points are based on the desktop onboarding flow and settings center. After restart, the app continues using saved local configuration.

These do not affect understanding the overall architecture, but before formal open source release, runtime data, example secrets, and encoding issues should be cleaned up further.

## License

This project uses the GPL-3.0 License. See [LICENSE](LICENSE) for details.
