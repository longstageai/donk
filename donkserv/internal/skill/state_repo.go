package skill

import (
	"database/sql"
	"fmt"
	"time"
)

// SkillState Skill 状态记录
type SkillState struct {
	Name        string    `json:"name" db:"name"`               // Skill 名称
	Description string    `json:"description" db:"description"` // Skill 描述
	Enabled     bool      `json:"enabled" db:"enabled"`         // 是否启用
	CreatedAt   time.Time `json:"created_at" db:"created_at"`   // 创建时间
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`   // 更新时间
}

// StateRepository Skill 状态数据访问层
type StateRepository struct {
	db *sql.DB
}

// NewStateRepository 创建状态仓库实例
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - *StateRepository: 仓库实例
func NewStateRepository(db *sql.DB) *StateRepository {
	return &StateRepository{db: db}
}

// Get 获取指定 Skill 的状态
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - *SkillState: 状态记录，不存在返回 nil
//   - error: 查询错误
func (r *StateRepository) Get(name string) (*SkillState, error) {
	query := `SELECT name, description, enabled, created_at, updated_at FROM skill_states WHERE name = ?`
	row := r.db.QueryRow(query, name)

	var state SkillState
	err := row.Scan(&state.Name, &state.Description, &state.Enabled, &state.CreatedAt, &state.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 Skill 状态失败: %w", err)
	}

	return &state, nil
}

// GetEnabled 获取所有启用的 Skill 状态
// 参数:
//   - 无
//
// 返回:
//   - []*SkillState: 启用的 Skill 列表
//   - error: 查询错误
func (r *StateRepository) GetEnabled() ([]*SkillState, error) {
	query := `SELECT name, description, enabled, created_at, updated_at FROM skill_states WHERE enabled = 1`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询启用的 Skill 失败: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// List 获取所有 Skill 状态
// 参数:
//   - 无
//
// 返回:
//   - []*SkillState: Skill 状态列表
//   - error: 查询错误
func (r *StateRepository) List() ([]*SkillState, error) {
	query := `SELECT name, description, enabled, created_at, updated_at FROM skill_states ORDER BY name`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询 Skill 列表失败: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// Save 保存或更新 Skill 状态
// 参数:
//   - name: Skill 名称
//   - description: Skill 描述
//   - enabled: 是否启用
//
// 返回:
//   - error: 保存错误
func (r *StateRepository) Save(name, description string, enabled bool) error {
	query := `
		INSERT INTO skill_states (name, description, enabled, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(name) DO UPDATE SET
			description = excluded.description,
			updated_at = datetime('now')
	`
	_, err := r.db.Exec(query, name, description, enabled)
	if err != nil {
		return fmt.Errorf("保存 Skill 状态失败: %w", err)
	}
	return nil
}

// UpdateEnabled 更新启用状态
// 参数:
//   - name: Skill 名称
//   - enabled: 是否启用
//
// 返回:
//   - error: 更新错误
func (r *StateRepository) UpdateEnabled(name string, enabled bool) error {
	query := `UPDATE skill_states SET enabled = ?, updated_at = datetime('now') WHERE name = ?`
	result, err := r.db.Exec(query, enabled, name)
	if err != nil {
		return fmt.Errorf("更新 Skill 状态失败: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("Skill 不存在: %s", name)
	}

	return nil
}

// Delete 删除 Skill 状态记录
// 参数:
//   - name: Skill 名称
//
// 返回:
//   - error: 删除错误
func (r *StateRepository) Delete(name string) error {
	query := `DELETE FROM skill_states WHERE name = ?`
	_, err := r.db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("删除 Skill 状态失败: %w", err)
	}
	return nil
}

// SyncFromLoader 从 Loader 同步 Skill 到数据库
// 扫描文件系统，将新 Skill 插入数据库（默认启用），更新描述
// 参数:
//   - loader: Skill 加载器
//
// 返回:
//   - error: 同步错误
func (r *StateRepository) SyncFromLoader(loader *SkillLoader) error {
	// 获取文件系统中的所有 Skill
	skills, err := loader.Load()
	if err != nil {
		return fmt.Errorf("加载 Skill 失败: %w", err)
	}

	// 获取数据库中已有的 Skill 名称
	existingNames := make(map[string]bool)
	states, err := r.List()
	if err != nil {
		return fmt.Errorf("获取已有 Skill 状态失败: %w", err)
	}
	for _, s := range states {
		existingNames[s.Name] = true
	}

	// 插入或更新
	loadedNames := make(map[string]bool)
	for _, skill := range skills {
		name := skill.Name()
		loadedNames[name] = true
		if existingNames[name] {
			// 已存在，只更新描述
			if err := r.updateDescription(name, skill.Description()); err != nil {
				return fmt.Errorf("更新 Skill 描述失败 %s: %w", name, err)
			}
		} else {
			// 不存在，插入新记录（默认启用）
			if err := r.Save(name, skill.Description(), true); err != nil {
				return fmt.Errorf("保存 Skill 状态失败 %s: %w", name, err)
			}
		}
	}

	// 删除数据库中存在但磁盘中已不存在或缺少 SKILL.md 的记录
	for _, state := range states {
		if !loadedNames[state.Name] {
			if err := r.Delete(state.Name); err != nil {
				return fmt.Errorf("删除不存在的 Skill 状态失败 %s: %w", state.Name, err)
			}
		}
	}

	return nil
}

// updateDescription 更新描述
func (r *StateRepository) updateDescription(name, description string) error {
	query := `UPDATE skill_states SET description = ?, updated_at = datetime('now') WHERE name = ?`
	_, err := r.db.Exec(query, description, name)
	return err
}

// scanRows 扫描查询结果
func (r *StateRepository) scanRows(rows *sql.Rows) ([]*SkillState, error) {
	var states []*SkillState
	for rows.Next() {
		var state SkillState
		err := rows.Scan(&state.Name, &state.Description, &state.Enabled, &state.CreatedAt, &state.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描 Skill 状态失败: %w", err)
		}
		states = append(states, &state)
	}
	return states, rows.Err()
}
