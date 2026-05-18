package config

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// VariableResolver 接口
// 用于自定义变量解析逻辑
type VariableResolver interface {
	Resolve(varName string, locals map[string]interface{}, globals map[string]interface{}) (string, error)
}

// DefaultVariableResolver 默认变量解析器实现
// 按顺序查找：局部变量 -> 全局变量 -> 默认值
type DefaultVariableResolver struct{}

// Resolve 解析变量值
func (r *DefaultVariableResolver) Resolve(varName string, locals map[string]interface{}, globals map[string]interface{}) (string, error) {
	name, defaultValue := parseVarNameWithDefault(varName)

	if val, ok := locals[name]; ok {
		return formatValue(val)
	}

	if val, ok := globals[name]; ok {
		return formatValue(val)
	}

	if defaultValue != "" {
		return defaultValue, nil
	}

	return "", fmt.Errorf("undefined variable: %s", name)
}

// parseVarNameWithDefault 解析变量名和默认值
// 支持语法: variable_name 或 variable_name:default_value
func parseVarNameWithDefault(varName string) (string, string) {
	parts := strings.SplitN(varName, ":", 2)
	if len(parts) > 1 {
		return parts[0], parts[1]
	}
	return varName, ""
}

// formatValue 格式化变量值为字符串
func formatValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", v), nil
	default:
		return "", fmt.Errorf("unsupported variable type: %T", value)
	}
}

// CycleDetector 循环引用检测器
// 用于检测配置中的循环变量引用
type CycleDetector struct {
	mu       sync.RWMutex
	tracking map[string]bool
}

// NewCycleDetector 创建新的循环检测器
func NewCycleDetector() *CycleDetector {
	return &CycleDetector{
		tracking: make(map[string]bool),
	}
}

// Enter 进入指定键的解析过程
// 如果该键正在被解析（即存在循环引用），返回错误
func (d *CycleDetector) Enter(key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.tracking[key] {
		return fmt.Errorf("circular variable reference detected: %s", key)
	}
	d.tracking[key] = true
	return nil
}

// Exit 退出指定键的解析过程
func (d *CycleDetector) Exit(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.tracking, key)
}

// Reset 重置检测器状态
func (d *CycleDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tracking = make(map[string]bool)
}

// AdvancedConfig 高级配置结构体
// 扩展了基础 Config 功能，支持自定义解析器和自定义函数
type AdvancedConfig struct {
	*Config
	resolver      VariableResolver                                // 变量解析器
	cycleDetector *CycleDetector                                  // 循环引用检测器
	customFuncs   map[string]func(args ...string) (string, error) // 自定义函数
}

// AdvancedOption 高级配置选项函数
type AdvancedOption func(*AdvancedConfig)

// WithResolver 设置自定义变量解析器
func WithResolver(resolver VariableResolver) AdvancedOption {
	return func(c *AdvancedConfig) {
		c.resolver = resolver
	}
}

// WithCustomFunction 注册自定义函数
// 可以在配置中使用 ${func.functionName(arg1,arg2)} 调用
func WithCustomFunction(name string, fn func(args ...string) (string, error)) AdvancedOption {
	return func(c *AdvancedConfig) {
		c.customFuncs[name] = fn
	}
}

// NewAdvanced 创建新的高级配置实例
func NewAdvanced(opts ...AdvancedOption) *AdvancedConfig {
	ac := &AdvancedConfig{
		Config:        New(),
		resolver:      &DefaultVariableResolver{},
		cycleDetector: NewCycleDetector(),
		customFuncs:   make(map[string]func(args ...string) (string, error)),
	}

	for _, opt := range opts {
		opt(ac)
	}

	return ac
}

// LoadAdvanced 加载并解析高级配置文件
func LoadAdvanced(path string, opts ...AdvancedOption) (*AdvancedConfig, error) {
	ac := NewAdvanced(opts...)
	if err := ac.LoadFile(path); err != nil {
		return nil, err
	}
	if err := ac.Resolve(); err != nil {
		return nil, err
	}
	return ac, nil
}

// Resolve 解析配置（高级版本）
// 包含循环检测和自定义函数处理
func (ac *AdvancedConfig) Resolve() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.cycleDetector.Reset()

	if err := ac.processIncludes(); err != nil {
		return err
	}

	return ac.resolveVariablesAdvanced()
}

