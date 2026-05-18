package skill

import (
	"os"
	"path/filepath"
	"time"
)

// Metadata Skill元数据
// 遵循 Claude Code / Open Agent Skills 规范
type Metadata struct {
	Name                   string            `yaml:"name"`                     // 技能名称（小写字母、数字、短横线，最多64字符）
	Description            string            `yaml:"description"`              // 技能描述及使用时机
	Version                string            `yaml:"version"`                  // 版本号
	Author                 string            `yaml:"author"`                   // 作者
	Homepage               string            `yaml:"homepage"`                 // 主页URL
	Tags                   []string          `yaml:"tags"`                     // 标签（用于分类）
	ArgumentHint           string            `yaml:"argument-hint"`            // 自动补全参数提示
	DisableModelInvocation bool              `yaml:"disable-model-invocation"` // 禁止自动触发，仅能手动调用
	UserInvocable          bool              `yaml:"user-invocable"`           // 是否可用户调用（斜杠命令）
	AllowedTools           []string          `yaml:"allowed-tools"`            // 允许使用的工具列表
	License                string            `yaml:"license"`                  // 许可证信息
	Compatibility          string            `yaml:"compatibility"`            // 环境要求
	Metadata               map[string]string `yaml:"metadata"`                 // 自定义键值对
}

// RuntimeConfig Skill运行时配置
type RuntimeConfig struct {
	Requires []string `yaml:"requires"` // 依赖的其他技能
	Examples []string `yaml:"examples"` // 使用示例
}

// SkillDirs Skill包含的目录
type SkillDirs struct {
	Scripts    string // scripts/ 目录路径
	References string // references/ 目录路径
	Assets     string // assets/ 目录路径
}

// Skill Skill核心结构
// 遵循 Claude Code Agent Skills 三层渐进式披露架构
type Skill struct {
	metadata     Metadata      // 元数据（始终加载到system prompt）
	runtime      RuntimeConfig // 运行时配置
	instructions string        // 核心指令（触发时加载）
	dirs         SkillDirs     // 子目录路径
	baseDir      string        // 技能根目录
	name         string        // 目录名称
	loaded       time.Time     // 加载时间
}

// NewSkill 创建新的Skill实例
// 参数:
//   - name: 技能目录名称
//   - description: 技能描述
//
// 返回:
//   - *Skill: Skill实例
func NewSkill(name, description string) *Skill {
	return &Skill{
		name: name,
		metadata: Metadata{
			Name:                   name,
			Description:            description,
			Version:                "1.0.0",
			UserInvocable:          true,
			DisableModelInvocation: false,
		},
		runtime: RuntimeConfig{},
		dirs:    SkillDirs{},
		loaded:  time.Now(),
	}
}

// Name 获取技能名称
// 参数:
//   - 无
//
// 返回:
//   - string: 技能名称
func (s *Skill) Name() string {
	if s.metadata.Name != "" {
		return s.metadata.Name
	}
	return s.name
}

// Description 获取技能描述
// 参数:
//   - 无
//
// 返回:
//   - string: 技能描述
func (s *Skill) Description() string {
	return s.metadata.Description
}

// Version 获取版本号
// 参数:
//   - 无
//
// 返回:
//   - string: 版本号
func (s *Skill) Version() string {
	return s.metadata.Version
}

// Author 获取作者
// 参数:
//   - 无
//
// 返回:
//   - string: 作者
func (s *Skill) Author() string {
	return s.metadata.Author
}

// Homepage 获取主页URL
// 参数:
//   - 无
//
// 返回:
//   - string: 主页URL
func (s *Skill) Homepage() string {
	return s.metadata.Homepage
}

// Tags 获取标签列表
// 参数:
//   - 无
//
// 返回:
//   - []string: 标签列表
func (s *Skill) Tags() []string {
	return s.metadata.Tags
}

// ArgumentHint 获取参数提示
// 参数:
//   - 无
//
// 返回:
//   - string: 参数提示
func (s *Skill) ArgumentHint() string {
	return s.metadata.ArgumentHint
}

// IsUserInvocable 是否可用户调用
// 参数:
//   - 无
//
// 返回:
//   - bool: 是否可用户调用
func (s *Skill) IsUserInvocable() bool {
	return s.metadata.UserInvocable
}

// DisableModelInvocation 是否禁用自动触发
// 参数:
//   - 无
//
// 返回:
//   - bool: 是否禁用自动触发
func (s *Skill) DisableModelInvocation() bool {
	return s.metadata.DisableModelInvocation
}

// AllowedTools 获取允许的工具列表
// 参数:
//   - 无
//
// 返回:
//   - []string: 允许的工具列表
func (s *Skill) AllowedTools() []string {
	return s.metadata.AllowedTools
}

// License 获取许可证信息
// 参数:
//   - 无
//
// 返回:
//   - string: 许可证信息
func (s *Skill) License() string {
	return s.metadata.License
}

// Compatibility 获取环境要求
// 参数:
//   - 无
//
// 返回:
//   - string: 环境要求
func (s *Skill) Compatibility() string {
	return s.metadata.Compatibility
}

// CustomMetadata 获取自定义元数据
// 参数:
//   - 无
//
// 返回:
//   - map[string]string: 自定义键值对
func (s *Skill) CustomMetadata() map[string]string {
	return s.metadata.Metadata
}

