package main

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/longstageai/donk/donk/internal/websocket"
	appctx "github.com/longstageai/donk/donk/pkg/context"
	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationMessage struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Level   string `json:"level"`
}

var notificationTemplates = []struct {
	title   string
	content string
	level   string
}{
	{"系统通知", "系统运行正常，所有服务已就绪", "info"},
	{"任务完成", "您的后台任务已成功执行完毕", "success"},
	{"警告提示", "检测到资源使用率超过80%", "warning"},
	{"错误报告", "连接超时，请稍后重试", "error"},
	{"新消息", "您有一条新的系统消息待查看", "info"},
	{"操作成功", "数据保存成功", "success"},
	{"注意", "系统将在10分钟后进行维护", "warning"},
	{"连接恢复", "网络连接已恢复正常", "success"},
	{"安全提醒", "检测到异常登录尝试", "error"},
	{"更新提示", "新版本已可用，建议及时更新", "info"},
}

// SetupWebSocket 创建WebSocket事件推送服务
// 参数app是应用程序上下文，engine是gin引擎实例
// 返回WebSocket服务器实例和错误信息
// 自动注册的端点：
//   - GET /ws/events (WebSocket事件推送端点)
func SetupWebSocket(app *appctx.Application, engine *gin.Engine) (*websocket.Server, error) {
	// 创建WebSocket服务器
	wsServer := websocket.NewServer()
	// 注册WebSocket事件推送端点
	engine.GET("/ws/events", wsServer.HandleWebSocket)
	logger.Info("WebSocket事件推送端点已注册: GET /ws/events", nil)

	// 注册手动测试推送端点
	engine.POST("/ws/test-push", func(c *gin.Context) {
		clientCount := wsServer.Hub().ClientCount()
		logger.Info("手动推送请求", map[string]interface{}{"clientCount": clientCount})

		if clientCount == 0 {
			c.JSON(200, gin.H{"status": "no_clients", "clientCount": 0})
			return
		}

		template := notificationTemplates[rand.Intn(len(notificationTemplates))]
		msg := NotificationMessage{
			Type:    "notification",
			ID:      uuid.New().String(),
			Title:   template.title,
			Content: template.content,
			Level:   template.level,
		}
		data, _ := json.Marshal(msg)
		wsServer.Hub().BroadcastJSON(data)

		c.JSON(200, gin.H{"status": "sent", "clientCount": clientCount, "title": msg.Title})
	})
	logger.Info("测试推送端点已注册: POST /ws/test-push", nil)

	// 启动测试推送goroutine（每10秒推送一次测试事件）
	//go startTestPush(wsServer)

	logger.Info("WebSocket事件推送服务初始化成功", nil)
	return wsServer, nil
}

func startTestPush(wsServer *websocket.Server) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		// 检查当前连接数
		clientCount := wsServer.Hub().ClientCount()
		logger.Info("定时推送检查", map[string]interface{}{"clientCount": clientCount})

		if clientCount == 0 {
			logger.Info("没有客户端连接，跳过本次推送", nil)
			continue
		}

		// 随机选择一条消息模板
		template := notificationTemplates[rand.Intn(len(notificationTemplates))]

		msg := NotificationMessage{
			Type:    "notification",
			ID:      uuid.New().String(),
			Title:   template.title,
			Content: template.content,
			Level:   template.level,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			logger.Error("序列化通知消息失败", map[string]interface{}{"error": err.Error()})
			continue
		}

		// 广播给所有连接的客户端
		wsServer.Hub().BroadcastJSON(data)
		logger.Info("通知推送已发送", map[string]interface{}{"title": msg.Title, "level": msg.Level, "clientCount": clientCount})
	}
}
