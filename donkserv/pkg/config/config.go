package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config 配置对象结构体
// 包含原始配置数据、全局变量、环境配置等信息
type Config struct {
	mu           sync.RWMutex           // 读写锁，保证并发安全
	rawData      map[string]interface{} // 原始配置数据
	globals      map[string]interface{} // 全局变量存储
	env          string                 // 当前环境名称
	includes     []string               // 要包含的配置文件列表
	basePath     string                 // 基础路径，用于解析相对路径
	loader       Loader                 // 文件加载器
	variableRule VariableRule           // 变量规则
}

// Loader 接口
// 用于自定义配置文件加载方式
type Loader interface {
	Load(path string) ([]byte, error)
}

// FileLoader 文件加载器实现
// 默认使用 os.ReadFile 读取文件
type FileLoader struct{}

// Load 加载指定路径的配置文件
func (f *FileLoader) Load(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// VariableRule 变量规则结构体
// 支持自定义变量匹配模式和替换函数
type VariableRule struct {
	Pattern     *regexp.Regexp
	ReplaceFunc func(match string) string
}

// 默认变量匹配模式: ${variable_name} 或 ${variable_name:default_value}
var defaultVariablePattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// Option 配置选项函数类型
// 用于通过函数式选项模式配置 Config
type Option func(*Config)

// WithLoader 设置自定义文件加载器
// 示例: config.WithLoader(&customLoader{})
func WithLoader(loader Loader) Option {
	return func(c *Config) {
		c.loader = loader
	}
}

// WithEnvironment 设置环境名称
// 示例: config.WithEnvironment("production")
func WithEnvironment(env string) Option {
	return func(c *Config) {
		c.env = env
	}
}

// WithGlobals 设置全局变量
// 示例: config.WithGlobals(map[string]interface{}{"key": "value"})
func WithGlobals(globals map[string]interface{}) Option {
	return func(c *Config) {
		c.globals = globals
	}
}

// WithVariableRule 设置自定义变量规则
func WithVariableRule(rule VariableRule) Option {
	return func(c *Config) {
		c.variableRule = rule
	}
}

// WithIncludes 设置要包含的配置文件列表
func WithIncludes(includes []string) Option {
	return func(c *Config) {
		c.includes = includes
	}
}

// New 创建新的 Config 实例
// 默认值:
//   - 环境: development
//   - 加载器: FileLoader{}
//   - 变量模式: ${variable_name}
func New(opts ...Option) *Config {
	c := &Config{
		rawData: make(map[string]interface{}),
		globals: make(map[string]interface{}),
		env:     "development",
		loader:  &FileLoader{},
		variableRule: VariableRule{
			Pattern: defaultVariablePattern,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Load 加载配置文件并返回 Config 实例
// 参数:
//   - path: 配置文件路径
//   - opts: 可选的配置选项
//
// 返回:
//   - *Config: 配置对象
//   - error: 加载错误
func Load(path string, opts ...Option) (*Config, error) {
	c := New(opts...)
	if err := c.LoadFile(path); err != nil {
		return nil, err
	}
	return c, nil
}

// LoadFile 加载配置文件到当前 Config 实例
func (c *Config) LoadFile(path string) error {
	data, err := c.loader.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config file %s: %w", path, err)
	}

	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	c.rawData = rawData

	// 提取基础路径，用于解析 include 的相对路径
	lastSlash := strings.LastIndexAny(path, "/\\")
	if lastSlash != -1 {
		c.basePath = path[:lastSlash+1]
	}

	return nil
}

// Get 根据 key 获取配置值
// key 支持点号分隔的嵌套访问，如 "app.name"
// 返回: (值, 是否存在)
func (c *Config) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.get(key, c.rawData)
}

// get 内部递归获取方法
func (c *Config) get(key string, data map[string]interface{}) (interface{}, bool) {
	keys := strings.Split(key, ".")
	current := data

	for i, k := range keys {
		if v, ok := current[k]; ok {
			if i == len(keys)-1 {
				return v, true
			}
			if nextMap, ok := v.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return nil, false
}

// GetString 获取字符串类型的配置值
// 如果值不存在或类型不匹配，返回空字符串
func (c *Config) GetString(key string) string {
	if v, ok := c.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt 获取整数类型的配置值
// 支持 int, int64, float64 类型
// 如果值不存在或类型不匹配，返回 0
func (c *Config) GetInt(key string) int {
	if v, ok := c.Get(key); ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

// GetBool 获取布尔类型的配置值
// 如果值不存在或类型不匹配，返回 false
func (c *Config) GetBool(key string) bool {
	if v, ok := c.Get(key); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetMap 获取 map 类型的配置值
// 如果值不存在或类型不匹配，返回 nil
func (c *Config) GetMap(key string) map[string]interface{} {
	if v, ok := c.Get(key); ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// GetSlice 获取切片类型的配置值
// 如果值不存在或类型不匹配，返回 nil
func (c *Config) GetSlice(key string) []interface{} {
	if v, ok := c.Get(key); ok {
		if s, ok := v.([]interface{}); ok {
			return s
		}
	}
	return nil
}

// GetStruct 获取指定 key 的配置并转换为 struct
// dest 必须是指向 struct 的指针
// 支持自动类型转换（字符串数字转为 int/float）
//
// 示例:
//
//	type ServerConfig struct {
//	    Host string `yaml:"host"`
//	    Port int    `yaml:"port"`
//	}
//
//	var config ServerConfig
//	if err := cfg.GetStruct("server", &config); err != nil {
//	    log.Fatal(err)
//	}
func (c *Config) GetStruct(key string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if v, ok := c.Get(key); ok {
		if m, ok := v.(map[string]interface{}); ok {
			// 转换类型：将字符串数字转为实际数字类型
			converted := c.convertTypes(m)
			bytes, err := yaml.Marshal(converted)
			if err != nil {
				return fmt.Errorf("failed to marshal section %s: %w", key, err)
			}
			if err := yaml.Unmarshal(bytes, dest); err != nil {
				return fmt.Errorf("failed to unmarshal section %s: %w", key, err)
			}
			return nil
		}
	}
	return fmt.Errorf("key not found or not a map: %s", key)
}

// convertTypes 递归转换 map 中的类型
// 将 int64 转为 int，将字符串数字转为实际数字类型
func (c *Config) convertTypes(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = c.convertTypes(val)
		case []interface{}:
			result[k] = c.convertSliceTypes(val)
		case int64:
			if i := int(val); int64(i) == val {
				result[k] = i
			} else {
				result[k] = val
			}
		case string:
			// 尝试将字符串形式的数字转换为实际数字类型
			if converted := c.convertStringToNumber(val); converted != nil {
				result[k] = converted
			} else {
				result[k] = val
			}
		default:
			result[k] = val
		}
	}
	return result
}

// convertStringToNumber 尝试将字符串转换为数字
// 如果字符串是纯数字（如 "8080"），则转换为 int
// 如果是浮点数（如 "3.14"），则转换为 float64
// 否则返回 nil
func (c *Config) convertStringToNumber(s string) interface{} {
	if !isNumericString(s) {
		return nil
	}

	var intVal int64
	if _, err := fmt.Sscanf(s, "%d", &intVal); err == nil {
		return int(intVal)
	}
	var floatVal float64
	if _, err := fmt.Sscanf(s, "%f", &floatVal); err == nil {
		return floatVal
	}
	return nil
}

// isNumericString 检查字符串是否为纯数字
// 支持整数、浮点数、正负数
func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	hasDot := false
	hasDigit := false
	for _, c := range s {
		if c == '.' {
			if hasDot {
				return false
			}
			hasDot = true
		} else if c >= '0' && c <= '9' {
			hasDigit = true
		} else if c == '-' || c == '+' {
			// 允许开头的正负号
		} else {
			return false
		}
	}
	return hasDigit
}

// convertSliceTypes 递归转换切片中的类型
func (c *Config) convertSliceTypes(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))
	for i, item := range slice {
		switch val := item.(type) {
		case map[string]interface{}:
			result[i] = c.convertTypes(val)
		case []interface{}:
			result[i] = c.convertSliceTypes(val)
		case int64:
			if j := int(val); int64(j) == val {
				result[i] = j
			} else {
				result[i] = val
			}
		case string:
			if converted := c.convertStringToNumber(val); converted != nil {
				result[i] = converted
			} else {
				result[i] = val
			}
		default:
			result[i] = val
		}
	}
	return result
}

// Set 设置指定 key 的值
// key 支持点号分隔的嵌套设置
func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := strings.Split(key, ".")
	c.set(keys, value, c.rawData)
}

// set 内部递归设置方法
func (c *Config) set(keys []string, value interface{}, data map[string]interface{}) {
	if len(keys) == 1 {
		data[keys[0]] = value
		return
	}

	current, ok := data[keys[0]]
	if !ok {
		current = make(map[string]interface{})
		data[keys[0]] = current
	}

	if nextMap, ok := current.(map[string]interface{}); ok {
		c.set(keys[1:], value, nextMap)
	}
}

// SetGlobal 设置全局变量
// 全局变量可在配置文件的任何地方通过 ${variable_name} 引用
func (c *Config) SetGlobal(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.globals[key] = value
}

// GetGlobal 获取全局变量
func (c *Config) GetGlobal(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.globals[key]
	return v, ok
}

// All 获取所有原始配置数据
func (c *Config) All() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rawData
}

// Resolve 解析配置文件
// 执行以下步骤：
// 1. 提取 variables 节定义的变量到全局变量
// 2. 处理 includes 包含其他配置文件
// 3. 解析所有变量引用
func (c *Config) Resolve() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.extractVariables()

	if err := c.processIncludes(); err != nil {
		return err
	}

	if err := c.resolveVariables(); err != nil {
		return err
	}

	return nil
}

