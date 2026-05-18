package memory

import (
	"sync"
	"time"

	"github.com/longstageai/donk/donk/pkg/schema"
)

// SessionMemory 当前会话记忆
// 管理当前会话的对话历史和任务状态
type SessionMemory struct {
	mu          sync.RWMutex     // 读写锁
	history     []schema.Message // 对话历史
	state       *TaskState       // 当前任务状态
	maxMessages int              // 最大消息条数
}

// TaskState 任务执行状态
type TaskState struct {
	Task    string         // 当前任务描述
	Step    int            // 当前执行步骤
	Actions []string       // 已执行的操作列表
	Results map[string]any // 操作结果存储
}

// NewSessionMemory 创建新的会话记忆
func NewSessionMemory() *SessionMemory {
	return &SessionMemory{
		history: make([]schema.Message, 0),
		state: &TaskState{
			Results: make(map[string]any),
		},
		maxMessages: 20,
	}
}

// NewSessionMemoryWithLimit 创建指定消息上限的会话记忆
func NewSessionMemoryWithLimit(maxMessages int) *SessionMemory {
	if maxMessages <= 0 {
		maxMessages = 20
	}
	return &SessionMemory{
		history: make([]schema.Message, 0),
		state: &TaskState{
			Results: make(map[string]any),
		},
		maxMessages: maxMessages,
	}
}

// NewSessionMemoryWithSystem 创建带系统消息的会话记忆
func NewSessionMemoryWithSystem(systemPrompt string, maxMessages int) *SessionMemory {
	m := NewSessionMemoryWithLimit(maxMessages)
	if systemPrompt != "" {
		m.history = append(m.history, schema.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	return m
}

// SetSystemMessage 设置系统消息（如果已存在则更新）
func (m *SessionMemory) SetSystemMessage(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// 检查是否已有 system 消息
	for i, msg := range m.history {
		if msg.Role == "system" {
			m.history[i].Content = content
			m.history[i].Timestamp = now
			return
		}
	}

	// 没有则添加到最前面
	m.history = append([]schema.Message{{Role: "system", Content: content, Timestamp: now}}, m.history...)
	m.trimHistory()
}

// AddMessage 添加一条消息到历史记录
func (m *SessionMemory) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, schema.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})

	// 超过最大条数时，删除最旧的消息
	m.trimHistory()
}

// trimHistory 裁剪历史记录
// 1. 保护 system 消息不被删除
// 2. 合并连续相同的助手消息（仅当 Content 相同且都没有 ToolCalls）
// 3. 保持消息条数在限制内
func (m *SessionMemory) trimHistory() {
	// 1. 分离 system 消息和其他消息
	var systemMsgs []schema.Message
	var normalMsgs []schema.Message

	for _, msg := range m.history {
		if msg.Role == "system" {
			systemMsgs = append(systemMsgs, msg)
		} else {
			normalMsgs = append(normalMsgs, msg)
		}
	}

	// 2. 合并连续相同的助手消息（只保留最后一个）
	// 仅当 Content 相同且都没有 ToolCalls 时才合并
	if len(normalMsgs) > 1 {
		merged := make([]schema.Message, 0, len(normalMsgs))
		for i := 0; i < len(normalMsgs); i++ {
			curr := normalMsgs[i]
			// 检查是否需要与前一个合并
			if len(merged) > 0 {
				prev := merged[len(merged)-1]
				// 连续相同内容的助手消息，且都没有 ToolCalls 时才合并
				if prev.Role == "assistant" && curr.Role == "assistant" &&
					prev.Content == curr.Content &&
					len(prev.ToolCalls) == 0 && len(curr.ToolCalls) == 0 {
					merged[len(merged)-1] = curr
					continue
				}
			}
			merged = append(merged, curr)
		}
		normalMsgs = merged
	}

	// 3. 限制条数：确保 system + 正常消息 <= maxMessages
	maxNormal := m.maxMessages
	if len(systemMsgs) > 0 {
		maxNormal = m.maxMessages - len(systemMsgs)
		if maxNormal < 1 {
			maxNormal = 1
		}
	}

	if len(normalMsgs) > maxNormal {
		normalMsgs = normalMsgs[len(normalMsgs)-maxNormal:]
	}

	// 4. 重新组装
	m.history = append(systemMsgs, normalMsgs...)
}

// AddUserMessage 添加用户消息
func (m *SessionMemory) AddUserMessage(content string) {
	m.AddMessage("user", content)
}

// AddAssistantMessage 添加助手消息
func (m *SessionMemory) AddAssistantMessage(content string) {
	m.AddMessage("assistant", content)
}

// AddAssistantMessageWithToolCalls 添加助手消息（包含工具调用）
func (m *SessionMemory) AddAssistantMessageWithToolCalls(content string, toolCalls []schema.ToolCall) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, schema.Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
		Timestamp: time.Now(),
	})

	// 超过最大条数时，删除最旧的消息
	m.trimHistory()
}

// AddToolMessage 添加工具执行结果消息
func (m *SessionMemory) AddToolMessage(content, toolCallID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = append(m.history, schema.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: toolCallID,
		Timestamp:  time.Now(),
	})

	// 超过最大条数时，删除最旧的消息
	m.trimHistory()
}

// GetMessages 返回完整的消息历史
func (m *SessionMemory) GetMessages() []schema.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]schema.Message, len(m.history))
	copy(result, m.history)
	return result
}

// GetLastN 返回最近N条消息
func (m *SessionMemory) GetLastN(n int) []schema.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n >= len(m.history) {
		return m.history
	}
	return m.history[len(m.history)-n:]
}

// UpdateState 更新任务状态
func (m *SessionMemory) UpdateState(task string, step int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果任务变更，重置状态
	if m.state.Task != task {
		m.state.Task = task
		m.state.Actions = make([]string, 0)
		m.state.Results = make(map[string]any)
	}
	m.state.Step = step
}

// AddAction 记录一个执行的操作
func (m *SessionMemory) AddAction(action string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Actions = append(m.state.Actions, action)
}

// AddResult 添加操作结果
func (m *SessionMemory) AddResult(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Results[key] = value
}

// GetState 获取当前任务状态的副本
func (m *SessionMemory) GetState() *TaskState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回状态副本，避免并发问题
	stateCopy := *m.state
	stateCopy.Actions = make([]string, len(m.state.Actions))
	copy(stateCopy.Actions, m.state.Actions)
	stateCopy.Results = make(map[string]any, len(m.state.Results))
	for k, v := range m.state.Results {
		stateCopy.Results[k] = v
	}
	return &stateCopy
}

// Clear 清空所有记忆
func (m *SessionMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = make([]schema.Message, 0)
	m.state = &TaskState{
		Results: make(map[string]any),
	}
}
