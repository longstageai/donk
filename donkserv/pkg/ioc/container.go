package ioc

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

// 容器错误类型定义
var (
	ErrBeanNotFound       = errors.New("bean not found")               // Bean 不存在错误
	ErrBeanAlreadyExists  = errors.New("bean already exists")          // Bean 已存在错误
	ErrCircularDependency = errors.New("circular dependency detected") // 循环依赖错误
	ErrInvalidBeanType    = errors.New("invalid bean type")            // 无效的 Bean 类型错误
	ErrInjectFailed       = errors.New("injection failed")             // 注入失败错误
)

// BeanScope 定义 Bean 的作用域类型
type BeanScope string

const (
	ScopeSingleton BeanScope = "singleton" // 单例作用域：整个容器只有一个实例
	ScopePrototype BeanScope = "prototype" // 原型作用域：每次获取都创建新实例
)

// BeanDefinition 定义 Bean 的元数据信息
type BeanDefinition struct {
	Name          string                    // Bean 的唯一标识名称
	Type          reflect.Type              // Bean 的类型信息
	Scope         BeanScope                 // Bean 的作用域（单例或原型）
	Factory       BeanFactory               // Bean 的工厂函数，用于创建 Bean 实例
	InitMethod    string                    // 初始化方法名称
	DestroyMethod string                    // 销毁方法名称
	Properties    map[string]*PropertyValue // 属性配置，用于属性注入
	DependsOn     []string                  // 依赖的其他 Bean 名称列表
	Primary       bool                      // 是否为主要 Bean（当有多个同类型 Bean 时）
}

// PropertyValue 定义属性注入的值
type PropertyValue struct {
	Value interface{} // 属性值
	Ref   string      // 引用的 Bean 名称
	IsRef bool        // 是否为引用类型
}

// BeanFactory 定义 Bean 的工厂函数类型
type BeanFactory func() (interface{}, error)

// BeanWrapper 包装 Bean 实例及其元数据
type BeanWrapper struct {
	Object     interface{}     // Bean 的实际对象实例
	Definition *BeanDefinition // Bean 的定义信息
	InitErr    error           // 初始化错误信息
	createdAt  time.Time       // Bean 创建时间
	mu         sync.RWMutex    // Bean 对象的读写锁
}

// GetObject 获取 Bean 对象（线程安全）
func (bw *BeanWrapper) GetObject() interface{} {
	bw.mu.RLock()
	defer bw.mu.RUnlock()
	return bw.Object
}

// SetObject 设置 Bean 对象（线程安全）
func (bw *BeanWrapper) SetObject(obj interface{}) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.Object = obj
}

// ApplicationContext 定义 IOC 容器的核心接口
type ApplicationContext interface {
	GetBean(name string) (interface{}, error)                       // 根据名称获取 Bean
	GetBeanByType(typ interface{}) (interface{}, error)             // 根据类型获取 Bean
	GetBeansOfType(typ interface{}) (map[string]interface{}, error) // 获取指定类型的所有 Bean
	RegisterBean(bean interface{}) error                            // 注册 Bean 实例
	RegisterBeanWithName(name string, bean interface{}) error       // 带名称注册 Bean 实例
	RegisterBeanDefinition(def *BeanDefinition) error               // 注册 Bean 定义
	InjectDependencies() error                                      // 注入所有依赖
	Start() error                                                   // 启动容器
	Stop() error                                                    // 停止容器
}

// Container 是 IOC 容器的核心实现
type Container struct {
	beanDefinitions map[string]*BeanDefinition // Bean 定义注册表
	beanWrappers    map[string]*BeanWrapper    // Bean 包装器注册表
	singletons      map[string]*BeanWrapper    // 单例 Bean 缓存
	prototypeCounts map[string]int             // 原型 Bean 创建计数

	mu           sync.RWMutex    // 容器级别的读写锁
	cyclingDeps  map[string]bool // 循环依赖检测标记
	cyclingStack []string        // 循环依赖路径栈

	logger    *log.Logger // 日志记录器
	enableLog bool        // 是否启用日志

	initializing bool // 容器是否正在初始化
	running      bool // 容器是否正在运行
}