// extractVariables 提取 variables 节定义的变量
// 将 variables 节点下的所有键值对添加到全局变量存储
func (c *Config) extractVariables() {
	if variables, ok := c.rawData["variables"].(map[string]interface{}); ok {
		delete(c.rawData, "variables")
		for k, v := range variables {
			c.globals[k] = v
		}
	}
}

// processIncludes 处理 includes 节
// 加载并合并所有包含的配置文件
func (c *Config) processIncludes() error {
	if includes, ok := c.rawData["includes"].([]interface{}); ok {
		delete(c.rawData, "includes")

		for _, inc := range includes {
			incPath, ok := inc.(string)
			if !ok {
				continue
			}

			subData, err := c.loadInclude(incPath)
			if err != nil {
				return err
			}

			c.mergeData(c.rawData, subData)
		}
	}

	// 处理 includes 中可能定义的变量
	if variables, ok := c.rawData["variables"].(map[string]interface{}); ok {
		delete(c.rawData, "variables")
		for k, v := range variables {
			c.globals[k] = v
		}
	}

	return nil
}

// loadInclude 加载单个包含文件
func (c *Config) loadInclude(path string) (map[string]interface{}, error) {
	// 处理相对路径
	if c.basePath != "" && !strings.HasPrefix(path, "/") && !strings.Contains(path, ":") {
		path = c.basePath + path
	}

	data, err := c.loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load include file %s: %w", path, err)
	}

	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse include YAML %s: %w", path, err)
	}

	include := &Config{
		rawData: rawData,
		globals: c.globals,
		env:     c.env,
		loader:  c.loader,
	}

	if err := include.processIncludes(); err != nil {
		return nil, err
	}

	return include.rawData, nil
}

