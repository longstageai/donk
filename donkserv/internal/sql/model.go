package sql

import "time"

// CreativeRecord 创意记录结构
// 对应数据库表 creatives，存储创意去重Agent的创意记录
type CreativeRecord struct {
	ID          int64     `json:"id" db:"id"`                   // 主键ID
	Title       string    `json:"title" db:"title"`             // 创意标题
	Description string    `json:"description" db:"description"` // 创意描述
	Content     string    `json:"content" db:"content"`         // 创意内容
	Source      string    `json:"source" db:"source"`           // 来源
	Status      string    `json:"status" db:"status"`           // 状态: active/inactive
	CreatedAt   time.Time `json:"created_at" db:"created_at"`   // 创建时间
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`   // 更新时间
}

type CreativeRuntimeState struct {
	ID        int64     `json:"id" db:"id"`
	Status    string    `json:"status" db:"status"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
