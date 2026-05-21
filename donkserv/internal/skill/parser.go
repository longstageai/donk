package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

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

	if err := validateParsedSkillMetadata(metadata, dirName); err != nil {
		return nil, err
	}

	return skill, nil
}

func validateParsedSkillMetadata(metadata Metadata, dirName string) error {
	if metadata.Name == "" {
		return fmt.Errorf("SKILL.md frontmatter缺少必需字段name")
	}
	if metadata.Description == "" {
		return fmt.Errorf("SKILL.md frontmatter缺少必需字段description")
	}
	if !isValidSkillName(metadata.Name) {
		return fmt.Errorf("Skill name不符合Open Agent Skills规范: %s", metadata.Name)
	}
	if metadata.Name != dirName {
		return fmt.Errorf("Skill name必须与父目录名称一致: name=%s dir=%s", metadata.Name, dirName)
	}
	if len([]rune(metadata.Description)) > 1024 {
		return fmt.Errorf("Skill description不能超过1024字符: %s", metadata.Name)
	}
	if metadata.Compatibility != "" && len([]rune(metadata.Compatibility)) > 500 {
		return fmt.Errorf("Skill compatibility不能超过500字符: %s", metadata.Name)
	}
	return nil
}

func isValidSkillName(name string) bool {
	return len([]rune(name)) >= 1 && len([]rune(name)) <= 64 && skillNamePattern.MatchString(name) && !strings.HasSuffix(name, "-") && !strings.Contains(name, "--")
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
		return "", "", fmt.Errorf("SKILL.md必须包含YAML frontmatter")
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

	if !yamlEndFound {
		return "", "", fmt.Errorf("frontmatter缺少结束分隔符---")
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
		Version:                getString(data, "version"),
		Author:                 getString(data, "author"),
		Homepage:               getString(data, "homepage"),
		Tags:                   getStringSlice(data, "tags"),
		ArgumentHint:           getString(data, "argument-hint"),
		DisableModelInvocation: getBool(data, "disable-model-invocation", false),
		UserInvocable:          getBool(data, "user-invocable", true),
		AllowedTools:           getSpaceSeparatedStringSlice(data, "allowed-tools"),
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
	metadata.Version = metadataValueDefault(metadata.Metadata, "version", metadata.Version)
	metadata.Author = metadataValueDefault(metadata.Metadata, "author", metadata.Author)
	metadata.Homepage = metadataValueDefault(metadata.Metadata, "homepage", metadata.Homepage)

	// 解析runtime配置
	runtime := RuntimeConfig{
		Requires:           getStringSlice(data, "requires"),
		Examples:           getStringSlice(data, "examples"),
		ScriptDependencies: getStringSliceMap(data, "script-dependencies"),
		Scripts:            getScriptRuntimeConfigMap(data, "scripts"),
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

func metadataValueDefault(metadata map[string]string, key, defaultValue string) string {
	if metadata == nil {
		return defaultValue
	}
	if value := metadata[key]; value != "" {
		return value
	}
	return defaultValue
}

// 辅助函数：获取字符串数组
func getStringSlice(data map[string]interface{}, key string) []string {
	if v, ok := data[key]; ok {
		return toStringSlice(v)
	}
	return nil
}

func getSpaceSeparatedStringSlice(data map[string]interface{}, key string) []string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return strings.Fields(s)
		}
		return toStringSlice(v)
	}
	return nil
}

func toStringSlice(value interface{}) []string {
	slice, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func getStringSliceMap(data map[string]interface{}, key string) map[string][]string {
	value, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}
	result := map[string][]string{}
	for k, v := range value {
		items := toStringSlice(v)
		if len(items) > 0 {
			result[k] = items
		}
	}
	return result
}

func getScriptRuntimeConfigMap(data map[string]interface{}, key string) map[string]ScriptRuntimeConfig {
	value, ok := data[key].(map[string]interface{})
	if !ok {
		return nil
	}
	result := map[string]ScriptRuntimeConfig{}
	for name, raw := range value {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		config := ScriptRuntimeConfig{
			Language:         getString(item, "language"),
			Dependencies:     getStringSlice(item, "dependencies"),
			DependencyPolicy: getStringDefault(item, "dependency-policy", ""),
			RuntimeVersion:   getString(item, "runtime-version"),
		}
		result[name] = config
	}
	return result
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