// resolveVariablesAdvanced 高级变量解析
// 支持自定义解析器和循环检测
func (ac *AdvancedConfig) resolveVariablesAdvanced() error {
	processed := make(map[string]bool)
	maxDepth := 100

	var resolve func(data map[string]interface{}, depth int, path string) error
	resolve = func(data map[string]interface{}, depth int, path string) error {
		if depth > maxDepth {
			return fmt.Errorf("maximum resolution depth exceeded at path: %s", path)
		}

		for k, v := range data {
			currentPath := path + "." + k

			if processed[k] && depth == 0 {
				continue
			}

			switch val := v.(type) {
			case string:
				resolved, err := ac.resolveStringAdvanced(val, data, depth, currentPath)
				if err != nil {
					return fmt.Errorf("error resolving variable at '%s': %w", currentPath, err)
				}
				data[k] = resolved
				processed[k] = true

			case map[string]interface{}:
				if err := resolve(val, depth+1, currentPath); err != nil {
					return err
				}

			case []interface{}:
				if err := ac.resolveSliceAdvanced(val, data, depth, currentPath); err != nil {
					return err
				}
			}
		}
		return nil
	}

	return resolve(ac.rawData, 0, "root")
}

// resolveStringAdvanced 高级字符串解析
// 支持环境变量、函数调用、嵌套变量
func (ac *AdvancedConfig) resolveStringAdvanced(s string, locals map[string]interface{}, depth int, path string) (string, error) {
	pattern := regexp.MustCompile(`\$\{([^}]+)\}`)

	result := s
	maxIterations := 100

	for i := 0; i < maxIterations; i++ {
		matches := pattern.FindAllString(result, -1)
		if len(matches) == 0 {
			break
		}

		for _, match := range matches {
			varName := match[2 : len(match)-1]

			// 检测循环引用
			if err := ac.cycleDetector.Enter(varName); err != nil {
				return "", fmt.Errorf("cycle detected at '%s': %w", path, err)
			}
			defer ac.cycleDetector.Exit(varName)

			// 处理环境变量 ${env.VAR_NAME}
			if strings.HasPrefix(varName, "env.") {
				envVar := strings.TrimPrefix(varName, "env.")
				val := getEnvVariable(envVar)
				result = strings.Replace(result, match, val, 1)
				continue
			}

			// 处理自定义函数 ${func.functionName(args)}
			if strings.HasPrefix(varName, "func.") {
				funcResult, err := ac.executeCustomFunction(varName)
				if err != nil {
					return "", err
				}
				result = strings.Replace(result, match, funcResult, 1)
				continue
			}

			// 获取变量值
			varValue, err := ac.getVariableValueAdvanced(varName, locals)
			if err != nil {
				return "", err
			}

			// 递归解析嵌套变量
			if strings.Contains(varValue, "${") && i < maxIterations-1 {
				nestedResolved, err := ac.resolveStringAdvanced(varValue, locals, depth+1, path)
				if err != nil {
					return "", err
				}
				varValue = nestedResolved
			}

			result = strings.Replace(result, match, varValue, 1)
		}
	}

	return result, nil
}

// getEnvVariable 获取环境变量
// 支持带前缀的环境变量查找
func getEnvVariable(name string) string {
	for _, e := range []string{"", "APP_", "CONFIG_"} {
		if val := getEnv(e + name); val != "" {
			return val
		}
	}
	return ""
}

// getEnv 获取环境变量的占位函数
// 实际项目中可以调用 os.Getenv
func getEnv(name string) string {
	switch name {
	case "PATH":
		return ""
	case "HOME":
		return ""
	default:
		return ""
	}
}

// executeCustomFunction 执行自定义函数
func (ac *AdvancedConfig) executeCustomFunction(varName string) (string, error) {
	parts := strings.SplitN(strings.TrimPrefix(varName, "func."), "(", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid function syntax: %s", varName)
	}

	funcName := parts[0]
	argsStr := strings.TrimRight(parts[1], ")")

	fn, ok := ac.customFuncs[funcName]
	if !ok {
		return "", fmt.Errorf("undefined function: %s", funcName)
	}

	var args []string
	if argsStr != "" {
		args = strings.Split(argsStr, ",")
		for i := range args {
			args[i] = strings.TrimSpace(args[i])
		}
	}

	return fn(args...)
}

