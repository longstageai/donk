package profile

import (
	"context"
	"strings"
	"text/template"

	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/pkg/schema"
)

const (
	// AnalyzePrompt LLM 分析画像的 Prompt 模板
	AnalyzePrompt = `
# 用户画像精准更新提示词
## 任务
基于**现有用户画像**和**本次对话内容**，进行**增量式、精准化**的用户画像更新。核心目标是**只保留长期稳定、对后续交互有高价值的用户特征**，严格过滤所有临时、琐碎、无复用价值的噪音信息，确保画像始终简洁、准确、有用，能够直接指导AI生成精准的回答。

## 现有画像
{{.ExistingProfile}}

## 本次对话
{{.Dialog}}

## 核心要求
### 一、必须优先记录的高价值稳定信息（按优先级排序）
1. **身份与背景**：职业、职位、行业、学历、年龄范围、所在地区、核心身份标签
2. **负面偏好与绝对禁忌**：明确拒绝的内容、讨厌的表达方式、禁止的话题、曾经纠正过的错误
3. **目标与行动**：长期目标、短期任务、当前正在推进的项目、待解决的核心问题（标注有效期）
4. **能力与认知**：技能水平、专业知识领域、技术栈、经验背景、认知水平
5. **偏好与习惯**：沟通风格偏好、内容偏好、工具使用习惯、决策方式、长期习惯
6. **语言与表达**：母语、外语水平、偏好的回答风格、偏好的输出格式
7. **态度与关注点**：稳定的情绪倾向、价值观、核心关注点、长期兴趣爱好
8. **历史反馈**：曾经满意或不满意的回答类型、曾经要求遵循的规则
9. **需求与痛点**：反复提到的需求、核心痛点、期望获得的帮助类型

### 二、绝对禁止记录的噪音信息（严格执行）
1. 一次性日常琐事（如"今天喝了咖啡"、"刚才堵车了"）
2. 临时情绪宣泄（如"今天好烦"、"气死我了"，除非是持续稳定的情绪状态）
3. 对话中的客套话、寒暄语、语气词、感叹词
4. 用户随口提及的无关话题、未确认的想法、临时的疑问
5. 与用户无关的任何内容
6. 重复冗余的信息、已经记录过且未发生变化的内容
7. 任何需要猜测、推断的信息（必须100%基于用户明确表述）
8. 对用户的主观评价（如"用户很聪明"、"用户很友好"）
9. 过于笼统或过于琐碎的信息
10. 用户开玩笑、说反话或使用隐喻的内容

### 三、画像更新执行规则
1. **增量更新**：完整保留现有画像中所有仍然有效的信息，仅基于本次对话补充新信息、修正过时信息、删除已失效信息
2. **时效性管理**：所有临时信息必须标注明确有效期，超过有效期的信息自动删除
3. **置信度管理**：区分确定信息、可能信息和计划信息，可能信息和计划信息必须标注
4. **冲突处理**：当信息冲突时，遵循"本次>历史、确定>不确定、具体>笼统"的原则
5. **简洁表达**：用简短的陈述句表达，每条信息独立一行
6. **准确客观**：严格照搬用户核心表述，不添加任何个人解读、引申或润色
7. **结构化组织**：所有信息按优先级从高到低排序，同一类别信息放在一起，用空行分隔
8. **无新信息直接返回**：如果本次对话没有产生任何符合上述高价值标准的新信息，直接完整返回现有画像即可

## 输出格式
直接输出最终的用户画像文本内容，纯文本格式，无需JSON、无需标题、无需任何额外说明。每条用户特征独立一行，便于快速阅读。
`
)

// IAnalyzer 画像分析器接口
// 用于分析对话内容，生成用户画像
type IAnalyzer interface {
	// Analyze 分析对话并生成画像
	//
	// 参数:
	//   ctx - 上下文
	//   dialog - 对话历史文本
	//   existingProfile - 现有画像内容（可能为空）
	//
	// 返回:
	//   string - 生成的新画像内容（纯文本）
	//   error - 错误信息
	Analyze(ctx context.Context, dialog string, existingProfile string) (string, error)
}

// LLMAnalyzer 基于 LLM 的画像分析器
// 调用大模型分析对话内容，生成用户画像
type LLMAnalyzer struct {
	model  model.Adapter // LLM 模型适配器
	prompt *template.Template
}

// LLMAnalyzerOption LLM 分析器配置选项
type LLMAnalyzerOption func(*LLMAnalyzer)

// NewLLMAnalyzer 创建 LLM 分析器
//
// 参数:
//
//	model: LLM 模型适配器
//	opts: 可选配置
//
// 返回:
//
//	*IAnalyzer: 画像分析器接口
func NewLLMAnalyzer(model model.Adapter, opts ...LLMAnalyzerOption) IAnalyzer {
	a := &LLMAnalyzer{
		model: model,
	}

	// 默认模板
	tmpl, err := template.New("profile").Parse(AnalyzePrompt)
	if err != nil {
		tmpl, _ = template.New("profile").Parse(AnalyzePrompt)
	}
	a.prompt = tmpl

	// 应用选项
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Analyze 分析对话并生成画像
// 调用 LLM 分析对话内容，结合现有画像生成新的画像
//
// 参数:
//
//	ctx: 上下文
//	dialog: 对话历史文本
//	existingProfile: 现有画像内容
//
// 返回:
//
//	string: 生成的新画像内容
//	error: 错误信息
func (a *LLMAnalyzer) Analyze(ctx context.Context, dialog string, existingProfile string) (string, error) {
	// 构建提示词
	var sb strings.Builder
	err := a.prompt.Execute(&sb, map[string]string{
		"ExistingProfile": existingProfile,
		"Dialog":          dialog,
	})
	if err != nil {
		return "", err
	}

	// 调用 LLM
	req := &schema.ChatRequest{
		Messages: []schema.Message{
			{Role: "user", Content: sb.String()},
		},
		Temperature: 0.5,
		MaxTokens:   1000,
	}

	resp, err := a.model.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}
