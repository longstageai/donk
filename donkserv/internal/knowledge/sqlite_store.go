// knowledge 知识库模块
package knowledge

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/longstageai/donk/donk/pkg/logger"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore SQLite实现的知识库存储
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 创建SQLite存储实例
// dbPath: 数据库文件路径
// 返回存储实例或错误
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("启用外键约束失败: %w", err)
	}

	store := &SQLiteStore{db: db}

	// 初始化表结构
	if err := store.initTables(); err != nil {
		return nil, fmt.Errorf("初始化表结构失败: %w", err)
	}

	logger.Info("知识库SQLite存储初始化成功", map[string]interface{}{
		"path": dbPath,
	})

	return store, nil
}

// initTables 初始化数据库表结构
// 创建kb_documents表和索引
// 返回错误
func (s *SQLiteStore) initTables() error {
	// 创建文档表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS kb_documents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content_hash TEXT UNIQUE,
		content TEXT,
		vector_id TEXT,
		file_path TEXT UNIQUE,
		file_size INTEGER,
		modified_time TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		access_count INTEGER DEFAULT 0,
		last_access_time TIMESTAMP,
		status TEXT DEFAULT 'pending',
		error_msg TEXT
	);`

	if _, err := s.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("创建文档表失败: %w", err)
	}

	// 创建统计表
	createStatsSQL := `
	CREATE TABLE IF NOT EXISTS kb_stats (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		last_scan_time TIMESTAMP,
		last_index_time TIMESTAMP,
		vector_dimension INTEGER DEFAULT 0
	);`

	if _, err := s.db.Exec(createStatsSQL); err != nil {
		return fmt.Errorf("创建统计表失败: %w", err)
	}

	// 初始化统计记录
	if _, err := s.db.Exec("INSERT OR IGNORE INTO kb_stats (id) VALUES (1)"); err != nil {
		return fmt.Errorf("初始化统计记录失败: %w", err)
	}

	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_kb_hash ON kb_documents(content_hash);`,
		`CREATE INDEX IF NOT EXISTS idx_kb_path ON kb_documents(file_path);`,
		`CREATE INDEX IF NOT EXISTS idx_kb_access ON kb_documents(last_access_time, access_count);`,
		`CREATE INDEX IF NOT EXISTS idx_kb_status ON kb_documents(status);`,
		`CREATE INDEX IF NOT EXISTS idx_kb_modified ON kb_documents(modified_time);`,
	}

	for _, idx := range indexes {
		if _, err := s.db.Exec(idx); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	logger.Debug("知识库表结构初始化完成", nil)
	return nil
}

// GetVectorDimension 获取当前记录的向量维度
// 返回记录的维度值，如果未设置返回 0
func (s *SQLiteStore) GetVectorDimension() (int, error) {
	var dimension int
	err := s.db.QueryRow("SELECT COALESCE(vector_dimension, 0) FROM kb_stats WHERE id = 1").Scan(&dimension)
	if err != nil {
		return 0, fmt.Errorf("获取向量维度失败: %w", err)
	}
	return dimension, nil
}

// SetVectorDimension 设置向量维度
// dimension: 新的维度值
// 返回错误
func (s *SQLiteStore) SetVectorDimension(dimension int) error {
	_, err := s.db.Exec("UPDATE kb_stats SET vector_dimension = ? WHERE id = 1", dimension)
	if err != nil {
		return fmt.Errorf("设置向量维度失败: %w", err)
	}
	return nil
}

// ClearAllDocuments 清空所有文档记录
// 用于维度变化时重置知识库
// 返回错误
func (s *SQLiteStore) ClearAllDocuments() error {
	_, err := s.db.Exec("DELETE FROM kb_documents")
	if err != nil {
		return fmt.Errorf("清空文档记录失败: %w", err)
	}
	logger.Info("已清空所有文档记录", nil)
	return nil
}

// AddDocument 添加文档记录
// 如果路径已存在则更新，否则插入新记录
// doc: 文档信息
// 返回错误
func (s *SQLiteStore) AddDocument(doc *Document) error {
	// 使用UPSERT语法（SQLite 3.24.0+）
	query := `
		INSERT INTO kb_documents 
		(content_hash, content, file_path, file_size, modified_time, status, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			content_hash = excluded.content_hash,
			content = excluded.content,
			file_size = excluded.file_size,
			modified_time = excluded.modified_time,
			status = excluded.status,
			error_msg = excluded.error_msg,
			updated_at = CURRENT_TIMESTAMP
		WHERE excluded.modified_time > kb_documents.modified_time`

	_, err := s.db.Exec(query,
		doc.ContentHash,
		doc.Content,
		doc.FilePath,
		doc.FileSize,
		doc.ModifiedTime,
		doc.Status,
		doc.ErrorMsg,
	)

	if err != nil {
		return fmt.Errorf("添加文档失败: %w", err)
	}

	logger.Debug("添加文档记录", map[string]interface{}{
		"path": doc.FilePath,
		"hash": doc.ContentHash,
	})

	return nil
}

