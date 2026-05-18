package message

import "time"

// Message 对话消息结构
// 用于记录用户和 Agent 之间的对话
// 被 profile 和 conversation 模块共用
type Message struct {
	Role    string    `json:"role"`    // 角色：user / assistant
	Content string    `json:"content"` // 消息内容
	Time    time.Time `json:"time"`    // 消息时间
}

// Messages 消息列表
// 提供便捷的批量操作方法
type Messages []Message

// FilterByRole 按角色过滤消息
//
// 参数:
//   - role: 角色名称
//
// 返回:
//   - Messages: 过滤后的消息列表
func (m Messages) FilterByRole(role string) Messages {
	var result Messages
	for _, msg := range m {
		if msg.Role == role {
			result = append(result, msg)
		}
	}
	return result
}

// ToText 将消息列表转换为文本格式
//
// 参数:
//   - separator: 分隔符
//
// 返回:
//   - string: 格式化后的文本
func (m Messages) ToText(separator string) string {
	var result string
	for i, msg := range m {
		if i > 0 {
			result += separator
		}
		result += msg.Role + ": " + msg.Content
	}
	return result
}
