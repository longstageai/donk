package skill

import (
	"fmt"
	"sync"

	"github.com/longstageai/donk/donk/internal/tool"
)

// SkillRegistry Skill注册表
// 负责管理所有已加载的Skill
// 遵循 Claude Code Agent Skills 规范
type SkillRegistry struct {
	skills map[string]*Skill // Skill名称到实例的映射
	mu     sync.RWMutex      // 读写锁
	loader *SkillLoader      // 加载器引用
}

// NewSkillRegistry 创建新的Skill注册表
// 参数:
//   - 无
//
// 返回:
//   - *SkillRegistry: 注册表实例
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]*Skill),
		loader: nil,
	}
}

// NewSkillRegistryWithLoader 创建带加载器的注册表
// 参数:
//   - loader: Skill加载器
//
// 返回:
//   - *SkillRegistry: 注册表实例
func NewSkillRegistryWithLoader(loader *SkillLoader) *SkillRegistry {
	registry := NewSkillRegistry()
	registry.loader = loader
	return registry
}

// Register 注册一个Skill
// 参数:
//   - skill: Skill实例
func (r *SkillRegistry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查Skill是否已存在
	if _, exists := r.skills[skill.Name()]; exists {
		return fmt.Errorf("Skill已存在: %s", skill.Name())
	}

	// 注册Skill
	r.skills[skill.Name()] = skill

	return nil
}

// Unregister 注销一个Skill
// 参数:
//   - name: Skill名称
func (r *SkillRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.skills[name]
	if !exists {
		return fmt.Errorf("Skill不存在: %s", name)
	}

	// 注销Skill
	delete(r.skills, name)

	return nil
}

// Get 获取指定名称的Skill
// 参数:
//   - name: Skill名称
//
// 返回:
//   - *Skill: Skill实例
//   - error: 获取错误
func (r *SkillRegistry) Get(name string) (*Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("Skill不存在: %s", name)
	}

	return skill, nil
}

// List 列出所有Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: Skill列表
func (r *SkillRegistry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}

	return skills
}

// Count 获取已注册的Skill数量
// 参数:
//   - 无
//
// 返回:
//   - int: Skill数量
func (r *SkillRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.skills)
}

// LoadAndRegister 加载并注册所有Skill
// 参数:
//   - 无
//
// 返回:
//   - error: 加载错误
func (r *SkillRegistry) LoadAndRegister() error {
	if r.loader == nil {
		return fmt.Errorf("未设置加载器")
	}

	skills, err := r.loader.Load()
	if err != nil {
		return err
	}

	for _, skill := range skills {
		if err := r.Register(skill); err != nil {
			return err
		}
	}

	return nil
}

// SetLoader 设置加载器
// 参数:
//   - loader: Skill加载器
func (r *SkillRegistry) SetLoader(loader *SkillLoader) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.loader = loader
}

// GetLoader 获取加载器
// 参数:
//   - 无
//
// 返回:
//   - *SkillLoader: 加载器
func (r *SkillRegistry) GetLoader() *SkillLoader {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.loader
}

// Reload 重新加载所有Skill
// 参数:
//   - 无
//
// 返回:
//   - error: 加载错误
func (r *SkillRegistry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空现有注册
	r.skills = make(map[string]*Skill)

	if r.loader == nil {
		return fmt.Errorf("未设置加载器")
	}

	// 重新加载
	skills, err := r.loader.Load()
	if err != nil {
		return err
	}

	for _, skill := range skills {
		r.skills[skill.Name()] = skill
	}

	return nil
}

// FindByTag 根据标签查找Skill
// 参数:
//   - tag: 标签名称
//
// 返回:
//   - []*Skill: 匹配的Skill列表
func (r *SkillRegistry) FindByTag(tag string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Skill
	for _, skill := range r.skills {
		for _, t := range skill.Tags() {
			if t == tag {
				result = append(result, skill)
				break
			}
		}
	}

	return result
}

// GetUserInvocableSkills 获取所有可用户调用的Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 可用户调用的Skill列表
func (r *SkillRegistry) GetUserInvocableSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Skill
	for _, skill := range r.skills {
		if skill.IsUserInvocable() {
			result = append(result, skill)
		}
	}

	return result
}

// GetAutoInvocableSkills 获取所有可自动触发的Skill
// 参数:
//   - 无
//
// 返回:
//   - []*Skill: 可自动触发的Skill列表
func (r *SkillRegistry) GetAutoInvocableSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Skill
	for _, skill := range r.skills {
		// 可自动触发：不禁用模型调用
		if !skill.DisableModelInvocation() {
			result = append(result, skill)
		}
	}

	return result
}

// HasSkill 检查Skill是否存在
// 参数:
//   - name: Skill名称
//
// 返回:
//   - bool: 是否存在
func (r *SkillRegistry) HasSkill(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.skills[name]
	return exists
}

// GetSkillMetadata 获取所有Skill的元数据
// 用于 Level 1：元数据加载
// 参数:
//   - 无
//
// 返回:
//   - []map[string]string: 元数据列表
func (r *SkillRegistry) GetSkillMetadata() []map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]map[string]string, 0, len(r.skills))
	for _, skill := range r.skills {
		result = append(result, map[string]string{
			"name":        skill.Name(),
			"description": skill.Description(),
		})
	}

	return result
}

// GetSkillToolRegistry 获取Skill关联的工具注册表
// 参数:
//   - 无
//
// 返回:
//   - *tool.Registry: 工具注册表
func (r *SkillRegistry) GetSkillToolRegistry() *tool.Registry {
	// 这里返回nil，实际使用时需要从外部传入工具注册表
	return nil
}