// GetDocumentByHash 通过内容哈希查询文档
// hash: MD5哈希值
// 返回文档信息，如果不存在返回nil
func (s *SQLiteStore) GetDocumentByHash(hash string) (*Document, error) {
	query := `
		SELECT id, content_hash, content, vector_id, file_path, file_size,
		       modified_time, created_at, updated_at, access_count, last_access_time, status, error_msg
		FROM kb_documents
		WHERE content_hash = ?
		LIMIT 1`

	row := s.db.QueryRow(query, hash)
	return s.scanDocument(row)
}

// GetDocumentByPath 通过文件路径查询文档
// path: 绝对路径
// 返回文档信息，如果不存在返回nil
func (s *SQLiteStore) GetDocumentByPath(path string) (*Document, error) {
	query := `
		SELECT id, content_hash, content, vector_id, file_path, file_size,
		       modified_time, created_at, updated_at, access_count, last_access_time, status, error_msg
		FROM kb_documents
		WHERE file_path = ?
		LIMIT 1`

	row := s.db.QueryRow(query, path)
	return s.scanDocument(row)
}

// GetDocumentByVectorID 通过向量ID查询文档
// vectorID: 向量ID
// 返回文档信息，如果不存在返回nil
func (s *SQLiteStore) GetDocumentByVectorID(vectorID string) (*Document, error) {
	query := `
		SELECT id, content_hash, content, vector_id, file_path, file_size,
		       modified_time, created_at, updated_at, access_count, last_access_time, status, error_msg
		FROM kb_documents
		WHERE vector_id = ?
		LIMIT 1`

	row := s.db.QueryRow(query, vectorID)
	return s.scanDocument(row)
}

// GetDocumentByContent 通过内容查询文档
// content: 文档内容
// 返回文档信息，如果不存在返回nil
func (s *SQLiteStore) GetDocumentByContent(content string) (*Document, error) {
	query := `
		SELECT id, content_hash, content, vector_id, file_path, file_size,
		       modified_time, created_at, updated_at, access_count, last_access_time, status, error_msg
		FROM kb_documents
		WHERE content = ?
		LIMIT 1`

	row := s.db.QueryRow(query, content)
	return s.scanDocument(row)
}

// scanDocument 扫描文档行
// row: 查询行
// 返回文档信息或错误
func (s *SQLiteStore) scanDocument(row *sql.Row) (*Document, error) {
	var doc Document
	var modifiedTime, createdAt, updatedAt, lastAccessTime sql.NullTime
	var vectorID sql.NullString

	err := row.Scan(
		&doc.ID,
		&doc.ContentHash,
		&doc.Content,
		&vectorID,
		&doc.FilePath,
		&doc.FileSize,
		&modifiedTime,
		&createdAt,
		&updatedAt,
		&doc.AccessCount,
		&lastAccessTime,
		&doc.Status,
		&doc.ErrorMsg,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	doc.VectorID = vectorID.String
	doc.ModifiedTime = modifiedTime.Time
	doc.CreatedAt = createdAt.Time
	doc.UpdatedAt = updatedAt.Time
	doc.LastAccessTime = lastAccessTime.Time

	return &doc, nil
}

// UpdateAccessStats 更新文档访问统计
// path: 文件路径
// 返回错误
func (s *SQLiteStore) UpdateAccessStats(path string) error {
	query := `
		UPDATE kb_documents
		SET access_count = access_count + 1,
		    last_access_time = CURRENT_TIMESTAMP
		WHERE file_path = ?`

	result, err := s.db.Exec(query, path)
	if err != nil {
		return fmt.Errorf("更新访问统计失败: %w", err)
	}

	rows, _ := result.RowsAffected()
	logger.Debug("更新访问统计", map[string]interface{}{
		"path": path,
		"rows": rows,
	})

	return nil
}

// UpdateAccess 更新文档访问次数（简化版）
// path: 文件路径
// 返回错误
func (s *SQLiteStore) UpdateAccess(path string) error {
	return s.UpdateAccessStats(path)
}

// GetPendingDocuments 获取待处理文档列表
// 按修改时间倒序（新的优先）
// limit: 最大返回数量
// 返回文档列表
func (s *SQLiteStore) GetPendingDocuments(limit int) ([]*Document, error) {
	query := `
		SELECT id, content_hash, content, vector_id, file_path, file_size,
		       modified_time, created_at, updated_at, access_count, last_access_time, status, error_msg
		FROM kb_documents
		WHERE status = 'pending'
		ORDER BY modified_time DESC
		LIMIT ?`

	return s.queryDocuments(query, limit)
}

// GetHotDocuments 获取热数据文档
// days: 最近N天内访问过
// limit: 最大返回数量
// 返回文档列表
func (s *SQLiteStore) GetHotDocuments(days int, limit int) ([]*Document, error) {
	query := `
		SELECT id, content_hash, content, vector_id, file_path, file_size,
		       modified_time, created_at, updated_at, access_count, last_access_time, status, error_msg
		FROM kb_documents
		WHERE last_access_time >= datetime('now', '-' || ? || ' days')
		ORDER BY access_count DESC, last_access_time DESC
		LIMIT ?`

	return s.queryDocuments(query, days, limit)
}

// GetColdDocuments 获取冷数据文档
// days: N天内未访问
// limit: 最大返回数量
// 返回文档列表
func (s *SQLiteStore) GetColdDocuments(days int, limit int) ([]*Document, error) {
	query := `
		SELECT id, content_hash, content, vector_id, file_path, file_size,
		       modified_time, created_at, updated_at, access_count, last_access_time, status, error_msg
		FROM kb_documents
		WHERE last_access_time < datetime('now', '-' || ? || ' days')
		   OR last_access_time IS NULL
		ORDER BY access_count ASC
		LIMIT ?`

	return s.queryDocuments(query, days, limit)
}

// queryDocuments 查询文档列表
// query: SQL查询语句
// args: 查询参数
// 返回文档列表或错误
func (s *SQLiteStore) queryDocuments(query string, args ...interface{}) ([]*Document, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询文档失败: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		var doc Document
		var modifiedTime, createdAt, updatedAt, lastAccessTime sql.NullTime
		var vectorID sql.NullString

		err := rows.Scan(
			&doc.ID,
			&doc.ContentHash,
			&doc.Content,
			&vectorID,
			&doc.FilePath,
			&doc.FileSize,
			&modifiedTime,
			&createdAt,
			&updatedAt,
			&doc.AccessCount,
			&lastAccessTime,
			&doc.Status,
			&doc.ErrorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描文档行失败: %w", err)
		}

		doc.VectorID = vectorID.String
		doc.ModifiedTime = modifiedTime.Time
		doc.CreatedAt = createdAt.Time
		doc.UpdatedAt = updatedAt.Time
		doc.LastAccessTime = lastAccessTime.Time

		documents = append(documents, &doc)
	}

	return documents, rows.Err()
}

