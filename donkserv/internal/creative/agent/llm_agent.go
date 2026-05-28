package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/creative"
	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/profile"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// CreativeLLMClient 定义 creative Agent 调用大模型的最小接口，便于单元测试替换真实模型。
type CreativeLLMClient interface {
	Chat(ctx context.Context, req *schema.ChatRequest) (*schema.ChatResponse, error)
}

// SettingModelLLMClient 每次请求都会从 setting 模块获取最新 LLM 配置，并使用 model 模块发起调用。
type SettingModelLLMClient struct{}

// NewSettingModelLLMClient 创建基于 setting + model 的 LLM 客户端。
func NewSettingModelLLMClient() *SettingModelLLMClient {
	return &SettingModelLLMClient{}
}

// Chat 执行一次 LLM 调用。这里故意每次调用都重新读取 setting 配置，保证运行时配置变更立即生效。
func (c *SettingModelLLMClient) Chat(ctx context.Context, req *schema.ChatRequest) (*schema.ChatResponse, error) {
	provider, modelName, apiKey, baseURL, err := c.loadLLMConfig()
	if err != nil {
		return nil, err
	}
	llm, err := model.NewAdapter(provider, apiKey, modelName, baseURL)
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 适配器失败: %w", err)
	}
	if llm == nil {
		return nil, fmt.Errorf("不支持的 LLM 提供商: %s", provider)
	}
	if req == nil {
		req = &schema.ChatRequest{}
	}
	resp, err := llm.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}
	if resp != nil && resp.Error != nil {
		return nil, fmt.Errorf("LLM 返回错误: %s", resp.Error.Message)
	}
	return resp, nil
}

func (c *SettingModelLLMClient) loadLLMConfig() (string, string, string, string, error) {
	provider := setting.GetProvider()
	if provider == nil {
		return "", "", "", "", fmt.Errorf("setting 配置提供者未初始化")
	}
	llmConfig, err := provider.GetLLMConfig()
	if err != nil {
		return "", "", "", "", fmt.Errorf("获取 LLM 配置失败: %w", err)
	}
	if llmConfig == nil {
		return "", "", "", "", fmt.Errorf("LLM 配置为空")
	}
	return llmConfig.Provider, llmConfig.Model, llmConfig.APIKey, llmConfig.BaseURL, nil
}

// LLMAgent 是基于 LLM 的 creative Agent 实现。
type LLMAgent struct {
	id            creative.ID
	name          string
	role          creative.AgentRole
	handles       map[creative.EventType]bool
	prompt        PromptSpec
	promptBuilder DynamicPromptBuilder
	llm           CreativeLLMClient
	tools         *tool.Registry
	maxSteps      int
	outputFunc    func(context.Context, creative.AgentInput, string, creative.TokenUsage) creative.AgentOutput
	historyStore  *memory.HistoryStore // 历史记录存储（用于获取最近对话）
	profile       *profile.UserProfile // 用户画像（用于个性化目标生成）
}

// PromptSpec 描述单个 Agent 的完整提示词。
type PromptSpec struct {
	SystemPrompt string
	UserPrompt   string
	OutputFormat string
}

// LLMAgentOption 用于让单个 Agent 自己配置工具、循环步数等运行参数。
type LLMAgentOption func(*LLMAgent)

// WithTools 为单个 Agent 配置来自 tool 模块的工具注册表。
func WithTools(tools *tool.Registry) LLMAgentOption {
	return func(a *LLMAgent) {
		a.tools = tools
	}
}

// WithMaxSteps 配置单个 Agent 的内部 ReAct 循环最大步数。
func WithMaxSteps(maxSteps int) LLMAgentOption {
	return func(a *LLMAgent) {
		if maxSteps > 0 {
			a.maxSteps = maxSteps
		}
	}
}

// WithHistoryStore 配置单个 Agent 的历史记录存储。
func WithHistoryStore(store *memory.HistoryStore) LLMAgentOption {
	return func(a *LLMAgent) {
		a.historyStore = store
	}
}

// WithProfile 配置单个 Agent 的用户画像。
func WithProfile(profile *profile.UserProfile) LLMAgentOption {
	return func(a *LLMAgent) {
		a.profile = profile
	}
}

// NewLLMAgent 创建 LLM Agent。
func NewLLMAgent(id creative.ID, name string, role creative.AgentRole, handles []creative.EventType, prompt PromptSpec, llm CreativeLLMClient, outputFunc func(context.Context, creative.AgentInput, string, creative.TokenUsage) creative.AgentOutput, opts ...LLMAgentOption) *LLMAgent {
	m := make(map[creative.EventType]bool, len(handles))
	for _, eventType := range handles {
		m[eventType] = true
	}
	if llm == nil {
		llm = NewSettingModelLLMClient()
	}
	agent := &LLMAgent{id: id, name: name, role: role, handles: m, prompt: prompt, llm: llm, maxSteps: 5, outputFunc: outputFunc}
	for _, opt := range opts {
		if opt != nil {
			opt(agent)
		}
	}
	return agent
}

