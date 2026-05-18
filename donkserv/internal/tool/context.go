package tool

import (
	"context"
	"sync"
	"time"
)

// Context 工具执行上下文
// 用于在工具执行过程中传递信息
type Context struct {
	// 请求信息
	RequestID string         // 请求唯一标识
	ToolName  string         // 工具名称
	Params    map[string]any // 传入参数
	Metadata  map[string]any // 元数据信息

	// 执行控制
	Timeout    time.Duration      // 超时时间
	Deadline   time.Time          // 截止时间
	CancelFunc context.CancelFunc // 取消函数

	// 运行时
	Values     context.Context // Go 上下文
	OutputChan chan *Chunk     // 流式输出通道

	// 状态
	mu         sync.RWMutex // 读写锁
	startTime  time.Time    // 开始时间
	endTime    time.Time    // 结束时间
	status     Status       // 执行状态
	retryCount int          // 当前重试次数
}

// NewContext 创建新的执行上下文
func NewContext(toolName string, params map[string]any) *Context {
	ctx, cancel := context.WithCancel(context.Background())
	return &Context{
		RequestID:  generateRequestID(),
		ToolName:   toolName,
		Params:     params,
		Metadata:   make(map[string]any),
		Timeout:    30 * time.Second,
		Values:     ctx,
		OutputChan: make(chan *Chunk, 10),
		status:     StatusPending,
		CancelFunc: cancel,
	}
}

// Status 执行状态
type Status string

const (
	StatusPending  Status = "pending"  // 等待执行
	StatusRunning  Status = "running"  // 执行中
	StatusSuccess  Status = "success"  // 执行成功
	StatusFailed   Status = "failed"   // 执行失败
	StatusTimeout  Status = "timeout"  // 执行超时
	StatusCanceled Status = "canceled" // 执行取消
)

// Status 获取当前状态
func (c *Context) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// SetStatus 设置状态
func (c *Context) SetStatus(status Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

// Start 开始执行
func (c *Context) Start() {
	c.mu.Lock()
	c.startTime = time.Now()
	c.status = StatusRunning
	c.mu.Unlock()
}

// End 结束执行
func (c *Context) End() {
	c.mu.Lock()
	c.endTime = time.Now()
	c.mu.Unlock()
}

// Duration 获取执行时长
func (c *Context) Duration() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.startTime.IsZero() {
		return 0
	}

	end := c.endTime
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(c.startTime)
}

// IncrRetry 增加重试次数
func (c *Context) IncrRetry() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retryCount++
}

// RetryCount 获取重试次数
func (c *Context) RetryCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.retryCount
}

// SetMetadata 设置元数据
func (c *Context) SetMetadata(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Metadata[key] = value
}

// GetMetadata 获取元数据
func (c *Context) GetMetadata(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.Metadata[key]
	return val, ok
}

// WithTimeout 设置超时时间
func (c *Context) WithTimeout(timeout time.Duration) *Context {
	c.Timeout = timeout
	if c.Values != nil {
		c.Values, c.CancelFunc = context.WithTimeout(c.Values, timeout)
	}
	return c
}

// WithDeadline 设置截止时间
func (c *Context) WithDeadline(deadline time.Time) *Context {
	c.Deadline = deadline
	if c.Values != nil {
		c.Values, c.CancelFunc = context.WithDeadline(c.Values, deadline)
	}
	return c
}

// Done 获取完成通道
func (c *Context) Done() <-chan struct{} {
	return c.Values.Done()
}

// Err 获取错误
func (c *Context) Err() error {
	return c.Values.Err()
}

// Value 获取上下文值
func (c *Context) Value(key any) any {
	return c.Values.Value(key)
}

// SendChunk 发送流式输出
func (c *Context) SendChunk(chunk *Chunk) {
	select {
	case c.OutputChan <- chunk:
	case <-c.Values.Done():
	}
}

// CloseOutput 关闭输出通道
func (c *Context) CloseOutput() {
	close(c.OutputChan)
}

// Cancel 取消执行
func (c *Context) Cancel() {
	if c.CancelFunc != nil {
		c.CancelFunc()
	}
}

// Chunk 流式输出块
type Chunk struct {
	Type    string      `json:"type"`    // 块类型: text/json/progress/error
	Content interface{} `json:"content"` // 内容
}

// NewTextChunk 创建文本块
func NewTextChunk(text string) *Chunk {
	return &Chunk{Type: "text", Content: text}
}

// NewJSONChunk 创建JSON块
func NewJSONChunk(data any) *Chunk {
	return &Chunk{Type: "json", Content: data}
}

// NewProgressChunk 创建进度块
func NewProgressChunk(progress float64) *Chunk {
	return &Chunk{Type: "progress", Content: progress}
}

// NewErrorChunk 创建错误块
func NewErrorChunk(err error) *Chunk {
	return &Chunk{Type: "error", Content: err.Error()}
}
