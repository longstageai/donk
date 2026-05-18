package prompt

import (
	"sort"
	"strings"
)

// componentRegistry 组件注册表
// 用于管理所有已注册的提示词组件
type componentRegistry struct {
	components []Component
}

// newComponentRegistry 创建新的组件注册表
func newComponentRegistry() *componentRegistry {
	return &componentRegistry{
		components: make([]Component, 0),
	}
}

// Add 添加组件到注册表
func (r *componentRegistry) Add(c Component) {
	r.components = append(r.components, c)
	r.sort()
}

// Get 根据名称获取组件
func (r *componentRegistry) Get(name string) Component {
	for _, c := range r.components {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// GetAll 获取所有组件
func (r *componentRegistry) GetAll() []Component {
	result := make([]Component, len(r.components))
	copy(result, r.components)
	return result
}

// sort 按优先级排序组件
func (r *componentRegistry) sort() {
	sort.Slice(r.components, func(i, j int) bool {
		return r.components[i].Priority() < r.components[j].Priority()
	})
}

// BuildAll 构建所有组件的内容
func (r *componentRegistry) BuildAll(vars Variables) (map[string]string, error) {
	result := make(map[string]string)
	for _, c := range r.components {
		content, err := c.Content(vars)
		if err != nil {
			return nil, err
		}
		result[c.Name()] = content
	}
	return result, nil
}

// baseComponent 组件基类
// 提供组件的基本实现
type baseComponent struct {
	name     string // 组件名称
	priority int    // 优先级
}

// Name 返回组件名称
func (c *baseComponent) Name() string {
	return c.name
}

// Priority 返回组件优先级
func (c *baseComponent) Priority() int {
	return c.priority
}

// StringResolver 字符串解析器
// 用于动态解析字符串中的占位符
type StringResolver struct {
	funcs map[string]func() string
}

// NewStringResolver 创建新的字符串解析器
func NewStringResolver() *StringResolver {
	return &StringResolver{
		funcs: make(map[string]func() string),
	}
}

// Register 注册解析函数
// name: 占位符名称
// fn: 解析函数
func (r *StringResolver) Register(name string, fn func() string) {
	r.funcs[name] = fn
}

// Resolve 解析字符串中的占位符
// text: 包含占位符的文本
// 返回解析后的文本
func (r *StringResolver) Resolve(text string) string {
	result := text
	for name, fn := range r.funcs {
		placeholder := "{{" + name + "}}"
		result = strings.ReplaceAll(result, placeholder, fn())
	}
	return result
}

// ResolveVariables 解析变量占位符
// text: 包含占位符的文本
// vars: 变量映射
// 返回解析后的文本
func ResolveVariables(text string, vars Variables) string {
	result := text
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, toString(value))
	}
	return result
}

// toString 将任意值转换为字符串
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64, float64:
		return formatAny(val)
	default:
		return ""
	}
}

// formatAny 格式化任意类型为字符串
func formatAny(v any) string {
	switch val := v.(type) {
	case int:
		return string(rune('0' + val%10))
	case int64:
		return string(rune('0' + int(val)%10))
	case float64:
		return "0"
	}
	return ""
}
