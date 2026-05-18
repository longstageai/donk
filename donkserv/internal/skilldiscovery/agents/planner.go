// agents 技能自动发现 Agent 模块
// Planner Agent 负责规划技能结构和内容
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

// PlannerAgent 技能规划 Agent
// 负责设计技能的完整结构，包括指令、工具、示例等
type PlannerAgent struct {
	llm    model.LLM
	prompt string
}

// NewPlannerAgent 创建规划 Agent
// 参数:
//   - llm: LLM 接口
//   - prompt: 规划 Prompt（可选，为空使用默认）
//
// 返回:
//   - *PlannerAgent: Agent 实例
func NewPlannerAgent(llm model.LLM, prompt string) *PlannerAgent {
	if prompt == "" {
		prompt = defaultPlannerPrompt
	}
	return &PlannerAgent{
		llm:    llm,
		prompt: prompt,
	}
}

// Plan 规划技能
// 参数:
//   - ctx: 上下文
//   - candidate: 技能候选
//
// 返回:
//   - *SkillPlan: 技能规划
//   - error: 错误信息
func (p *PlannerAgent) Plan(ctx context.Context, candidate *SkillCandidate) (*SkillPlan, error) {
	plan, _, err := p.PlanWithUsage(ctx, candidate)
	return plan, err
}

// PlanWithUsage 规划技能并返回 Token 使用情况
// 参数:
//   - ctx: 上下文
//   - candidate: 技能候选
//
// 返回:
//   - *SkillPlan: 技能规划
//   - *schema.UsageInfo: Token 使用信息
//   - error: 错误信息
func (p *PlannerAgent) PlanWithUsage(ctx context.Context, candidate *SkillCandidate) (*SkillPlan, *schema.UsageInfo, error) {
	logger.Info("开始规划技能", map[string]interface{}{
		"candidate_name": candidate.Name,
	})

	// 构建 Prompt
	prompt := fmt.Sprintf(p.prompt,
		candidate.Name,
		candidate.Description,
		candidate.Trigger,
	)

	// 使用 model.LLM 接口的 Chat 方法
	req := &schema.ChatRequest{
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	response, err := p.llm.Chat(ctx, req)
	if err != nil {
		logger.Error("技能规划失败", map[string]interface{}{
			"candidate_name": candidate.Name,
			"error":          err.Error(),
		})
		return nil, nil, fmt.Errorf("技能规划失败: %w", err)
	}

	logger.Debug("技能规划 LLM 响应完成", map[string]interface{}{
		"candidate_name":    candidate.Name,
		"response_length":   len(response.Content),
		"prompt_tokens":     response.Usage.PromptTokens,
		"completion_tokens": response.Usage.CompletionTokens,
	})

	// 解析结果
	plan, err := p.parseResponse(response.Content)
	if err != nil {
		logger.Error("解析规划结果失败", map[string]interface{}{
			"candidate_name": candidate.Name,
			"error":          err.Error(),
		})
		return nil, nil, fmt.Errorf("解析规划结果失败: %w", err)
	}

	// 确保名称一致
	plan.Name = candidate.Name

	logger.Info("技能规划完成", map[string]interface{}{
		"skill_name":     plan.Name,
		"tools_count":    len(plan.AllowedTools),
		"examples_count": len(plan.Examples),
	})

	return plan, &response.Usage, nil
}

// parseResponse 解析 LLM 响应
// 参数:
//   - response: LLM 响应文本
//
// 返回:
//   - *SkillPlan: 技能规划
//   - error: 错误信息
func (p *PlannerAgent) parseResponse(response string) (*SkillPlan, error) {
	// 提取 JSON
	jsonStr := p.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("未能从响应中提取 JSON")
	}

	// 解析 JSON
	var result struct {
		Name         string         `json:"name"`
		Description  string         `json:"description"`
		Instructions string         `json:"instructions"`
		Tags         []string       `json:"tags"`
		AllowedTools []string       `json:"allowed_tools"`
		Examples     []string       `json:"examples"`
		Metadata     map[string]any `json:"metadata"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 验证必要字段
	if result.Name == "" || result.Description == "" || result.Instructions == "" {
		return nil, fmt.Errorf("规划结果缺少必要字段")
	}

	return &SkillPlan{
		Name:         result.Name,
		Description:  result.Description,
		Instructions: result.Instructions,
		Tags:         result.Tags,
		AllowedTools: result.AllowedTools,
		Examples:     result.Examples,
		Metadata:     result.Metadata,
	}, nil
}

// extractJSON 从文本中提取 JSON
// 参数:
//   - text: 包含 JSON 的文本
//
// 返回:
//   - string: 提取的 JSON 字符串
func (p *PlannerAgent) extractJSON(text string) string {
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

// defaultPlannerPrompt 默认规划 Prompt
const defaultPlannerPrompt = `
#你是专业的技能规划师，擅长将创意技能设计为结构清晰、可执行的完整技能。

## 输入
为给定的技能设计完整的结构和内容，确保技能易于理解和使用。

## 设计任务
1. 核心指令：
   - 详细执行步骤
   - 判断逻辑和分支处理
   - 输出格式
   - 错误处理
2. 标签：3-5 个相关分类标签
3. 允许工具：列出技能可能需要调用的工具，如 file_reader, file_writer
4. 使用示例：2-3 个具体触发场景和期望输出
5. 元数据：author, version, 可自定义字段

## 方法提示
- 避免重复已有技能（past_skills）
- 可调用联网或向量数据库获取补充信息
- instructions 要具体且可执行

## 输出格式（JSON）
{
  "name": "技能名称",
  "description": "技能描述",
  "instructions": "Markdown格式，包含执行步骤、逻辑、示例",
  "tags": ["tag1","tag2","tag3"],
  "allowed_tools": ["file_reader","file_writer"],
  "examples": [
    "示例1：用户说...",
    "示例2：用户说..."
  ],
  "metadata": {
    "author": "auto-discovery",
    "version": "1.0.0"
  }
}

注意：
- instructions 尽量详细，可直接执行
- 工具必须从 allowed_tools 选择
- 确保技能创新、有价值且可落地
`
