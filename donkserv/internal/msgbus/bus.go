package msgbus

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// ErrTopicNotFound 主题不存在错误
var ErrTopicNotFound = errors.New("topic not found")

// ErrSubscriberNotFound 订阅者不存在错误
var ErrSubscriberNotFound = errors.New("subscriber not found")

// Handler 消息处理函数类型
// 定义订阅者处理消息的函数签名
type Handler func(msg *Message) error

// Subscriber 订阅者结构
// 订阅特定主题并处理收到消息
type Subscriber struct {
	// 订阅者唯一标识符
	ID string

	// 订阅的主题列表
	Topics []string

	// 消息处理函数
	Handler Handler

	// 消息队列（用于保证有序）
	queue chan *Message

	// 退出信号
	done chan struct{}

	// 所属的 Bus 实例
	bus *Bus
}

// Bus 消息总线核心结构
// 实现发布-订阅模式，负责消息的路由和分发
type Bus struct {
	// 主题到消息队列的映射
	// 每个主题有自己的 FIFO 队列，保证该主题消息有序
	topics map[string]chan *Message

	// 主题到订阅者列表的映射
	// 每个主题可以有一个或多个订阅者
	subscribers map[string][]*Subscriber

	// 已注册的订阅者集合（用于快速查找）
	allSubscribers map[string]*Subscriber

	// 同步保护
	mu sync.RWMutex

	// 队列缓冲区大小
	queueSize int

	// 消息处理超时时间
	handlerTimeout time.Duration

	// 是否正在运行
	isRunning bool

	// 退出信号
	done chan struct{}
}

// BusOption 函数选项模式，用于配置 Bus
type BusOption func(*Bus)

// WithQueueSize 设置消息队列大小
// size: 队列缓冲区大小
// 返回: 配置函数
func WithQueueSize(size int) BusOption {
	return func(b *Bus) {
		b.queueSize = size
	}
}

// WithHandlerTimeout 设置消息处理超时时间
// timeout: 超时时间
// 返回: 配置函数
func WithHandlerTimeout(timeout time.Duration) BusOption {
	return func(b *Bus) {
		b.handlerTimeout = timeout
	}
}

// NewBus 创建并初始化消息总线
// opts: 配置选项
// 返回: 初始化好的消息总线实例
func NewBus(opts ...BusOption) *Bus {
	bus := &Bus{
		topics:         make(map[string]chan *Message),
		subscribers:    make(map[string][]*Subscriber),
		allSubscribers: make(map[string]*Subscriber),
		queueSize:      256,
		handlerTimeout: 30 * time.Second,
		done:           make(chan struct{}),
		isRunning:      true,
	}

	// 应用选项
	for _, opt := range opts {
		opt(bus)
	}

	logger.Debug("消息总线已创建", map[string]interface{}{
		"queueSize": bus.queueSize,
		"timeout":   bus.handlerTimeout.String(),
	})

	return bus
}

// Publish 发布消息到指定主题
// topic: 主题名称
// payload: 消息负载
// sender: 发送者标识
// 返回: 错误信息
func (b *Bus) Publish(topic string, payload interface{}, sender string) error {
	msg := &Message{
		ID:        generateMessageID(),
		Type:      TypeMessage,
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
		Sender:    sender,
	}

	b.mu.RLock()
	queue, ok := b.topics[topic]
	b.mu.RUnlock()

	if !ok {
		// 主题不存在，静默处理（也可以选择创建）
		logger.Debug("消息发布到不存在的主题", map[string]interface{}{
			"topic": topic,
		})
		return nil
	}

	// 放入队列（阻塞直到队列有空位）
	select {
	case queue <- msg:
		logger.Debug("消息已发布到主题", map[string]interface{}{
			"topic":     topic,
			"messageID": msg.ID,
		})
		return nil
	case <-b.done:
		return errors.New("bus is shutting down")
	case <-time.After(b.handlerTimeout):
		return errors.New("publish timeout: queue is full")
	}
}

// Subscribe 订阅指定主题
// topic: 主题名称
// handler: 消息处理函数
// 返回: 订阅者实例和错误信息
func (b *Bus) Subscribe(topic string, handler Handler) (*Subscriber, error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	sub := &Subscriber{
		ID:      generateSubscriberID(),
		Topics:  []string{topic},
		Handler: handler,
		queue:   make(chan *Message, 256),
		done:    make(chan struct{}),
		bus:     b,
	}

	b.mu.Lock()

	// 确保主题存在
	if b.topics[topic] == nil {
		b.topics[topic] = make(chan *Message, b.queueSize)
		// 启动主题消费协程
		go b.consumeTopic(topic)
	}

	// 注册订阅者
	b.subscribers[topic] = append(b.subscribers[topic], sub)
	b.allSubscribers[sub.ID] = sub

	b.mu.Unlock()

	// 启动订阅者消费协程
	go sub.consume()

	logger.Debug("订阅者已订阅主题", map[string]interface{}{
		"subscriberID": sub.ID,
		"topic":        topic,
	})

	return sub, nil
}

// SubscribeMultiple 订阅多个主题
// topics: 主题名称列表
// handler: 消息处理函数
// 返回: 订阅者实例和错误信息
func (b *Bus) SubscribeMultiple(topics []string, handler Handler) (*Subscriber, error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	sub := &Subscriber{
		ID:      generateSubscriberID(),
		Topics:  topics,
		Handler: handler,
		queue:   make(chan *Message, 256),
		done:    make(chan struct{}),
		bus:     b,
	}

	b.mu.Lock()

	// 确保所有主题存在并注册订阅者
	for _, topic := range topics {
		if b.topics[topic] == nil {
			b.topics[topic] = make(chan *Message, b.queueSize)
			go b.consumeTopic(topic)
		}
		b.subscribers[topic] = append(b.subscribers[topic], sub)
	}

	b.allSubscribers[sub.ID] = sub

	b.mu.Unlock()

	// 启动订阅者消费协程
	go sub.consume()

	logger.Debug("订阅者已订阅多个主题", map[string]interface{}{
		"subscriberID": sub.ID,
		"topics":       topics,
	})

	return sub, nil
}