// GetCustomMetadata 获取指定的自定义元数据值
// 参数:
//   - key: 键名
//
// 返回:
//   - string: 元数据值，如果不存在则返回空字符串
func (s *Skill) GetCustomMetadata(key string) string {
	if s.metadata.Metadata == nil {
		return ""
	}
	return s.metadata.Metadata[key]
}

// Requires 获取依赖列表
// 参数:
//   - 无
//
// 返回:
//   - []string: 依赖的技能列表
func (s *Skill) Requires() []string {
	return s.runtime.Requires
}

// Examples 获取使用示例
// 参数:
//   - 无
//
// 返回:
//   - []string: 使用示例列表
func (s *Skill) Examples() []string {
	return s.runtime.Examples
}

// Instructions 获取核心指令
// 参数:
//   - 无
//
// 返回:
//   - string: 核心指令内容
func (s *Skill) Instructions() string {
	return s.instructions
}

// BaseDir 获取技能根目录
// 参数:
//   - 无
//
// 返回:
//   - string: 根目录路径
func (s *Skill) BaseDir() string {
	return s.baseDir
}

// Dirs 获取子目录信息
// 参数:
//   - 无
//
// 返回:
//   - SkillDirs: 子目录路径
func (s *Skill) Dirs() SkillDirs {
	return s.dirs
}

// Loaded 获取加载时间
// 参数:
//   - 无
//
// 返回:
//   - time.Time: 加载时间
func (s *Skill) Loaded() time.Time {
	return s.loaded
}

// SetMetadata 设置元数据
// 参数:
//   - m: Metadata实例
func (s *Skill) SetMetadata(m Metadata) {
	s.metadata = m
}

// SetRuntime 设置运行时配置
// 参数:
//   - r: RuntimeConfig实例
func (s *Skill) SetRuntime(r RuntimeConfig) {
	s.runtime = r
}

// SetInstructions 设置核心指令
// 参数:
//   - instructions: 指令内容
func (s *Skill) SetInstructions(instructions string) {
	s.instructions = instructions
}

// SetBaseDir 设置根目录
// 参数:
//   - baseDir: 根目录路径
func (s *Skill) SetBaseDir(baseDir string) {
	s.baseDir = baseDir
	s.dirs = SkillDirs{
		Scripts:    filepath.Join(baseDir, "scripts"),
		References: filepath.Join(baseDir, "references"),
		Assets:     filepath.Join(baseDir, "assets"),
	}
}

// SetName 设置目录名称
// 参数:
//   - name: 目录名称
func (s *Skill) SetName(name string) {
	s.name = name
}

// GetDirPath 获取子目录路径
// 参数:
//   - dirType: 目录类型 (scripts, references, assets)
//
// 返回:
//   - string: 目录路径
func (s *Skill) GetDirPath(dirType string) string {
	switch dirType {
	case "scripts":
		return s.dirs.Scripts
	case "references":
		return s.dirs.References
	case "assets":
		return s.dirs.Assets
	default:
		return ""
	}
}

// HasScripts 检查是否有scripts目录
// 参数:
//   - 无
//
// 返回:
//   - bool: 是否存在
func (s *Skill) HasScripts() bool {
	info, err := os.Stat(s.dirs.Scripts)
	return err == nil && info.IsDir()
}

// HasReferences 检查是否有references目录
// 参数:
//   - 无
//
// 返回:
//   - bool: 是否存在
func (s *Skill) HasReferences() bool {
	info, err := os.Stat(s.dirs.References)
	return err == nil && info.IsDir()
}

// HasAssets 检查是否有assets目录
// 参数:
//   - 无
//
// 返回:
//   - bool: 是否存在
func (s *Skill) HasAssets() bool {
	info, err := os.Stat(s.dirs.Assets)
	return err == nil && info.IsDir()
}

// ListScripts 列出scripts目录下的文件
// 参数:
//   - 无
//
// 返回:
//   - []string: 文件路径列表
func (s *Skill) ListScripts() ([]string, error) {
	return s.listDirFiles("scripts")
}

// ListReferences 列出references目录下的文件
// 参数:
//   - 无
//
// 返回:
//   - []string: 文件路径列表
func (s *Skill) ListReferences() ([]string, error) {
	return s.listDirFiles("references")
}

// ListAssets 列出assets目录下的文件
// 参数:
//   - 无
//
// 返回:
//   - []string: 文件路径列表
func (s *Skill) ListAssets() ([]string, error) {
	return s.listDirFiles("assets")
}

// listDirFiles 列出指定目录下的所有文件
// 参数:
//   - dirType: 目录类型
//
// 返回:
//   - []string: 文件路径列表
//   - error: 错误信息
func (s *Skill) listDirFiles(dirType string) ([]string, error) {
	dirPath := s.GetDirPath(dirType)
	if dirPath == "" {
		return nil, nil
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, nil
	}

	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dirPath, entry.Name()))
		}
	}

	return files, nil
}

// GetDirName 获取目录名称
// 参数:
//   - 无
//
// 返回:
//   - string: 目录名称
func (s *Skill) GetDirName() string {
	return s.name
}
