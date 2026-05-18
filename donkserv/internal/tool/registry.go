package tool

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// Registry 工具注册表
// 负责管理所有可用的工具，提供注册、获取、列举等功能
type Registry struct {
	mu         sync.RWMutex    // 读写锁，保证并发安全
	tools      map[string]Tool // 工具集合，key为工具名称
	middleware []Middleware    // 全局中间件
}

// NewRegistry 创建新的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools:      make(map[string]Tool),
		middleware: make([]Middleware, 0),
	}
}

// Register 注册一个工具
// 如果工具名称已存在，会返回错误
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("工具已存在: %s", tool.Name())
	}

	r.tools[tool.Name()] = tool
	return nil
}

// Unregister 注销一个工具
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get 根据名称获取工具
// 返回工具和是否存在的标识
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List 返回所有已注册工具的名称列表
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetToolDefinitions 返回所有工具的定义列表
// 用于告诉模型有哪些工具可用
func (r *Registry) GetToolDefinitions() []schema.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]schema.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		props := tool.Parameters()
		params := make(map[string]any)

		// 构建JSON Schema格式的参数定义
		if props != nil && props.Properties != nil {
			properties := make(map[string]any)
			for name, prop := range props.Properties {
				paramType := prop.Type
				if paramType == "" {
					paramType = "string"
				}
				properties[name] = map[string]any{
					"type":        paramType,
					"description": prop.Description,
				}
				if prop.Default != nil {
					properties[name].(map[string]any)["default"] = prop.Default
				}
			}
			params["type"] = "object"
			params["properties"] = properties
			if len(props.Required) > 0 {
				params["required"] = props.Required
			}
		}

		definitions = append(definitions, schema.ToolDefinition{
			Type: "function",
			Function: schema.FunctionProperty{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  params,
			},
		})
	}

	return definitions
}

// Use 添加全局中间件
func (r *Registry) Use(m ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, m...)
}

// Execute 执行指定名称的工具
// 根据参数调用工具并返回执行结果
func (r *Registry) Execute(name string, params map[string]any) (*Result, error) {
	ctx := NewContext(name, params)
	return r.ExecuteWithContext(ctx)
}

// ExecuteWithContext 使用自定义上下文执行工具
func (r *Registry) ExecuteWithContext(ctx *Context) (*Result, error) {
	r.mu.RLock()
	tool, ok := r.tools[ctx.ToolName]
	middleware := make([]Middleware, len(r.middleware))
	copy(middleware, r.middleware)
	r.mu.RUnlock()

	if !ok {
		return NewErrorResult(NewToolError(ErrCodeNotFound, fmt.Sprintf("工具不存在: %s", ctx.ToolName))),
			fmt.Errorf("工具不存在: %s", ctx.ToolName)
	}

	// 打印工具调用日志
	paramsJSON, _ := json.Marshal(ctx.Params)
	logger.Info(fmt.Sprintf("[工具调用] 名称: %s, 参数: %s", ctx.ToolName, paramsJSON), nil)

	// 构建执行链
	handler := func(ctx *Context) (*Result, error) {
		return tool.Execute(ctx)
	}

	// 应用中间件（逆序）
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	// 执行
	return handler(ctx)
}
