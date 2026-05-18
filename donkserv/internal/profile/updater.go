package profile

import (
	"context"
	"math"
	"time"
)

// Updater 画像更新器
type Updater struct {
	storage Storage
}

// NewUpdater 创建更新器
func NewUpdater(storage Storage) *Updater {
	return &Updater{storage: storage}
}

// Update 根据提取结果更新画像
//
// 参数:
//   - ctx: 上下文
//   - profile: 当前画像
//   - result: 提取结果
//
// 返回:
//   - error: 错误信息
func (u *Updater) Update(ctx context.Context, profile *UserProfile, result *ExtractionResult) error {
	now := time.Now()

	// 更新标签
	for _, newTag := range result.Tags {
		if existing, ok := profile.Tags[newTag.Name]; ok {
			// 已存在，保留置信度高的
			if newTag.Confidence > existing.Confidence {
				// 记录历史
				u.storage.RecordHistory(ctx, profile.UserID, "update", newTag.Name, existing, newTag)
				// 更新
				profile.Tags[newTag.Name] = Tag{
					Name:       newTag.Name,
					Type:       newTag.Type,
					Confidence: newTag.Confidence,
					Evidence:   newTag.Evidence,
					CreatedAt:  existing.CreatedAt,
					UpdatedAt:  now,
				}
			}
		} else {
			// 新标签
			profile.Tags[newTag.Name] = Tag{
				Name:       newTag.Name,
				Type:       newTag.Type,
				Confidence: newTag.Confidence,
				Evidence:   newTag.Evidence,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			u.storage.RecordHistory(ctx, profile.UserID, "add", newTag.Name, nil, newTag)
		}
	}

	// 更新偏好
	for k, v := range result.Preferences {
		if oldValue, ok := profile.Preferences[k]; ok && oldValue != v {
			u.storage.RecordHistory(ctx, profile.UserID, "update", "preference:"+k, oldValue, v)
		}
		profile.Preferences[k] = v
	}

	// 应用时间衰减
	u.applyDecay(profile, now)

	// 清理低置信度标签
	u.cleanupLowConfidence(profile, 0.3)

	profile.UpdatedAt = now

	// 保存到数据库
	return u.storage.Save(ctx, profile)
}

// applyDecay 应用时间衰减
// 旧标签置信度随时间降低
func (u *Updater) applyDecay(profile *UserProfile, now time.Time) {
	for name, tag := range profile.Tags {
		days := now.Sub(tag.UpdatedAt).Hours() / 24
		if days > 0 {
			// 每天衰减5%
			decay := math.Pow(0.95, days)
			tag.Confidence *= decay
			profile.Tags[name] = tag
		}
	}
}

// cleanupLowConfidence 清理低置信度标签
func (u *Updater) cleanupLowConfidence(profile *UserProfile, threshold float64) {
	for name, tag := range profile.Tags {
		if tag.Confidence < threshold {
			delete(profile.Tags, name)
		}
	}
}