// getVariableValueAdvanced 高级获取变量值
func (ac *AdvancedConfig) getVariableValueAdvanced(varName string, locals map[string]interface{}) (string, error) {
	parts := strings.SplitN(varName, ":", 2)

	name := parts[0]
	defaultValue := ""
	if len(parts) > 1 {
		defaultValue = parts[1]
	}

	if val, ok := locals[name]; ok {
		return formatValue(val)
	}

	if val, ok := ac.globals[name]; ok {
		return formatValue(val)
	}

	if ac.env != "" {
		envKey := ac.env + "." + name
		if val, ok := ac.globals[envKey]; ok {
			return formatValue(val)
		}
	}

	if defaultValue != "" {
		return defaultValue, nil
	}

	return "", fmt.Errorf("undefined variable: %s", name)
}

// resolveSliceAdvanced 高级切片解析
func (ac *AdvancedConfig) resolveSliceAdvanced(slice []interface{}, locals map[string]interface{}, depth int, path string) error {
	for i, item := range slice {
		currentPath := fmt.Sprintf("%s[%d]", path, i)

		switch val := item.(type) {
		case string:
			resolved, err := ac.resolveStringAdvanced(val, locals, depth, currentPath)
			if err != nil {
				return fmt.Errorf("error resolving slice element at %s: %w", currentPath, err)
			}
			slice[i] = resolved

		case map[string]interface{}:
			if err := ac.resolveVariablesInMapAdvanced(val, depth, currentPath); err != nil {
				return err
			}

		case []interface{}:
			if err := ac.resolveSliceAdvanced(val, locals, depth, currentPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveVariablesInMapAdvanced 高级 Map 解析
func (ac *AdvancedConfig) resolveVariablesInMapAdvanced(data map[string]interface{}, depth int, path string) error {
	if depth > 100 {
		return fmt.Errorf("maximum resolution depth exceeded at path: %s", path)
	}

	for k, v := range data {
		currentPath := path + "." + k

		switch val := v.(type) {
		case string:
			resolved, err := ac.resolveStringAdvanced(val, data, depth, currentPath)
			if err != nil {
				return fmt.Errorf("error resolving variable at '%s': %w", currentPath, err)
			}
			data[k] = resolved

		case map[string]interface{}:
			if err := ac.resolveVariablesInMapAdvanced(val, depth+1, currentPath); err != nil {
				return err
			}

		case []interface{}:
			if err := ac.resolveSliceAdvanced(val, data, depth, currentPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// RegisterFunction 注册自定义函数
// 可以在配置中使用 ${func.functionName(args)} 调用
func (ac *AdvancedConfig) RegisterFunction(name string, fn func(args ...string) (string, error)) {
	ac.customFuncs[name] = fn
}

// Merge 合并另一个配置对象
func (ac *AdvancedConfig) Merge(other *Config) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	ac.mergeData(ac.rawData, other.rawData)
	return nil
}

// EnvironmentConfig 环境配置管理器
// 用于管理多环境配置（如开发、测试、生产环境）
type EnvironmentConfig struct {
	Configs map[string]*Config // 各环境的配置
	current string             // 当前环境
	mu      sync.RWMutex       // 读写锁
}

// NewEnvironmentConfig 创建新的环境配置管理器
func NewEnvironmentConfig() *EnvironmentConfig {
	return &EnvironmentConfig{
		Configs: make(map[string]*Config),
		current: "development",
	}
}

// LoadEnvironment 加载指定环境的配置
func (ec *EnvironmentConfig) LoadEnvironment(env string, path string, opts ...Option) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	cfg, err := Load(path, opts...)
	if err != nil {
		return fmt.Errorf("failed to load config for environment '%s': %w", env, err)
	}

	if err := cfg.Resolve(); err != nil {
		return fmt.Errorf("failed to resolve config for environment '%s': %w", env, err)
	}

	ec.Configs[env] = cfg
	return nil
}

// Switch 切换到指定环境
func (ec *EnvironmentConfig) Switch(env string) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if _, ok := ec.Configs[env]; !ok {
		return fmt.Errorf("environment not loaded: %s", env)
	}

	ec.current = env
	return nil
}

// Get 获取当前环境的配置
func (ec *EnvironmentConfig) Get() *Config {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.Configs[ec.current]
}

// Current 获取当前环境名称
func (ec *EnvironmentConfig) Current() string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.current
}
