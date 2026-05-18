# Skill 管理 API 文档

本文档介绍 donk Skill 管理系统的 RESTful API 使用方法。

## 概述

Skill 管理 API 提供对 Skill 的查询、启用/禁用、删除等操作。Skill 的基本信息存储在文件系统中（SKILL.md），启用状态存储在数据库中。

**基础 URL**: `http://localhost:8080/api/v1/skills`

---

## API 列表

### 1. 获取 Skill 列表

获取所有 Skill 的基本信息（包括启用状态）。

**请求**
```http
GET /api/v1/skills
```

**响应示例**
```json
{
  "data": [
    {
      "name": "test-skill",
      "description": "一个用于测试的示例 Skill，演示了 Skill 系统的基本功能",
      "version": "1.0.0",
      "author": "donk Team",
      "tags": ["test", "example", "demo"],
      "enabled": true,
      "user_invocable": true,
      "disable_model_invocation": false,
      "path": "data/skills/test-skill",
      "has_scripts": true,
      "has_references": false,
      "has_assets": false
    }
  ],
  "total": 1
}
```

**字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | Skill 名称（唯一标识） |
| description | string | Skill 描述 |
| version | string | 版本号 |
| author | string | 作者 |
| tags | array | 标签列表 |
| enabled | boolean | 是否启用 |
| user_invocable | boolean | 是否允许用户通过斜杠命令调用 |
| disable_model_invocation | boolean | 是否禁止自动触发 |
| path | string | Skill 目录路径 |
| has_scripts | boolean | 是否包含 scripts 目录 |
| has_references | boolean | 是否包含 references 目录 |
| has_assets | boolean | 是否包含 assets 目录 |

---

### 2. 获取 Skill 详情

获取指定 Skill 的详细信息。

**请求**
```http
GET /api/v1/skills/:name
```

**示例**
```bash
curl http://localhost:8080/api/v1/skills/test-skill
```

**响应示例**
```json
{
  "name": "test-skill",
  "description": "一个用于测试的示例 Skill...",
  "version": "1.0.0",
  "author": "donk Team",
  "tags": ["test", "example", "demo"],
  "enabled": true,
  "user_invocable": true,
  "disable_model_invocation": false,
  "path": "data/skills/test-skill",
  "has_scripts": true,
  "has_references": false,
  "has_assets": false
}
```

---

### 3. 启用 Skill

启用指定的 Skill，启用后 Agent 可以加载和使用该 Skill。

**请求**
```http
POST /api/v1/skills/:name/enable
```

**示例**
```bash
curl -X POST http://localhost:8080/api/v1/skills/test-skill/enable
```

**响应示例**
```json
{
  "message": "Skill 已启用"
}
```

---

### 4. 禁用 Skill

禁用指定的 Skill，禁用后 Agent 无法加载该 Skill。

**请求**
```http
POST /api/v1/skills/:name/disable
```

**示例**
```bash
curl -X POST http://localhost:8080/api/v1/skills/test-skill/disable
```

**响应示例**
```json
{
  "message": "Skill 已禁用"
}
```

---

### 5. 删除 Skill

删除指定的 Skill，**会同时删除文件系统目录和数据库记录**。

**警告**: 此操作不可恢复！

**请求**
```http
DELETE /api/v1/skills/:name
```

**示例**
```bash
curl -X DELETE http://localhost:8080/api/v1/skills/test-skill
```

**响应示例**
```json
{
  "message": "Skill 已删除"
}
```

---

### 6. 重新扫描文件系统

扫描 `data/skills` 目录，将新发现的 Skill 同步到数据库（默认启用）。

**请求**
```http
POST /api/v1/skills/rescan
```

**示例**
```bash
curl -X POST http://localhost:8080/api/v1/skills/rescan
```

**响应示例**
```json
{
  "message": "扫描完成"
}
```

**说明**
- 新发现的 Skill 会自动插入数据库，默认状态为启用
- 已存在的 Skill 会更新描述信息（如果 SKILL.md 有变化）

---

### 7. 获取 Skill 指令

获取 Skill 的完整指令内容（Markdown 格式）。

