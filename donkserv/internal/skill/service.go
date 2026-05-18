package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkillInfo Skill 信息（用于列表和详情展示）
type SkillInfo struct {
	Name                   string   `json:"name"`                     // Skill 名称
	Description            string   `json:"description"`              // 描述
	Version                string   `json:"version"`                  // 版本
	Author                 string   `json:"author"`                   // 作者
	Tags                   []string `json:"tags"`                     // 标签列表
	Enabled                bool     `json:"enabled"`                  // 是否启用
	UserInvocable          bool     `json:"user_invocable"`           // 是否允许用户调用
	DisableModelInvocation bool     `json:"disable_model_invocation"` // 是否禁用自动触发
	Path                   string   `json:"path"`                     // 文件路径
	HasScripts             bool     `json:"has_scripts"`              // 是否有脚本目录
	HasReferences          bool     `json:"has_references"`           // 是否有参考资料目录
	HasAssets              bool     `json:"has_assets"`               // 是否有资源目录
}

// Service Skill 业务逻辑层
type Service struct {
	repo     *StateRepository
	loader   *SkillLoader
	registry *SkillRegistry
}

// NewService 创建 Skill 服务实例
// 参数:
//   - repo: 状态仓库
//   - loader: Skill 加载器
//   - registry: Skill 注册表
//
// 返回:
//   - *Service: 服务实例
func NewService(repo *StateRepository, loader *SkillLoader, registry *SkillRegistry) *Service {
	return &Service{
		repo:     repo,
		loader:   loader,
		registry: registry,
	}
}

// List 获取所有 Skill 列表
// 从数据库获取状态，从文件系统获取完整信息
// 参数:
//   - 无
//
// 返回:
//   - []*SkillInfo: Skill 信息列表
//   - error: 查询错误
func (s *Service) List() ([]*SkillInfo, error) {
	// 获取所有状态
	states, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	// 构建名称到状态的映射
	stateMap := make(map[string]*SkillState)
	for _, state := range states {
		stateMap[state.Name] = state
	}

	// 从文件系统加载所有 Skill
	skills, err := s.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("加载 Skill 失败: %w", err)
	}

	// 合并信息
	infos := make([]*SkillInfo, 0)
	for _, skill := range skills {
		state, exists := stateMap[skill.Name()]
		if !exists {
			// 数据库中没有，使用默认值（启用）
			state = &SkillState{
				Name:        skill.Name(),
				Description: skill.Description(),
				Enabled:     true,
			}
		}

		info := s.buildSkillInfo(skill, state)
		infos = append(infos, info)
	}

	return infos, nil
}

// Get 获取指定 Skill 详情
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - *SkillInfo: Skill 信息
//   - error: 查询错误
func (s *Service) Get(name string) (*SkillInfo, error) {
	// 获取状态
	state, err := s.repo.Get(name)
	if err != nil {
		return nil, err
	}

	// 从文件系统加载
	skill, err := s.loader.LoadByName(name)
	if err != nil {
		return nil, fmt.Errorf("加载 Skill 失败: %w", err)
	}

	if state == nil {
		state = &SkillState{
			Name:        skill.Name(),
			Description: skill.Description(),
			Enabled:     true,
		}
	}

	return s.buildSkillInfo(skill, state), nil
}

// GetInstructions 获取 Skill 完整指令
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - string: 指令内容
//   - error: 查询错误
func (s *Service) GetInstructions(name string) (string, error) {
	skill, err := s.loader.LoadByName(name)
	if err != nil {
		return "", fmt.Errorf("加载 Skill 失败: %w", err)
	}
	return skill.Instructions(), nil
}

// Enable 启用 Skill
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - error: 操作错误
func (s *Service) Enable(name string) error {
	// 检查 Skill 是否存在
	if _, err := s.loader.LoadByName(name); err != nil {
		return fmt.Errorf("Skill 不存在: %w", err)
	}

	// 更新数据库状态
	if err := s.repo.UpdateEnabled(name, true); err != nil {
		return err
	}

	// 重新加载到注册表
	skill, _ := s.loader.LoadByName(name)
	_ = s.registry.Register(skill)

	return nil
}

