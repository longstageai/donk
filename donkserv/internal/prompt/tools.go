package prompt

import (
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/tool"
)

// ToolsComponent 工具描述组件
// 从 tool.Registry 获取工具定义，生成可供大模型理解的工具描述
// 用于告诉模型有哪些工具可用以及各工具的功能说明
type ToolsComponent struct {
	baseComponent
	registry *tool.Registry
}

// NewToolsComponent 创建工具描述组件
// registry: 工具注册表，用于获取工具定义
func NewToolsComponent(registry *tool.Registry) *ToolsComponent {
	return &ToolsComponent{
		baseComponent: baseComponent{
			name:     "tools",
			priority: 40,
		},
		registry: registry,
	}
}

// Content 获取工具描述内容
// vars: 变量映射，用于替换模板中的占位符（此处未使用）
// 返回格式化的工具列表描述文本
func (c *ToolsComponent) Content(vars Variables) (string, error) {
	// 从注册表获取工具定义
	defs := c.registry.GetToolDefinitions()
	if len(defs) == 0 {
		return "", nil
	}

	// 构建工具列表
	var parts []string
	for _, def := range defs {
		name := def.Function.Name
		desc := def.Function.Description
		if name != "" {
			parts = append(parts, fmt.Sprintf("- %s: %s", name, desc))
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	// 格式化输出
	result := "## 可用工具\n\n"
	result += "你可以使用以下工具来帮助完成用户请求：\n\n"
	result += strings.Join(parts, "\n\n")
	return result, nil
}
