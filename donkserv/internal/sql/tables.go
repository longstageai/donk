package sql

// TableSchemas 数据库表结构定义
// 统一管理所有数据库表的创建语句，确保表结构一致性和可维护性
var TableSchemas = []string{
	// ============================================
	// 系统配置表
	// 存储LLM、Embedding、Agent等核心配置参数
	// ============================================
	`CREATE TABLE IF NOT EXISTS config (
		-- 主键
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		-- LLM配置
		llm_provider TEXT NOT NULL DEFAULT 'openai',           -- LLM提供商: openai/qwen/deepseek/doubao
		llm_model TEXT NOT NULL DEFAULT 'gpt-4o-mini',         -- LLM模型名称
		llm_api_key TEXT NOT NULL DEFAULT '',                  -- API密钥
		llm_base_url TEXT NOT NULL DEFAULT '',                 -- API基础地址
		llm_temperature REAL NOT NULL DEFAULT 0.7,             -- 温度参数(0-2)
		llm_max_tokens INTEGER NOT NULL DEFAULT 4096,          -- 最大Token数
		-- Embedding配置
		embedding_provider TEXT NOT NULL DEFAULT 'openai',     -- Embedding提供商
		embedding_model TEXT NOT NULL DEFAULT 'text-embedding-3-small', -- Embedding模型
		embedding_api_key TEXT NOT NULL DEFAULT '',            -- Embedding API密钥
		embedding_base_url TEXT NOT NULL DEFAULT '',           -- Embedding基础地址
		embedding_dimension INTEGER NOT NULL DEFAULT 1536,     -- 向量维度
		-- Agent配置
		agent_name TEXT NOT NULL DEFAULT 'donk',              -- Agent名称
		agent_max_loop INTEGER NOT NULL DEFAULT 10,            -- 最大循环次数
		agent_converge_after INTEGER NOT NULL DEFAULT 3,       -- 收敛阈值
		agent_timeout INTEGER NOT NULL DEFAULT 300,            -- 超时时间(秒)
		agent_daily_token_limit INTEGER NOT NULL DEFAULT -1,   -- 每日Token限额(-1表示无限制)
		agent_history_max_entries INTEGER NOT NULL DEFAULT 100,-- 历史记录最大条数
		agent_history_max_days INTEGER NOT NULL DEFAULT 30,    -- 历史记录保留天数
		-- 知识库配置
		knowledge_enabled INTEGER NOT NULL DEFAULT 1,          -- 知识库是否启用(0=禁用,1=启用)
		-- 时间戳
		created_at DATETIME NOT NULL DEFAULT (datetime('now')),-- 创建时间
		updated_at DATETIME NOT NULL DEFAULT (datetime('now')) -- 更新时间
	);`,

	// ============================================
	// Token使用统计表
	// 记录每日Token消耗情况，用于预算控制
	// ============================================
	`CREATE TABLE IF NOT EXISTS token_daily_usage (
		date TEXT PRIMARY KEY,                                 -- 日期(YYYYMMDD格式)
		total_tokens INTEGER NOT NULL DEFAULT 0,               -- 总Token数
		input_tokens INTEGER NOT NULL DEFAULT 0,               -- 输入Token数
		output_tokens INTEGER NOT NULL DEFAULT 0,              -- 输出Token数
		updated_at DATETIME NOT NULL DEFAULT (datetime('now')) -- 更新时间
	);`,

	// ============================================
	// 用户画像主表
	// 存储用户的基础信息和偏好设置
	// ============================================
	`CREATE TABLE IF NOT EXISTS user_profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,                  -- 主键
		user_id TEXT UNIQUE NOT NULL,                          -- 用户唯一标识
		preferences TEXT,                                      -- 偏好设置(JSON格式)
		created_at INTEGER NOT NULL,                           -- 创建时间(Unix时间戳)
		updated_at INTEGER NOT NULL                            -- 更新时间(Unix时间戳)
	);`,

	// ============================================
	// 画像标签表
	// 存储从对话中提取的用户标签(技能、兴趣等)
	// ============================================
	`CREATE TABLE IF NOT EXISTS profile_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,                  -- 主键
		user_id TEXT NOT NULL,                                 -- 关联用户ID
		name TEXT NOT NULL,                                    -- 标签名称
		type TEXT NOT NULL,                                    -- 标签类型: skill/interest/preference
		confidence REAL NOT NULL,                              -- 置信度(0.0-1.0)
		evidence TEXT,                                         -- 原文证据
		created_at INTEGER NOT NULL,                           -- 创建时间
		updated_at INTEGER NOT NULL,                           -- 更新时间
		-- 约束
		UNIQUE(user_id, name),                                 -- 同一用户标签唯一
		FOREIGN KEY (user_id) REFERENCES user_profiles(user_id) ON DELETE CASCADE
	);`,

	// ============================================
	// 画像变更历史表
	// 记录画像的增删改操作，用于审计和追踪
	// ============================================
	`CREATE TABLE IF NOT EXISTS profile_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,                  -- 主键
		user_id TEXT NOT NULL,                                 -- 关联用户ID
		change_type TEXT NOT NULL,                             -- 变更类型: add/update/delete
		tag_name TEXT,                                         -- 变更的标签名
		old_value TEXT,                                        -- 旧值(JSON格式)
		new_value TEXT,                                        -- 新值(JSON格式)
		created_at INTEGER NOT NULL,                           -- 变更时间
		FOREIGN KEY (user_id) REFERENCES user_profiles(user_id) ON DELETE CASCADE
	);`,

	// ============================================
	// 画像标签索引
	// 加速按用户和类型查询标签
	// ============================================
	`CREATE INDEX IF NOT EXISTS idx_profile_tags_user ON profile_tags(user_id);`,
	`CREATE INDEX IF NOT EXISTS idx_profile_tags_type ON profile_tags(type);`,

	// ============================================
	// 画像历史索引
	// 加速按用户和时间查询变更历史
	// ============================================
	`CREATE INDEX IF NOT EXISTS idx_profile_history_user ON profile_history(user_id);`,
	`CREATE INDEX IF NOT EXISTS idx_profile_history_time ON profile_history(created_at);`,

	// ============================================
	// 定时任务表
	// 存储调度器任务配置和状态
	// ============================================
	`CREATE TABLE IF NOT EXISTS scheduled_tasks (
		id          TEXT PRIMARY KEY,                          -- 任务唯一标识
		name        TEXT NOT NULL,                             -- 任务名称
		task_type   TEXT NOT NULL,                             -- 任务类型: cron(定时循环)/delay(延迟执行)/once(单次执行)
		executor    TEXT NOT NULL,                             -- 执行器类型: script(脚本)/api(API调用)/agent(Agent调用)
		schedule    TEXT NOT NULL,                             -- 调度表达式: cron表达式/延迟时间/时间戳
		next_run_at INTEGER NOT NULL,                          -- 下次执行时间(Unix时间戳)
		last_run_at INTEGER DEFAULT 0,                         -- 上次执行时间(Unix时间戳)
		config      TEXT,                                      -- 执行配置(JSON格式)
		status      TEXT DEFAULT 'pending',                    -- 任务状态: pending(待执行)/running(执行中)/paused(暂停)/completed(完成)/failed(失败)/cancelled(已取消)
		result      TEXT,                                      -- 上次执行结果(JSON格式)
		retries     INTEGER DEFAULT 0,                         -- 当前重试次数
		max_retries INTEGER DEFAULT 3,                         -- 最大重试次数
		created_at  INTEGER NOT NULL,                          -- 创建时间(Unix时间戳)
		updated_at  INTEGER NOT NULL,                          -- 更新时间(Unix时间戳)
		created_by  TEXT                                       -- 创建者标识
	);`,

	// ============================================
	// 任务执行记录表
	// 存储任务每次执行的详细记录
	// ============================================
	`CREATE TABLE IF NOT EXISTS task_runs (
		id           TEXT PRIMARY KEY,                         -- 执行记录唯一标识
		task_id      TEXT NOT NULL,                            -- 关联的任务ID
		task_name    TEXT NOT NULL,                            -- 任务名称(冗余存储便于查询)
		executor     TEXT NOT NULL,                            -- 执行器类型
		input        TEXT,                                     -- 执行输入参数(JSON格式)
		status       TEXT DEFAULT 'running',                   -- 执行状态: running(运行中)/completed(成功完成)/failed(失败)/cancelled(已取消)
		start_time   INTEGER NOT NULL,                         -- 开始时间(Unix时间戳)
		end_time     INTEGER DEFAULT 0,                        -- 结束时间(Unix时间戳)
		duration     INTEGER DEFAULT 0,                        -- 执行耗时(毫秒)
		output       TEXT,                                     -- 执行输出结果
		error        TEXT,                                     -- 错误信息(失败时记录)
		exit_code    INTEGER DEFAULT 0,                        -- 退出码(0表示成功)
		retry_count  INTEGER DEFAULT 0,                        -- 当前重试次数
		created_at   INTEGER NOT NULL,                         -- 创建时间(Unix时间戳)
		updated_at   INTEGER NOT NULL                          -- 更新时间(Unix时间戳)
	);`,

	// ============================================
	// 定时任务索引
	// idx_status: 按状态查询任务
	// idx_next_run: 按下次执行时间查询待执行任务
	// idx_created_by: 按创建者查询任务
	// ============================================
	`CREATE INDEX IF NOT EXISTS idx_status ON scheduled_tasks(status);`,
	`CREATE INDEX IF NOT EXISTS idx_next_run ON scheduled_tasks(next_run_at);`,
	`CREATE INDEX IF NOT EXISTS idx_created_by ON scheduled_tasks(created_by);`,

	// ============================================
	// 任务执行记录索引
	// idx_task_runs_task_id: 按任务ID查询执行历史
	// idx_task_runs_status: 按状态查询执行记录
	// idx_task_runs_created_at: 按创建时间查询执行记录
	// ============================================
	`CREATE INDEX IF NOT EXISTS idx_task_runs_task_id ON task_runs(task_id);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_status ON task_runs(status);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_created_at ON task_runs(created_at);`,

	// ============================================
	// Skill 状态表
	// 存储 Skill 的启用/禁用状态和描述信息
	// ============================================
	`CREATE TABLE IF NOT EXISTS skill_states (
		name        TEXT PRIMARY KEY,                          -- Skill 名称（对应目录名）
		description TEXT,                                      -- Skill 描述（从 SKILL.md 同步）
		enabled     BOOLEAN DEFAULT 1,                         -- 是否启用（1=启用，0=禁用）
		created_at  DATETIME NOT NULL DEFAULT (datetime('now')), -- 创建时间
		updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))  -- 更新时间
	);`,

	// ============================================
	// Skill 状态表索引
	// idx_skill_states_enabled: 按启用状态查询
	// ============================================
	`CREATE INDEX IF NOT EXISTS idx_skill_states_enabled ON skill_states(enabled);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_task_id ON task_runs(task_id);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_status ON task_runs(status);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_created_at ON task_runs(created_at);`,

	// ============================================
	// 系统状态表
	// 存储系统运行时的状态信息，如睡眠阻止状态等
	// 用于程序重启后恢复之前的设置
	// ============================================
	`CREATE TABLE IF NOT EXISTS system_state (
		key TEXT PRIMARY KEY,                                  -- 状态键名
		value TEXT NOT NULL,                                   -- 状态值
		updated_at DATETIME NOT NULL DEFAULT (datetime('now')) -- 更新时间
	);`,
}