// Disable 禁用 Skill
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - error: 操作错误
func (s *Service) Disable(name string) error {
	// 检查 Skill 是否存在
	if _, err := s.loader.LoadByName(name); err != nil {
		return fmt.Errorf("Skill 不存在: %w", err)
	}

	// 更新数据库状态
	if err := s.repo.UpdateEnabled(name, false); err != nil {
		return err
	}

	// 从注册表注销
	_ = s.registry.Unregister(name)

	return nil
}

// Delete 删除 Skill
// 删除文件系统目录和数据库记录
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - error: 操作错误
func (s *Service) Delete(name string) error {
	// 获取 Skill
	skill, err := s.loader.LoadByName(name)
	if err != nil {
		return fmt.Errorf("Skill 不存在: %w", err)
	}

	// 删除文件系统目录
	if err := os.RemoveAll(skill.BaseDir()); err != nil {
		return fmt.Errorf("删除 Skill 目录失败: %w", err)
	}

	// 从注册表注销
	_ = s.registry.Unregister(name)

	// 删除数据库记录
	if err := s.repo.Delete(name); err != nil {
		return fmt.Errorf("删除 Skill 状态记录失败: %w", err)
	}

	return nil
}

// Rescan 重新扫描文件系统
// 将新 Skill 同步到数据库
// 参数:
//   - 无
//
// 返回:
//   - error: 同步错误
func (s *Service) Rescan() error {
	return s.repo.SyncFromLoader(s.loader)
}

// buildSkillInfo 构建 Skill 信息
func (s *Service) buildSkillInfo(skill *Skill, state *SkillState) *SkillInfo {
	return &SkillInfo{
		Name:                   skill.Name(),
		Description:            skill.Description(),
		Version:                skill.Version(),
		Author:                 skill.Author(),
		Tags:                   skill.Tags(),
		Enabled:                state.Enabled,
		UserInvocable:          skill.IsUserInvocable(),
		DisableModelInvocation: skill.DisableModelInvocation(),
		Path:                   skill.BaseDir(),
		HasScripts:             skill.HasScripts(),
		HasReferences:          skill.HasReferences(),
		HasAssets:              skill.HasAssets(),
	}
}

// GetScriptContent 获取脚本内容
// 参数:
//   - skillName: Skill 名称
//   - scriptName: 脚本名称
//
// 返回:
//   - string: 脚本内容
//   - error: 读取错误
func (s *Service) GetScriptContent(skillName, scriptName string) (string, error) {
	skill, err := s.loader.LoadByName(skillName)
	if err != nil {
		return "", fmt.Errorf("Skill 不存在: %w", err)
	}

	scripts, err := skill.ListScripts()
	if err != nil {
		return "", fmt.Errorf("列出脚本失败: %w", err)
	}

	// 查找匹配的脚本
	for _, script := range scripts {
		if filepath.Base(script) == scriptName {
			content, err := os.ReadFile(script)
			if err != nil {
				return "", fmt.Errorf("读取脚本失败: %w", err)
			}
			return string(content), nil
		}
	}

	return "", fmt.Errorf("脚本不存在: %s", scriptName)
}

// ListScripts 获取 Skill 的脚本列表
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - []string: 脚本文件名列表
//   - error: 查询错误
func (s *Service) ListScripts(name string) ([]string, error) {
	skill, err := s.loader.LoadByName(name)
	if err != nil {
		return nil, fmt.Errorf("Skill 不存在: %w", err)
	}

	scripts, err := skill.ListScripts()
	if err != nil {
		return nil, err
	}

	// 只返回文件名
	var names []string
	for _, script := range scripts {
		names = append(names, filepath.Base(script))
	}
	return names, nil
}
