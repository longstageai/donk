package profile

import (
	"context"
)

// Storage 画像存储接口
// 定义用户画像的存储和读取操作
type Storage interface {
	// Load 加载用户画像
	//
	// 参数:
	//   - ctx: 上下文
	//   - userID: 用户ID
	//
	// 返回:
	//   - *UserProfile: 用户画像
	//   - error: 错误信息
	Load(ctx context.Context, userID string) (*UserProfile, error)

	// Save 保存用户画像
	//
	// 参数:
	//   - ctx: 上下文
	//   - profile: 用户画像
	//
	// 返回:
	//   - error: 错误信息
	Save(ctx context.Context, profile *UserProfile) error

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
	RecordHistory(ctx context.Context, userID string, changeType string, tagName string, oldValue, newValue interface{}) error

	// DeleteTag 删除标签
	//
	// 参数:
	//   - ctx: 上下文
	//   - userID: 用户ID
	//   - tagName: 标签名
	//
	// 返回:
	//   - error: 错误信息
	DeleteTag(ctx context.Context, userID string, tagName string) error

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
	GetHistory(ctx context.Context, userID string, limit int) ([]ProfileHistory, error)
}
