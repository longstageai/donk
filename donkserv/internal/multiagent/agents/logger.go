// agents Agent日志工具
package agents

import (
	"encoding/json"

	"github.com/longstageai/donk/donk/internal/multiagent/types"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// AgentLogger Agent日志工具
type AgentLogger struct {
	logger *logger.Logger
}

// NewAgentLogger 创建Agent日志工具
func NewAgentLogger(log *logger.Logger) *AgentLogger {
	return &AgentLogger{logger: log}
}

// LogLLMInput 打印LLM输入参数
func (l *AgentLogger) LogLLMInput(agentName string, messages []types.Message, tools []types.ToolDefinition) {
	if l.logger == nil {
		return
	}

	messagesJSON, _ := json.MarshalIndent(messages, "", "  ")
	toolsJSON, _ := json.MarshalIndent(tools, "", "  ")

	l.logger.Debug("LLM调用输入", map[string]interface{}{
		"agent":    agentName,
		"messages": string(messagesJSON),
		"tools":    string(toolsJSON),
	})
}

// LogLLMOutput 打印LLM输出参数
func (l *AgentLogger) LogLLMOutput(agentName string, resp *types.LLMResponse) {
	if l.logger == nil {
		return
	}

	l.logger.Debug("LLM调用输出", map[string]interface{}{
		"agent":             agentName,
		"content":           resp.Content,
		"reasoning":         resp.Reasoning,
		"tool_calls_count":  len(resp.ToolCalls),
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
	})
}

// Info 打印信息日志
func (l *AgentLogger) Info(message string, fields map[string]interface{}) {
	if l.logger == nil {
		return
	}
	l.logger.Info(message, fields)
}
