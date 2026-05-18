package prompt

// SystemComponent 系统提示组件
// 加载 SYSTEM.md 文件作为系统提示
type SystemComponent struct {
	baseComponent
	loader *FileLoader
}

// NewSystemComponent 创建系统提示组件
// loader: 文件加载器，用于读取系统配置文件
func NewSystemComponent(loader *FileLoader) *SystemComponent {
	return &SystemComponent{
		baseComponent: baseComponent{
			name:     "system",
			priority: 30,
		},
		loader: loader,
	}
}

// Content 获取系统提示内容
// vars: 变量映射，用于替换模板中的占位符
// 返回组合后的系统提示文本
func (c *SystemComponent) Content(vars Variables) (string, error) {
	contents := make([]string, 0)

	// 加载 SYSTEM.md - 系统提示
	system, err := c.loader.Load("system")
	if err == nil && system != "" {
		contents = append(contents, system)
	}

	// 拼接所有内容
	result := ""
	for i, content := range contents {
		if i > 0 {
			result += "\n\n"
		}
		result += ResolveVariables(content, vars)
	}

	return result, nil
}