// mergeData 递归合并配置数据
// 源配置中的值会覆盖目标配置中的值
// 对于嵌套的 map，会递归合并
func (c *Config) mergeData(target, source map[string]interface{}) {
	for k, v := range source {
		if existing, ok := target[k]; ok {
			if existingMap, ok1 := existing.(map[string]interface{}); ok1 {
				if sourceMap, ok2 := v.(map[string]interface{}); ok2 {
					c.mergeData(existingMap, sourceMap)
					continue
				}
			}
		}
		target[k] = v
	}
}

// resolveVariables 解析所有变量引用
// 遍历配置数据，将 ${variable_name} 替换为实际值
// 支持嵌套变量引用和默认值
func (c *Config) resolveVariables() error {
	visiting := make(map[string]string) // 用于检测循环引用
	maxIterations := 100                // 最大迭代次数，防止无限循环

	for iteration := 0; iteration < maxIterations; iteration++ {
		processed := make(map[string]bool) // 记录已处理的键
		changed := false

		var resolve func(data map[string]interface{}, path string) error
		resolve = func(data map[string]interface{}, path string) error {
			for k, v := range data {
				currentPath := path + "." + k

				if processed[currentPath] {
					continue
				}

				switch val := v.(type) {
				case string:
					if strings.Contains(val, "${") {
						resolved, err := c.resolveString(val, data, currentPath, visiting)
						if err != nil {
							return fmt.Errorf("error resolving variable at key '%s': %w", k, err)
						}
						if resolved != val {
							data[k] = resolved
							changed = true
						}
					}
					processed[currentPath] = true

				case map[string]interface{}:
					if err := resolve(val, currentPath); err != nil {
						return err
					}
					processed[currentPath] = true

				case []interface{}:
					if err := c.resolveSlice(val, data, currentPath, visiting); err != nil {
						return err
					}
					processed[currentPath] = true
				}
			}
			return nil
		}

		if err := resolve(c.rawData, "root"); err != nil {
			return err
		}

		// 如果一轮迭代中没有发生任何变化，则退出
		if !changed {
			break
		}
	}

	return nil
}

