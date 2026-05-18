package context

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/longstageai/donk/donk/configs"
	"github.com/longstageai/donk/donk/pkg/config"
	"github.com/longstageai/donk/donk/pkg/graceful"
	"github.com/longstageai/donk/donk/pkg/ioc"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// beanOption Bean注册选项
// 用于存储待注册到IOC容器的Bean信息
type beanOption struct {
	name  string      // Bean名称，空字符串表示按类型自动命名
	bean  interface{} // Bean实例
	isNil bool        // 是否为nil Bean，用于占位
}

// Application 应用程序上下文
// 整合配置加载、日志初始化、IOC容器管理、优雅退出等功能
type Application struct {
	mu          sync.RWMutex           // 读写锁，保证并发安全
	config      *config.Config         // 配置管理器
	logger      *logger.Logger         // 日志记录器
	container   ioc.ApplicationContext // IOC容器
	runner      *graceful.Runner       // 任务运行器，管理后台任务和优雅退出
	appName     string                 // 应用名称
	version     string                 // 应用版本
	env         string                 // 应用环境 (development/test/production)
	configPaths []string               // 配置文件路径列表
	conf        configs.Conf           // 配置绑定对象
	beans       []beanOption           // 待注册的Bean列表
	startTime   time.Time              // 启动时间
	initialized bool                   // 是否已初始化
	stopped     bool                   // 是否已停止
	stopCh      chan struct{}          // 停止通道
	wg          sync.WaitGroup         // 等待组，用于等待goroutine完成
}

// Config 获取配置对象
// 返回配置管理器，可用于获取配置项
func (a *Application) Config() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// ConfigBean 获取配置绑定对象
// 返回通过 WithConfigBean 设置的配置对象
func (a *Application) ConfigBean() *configs.Conf {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return &a.conf
}

// Logger 获取日志对象
// 返回日志记录器，可用于记录日志
func (a *Application) Logger() *logger.Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.logger
}

// Container 获取IOC容器
// 返回IOC容器，可用于Bean管理
func (a *Application) Container() ioc.ApplicationContext {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.container
}

// Runner 获取优雅退出运行器
// 返回任务运行器，用于管理后台任务
func (a *Application) Runner() *graceful.Runner {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.runner
}

// AppName 获取应用名称
func (a *Application) AppName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.appName
}

// Version 获取应用版本
func (a *Application) Version() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.version
}

// Env 获取当前环境
// 返回 development, production 等环境名称
func (a *Application) Env() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.env
}

// UpTime 获取应用运行时长
// 返回自应用启动以来经过的时间
func (a *Application) UpTime() time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return time.Since(a.startTime)
}

// IsInitialized 检查应用是否已初始化
func (a *Application) IsInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

// Debug 输出 Debug 级别日志
func (a *Application) Debug(message string, fields map[string]interface{}) {
	if a.logger != nil {
		a.logger.Debug(message, fields)
	}
}

// Info 输出 Info 级别日志
func (a *Application) Info(message string, fields map[string]interface{}) {
	if a.logger != nil {
		a.logger.Info(message, fields)
	}
}

// Warn 输出 Warn 级别日志
func (a *Application) Warn(message string, fields map[string]interface{}) {
	if a.logger != nil {
		a.logger.Warn(message, fields)
	}
}

// Error 输出 Error 级别日志
func (a *Application) Error(message string, fields map[string]interface{}) {
	if a.logger != nil {
		a.logger.Error(message, fields)
	}
}

// Fatal 输出 Fatal 级别日志并退出
func (a *Application) Fatal(message string, fields map[string]interface{}) {
	if a.logger != nil {
		a.logger.Fatal(message, fields)
	}
}

// Debugf 格式化输出 Debug 级别日志
func (a *Application) Debugf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Debug(fmt.Sprintf(format, args...), nil)
	}
}

// Infof 格式化输出 Info 级别日志
func (a *Application) Infof(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Info(fmt.Sprintf(format, args...), nil)
	}
}

// Warnf 格式化输出 Warn 级别日志
func (a *Application) Warnf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Warn(fmt.Sprintf(format, args...), nil)
	}
}

// Errorf 格式化输出 Error 级别日志
func (a *Application) Errorf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Error(fmt.Sprintf(format, args...), nil)
	}
}

