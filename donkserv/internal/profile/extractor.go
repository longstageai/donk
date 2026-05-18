package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/longstageai/donk/donk/internal/message"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/pkg/logger"
	"github.com/longstageai/donk/donk/pkg/schema"
)

// 有效的标签类型
var validTagTypes = map[string]bool{
	"skill":      true,
	"interest":   true,
	"preference": true,
	"role":       true,
	"level":      true,
	"goal":       true,
	"tool":       true,
}

// MaxExtractMessages 最大提取消息数
// 与 trigger.MaxTriggerCount 保持一致
const MaxExtractMessages = 20

// ExtractionPrompt 用户画像提取提示词
// 全面提取用户技能、兴趣、偏好、背景等多维度信息
const ExtractionPrompt = `## 任务
从用户与AI的对话中提取用户画像信息，构建完整的用户画像档案。

## 输入对话
{{.Dialog}}

## 可提取的信息类型

### 1. 技能（skill）- type: "skill"
用户已具备的技术能力、专业知识或实操技能。
- ✅ 正面示例：
  - "我会Go编程，有3年经验" → Go, skill, 0.95
  - "我用Python做数据分析" → Python数据分析, skill, 0.90
  - "我熟悉Kubernetes和Docker" → Kubernetes, skill, 0.90; Docker, skill, 0.90
  - "我是前端开发工程师" → 前端开发, skill, 0.90
- ❌ 反面示例：
  - "我想学Rust" → 不提取（愿望≠技能）
  - "听说Go很火" → 不提取（泛泛而谈）
  - "我朋友会Java" → 不提取（他人技能）

### 2. 兴趣（interest）- type: "interest"
用户主动表达的关注领域、研究方向或业余爱好。
- ✅ 正面示例：
  - "我对AI很感兴趣" → AI, interest, 0.90
  - "我在研究分布式系统" → 分布式系统, interest, 0.85
  - "业余时间喜欢研究区块链技术" → 区块链, interest, 0.85
  - "最近在看机器学习相关的论文" → 机器学习, interest, 0.85
- ❌ 反面示例：
  - "AI很火啊" → 不提取（无个人关联）
  - "大家都聊元宇宙" → 不提取（跟风话题）

### 3. 偏好（preference）- type: "preference"
用户对交互方式、内容形式、沟通风格的明确要求。
- ✅ 正面示例：
  - "请详细解释" → response_style: detailed, preference, 0.90
  - "给我简洁的回答" → response_style: concise, preference, 0.90
  - "用代码示例说明" → explanation_style: code_example, preference, 0.90
  - "请用中文回答" → language: zh, preference, 0.95
  - "我喜欢图文并茂的解释" → explanation_style: visual, preference, 0.85

### 4. 职业/身份（role）- type: "role"
用户的职业身份、工作角色或专业背景。
- ✅ 正面示例：
  - "我是一名软件工程师" → 软件工程师, role, 0.95
  - "我在大厂做后端开发" → 后端开发工程师, role, 0.90
  - "我是计算机专业的大学生" → 计算机专业学生, role, 0.90
  - "我是技术团队负责人" → 技术负责人, role, 0.90

### 5. 经验水平（level）- type: "level"
用户在特定领域的经验程度或熟练度。
- ✅ 正面示例：
  - "我有5年Go开发经验" → Go开发: 专家, level, 0.95
  - "刚入门Python，还在学习" → Python: 初学者, level, 0.90
  - "我对微服务架构比较熟悉" → 微服务: 熟练, level, 0.85

### 6. 目标/需求（goal）- type: "goal"
用户明确表达的学习目标、项目需求或职业规划。
- ✅ 正面示例：
  - "我想学习云原生技术" → 学习云原生, goal, 0.90
  - "准备面试大厂" → 大厂面试准备, goal, 0.90
  - "想做一个个人博客项目" → 开发个人博客, goal, 0.85
  - "打算转行做AI" → 转行AI领域, goal, 0.90

### 7. 工具/环境（tool）- type: "tool"
用户使用的开发工具、IDE、操作系统或技术栈。
- ✅ 正面示例：
  - "我用VS Code写代码" → VS Code, tool, 0.90
  - "我的开发环境是Mac" → macOS, tool, 0.90
  - "我们用GitLab做CI/CD" → GitLab, tool, 0.85
  - "团队使用Jira管理项目" → Jira, tool, 0.85

## 提取原则

### 置信度评分标准
- 0.90-1.00：用户明确陈述（"我是程序员，用Go写了3年代码"）
- 0.75-0.89：强烈暗示或间接确认（"我每天写Go代码"）
- 0.60-0.74：合理推断但有上下文支持（"这个项目用Go重构的"）
- <0.60：不提取（置信度太低）

### 禁止提取的内容
- ❌ 猜测或推断的信息（无明确依据）
- ❌ 其他人的信息（"我同事会Java"）
- ❌ 临时性、一次性的需求（"帮我查一下今天的天气"）
- ❌ 过于笼统的描述（"我喜欢编程"）

### 证据要求
每个标签必须附带原文引用作为证据，格式："用户说：'[原文内容]'"

## 输出格式
严格返回JSON，不要包含任何其他说明文字：

{
  "tags": [
    {
      "name": "Go",
      "type": "skill",
      "confidence": 0.95,
      "evidence": "用户说：'我会Go编程，有3年经验'"
    },
    {
      "name": "AI",
      "type": "interest",
      "confidence": 0.90,
      "evidence": "用户说：'我对AI很感兴趣，经常看相关论文'"
    },
    {
      "name": "后端开发工程师",
      "type": "role",
      "confidence": 0.90,
      "evidence": "用户说：'我在大厂做后端开发'"
    }
  ],
  "preferences": {
    "response_style": "detailed",
    "language": "zh",
    "explanation_style": "code_example"
  },
  "confidence": 0.88
}`

