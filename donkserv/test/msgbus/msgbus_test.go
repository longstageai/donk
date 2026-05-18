package msgbus_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/longstageai/donk/donk/internal/msgbus"
)

func TestAdapter_Usage(t *testing.T) {
	bus := msgbus.NewBus()

	adapter := msgbus.NewAdapter(bus, msgbus.NewServer(":9999"),
		msgbus.WithConnectHandler(func(clientID string) {
			fmt.Printf("[WS] 客户端连接: %s\n", clientID)
		}),
		msgbus.WithDisconnectHandler(func(clientID string) {
			fmt.Printf("[WS] 客户端断开: %s\n", clientID)
		}),
		msgbus.WithMessageHandler(func(clientID, topic string, payload interface{}) error {
			fmt.Printf("[WS] 收到客户端消息: clientID=%s, topic=%s, payload=%v\n",
				clientID, topic, payload)
			return nil
		}),
	)

	server := adapter.Server()
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("服务器错误: %v\n", err)
		}
	}()
	defer server.Stop()

	// 等待服务器启动
	time.Sleep(500 * time.Millisecond)

	handler1 := func(msg *msgbus.Message) error {
		fmt.Printf("[订阅者1] 收到消息: %v\n", msg.Payload)
		return nil
	}

	handler2 := func(msg *msgbus.Message) error {
		fmt.Printf("[订阅者2] 收到消息: %v\n", msg.Payload)
		return nil
	}

	sub1, err := bus.Subscribe("agent.001.task", handler1)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}

	sub2, err := bus.Subscribe("agent.002.task", handler2)
	if err != nil {
		t.Fatalf("订阅失败: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	//
	//dialer := websocket.DefaultDialer
	//ws, _, err := dialer.Dial("ws://localhost:9999", nil)
	//if err != nil {
	//	t.Fatalf("连接服务器失败: %v", err)
	//}
	//defer ws.Close()
	//
	//subscribeMsg := `{"type":"subscribe","topic":"agent.001.task"}`
	//if err := ws.WriteMessage(websocket.TextMessage, []byte(subscribeMsg)); err != nil {
	//	t.Fatalf("发送订阅消息失败: %v", err)
	//}
	//fmt.Println("已发送订阅消息")
	//
	//time.Sleep(100 * time.Millisecond)
	//
	//if err := bus.Publish("agent.001.task", map[string]interface{}{
	//	"content": "测试消息",
	//}, "test-sender"); err != nil {
	//	t.Fatalf("发布消息失败: %v", err)
	//}
	//
	time.Sleep(200 * time.Minute)

	bus.Unsubscribe(sub1)
	bus.Unsubscribe(sub2)

	fmt.Println("测试完成")
}
