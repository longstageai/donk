package builtin

import (
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/conversation"
	"github.com/longstageai/donk/donk/internal/tool"
)

// ConversationSearchTool 对话历史搜索工具
// 从已向量化的对话历史中检索相关信息
type ConversationSearchTool struct {
	manager *conversation.Manager // 对话历史管理器
}

// NewConversationSearchTool 创建对话历史搜索工具
//
// 参数:
//
//	manager: 对话历史管理器
//
// 返回:
//
//	*ConversationSearchTool: 对话历史搜索工具实例
func NewConversationSearchTool(manager *conversation.Manager) *ConversationSearchTool {
	return &ConversationSearchTool{
		manager: manager,
	}
}

// Name 返回工具名称
func (k *ConversationSearchTool) Name() string {
	return "conversation_search"
}

// Description 返回工具描述
func (k *ConversationSearchTool) Description() string {
	return "从对话历史中检索相关信息（语义搜索）"
}

// Version 返回版本
func (k *ConversationSearchTool) Version() string {
	return "1.0.0"
}

// Category 返回分类
func (k *ConversationSearchTool) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回参数定义
func (k *ConversationSearchTool) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"query": {
			Type:        "string",
			Description: "搜索查询内容",
		},
		"top_k": {
			Type:        "integer",
			Description: "返回结果数量（默认5）",
		},
		"start_time": {
			Type:        "string",
			Description: "开始时间（RFC3339格式，如 2026-03-01T00:00:00Z）",
		},
		"end_time": {
			Type:        "string",
			Description: "结束时间（RFC3339格式，如 2026-03-26T23:59:59Z）",
		},
	}
	schema.Required = []string{"query"}
	return schema
}

// Execute 执行工具
// 从对话历史中检索相关信息
//
// 参数:
//
//	ctx: 工具上下文
//
// 返回:
//
//	*tool.Result: 执行结果
//	error: 错误信息
func (k *ConversationSearchTool) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 解析参数
	params, err := conversation.ParseSearchParams(ctx.Params)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
	}

	// 构建时间过滤条件
	var timeFilter *conversation.TimeFilter
	if params.StartTime != nil || params.EndTime != nil {
		timeFilter = &conversation.TimeFilter{
			StartTime: params.StartTime,
			EndTime:   params.EndTime,
		}
	}

	// 执行搜索
	results, err := k.manager.Search(ctx.Values, params.Query, params.TopK, timeFilter)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("搜索失败: %v", err)), nil
	}

	// 格式化结果
	if len(results) == 0 {
		return tool.NewResult(map[string]any{
			"message": "未找到相关内容",
		}), nil
	}

	var sb strings.Builder
	sb.WriteString("检索结果：\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("【结果 %d】相似度: %.2f%%\n", i+1, r.Score*100))
		sb.WriteString(fmt.Sprintf("时间: %s\n", r.Date))
		sb.WriteString(fmt.Sprintf("内容:\n%s\n\n", r.Content))
	}

	return tool.NewResult(map[string]any{
		"message": sb.String(),
	}), nil
}
