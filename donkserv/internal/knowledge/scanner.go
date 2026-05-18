// knowledge 知识库模块
package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/longstageai/donk/donk/pkg/logger"
)

// Scanner 文件扫描器
// 负责扫描指定目录，发现符合要求的文档文件
type Scanner struct {
	extensions  map[string]bool // 支持的文件扩展名
	maxDepth    int             // 最大扫描深度
	maxFileSize int64           // 最大文件大小（字节）
}

// NewScanner 创建文件扫描器
// maxDepth: 最大扫描深度（0表示不限制，1表示仅当前目录）
// maxFileSize: 最大文件大小（字节，0表示不限制）
// 返回扫描器实例
func NewScanner(maxDepth int, maxFileSize int64) *Scanner {
	return &Scanner{
		extensions: map[string]bool{
			".txt":  true,
			".md":   true,
			".pdf":  true,
			".docx": true,
		},
		maxDepth:    maxDepth,
		maxFileSize: maxFileSize,
	}
}

// ScanDirectories 扫描多个目录
// directories: 目录路径列表
// 返回扫描结果列表或错误
func (s *Scanner) ScanDirectories(directories []string) ([]*ScanResult, error) {
	var allResults []*ScanResult

	for _, dir := range directories {
		results, err := s.ScanDirectory(dir)
		if err != nil {
			logger.Warn("扫描目录失败", map[string]interface{}{
				"directory": dir,
				"error":     err.Error(),
			})
			continue
		}
		allResults = append(allResults, results...)
	}

	logger.Info("目录扫描完成", map[string]interface{}{
		"directories": len(directories),
		"total_files": len(allResults),
	})

	return allResults, nil
}

// ScanDirectory 扫描单个目录
// root: 根目录路径
// 返回扫描结果列表或错误
func (s *Scanner) ScanDirectory(root string) ([]*ScanResult, error) {
	// 展开用户目录（如 ~/Desktop）
	expandedPath := expandUserPath(root)

	// 检查目录是否存在
	info, err := os.Stat(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("无法访问目录: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", expandedPath)
	}

	var results []*ScanResult

	// 使用WalkDir遍历目录
	err = filepath.WalkDir(expandedPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			logger.Debug("访问路径失败", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			return nil // 继续遍历其他路径
		}

		// 跳过目录
		if d.IsDir() {
			// 检查深度限制
			if s.maxDepth > 0 {
				depth := calculateDepth(expandedPath, path)
				if depth > s.maxDepth {
					return filepath.SkipDir // 跳过此目录
				}
			}
			return nil
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(path))
		if !s.extensions[ext] {
			return nil
		}

		// 获取文件信息
		fileInfo, err := d.Info()
		if err != nil {
			logger.Debug("获取文件信息失败", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			return nil
		}

		// 检查文件大小限制
		if s.maxFileSize > 0 && fileInfo.Size() > s.maxFileSize {
			logger.Debug("文件超过大小限制，跳过", map[string]interface{}{
				"path": path,
				"size": fileInfo.Size(),
			})
			return nil
		}

		// 添加到结果
		result := &ScanResult{
			FilePath:     path,
			FileSize:     fileInfo.Size(),
			ModifiedTime: fileInfo.ModTime(),
			Extension:    ext,
		}
		results = append(results, result)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}

	logger.Debug("目录扫描完成", map[string]interface{}{
		"directory":  expandedPath,
		"file_count": len(results),
	})

	return results, nil
}

// calculateDepth 计算相对于根目录的深度
// root: 根目录
// path: 当前路径
// 返回深度（1表示根目录下的直接子目录）
func calculateDepth(root, path string) int {
	// 获取相对路径
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}

	// 计算路径分隔符数量
	// 例如："subdir/file.txt" 有1个分隔符，深度为2
	sepCount := strings.Count(rel, string(filepath.Separator))
	return sepCount + 1
}

// expandUserPath 展开用户目录路径
// 将 ~/Desktop 转换为 C:\Users\用户名\Desktop
// path: 原始路径
// 返回展开后的路径
func expandUserPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	// 替换 ~ 为用户主目录
	if path == "~" {
		return homeDir
	}

	return filepath.Join(homeDir, path[2:])
}

// GetDefaultDirectories 获取默认扫描目录（Windows用户目录）
// 返回默认目录列表
func GetDefaultDirectories() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("获取用户主目录失败", map[string]interface{}{
			"error": err.Error(),
		})
		return []string{}
	}

	return []string{
		filepath.Join(homeDir, "Desktop"),   // 桌面
		filepath.Join(homeDir, "Downloads"), // 下载
		filepath.Join(homeDir, "Documents"), // 文档
	}
}

// FilterNewFiles 筛选新文件（未在数据库中或已修改）
// scanResults: 扫描结果列表
// store: 知识库存储
// 返回新文件列表
func FilterNewFiles(scanResults []*ScanResult, store Store) ([]*ScanResult, error) {
	var newFiles []*ScanResult

	for _, result := range scanResults {
		// 查询数据库
		doc, err := store.GetDocumentByPath(result.FilePath)
		if err != nil {
			logger.Warn("查询文档失败", map[string]interface{}{
				"path":  result.FilePath,
				"error": err.Error(),
			})
			continue
		}

		// 如果文档不存在，或已修改，则视为新文件
		if doc == nil || doc.ModifiedTime.Before(result.ModifiedTime) {
			newFiles = append(newFiles, result)
		}
	}

	logger.Info("新文件筛选完成", map[string]interface{}{
		"total_scanned": len(scanResults),
		"new_files":     len(newFiles),
	})

	return newFiles, nil
}

// ScanResultToDocument 将扫描结果转换为文档对象
// result: 扫描结果
// content: 文档内容（前1000字摘要）
// hash: 内容哈希
// 返回文档对象
func ScanResultToDocument(result *ScanResult, content string, hash string) *Document {
	return &Document{
		ContentHash:  hash,
		Content:      truncateString(content, 1000),
		FilePath:     result.FilePath,
		FileSize:     result.FileSize,
		ModifiedTime: result.ModifiedTime,
		Status:       StatusPending,
	}
}

// truncateString 截断字符串到指定长度
// s: 原始字符串
// maxLen: 最大长度
// 返回截断后的字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
