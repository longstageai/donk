package agent

import (
	"context"
	"sync"

	"github.com/longstageai/donk/donk/pkg/config"
)

// Task 代表一个 Agent 任务
// 包含任务ID、输入内容、客户端ID、上下文、事件通道等
// 注意：Task主要用于WebSocket模式，SSE模式直接通过回调返回事件
type Task struct {
	ID       string            // 任务唯一标识（与ClientID相同）
	Input    string            // 用户输入内容
	ClientID string            // 客户端ID，用于关联事件通道
	Context  context.Context   // 任务上下文，用于取消操作
	Events   chan *StreamEvent // 事件通道，用于接收Agent的流式事件（WebSocket模式使用）
	Done     chan struct{}     // 任务完成信号
	Err      error             // 任务执行错误
}

// AgentService Agent服务层
// 负责管理Agent任务的执行
// 支持两种模式：
//   - WebSocket模式：通过Task任务队列和channel与WebSocket客户端通信
//   - HTTP SSE模式：直接通过Agent回调返回事件（见http/chat.go）
type AgentService struct {
	agent   *Agent                       // Agent实例
	config  *config.Config               // 配置对象
	tasks   chan *Task                   // 任务队列（WebSocket模式使用）
	mu      sync.RWMutex                 // 读写锁，保护clients map
	clients map[string]chan *StreamEvent // 客户端事件通道映射（WebSocket模式使用）
}

// GetConfig 返回配置对象
func (s *AgentService) GetConfig() *config.Config {
	return s.config
}

// NewAgentService 创建AgentService实例
// 参数agent是底层的Agent实例，config是配置对象
func NewAgentService(agent *Agent, config *config.Config) *AgentService {
	svc := &AgentService{
		agent:   agent,
		config:  config,
		tasks:   make(chan *Task, 100),
		clients: make(map[string]chan *StreamEvent),
	}
	return svc
}

// GetAgent 返回Agent实例
// 用于HTTP SSE handler直接调用Agent执行
func (s *AgentService) GetAgent() *Agent {
	return s.agent
}

// UnregisterClient 注销客户端
// 关闭并删除指定客户端的事件通道
// 当客户端断开连接时调用，释放资源
func (s *AgentService) UnregisterClient(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[clientID]; ok {
		delete(s.clients, clientID)
		// 不再关闭channel，避免与回调中的发送产生竞态
		// channel会在HTTP handler的goroutine结束时自动被GC回收
	}
}

// SubmitTask 提交任务（WebSocket模式）
// 将用户输入封装为任务并加入任务队列，返回任务对象
// 任务完成后可通过task.Events通道接收Agent的流式事件
// 注意：HTTP SSE模式不使用此方法，直接调用GetAgent()获取Agent实例
func (s *AgentService) SubmitTask(clientID string, input string, ctx context.Context) *Task {
	task := &Task{
		ID:       clientID,
		Input:    input,
		ClientID: clientID,
		Context:  ctx,
		Events:   make(chan *StreamEvent, 100),
		Done:     make(chan struct{}),
	}

	// 将任务的事件通道注册到 clients map
	events := task.Events
	s.mu.Lock()
	s.clients[clientID] = events
	s.mu.Unlock()

	// 提交任务到队列
	s.tasks <- task

	return task
}

// Start 启动Agent服务
// 开启一个goroutine来处理任务队列中的任务
// 阻塞直到上下文取消
func (s *AgentService) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-s.tasks:
				s.runTask(task)
			}
		}
	}()
}

// runTask 执行单个任务
// 设置Agent的流式回调，将Agent产生的事件通过task.Events通道发送
func (s *AgentService) runTask(task *Task) {
	defer func() {
		// 任务完成后清理客户端注册
		s.UnregisterClient(task.ClientID)
		close(task.Done)
	}()

	// 设置流式回调
	s.agent.SetStreamCallback(func(event *StreamEvent) {
		select {
		case task.Events <- event:
		case <-task.Context.Done():
			// 上下文已取消，不再发送事件
		}
	})

	// 执行Agent的流式对话
	err := s.agent.RunStream(task.Context, task.Input)
	task.Err = err

	// 如果执行出错，发送错误事件给客户端
	if err != nil {
		select {
		case task.Events <- &StreamEvent{Type: EventError, Error: err.Error()}:
		case <-task.Context.Done():
		}
	}
}