// Unsubscribe 取消订阅
// sub: 要取消的订阅者实例
// 返回: 错误信息
func (b *Bus) Unsubscribe(sub *Subscriber) error {
	if sub == nil {
		return ErrSubscriberNotFound
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// 从所有订阅的主题中移除
	for _, topic := range sub.Topics {
		subs := b.subscribers[topic]
		for i, s := range subs {
			if s.ID == sub.ID {
				// 移除订阅者
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}

	// 从全局订阅者集合中移除
	delete(b.allSubscribers, sub.ID)

	// 关闭订阅者的队列和退出信号
	close(sub.done)
	close(sub.queue)

	logger.Debug("订阅者已取消订阅", map[string]interface{}{
		"subscriberID": sub.ID,
	})

	return nil
}

// UnsubscribeByID 根据ID取消订阅
// subscriberID: 订阅者ID
// 返回: 错误信息
func (b *Bus) UnsubscribeByID(subscriberID string) error {
	b.mu.RLock()
	sub, ok := b.allSubscribers[subscriberID]
	b.mu.RUnlock()

	if !ok {
		return ErrSubscriberNotFound
	}

	return b.Unsubscribe(sub)
}

// GetSubscriber 获取订阅者
// subscriberID: 订阅者ID
// 返回: 订阅者实例和是否存在的布尔值
func (b *Bus) GetSubscriber(subscriberID string) (*Subscriber, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	sub, ok := b.allSubscribers[subscriberID]
	return sub, ok
}

// TopicExists 检查主题是否存在
// topic: 主题名称
// 返回: 是否存在
func (b *Bus) TopicExists(topic string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.topics[topic] != nil
}

// TopicSubscriberCount 获取主题的订阅者数量
// topic: 主题名称
// 返回: 订阅者数量
func (b *Bus) TopicSubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[topic])
}

// TotalSubscribers 获取总订阅者数量
// 返回: 总订阅者数量
func (b *Bus) TotalSubscribers() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.allSubscribers)
}

// Shutdown 关闭消息总线
// 停止所有订阅者并清理资源
func (b *Bus) Shutdown() {
	b.mu.Lock()
	if !b.isRunning {
		b.mu.Unlock()
		return
	}
	b.isRunning = false
	b.mu.Unlock()

	logger.Debug("消息总线正在关闭...", nil)

	// 关闭退出信号
	close(b.done)

	// 关闭所有订阅者
	b.mu.RLock()
	for _, sub := range b.allSubscribers {
		close(sub.done)
		close(sub.queue)
	}
	b.mu.RUnlock()

	logger.Debug("消息总线已关闭", nil)
}

// consumeTopic 消费指定主题的消息
// 运行在独立的 goroutine 中，从队列读取消息并分发给所有订阅者
func (b *Bus) consumeTopic(topic string) {
	b.mu.RLock()
	queue := b.topics[topic]
	subscribers := b.subscribers[topic]
	b.mu.RUnlock()

	// 如果没有订阅者，不启动消费协程
	if len(subscribers) == 0 {
		return
	}

	for {
		select {
		case msg := <-queue:
			// 分发给所有订阅者
			for _, sub := range b.getSubscribersSnapshot(topic) {
				select {
				case sub.queue <- msg:
				case <-sub.done:
					// 订阅者已关闭，跳过
				default:
					// 队列满，记录警告
					logger.Warn("订阅者队列已满，消息可能被丢弃", map[string]interface{}{
						"subscriberID": sub.ID,
						"topic":        topic,
					})
				}
			}

		case <-b.done:
			// Bus 关闭，退出
			return
		}
	}
}

// getSubscribersSnapshot 获取主题订阅者的快照
// 用于在遍历时不影响其他操作
func (b *Bus) getSubscribersSnapshot(topic string) []*Subscriber {
	b.mu.RLock()
	defer b.mu.RUnlock()
	subs := b.subscribers[topic]
	result := make([]*Subscriber, len(subs))
	copy(result, subs)
	return result
}

// consume 订阅者消费消息的协程
// 从自己的队列中读取消息并调用处理函数
func (s *Subscriber) consume() {
	for {
		select {
		case msg := <-s.queue:
			// 调用处理函数，设置超时
			s.processWithTimeout(msg)

		case <-s.done:
			// 收到退出信号
			return
		}
	}
}

// processWithTimeout 带超时的消息处理
func (s *Subscriber) processWithTimeout(msg *Message) {
	done := make(chan error, 1)

	go func() {
		done <- s.Handler(msg)
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error("处理消息失败", map[string]interface{}{
				"subscriberID": s.ID,
				"messageID":    msg.ID,
				"error":        err.Error(),
			})
		}
	case <-time.After(s.bus.handlerTimeout):
		logger.Warn("处理消息超时", map[string]interface{}{
			"subscriberID": s.ID,
			"messageID":    msg.ID,
		})
	case <-s.done:
		return
	}
}

// generateMessageID 生成唯一的消息ID
func generateMessageID() string {
	return fmt.Sprintf("msg-%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%10000)
}

// generateSubscriberID 生成唯一的订阅者ID
func generateSubscriberID() string {
	return fmt.Sprintf("sub-%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%10000)
}