**请求**
```http
GET /api/v1/skills/:name/instructions
```

**示例**
```bash
curl http://localhost:8080/api/v1/skills/test-skill/instructions
```

**响应示例**
```json
{
  "instructions": "# Test Skill\n\n这是一个用于测试的 Skill..."
}
```

---

### 8. 获取脚本列表

获取指定 Skill 的 scripts 目录下的所有脚本文件。

**请求**
```http
GET /api/v1/skills/:name/scripts
```

**示例**
```bash
curl http://localhost:8080/api/v1/skills/test-skill/scripts
```

**响应示例**
```json
{
  "scripts": ["hello.py", "info.ps1"]
}
```

---

### 9. 获取脚本内容

获取指定脚本的内容。

**请求**
```http
GET /api/v1/skills/:name/scripts/:script
```

**示例**
```bash
curl http://localhost:8080/api/v1/skills/test-skill/scripts/hello.py
```

**响应示例**
```json
{
  "name": "hello.py",
  "content": "#!/usr/bin/env python3\n# -*- coding: utf-8 -*-\n..."
}
```

---

## 使用场景示例

### 场景 1：查看所有可用 Skill

```bash
# 获取列表
curl http://localhost:8080/api/v1/skills | jq

# 输出：
# {
#   "data": [...],
#   "total": 5
# }
```

### 场景 2：临时禁用某个 Skill

```bash
# 禁用
curl -X POST http://localhost:8080/api/v1/skills/test-skill/disable

# 验证
curl http://localhost:8080/api/v1/skills/test-skill | jq '.enabled'
# 输出：false
```

### 场景 3：重新启用 Skill

```bash
# 启用
curl -X POST http://localhost:8080/api/v1/skills/test-skill/enable

# 验证
curl http://localhost:8080/api/v1/skills/test-skill | jq '.enabled'
# 输出：true
```

### 场景 4：添加新 Skill 后同步

```bash
# 1. 将新 Skill 复制到 data/skills/ 目录
# cp -r my-new-skill data/skills/

# 2. 重新扫描
curl -X POST http://localhost:8080/api/v1/skills/rescan

# 3. 验证新 Skill 已加载
curl http://localhost:8080/api/v1/skills/my-new-skill
```

### 场景 5：查看 Skill 脚本

```bash
# 获取脚本列表
curl http://localhost:8080/api/v1/skills/test-skill/scripts | jq

# 获取脚本内容
curl http://localhost:8080/api/v1/skills/test-skill/scripts/hello.py | jq '.content'
```

---

## 错误处理

所有 API 在出错时返回以下格式：

```json
{
  "error": "错误描述"
}
```

**常见错误码**

| HTTP 状态码 | 说明 |
|------------|------|
| 404 | Skill 不存在 |
| 500 | 服务器内部错误 |

---

## 数据存储说明

### 文件系统（主存储）

```
data/skills/
├── test-skill/
│   ├── SKILL.md          # 元数据和指令
│   ├── scripts/          # 可执行脚本
│   ├── references/       # 参考资料
│   └── assets/           # 资源文件
└── another-skill/
    └── ...
```

### 数据库（状态存储）

```sql
-- skill_states 表
name        TEXT PRIMARY KEY      -- Skill 名称
description TEXT                  -- 描述（缓存）
enabled     BOOLEAN DEFAULT 1     -- 启用状态
created_at  DATETIME              -- 创建时间
updated_at  DATETIME              -- 更新时间
```

**同步逻辑**
- 启动时自动扫描文件系统，同步到数据库
- 新 Skill 默认启用
- 已存在 Skill 只更新描述
- 启用/禁用只修改数据库，不影响文件系统

---

## 注意事项

1. **启用/禁用实时生效**：修改后 Agent 会立即感知（通过 Registry 动态加载/卸载）
2. **删除不可恢复**：DELETE 操作会永久删除文件系统目录
3. **名称唯一性**：Skill 名称由目录名决定，不可重复
4. **脚本执行权限**：API 只返回脚本内容，执行需通过 Agent 或 Executor
