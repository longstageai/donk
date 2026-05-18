package handler

import (
	"encoding/json"
	"fmt"
	"github.com/longstageai/donk/donk/internal/agent"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/websocket"
	"log"
)

// ChatMessage 聊天消息结构
// 客户端发送的聊天请求消息体
type ChatMessage struct {
	Type    string `json:"type"`    // 消息类型
	Content string `json:"content"` // 消息内容
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

// ChatHandler 聊天消息处理器
// 负责处理WebSocket收到的聊天消息，并协调Agent服务执行任务
type ChatHandler struct {
	hub          *websocket.Hub      // WebSocket Hub引用
	agentService *agent.AgentService // Agent服务实例
}

// NewChatHandler 创建聊天处理器实例
// 参数hub是WebSocket连接管理器，agentSvc是Agent服务
func NewChatHandler(hub *websocket.Hub, agentSvc *agent.AgentService) *ChatHandler {
	return &ChatHandler{hub: hub, agentService: agentSvc}
}

// GetMessageType 返回该处理器感兴趣的消息类型
// 实现BusinessHandler接口
func (h *ChatHandler) GetMessageType() websocket.MessageType {
	return websocket.TypeChat
}

// Handle 处理接收到的聊天消息
// 解析消息，提交给Agent服务执行，并处理返回的流式事件
func (h *ChatHandler) Handle(msg *websocket.Message) {
	// 解析客户端发送的JSON消息
	var chatMsg ChatMessage
	if err := json.Unmarshal(msg.Content, &chatMsg); err != nil {
		log.Printf("解析聊天消息失败: %v", err)
		h.sendError(msg.Client, "解析消息失败: "+err.Error())
		return
	}

	// 记录收到的聊天消息
	logger.Info(fmt.Sprintf("收到聊天消息: %s", chatMsg.Content), nil)

	// 检查Agent服务是否初始化
	if h.agentService == nil {
		h.sendError(msg.Client, "Agent 服务未初始化")
		return
	}

	// 提交任务到Agent服务
	clientID := msg.Client.ID
	task := h.agentService.SubmitTask(clientID, chatMsg.Content, msg.Client.Context)

	// 启动goroutine处理Agent返回的流式事件
	go func() {
		for {
			select {
			case event, ok := <-task.Events:
				if !ok {
					return
				}
				h.sendStreamEvent(msg.Client, event)
			case <-msg.Client.Context.Done():
				h.agentService.UnregisterClient(clientID)
				return
			}
		}
	}()
}

// sendStreamEvent 发送流式事件到客户端
// 将Agent产生的事件序列化为JSON并发送到WebSocket客户端
func (h *ChatHandler) sendStreamEvent(client *websocket.Client, event *agent.StreamEvent) {
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
	responseBytes, err := json.Marshal(response)
	if err != nil {
		logger.Error(fmt.Sprintf("序列化流式响应失败: %v", err), nil)
		return
	}
	client.Send(responseBytes)
}

// sendError 发送错误消息到客户端
func (h *ChatHandler) sendError(client *websocket.Client, errMsg string) {
	response := StreamResponse{
		Type:  "error",
		Error: errMsg,
	}
	responseBytes, _ := json.Marshal(response)
	client.Send(responseBytes)
}
