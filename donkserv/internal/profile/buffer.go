package profile

import (
	"sync"
	"time"
)

// IDialogBuffer 对话缓存接口
// 用于临时存储对话消息
type IDialogBuffer interface {
	// Add 添加消息
	Add(msg Message)

	// GetAndClear 获取消息并清空
	GetAndClear() []Message

	// Count 消息数量
	Count() int

	// IsTimeout 是否超时
	IsTimeout(timeout time.Duration) bool

	// LastTime 最后消息时间
	LastTime() time.Time
}

// DialogBuffer 对话缓存实现
// 线程安全，支持超时检测
type DialogBuffer struct {
	mu       sync.Mutex
	messages []Message
	lastTime time.Time
}

// NewDialogBuffer 创建对话缓存
func NewDialogBuffer() *DialogBuffer {
	return &DialogBuffer{
		messages: make([]Message, 0),
	}
}

// Add 添加消息到缓存
//
// 参数:
//
//	msg: 待添加的消息
func (d *DialogBuffer) Add(msg Message) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.messages = append(d.messages, msg)
	d.lastTime = msg.Time
}

// GetAndClear 获取所有消息并清空缓存
//
// 返回:
//
//	[]Message: 当前缓存中的所有消息
func (d *DialogBuffer) GetAndClear() []Message {
	d.mu.Lock()
	defer d.mu.Unlock()

	messages := make([]Message, len(d.messages))
	copy(messages, d.messages)

	d.messages = make([]Message, 0)
	d.lastTime = time.Time{}

	return messages
}

// Count 返回当前缓存中的消息数量
//
// 返回:
//
//	int: 消息数量
func (d *DialogBuffer) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return len(d.messages)
}

// IsTimeout 检查是否超时
// 如果最后一条消息距离现在超过指定时间，且有消息缓存，则返回 true
//
// 参数:
//
//	timeout: 超时时间
//
// 返回:
//
//	bool: 是否超时
func (d *DialogBuffer) IsTimeout(timeout time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.messages) == 0 {
		return false
	}

	return time.Since(d.lastTime) > timeout
}

// LastTime 返回最后一条消息的时间
//
// 返回:
//
//	time.Time: 最后消息时间
func (d *DialogBuffer) LastTime() time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.lastTime
}
