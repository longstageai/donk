# Skill 管理 API

Skill 管理 API 用于查看、启用、禁用、删除和重新扫描本地 Skill。Skill 文件存放在 `data/skills`，基础信息来自 `SKILL.md`，启用状态保存在 SQLite。

## 基础地址

```text
http://localhost:65434/api/v1/skills
```

## 接口列表

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/skills` | 获取 Skill 列表 |
| `POST` | `/api/v1/skills/rescan` | 重新扫描文件系统 |
| `GET` | `/api/v1/skills/:name` | 获取 Skill 详情 |
| `DELETE` | `/api/v1/skills/:name` | 删除 Skill |
| `POST` | `/api/v1/skills/:name/enable` | 启用 Skill |
| `POST` | `/api/v1/skills/:name/disable` | 禁用 Skill |
| `GET` | `/api/v1/skills/:name/instructions` | 获取 Skill 指令文本 |
| `GET` | `/api/v1/skills/:name/scripts` | 获取脚本列表 |
| `GET` | `/api/v1/skills/:name/scripts/:script` | 获取脚本内容 |

## 获取列表

```http
GET /api/v1/skills
```

响应：

```json
{
  "data": [
    {
      "name": "desktop-path-getter",
      "description": "获取桌面路径",
      "version": "1.0.0",
      "enabled": true,
      "path": "data/skills/desktop-path-getter",
      "has_scripts": true,
      "has_references": false,
      "has_assets": false
    }
  ],
  "total": 1
}
```

列表排序规则：启用的 Skill 在前，同状态下按创建时间倒序。

## 获取详情

```http
GET /api/v1/skills/:name
```

未找到时返回：

```json
{
  "error": "skill not found"
}
```

## 启用和禁用

```http
POST /api/v1/skills/:name/enable
POST /api/v1/skills/:name/disable
```

响应：

```json
{
  "message": "Skill 已启用"
}
```

## 删除

```http
DELETE /api/v1/skills/:name
```

响应：

```json
{
  "message": "Skill 已删除"
}
```

## 重新扫描

```http
POST /api/v1/skills/rescan
```

用于新增、删除或修改 Skill 文件后手动同步。

响应：

```json
{
  "message": "扫描完成"
}
```

## 指令与脚本

```http
GET /api/v1/skills/:name/instructions
GET /api/v1/skills/:name/scripts
GET /api/v1/skills/:name/scripts/:script
```

脚本内容响应：

```json
{
  "name": "main.py",
  "content": "print('hello')"
}
```

## Skill 目录结构

```text
data/skills/
└── skill-name/
    ├── SKILL.md
    ├── scripts/
    ├── references/
    └── assets/
```

## 注意事项

- Skill 名称来自 `SKILL.md` frontmatter。
- 删除接口会删除 Skill 文件和数据库状态。
- 文件监听器会尝试自动同步变更，但开发调试时可以调用 `POST /api/v1/skills/rescan`。
