package context

import (
	"os"
	"time"

	"github.com/longstageai/donk/donk/configs"
	"github.com/longstageai/donk/donk/pkg/config"
	"github.com/longstageai/donk/donk/pkg/graceful"
	"github.com/longstageai/donk/donk/pkg/ioc"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// Option 应用配置选项函数类型
// 用于通过函数式选项模式配置 Application
type Option func(*Application)

// Builder 应用构建器
// 支持链式调用配置应用
type Builder struct {
	appName          string                        // 应用名称
	version          string                        // 应用版本
	env              string                        // 应用环境 (development/test/production)
	config           *config.Config                // 配置管理器
	configPaths      []string                      // 配置文件路径列表
	conf             configs.Conf                  // 配置绑定对象
	beans            []beanOption                  // 待注册的Bean列表
	logger           *logger.Logger                // 日志记录器
	loggerBuilder    *logger.LoggerBuilder         // 日志构建器
	container        ioc.ApplicationContext        // IOC容器
	containerFactory func() ioc.ApplicationContext // IOC容器工厂函数
	runner           *graceful.Runner              // 任务运行器
}

// NewBuilder 创建新的应用构建器
// 默认配置路径: ./conf/config.yaml
// 示例:
//
//	builder := context.NewBuilder()
//	app := builder.WithAppName("myapp").Build()
func NewBuilder() *Builder {
	return &Builder{
		appName:     "goboot",
		version:     "1.0.0",
		env:         "development",
		configPaths: []string{"./conf/config.yaml"},
	}
}

// WithAppName 设置应用名称
// 示例: builder.WithAppName("myapp")
func (b *Builder) WithAppName(name string) *Builder {
	b.appName = name
	return b
}

// WithVersion 设置应用版本
// 示例: builder.WithVersion("1.0.0")
func (b *Builder) WithVersion(version string) *Builder {
	b.version = version
	return b
}

// WithEnv 设置应用环境
// 示例: builder.WithEnv("production")
// 支持: development, test, production 等
func (b *Builder) WithEnv(env string) *Builder {
	b.env = env
	return b
}

// WithConfig 设置配置对象
// 使用自定义的 Config 实例
func (b *Builder) WithConfig(cfg *config.Config) *Builder {
	b.config = cfg
	return b
}

// WithConfigPath 设置配置文件路径
// 加载单个配置文件，默认: ./conf/config.yaml
// 示例: builder.WithConfigPath("./configs/app.yaml")
func (b *Builder) WithConfigPath(path string) *Builder {
	if b.config == nil {
		b.config = config.New(
			config.WithEnvironment(b.env),
		)
	}
	b.configPaths = []string{path}
	return b
}

// WithConfigPaths 设置多个配置文件路径
// 按顺序加载，多个文件会被合并
// 示例: builder.WithConfigPaths([]string{"./configs/base.yaml", "./configs/dev.yaml"})
func (b *Builder) WithConfigPaths(paths []string) *Builder {
	if b.config == nil {
		b.config = config.New(
			config.WithEnvironment(b.env),
		)
	}
	b.configPaths = paths
	return b
}

// WithBean 注册Bean到IOC容器
// 示例: builder.WithBean(&UserService{Name: "John"})
func (b *Builder) WithBean(bean interface{}) *Builder {
	b.beans = append(b.beans, beanOption{bean: bean, isNil: bean == nil})
	return b
}

// WithBeans 注册多个Bean到IOC容器
// 示例: builder.WithBeans(&UserService{}, &OrderService{})
func (b *Builder) WithBeans(beans ...interface{}) *Builder {
	for _, bean := range beans {
		b.beans = append(b.beans, beanOption{bean: bean, isNil: bean == nil})
	}
	return b
}

// WithNamedBean 注册命名Bean到IOC容器
// name: Bean的名称，用于唯一标识
// 示例: builder.WithNamedBean("userService", &UserService{Name: "John"})
func (b *Builder) WithNamedBean(name string, bean interface{}) *Builder {
	b.beans = append(b.beans, beanOption{name: name, bean: bean, isNil: bean == nil})
	return b
}

// WithLogger 设置日志对象
// 使用自定义的 Logger 实例
func (b *Builder) WithLogger(log *logger.Logger) *Builder {
	b.logger = log
	return b
}

// WithLoggerBuilder 设置日志构建器
// 使用构建器创建 Logger
// 示例:
//
//	builder.WithLoggerBuilder(
//	    logger.NewLogger().
//	        SetLevel(logger.INFO).
//	        AddConsoleWriter(true)
//	)
func (b *Builder) WithLoggerBuilder(builder *logger.LoggerBuilder) *Builder {
	b.loggerBuilder = builder
	return b
}

// WithLoggerLevel 设置日志级别
// 示例: builder.WithLoggerLevel(logger.INFO)
func (b *Builder) WithLoggerLevel(level logger.Level) *Builder {
	if b.loggerBuilder == nil {
		b.loggerBuilder = logger.NewLogger()
	}
	b.loggerBuilder.SetLevel(level)
	return b
}

// WithConsoleLogger 添加控制台日志输出
// 示例: builder.WithConsoleLogger(true)
func (b *Builder) WithConsoleLogger(enableColor bool) *Builder {
	if b.loggerBuilder == nil {
		b.loggerBuilder = logger.NewLogger()
	}
	b.loggerBuilder.AddConsoleWriter(enableColor)
	return b
}

// WithFileLogger 添加文件日志输出
// enable: 是否启用文件日志
// 默认配置: 目录 "./logs", 级别 INFO, 10MB, 保留7天, 最多3个备份
func (b *Builder) WithFileLogger(enable bool) *Builder {
	if !enable {
		return b
	}
	if b.loggerBuilder == nil {
		b.loggerBuilder = logger.NewLogger()
	}
	b.loggerBuilder.AddFileWriter("./logs", logger.INFO, 10*1024*1024, 7, 3)
	return b
}

// WithContainer 设置IOC容器
// 使用自定义的容器实例
func (b *Builder) WithContainer(container ioc.ApplicationContext) *Builder {
	b.container = container
	return b
}

// WithContainerFactory 设置IOC容器工厂函数
// 使用工厂函数创建容器
func (b *Builder) WithContainerFactory(factory func() ioc.ApplicationContext) *Builder {
	b.containerFactory = factory
	return b
}

// WithRunner 设置任务运行器
// 使用自定义的 Runner 实例
func (b *Builder) WithRunner(runner *graceful.Runner) *Builder {
	b.runner = runner
	return b
}

// WithShutdownSignals 设置退出信号
// 默认捕获 os.Interrupt 和 syscall.SIGTERM
// 示例: builder.WithShutdownSignals(os.Interrupt, syscall.SIGTERM)
func (b *Builder) WithShutdownSignals(signals ...os.Signal) *Builder {
	if b.runner == nil {
		b.runner = graceful.New(graceful.WithShutdownSignals(signals...))
	}
	return b
}

// Build 构建 Application 实例
func (b *Builder) Build() *Application {
	app := &Application{
		appName:     b.appName,
		version:     b.version,
		env:         b.env,
		config:      b.config,
		configPaths: b.configPaths,
		conf:        b.conf,
		beans:       b.beans,
		container:   b.container,
		runner:      b.runner,
		startTime:   time.Time{},
	}

	if b.loggerBuilder != nil {
		app.logger = b.loggerBuilder.Build()
	} else if b.logger != nil {
		app.logger = b.logger
	}

	if b.containerFactory != nil && app.container == nil {
		app.container = b.containerFactory()
	}

	return app
}

// BuildAndRun 构建并运行应用
// 等同于 Build().Initialize().Run()
func (b *Builder) BuildAndRun() error {
	app := b.Build()
	if err := app.Initialize(); err != nil {
		return err
	}
	return app.Run()
}

// New 创建新的 Application 实例
// 已废弃，请使用 NewBuilder() 代替
// 示例:
//
//	app := context.New(
//	    context.WithAppName("myapp"),
//	)
//
// 推荐用法:
//
//	app := context.NewBuilder().
//	    WithAppName("myapp").
//	    Build()
func New(opts ...Option) *Application {
	a := &Application{
		appName:     "goboot-app",
		version:     "1.0.0",
		env:         "development",
		configPaths: []string{"./conf/config.yaml"},
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}
