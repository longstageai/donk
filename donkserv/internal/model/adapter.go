package model

import (
	"sync"

	"github.com/longstageai/donk/donk/pkg/schema"
)

// Adapter LLM 接口类型别名
// 用于保持向后兼容
type Adapter = LLM

// StreamResponse 流式响应结构
// 用于流式对话的响应数据
type StreamResponse struct {
	Chunks       chan *schema.StreamChunk // 消息块通道
	Error        error                    // 错误信息
	Done         chan struct{}            // 完成信号
	Usage        schema.UsageInfo         // Token 使用统计
	FinishReason string                   // 完成原因 (stop, length 等)
	closed       bool                     // 是否已关闭
	closeMu      sync.Mutex               // 关闭锁
}

// isClosed 检查响应是否已关闭
func (s *StreamResponse) isClosed() bool {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	return s.closed
}

// Close 关闭流式响应
// 关闭后不再发送新的消息块
func (s *StreamResponse) Close() {
	s.closeMu.Lock()
	if !s.closed {
		s.closed = true
		close(s.Chunks)
	}
	s.closeMu.Unlock()
}

// NewAdapter 创建模型适配器
// 根据 provider 参数创建对应的模型适配器
//
// 参数:
//   - provider: 模型提供商 (openai, qwen, deepseek, doubao)
//   - apiKey: API 密钥
//   - model: 模型名称
//   - baseURL: API 基础地址
//
// 返回:
//   - Adapter: 模型适配器
//   - error: 错误信息
func NewAdapter(provider, apiKey, model, baseURL string) (Adapter, error) {
	switch provider {
	case "deepseek":
		return NewDeepSeekAdapter(apiKey, model, baseURL), nil
	case "openai":
		return NewOpenAIAdapter(apiKey, model, baseURL), nil
	case "qwen":
		return NewQwenAdapter(apiKey, model, baseURL), nil
	case "doubao":
		return NewDoubaoAdapter(apiKey, model, baseURL), nil
	default:
		return nil, nil
	}
}
