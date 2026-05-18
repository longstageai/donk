package prompt

import (
	"sort"

	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// DefaultWorkspace 默认工作目录
const DefaultWorkspace = "./data/workspace"

// Component 提示词组件接口
// 所有提示词组件都需要实现此接口
type Component interface {
	// Name 返回组件名称（唯一标识）
	Name() string
	// Content 获取组件内容
	// vars: 变量映射，用于替换模板中的占位符
	Content(vars Variables) (string, error)
	// Priority 返回组件优先级（数字越小越先加载）
	Priority() int
}

// Variables 提示词变量映射
// 用于在模板中替换占位符
type Variables map[string]any

// BuildOptions 构建选项
// 控制提示词构建过程中的各种行为
type BuildOptions struct {
	MaxTokens    int  // 最大token数量限制
	Compress     bool // 是否启用压缩
	IncludeTools bool // 是否包含工具描述
	HistoryLimit int  // 历史消息数量限制
}

// BuildResult 构建结果
// 包含构建后的各个部分
type BuildResult struct {
	System     string                  // 系统提示词
	Components map[string]string       // 各组件的原始内容
	Tools      []schema.ToolDefinition // 工具定义列表
}

// PromptManager 提示词管理器
// 负责管理所有提示词组件并构建最终提示词
type PromptManager struct {
	workspace  string               // 工作目录路径
	components map[string]Component // 已注册的组件
	registry   *tool.Registry       // 工具注册表
	loader     *FileLoader          // 文件加载器
	options    BuildOptions         // 构建选项
}

// NewPrompt 创建提示词管理器
// workspace: 工作目录路径
// registry: 工具注册表
func NewPrompt(workspace string, registry *tool.Registry) *PromptManager {
	loader := NewFileLoader(workspace)
	pm := &PromptManager{
		workspace:  workspace,
		components: make(map[string]Component),
		registry:   registry,
		loader:     loader,
		options: BuildOptions{
			MaxTokens:    8000,
			Compress:     false,
			IncludeTools: true,
			HistoryLimit: 10,
		},
	}
	pm.registerDefaultComponents()
	return pm
}

// NewDefaultPrompt 使用默认工作目录创建提示词管理器
// registry: 工具注册表
func NewDefaultPrompt(registry *tool.Registry) *PromptManager {
	return NewPrompt(DefaultWorkspace, registry)
}

// registerDefaultComponents 注册默认组件
// 添加系统内置的提示词组件
func (p *PromptManager) registerDefaultComponents() {
	// 注册基础组件
	p.components["system"] = NewSystemComponent(p.loader)
	// 注册工具组件（如果提供了注册表）
	if p.registry != nil {
		p.components["tools"] = NewToolsComponent(p.registry)
	}
}

// Register 注册自定义组件
// name: 组件名称
// c: 组件实例
func (p *PromptManager) Register(name string, c Component) {
	p.components[name] = c
}

// GetComponent 获取指定名称的组件
// name: 组件名称
// 返回组件实例，如果不存在则返回nil
func (p *PromptManager) GetComponent(name string) Component {
	return p.components[name]
}

// SetOptions 设置构建选项
// opts: 新的构建选项
func (p *PromptManager) SetOptions(opts BuildOptions) {
	p.options = opts
}

// Build 构建提示词
// vars: 变量映射
// 返回构建结果，包含系统提示词、组件内容和工具定义
func (p *PromptManager) Build(vars Variables) (*BuildResult, error) {
	// 按优先级排序组件
	comps := p.getSortedComponents()
	components := make(map[string]string)
	systemParts := make([]string, 0)

	// 依次获取各组件内容
	for _, c := range comps {
		content, err := c.Content(vars)
		if err != nil {
			continue
		}
		if content != "" {
			components[c.Name()] = content
			// 工具组件单独处理，不加入system
			if c.Name() != "tools" {
				systemParts = append(systemParts, content)
			}
		}
	}

	// 获取工具定义
	var tools []schema.ToolDefinition
	if p.options.IncludeTools && p.registry != nil {
		tools = p.registry.GetToolDefinitions()
	}

	return &BuildResult{
		System:     joinWithNewline(systemParts...),
		Components: components,
		Tools:      tools,
	}, nil
}

// getSortedComponents 获取排序后的组件列表
// 返回按优先级排序的组件数组
func (p *PromptManager) getSortedComponents() []Component {
	comps := make([]Component, 0, len(p.components))
	for _, c := range p.components {
		comps = append(comps, c)
	}
	sort.Slice(comps, func(i, j int) bool {
		return comps[i].Priority() < comps[j].Priority()
	})
	return comps
}

// LoadPromptFile 加载提示词文件
// name: 文件名（不含扩展名）
// 返回文件内容
func (p *PromptManager) LoadPromptFile(name string) (string, error) {
	return p.loader.Load(name)
}

// SavePromptFile 保存提示词文件
// name: 文件名（不含扩展名）
// content: 文件内容
func (p *PromptManager) SavePromptFile(name, content string) error {
	return p.loader.Save(name, content)
}

// joinWithNewline 使用双换行符拼接字符串
// parts: 要拼接的字符串数组
// 返回拼接后的字符串
func joinWithNewline(parts ...string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += part
	}
	return result
}
