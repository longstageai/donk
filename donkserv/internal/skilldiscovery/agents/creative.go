// agents 技能自动发现 Agent 模块
// Creative Agent 负责在无需求时生成创意技能
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// CreativeAgent 创意生成 Agent
// 当对话分析没有发现有价值的技能需求时，自动生成有趣实用的技能创意
type CreativeAgent struct {
	llm    model.LLM
	prompt string
}

// NewCreativeAgent 创建创意 Agent
// 参数:
//   - llm: LLM 接口
//   - prompt: 创意生成 Prompt（可选，为空使用默认）
//
// 返回:
//   - *CreativeAgent: Agent 实例
func NewCreativeAgent(llm model.LLM, prompt string) *CreativeAgent {
	if prompt == "" {
		prompt = defaultCreativePrompt
	}
	return &CreativeAgent{
		llm:    llm,
		prompt: prompt,
	}
}

// Generate 生成创意技能
// 参数:
//   - ctx: 上下文
//   - count: 生成数量
//
// 返回:
//   - []*SkillCandidate: 创意技能候选列表
func (a *CreativeAgent) Generate(ctx context.Context, count int) []*SkillCandidate {
	if count <= 0 {
		count = 1
	}

	logger.Info("开始生成创意技能", map[string]interface{}{
		"count": count,
	})

	candidates := make([]*SkillCandidate, 0, count)

	for i := 0; i < count; i++ {
		candidate, err := a.generateOne(ctx)
		if err != nil {
			logger.Error("生成创意技能失败", map[string]interface{}{
				"index": i,
				"error": err.Error(),
			})
			continue
		}

		candidates = append(candidates, candidate)
		logger.Info("生成创意技能", map[string]interface{}{
			"name":        candidate.Name,
			"description": candidate.Description,
		})
	}

	logger.Info("创意技能生成完成", map[string]interface{}{
		"generated_count": len(candidates),
	})

	return candidates
}

// generateOne 生成单个创意技能
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - *SkillCandidate: 创意技能候选
//   - error: 错误信息
func (a *CreativeAgent) generateOne(ctx context.Context) (*SkillCandidate, error) {
	// 使用 model.LLM 接口的 Chat 方法
	req := &schema.ChatRequest{
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: a.prompt,
			},
		},
	}

	response, err := a.llm.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM 生成失败: %w", err)
	}

	candidate, err := a.parseResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("解析生成结果失败: %w", err)
	}

	// 标记为创意生成
	candidate.Confidence = 0.7
	candidate.Evidence = []string{"创意生成"}

	return candidate, nil
}

// parseResponse 解析 LLM 响应
// 参数:
//   - response: LLM 响应文本
//
// 返回:
//   - *SkillCandidate: 候选技能
//   - error: 错误信息
func (a *CreativeAgent) parseResponse(response string) (*SkillCandidate, error) {
	// 提取 JSON 部分
	jsonStr := a.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("未能从响应中提取 JSON")
	}

	// 解析 JSON
	var result struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Trigger     string `json:"trigger"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 验证必要字段
	if result.Name == "" || result.Description == "" {
		return nil, fmt.Errorf("生成结果缺少必要字段")
	}

	return &SkillCandidate{
		Name:        result.Name,
		Description: result.Description,
		Trigger:     result.Trigger,
	}, nil
}

// extractJSON 从文本中提取 JSON
// 参数:
//   - text: 包含 JSON 的文本
//
// 返回:
//   - string: 提取的 JSON 字符串
func (a *CreativeAgent) extractJSON(text string) string {
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.LastIndex(text, "}")
	if endIdx == -1 || endIdx <= startIdx {
		return ""
	}

	return text[startIdx : endIdx+1]
}

// defaultCreativePrompt 默认创意生成 Prompt
const defaultCreativePrompt = `# Creative Agent Prompt

你是一个富有创造力的 AI 助手，擅长设计有趣且实用的技能。

## 设计原则

1. **有趣且实用**
   - 能够解决实际问题或提升效率
   - 功能明确，易于理解和使用
   - 避免与常见的通用技能重复

2. **创新性**
   - 可以是全新的创意
   - 也可以是对现有功能的创新改进
   - 鼓励跨界组合不同功能

3. **可实现性**
   - 功能要在当前技术能力范围内
   - 避免过于复杂或依赖外部不可控资源

## 创意方向

### 数据处理类
- 格式化、转换、分析各类数据
- 批量处理文件或数据
- 数据清洗和整理

### 内容创作类
- 辅助写作、润色文字
- 生成创意内容
- 内容结构化处理

### 效率工具类
- 自动化重复任务
- 批量处理操作
- 工作流程优化

### 学习辅助类
- 知识整理和归纳
- 记忆辅助工具
- 学习规划和追踪

### 生活助手类
- 日程管理和提醒
- 习惯追踪
- 决策辅助

## 输出格式

请以 JSON 格式输出：

` + "```json" + `
{
  "name": "skill-name",
  "description": "技能的详细描述，说明它能做什么、解决什么问题、为用户带来什么价值",
  "trigger": "触发场景，说明用户什么时候会需要这个技能"
}
` + "```" + `

## 注意事项

1. 只输出 JSON，不要输出其他内容
2. 技能名称使用英文小写，单词间用短横线连接
3. 描述要具体，让用户明白这个技能的价值
4. 避免创建过于简单或过于复杂的技能
5. 每次生成一个独特的创意

请设计一个创新的技能，以 JSON 格式输出。`