// Extractor LLM信息提取器
type Extractor struct {
	model model.Adapter
}

// NewExtractor 创建提取器
func NewExtractor(model model.Adapter) *Extractor {
	return &Extractor{model: model}
}

// ExtractionResult 提取结果
type ExtractionResult struct {
	Tags        []ExtractedTag    `json:"tags"`
	Preferences map[string]string `json:"preferences"`
	Confidence  float64           `json:"confidence"`
	Usage       schema.UsageInfo  `json:"usage"` // Token 使用量
}

// ExtractedTag 提取的标签
type ExtractedTag struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// Extract 从消息中提取画像信息
//
// 参数:
//   - ctx: 上下文
//   - msgs: 消息列表
//
// 返回:
//   - *ExtractionResult: 提取结果
//   - error: 错误信息
func (e *Extractor) Extract(ctx context.Context, msgs []message.Message) (*ExtractionResult, error) {
	if len(msgs) == 0 {
		return &ExtractionResult{
			Tags:        []ExtractedTag{},
			Preferences: make(map[string]string),
			Confidence:  0,
		}, nil
	}

	// 限制分析长度，避免Token过多
	// 保留最新的消息，因为它们更相关
	if len(msgs) > MaxExtractMessages {
		msgs = msgs[len(msgs)-MaxExtractMessages:]
	}

	dialog := formatMessages(msgs)
	prompt := strings.ReplaceAll(ExtractionPrompt, "{{.Dialog}}", dialog)

	// 打印LLM输入参数
	logger.Debug("[ProfileExtract] LLM输入", map[string]interface{}{
		"prompt":       prompt,
		"dialog":       dialog,
		"messageCount": len(msgs),
	})

	// 调用LLM
	req := &schema.ChatRequest{
		Messages: []schema.Message{
			{Role: "user", Content: prompt},
		},
	}
	chatResp, err := e.model.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM提取失败: %w", err)
	}

	resp := chatResp.Content

	// 打印LLM原始输出
	logger.Debug("[ProfileExtract] LLM原始输出", map[string]interface{}{
		"response": resp,
	})

	// 解析JSON
	var result ExtractionResult
	if err := json.Unmarshal([]byte(extractJSON(resp)), &result); err != nil {
		logger.Warn("解析提取结果失败，返回空结果", map[string]interface{}{
			"error":    err.Error(),
			"response": resp,
		})
		return &ExtractionResult{
			Tags:        []ExtractedTag{},
			Preferences: make(map[string]string),
			Confidence:  0,
		}, nil
	}

	// 验证和清理标签
	result.Tags = validateAndCleanTags(result.Tags)

	// 清理偏好设置
	result.Preferences = cleanPreferences(result.Preferences)

	// 填充 Token 使用量
	result.Usage = chatResp.Usage

	// 打印解析后的输出参数
	logger.Debug("[ProfileExtract] LLM解析结果", map[string]interface{}{
		"tags":        result.Tags,
		"preferences": result.Preferences,
		"confidence":  result.Confidence,
		"usage":       result.Usage,
	})

	return &result, nil
}

