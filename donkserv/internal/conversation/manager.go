package conversation

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/internal/message"
	"github.com/longstageai/donk/donk/internal/profile"
)

// Config 向量化管理器配置
type Config struct {
	Timeout       time.Duration // 对话结束超时时间
	MinMessages   int           // 最小消息数，少于此数不触发向量化
	CheckInterval time.Duration // 检查间隔
	StoreDir      string        // 对话历史存储目录（已废弃，由 VectorDBManager 统一管理）
}

// DefaultConfig 默认配置
// 注意：实际存储路径由 config.DataPaths 统一管理
var DefaultConfig = Config{
	Timeout:       1 * time.Minute,  // 1 分钟无新消息视为对话结束
	MinMessages:   3,                // 至少 3 条消息才向量化
	CheckInterval: 30 * time.Second, // 30 秒检查一次
	StoreDir:      "",               // 空表示使用默认路径（由 VectorDBManager 管理）
}

// Manager 对话历史管理器
// 负责对话缓存、触发检测、向量化存储
type Manager struct {
	config    Config                // 配置
	mu        sync.RWMutex          // 读写锁
	buffer    profile.IDialogBuffer // 对话缓存（复用 profile 模块）
	store     *Store                // 对话历史存储
	search    *Search               // 检索器
	isRunning bool                  // 是否正在运行
}

// NewManager 创建对话历史管理器
//
// 参数:
//
//	config: 配置
//	store: 对话历史存储
//	search: 检索器
//
// 返回:
//
//	*Manager: 对话历史管理器
func NewManager(config Config, store *Store, search *Search) *Manager {
	return &Manager{
		config: config,
		buffer: profile.NewDialogBuffer(),
		store:  store,
		search: search,
	}
}

// Start 启动后台向量化任务
func (m *Manager) Start() {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.mu.Unlock()

	go m.runVectorizer()
}

// Stop 停止后台向量化任务
func (m *Manager) Stop() {
	m.mu.Lock()
	m.isRunning = false
	m.mu.Unlock()
}

// runVectorizer 后台向量化任务
func (m *Manager) runVectorizer() {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		<-ticker.C

		m.mu.RLock()
		if !m.isRunning {
			m.mu.RUnlock()
			return
		}
		m.mu.RUnlock()

		m.maybeVectorize()
	}
}

// maybeVectorize 检查并向量化对话
func (m *Manager) maybeVectorize() {
	// 检查是否满足向量化条件
	if !m.buffer.IsTimeout(m.config.Timeout) {
		return
	}

	if m.buffer.Count() < m.config.MinMessages {
		return
	}

	// 异步执行向量化
	go m.vectorizeConversation()
}

// vectorizeConversation 对对话进行向量化
func (m *Manager) vectorizeConversation() {
	ctx := context.Background()

	// 1. 获取对话内容并清空缓存
	messages := m.buffer.GetAndClear()
	if len(messages) == 0 {
		return
	}

	// 2. 格式化对话内容
	conversationID := generateConversationID()
	messagesText := formatConversation(messages)
	startTime := messages[0].Time
	endTime := messages[len(messages)-1].Time

	// 3. 添加到对话历史（自动切片、向量化、存储）
	err := m.store.AddConversation(ctx, conversationID, messagesText, startTime, endTime)
	if err != nil {
		// 向量化失败，记录日志（实际应用中可考虑重试）
		return
	}
}

// AddMessage 添加对话消息
// 在每次 Agent 回复后调用，记录对话
//
// 参数:
//
//	msg: 对话消息
func (m *Manager) AddMessage(msg message.Message) {
	m.buffer.Add(msg)
}

// Search 搜索对话历史
// 使用 LLM 自动判断时间过滤
//
// 参数:
//
//	ctx: 上下文
//	query: 查询文本
//	topK: 返回数量
//
// 返回:
//
//	[]SearchResult: 搜索结果
//	error: 错误信息
func (m *Manager) Search(ctx context.Context, query string, topK int, timeFilter *TimeFilter) ([]SearchResult, error) {
	return m.search.Search(ctx, query, topK, timeFilter)
}

// formatConversation 将消息列表格式化为可检索文本
//
// 参数:
//
//	messages: 消息列表
//
// 返回:
//
//	string: 格式化后的文本
func formatConversation(messages []profile.Message) string {
	var sb strings.Builder

	for _, msg := range messages {
		sb.WriteString(msg.Role)
		sb.WriteString(": ")
		sb.WriteString(msg.Content)
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateConversationID 生成对话ID
//
// 返回:
//
//	string: 对话ID
func generateConversationID() string {
	return time.Now().Format("20060102150405") + "_" + randomString(6)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(letters[time.Now().UnixNano()%int64(len(letters))])
	}
	return sb.String()
}
