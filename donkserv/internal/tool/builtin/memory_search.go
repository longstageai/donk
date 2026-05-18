package builtin

import (
	"strings"

	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/tool"
)

// MemorySearcher 记忆检索工具
// Agent 可以调用此工具从长期记忆中检索信息
// 支持语义搜索和关键词过滤
type MemorySearcher struct {
	longMemory *memory.LongMemory // 长期记忆存储实例
}

// MemorySearcherOption 记忆检索器配置选项
// 允许在创建工具时进行个性化配置
type MemorySearcherOption func(*MemorySearcher)

// NewMemorySearcher 创建记忆检索工具
// 初始化一个可用于检索长期记忆的工具实例
//
// 参数:
//
//	longMemory: 长期记忆存储实例，用于执行搜索查询
//	opts: 可选的配置选项
//
// 返回:
//
//	*MemorySearcher: 记忆检索工具实例
func NewMemorySearcher(longMemory *memory.LongMemory, opts ...MemorySearcherOption) *MemorySearcher {
	s := &MemorySearcher{
		longMemory: longMemory,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name 返回工具名称
//
// 返回:
//
//	string: 工具名称 "memory_search"
func (m *MemorySearcher) Name() string {
	return "memory_search"
}

// Description 返回工具描述
//
// 返回:
//
//	string: 工具功能描述
func (m *MemorySearcher) Description() string {
	return "从长期记忆中检索相关信息"
}

// Version 返回版本
//
// 返回:
//
//	string: 工具版本号
func (m *MemorySearcher) Version() string {
	return "1.0.0"
}

// Category 返回分类
//
// 返回:
//
//	string: 工具分类（工具类）
func (m *MemorySearcher) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回参数定义
// 定义了工具所需的输入参数规范
//
// 返回:
//
//	*tool.Schema: JSON Schema 格式的参数定义
func (m *MemorySearcher) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"query": {
			Type:        "string",
			Description: "语义搜索查询语句（支持语义理解）",
		},
		"keywords": {
			Type:        "string",
			Description: "搜索关键词（多个用逗号分隔），当query为空时使用",
		},
		"tags": {
			Type:        "string",
			Description: "按标签筛选（多个用逗号分隔）",
		},
		"limit": {
			Type:        "integer",
			Description: "返回结果数量限制，默认 5",
			Default:     5,
		},
	}
	return schema
}

// Execute 执行记忆检索
// 根据用户提供的查询条件搜索长期记忆
// 支持语义搜索、关键词过滤和标签筛选
//
// 参数:
//
//	ctx: 工具执行上下文，包含参数和运行时值
//
// 返回:
//
//	*tool.Result: 执行结果
//	error: 执行过程中的错误（返回nil表示无错误）
func (m *MemorySearcher) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 获取语义查询语句
	query, hasQuery := ctx.Params["query"].(string)

	// 解析关键词列表
	var keywords []string
	if kwStr, ok := ctx.Params["keywords"].(string); ok && kwStr != "" {
		keywords = strings.Split(kwStr, ",")
		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}
	}

	// 如果提供了query但没有keywords，将query作为关键词
	if hasQuery && query != "" && len(keywords) == 0 {
		keywords = []string{query}
	}

	// 解析标签筛选条件
	var tags []string
	if tagsStr, ok := ctx.Params["tags"].(string); ok && tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// 获取返回结果数量限制
	limit := 5
	if l, ok := ctx.Params["limit"].(float64); ok {
		limit = int(l)
	}

	// 构建搜索请求
	req := memory.SearchRequest{
		Keywords:   keywords,
		Tags:       tags,
		MemoryType: memory.MemoryTypeLong,
		Limit:      limit,
	}

	// 执行搜索
	result, err := m.longMemory.Search(req)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "搜索记忆失败: "+err.Error()), nil
	}

	// 格式化搜索结果
	var results []map[string]any
	for _, entry := range result.Entries {
		results = append(results, map[string]any{
			"key":       entry.Key,
			"summary":   entry.Summary,
			"content":   entry.Content,
			"keywords":  entry.Keywords,
			"tags":      entry.Tags,
			"timestamp": entry.Timestamp.Format("2006-01-02 15:04:05"),
			"score":     entry.Score,
		})
	}

	// 没有找到相关记忆
	if len(results) == 0 {
		return tool.NewResult(map[string]any{
			"total":   0,
			"results": []map[string]any{},
			"message": "未找到相关记忆",
		}), nil
	}

	return tool.NewResult(map[string]any{
		"total":   result.Total,
		"results": results,
	}), nil
}
