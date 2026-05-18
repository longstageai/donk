// knowledge 知识库模块
package knowledge

import "time"

// Document 知识库文档结构
type Document struct {
	ID             int       // 主键ID
	ContentHash    string    // 内容哈希（MD5），用于去重
	Content        string    // 文档内容（前1000字摘要）
	VectorID       string    // 向量存储中的ID
	FilePath       string    // 文件绝对路径
	FileSize       int64     // 文件大小（字节）
	ModifiedTime   time.Time // 文件修改时间
	CreatedAt      time.Time // 首次索引时间
	UpdatedAt      time.Time // 最后更新时间
	AccessCount    int       // 访问次数（冷热分离）
	LastAccessTime time.Time // 最后访问时间
	Status         string    // 状态：pending / indexed / failed / duplicate
	ErrorMsg       string    // 失败原因
}

// DocumentStatus 文档状态常量
const (
	StatusPending   = "pending"   // 待处理
	StatusIndexed   = "indexed"   // 已索引
	StatusFailed    = "failed"    // 处理失败
	StatusDuplicate = "duplicate" // 重复内容
)

// Stats 知识库统计信息
type Stats struct {
	TotalDocuments int       // 总文档数
	IndexedCount   int       // 已索引数
	PendingCount   int       // 待处理数
	FailedCount    int       // 失败数
	DuplicateCount int       // 重复数
	TotalFileSize  int64     // 总文件大小
	LastScanTime   time.Time // 最后扫描时间
	LastIndexTime  time.Time // 最后索引时间
}

// ScanResult 扫描结果
type ScanResult struct {
	FilePath     string
	FileSize     int64
	ModifiedTime time.Time
	Extension    string
}

// PriorityScore 优先级分数
type PriorityScore struct {
	Document *Document
	Score    float64 // 优先级分数（越高越优先）
}
