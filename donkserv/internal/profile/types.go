package profile

import (
	"fmt"
	"time"
)

// UserProfile 用户画像
type UserProfile struct {
	UserID      string            `json:"user_id"`
	Tags        map[string]Tag    `json:"tags"`
	Preferences map[string]string `json:"preferences"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Tag 画像标签
type Tag struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`       // skill/interest/preference
	Confidence float64   `json:"confidence"` // 0.0-1.0
	Evidence   string    `json:"evidence"`   // 原文证据
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewEmptyProfile 创建空画像
func NewEmptyProfile(userID string) *UserProfile {
	return &UserProfile{
		UserID:      userID,
		Tags:        make(map[string]Tag),
		Preferences: make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ToPrompt 将画像转换为Prompt文本
//
// 返回:
//   - string: Prompt文本
func (p *UserProfile) ToPrompt() string {
	if len(p.Tags) == 0 && len(p.Preferences) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "## 用户画像")

	// 高置信度标签（>0.7）
	if len(p.Tags) > 0 {
		var tags []string
		for _, tag := range p.Tags {
			if tag.Confidence > 0.7 {
				tags = append(tags, "- "+tag.Name+" ("+tag.Type+")")
			}
		}
		if len(tags) > 0 {
			parts = append(parts, "### 特征\n"+joinStrings(tags, "\n"))
		}
	}

	// 偏好
	if len(p.Preferences) > 0 {
		var prefs []string
		for k, v := range p.Preferences {
			prefs = append(prefs, "- "+k+": "+v)
		}
		parts = append(parts, "### 偏好\n"+joinStrings(prefs, "\n"))
	}

	return joinStrings(parts, "\n\n")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// ProfileHistory 画像变更历史记录
type ProfileHistory struct {
	ID         int64     `json:"id"`
	UserID     string    `json:"user_id"`
	ChangeType string    `json:"change_type"` // add/update/delete
	TagName    string    `json:"tag_name"`
	OldValue   string    `json:"old_value"`
	NewValue   string    `json:"new_value"`
	CreatedAt  time.Time `json:"created_at"`
}

// ToPrompt 将历史记录转换为Prompt文本
//
// 返回:
//   - string: Prompt文本
func (h *ProfileHistory) ToPrompt() string {
	var changeDesc string
	switch h.ChangeType {
	case "add":
		changeDesc = fmt.Sprintf("新增标签 [%s]: %s", h.TagName, h.NewValue)
	case "update":
		changeDesc = fmt.Sprintf("更新标签 [%s]: %s -> %s", h.TagName, h.OldValue, h.NewValue)
	case "delete":
		changeDesc = fmt.Sprintf("删除标签 [%s]", h.TagName)
	default:
		changeDesc = fmt.Sprintf("变更标签 [%s]: %s", h.TagName, h.NewValue)
	}
	return fmt.Sprintf("- [%s] %s", h.CreatedAt.Format("2006-01-02 15:04"), changeDesc)
}
