package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEWriter SSE流式响应写入器
// 用于将数据以Server-Sent Events格式推送给客户端
type SSEWriter struct {
	writer  http.ResponseWriter // HTTP响应写入器
	flusher http.Flusher        // 刷新接口（用于实时推送）
	encoder *json.Encoder       // JSON编码器
}

// NewSSEWriter 创建SSE写入器
// 自动设置SSE所需的HTTP响应头
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	// 设置SSE必需的响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("http.ResponseWriter 不支持 Flush")
	}

	return &SSEWriter{
		writer:  w,
		flusher: flusher,
		encoder: json.NewEncoder(w),
	}
}

// Send 发送SSE事件
// event: 事件类型
// data: 事件数据
func (s *SSEWriter) Send(event, data string) error {
	defer func() {
		if r := recover(); r != nil {
			// 忽略panic，防止向已关闭的HTTP连接写入
		}
	}()

	if s.writer == nil || s.flusher == nil {
		return fmt.Errorf("SSE writer已经失效")
	}
	// 写入事件类型行
	fmt.Fprintf(s.writer, "event: %s\n", event)
	// 写入数据行（数据后需要两个换行表示结束）
	fmt.Fprintf(s.writer, "data: %s\n\n", data)
	// 刷新缓冲区，立即推送
	s.flusher.Flush()
	return nil
}

// SendJSON 发送JSON格式的SSE事件
func (s *SSEWriter) SendJSON(event string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.Send(event, string(jsonData))
}

// SendHeartbeat 发送心跳消息
// 用于保持SSE连接活跃，防止代理/负载均衡器关闭空闲连接
func (s *SSEWriter) SendHeartbeat() error {
	return s.Send("heartbeat", "ping")
}

// SendError 发送错误事件
func (s *SSEWriter) SendError(event string, err error) error {
	return s.Send(event, fmt.Sprintf(`{"error":%q}`, err.Error()))
}
