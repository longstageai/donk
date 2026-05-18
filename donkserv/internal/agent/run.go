package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/message"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/tool"
)

// New 创建新的Agent实例
// 参数：模型适配器、工具注册表、长期记忆、历史记录存储
func New(modelAdapter model.Adapter, tools *tool.Registry, longMemory *memory.LongMemory, historyStore *memory.HistoryStore, opts ...Option) *Agent {
	agent := &Agent{
		model:          modelAdapter,
		tools:          tools,
		sessionMemory:  memory.NewSessionMemory(),
		longMemory:     longMemory,
		historyStore:   historyStore,
		executionTrace: memory.NewExecutionTrace(),
		maxLoop:        10,              // 默认最大循环10次
		timeout:        5 * time.Minute, // 默认超时5分钟
		cancel:         make(chan struct{}),
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

// Run 执行单轮对话（非流式）
// 输入用户问题，返回Agent的最终回复
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	// 保存当前输入
	a.currentInput = input
	a.sessionToolCalls = make([]memory.ToolCallInfo, 0)

	// 初始化工作记忆
	a.sessionMemory.Clear()

	// 设置系统提示词（包含用户画像、技能列表、上下文信息）
	systemMessage := a.buildSystemMessage()
	if systemMessage != "" {
		a.sessionMemory.SetSystemMessage(systemMessage)
	}

	// 先检查Token预算，获取剩余可用量
	remainingBudget := 0
	if a.tokenStats != nil {
		ok, remaining := a.tokenStats.CheckBudget()
		if !ok {
			return "", ErrTokenBudgetExceeded
		}
		remainingBudget = remaining
		// 当 remaining > 0 且 < 10000 时才提醒（remaining=0 表示不限制）
		if remaining > 0 && remaining < 10000 {
			if a.onStream != nil {
				a.onStream(&StreamEvent{Type: EventWarning, Content: "Token预算即将耗尽"})
			}
		}
	}

	// 动态计算加载多少历史消息
	historyLimit := 30
	if a.historyLoader != nil && remainingBudget > 0 {
		historyLimit = a.historyLoader.CalculateLoadCount(input, remainingBudget)
	}

	// 加载历史记录到工作记忆
	if a.historyStore != nil {
		history, err := a.historyStore.GetRecent(historyLimit)
		if err == nil && len(history) > 0 {
			for _, entry := range history {
				switch entry.Role {
				case memory.RoleUser:
					a.sessionMemory.AddMessage("user", entry.Content)
				case "assistant":
					a.sessionMemory.AddMessage("assistant", entry.Content)
				}
			}
		}
	}

	// 添加当前用户输入
	a.sessionMemory.AddUserMessage(input)

	// 发送用户输入事件
	if a.onStream != nil {
		a.onStream(&StreamEvent{Type: EventUserInput, Content: input})
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	var lastErr error
	consecutiveEmpty := 0

	// ReAct循环：思考->行动->观察
	for i := 0; i < a.maxLoop; i++ {
		select {
		case <-ctx.Done():
			a.saveToHistory(a.currentInput, "", a.sessionToolCalls...)
			return "", ctx.Err()
		case <-a.cancel:
			a.saveToHistory(a.currentInput, "", a.sessionToolCalls...)
			return "", ErrCanceled
		default:
		}

		// 检查Token预算
		if a.tokenStats != nil {
			ok, _ := a.tokenStats.CheckBudget()
			if !ok {
				return "", ErrTokenBudgetExceeded
			}
		}

		// 执行一步
		result, err := a.step(ctx, i)
		if err != nil {
			// 任务完成，直接返回结果
			if errors.Is(err, ErrTaskCompleted) {
				return result, nil
			}
			lastErr = err
			continue
		}

		// 检查是否有工具调用（用于收敛检测）
		hasTools := len(a.sessionToolCalls) > 0 && a.sessionToolCalls[len(a.sessionToolCalls)-1].Round == i
		if !hasTools && i > 0 {
			consecutiveEmpty++
			if a.convergeAfter > 0 && consecutiveEmpty >= a.convergeAfter {
				errMsg := "连续多轮无工具调用"
				if lastErr != nil {
					errMsg = fmt.Sprintf("%s: %v", errMsg, lastErr)
				}
				a.saveToHistory(input, result, a.sessionToolCalls...)
				return result, fmt.Errorf("%w: %s", ErrConvergeTimeout, errMsg)
			}
		} else {
			consecutiveEmpty = 0
		}

		// 返回空字符串表示继续循环（非空字符串表示继续循环）
		if result != "" {
			// 保存会话到历史记录
			a.saveToHistory(input, result, a.sessionToolCalls...)
			return result, nil
		}
	}

	return "", ErrMaxLoopExceeded
}

// RunStream 执行流式对话
// 实时返回Agent的执行过程
func (a *Agent) RunStream(ctx context.Context, input string) error {
	// 保存当前输入
	a.currentInput = input
	a.sessionToolCalls = make([]memory.ToolCallInfo, 0)

	// 初始化工作记忆
	a.sessionMemory.Clear()

	// 设置系统提示词（包含用户画像、技能列表、上下文信息）
	systemMessage := a.buildSystemMessage()
	if systemMessage != "" {
		a.sessionMemory.SetSystemMessage(systemMessage)
	}

	// 加载历史记录到工作记忆
	if a.historyStore != nil {
		history, err := a.historyStore.GetRecent(10)
		if err == nil && len(history) > 0 {
			for _, entry := range history {
				switch entry.Role {
				case memory.RoleUser:
					a.sessionMemory.AddMessage("user", entry.Content)
				case memory.RoleAssistant:
					a.sessionMemory.AddMessage("assistant", entry.Content)
				}
			}
		}
	}

	// 添加当前用户输入
	a.sessionMemory.AddUserMessage(input)

	// 发送用户输入事件
	if a.onStream != nil {
		a.onStream(&StreamEvent{Type: EventUserInput, Content: input})
	}

	// 检查Token预算
	if a.tokenStats != nil {
		ok, remaining := a.tokenStats.CheckBudget()
		if !ok {
			return ErrTokenBudgetExceeded
		}
		// 当 remaining > 0 且 < 10000 时才提醒（remaining=0 表示不限制）
		if remaining > 0 && remaining < 10000 {
			if a.onStream != nil {
				a.onStream(&StreamEvent{Type: EventWarning, Content: "Token预算即将耗尽"})
			}
		}
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	consecutiveEmpty := 0

	// ReAct循环
	for i := 0; i < a.maxLoop; i++ {
		select {
		case <-ctx.Done():
			a.saveToHistory(input, "", a.sessionToolCalls...)
			return ctx.Err()
		case <-a.cancel:
			a.saveToHistory(input, "", a.sessionToolCalls...)
			return ErrCanceled
		default:
		}

		// 检查Token预算
		if a.tokenStats != nil {
			ok, _ := a.tokenStats.CheckBudget()
			if !ok {
				return ErrTokenBudgetExceeded
			}
		}

		err := a.stepStream(ctx, i)
		if err != nil {
			if errors.Is(err, ErrTaskCompleted) {
				return nil
			}
			if errors.Is(err, ErrCanceled) {
				a.sendCancelEvent()
				return nil
			}
			return err
		}

		// 收敛检测
		hasTools := len(a.sessionToolCalls) > 0 && a.sessionToolCalls[len(a.sessionToolCalls)-1].Round == i
		if !hasTools && i > 0 {
			consecutiveEmpty++
			if a.convergeAfter > 0 && consecutiveEmpty >= a.convergeAfter {
				a.saveToHistory(input, "", a.sessionToolCalls...)
				return ErrConvergeTimeout
			}
		} else {
			consecutiveEmpty = 0
		}
	}

	// 超过最大循环次数，保存到历史记录
	a.saveToHistory(a.currentInput, "", a.sessionToolCalls...)

	return ErrMaxLoopExceeded
}

// Cancel 取消当前执行的Agent
// 发送取消信号，终止正在进行的请求
func (a *Agent) Cancel() {
	select {
	case a.cancel <- struct{}{}:
	default:
	}
}

// sendCancelEvent 发送取消事件
func (a *Agent) sendCancelEvent() {
	if a.onStream != nil {
		a.onStream(&StreamEvent{Type: EventCanceled})
	}
}

// Reset 重置Agent状态
// 清空所有记忆，重新开始
func (a *Agent) Reset() {
	a.sessionMemory.Clear()
}

// saveToHistory 保存会话到历史记录
// 将用户输入和助手回复保存到历史记录存储
func (a *Agent) saveToHistory(userInput, assistantOutput string, toolCalls ...memory.ToolCallInfo) {
	if a.historyStore == nil {
		return
	}

	// 保存用户消息
	if userInput != "" {
		a.historyStore.Add(&memory.MemoryEntry{
			Role:    memory.RoleUser,
			Content: userInput,
		})

		// 记录到用户画像和对话历史
		msg := message.Message{
			Role:    "user",
			Content: userInput,
			Time:    time.Now(),
		}
		if a.profileManager != nil {
			a.profileManager.AddMessage(msg)
		}
		if a.conversationManager != nil {
			a.conversationManager.AddMessage(msg)
		}
	}

	// 保存助手消息
	if assistantOutput != "" {
		a.historyStore.Add(&memory.MemoryEntry{
			Role:      memory.RoleAssistant,
			Content:   assistantOutput,
			ToolCalls: toolCalls,
		})

		// 记录到用户画像和对话历史
		msg := message.Message{
			Role:    "assistant",
			Content: assistantOutput,
			Time:    time.Now(),
		}
		if a.profileManager != nil {
			a.profileManager.AddMessage(msg)
		}
		if a.conversationManager != nil {
			a.conversationManager.AddMessage(msg)
		}
	}
}
