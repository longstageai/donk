package agent

// ControlSignal 控制信号
// 用于在执行过程中控制Agent行为
type ControlSignal int

const (
	SignalContinue     ControlSignal = iota // 继续执行
	SignalStop                              // 停止执行
	SignalPause                             // 暂停执行
	SignalSkipNextTool                      // 跳过下一个工具调用
	SignalRetryLast                         // 重试上一步操作
)

// String 返回控制信号的字符串表示
func (c ControlSignal) String() string {
	switch c {
	case SignalContinue:
		return "continue"
	case SignalStop:
		return "stop"
	case SignalPause:
		return "pause"
	case SignalSkipNextTool:
		return "skip_next_tool"
	case SignalRetryLast:
		return "retry_last"
	default:
		return "unknown"
	}
}

// SignalHandler 信号处理函数类型
// 接收当前信号，返回处理后的信号
type SignalHandler func(signal ControlSignal) ControlSignal

// Interceptor 拦截器
// 用于在关键节点插入自定义逻辑，控制Agent行为
type Interceptor struct {
	OnUserInput    SignalHandler // 用户输入回调
	OnModelRequest SignalHandler // 模型请求回调
	OnToolCall     SignalHandler // 工具调用回调
	OnToolResult   SignalHandler // 工具结果回调
	OnAssistant    SignalHandler // 助手回复回调
}

// HandleUserInput 处理用户输入
// msg: 用户输入内容
// 返回: 控制信号
func (i *Interceptor) HandleUserInput(msg string) ControlSignal {
	if i.OnUserInput != nil {
		return i.OnUserInput(SignalContinue)
	}
	return SignalContinue
}

// HandleModelRequest 处理模型请求
// 返回: 控制信号
func (i *Interceptor) HandleModelRequest() ControlSignal {
	if i.OnModelRequest != nil {
		return i.OnModelRequest(SignalContinue)
	}
	return SignalContinue
}

// HandleToolCall 处理工具调用
// toolName: 工具名称
// 返回: 控制信号
func (i *Interceptor) HandleToolCall(toolName string) ControlSignal {
	if i.OnToolCall != nil {
		return i.OnToolCall(SignalContinue)
	}
	return SignalContinue
}

// HandleToolResult 处理工具结果
// toolName: 工具名称
// result: 工具返回结果
// 返回: 控制信号
func (i *Interceptor) HandleToolResult(toolName string, result string) ControlSignal {
	if i.OnToolResult != nil {
		return i.OnToolResult(SignalContinue)
	}
	return SignalContinue
}

// HandleAssistant 处理助手回复
// content: 助手回复内容
// 返回: 控制信号
func (i *Interceptor) HandleAssistant(content string) ControlSignal {
	if i.OnAssistant != nil {
		return i.OnAssistant(SignalContinue)
	}
	return SignalContinue
}