// resolveString 解析单个字符串中的变量引用
// 支持嵌套变量引用
func (c *Config) resolveString(s string, locals map[string]interface{}, path string, visiting map[string]string) (string, error) {
	pattern := defaultVariablePattern

	result := s
	maxIterations := 100

	for i := 0; i < maxIterations; i++ {
		match := pattern.FindString(result)
		if match == "" {
			break
		}

		// 提取变量名（去掉 ${ 和 }）
		varName := match[2 : len(match)-1]

		// 检测循环引用
		if existingPath, exists := visiting[varName]; exists {
			return "", fmt.Errorf("circular reference detected: %s -> %s", existingPath, path)
		}

		visiting[varName] = path
		defer delete(visiting, varName)

		// 获取变量值
		varValue, err := c.getVariableValue(varName, locals)
		if err != nil {
			return "", err
		}

		// 递归解析嵌套变量引用
		if strings.Contains(varValue, "${") && i < maxIterations-1 {
			nestedResolved, err := c.resolveString(varValue, locals, path+".nested", visiting)
			if err != nil {
				return "", err
			}
			varValue = nestedResolved
		}

		result = strings.Replace(result, match, varValue, 1)
	}

	return result, nil
}

// getVariableValue 获取变量值
// 查找顺序：局部变量 -> 全局变量 -> 环境变量 -> 默认值 -> 报错
func (c *Config) getVariableValue(varName string, locals map[string]interface{}) (string, error) {
	hasDefault := false
	defaultValue := ""
	var name string

	// 解析默认值语法: ${variable_name:default_value}
	if idx := strings.Index(varName, ":"); idx != -1 {
		name = varName[:idx]
		defaultValue = varName[idx+1:]
		hasDefault = true
	} else {
		name = varName
	}

	// 1. 查找局部变量
	if val, ok := locals[name]; ok {
		return c.valueToString(val)
	}

	// 2. 查找全局变量
	if val, ok := c.globals[name]; ok {
		return c.valueToString(val)
	}

	// 3. 查找环境特定变量 (如 production.port)
	if c.env != "" {
		envKey := c.env + "." + name
		if val, ok := c.globals[envKey]; ok {
			return c.valueToString(val)
		}
	}

	// 4. 使用默认值
	if hasDefault {
		return defaultValue, nil
	}

	return "", fmt.Errorf("undefined variable: %s", name)
}

