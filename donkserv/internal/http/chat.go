package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/agent"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/stream"
	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ChatRequest 聊天请求结构
// 客户端发送的聊天请求消息体
type ChatRequest struct {
	Content  string `json:"content" binding:"required"`
	FilePath string `json:"file_path,omitempty"`
	FileType string `json:"file_type,omitempty"`
}

// ChatHandler 聊天HTTP处理器
// 通过HTTP SSE协议实现与客户端的双向对话
type ChatHandler struct {
	agentService  *agent.AgentService  // Agent服务实例
	providerCache *model.ProviderCache // LLM Provider缓存
}

// NewChatHandler 创建聊天HTTP处理器实例
// 参数agentSvc是Agent服务实例，providerCache是LLM Provider缓存
func NewChatHandler(agentSvc *agent.AgentService, providerCache *model.ProviderCache) *ChatHandler {
	return &ChatHandler{
		agentService:  agentSvc,
		providerCache: providerCache,
	}
}

// RegisterRoutes 注册聊天路由
// 将 POST /api/v1/chat 路由注册到gin引擎
func (h *ChatHandler) RegisterRoutes(e *gin.Engine) {
	e.POST("/api/v1/chat", h.HandleChat)
}

// HandleChat 处理聊天请求
// POST /api/chat
// 接收用户输入，通过Agent执行并以SSE流式返回结果
func (h *ChatHandler) HandleChat(c *gin.Context) {
	// 解析并验证请求
	req, ok := h.parseRequest(c)
	if !ok {
		return
	}

	logger.Info("收到聊天请求", map[string]interface{}{
		"content":   req.Content,
		"file_path": req.FilePath,
		"file_type": req.FileType,
	})

	// 获取Agent实例
	agentInstance, ok := h.getAgentInstance(c)
	if !ok {
		return
	}

	// 设置LLM Provider配置
	if ok := h.setupLLMProvider(c, agentInstance); !ok {
		return
	}

	// 获取超时配置
	timeout := h.getAgentTimeout()

	// 执行Agent对话
	h.executeAgentChat(c, agentInstance, req.AgentContent(), timeout)
}

// parseRequest 解析并验证聊天请求
//
// 参数:
//   - c: gin上下文
//
// 返回:
//   - ChatRequest: 解析后的请求
//   - bool: 是否成功
func (h *ChatHandler) parseRequest(c *gin.Context) (ChatRequest, bool) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return ChatRequest{}, false
	}

	req.Content = strings.TrimSpace(req.Content)
	req.FilePath = strings.TrimSpace(req.FilePath)
	req.FileType = strings.ToLower(strings.TrimSpace(req.FileType))

	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息内容不能为空"})
		return ChatRequest{}, false
	}

	if req.FilePath != "" && !isSupportedChatFileType(req.FileType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型: " + req.FileType})
		return ChatRequest{}, false
	}

	return req, true
}

func (r ChatRequest) AgentContent() string {
	if r.FilePath == "" {
		return r.Content
	}

	return fmt.Sprintf("文件类型：%s\n文件路径：%s\n需求：%s", r.FileType, r.FilePath, r.Content)
}

func isSupportedChatFileType(fileType string) bool {
	switch fileType {
	case "pdf", "docx", "txt", "md":
		return true
	default:
		return false
	}
}

// getAgentInstance 获取Agent实例
//
// 参数:
//   - c: gin上下文
//
// 返回:
//   - *agent.Agent: Agent实例
//   - bool: 是否成功
func (h *ChatHandler) getAgentInstance(c *gin.Context) (*agent.Agent, bool) {
	if h.agentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent服务未初始化"})
		return nil, false
	}

	agentInstance := h.agentService.GetAgent()
	if agentInstance == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agent实例未初始化"})
		return nil, false
	}

	return agentInstance, true
}

// setupLLMProvider 设置LLM Provider配置
// 从数据库读取配置并设置到Agent
//
// 参数:
//   - c: gin上下文
//   - agentInstance: Agent实例
//
// 返回:
//   - bool: 是否成功
func (h *ChatHandler) setupLLMProvider(c *gin.Context, agentInstance *agent.Agent) bool {
	// 检查Provider缓存
	if h.providerCache == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Provider缓存未初始化"})
		return false
	}

	// 获取配置提供者
	provider := setting.GetProvider()
	if provider == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ConfigProvider未初始化"})
		return false
	}

	// 获取LLM配置
	llmCfg, err := provider.GetLLMConfig()
	if err != nil {
		logger.Error("获取LLM配置失败", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取LLM配置失败"})
		return false
	}
	if llmCfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM配置不存在"})
		return false
	}

	// 获取Provider Adapter
	adapter, exists := h.providerCache.Get(llmCfg.Provider)
	if !exists {
		logger.Error("Provider不在缓存中", map[string]interface{}{"provider": llmCfg.Provider})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持的LLM Provider: " + llmCfg.Provider})
		return false
	}

	// 设置Provider配置并更新Agent
	adapter.SetConfig(llmCfg.Model, llmCfg.APIKey, llmCfg.BaseURL)
	agentInstance.SetModel(adapter)

	return true
}

