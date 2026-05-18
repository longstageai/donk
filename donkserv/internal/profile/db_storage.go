package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// DBStorage 数据库存储实现
// 依赖 sql/setting.go 中定义的表结构
type DBStorage struct {
	db *sql.DB
}

// NewDBStorage 创建数据库存储
// 注意：表结构由 sql/setting.go 统一管理初始化
//
// 参数:
//   - db: 数据库连接
//
// 返回:
//   - *DBStorage: 存储实例
func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{db: db}
}

// Load 加载用户画像
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//
// 返回:
//   - *UserProfile: 用户画像
//   - error: 错误信息
func (s *DBStorage) Load(ctx context.Context, userID string) (*UserProfile, error) {
	profile := &UserProfile{
		UserID: userID,
		Tags:   make(map[string]Tag),
	}

	// 加载主表
	var preferencesJSON string
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(ctx,
		"SELECT preferences, created_at, updated_at FROM user_profiles WHERE user_id = ?",
		userID,
	).Scan(&preferencesJSON, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		// 不存在，返回空画像
		profile.Preferences = make(map[string]string)
		return profile, nil
	}
	if err != nil {
		return nil, fmt.Errorf("加载画像失败: %w", err)
	}

	profile.CreatedAt = time.Unix(createdAt, 0)
	profile.UpdatedAt = time.Unix(updatedAt, 0)

	// 解析偏好
	if preferencesJSON != "" {
		json.Unmarshal([]byte(preferencesJSON), &profile.Preferences)
	} else {
		profile.Preferences = make(map[string]string)
	}

	// 加载标签
	tags, err := s.loadTags(ctx, userID)
	if err != nil {
		return nil, err
	}
	profile.Tags = tags

	return profile, nil
}

// loadTags 加载用户标签
func (s *DBStorage) loadTags(ctx context.Context, userID string) (map[string]Tag, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT name, type, confidence, evidence, created_at, updated_at FROM profile_tags WHERE user_id = ?",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make(map[string]Tag)
	for rows.Next() {
		var tag Tag
		var createdAt, updatedAt int64
		err := rows.Scan(&tag.Name, &tag.Type, &tag.Confidence, &tag.Evidence, &createdAt, &updatedAt)
		if err != nil {
			continue
		}
		tag.CreatedAt = time.Unix(createdAt, 0)
		tag.UpdatedAt = time.Unix(updatedAt, 0)
		tags[tag.Name] = tag
	}

	return tags, nil
}

// Save 保存用户画像
//
// 参数:
//   - ctx: 上下文
//   - profile: 用户画像
//
// 返回:
//   - error: 错误信息
func (s *DBStorage) Save(ctx context.Context, profile *UserProfile) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 保存主表
	preferencesJSON, _ := json.Marshal(profile.Preferences)
	now := time.Now().Unix()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_profiles (user_id, preferences, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			preferences = excluded.preferences,
			updated_at = excluded.updated_at
	`, profile.UserID, string(preferencesJSON), now, now)

	if err != nil {
		return fmt.Errorf("保存画像主表失败: %w", err)
	}

	// 保存标签
	for _, tag := range profile.Tags {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO profile_tags (user_id, name, type, confidence, evidence, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(user_id, name) DO UPDATE SET
				type = excluded.type,
				confidence = excluded.confidence,
				evidence = excluded.evidence,
				updated_at = excluded.updated_at
		`, profile.UserID, tag.Name, tag.Type, tag.Confidence, tag.Evidence, tag.CreatedAt.Unix(), tag.UpdatedAt.Unix())

		if err != nil {
			return fmt.Errorf("保存标签失败: %w", err)
		}
	}

	return tx.Commit()
}

// RecordHistory 记录变更历史
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - changeType: 变更类型
//   - tagName: 标签名
//   - oldValue: 旧值
//   - newValue: 新值
//
// 返回:
//   - error: 错误信息
func (s *DBStorage) RecordHistory(ctx context.Context, userID string, changeType string, tagName string, oldValue, newValue interface{}) error {
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO profile_history (user_id, change_type, tag_name, old_value, new_value, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		userID, changeType, tagName, string(oldJSON), string(newJSON), time.Now().Unix(),
	)

	return err
}

// DeleteTag 删除标签
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - tagName: 标签名
//
// 返回:
//   - error: 错误信息
func (s *DBStorage) DeleteTag(ctx context.Context, userID string, tagName string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM profile_tags WHERE user_id = ? AND name = ?",
		userID, tagName,
	)
	return err
}

// GetHistory 获取画像变更历史
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - limit: 返回记录数量限制（0表示不限制）
//
// 返回:
//   - []ProfileHistory: 历史记录列表
//   - error: 错误信息
func (s *DBStorage) GetHistory(ctx context.Context, userID string, limit int) ([]ProfileHistory, error) {
	var query string
	var args []interface{}

	if limit > 0 {
		query = "SELECT id, user_id, change_type, tag_name, old_value, new_value, created_at FROM profile_history WHERE user_id = ? ORDER BY created_at DESC LIMIT ?"
		args = []interface{}{userID, limit}
	} else {
		query = "SELECT id, user_id, change_type, tag_name, old_value, new_value, created_at FROM profile_history WHERE user_id = ? ORDER BY created_at DESC"
		args = []interface{}{userID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询画像历史失败: %w", err)
	}
	defer rows.Close()

	var history []ProfileHistory
	for rows.Next() {
		var h ProfileHistory
		var createdAt int64
		err := rows.Scan(&h.ID, &h.UserID, &h.ChangeType, &h.TagName, &h.OldValue, &h.NewValue, &createdAt)
		if err != nil {
			continue
		}
		h.CreatedAt = time.Unix(createdAt, 0)
		history = append(history, h)
	}

	return history, nil
}
