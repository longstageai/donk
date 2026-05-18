// skilldiscovery 技能自动发现模块
// 重复检测器实现
package skilldiscovery

import (
	"context"
	"strings"

	"github.com/longstageai/donk/donk/internal/skill"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// DuplicateChecker 技能重复检测器
// 负责检测候选技能是否与现有技能重复
type DuplicateChecker struct {
	stateRepo *skill.StateRepository
	threshold float64
}

// NewDuplicateChecker 创建重复检测器
// 参数:
//   - stateRepo: Skill 状态仓库
//   - threshold: 相似度阈值，默认 0.6
//
// 返回:
//   - *DuplicateChecker: 检测器实例
func NewDuplicateChecker(stateRepo *skill.StateRepository, threshold float64) *DuplicateChecker {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.6
	}
	return &DuplicateChecker{
		stateRepo: stateRepo,
		threshold: threshold,
	}
}

// CheckDuplicate 检查候选技能是否重复
// 参数:
//   - ctx: 上下文
//   - candidate: 候选技能
//
// 返回:
//   - *DuplicateCheckResult: 检查结果
//   - error: 错误信息
func (c *DuplicateChecker) CheckDuplicate(ctx context.Context, candidate *SkillCandidate) (*DuplicateCheckResult, error) {
	logger.Debug("开始检查技能重复", map[string]interface{}{
		"candidate_name": candidate.Name,
	})

	// 查询所有现有技能
	skills, err := c.stateRepo.List()
	if err != nil {
		logger.Error("查询技能列表失败", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	logger.Debug("获取现有技能列表", map[string]interface{}{
		"existing_count": len(skills),
	})

	// 遍历现有技能进行比对
	for _, existing := range skills {
		// 名称精确匹配（不区分大小写）
		if strings.EqualFold(existing.Name, candidate.Name) {
			logger.Info("检测到重复技能(名称匹配)", map[string]interface{}{
				"candidate_name": candidate.Name,
				"existing_name":  existing.Name,
			})
			return &DuplicateCheckResult{
				IsDuplicate: true,
				Reason:      "技能名称已存在",
				Existing:    existing,
				Similarity:  1.0,
			}, nil
		}

		// 描述相似度检查
		similarity := c.calculateSimilarity(existing.Description, candidate.Description)
		if similarity >= c.threshold {
			logger.Info("检测到重复技能(描述相似)", map[string]interface{}{
				"candidate_name": candidate.Name,
				"existing_name":  existing.Name,
				"similarity":     similarity,
			})
			return &DuplicateCheckResult{
				IsDuplicate: true,
				Reason:      "技能描述与现有技能相似",
				Existing:    existing,
				Similarity:  similarity,
			}, nil
		}
	}

	logger.Debug("技能未重复", map[string]interface{}{
		"candidate_name": candidate.Name,
	})

	return &DuplicateCheckResult{
		IsDuplicate: false,
		Reason:      "",
		Existing:    nil,
		Similarity:  0,
	}, nil
}

// calculateSimilarity 计算两个描述的相似度
// 使用 Jaccard 相似度算法
// 参数:
//   - desc1: 描述1
//   - desc2: 描述2
//
// 返回:
//   - float64: 相似度 (0.0-1.0)
func (c *DuplicateChecker) calculateSimilarity(desc1, desc2 string) float64 {
	// 标准化处理
	d1 := strings.ToLower(strings.TrimSpace(desc1))
	d2 := strings.ToLower(strings.TrimSpace(desc2))

	// 完全相同
	if d1 == d2 {
		return 1.0
	}

	// 包含关系
	if strings.Contains(d1, d2) || strings.Contains(d2, d1) {
		return 0.9
	}

	// 提取关键词
	words1 := c.extractKeywords(d1)
	words2 := c.extractKeywords(d2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0
	}

	// 计算交集
	intersection := 0
	for w1 := range words1 {
		if words2[w1] {
			intersection++
		}
	}

	// Jaccard 相似度 = 交集 / 并集
	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0
	}

	similarity := float64(intersection) / float64(union)
	return similarity
}

// extractKeywords 提取关键词
// 参数:
//   - text: 文本内容
//
// 返回:
//   - map[string]bool: 关键词集合
func (c *DuplicateChecker) extractKeywords(text string) map[string]bool {
	// 停用词表
	stopWords := map[string]bool{
		// 中文停用词
		"的": true, "了": true, "和": true, "是": true, "在": true,
		"有": true, "我": true, "他": true, "她": true, "它": true,
		"们": true, "这": true, "那": true, "个": true, "之": true,
		"与": true, "及": true, "或": true, "但": true, "而": true,
		"因为": true, "所以": true, "如果": true, "就": true, "都": true,
		"要": true, "会": true, "能": true, "可以": true, "把": true,
		"被": true, "让": true, "给": true, "为": true, "于": true,
		"以": true, "等": true, "对": true, "将": true,
		// 英文停用词
		"a": true, "an": true, "the": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"to": true, "of": true, "for": true, "and": true, "or": true,
		"in": true, "on": true, "at": true, "by": true, "with": true,
		"from": true, "as": true, "that": true, "this": true, "it": true,
		"its": true, "not": true, "no": true, "yes": true, "can": true,
		"will": true, "would": true, "could": true, "should": true,
	}

	words := make(map[string]bool)

	// 使用多种分隔符分词
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == '，' || r == '。' ||
			r == '、' || r == ';' || r == '；' || r == ':' || r == '：' ||
			r == '!' || r == '！' || r == '?' || r == '？' || r == '(' ||
			r == ')' || r == '（' || r == '）' || r == '[' || r == ']' ||
			r == '"' || r == '-' || r == '_'
	})

	for _, w := range fields {
		w = strings.TrimSpace(w)
		// 过滤停用词和短词
		if len(w) > 1 && !stopWords[w] {
			words[w] = true
		}
	}

	return words
}
