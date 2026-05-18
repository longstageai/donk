package agent

import (
	"database/sql"
	"errors"
	"time"

	"github.com/longstageai/donk/donk/internal/conversation"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/internal/tool"

	"github.com/longstageai/donk/donk/pkg/schema"
)

// 错误定义
var (
	ErrMaxLoopExceeded     = errors.New("达到最大循环次数")   // ReAct循环超过最大次数
	ErrTaskCompleted       = errors.New("任务已完成")      // 任务执行完成
	ErrAgentStopped        = errors.New("Agent已停止")   // Agent被主动停止
	ErrTokenBudgetExceeded = errors.New("Token预算已超出") // Token预算超限
	ErrConvergeTimeout     = errors.New("任务收敛超时")     // 连续多轮无实质进展
	ErrCanceled            = errors.New("任务被用户取消")    // 用户取消执行
)

// Agent AI智能体核心结构
// 整合模型、工具和记忆，实现自主任务执行
type Agent struct {
	model               model.Adapter           // 模型适配器
	tools               *tool.Registry          // 工具注册表
	skillRegistry       *skill.SkillRegistry    // Skill注册表
	sessionMemory       *memory.SessionMemory   // 当前会话记忆
	longMemory          *memory.LongMemory      // 长期记忆（持久化知识）
	historyStore        *memory.HistoryStore    // 历史记录（会话历史）
	profileManager      *profile.ProfileManager // 用户画像管理器
	conversationManager *conversation.Manager   // 对话历史管理器
	tokenStats          *token.TokenStats       // Token消耗统计
	executionTrace      *memory.ExecutionTrace  // 执行轨迹（解耦）
	historyLoader       *HistoryLoader          // 动态历史加载器
	interceptor         *Interceptor            // 拦截器（控制信号）
	maxLoop             int                     // 最大循环次数
	convergeAfter       int                     // 连续N轮无工具调用则终止
	timeout             time.Duration           // 超时时间
	onStream            StreamCallback          // 流式回调函数
	currentInput        string                  // 当前会话的用户输入
	sessionToolCalls    []memory.ToolCallInfo   // 当前会话的工具调用记录（兼容旧代码）
	systemPrompt        string                  // 系统提示词
	workspace           string                  // 工作空间目录
	cancel              chan struct{}           // 取消信号通道
	db                  *sql.DB                 // 数据库连接（用于动态加载技能）
	skillDir            string                  // Skill目录路径
}

// StreamCallback 流式事件回调类型
type StreamCallback func(event *StreamEvent)

// SetModel 设置模型适配器
// 用于在运行时动态更换 LLM 模型（例如配置更新后）
func (a *Agent) SetModel(modelAdapter model.Adapter) {
	a.model = modelAdapter
}

// StreamEventType 流式事件类型枚举
type StreamEventType string

const (
	EventUserInput      StreamEventType = "user_input"      // 用户输入
	EventReasoningDelta StreamEventType = "reasoning_delta" // 思考过程增量
	EventContentDelta   StreamEventType = "content_delta"   // 内容增量
	EventAssistant      StreamEventType = "assistant"       // 助手完整回复
	EventToolCall       StreamEventType = "tool_call"       // 工具调用
	EventToolResult     StreamEventType = "tool_result"     // 工具执行结果
	EventWarning        StreamEventType = "warning"         // 警告
	EventError          StreamEventType = "error"           // 错误
	EventStop           StreamEventType = "stop"            // 正常停止
	EventCanceled       StreamEventType = "canceled"        // 用户取消
)

// StreamEvent 流式事件结构
type StreamEvent struct {
	Type             StreamEventType  // 事件类型
	Content          string           // 文本内容
	ReasoningContent string           // 思考过程内容
	ToolName         string           // 工具名称
	ToolInput        string           // 工具输入参数
	ToolResult       string           // 工具执行结果
	Error            string           // 错误信息
	Usage            schema.UsageInfo // Token使用量统计
}