// MarkAsIndexed 标记文档为已索引
// id: 文档ID
// vectorID: 向量存储返回的ID
// 返回错误
func (s *SQLiteStore) MarkAsIndexed(id int, vectorID string) error {
	query := `
		UPDATE kb_documents
		SET status = 'indexed',
		    vector_id = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err := s.db.Exec(query, vectorID, id)
	if err != nil {
		return fmt.Errorf("标记为已索引失败: %w", err)
	}

	return nil
}

// MarkAsFailed 标记文档为处理失败
// id: 文档ID
// errMsg: 错误信息
// 返回错误
func (s *SQLiteStore) MarkAsFailed(id int, errMsg string) error {
	query := `
		UPDATE kb_documents
		SET status = 'failed',
		    error_msg = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err := s.db.Exec(query, errMsg, id)
	if err != nil {
		return fmt.Errorf("标记为失败失败: %w", err)
	}

	return nil
}

// MarkAsDuplicate 标记文档为重复
// id: 文档ID
// originalHash: 原始文档的哈希
// 返回错误
func (s *SQLiteStore) MarkAsDuplicate(id int, originalHash string) error {
	query := `
		UPDATE kb_documents
		SET status = 'duplicate',
		    error_msg = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	_, err := s.db.Exec(query, fmt.Sprintf("Duplicate of hash: %s", originalHash), id)
	if err != nil {
		return fmt.Errorf("标记为重复失败: %w", err)
	}

	return nil
}

// GetStats 获取知识库统计信息
// 返回统计信息
func (s *SQLiteStore) GetStats() (*Stats, error) {
	stats := &Stats{}

	// 统计各状态文档数量
	query := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'indexed' THEN 1 ELSE 0 END) as indexed,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			SUM(CASE WHEN status = 'duplicate' THEN 1 ELSE 0 END) as duplicate,
			SUM(file_size) as total_size
		FROM kb_documents`

	err := s.db.QueryRow(query).Scan(
		&stats.TotalDocuments,
		&stats.IndexedCount,
		&stats.PendingCount,
		&stats.FailedCount,
		&stats.DuplicateCount,
		&stats.TotalFileSize,
	)
	if err != nil {
		return nil, fmt.Errorf("获取统计信息失败: %w", err)
	}

	// 获取时间信息
	var lastScanTime, lastIndexTime sql.NullTime
	err = s.db.QueryRow("SELECT last_scan_time, last_index_time FROM kb_stats WHERE id = 1").Scan(
		&lastScanTime, &lastIndexTime,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("获取时间信息失败: %w", err)
	}

	stats.LastScanTime = lastScanTime.Time
	stats.LastIndexTime = lastIndexTime.Time

	return stats, nil
}

// UpdateScanTime 更新扫描时间
// 返回错误
func (s *SQLiteStore) UpdateScanTime() error {
	query := `UPDATE kb_stats SET last_scan_time = CURRENT_TIMESTAMP WHERE id = 1`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("更新扫描时间失败: %w", err)
	}
	return nil
}

// UpdateIndexTime 更新索引时间
// 返回错误
func (s *SQLiteStore) UpdateIndexTime() error {
	query := `UPDATE kb_stats SET last_index_time = CURRENT_TIMESTAMP WHERE id = 1`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("更新索引时间失败: %w", err)
	}
	return nil
}

// Close 关闭存储连接
// 返回错误
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
