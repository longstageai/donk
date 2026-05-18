package builtin

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/longstageai/donk/donk/internal/memory"
	"github.com/longstageai/donk/donk/internal/tool"
)

// MemorySaver 记忆保存工具
// Agent 可以调用此工具将重要信息保存到长期记忆
type MemorySaver struct {
	longMemory *memory.LongMemory // 长期记忆存储实例
}

// NewMemorySaver 创建记忆保存工具
// 初始化一个可用于保存长期记忆的工具实例
//
// 参数:
//
//	longMemory: 长期记忆存储实例，用于持久化保存记忆
//
// 返回:
//
//	*MemorySaver: 记忆保存工具实例
func NewMemorySaver(longMemory *memory.LongMemory) *MemorySaver {
	return &MemorySaver{
		longMemory: longMemory,
	}
}

// Name 返回工具名称
//
// 返回:
//
//	string: 工具名称 "memory_save"
func (m *MemorySaver) Name() string {
	return "memory_save"
}

// Description 返回工具描述
//
// 返回:
//
//	string: 工具功能描述
func (m *MemorySaver) Description() string {
	return "保存重要信息到长期记忆，供以后检索使用"
}

// Version 返回版本
//
// 返回:
//
//	string: 工具版本号
func (m *MemorySaver) Version() string {
	return "1.0.0"
}

// Category 返回分类
//
// 返回:
//
//	string: 工具分类（工具类）
func (m *MemorySaver) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回参数定义
// 定义了工具所需的输入参数规范
//
// 返回:
//
//	*tool.Schema: JSON Schema 格式的参数定义
func (m *MemorySaver) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"content": {
			Type:        "string",
			Description: "要保存的内容",
		},
		"summary": {
			Type:        "string",
			Description: "内容摘要（可选，不填则自动提取）",
		},
		"keywords": {
			Type:        "string",
			Description: "关键词（多个用逗号分隔，可选）",
		},
		"tags": {
			Type:        "string",
			Description: "标签列表（多个用逗号分隔），便于后续检索",
		},
	}
	schema.Required = []string{"content"}
	return schema
}

// Execute 执行记忆保存
// 将用户提供的记忆内容保存到长期存储中
//
// 参数:
//
//	ctx: 工具执行上下文，包含参数和运行时值
//
// 返回:
//
//	*tool.Result: 执行结果
//	error: 执行过程中的错误（返回nil表示无错误）
func (m *MemorySaver) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 获取并验证记忆内容
	content, ok := ctx.Params["content"].(string)
	if !ok || content == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "记忆内容不能为空"), nil
	}

	// 获取摘要（可选）
	var summary string
	if summaryVal, ok := ctx.Params["summary"].(string); ok && summaryVal != "" {
		summary = summaryVal
	} else {
		// 自动生成摘要：取内容前50字
		if len(content) > 50 {
			summary = content[:50] + "..."
		} else {
			summary = content
		}
	}

	// 解析关键词（可选）
	var keywords []string
	if kwStr, ok := ctx.Params["keywords"].(string); ok && kwStr != "" {
		keywords = strings.Split(kwStr, ",")
		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}
	}

	// 解析标签列表
	var tags []string
	if tagsStr, ok := ctx.Params["tags"].(string); ok && tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// 自动生成唯一标识
	key := uuid.New().String()

	// 创建记忆条目
	entry := memory.NewMemoryEntry(key, content, memory.MemoryTypeLong)
	entry.Summary = summary
	entry.Keywords = keywords
	entry.AddTag(tags...)
	entry.Metadata.Source = "agent"
	entry.Timestamp = time.Now()

	// 保存到长期记忆存储
	if err := m.longMemory.Save(entry); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "保存记忆失败: "+err.Error()), nil
	}

	return tool.NewResult(map[string]any{
		"key":      key,
		"content":  content,
		"summary":  summary,
		"keywords": keywords,
		"tags":     tags,
		"message":  "已保存到长期记忆",
	}), nil
}