// Fatalf 格式化输出 Fatal 级别日志并退出
func (a *Application) Fatalf(format string, args ...interface{}) {
	if a.logger != nil {
		a.logger.Fatal(fmt.Sprintf(format, args...), nil)
	}
}

// IsStopped 检查应用是否已停止
func (a *Application) IsStopped() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.stopped
}

// Done 返回应用停止信号通道
// 当应用收到退出信号时，该通道会被关闭
func (a *Application) Done() <-chan struct{} {
	return a.stopCh
}

// Context 获取Go标准库的Context
// 用于在任务中传递取消信号
func (a *Application) Context() context.Context {
	if a.runner != nil {
		return a.runner.Context()
	}
	return context.Background()
}

// Initialize 初始化应用
// 依次执行：配置加载 -> 日志初始化 -> IOC容器启动 -> 任务运行器初始化
func (a *Application) Initialize() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return fmt.Errorf("应用已初始化，不能重复初始化")
	}

	if a.logger != nil {
		a.logger.Info("正在初始化应用: "+a.appName, nil)
	}

	if a.config == nil && len(a.configPaths) > 0 {
		a.config = config.New(config.WithEnvironment(a.env))
	}

	if a.config != nil {
		if err := a.initConfig(); err != nil {
			return fmt.Errorf("配置初始化失败: %w", err)
		}
	}

	if a.logger != nil {
		if err := a.initLogger(); err != nil {
			return fmt.Errorf("日志初始化失败: %w", err)
		}
	}

	if err := a.initContainer(); err != nil {
		return fmt.Errorf("IOC容器初始化失败: %w", err)
	}

	a.initRunner()

	a.startTime = time.Now()
	a.initialized = true
	a.stopCh = make(chan struct{})

	if a.logger != nil {
		a.logger.Info(fmt.Sprintf("应用初始化完成，耗时: %v", time.Since(a.startTime)), nil)
	}

	return nil
}

// initConfig 初始化配置
// 加载配置文件、解析变量、绑定到配置Bean
func (a *Application) initConfig() error {
	if a.config == nil {
		return nil
	}

	if a.logger != nil {
		a.logger.Debug("正在加载配置文件...", nil)
	}

	for _, path := range a.configPaths {
		if err := a.config.LoadFile(path); err != nil {
			return fmt.Errorf("加载配置文件失败 [%s]: %w", path, err)
		}
	}

	if err := a.config.Resolve(); err != nil {
		return fmt.Errorf("解析配置变量失败: %w", err)
	}

	if err := a.config.Unmarshal(&a.conf); err != nil {
		return fmt.Errorf("绑定配置到对象失败: %w", err)
	}
	if a.logger != nil {
		a.logger.Info(fmt.Sprintf("配置已绑定到: %T", a.conf), nil)
	}

	if a.logger != nil {
		a.logger.Info("配置文件加载完成", nil)
	}

	return nil
}

// initLogger 初始化日志
func (a *Application) initLogger() error {
	if a.logger == nil {
		return nil
	}
	logger.SetDefault(a.logger)
	if a.logger != nil {
		a.logger.Debug("正在初始化日志...", nil)
	}

	a.logger.Info(fmt.Sprintf("日志系统已启动 [应用: %s, 环境: %s]", a.appName, a.env), nil)

	return nil
}

// initContainer 初始化IOC容器
// 注册Bean并启动容器
func (a *Application) initContainer() error {
	if a.container == nil {
		a.container = ioc.NewContainer()
	}

	if a.logger != nil {
		a.logger.Debug("正在启动IOC容器...", nil)
	}

	for _, bo := range a.beans {
		if bo.isNil {
			continue
		}
		if bo.name != "" {
			if err := a.container.RegisterBeanWithName(bo.name, bo.bean); err != nil {
				return fmt.Errorf("注册Bean失败 [%s]: %w", bo.name, err)
			}
		} else {
			if err := a.container.RegisterBean(bo.bean); err != nil {
				return fmt.Errorf("注册Bean失败: %w", err)
			}
		}
		if a.logger != nil {
			beanName := bo.name
			if beanName == "" {
				beanName = reflect.TypeOf(bo.bean).String()
			}
			a.logger.Debug(fmt.Sprintf("已注册Bean: %s", beanName), nil)
		}
	}

	if err := a.container.Start(); err != nil {
		return err
	}

	if a.logger != nil {
		a.logger.Info("IOC容器启动完成", nil)
	}

	return nil
}