// DynamicPromptBuilder 动态提示词构建函数类型
type DynamicPromptBuilder func(input creative.AgentInput) PromptSpec

// NewLLMAgentWithDynamicPrompt 创建支持动态提示词的 LLM Agent。
func NewLLMAgentWithDynamicPrompt(id creative.ID, name string, role creative.AgentRole, handles []creative.EventType, promptBuilder DynamicPromptBuilder, llm CreativeLLMClient, outputFunc func(context.Context, creative.AgentInput, string, creative.TokenUsage) creative.AgentOutput, opts ...LLMAgentOption) *LLMAgent {
	agent := NewLLMAgent(id, name, role, handles, PromptSpec{}, llm, outputFunc, opts...)
	agent.promptBuilder = promptBuilder
	return agent
}

func (a *LLMAgent) ID() creative.ID          { return a.id }
func (a *LLMAgent) Name() string             { return a.name }
func (a *LLMAgent) Role() creative.AgentRole { return a.role }
func (a *LLMAgent) CanHandle(ctx context.Context, event creative.Event, room creative.Room) creative.ClaimDecision {
	if a.handles[event.Type] {
		return creative.ClaimDecision{CanClaim: true, Confidence: 1, Reason: "事件类型匹配", Priority: event.Priority}
	}
	return creative.ClaimDecision{CanClaim: false, Reason: "事件类型不匹配"}
}

// Handle 构建提示词、执行 Agent 内部 ReAct 循环，并将最终模型文本转换为 Runtime 可提交的 AgentOutput。
func (a *LLMAgent) Handle(ctx context.Context, input creative.AgentInput) creative.AgentOutput {
	var prompt PromptSpec
	if a.promptBuilder != nil {
		// 动态构建：Agent 自己管理完整的系统提示词和用户提示词
		prompt = a.promptBuilder(input)
	} else {
		// 静态提示词：使用预定义的 PromptSpec，用户提示词由基类构建
		prompt = a.prompt
		prompt.UserPrompt = a.buildUserPrompt(input, prompt)
	}

	messages := []schema.Message{
		{Role: "system", Content: prompt.SystemPrompt},
		{Role: "user", Content: prompt.UserPrompt},
	}
	resp, usage, err := a.runLoop(ctx, messages)
	if err != nil {
		return creative.AgentOutput{Status: creative.AgentRunFailed, Decision: creative.DecisionFailed, TokenUsage: usage, Error: err}
	}
	content := ""
	if resp != nil {
		content = strings.TrimSpace(resp.Content)
	}
	if a.outputFunc == nil {
		return creative.AgentOutput{Status: creative.AgentRunSucceeded, Decision: creative.DecisionSucceeded, TokenUsage: usage, Messages: []creative.MessageDraft{{Role: creative.MessageRoleAgent, Content: content}}}
	}
	return a.outputFunc(ctx, input, content, usage)
}

func (a *LLMAgent) runLoop(ctx context.Context, messages []schema.Message) (*schema.ChatResponse, creative.TokenUsage, error) {
	var totalUsage creative.TokenUsage
	var lastResp *schema.ChatResponse
	tools := a.toolDefinitions()
	for step := 0; step < a.maxSteps; step++ {
		a.logLLMInput(step+1, messages, tools)
		resp, err := a.llm.Chat(ctx, &schema.ChatRequest{Messages: messages, Tools: tools})
		if err != nil {
			return lastResp, totalUsage, err
		}
		lastResp = resp
		a.logLLMOutput(step+1, resp)
		mergeTokenUsage(&totalUsage, resp)
		if resp == nil || len(resp.ToolCalls) == 0 {
			return resp, totalUsage, nil
		}
		messages = append(messages, schema.Message{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls})
		for _, toolCall := range resp.ToolCalls {
			messages = append(messages, schema.Message{Role: "tool", Content: a.executeTool(ctx, toolCall), ToolCallID: toolCall.ID})
		}
	}
	return lastResp, totalUsage, nil
}

func (a *LLMAgent) buildUserPrompt(input creative.AgentInput, prompt PromptSpec) string {
	if a.promptBuilder != nil {
		// 动态 Agent 自己管理完整的用户提示词
		return prompt.UserPrompt
	}

	contextJSON, _ := json.MarshalIndent(buildLLMContextView(input), "", "  ")
	return fmt.Sprintf(`当前事件：%s
当前阶段：%s

请基于以下 Runtime 上下文完成你的职责：
%s

输出要求：
%s`, input.Event.Type, input.Session.CurrentPhase, string(contextJSON), prompt.OutputFormat)
}

