package main

import (
	"encoding/json"
	"log"

	"github.com/longstageai/donk/donk/pkg/websocket"
)

// ChatMessage 聊天消息结构
type ChatMessage struct {
	Type    string `json:"type"`    // 消息类型
	Content string `json:"content"` // 消息内容
}

// ChatHandler 聊天消息处理器
type ChatHandler struct {
	hub *websocket.Hub // WebSocket Hub引用
}

// GetMessageType 返回该处理器感兴趣的消息类型
func (h *ChatHandler) GetMessageType() websocket.MessageType {
	return websocket.TypeChat
}

// Handle 处理聊天消息
func (h *ChatHandler) Handle(msg *websocket.Message) {
	var chatMsg ChatMessage
	if err := json.Unmarshal(msg.Content, &chatMsg); err != nil {
		log.Printf("解析聊天消息失败: %v", err)
		return
	}

	log.Printf("收到聊天消息: %s", chatMsg.Content)

	// 构造响应消息
	response := map[string]string{
		"type":    "chat_response",
		"content": "服务器收到: " + chatMsg.Content,
	}
	responseBytes, _ := json.Marshal(response)
	msg.Client.Send(responseBytes)
}

// NotificationHandler 通知消息处理器
type NotificationHandler struct {
	hub *websocket.Hub // WebSocket Hub引用
}

// GetMessageType 返回该处理器感兴趣的消息类型
func (h *NotificationHandler) GetMessageType() websocket.MessageType {
	return websocket.TypeNotification
}

// Handle 处理通知消息
func (h *NotificationHandler) Handle(msg *websocket.Message) {
	log.Printf("收到通知消息: %s", string(msg.Content))

	// 构造响应消息
	response := map[string]string{
		"type":    "notification_response",
		"content": "通知已收到",
	}
	responseBytes, _ := json.Marshal(response)
	msg.Client.Send(responseBytes)
}

// CustomTokenValidator 自定义Token校验器
type CustomTokenValidator struct{}

// ValidateToken 校验Token是否有效
// 只有Token长度大于0时才通过
func (v *CustomTokenValidator) ValidateToken(token string) bool {
	log.Printf("校验 Token: %s", token)
	return len(token) > 0
}

// main 程序入口
func main() {
	// 创建WebSocket服务器，监听8080端口
	server := websocket.NewServer(":8080")

	// 设置自定义Token校验器
	server.SetTokenValidator(&CustomTokenValidator{})

	// 创建消息处理器
	chatHandler := &ChatHandler{hub: server.GetHub()}
	notificationHandler := &NotificationHandler{hub: server.GetHub()}

	// 注册消息处理器到路由器
	router := server.GetRouter()
	router.Register(chatHandler)
	router.Register(notificationHandler)

	// 打印服务器启动信息
	log.Printf("WebSocket 服务器启动")
	log.Printf("服务器地址: ws://localhost:8080/ws")
	log.Printf("Header Token: Authorization: <token>")
	log.Printf("健康检查: http://localhost:8080/health")
	log.Printf("")
	log.Printf("测试消息格式:")
	log.Printf(`  聊天消息: {"type": "chat", "content": "hello"}`)
	log.Printf(`  通知消息: {"type": "notification", "content": "notice"}`)

	// 启动服务器（阻塞）
	server.Start()
}
