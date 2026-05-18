// knowledge 知识库模块
package knowledge

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/pkg/logger"

	"github.com/fumiama/go-docx"
	"github.com/ledongthuc/pdf"
)

// Indexer 文档索引器
// 负责读取文件内容、生成哈希、创建向量索引
type Indexer struct {
	metaStore   Store              // 元数据存储
	vectorStore *VectorStore       // 向量存储
	embedder    embedding.Embedder // 嵌入器
}

// NewIndexer 创建文档索引器
// metaStore: 元数据存储
// vectorStore: 向量存储
// embedder: 嵌入器
// 返回索引器实例
func NewIndexer(metaStore Store, vectorStore *VectorStore, embedder embedding.Embedder) *Indexer {
	return &Indexer{
		metaStore:   metaStore,
		vectorStore: vectorStore,
		embedder:    embedder,
	}
}

// IndexDocument 索引单个文档
// ctx: 上下文
// scanResult: 扫描结果
// 返回错误
func (idx *Indexer) IndexDocument(ctx context.Context, scanResult *ScanResult) error {
	logger.Info("开始索引文档", map[string]interface{}{
		"path": scanResult.FilePath,
		"size": scanResult.FileSize,
	})

	// 1. 读取文件内容
	content, err := idx.readFileContent(scanResult.FilePath, scanResult.Extension)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 2. 计算内容哈希（用于去重）
	hash := calculateHash(content)

	// 3. 检查是否已存在（去重）
	existingDoc, err := idx.metaStore.GetDocumentByHash(hash)
	if err != nil {
		return fmt.Errorf("查询文档哈希失败: %w", err)
	}

	// 如果哈希已存在，说明内容重复
	if existingDoc != nil {
		logger.Info("文档内容重复，跳过索引", map[string]interface{}{
			"path": scanResult.FilePath,
			"hash": hash,
		})

		// 记录为重复文档
		doc := ScanResultToDocument(scanResult, content, hash)
		doc.Status = StatusDuplicate
		if err := idx.metaStore.AddDocument(doc); err != nil {
			return fmt.Errorf("记录重复文档失败: %w", err)
		}

		return nil
	}

	// 4. 生成向量嵌入
	vector, err := idx.embedder.GetEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("生成向量嵌入失败: %w", err)
	}

	// 转换为float32
	vector32 := make([]float32, len(vector))
	for i, v := range vector {
		vector32[i] = float32(v)
	}

	// 5. 存入向量库
	vectorID, err := idx.vectorStore.Add(ctx, vector32, content)
	if err != nil {
		return fmt.Errorf("存入向量库失败: %w", err)
	}

	// 6. 记录元数据
	doc := ScanResultToDocument(scanResult, content, hash)
	doc.VectorID = vectorID
	doc.Status = StatusIndexed

	if err := idx.metaStore.AddDocument(doc); err != nil {
		return fmt.Errorf("记录文档元数据失败: %w", err)
	}

	logger.Info("文档索引完成", map[string]interface{}{
		"path":      scanResult.FilePath,
		"vector_id": vectorID,
		"hash":      hash,
	})

	return nil
}

// IndexDocuments 批量索引文档
// ctx: 上下文
// results: 扫描结果列表
// sleepMs: 处理间隔（毫秒，用于控制CPU）
// 返回成功数和错误数
func (idx *Indexer) IndexDocuments(ctx context.Context, results []*ScanResult, sleepMs int) (successCount, failCount int) {
	for i, result := range results {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			logger.Warn("索引任务被取消", map[string]interface{}{
				"processed": i,
				"total":     len(results),
			})
			return successCount, failCount + len(results) - i
		default:
		}

		// 索引文档
		if err := idx.IndexDocument(ctx, result); err != nil {
			logger.Error("索引文档失败", map[string]interface{}{
				"path":  result.FilePath,
				"error": err.Error(),
			})
			failCount++
		} else {
			successCount++
		}

		// 控制CPU：处理间隔
		if sleepMs > 0 && i < len(results)-1 {
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
		}
	}

	return successCount, failCount
}

