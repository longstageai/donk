package builder

import (
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

// Builder 工具构建器
// 用于以函数式方式创建工具
type Builder struct {
	name        string
	description string
	version     string
	category    string
	parameters  *tool.Schema
	handler     tool.Handler
	timeout     time.Duration
}

// New 创建新的工具构建器
func New(name, description string) *Builder {
	return &Builder{
		name:        name,
		description: description,
		version:     "1.0.0",
		parameters:  tool.NewSchema(),
		timeout:     30 * time.Second,
	}
}

// Name 获取工具名称
func (b *Builder) Name() string {
	return b.name
}

// Description 获取工具描述
func (b *Builder) Description() string {
	return b.description
}

// Version 设置工具版本
func (b *Builder) Version(version string) *Builder {
	b.version = version
	return b
}

// Category 设置工具分类
func (b *Builder) Category(category string) *Builder {
	b.category = category
	return b
}

// Param 添加参数
// name: 参数名称
// paramType: 参数类型 (string, number, integer, boolean, object, array)
// description: 参数描述
// required: 是否必填
func (b *Builder) Param(name, paramType, description string, required bool) *Builder {
	b.parameters.AddProperty(name, paramType, description, required)
	return b
}

// StringParam 添加字符串参数
func (b *Builder) StringParam(name, description string, required bool) *Builder {
	return b.Param(name, "string", description, required)
}

// IntParam 添加整数参数
func (b *Builder) IntParam(name, description string, required bool) *Builder {
	return b.Param(name, "integer", description, required)
}

// NumberParam 添加数字参数
func (b *Builder) NumberParam(name, description string, required bool) *Builder {
	return b.Param(name, "number", description, required)
}

// BoolParam 添加布尔参数
func (b *Builder) BoolParam(name, description string, required bool) *Builder {
	return b.Param(name, "boolean", description, required)
}

// ObjectParam 添加对象参数
func (b *Builder) ObjectParam(name, description string, required bool) *Builder {
	return b.Param(name, "object", description, required)
}

// ArrayParam 添加数组参数
func (b *Builder) ArrayParam(name, description string, required bool) *Builder {
	return b.Param(name, "array", description, required)
}

// SetDefault 设置参数默认值
func (b *Builder) SetDefault(name string, value interface{}) *Builder {
	b.parameters.SetDefault(name, value)
	return b
}

// SetEnum 设置枚举值
func (b *Builder) SetEnum(name string, values ...interface{}) *Builder {
	b.parameters.SetEnum(name, values...)
	return b
}

// Timeout 设置超时时间
func (b *Builder) Timeout(timeout time.Duration) *Builder {
	b.timeout = timeout
	return b
}

// Handler 设置处理函数
func (b *Builder) Handler(handler tool.Handler) *Builder {
	b.handler = handler
	return b
}

// HandlerFunc 设置处理函数(函数类型)
func (b *Builder) HandlerFunc(handler func(ctx *tool.Context) (*tool.Result, error)) *Builder {
	b.handler = handler
	return b
}

// Build 构建工具
func (b *Builder) Build() (tool.Tool, error) {
	if b.name == "" {
		return nil, &buildError{"工具名称不能为空"}
	}
	if b.description == "" {
		return nil, &buildError{"工具描述不能为空"}
	}
	if b.handler == nil {
		return nil, &buildError{"工具处理函数不能为空"}
	}

	// 创建基础工具
	tool := tool.NewBaseTool(b.name, b.description)
	tool.SetVersion(b.version)
	tool.SetCategory(b.category)
	tool.SetTimeout(b.timeout)

	// 设置参数
	for name, prop := range b.parameters.Properties {
		tool.Parameters().Properties[name] = prop
	}
	tool.Parameters().Required = b.parameters.Required

	// 设置处理函数
	tool.SetHandler(b.handler)

	return tool, nil
}

// MustBuild 构建工具(panic on error)
func (b *Builder) MustBuild() tool.Tool {
	tool, err := b.Build()
	if err != nil {
		panic(err)
	}
	return tool
}

// buildError 构建错误
type buildError struct {
	message string
}

func (e *buildError) Error() string {
	return e.message
}
