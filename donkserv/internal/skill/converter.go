package skill

import (
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/prompt"
)

// SkillConverter Skill到其他组件的转换器
// 负责将Skill转换为提示词组件、工具定义等
type SkillConverter struct{}

// NewSkillConverter 创建新的Skill转换器
// 参数:
//   - 无
//
// 返回:
//   - *SkillConverter: 转换器实例
func NewSkillConverter() *SkillConverter {
	return &SkillConverter{}
}

// ToPromptComponent 将Skill转换为提示词组件
// 参数:
//   - skill: Skill实例
//
// 返回:
//   - prompt.Component: 提示词组件
func (c *SkillConverter) ToPromptComponent(skill *Skill) prompt.Component {
	return &skillPromptComponent{
		skill: skill,
	}
}

// ToPromptComponents 将多个Skill转换为提示词组件
// 参数:
//   - skills: Skill列表
//
// 返回:
//   - []prompt.Component: 提示词组件列表
func (c *SkillConverter) ToPromptComponents(skills []*Skill) []prompt.Component {
	components := make([]prompt.Component, len(skills))
	for i, skill := range skills {
		components[i] = c.ToPromptComponent(skill)
	}
	return components
}

// ConvertToInstructions 将Skill转换为Agent可用的指令格式
// 参数:
//   - skill: Skill实例
//
// 返回:
//   - string: 转换后的指令
func (c *SkillConverter) ConvertToInstructions(skill *Skill) string {
	var sb strings.Builder

	// 添加Skill名称和描述
	sb.WriteString(fmt.Sprintf("## %s\n\n", skill.Name()))
	sb.WriteString(fmt.Sprintf("%s\n\n", skill.Description()))

	// 添加使用示例（如果有）
	if len(skill.Examples()) > 0 {
		sb.WriteString("**使用示例**:\n")
		for _, example := range skill.Examples() {
			sb.WriteString(fmt.Sprintf("- %s\n", example))
		}
		sb.WriteString("\n")
	}

	// 添加指令内容
	sb.WriteString("**指令**:\n")
	sb.WriteString(skill.Instructions())

	return sb.String()
}

// ConvertAllToInstructions 将所有Skill转换为合并的指令
// 参数:
//   - skills: Skill列表
//
// 返回:
//   - string: 合并后的指令
func (c *SkillConverter) ConvertAllToInstructions(skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 可用技能\n\n")

	for _, skill := range skills {
		sb.WriteString(c.ConvertToInstructions(skill))
		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

// skillPromptComponent Skill提示词组件
// 实现prompt.Component接口
type skillPromptComponent struct {
	skill *Skill
}

// Name 返回组件名称
// 参数:
//   - 无
//
// 返回:
//   - string: 组件名称
func (c *skillPromptComponent) Name() string {
	return "skill_" + c.skill.Name()
}

// Priority 返回组件优先级
// 参数:
//   - 无
//
// 返回:
//   - int: 优先级
func (c *skillPromptComponent) Priority() int {
	return 100 // 在系统组件之后
}

// Content 获取组件内容
// 参数:
//   - vars: 变量映射
//
// 返回:
//   - string: 组件内容
//   - error: 错误信息
func (c *skillPromptComponent) Content(vars prompt.Variables) (string, error) {
	instructions := c.skill.Instructions()

	// 替换变量
	for key, value := range vars {
		placeholder := fmt.Sprintf("{%s}", key)
		instructions = strings.ReplaceAll(instructions, placeholder, fmt.Sprintf("%v", value))
	}

	// 替换{baseDir}
	instructions = strings.ReplaceAll(instructions, "{baseDir}", c.skill.BaseDir())

	// 构建输出
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 技能: %s\n\n", c.skill.Name()))
	sb.WriteString(fmt.Sprintf("%s\n\n", c.skill.Description()))

	// 添加详细指令
	sb.WriteString(instructions)

	return sb.String(), nil
}

// GenerateSystemPromptMetadata 生成用于系统提示的Skill元数据
// 遵循 Claude Code 规范：仅包含 name 和 description
// 参数:
//   - skills: Skill列表
//
// 返回:
//   - string: 元数据提示词
func (c *SkillConverter) GenerateSystemPromptMetadata(skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## 可用技能\n\n")
	sb.WriteString("你可以通过以下技能来帮助用户完成任务：\n\n")

	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", skill.Name(), skill.Description()))
	}

	return sb.String()
}

// GenerateFullSystemPrompt 生成完整的系统提示词
// 包含所有Skill的元数据和指令
// 参数:
//   - skills: Skill列表
//
// 返回:
//   - string: 系统提示词
func (c *SkillConverter) GenerateFullSystemPrompt(skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("## 可用技能\n\n")
	sb.WriteString("你可以通过以下技能来帮助用户完成任务：\n\n")

	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("### %s\n", skill.Name()))
		sb.WriteString(fmt.Sprintf("%s\n", skill.Description()))

		if len(skill.Examples()) > 0 {
			sb.WriteString("**示例**:\n")
			for _, ex := range skill.Examples() {
				sb.WriteString(fmt.Sprintf("- %s\n", ex))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// GetSkillInstructionsForAgent 获取适合Agent理解的Skill指令
// 参数:
//   - skill: Skill实例
//   - includeMetadata: 是否包含元数据
//
// 返回:
//   - string: 格式化后的指令
func (c *SkillConverter) GetSkillInstructionsForAgent(skill *Skill, includeMetadata bool) string {
	var sb strings.Builder

	if includeMetadata {
		sb.WriteString(fmt.Sprintf("### 技能: %s\n", skill.Name()))
		sb.WriteString(fmt.Sprintf("版本: %s | 作者: %s\n", skill.Version(), skill.Author()))
		sb.WriteString(fmt.Sprintf("描述: %s\n\n", skill.Description()))
	} else {
		sb.WriteString(fmt.Sprintf("## %s\n\n", skill.Name()))
		sb.WriteString(fmt.Sprintf("%s\n\n", skill.Description()))
	}

	// 添加触发条件
	sb.WriteString("**使用场景**: ")
	sb.WriteString(skill.Description())
	sb.WriteString("\n\n")

	// 添加详细指令
	sb.WriteString("**使用说明**:\n")
	sb.WriteString(skill.Instructions())

	return sb.String()
}

// ToSkillSummary 将Skill转换为摘要格式
// 用于Level 1：元数据加载
// 参数:
//   - skill: Skill实例
//
// 返回:
//   - map[string]string: 摘要映射
func (c *SkillConverter) ToSkillSummary(skill *Skill) map[string]string {
	return map[string]string{
		"name":        skill.Name(),
		"description": skill.Description(),
	}
}

// ToSkillSummaries 将多个Skill转换为摘要列表
// 参数:
//   - skills: Skill列表
//
// 返回:
//   - []map[string]string: 摘要列表
func (c *SkillConverter) ToSkillSummaries(skills []*Skill) []map[string]string {
	summaries := make([]map[string]string, len(skills))
	for i, skill := range skills {
		summaries[i] = c.ToSkillSummary(skill)
	}
	return summaries
}

// GetAllowedTools 获取Skill允许的工具列表
// 参数:
//   - skill: Skill实例
//
// 返回:
//   - []string: 允许的工具列表
func (c *SkillConverter) GetAllowedTools(skill *Skill) []string {
	return skill.AllowedTools()
}
