package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkillLoader Skill加载器
// 负责从文件系统中加载Skill
type SkillLoader struct {
	parser    *SkillParser // 解析器
	skillDirs []string     // Skill根目录列表
}

// NewSkillLoader 创建新的Skill加载器
// 参数:
//   - dirs: Skill根目录列表
//
// 返回:
//   - *SkillLoader: 加载器实例
func NewSkillLoader(dirs ...string) *SkillLoader {
	return &SkillLoader{
		parser:    NewSkillParser(),
		skillDirs: dirs,
	}
}

// AddDir 添加Skill根目录
// 参数:
//   - dir: 目录路径
func (l *SkillLoader) AddDir(dir string) {
	l.skillDirs = append(l.skillDirs, dir)
}

// LoadAll 加载所有Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 加载的Skill列表
//   - error: 加载错误
func (l *SkillLoader) LoadAll() ([]string, []*Skill, error) {
	var allSkills []*Skill
	var allDirs []string
	loaded := make(map[string]bool) // 避免重复加载

	for _, dir := range l.skillDirs {
		// 检查目录是否存在
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				// 目录不存在，跳过
				continue
			}
			return nil, nil, fmt.Errorf("访问目录失败 %s: %w", dir, err)
		}

		// 扫描Skill目录
		skillDirs, err := l.scanSkillDirs(dir)
		if err != nil {
			return nil, nil, fmt.Errorf("扫描Skill目录失败: %w", err)
		}

		// 加载每个Skill
		for _, skillDir := range skillDirs {
			// 避免重复加载同名的Skill
			if loaded[skillDir] {
				continue
			}
			loaded[skillDir] = true

			// 查找SKILL.md文件
			skillFile := filepath.Join(skillDir, "SKILL.md")
			if _, err := os.Stat(skillFile); err != nil {
				continue
			}

			// 解析Skill
			skill, err := l.parser.ParseFile(skillFile)
			if err != nil {
				return nil, nil, fmt.Errorf("解析Skill失败 %s: %w", skillFile, err)
			}

			allSkills = append(allSkills, skill)
			allDirs = append(allDirs, skillDir)
		}
	}

	return allDirs, allSkills, nil
}

// Load 加载所有Skill（返回Skill列表）
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 加载的Skill列表
//   - error: 加载错误
func (l *SkillLoader) Load() ([]*Skill, error) {
	_, skills, err := l.LoadAll()
	return skills, err
}

// LoadByName 根据名称加载指定Skill
// 参数:
//   - name: Skill名称
//
// 返回:
//   - *Skill: 加载的Skill
//   - error: 加载错误
func (l *SkillLoader) LoadByName(name string) (*Skill, error) {
	skills, err := l.Load()
	if err != nil {
		return nil, err
	}

	for _, skill := range skills {
		if skill.Name() == name {
			return skill, nil
		}
	}

	return nil, fmt.Errorf("未找到Skill: %s", name)
}

// scanSkillDirs 扫描目录下的所有Skill目录
// 参数:
//   - rootDir: 根目录
//
// 返回:
//   - []string: Skill目录列表
func (l *SkillLoader) scanSkillDirs(rootDir string) ([]string, error) {
	var dirs []string

	// 读取目录内容
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

// GetSkillDirs 获取所有Skill根目录
// 参数:
//   - 无
//
// 返回:
//   - []string: 目录列表
func (l *SkillLoader) GetSkillDirs() []string {
	return l.skillDirs
}

// Reload 重新加载所有Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 重新加载的Skill列表
//   - error: 加载错误
func (l *SkillLoader) Reload() ([]*Skill, error) {
	return l.Load()
}

// LoadFromDir 从指定目录加载Skill
// 参数:
//   - dir: Skill目录
//
// 返回:
//   - *Skill: 加载的Skill
//   - error: 加载错误
func (l *SkillLoader) LoadFromDir(dir string) (*Skill, error) {
	skillFile := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		return nil, fmt.Errorf("SKILL.md文件不存在: %w", err)
	}

	return l.parser.ParseFile(skillFile)
}

// DefaultSkillDir 获取默认的Skill目录
// 参数:
//   - 无
//
// 返回:
//   - string: 默认Skill目录路径
func DefaultSkillDir() string {
	return "./data/skills"
}

// NewDefaultLoader 创建使用默认目录的加载器
// 参数:
//   - 无
//
// 返回:
//   - *SkillLoader: 默认加载器
func NewDefaultLoader() *SkillLoader {
	return NewSkillLoader(DefaultSkillDir())
}

// ListSkillNames 列出所有已加载的Skill名称
// 参数:
//   - 无
//
// 返回:
//   - []string: Skill名称列表
func (l *SkillLoader) ListSkillNames() ([]string, error) {
	skills, err := l.Load()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(skills))
	for i, skill := range skills {
		names[i] = skill.Name()
	}

	return names, nil
}

// FindSkillsByTag 根据标签查找Skill
// 参数:
//   - tag: 标签名称
//
// 返回:
//   - []*Skill: 匹配的Skill列表
func (l *SkillLoader) FindSkillsByTag(tag string) ([]*Skill, error) {
	skills, err := l.Load()
	if err != nil {
		return nil, err
	}

	var result []*Skill
	for _, skill := range skills {
		for _, t := range skill.Tags() {
			if t == tag {
				result = append(result, skill)
				break
			}
		}
	}

	return result, nil
}

// FindUserInvocableSkills 获取所有可用户调用的Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 可用户调用的Skill列表
func (l *SkillLoader) FindUserInvocableSkills() ([]*Skill, error) {
	skills, err := l.Load()
	if err != nil {
		return nil, err
	}

	var result []*Skill
	for _, skill := range skills {
		if skill.IsUserInvocable() {
			result = append(result, skill)
		}
	}

	return result, nil
}

// FindAutoInvocableSkills 获取所有可自动触发的Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 可自动触发的Skill列表
func (l *SkillLoader) FindAutoInvocableSkills() ([]*Skill, error) {
	skills, err := l.Load()
	if err != nil {
		return nil, err
	}

	var result []*Skill
	for _, skill := range skills {
		// 可自动触发：不禁用模型调用 且 可用户调用
		if !skill.DisableModelInvocation() {
			result = append(result, skill)
		}
	}

	return result, nil
}