// getAgentTimeout 获取Agent执行超时时间
// 从数据库配置读取，默认300秒
//
// 返回:
//   - int: 超时时间（秒）
func (h *ChatHandler) getAgentTimeout() int {
	provider := setting.GetProvider()
	if provider == nil {
		return 300
	}

	if agentCfg, err := provider.GetAgentConfig(); err == nil && agentCfg != nil && agentCfg.Timeout > 0 {
		return agentCfg.Timeout
	}

	return 300
}

// executeAgentChat 执行Agent对话
// 设置SSE流式响应并执行Agent
//
// 参数:
//   - c: gin上下文
//   - agentInstance: Agent实例
//   - content: 用户输入内容
//   - timeout: 超时时间（秒）
func (h *ChatHandler) executeAgentChat(c *gin.Context, agentInstance *agent.Agent, content string, timeout int) {
	// 创建带超时的上下文，同时监听HTTP请求的取消信号
	// 使用c.Request.Context()确保客户端断开连接时能及时取消Agent执行
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 创建SSE写入器
	sseWriter := stream.NewSSEWriter(c.Writer)

	// 启动心跳保持连接
	heartbeatStop := h.startHeartbeat(sseWriter)
	defer close(heartbeatStop)

	// 设置流式回调
	agentInstance.SetStreamCallback(func(event *agent.StreamEvent) {
		if err := h.sendStreamEvent(sseWriter, event); err != nil {
			logger.Error("发送SSE事件失败", map[string]interface{}{
				"event": string(event.Type),
				"error": err.Error(),
			})
		}
	})

	// 执行Agent对话
	if err := agentInstance.RunStream(ctx, content); err != nil {
		logger.Error("Agent执行出错", map[string]interface{}{"error": err.Error()})
		h.sendStreamEvent(sseWriter, &agent.StreamEvent{
			Type:  h.getErrorEventType(err),
			Error: err.Error(),
		})
	}
}

// startHeartbeat 启动心跳goroutine
// 定期发送心跳保持SSE连接活跃
//
// 参数:
//   - sseWriter: SSE写入器
//
// 返回:
//   - chan struct{}: 停止信号通道
func (h *ChatHandler) startHeartbeat(sseWriter *stream.SSEWriter) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if err := sseWriter.SendHeartbeat(); err != nil {
					logger.Error("发送心跳失败", map[string]interface{}{"error": err.Error()})
					return
				}
			}
		}
	}()

	return stop
}

// sendStreamEvent 发送流式事件到SSE客户端
// 将Agent产生的StreamEvent转换为SSE格式并发送
//
// 参数:
//   - sseWriter: SSE写入器
//   - event: 流式事件
//
// 返回:
//   - error: 错误信息
func (h *ChatHandler) sendStreamEvent(sseWriter *stream.SSEWriter, event *agent.StreamEvent) error {
	response := StreamResponse{
		Type:             "stream",
		Event:            string(event.Type),
		Content:          event.Content,
		ReasoningContent: event.ReasoningContent,
		ToolName:         event.ToolName,
		ToolInput:        event.ToolInput,
		ToolResult:       event.ToolResult,
		Error:            event.Error,
	}

	return sseWriter.SendJSON(string(event.Type), response)
}

// StreamResponse 流式响应结构
// 服务端返回给客户端的流式响应消息体
type StreamResponse struct {
	Type             string `json:"type"`                        // 消息类型："stream"
	Event            string `json:"event"`                       // 事件类型
	Content          string `json:"content,omitempty"`           // 文本内容
	ReasoningContent string `json:"reasoning_content,omitempty"` // 思考过程内容
	ToolName         string `json:"tool_name,omitempty"`         // 工具名称
	ToolInput        string `json:"tool_input,omitempty"`        // 工具输入参数
	ToolResult       string `json:"tool_result,omitempty"`       // 工具执行结果
	Error            string `json:"error,omitempty"`             // 错误信息
}

// getErrorEventType 根据错误类型返回对应的事件类型
// 将Agent错误映射为SSE事件类型
//
// 参数:
//   - err: 错误信息
//
// 返回:
//   - agent.StreamEventType: 事件类型
func (h *ChatHandler) getErrorEventType(err error) agent.StreamEventType {
	switch err {
	case agent.ErrMaxLoopExceeded:
		return agent.EventError
	case agent.ErrTokenBudgetExceeded:
		return agent.EventError
	case agent.ErrConvergeTimeout:
		return agent.EventError
	case agent.ErrCanceled:
		return agent.EventCanceled
	case agent.ErrAgentStopped:
		return agent.EventStop
	case agent.ErrTaskCompleted:
		return agent.EventStop
	default:
		return agent.EventError
	}
}
