package profile

import (
	"strings"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/internal/message"
)

const (
	// MinTriggerCount 最小触发消息数
	// 少于这个数量不触发，避免信息不足
	MinTriggerCount = 6

	// MaxTriggerCount 最大触发消息数
	// 超过这个数量需要分批处理
	MaxTriggerCount = 20

	// MaxWaitTime 最大等待时间
	// 超过这个时间强制触发
	MaxWaitTime = 5 * time.Minute
)

// Trigger 画像更新触发器
// 管理消息缓冲和触发时机判断
type Trigger struct {
	Keywords        []string // 立即触发关键词
	messages        []message.Message
	lastTriggerTime time.Time
	mu              sync.Mutex
}

// NewTrigger 创建触发器
//
// 返回:
//   - *Trigger: 触发器实例
func NewTrigger() *Trigger {
	return &Trigger{
		Keywords:        []string{"我喜欢", "我讨厌", "请记住", "我是", "我会", "我不会"},
		lastTriggerTime: time.Now(),
	}
}

// OnMessage 处理新消息
// 返回是否需要立即触发提取
//
// 参数:
//   - msg: 新消息
//
// 返回:
//   - bool: 是否触发
//   - []message.Message: 待处理的消息列表
func (t *Trigger) OnMessage(msg message.Message) (bool, []message.Message) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.messages = append(t.messages, msg)

	// 检查关键词立即触发（优先级最高，不受数量限制）
	if t.hasKeyword(msg.Content) {
		return t.extractAndClear()
	}

	// 检查是否达到最大消息数，需要分批处理
	if len(t.messages) >= MaxTriggerCount {
		return t.extractAndClear()
	}

	return false, nil
}

// GetPendingMessages 获取待处理消息（用于定时触发）
// 定时检查是否满足触发条件
//
// 返回:
//   - []message.Message: 待处理消息列表，不满足条件返回nil
func (t *Trigger) GetPendingMessages() []message.Message {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 消息数不足，不触发
	if len(t.messages) < MinTriggerCount {
		return nil
	}

	// 检查是否超过最大等待时间
	if time.Since(t.lastTriggerTime) < MaxWaitTime {
		return nil
	}

	_, msgs := t.extractAndClear()
	return msgs
}

// extractAndClear 提取消息并清空缓冲区
// 内部方法，调用前必须持有锁
//
// 返回:
//   - bool: 是否成功提取
//   - []message.Message: 提取的消息列表
func (t *Trigger) extractAndClear() (bool, []message.Message) {
	if len(t.messages) == 0 {
		return false, nil
	}

	// 复制消息
	msgs := make([]message.Message, len(t.messages))
	copy(msgs, t.messages)

	// 清空缓冲区（重要：确保提取过的消息被清除）
	t.messages = t.messages[:0]
	t.lastTriggerTime = time.Now()

	return true, msgs
}

// hasKeyword 检查内容是否包含触发关键词
//
// 参数:
//   - content: 消息内容
//
// 返回:
//   - bool: 是否包含关键词
func (t *Trigger) hasKeyword(content string) bool {
	content = strings.ToLower(content)
	for _, keyword := range t.Keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// GetBufferSize 获取当前缓冲区消息数
// 用于监控和调试
//
// 返回:
//   - int: 缓冲区消息数量
func (t *Trigger) GetBufferSize() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.messages)
}

// Clear 清空缓冲区
// 用于优雅关闭或重置状态
func (t *Trigger) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages = t.messages[:0]
}
