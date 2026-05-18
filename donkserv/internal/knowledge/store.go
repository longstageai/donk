// knowledge 知识库模块
package knowledge

// Store 知识库存储接口
// 定义知识库元数据管理的统一接口
type Store interface {
	// AddDocument 添加文档记录
	// 如果文档已存在（相同路径），则更新记录
	// doc: 文档信息
	// 返回错误
	AddDocument(doc *Document) error

	// GetDocumentByHash 通过内容哈希查询文档
	// 用于去重检查
	// hash: MD5哈希值
	// 返回文档信息，如果不存在返回nil
	GetDocumentByHash(hash string) (*Document, error)

	// GetDocumentByPath 通过文件路径查询文档
	// path: 绝对路径
	// 返回文档信息，如果不存在返回nil
	GetDocumentByPath(path string) (*Document, error)

	// UpdateAccessStats 更新文档访问统计
	// 每次查询文档时调用，用于冷热分离
	// path: 文件路径
	// 返回错误
	UpdateAccessStats(path string) error

	// GetPendingDocuments 获取待处理文档列表
	// 按优先级排序（修改时间倒序）
	// limit: 最大返回数量
	// 返回文档列表
	GetPendingDocuments(limit int) ([]*Document, error)

	// GetHotDocuments 获取热数据文档
	// days: 最近N天内访问过
	// limit: 最大返回数量
	// 返回文档列表
	GetHotDocuments(days int, limit int) ([]*Document, error)

	// GetColdDocuments 获取冷数据文档
	// days: N天内未访问
	// limit: 最大返回数量
	// 返回文档列表
	GetColdDocuments(days int, limit int) ([]*Document, error)

	// MarkAsIndexed 标记文档为已索引
	// id: 文档ID
	// vectorID: 向量存储返回的ID
	// 返回错误
	MarkAsIndexed(id int, vectorID string) error

	// MarkAsFailed 标记文档为处理失败
	// id: 文档ID
	// errMsg: 错误信息
	// 返回错误
	MarkAsFailed(id int, errMsg string) error

	// MarkAsDuplicate 标记文档为重复
	// id: 文档ID
	// originalHash: 原始文档的哈希
	// 返回错误
	MarkAsDuplicate(id int, originalHash string) error

	// GetStats 获取知识库统计信息
	// 返回统计信息
	GetStats() (*Stats, error)

	// UpdateScanTime 更新扫描时间
	// 每次扫描完成后调用
	// 返回错误
	UpdateScanTime() error

	// Close 关闭存储连接
	// 返回错误
	Close() error

	// GetVectorDimension 获取当前记录的向量维度
	// 返回记录的维度值，如果未设置返回 0
	GetVectorDimension() (int, error)

	// SetVectorDimension 设置向量维度
	// dimension: 新的维度值
	// 返回错误
	SetVectorDimension(dimension int) error

	// ClearAllDocuments 清空所有文档记录
	// 用于维度变化时重置知识库
	// 返回错误
	ClearAllDocuments() error
}