// initRunner 初始化任务运行器
func (a *Application) initRunner() {
	if a.runner == nil {
		a.runner = graceful.New(graceful.WithApp(a))
	}
	if a.logger != nil {
		a.logger.Debug("任务运行器已就绪", nil)
	}
}

// Run 启动应用
// 启动所有注册的后台任务
// 阻塞直到收到退出信号，然后自动调用 Stop() 优雅退出
func (a *Application) Run() error {
	if !a.initialized {
		return fmt.Errorf("应用未初始化，请先调用 Initialize()")
	}

	if a.logger != nil {
		a.logger.Info("正在启动应用: "+a.appName, nil)
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.runner.Run()
		a.Stop()
	}()

	if a.logger != nil {
		a.logger.Info("应用启动成功，等待退出信号 (Ctrl+C)...", nil)
	}

	a.wg.Wait()

	return nil
}

// Stop 停止应用
// 优雅停止所有组件：关闭停止通道、等待任务完成、停止IOC容器、关闭日志
func (a *Application) Stop() error {
	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return nil
	}
	a.stopped = true
	a.mu.Unlock()

	if a.logger != nil {
		a.logger.Info("正在停止应用: "+a.appName, nil)
	}

	close(a.stopCh)

	if a.container != nil {
		if err := a.container.Stop(); err != nil {
			if a.logger != nil {
				a.logger.Error(fmt.Sprintf("停止IOC容器失败: %v", err), nil)
			}
		} else {
			if a.logger != nil {
				a.logger.Info("IOC容器已停止", nil)
			}
		}
	}

	if a.logger != nil {
		a.logger.Info(fmt.Sprintf("应用已停止，运行时长: %v", a.UpTime()), nil)
		a.logger.Close()
	}

	return nil
}

// Wait 等待所有任务完成
// 阻塞直到所有后台任务完成或收到退出信号
func (a *Application) Wait() {
	a.runner.Wait()
}

// RegisterTask 注册后台任务
// name: 任务名称
// handler: 任务处理函数，需监听 context.Done() 以支持优雅退出
// timeout: 任务超时时间，0表示无超时限制
func (a *Application) RegisterTask(name string, handler graceful.Task, timeout time.Duration) *Application {
	if a.runner == nil {
		a.runner = graceful.New(graceful.WithApp(a))
	}
	a.runner.Register(name, handler, timeout)
	return a
}

// RegisterTaskFunc 注册后台任务（便捷方法）
// handler: 任务处理函数，第一个参数为 context.Context，第二个参数为 *Application
// 示例:
//
//	app.RegisterTaskFunc("http-server", func(ctx context.Context, app *Application) error {
//	    // 使用 app 获取配置、日志等
//	    cfg := app.ConfigBean().(*AppConfig)
//	    app.Info("服务启动", nil)
//	    <-ctx.Done()
//	    return nil
//	}, 0)
func (a *Application) RegisterTaskFunc(name string, handler func(context.Context, *Application) error, timeout time.Duration) *Application {
	wrappedHandler := func(ctx context.Context, _ graceful.AppContext) error {
		return handler(ctx, a)
	}
	return a.RegisterTask(name, wrappedHandler, timeout)
}

// RegisterBean 注册Bean到IOC容器
// bean: 要注册的Bean实例，支持自动注入
func (a *Application) RegisterBean(bean interface{}) *Application {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.beans = append(a.beans, beanOption{bean: bean, isNil: bean == nil})
	return a
}

// RegisterBeanWithName 注册命名Bean到IOC容器
// name: Bean名称
// bean: 要注册的Bean实例
func (a *Application) RegisterBeanWithName(name string, bean interface{}) *Application {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.beans = append(a.beans, beanOption{name: name, bean: bean, isNil: bean == nil})
	return a
}

// GetBean 根据名称获取Bean
// name: Bean的名称
// 返回: Bean实例和错误
func (a *Application) GetBean(name string) (interface{}, error) {
	return a.container.GetBean(name)
}

// GetBeanByType 根据类型获取Bean
// typ: Bean的类型指针，如 &UserService{}
// 返回: Bean实例和错误
func (a *Application) GetBeanByType(typ interface{}) (interface{}, error) {
	return a.container.GetBeanByType(typ)
}
