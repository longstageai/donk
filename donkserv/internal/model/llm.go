package model

import (
	"context"

	"github.com/longstageai/donk/donk/pkg/schema"
)

// LLM 对话模型接口
// 定义与大语言模型交互的统一接口，便于切换不同的模型实现
type LLM interface {
	// Name 返回模型名称
	Name() string

	// Chat 同步对话
	// 参数:
	//   - ctx: 上下文
	//   - req: 对话请求
	// 返回:
	//   - *schema.ChatResponse: 对话响应
	//   - error: 错误信息
	Chat(ctx context.Context, req *schema.ChatRequest) (*schema.ChatResponse, error)

	// StreamChat 流式对话
	// 参数:
	//   - ctx: 上下文
	//   - req: 对话请求
	// 返回:
	//   - *StreamResponse: 流式响应
	//   - error: 错误信息
	StreamChat(ctx context.Context, req *schema.ChatRequest) (*StreamResponse, error)

	// SetConfig 设置配置参数
	// 用于在运行时动态更新模型配置（model、apiKey、baseURL）
	// 参数:
	//   - model: 模型名称
	//   - apiKey: API密钥
	//   - baseURL: API基础地址
	SetConfig(model, apiKey, baseURL string)
}