// valueToString 将任意类型的值转换为字符串
func (c *Config) valueToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", v), nil
	default:
		return "", fmt.Errorf("unsupported variable type: %T", value)
	}
}

// resolveSlice 解析切片中的变量引用
func (c *Config) resolveSlice(slice []interface{}, locals map[string]interface{}, path string, visiting map[string]string) error {
	for i, item := range slice {
		switch val := item.(type) {
		case string:
			resolved, err := c.resolveString(val, locals, path+"["+fmt.Sprint(i)+"]", visiting)
			if err != nil {
				return fmt.Errorf("error resolving slice element %d: %w", i, err)
			}
			slice[i] = resolved

		case map[string]interface{}:
			if err := c.resolveVariablesInMap(val, path+"["+fmt.Sprint(i)+"]", visiting); err != nil {
				return err
			}

		case []interface{}:
			if err := c.resolveSlice(val, locals, path+"["+fmt.Sprint(i)+"]", visiting); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveVariablesInMap 解析 map 中的变量引用
func (c *Config) resolveVariablesInMap(data map[string]interface{}, path string, visiting map[string]string) error {
	for k, v := range data {
		currentPath := path + "." + k
		switch val := v.(type) {
		case string:
			resolved, err := c.resolveString(val, data, currentPath, visiting)
			if err != nil {
				return fmt.Errorf("error resolving variable at key '%s': %w", k, err)
			}
			data[k] = resolved

		case map[string]interface{}:
			if err := c.resolveVariablesInMap(val, currentPath, visiting); err != nil {
				return err
			}

		case []interface{}:
			if err := c.resolveSlice(val, data, currentPath, visiting); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetEnvironment 设置当前环境
func (c *Config) SetEnvironment(env string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.env = env
}

// GetEnvironment 获取当前环境
func (c *Config) GetEnvironment() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.env
}

// MustGet 获取配置值，如果不存在则 panic
func (c *Config) MustGet(key string) interface{} {
	if v, ok := c.Get(key); ok {
		return v
	}
	panic(fmt.Errorf("config key not found: %s", key))
}

// MustGetString 获取字符串配置值，如果不存在则 panic
func (c *Config) MustGetString(key string) string {
	return c.GetString(key)
}

// MustGetInt 获取整数配置值，如果不存在则 panic
func (c *Config) MustGetInt(key string) int {
	return c.GetInt(key)
}

// MustGetBool 获取布尔配置值，如果不存在则 panic
func (c *Config) MustGetBool(key string) bool {
	return c.GetBool(key)
}

// Unmarshal 将整个配置解析并转换为 struct
// 自动处理类型转换
//
// 示例:
//
//	type FullConfig struct {
//	    App      AppConfig      `yaml:"app"`
//	    Server   ServerConfig   `yaml:"server"`
//	    Database DatabaseConfig `yaml:"database"`
//	}
//
//	var config FullConfig
//	if err := cfg.Unmarshal(&config); err != nil {
//	    log.Fatal(err)
//	}
func (c *Config) Unmarshal(dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	converted := c.convertTypes(c.rawData)

	bytes, err := yaml.Marshal(converted)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	if err := yaml.Unmarshal(bytes, dest); err != nil {
		return fmt.Errorf("failed to unmarshal into struct: %w", err)
	}

	return nil
}

// flattenMap 将嵌套的 map 展平为单层 map
// key 使用点号分隔，如 {app: {name: "test"}} -> {"app.name": "test"}
func (c *Config) flattenMap(prefix string, data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range data {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			nested := c.flattenMap(key, val)
			for nk, nv := range nested {
				result[nk] = nv
			}
		default:
			result[key] = val
		}
	}

	return result
}

// UnmarshalWithPrefix 将指定前缀的配置转换为 struct
// 只转换匹配前缀的配置项
func (c *Config) UnmarshalWithPrefix(prefix string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	prefix = prefix + "."
	data := c.flattenMap(prefix, c.rawData)

	bytes, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	if err := yaml.Unmarshal(bytes, dest); err != nil {
		return fmt.Errorf("failed to unmarshal into struct: %w", err)
	}

	return nil
}
