package agent

import (
	"database/sql"
	"time"

	"github.com/longstageai/donk/donk/internal/conversation"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/token"

	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/internal/tool/builtin"
)

// Option Agent配置选项类型
type Option func(*Agent)

// WithMaxLoop 设置最大循环次数
// 用于防止Agent在复杂任务中无限循环
func WithMaxLoop(max int) Option {
	return func(a *Agent) {
		a.maxLoop = max
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(a *Agent) {
		a.timeout = timeout
	}
}

// WithStreamCallback 设置流式回调函数
// 用于实时获取Agent的执行过程
func WithStreamCallback(callback StreamCallback) Option {
	return func(a *Agent) {
		a.onStream = callback
	}
}

// SetStreamCallback 动态设置流式回调函数
func (a *Agent) SetStreamCallback(callback StreamCallback) {
	a.onStream = callback
}

// WithSystemPrompt 设置系统提示词
func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) {
		a.systemPrompt = prompt
	}
}

// WithWorkspace 设置工作空间目录
func WithWorkspace(workspace string) Option {
	return func(a *Agent) {
		a.workspace = workspace
	}
}

// WithProfileManager 设置用户画像管理器
// 用于在对话中加载用户画像，并异步更新画像
func WithProfileManager(pm *profile.ProfileManager) Option {
	return func(a *Agent) {
		a.profileManager = pm
	}
}

// WithConversationManager 设置对话历史管理器
// 用于对话历史的向量化存储和检索
func WithConversationManager(cm *conversation.Manager) Option {
	return func(a *Agent) {
		a.conversationManager = cm
	}
}

// WithTokenStats 设置Token统计器
// 用于追踪Token消耗并在超限时停止Agent
func WithTokenStats(ts *token.TokenStats) Option {
	return func(a *Agent) {
		a.tokenStats = ts
	}
}

// WithConvergeAfter 设置收敛检测
// 连续N轮无工具调用则终止任务，防止无限循环
func WithConvergeAfter(n int) Option {
	return func(a *Agent) {
		a.convergeAfter = n
	}
}

// WithHistoryLoader 设置动态历史加载器
// 根据Token预算动态决定加载多少历史消息
func WithHistoryLoader(hl *HistoryLoader) Option {
	return func(a *Agent) {
		a.historyLoader = hl
	}
}

// WithInterceptor 设置拦截器
// 用于在关键节点插入自定义逻辑，控制Agent行为
func WithInterceptor(i *Interceptor) Option {
	return func(a *Agent) {
		a.interceptor = i
	}
}

// WithSkillTool 注册Skill工具到工具注册表
// 参数:
//   - skillTool: Skill工具实例
//
// 返回:
//   - Option: 配置选项函数
func WithSkillTool(skillTool tool.Tool) Option {
	return func(a *Agent) {
		if a.tools != nil {
			a.tools.Register(skillTool)
		}
	}
}

// WithSkillRegistry 注册Skill系统到Agent
// 自动创建Skill工具并注册到工具注册表
// 同时设置 Agent 的 skillRegistry 以便在系统提示词中自动加载技能指令
// 参数:
//   - registry: Skill注册表
//   - workingDir: 工作目录
//
// 返回:
//   - Option: 配置选项函数
func WithSkillRegistry(registry *skill.SkillRegistry, workingDir string) Option {
	return func(a *Agent) {
		// 保存 skillRegistry 引用，用于系统提示词
		a.skillRegistry = registry

		if a.tools == nil {
			return
		}

		executor := skill.NewExecutor(registry, skill.WithWorkingDir(workingDir))
		skillTool := builtin.NewSkillTool(registry, executor, workingDir)

		a.tools.Register(skillTool)
	}
}

// WithSkillDB 设置Skill数据库连接和目录
// 用于每次对话时动态从数据库加载启用的技能
// 参数:
//   - db: 数据库连接
//   - skillDir: Skill目录路径
//
// 返回:
//   - Option: 配置选项函数
func WithSkillDB(db *sql.DB, skillDir string) Option {
	return func(a *Agent) {
		a.db = db
		a.skillDir = skillDir
	}
}
