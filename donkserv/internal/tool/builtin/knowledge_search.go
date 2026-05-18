// builtin 内置工具包
package builtin

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/embedding"
	"github.com/longstageai/donk/donk/internal/knowledge"
	"github.com/longstageai/donk/donk/internal/tool"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// KnowledgeSearcher 知识库搜索工具
// Agent 可以调用此工具从知识库中检索相关文档
// 支持语义搜索和关键词过滤
type KnowledgeSearcher struct {
	dataDir  string             // 数据目录路径
	dbPath   string             // 数据库文件路径
	embedder embedding.Embedder // 嵌入器实例
}

// KnowledgeSearcherOption 知识库搜索器配置选项
// 允许在创建工具时进行个性化配置
type KnowledgeSearcherOption func(*KnowledgeSearcher)

// NewKnowledgeSearcher 创建知识库搜索工具
// 初始化一个可用于检索知识库文档的工具实例
//
// 参数:
//   - dataDir: 数据目录路径
//   - dbPath: 数据库文件路径
//   - embedder: 嵌入器实例
//   - opts: 可选的配置选项
//
// 返回:
//   - *KnowledgeSearcher: 知识库搜索工具实例
func NewKnowledgeSearcher(dataDir string, dbPath string, embedder embedding.Embedder, opts ...KnowledgeSearcherOption) *KnowledgeSearcher {
	s := &KnowledgeSearcher{
		dataDir:  dataDir,
		dbPath:   dbPath,
		embedder: embedder,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Name 返回工具名称
//
// 返回:
//   - string: 工具名称 "knowledge_search"
func (k *KnowledgeSearcher) Name() string {
	return "knowledge_search"
}

// Description 返回工具描述
//
// 返回:
//   - string: 工具功能描述
func (k *KnowledgeSearcher) Description() string {
	return "从知识库中获取用户隐私信息"
}

// Version 返回版本
//
// 返回:
//   - string: 工具版本号
func (k *KnowledgeSearcher) Version() string {
	return "1.0.0"
}

// Category 返回分类
//
// 返回:
//   - string: 工具分类
func (k *KnowledgeSearcher) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回参数定义
// 定义了工具所需的输入参数规范
//
// 返回:
//   - *tool.Schema: JSON Schema 格式的参数定义
func (k *KnowledgeSearcher) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"query": {
			Type:        "string",
			Description: "搜索查询语句（支持语义理解）",
		},
		"keywords": {
			Type:        "string",
			Description: "关键词过滤（多个用逗号分隔）",
		},
		"limit": {
			Type:        "integer",
			Description: "返回结果数量限制，默认 5，最大 20",
			Default:     5,
		},
		"file_type": {
			Type:        "string",
			Description: "文件类型过滤，如 .txt, .md, .pdf,.docx",
		},
	}
	return schema
}

// Execute 执行知识库搜索
// 根据用户提供的查询条件搜索知识库文档
// 支持语义搜索和关键词过滤
//
// 参数:
//   - ctx: 工具执行上下文，包含参数和运行时值
//
// 返回:
//   - *tool.Result: 执行结果
//   - error: 执行过程中的错误
func (k *KnowledgeSearcher) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 获取查询语句
	query, _ := ctx.Params["query"].(string)
	if query == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "查询语句不能为空"), nil
	}

	// 获取返回结果数量限制
	limit := 5
	if l, ok := ctx.Params["limit"].(float64); ok {
		limit = int(l)
		if limit > 20 {
			limit = 20
		}
		if limit < 1 {
			limit = 1
		}
	}

	// 获取关键词过滤
	var keywords []string
	if kwStr, ok := ctx.Params["keywords"].(string); ok && kwStr != "" {
		keywords = splitKeywords(kwStr)
	}

	// 获取文件类型过滤
	fileType, _ := ctx.Params["file_type"].(string)

	logger.Info("执行知识库搜索", map[string]interface{}{
		"query":     query,
		"limit":     limit,
		"keywords":  keywords,
		"file_type": fileType,
	})

	// 创建向量存储
	vectorStore, err := knowledge.NewVectorStore(k.dataDir)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "创建向量存储失败: "+err.Error()), nil
	}

	// 创建 SQLite 存储
	sqliteStore, err := knowledge.NewSQLiteStore(k.dbPath)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "创建SQLite存储失败: "+err.Error()), nil
	}
	defer sqliteStore.Close()

	// 生成查询向量
	queryVector, err := k.embedder.GetEmbedding(context.Background(), query)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "生成查询向量失败: "+err.Error()), nil
	}

	// 转换为 float32
	queryVector32 := make([]float32, len(queryVector))
	for i, v := range queryVector {
		queryVector32[i] = float32(v)
	}

	// 执行向量搜索
	searchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := vectorStore.Search(searchCtx, queryVector32, limit*2) // 多搜索一些用于过滤
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "向量搜索失败: "+err.Error()), nil
	}

	// 格式化搜索结果
	var searchResults []map[string]any
	for _, result := range results {
		// 获取文档元数据
		doc, err := sqliteStore.GetDocumentByContent(result.Content)
		if err != nil {
			logger.Debug("获取文档元数据失败", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		if doc == nil {
			// 如果找不到元数据，使用搜索结果的内容
			doc = &knowledge.Document{
				Content: result.Content,
			}
		}

		// 关键词过滤
		if len(keywords) > 0 && !matchKeywords(doc.Content, keywords) {
			continue
		}

		// 文件类型过滤
		if fileType != "" && filepath.Ext(doc.FilePath) != fileType {
			continue
		}

		searchResults = append(searchResults, map[string]any{
			"file_path":    doc.FilePath,
			"content":      truncateString(doc.Content, 500),
			"score":        result.Score,
			"file_size":    doc.FileSize,
			"modified_at":  doc.ModifiedTime.Format("2006-01-02 15:04:05"),
			"access_count": doc.AccessCount,
		})

		// 更新访问次数
		if doc.FilePath != "" {
			sqliteStore.UpdateAccess(doc.FilePath)
		}

		if len(searchResults) >= limit {
			break
		}
	}

	// 构建返回结果
	resultData := map[string]any{
		"query":   query,
		"total":   len(searchResults),
		"results": searchResults,
	}

	return tool.NewResult(resultData), nil
}

// splitKeywords 分割关键词字符串
//
// 参数:
//   - kwStr: 关键词字符串，用逗号分隔
//
// 返回:
//   - []string: 关键词列表
func splitKeywords(kwStr string) []string {
	var keywords []string
	current := ""
	for _, r := range kwStr {
		if r == ',' || r == '，' {
			if current != "" {
				keywords = append(keywords, strings.TrimSpace(current))
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		keywords = append(keywords, strings.TrimSpace(current))
	}
	return keywords
}

// matchKeywords 检查内容是否包含关键词
//
// 参数:
//   - content: 文档内容
//   - keywords: 关键词列表
//
// 返回:
//   - bool: 是否匹配
func matchKeywords(content string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// truncateString 截断字符串
//
// 参数:
//   - s: 原字符串
//   - maxLen: 最大长度
//
// 返回:
//   - string: 截断后的字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
