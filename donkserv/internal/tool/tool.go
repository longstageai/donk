package tool

import (
	"errors"
	"fmt"
	"time"
)

// generateRequestID 生成唯一请求ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// Middleware 中间件类型
// 中间件是一个函数，接收一个Handler并返回一个新的Handler
type Middleware func(next Handler) Handler

// Tool 工具接口
// 所有可被Agent调用的工具都需要实现此接口
type Tool interface {
	Name() string
	Description() string
	Version() string
	Category() string
	Parameters() *Schema
	Execute(ctx *Context) (*Result, error)
}

// BaseTool 工具基础实现
// 提供工具的默认实现，方便扩展
type BaseTool struct {
	name        string
	description string
	version     string
	parameters  *Schema
	handler     Handler
	category    string
	timeout     time.Duration
}

// NewBaseTool 创建基础工具
func NewBaseTool(name, description string) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		version:     "1.0.0",
		parameters:  NewSchema(),
		timeout:     30 * time.Second,
	}
}

// Name 返回工具名称
func (t *BaseTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *BaseTool) Description() string {
	return t.description
}

// Version 返回工具版本
func (t *BaseTool) Version() string {
	return t.version
}

// Parameters 返回参数定义
func (t *BaseTool) Parameters() *Schema {
	return t.parameters
}

// SetVersion 设置版本
func (t *BaseTool) SetVersion(version string) *BaseTool {
	t.version = version
	return t
}

// SetCategory 设置分类
func (t *BaseTool) SetCategory(category string) *BaseTool {
	t.category = category
	return t
}

// Category 获取分类
func (t *BaseTool) Category() string {
	return t.category
}

// SetTimeout 设置超时时间
func (t *BaseTool) SetTimeout(timeout time.Duration) *BaseTool {
	t.timeout = timeout
	return t
}

// Timeout 获取超时时间
func (t *BaseTool) Timeout() time.Duration {
	return t.timeout
}

// SetHandler 设置处理函数
func (t *BaseTool) SetHandler(handler Handler) *BaseTool {
	t.handler = handler
	return t
}

// Execute 执行工具
func (t *BaseTool) Execute(ctx *Context) (*Result, error) {
	if t.handler == nil {
		return nil, fmt.Errorf("工具 %s 未设置处理函数", t.name)
	}
	return t.handler(ctx)
}

// Handler 工具执行函数类型
type Handler func(ctx *Context) (*Result, error)

// Schema 参数Schema定义
type Schema struct {
	Type       string               `json:"type"`
	Properties map[string]*Property `json:"properties"`
	Required   []string             `json:"required"`
}

// NewSchema 创建新的Schema
func NewSchema() *Schema {
	return &Schema{
		Type:       "object",
		Properties: make(map[string]*Property),
		Required:   make([]string, 0),
	}
}

// Property 参数属性
type Property struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default"`
	Enum        []interface{} `json:"enum"`
	Format      string        `json:"format"`
	Minimum     *float64      `json:"minimum"`
	Maximum     *float64      `json:"maximum"`
	MinLength   *int          `json:"minLength"`
	MaxLength   *int          `json:"maxLength"`
}

// AddProperty 添加属性
func (s *Schema) AddProperty(name, paramType, description string, required bool) *Schema {
	s.Properties[name] = &Property{
		Type:        paramType,
		Description: description,
	}
	if required {
		s.Required = append(s.Required, name)
	}
	return s
}

// AddStringProperty 添加字符串属性
func (s *Schema) AddStringProperty(name, description string, required bool) *Schema {
	return s.AddProperty(name, "string", description, required)
}

// AddIntegerProperty 添加整数属性
func (s *Schema) AddIntegerProperty(name, description string, required bool) *Schema {
	return s.AddProperty(name, "integer", description, required)
}

// AddNumberProperty 添加数字属性
func (s *Schema) AddNumberProperty(name, description string, required bool) *Schema {
	return s.AddProperty(name, "number", description, required)
}

// AddBooleanProperty 添加布尔属性
func (s *Schema) AddBooleanProperty(name, description string, required bool) *Schema {
	return s.AddProperty(name, "boolean", description, required)
}

// AddArrayProperty 添加数组属性
func (s *Schema) AddArrayProperty(name, description string, required bool) *Schema {
	return s.AddProperty(name, "array", description, required)
}

// AddObjectProperty 添加对象属性
func (s *Schema) AddObjectProperty(name, description string, required bool) *Schema {
	return s.AddProperty(name, "object", description, required)
}

// SetDefault 设置默认值
func (s *Schema) SetDefault(name string, value interface{}) *Schema {
	if prop, ok := s.Properties[name]; ok {
		prop.Default = value
	}
	return s
}

// SetEnum 设置枚举值
func (s *Schema) SetEnum(name string, values ...interface{}) *Schema {
	if prop, ok := s.Properties[name]; ok {
		prop.Enum = values
	}
	return s
}

// Category 工具分类常量
type Category string

const (
	CategorySearch  Category = "search"
	CategoryData    Category = "data"
	CategoryUtility Category = "utility"
	CategoryFile    Category = "file"
	CategoryNetwork Category = "network"
	CategoryCompute Category = "compute"
)

// ToolError 工具执行错误
type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

// Error 实现 error 接口
func (e *ToolError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewToolError 创建工具错误
func NewToolError(code, message string, details ...any) *ToolError {
	err := &ToolError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// CommonErrorCodes 常用错误码
var (
	ErrCodeNotFound       = "TOOL_NOT_FOUND"
	ErrCodeInvalidParams  = "INVALID_PARAMS"
	ErrCodeExecution      = "EXECUTION_ERROR"
	ErrCodeTimeout        = "TIMEOUT"
	ErrCodeCancelled      = "CANCELLED"
	ErrCodeRateLimit      = "RATE_LIMIT"
	ErrCodeNotImplemented = "NOT_IMPLEMENTED"
)

// IsToolError 判断是否为工具错误
func IsToolError(err error) bool {
	var toolErr *ToolError
	return errors.As(err, &toolErr)
}

// GetToolError 获取工具错误
func GetToolError(err error) *ToolError {
	var toolErr *ToolError
	if errors.As(err, &toolErr) {
		return toolErr
	}
	return nil
}
