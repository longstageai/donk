// agents 技能自动发现 Agent 模块
// Creator Agent 负责创建技能文件
package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/pkg/logger"
	"gopkg.in/yaml.v3"
)

// CreatorAgent 技能创建 Agent
// 负责将技能规划转换为实际的技能文件
type CreatorAgent struct {
	skillsDir string
}

// NewCreatorAgent 创建技能创建 Agent
// 参数:
//   - skillsDir: 技能存储目录
//
// 返回:
//   - *CreatorAgent: Agent 实例
func NewCreatorAgent(skillsDir string) *CreatorAgent {
	return &CreatorAgent{
		skillsDir: skillsDir,
	}
}

// Create 创建技能
// 参数:
//   - ctx: 上下文
//   - plan: 技能规划
//
// 返回:
//   - *skill.Skill: 创建的技能
//   - error: 错误信息
func (c *CreatorAgent) Create(ctx context.Context, plan *SkillPlan) (*skill.Skill, error) {
	logger.Info("开始创建技能", map[string]interface{}{
		"skill_name": plan.Name,
	})

	// 创建技能目录
	skillDir := filepath.Join(c.skillsDir, plan.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		logger.Error("创建技能目录失败", map[string]interface{}{
			"skill_name": plan.Name,
			"skill_dir":  skillDir,
			"error":      err.Error(),
		})
		return nil, fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 创建子目录
	subDirs := []string{"scripts", "references", "assets"}
	for _, dir := range subDirs {
		subDir := filepath.Join(skillDir, dir)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			logger.Error("创建子目录失败", map[string]interface{}{
				"skill_name": plan.Name,
				"sub_dir":    subDir,
				"error":      err.Error(),
			})
			return nil, fmt.Errorf("创建子目录失败: %w", err)
		}
	}

	// 创建 SKILL.md 文件
	if err := c.createSkillMD(skillDir, plan); err != nil {
		logger.Error("创建 SKILL.md 失败", map[string]interface{}{
			"skill_name": plan.Name,
			"error":      err.Error(),
		})
		return nil, fmt.Errorf("创建 SKILL.md 失败: %w", err)
	}

	logger.Info("技能创建完成", map[string]interface{}{
		"skill_name": plan.Name,
		"skill_dir":  skillDir,
	})

	// 加载并返回技能
	loader := skill.NewSkillLoader(c.skillsDir)
	s, err := loader.LoadByName(plan.Name)
	if err != nil {
		logger.Error("加载创建的技能失败", map[string]interface{}{
			"skill_name": plan.Name,
			"error":      err.Error(),
		})
		return nil, fmt.Errorf("加载创建的技能失败: %w", err)
	}

	return s, nil
}

// createSkillMD 创建 SKILL.md 文件
// SKILL.md 包含 YAML frontmatter 和 Markdown 指令内容
// 参数:
//   - skillDir: 技能目录
//   - plan: 技能规划
//
// 返回:
//   - error: 错误信息
func (c *CreatorAgent) createSkillMD(skillDir string, plan *SkillPlan) error {
	// 构建 YAML frontmatter
	frontmatter := map[string]interface{}{
		"name":        plan.Name,
		"description": plan.Description,
		"version":     "1.0.0",
		"author":      "auto-discovery",
		"tags":        plan.Tags,
		"runtime": map[string]interface{}{
			"user-invocable":           true,
			"disable-model-invocation": false,
		},
	}

	if len(plan.AllowedTools) > 0 {
		frontmatter["allowed-tools"] = plan.AllowedTools
	}

	if len(plan.Examples) > 0 {
		frontmatter["examples"] = plan.Examples
	}

	if len(plan.Metadata) > 0 {
		// 合并 metadata 到 frontmatter
		for k, v := range plan.Metadata {
			if _, exists := frontmatter[k]; !exists {
				frontmatter[k] = v
			}
		}
	}

	// 序列化 YAML
	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("YAML 序列化失败: %w", err)
	}

	// 构建完整的 SKILL.md 内容
	var content strings.Builder

	// YAML frontmatter
	content.WriteString("---\n")
	content.WriteString(string(yamlData))
	content.WriteString("---\n\n")

	// 标题和描述
	content.WriteString(fmt.Sprintf("# %s\n\n", plan.Name))
	content.WriteString(fmt.Sprintf("%s\n\n", plan.Description))

	// 使用示例
	if len(plan.Examples) > 0 {
		content.WriteString("## 使用示例\n\n")
		for i, example := range plan.Examples {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, example))
		}
		content.WriteString("\n")
	}

	// 指令内容
	content.WriteString("## 执行流程\n\n")
	content.WriteString(plan.Instructions)

	// 写入文件
	skillFilePath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFilePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("写入 SKILL.md 失败: %w", err)
	}

	logger.Debug("创建 SKILL.md 成功", map[string]interface{}{
		"skill_name":    plan.Name,
		"skill_md_path": skillFilePath,
	})

	return nil
}