// readFileContent 读取文件内容
// filePath: 文件路径
// extension: 文件扩展名
// 返回文件内容或错误
func (idx *Indexer) readFileContent(filePath string, extension string) (string, error) {
	switch extension {
	case ".txt", ".md":
		return idx.readTextFile(filePath)
	case ".pdf":
		return idx.readPDFFile(filePath)
	case ".docx":
		return idx.readDocxFile(filePath)
	default:
		return "", fmt.Errorf("不支持的文件格式: %s", extension)
	}
}

// readTextFile 读取文本文件
// filePath: 文件路径
// 返回文件内容或错误
func (idx *Indexer) readTextFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文本文件失败: %w", err)
	}

	content := string(data)

	// 清理内容
	content = cleanContent(content)

	logger.Debug("读取文本文件成功", map[string]interface{}{
		"path": filePath,
		"size": len(content),
	})

	return content, nil
}

// readPDFFile 读取PDF文件
// filePath: 文件路径
// 返回文件内容或错误
func (idx *Indexer) readPDFFile(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开PDF文件失败: %w", err)
	}
	defer f.Close()

	var content strings.Builder

	// 读取所有页面
	totalPage := r.NumPage()
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		// 提取文本
		text, err := p.GetPlainText(nil)
		if err != nil {
			logger.Warn("提取PDF页面文本失败", map[string]interface{}{
				"page":  pageIndex,
				"error": err.Error(),
			})
			continue
		}

		content.WriteString(text)
		content.WriteString("\n")
	}

	result := cleanContent(content.String())

	logger.Debug("读取PDF文件成功", map[string]interface{}{
		"path":      filePath,
		"pages":     totalPage,
		"text_size": len(result),
	})

	return result, nil
}

// readDocxFile 读取Word文档
// 使用 github.com/fumiama/go-docx 库解析 DOCX 文件
// filePath: 文件路径
// 返回文件内容或错误
func (idx *Indexer) readDocxFile(filePath string) (string, error) {
	// 打开DOCX文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开DOCX文件失败: %w", err)
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 解析DOCX
	doc, err := docx.Parse(file, fileInfo.Size())
	if err != nil {
		return "", fmt.Errorf("解析DOCX文件失败: %w", err)
	}

	var content strings.Builder

	// 遍历文档Body中的所有项目
	for _, item := range doc.Document.Body.Items {
		// 尝试转换为字符串并提取文本
		itemStr := fmt.Sprintf("%v", item)
		// 使用正则或字符串处理提取文本内容
		// 这里简化处理，直接追加
		if itemStr != "" && itemStr != "<nil>" {
			content.WriteString(itemStr)
			content.WriteString("\n")
		}
	}

	result := cleanContent(content.String())

	logger.Debug("读取DOCX文件成功", map[string]interface{}{
		"path":      filePath,
		"text_size": len(result),
	})

	return result, nil
}

// isValidDocx 检查文件是否是有效的DOCX文件（ZIP格式）
// filePath: 文件路径
// 返回是否是有效的DOCX
func isValidDocx(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// 读取文件头（ZIP文件的魔数是 0x50 0x4B 0x03 0x04）
	header := make([]byte, 4)
	_, err = file.Read(header)
	if err != nil {
		return false
	}

	// 检查ZIP魔数
	return header[0] == 0x50 && header[1] == 0x4B && header[2] == 0x03 && header[3] == 0x04
}

// calculateHash 计算内容哈希
// content: 文件内容
// 返回MD5哈希字符串
func calculateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

// cleanContent 清理内容
// 去除多余空白、控制字符等
// content: 原始内容
// 返回清理后的内容
func cleanContent(content string) string {
	// 去除BOM
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")

	// 替换多种空白为单个空格
	content = strings.Join(strings.Fields(content), " ")

	// 去除首尾空白
	content = strings.TrimSpace(content)

	return content
}

// GetStats 获取索引统计
// 返回统计信息
func (idx *Indexer) GetStats() (*Stats, error) {
	return idx.metaStore.GetStats()
}
