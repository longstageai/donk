package memory

import (
	"math"
	"time"
)

// MemoryType 记忆类型
type MemoryType string

const (
	MemoryTypeShort   MemoryType = "short"   // 短期记忆
	MemoryTypeLong    MemoryType = "long"    // 长期记忆
	MemoryTypeHistory MemoryType = "history" // 历史记录
)

// Metadata 元数据
// 存储记忆的额外信息，如来源、会话ID等
type Metadata struct {
	Source       string            `json:"source"`        // 来源：user/agent/tool
	SessionID    string            `json:"session_id"`    // 会话ID
	ActionCount  int               `json:"action_count"`  // 累计操作次数
	ReferencedBy int               `json:"referenced_by"` // 被引用次数
	Extra        map[string]string `json:"extra"`         // 扩展字段
}

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"      // 用户消息
	RoleAssistant MessageRole = "assistant" // 助手消息
	RoleTool      MessageRole = "tool"      // 工具消息
)

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
	ID       string `json:"id"`       // 工具调用ID
	Name     string `json:"name"`     // 工具名称
	Input    string `json:"input"`    // 输入参数
	Output   string `json:"output"`   // 输出结果
	Duration int64  `json:"duration"` // 执行耗时(毫秒)
	Round    int    `json:"round"`    // 所属轮次
}

// MemoryEntry 记忆条目
// 统一的记忆数据结构，用于短期、长期、历史记录
type MemoryEntry struct {
	Key       string         `json:"key"`        // 唯一标识符
	Type      MemoryType     `json:"type"`       // 记忆类型
	Role      MessageRole    `json:"role"`       // 消息角色：user/assistant/tool
	Content   string         `json:"content"`    // 记忆原始内容
	Summary   string         `json:"summary"`    // LLM生成的摘要
	Keywords  []string       `json:"keywords"`   // 提取的关键词
	Metadata  Metadata       `json:"metadata"`   // 元数据
	Timestamp time.Time      `json:"timestamp"`  // 时间戳
	Tags      []string       `json:"tags"`       // 标签列表
	Score     float64        `json:"score"`      // 重要性评分
	ToolCalls []ToolCallInfo `json:"tool_calls"` // 工具调用列表
}

// NewMemoryEntry 创建新的记忆条目
func NewMemoryEntry(key, content string, memType MemoryType) *MemoryEntry {
	return &MemoryEntry{
		Key:       key,
		Type:      memType,
		Content:   content,
		Timestamp: time.Now(),
		Tags:      make([]string, 0),
		Score:     0.0,
		Metadata: Metadata{
			Extra: make(map[string]string),
		},
		ToolCalls: make([]ToolCallInfo, 0),
	}
}

// AddTag 添加标签
func (e *MemoryEntry) AddTag(tags ...string) {
	e.Tags = append(e.Tags, tags...)
}

// CalculateScore 计算重要性评分
// 基于被引用次数和时间衰减计算评分
func (e *MemoryEntry) CalculateScore() float64 {
	score := 0.0
	// 被引用次数越多，评分越高
	if e.Metadata.ReferencedBy > 0 {
		score += float64(e.Metadata.ReferencedBy) * 0.3
	}
	// 时间衰减：越新的记忆评分略高
	age := time.Since(e.Timestamp)
	score += 0.5 * math.Exp(-age.Hours()/(30*24))
	return score
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Context    string     // 语义搜索查询文本
	Keywords   []string   // 关键词列表
	Tags       []string   // 标签筛选
	MemoryType MemoryType // 记忆类型筛选
	Role       MessageRole
	Limit      int
}

// SearchResult 搜索结果
type SearchResult struct {
	Entries []MemoryEntry // 匹配的条目列表
	Total   int           // 总匹配数量
}

// MemoryStore 记忆存储接口
// 定义记忆的通用操作
type MemoryStore interface {
	Save(entry *MemoryEntry) error                             // 保存记忆
	Search(req SearchRequest) (*SearchResult, error)           // 搜索记忆
	Get(key string) (*MemoryEntry, error)                      // 根据key获取记忆
	Delete(key string) error                                   // 删除记忆
	List(memType MemoryType, limit int) ([]MemoryEntry, error) // 列出记忆
}
