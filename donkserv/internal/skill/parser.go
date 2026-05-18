package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillParser Skill解析器
// 负责解析SKILL.md文件，提取元数据和指令
// 遵循 Claude Code Agent Skills 规范
type SkillParser struct{}

// NewSkillParser 创建新的Skill解析器
// 参数:
//   - 无
//
// 返回:
//   - *SkillParser: 解析器实例
func NewSkillParser() *SkillParser {
	return &SkillParser{}
}

// Parse 解析SKILL.md文件
// 参数:
//   - filePath: SKILL.md文件路径
//   - baseDir: Skill所在目录
//
// 返回:
//   - *Skill: 解析后的Skill实例
//   - error: 解析错误
func (p *SkillParser) Parse(filePath, baseDir string) (*Skill, error) {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 分离YAML frontmatter和Markdown内容
	metadataContent, instructions, err := p.extractFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("解析frontmatter失败: %w", err)
	}

	// 解析YAML元数据
	metadata, runtime, err := p.parseMetadata(metadataContent)
	if err != nil {
		return nil, fmt.Errorf("解析元数据失败: %w", err)
	}

	// 获取目录名称
	dirName := filepath.Base(baseDir)

	// 创建Skill实例
	skill := NewSkill(dirName, metadata.Description)
	skill.SetName(dirName)
	skill.SetMetadata(metadata)
	skill.SetRuntime(runtime)
	skill.SetInstructions(instructions)
	skill.SetBaseDir(baseDir)

	// 如果frontmatter中没有name，使用目录名
	if skill.metadata.Name == "" {
		skill.metadata.Name = dirName
	}

	return skill, nil
}

// extractFrontmatter 从markdown内容中提取YAML frontmatter
// 参数:
//   - content: 完整的markdown内容
//
// 返回:
//   - string: YAML部分内容
//   - string: Markdown指令部分
//   - error: 提取错误
func (p *SkillParser) extractFrontmatter(content string) (string, string, error) {
	// 查找frontmatter分隔符
	lines := strings.Split(content, "\n")

	// 检查是否以---开头
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		// 没有frontmatter，直接返回整个内容
		return "", content, nil
	}

	// 找到结束分隔符
	var yamlLines []string
	var mdLines []string
	yamlEndFound := false

	for i, line := range lines {
		if i == 0 {
			continue // 跳过开始的---
		}

		trimmed := strings.TrimSpace(line)

		if !yamlEndFound {
			if trimmed == "---" {
				yamlEndFound = true
				continue
			}
			yamlLines = append(yamlLines, line)
		} else {
			mdLines = append(mdLines, line)
		}
	}

	yamlContent := strings.TrimSpace(strings.Join(yamlLines, "\n"))
	mdContent := strings.TrimSpace(strings.Join(mdLines, "\n"))

	return yamlContent, mdContent, nil
}

// parseMetadata 解析YAML元数据
// 参数:
//   - yamlContent: YAML内容
//
// 返回:
//   - Metadata: 元数据结构
//   - RuntimeConfig: 运行时配置
//   - error: 解析错误
func (p *SkillParser) parseMetadata(yamlContent string) (Metadata, RuntimeConfig, error) {
	if yamlContent == "" {
		return Metadata{}, RuntimeConfig{}, nil
	}

	// 使用map来解析，这样可以同时获取metadata和runtime字段
	var data map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlContent), &data)
	if err != nil {
		return Metadata{}, RuntimeConfig{}, err
	}

	// 解析metadata部分
	metadata := Metadata{
		Name:                   getString(data, "name"),
		Description:            getString(data, "description"),
		Version:                getStringDefault(data, "version", "1.0.0"),
		Author:                 getString(data, "author"),
		Homepage:               getString(data, "homepage"),
		Tags:                   getStringSlice(data, "tags"),
		ArgumentHint:           getString(data, "argument-hint"),
		DisableModelInvocation: getBool(data, "disable-model-invocation", false),
		UserInvocable:          getBool(data, "user-invocable", true),
		AllowedTools:           getStringSlice(data, "allowed-tools"),
		License:                getString(data, "license"),
		Compatibility:          getString(data, "compatibility"),
	}

	// 解析自定义metadata
	if meta, ok := data["metadata"].(map[string]interface{}); ok {
		metadata.Metadata = make(map[string]string)
		for k, v := range meta {
			if str, ok := v.(string); ok {
				metadata.Metadata[k] = str
			}
		}
	}

	// 解析runtime配置
	runtime := RuntimeConfig{
		Requires: getStringSlice(data, "requires"),
		Examples: getStringSlice(data, "examples"),
	}

	return metadata, runtime, nil
}

// ParseFile 解析指定路径的SKILL.md文件
// 参数:
//   - filePath: 文件完整路径
//
// 返回:
//   - *Skill: 解析后的Skill
//   - error: 解析错误
func (p *SkillParser) ParseFile(filePath string) (*Skill, error) {
	// 获取文件所在目录
	dir := filepath.Dir(filePath)
	return p.Parse(filePath, dir)
}

// ScanSkillDirs 扫描目录下的所有Skill目录
// 参数:
//   - rootDir: 根目录
//
// 返回:
//   - []string: Skill目录列表
func (p *SkillParser) ScanSkillDirs(rootDir string) ([]string, error) {
	var dirs []string

	// 检查目录是否存在
	info, err := os.Stat(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return dirs, nil
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s 不是目录", rootDir)
	}

	// 遍历目录
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 检查目录下是否有SKILL.md
		skillFile := filepath.Join(rootDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			dirs = append(dirs, filepath.Join(rootDir, entry.Name()))
		}
	}

	return dirs, nil
}

// 辅助函数：获取字符串值
func getString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := data[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// 辅助函数：获取字符串（带默认值）
func getStringDefault(data map[string]interface{}, key, defaultValue string) string {
	if v := getString(data, key); v != "" {
		return v
	}
	return defaultValue
}

// 辅助函数：获取字符串数组
func getStringSlice(data map[string]interface{}, key string) []string {
	if v, ok := data[key]; ok {
		if slice, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// 辅助函数：获取布尔值
func getBool(data map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := data[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}