// ContainerOption 定义容器配置选项
type ContainerOption func(*Container)

// WithLog 启用容器日志输出
func WithLog() ContainerOption {
	return func(c *Container) {
		c.enableLog = true
	}
}

// NewContainer 创建新的 IOC 容器实例
func NewContainer(opts ...ContainerOption) *Container {
	c := &Container{
		beanDefinitions: make(map[string]*BeanDefinition),
		beanWrappers:    make(map[string]*BeanWrapper),
		singletons:      make(map[string]*BeanWrapper),
		prototypeCounts: make(map[string]int),
		cyclingDeps:     make(map[string]bool),
		cyclingStack:    make([]string, 0),
		logger:          log.Default(),
		enableLog:       false,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// log 记录容器日志
func (c *Container) log(format string, v ...interface{}) {
	if c.enableLog {
		c.logger.Output(2, fmt.Sprintf(format, v...))
	}
}

// RegisterBean 注册 Bean 实例（使用类型名作为 Bean 名称）
func (c *Container) RegisterBean(bean interface{}) error {
	return c.RegisterBeanWithName("", bean)
}

// RegisterBeanWithName 使用指定名称注册 Bean 实例
func (c *Container) RegisterBeanWithName(name string, bean interface{}) error {
	if bean == nil {
		return ErrInvalidBeanType
	}

	// 获取 Bean 类型信息
	beanType := reflect.TypeOf(bean)
	if beanType.Kind() == reflect.Ptr {
		beanType = beanType.Elem()
	}

	// 如果未指定名称，使用类型名
	if name == "" {
		name = beanType.Name()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查 Bean 是否已存在
	if _, exists := c.beanDefinitions[name]; exists {
		return fmt.Errorf("%w: %s", ErrBeanAlreadyExists, name)
	}

	// 创建工厂函数
	factory := func() (interface{}, error) {
		return bean, nil
	}

	// 创建 Bean 定义
	def := &BeanDefinition{
		Name:    name,
		Type:    reflect.TypeOf(bean),
		Scope:   ScopeSingleton,
		Factory: factory,
		Primary: true,
	}

	// 注册 Bean 定义和包装器
	c.beanDefinitions[name] = def
	c.beanWrappers[name] = &BeanWrapper{
		Object:     bean,
		Definition: def,
		createdAt:  time.Now(),
	}

	// 获取类型名称用于日志输出
	typeName := def.Type.Name()
	if def.Type.Kind() == reflect.Ptr {
		typeName = def.Type.Elem().Name()
	}
	c.log("Registered bean: %s (type: %s, scope: %s)", name, typeName, def.Scope)
	return nil
}

// RegisterBeanDefinition 注册 Bean 定义
func (c *Container) RegisterBeanDefinition(def *BeanDefinition) error {
	if def == nil {
		return ErrInvalidBeanType
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查 Bean 是否已存在
	if _, exists := c.beanDefinitions[def.Name]; exists {
		return fmt.Errorf("%w: %s", ErrBeanAlreadyExists, def.Name)
	}

	// 设置默认作用域为单例
	if def.Scope == "" {
		def.Scope = ScopeSingleton
	}

	// 注册 Bean 定义
	c.beanDefinitions[def.Name] = def

	// 获取类型名称用于日志输出
	typeName := def.Type.Name()
	if def.Type.Kind() == reflect.Ptr {
		typeName = def.Type.Elem().Name()
	}
	c.log("Registered bean definition: %s (type: %s, scope: %s)", def.Name, typeName, def.Scope)
	return nil
}

// GetBean 根据名称获取 Bean 实例
func (c *Container) GetBean(name string) (interface{}, error) {
	c.mu.RLock()
	def, exists := c.beanDefinitions[name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrBeanNotFound, name)
	}

	return c.getOrCreateBean(name, def)
}

// getBeanUnsafe 内部方法：不获取锁的情况下获取 Bean（用于避免死锁）
func (c *Container) getBeanUnsafe(name string) (interface{}, error) {
	def, exists := c.beanDefinitions[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrBeanNotFound, name)
	}

	return c.getOrCreateBean(name, def)
}

// GetBeanByType 根据类型获取 Bean 实例
func (c *Container) GetBeanByType(typ interface{}) (interface{}, error) {
	targetType := reflect.TypeOf(typ)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 遍历所有 Bean 定义，查找匹配的类型
	for name, def := range c.beanDefinitions {
		if def.Type == targetType || (def.Type.Kind() == reflect.Ptr && def.Type.Elem() == targetType) {
			wrapper, exists := c.getBeanWrapperUnsafe(name, def.Scope)
			if exists && wrapper != nil {
				return wrapper.GetObject(), nil
			}
		}

		// 检查类型是否可赋值（接口实现）
		if isAssignable(targetType, def.Type) {
			wrapper, exists := c.getBeanWrapperUnsafe(name, def.Scope)
			if exists && wrapper != nil {
				return wrapper.GetObject(), nil
			}
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrBeanNotFound, targetType.Name())
}

// GetBeansOfType 获取指定类型的所有 Bean
func (c *Container) GetBeansOfType(typ interface{}) (map[string]interface{}, error) {
	targetType := reflect.TypeOf(typ)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	result := make(map[string]interface{})

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 遍历所有 Bean 定义，收集匹配的类型
	for name, def := range c.beanDefinitions {
		if isAssignable(targetType, def.Type) || def.Type == targetType {
			wrapper, exists := c.getBeanWrapperUnsafe(name, def.Scope)
			if exists && wrapper != nil {
				result[name] = wrapper.GetObject()
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrBeanNotFound, targetType.Name())
	}

	return result, nil
}

// GetBeanByName 泛型方法：根据名称获取 Bean
func GetBeanByName[T any](c *Container, name string) (T, error) {
	var zero T
	bean, err := c.GetBean(name)
	if err != nil {
		return zero, err
	}
	return convertToType[T](bean)
}

// GetBeanByTypeGeneric 泛型方法：根据类型获取 Bean
func GetBeanByTypeGeneric[T any](c *Container) (T, error) {
	var zero T
	bean, err := c.GetBeanByType(&zero)
	if err != nil {
		return zero, err
	}
	return convertToType[T](bean)
}

func convertToType[T any](bean interface{}) (T, error) {
	var zero T
	if bean == nil {
		return zero, nil
	}

	beanValue := reflect.ValueOf(bean)
	beanType := beanValue.Type()
	targetType := reflect.TypeOf(zero)

	if targetType == nil {
		return zero, nil
	}

	if beanType.AssignableTo(targetType) {
		return bean.(T), nil
	}

	if beanType.Kind() == reflect.Ptr && beanType.Elem().AssignableTo(targetType) {
		if beanValue.IsNil() {
			return zero, nil
		}
		return beanValue.Elem().Interface().(T), nil
	}

	val, ok := bean.(T)
	if !ok {
		return zero, fmt.Errorf("%w: bean type %v cannot be assigned to %T", ErrInvalidBeanType, beanType, zero)
	}
	return val, nil
}

// getBeanWrapperUnsafe 内部方法：不获取锁的情况下获取 Bean 包装器
func (c *Container) getBeanWrapperUnsafe(name string, scope BeanScope) (*BeanWrapper, bool) {
	switch scope {
	case ScopeSingleton:
		wrapper, exists := c.singletons[name]
		return wrapper, exists
	case ScopePrototype:
		wrapper, exists := c.beanWrappers[name]
		return wrapper, exists
	default:
		wrapper, exists := c.singletons[name]
		return wrapper, exists
	}
}

// getOrCreateBean 根据作用域获取或创建 Bean
func (c *Container) getOrCreateBean(name string, def *BeanDefinition) (interface{}, error) {
	switch def.Scope {
	case ScopeSingleton:
		return c.getSingleton(name, def)
	case ScopePrototype:
		return c.createPrototype(name, def)
	default:
		return c.getSingleton(name, def)
	}
}

// getSingleton 获取或创建单例 Bean
func (c *Container) getSingleton(name string, def *BeanDefinition) (interface{}, error) {
	// 先尝试读锁获取
	c.mu.RLock()
	if wrapper, exists := c.singletons[name]; exists {
		c.mu.RUnlock()
		return wrapper.GetObject(), nil
	}
	c.mu.RUnlock()

	// 获取写锁创建 Bean
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	if wrapper, exists := c.singletons[name]; exists {
		return wrapper.GetObject(), nil
	}

	// 创建 Bean
	wrapper, err := c.createBeanUnsafe(name, def)
	if err != nil {
		return nil, err
	}

	// 缓存单例
	c.singletons[name] = wrapper
	return wrapper.GetObject(), nil
}

// createPrototype 创建原型 Bean（每次都创建新实例）
func (c *Container) createPrototype(name string, def *BeanDefinition) (interface{}, error) {
	c.prototypeCounts[name]++
	return c.createBeanUnsafe(name, def)
}

// createBean 创建 Bean 并执行初始化和属性注入
func (c *Container) createBean(name string, def *BeanDefinition) (*BeanWrapper, error) {
	wrapper, err := c.createBeanUnsafe(name, def)
	if err != nil {
		return nil, err
	}

	// 注入属性
	if err := c.injectProperties(wrapper, def); err != nil {
		wrapper.InitErr = err
		c.log("Failed to inject properties for bean %s: %v", name, err)
		return nil, err
	}

	// 调用初始化方法
	if err := c.callInitMethod(wrapper, def); err != nil {
		c.log("Failed to call init method for bean %s: %v", name, err)
		return nil, err
	}

	c.log("Created bean: %s", name)
	return wrapper, nil
}

// createBeanUnsafe 创建 Bean 实例（不执行注入和初始化）
func (c *Container) createBeanUnsafe(name string, def *BeanDefinition) (*BeanWrapper, error) {
	// 检测循环依赖
	if c.cyclingDeps[name] {
		c.cyclingStack = append(c.cyclingStack, name)
		return nil, fmt.Errorf("%w: %v", ErrCircularDependency, c.cyclingStack)
	}

	// 标记为正在创建
	c.cyclingDeps[name] = true
	c.cyclingStack = append(c.cyclingStack, name)
	defer func() {
		c.cyclingDeps[name] = false
		c.cyclingStack = c.cyclingStack[:len(c.cyclingStack)-1]
	}()

	// 调用工厂函数创建 Bean
	obj, err := def.Factory()
	if err != nil {
		return nil, fmt.Errorf("failed to create bean %s: %w", name, err)
	}

	// 创建 Bean 包装器
	wrapper := &BeanWrapper{
		Object:     obj,
		Definition: def,
		createdAt:  time.Now(),
	}

	c.log("Created bean: %s", name)
	return wrapper, nil
}

// isAssignable 检查源类型是否可以赋值给目标类型
func isAssignable(target, source reflect.Type) bool {
	if target == source {
		return true
	}

	// 处理指针类型
	if source.Kind() == reflect.Ptr {
		source = source.Elem()
	}
	if target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	// 检查是否实现接口
	// source 必须是接口类型，target 必须是 reflect.Type
	if source.Kind() == reflect.Interface && target.Kind() == reflect.Interface {
		return source.Implements(target)
	}

	return false
}

// InjectDependencies 注入所有 Bean 的依赖关系
func (c *Container) InjectDependencies() error {
	c.mu.RLock()
	defs := make([]*BeanDefinition, 0, len(c.beanDefinitions))
	for _, def := range c.beanDefinitions {
		defs = append(defs, def)
	}
	c.mu.RUnlock()

	// 初始化所有 Bean
	for _, def := range defs {
		if err := c.initializeBean(def); err != nil {
			return err
		}
	}

	return nil
}

// initializeBean 初始化单个 Bean
func (c *Container) initializeBean(def *BeanDefinition) error {
	_, err := c.getOrCreateBean(def.Name, def)
	if err != nil {
		return err
	}

	wrapper, err := c.getOrCreateWrapper(def.Name, def)
	if err != nil {
		return err
	}

	if err := c.injectProperties(wrapper, def); err != nil {
		return err
	}

	return nil
}

// getOrCreateWrapper 获取或创建 Bean 包装器
func (c *Container) getOrCreateWrapper(name string, def *BeanDefinition) (*BeanWrapper, error) {
	if def.Scope == ScopeSingleton {
		if wrapper, exists := c.singletons[name]; exists {
			return wrapper, nil
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if wrapper, exists := c.singletons[name]; exists {
		return wrapper, nil
	}

	return c.createBean(name, def)
}

// injectProperties 执行属性注入
func (c *Container) injectProperties(wrapper *BeanWrapper, def *BeanDefinition) error {
	// 如果没有显式属性配置，执行自动装配
	if def.Properties == nil || len(def.Properties) == 0 {
		return c.autowireFields(wrapper)
	}

	obj := wrapper.GetObject()
	objVal := reflect.ValueOf(obj)

	// 解引用指针类型
	if objVal.Kind() == reflect.Ptr {
		objVal = objVal.Elem()
	}

	if objVal.Kind() != reflect.Struct {
		return nil
	}

	objType := objVal.Type()

	// 注入显式配置的属性
	for fieldName, prop := range def.Properties {
		_, found := objType.FieldByName(fieldName)
		if !found {
			c.log("Field %s not found in bean %s", fieldName, def.Name)
			continue
		}

		fieldVal := objVal.FieldByName(fieldName)
		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			c.log("Field %s cannot be set in bean %s", fieldName, def.Name)
			continue
		}

		var value reflect.Value
		// 引用注入
		if prop.IsRef && prop.Ref != "" {
			refBean, err := c.getBeanUnsafe(prop.Ref)
			if err != nil {
				return fmt.Errorf("failed to inject field %s: %w", fieldName, err)
			}
			value = reflect.ValueOf(refBean)
		} else {
			// 值注入
			value = reflect.ValueOf(prop.Value)
		}

		if !value.IsValid() {
			continue
		}

		fieldType := fieldVal.Type()
		valueType := value.Type()

		// 类型匹配并设置字段值
		if valueType.AssignableTo(fieldType) {
			fieldVal.Set(value)
		} else if valueType.Kind() == reflect.Ptr && valueType.Elem().AssignableTo(fieldType) {
			fieldVal.Set(value)
		} else if valueType.Kind() == reflect.Ptr && valueType.Elem().Implements(fieldType) {
			fieldVal.Set(value)
		}
	}

	// 执行自动装配
	return c.autowireFields(wrapper)
}

// autowireFields 基于结构体标签自动装配字段
func (c *Container) autowireFields(wrapper *BeanWrapper) error {
	obj := wrapper.GetObject()
	objVal := reflect.ValueOf(obj)

	// 解引用指针类型
	if objVal.Kind() == reflect.Ptr {
		objVal = objVal.Elem()
	}

	if objVal.Kind() != reflect.Struct {
		return nil
	}

	objType := objVal.Type()
	numFields := objType.NumField()

	// 遍历所有字段，查找 ioc 标签
	for i := 0; i < numFields; i++ {
		field := objType.Field(i)
		tag := field.Tag.Get("ioc")

		// 跳过没有 ioc 标签或标记为 "-" 的字段
		if tag == "" || tag == "-" {
			continue
		}

		// 确定要注入的 Bean 名称
		beanName := tag
		if beanName == "" {
			beanName = field.Type.Name()
		}

		// 获取依赖的 Bean
		refBean, err := c.getBeanUnsafe(beanName)
		if err != nil {
			c.log("Failed to autowire field %s (bean %s): %v", field.Name, beanName, err)
			continue
		}

		c.log("Autowiring field %s with bean %s", field.Name, beanName)

		// 设置字段值
		fieldVal := objVal.FieldByName(field.Name)
		if fieldVal.IsValid() && fieldVal.CanSet() {
			fieldVal.Set(reflect.ValueOf(refBean))
		}
	}

	return nil
}

// callInitMethod 调用 Bean 的初始化方法
func (c *Container) callInitMethod(wrapper *BeanWrapper, def *BeanDefinition) error {
	if def.InitMethod == "" {
		return nil
	}

	obj := wrapper.GetObject()
	objVal := reflect.ValueOf(obj)

	// 解引用指针类型
	if objVal.Kind() == reflect.Ptr {
		objVal = objVal.Elem()
	}

	// 获取初始化方法
	initMethod := objVal.MethodByName(def.InitMethod)
	if !initMethod.IsValid() {
		return nil
	}

	// 调用方法
	results := initMethod.Call(nil)
	if len(results) > 0 {
		if err, ok := results[0].Interface().(error); ok && err != nil {
			return fmt.Errorf("init method failed: %w", err)
		}
	}

	c.log("Called init method %s for bean %s", def.InitMethod, def.Name)
	return nil
}

// callDestroyMethod 调用 Bean 的销毁方法
func (c *Container) callDestroyMethod(wrapper *BeanWrapper, def *BeanDefinition) error {
	if def.DestroyMethod == "" {
		return nil
	}

	obj := wrapper.GetObject()
	if obj == nil {
		return nil
	}

	objVal := reflect.ValueOf(obj)

	// 解引用指针类型
	if objVal.Kind() == reflect.Ptr {
		objVal = objVal.Elem()
	}

	// 获取销毁方法
	destroyMethod := objVal.MethodByName(def.DestroyMethod)
	if !destroyMethod.IsValid() {
		return nil
	}

	// 调用方法
	results := destroyMethod.Call(nil)
	if len(results) > 0 {
		if err, ok := results[0].Interface().(error); ok && err != nil {
			return fmt.Errorf("destroy method failed: %w", err)
		}
	}

	c.log("Called destroy method %s for bean %s", def.DestroyMethod, def.Name)
	return nil
}

// Start 启动 IOC 容器
func (c *Container) Start() error {
	c.mu.Lock()
	if c.initializing || c.running {
		c.mu.Unlock()
		return errors.New("container is already running")
	}
	c.initializing = true
	c.mu.Unlock()

	c.log("Starting IOC container...")

	c.mu.RLock()
	defs := make([]*BeanDefinition, 0, len(c.beanDefinitions))
	for _, def := range c.beanDefinitions {
		defs = append(defs, def)
	}
	c.mu.RUnlock()

	// 初始化所有 Bean
	for _, def := range defs {
		if _, err := c.getOrCreateBean(def.Name, def); err != nil {
			c.mu.Lock()
			c.initializing = false
			c.mu.Unlock()
			return fmt.Errorf("failed to initialize bean %s: %w", def.Name, err)
		}

		wrapper, err := c.getOrCreateWrapper(def.Name, def)
		if err != nil {
			c.mu.Lock()
			c.initializing = false
			c.mu.Unlock()
			return err
		}

		if err := c.injectProperties(wrapper, def); err != nil {
			c.mu.Lock()
			c.initializing = false
			c.mu.Unlock()
			return fmt.Errorf("failed to inject properties for bean %s: %w", def.Name, err)
		}
	}

	c.mu.Lock()
	c.initializing = false
	c.running = true
	c.mu.Unlock()

	c.log("IOC container started successfully")
	return nil
}

// Stop 停止 IOC 容器
func (c *Container) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running && len(c.singletons) == 0 {
		return nil
	}

	c.log("Stopping IOC container...")

	// 销毁所有单例 Bean
	for name, wrapper := range c.singletons {
		if wrapper == nil || wrapper.Definition == nil {
			continue
		}

		if err := c.callDestroyMethod(wrapper, wrapper.Definition); err != nil {
			c.log("Error calling destroy method for bean %s: %v", name, err)
		}
	}

	// 清理单例缓存
	c.singletons = make(map[string]*BeanWrapper)
	c.running = false
	c.log("IOC container stopped")

	return nil
}

// IsRunning 检查容器是否正在运行
func (c *Container) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetBeanCount 获取已注册的 Bean 总数
func (c *Container) GetBeanCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.beanDefinitions)
}

// GetSingletonCount 获取单例 Bean 的数量
func (c *Container) GetSingletonCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.singletons)
}

// GetPrototypeCount 获取原型 Bean 的创建次数
func (c *Container) GetPrototypeCount(name string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.prototypeCounts[name]
}