func (a *LLMAgent) toolDefinitions() []schema.ToolDefinition {
	if a.tools == nil {
		return nil
	}
	return a.tools.GetToolDefinitions()
}

func (a *LLMAgent) executeTool(ctx context.Context, toolCall schema.ToolCall) string {
	if a.tools == nil {
		return "工具执行失败：当前 Agent 未配置工具注册表"
	}
	toolName := toolCall.Function.Name
	t, ok := a.tools.Get(toolName)
	if !ok || t == nil {
		return fmt.Sprintf("工具执行失败：未找到工具 %s", toolName)
	}
	params := map[string]any{}
	if strings.TrimSpace(toolCall.Function.Arguments) != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return fmt.Sprintf("工具执行失败：参数解析失败 %v", err)
		}
	}
	toolCtx := tool.NewContext(toolName, params)
	toolCtx.Values = ctx
	toolCtx.Metadata["agent_id"] = a.id
	result, err := t.Execute(toolCtx)
	if err != nil {
		return fmt.Sprintf("工具执行失败：%v", err)
	}
	return result.String()
}

func mergeTokenUsage(total *creative.TokenUsage, resp *schema.ChatResponse) {
	if total == nil || resp == nil {
		return
	}
	total.PromptTokens += resp.Usage.PromptTokens
	total.CompletionTokens += resp.Usage.CompletionTokens
	total.TotalTokens += resp.Usage.TotalTokens
	if resp.Model != "" {
		total.ModelName = resp.Model
	}
}

func (a *LLMAgent) logLLMInput(step int, messages []schema.Message, tools []schema.ToolDefinition) {
	payload, _ := json.Marshal(messages)
	logger.Debug("creative Agent LLM 调用输入", map[string]interface{}{"agent_id": a.id, "agent_name": a.name, "step": step, "messages": string(payload), "tool_count": len(tools)})
}

func (a *LLMAgent) logLLMOutput(step int, resp *schema.ChatResponse) {
	if resp == nil {
		return
	}
	logger.Debug("creative Agent LLM 调用输出", map[string]interface{}{"agent_id": a.id, "agent_name": a.name, "step": step, "content": resp.Content, "tool_call_count": len(resp.ToolCalls), "prompt_tokens": resp.Usage.PromptTokens, "completion_tokens": resp.Usage.CompletionTokens, "total_tokens": resp.Usage.TotalTokens})
}

func buildLLMContextView(input creative.AgentInput) map[string]any {
	messages := make([]map[string]any, 0, len(input.Messages))
	for _, message := range input.Messages {
		messages = append(messages, map[string]any{"agent_id": message.AgentID, "role": message.Role, "content": message.Content, "artifact_ids": message.ArtifactIDs})
	}
	artifacts := make([]map[string]any, 0, len(input.Artifacts))
	for _, artifact := range input.Artifacts {
		artifacts = append(artifacts, map[string]any{"id": artifact.ID, "type": artifact.Type, "agent_id": artifact.AgentID, "data": artifact.Data})
	}
	return map[string]any{"event": input.Event, "session": input.Session, "messages": messages, "artifacts": artifacts, "snapshot": input.Snapshot}
}

func promptSpec(systemPrompt string) PromptSpec {
	return PromptSpec{SystemPrompt: systemPrompt, OutputFormat: "请严格按系统提示词中的输出格式回答。评审类 Agent 必须明确写出 判定结果：通过 或 判定结果：不通过，并给出原因。"}
}

func parseAgentDecision(content string) creative.AgentDecision {

	lower := strings.ToLower(content)
	if strings.Contains(content, "失败") || strings.Contains(lower, "failed") || strings.Contains(lower, "failure") || strings.Contains(lower, `"decision":"failed"`) || strings.Contains(lower, `"decision": "failed"`) {
		return creative.DecisionFailed
	}
	if strings.Contains(content, "否决") || strings.Contains(content, "不通过") || strings.Contains(content, "未通过") || strings.Contains(lower, "reject") || strings.Contains(lower, "rejected") || strings.Contains(lower, "not pass") || strings.Contains(lower, `"decision":"rejected"`) || strings.Contains(lower, `"decision": "rejected"`) || strings.Contains(lower, "fail") {
		return creative.DecisionRejected
	}
	if strings.Contains(content, "通过") || strings.Contains(content, "继续") || strings.Contains(lower, "pass") || strings.Contains(lower, "approved") || strings.Contains(lower, "continue") || strings.Contains(lower, `"decision":"succeeded"`) || strings.Contains(lower, `"decision": "succeeded"`) {
		return creative.DecisionSucceeded
	}
	return creative.DecisionRejected
}

func extractJSONField(content string, field string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return ""
	}
	value, ok := data[field]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