// validateAndCleanTags 验证并清理标签
// 1. 过滤低置信度
// 2. 验证标签类型
// 3. 验证标签名称非空
// 4. 去重（保留高置信度）
func validateAndCleanTags(tags []ExtractedTag) []ExtractedTag {
	const (
		minConfidence = 0.6 // 最低置信度
		maxConfidence = 1.0 // 最高置信度
	)

	// 使用map去重，保留高置信度
	seen := make(map[string]ExtractedTag)

	for _, tag := range tags {
		// 验证置信度范围
		if tag.Confidence < minConfidence || tag.Confidence > maxConfidence {
			continue
		}

		// 验证标签名称非空
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		tag.Name = name

		// 验证标签类型
		tagType := strings.ToLower(strings.TrimSpace(tag.Type))
		if !validTagTypes[tagType] {
			logger.Debug("忽略无效标签类型", map[string]interface{}{
				"name": name,
				"type": tagType,
			})
			continue
		}
		tag.Type = tagType

		// 清理证据文本
		tag.Evidence = strings.TrimSpace(tag.Evidence)

		// 去重：如果已存在同名标签，保留置信度高的
		if existing, ok := seen[name]; ok {
			if tag.Confidence > existing.Confidence {
				seen[name] = tag
			}
		} else {
			seen[name] = tag
		}
	}

	// 转换回切片
	result := make([]ExtractedTag, 0, len(seen))
	for _, tag := range seen {
		result = append(result, tag)
	}

	return result
}

// cleanPreferences 清理偏好设置
// 过滤空值和非字符串值
func cleanPreferences(prefs map[string]string) map[string]string {
	if prefs == nil {
		return make(map[string]string)
	}

	cleaned := make(map[string]string)
	for key, value := range prefs {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			cleaned[key] = value
		}
	}

	return cleaned
}

// formatMessages 格式化消息列表
func formatMessages(msgs []message.Message) string {
	var parts []string
	for _, msg := range msgs {
		parts = append(parts, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
	}
	return strings.Join(parts, "\n")
}

// extractJSON 从响应中提取JSON
// 使用栈匹配找到最外层的JSON对象
func extractJSON(resp string) string {
	resp = strings.TrimSpace(resp)

	// 查找第一个 {
	start := strings.Index(resp, "{")
	if start == -1 {
		return resp
	}

	// 使用栈匹配找到对应的 }
	stack := 0
	end := -1
	inString := false
	escapeNext := false

	for i := start; i < len(resp); i++ {
		ch := resp[i]

		if escapeNext {
			escapeNext = false
			continue
		}

		if ch == '\\' {
			escapeNext = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if ch == '{' {
			stack++
		} else if ch == '}' {
			stack--
			if stack == 0 {
				end = i
				break
			}
		}
	}

	if end == -1 {
		return resp[start:]
	}

	return resp[start : end+1]
}
