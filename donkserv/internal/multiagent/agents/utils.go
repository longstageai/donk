// agents Agent工具函数
package agents

import (
	"regexp"
	"strings"
)

// extractJSON 从文本中提取JSON
func extractJSON(content string) string {
	// 查找JSON对象
	start := strings.Index(content, "{")
	if start == -1 {
		return content
	}

	// 找到匹配的结束括号
	braceCount := 0
	end := start
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				end = i + 1
				break
			}
		}
		if braceCount == 0 {
			break
		}
	}

	if end > start {
		return content[start:end]
	}

	return content
}

// cleanJSONString 清理JSON字符串
func cleanJSONString(s string) string {
	// 去除markdown代码块标记
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	return s
}

// extractArray 从文本中提取JSON数组
func extractArray(content string) string {
	// 查找JSON数组
	start := strings.Index(content, "[")
	if start == -1 {
		return content
	}

	// 找到匹配的结束括号
	bracketCount := 0
	end := start
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '[':
			bracketCount++
		case ']':
			bracketCount--
			if bracketCount == 0 {
				end = i + 1
				break
			}
		}
		if bracketCount == 0 {
			break
		}
	}

	if end > start {
		return content[start:end]
	}

	return content
}

// sanitizeString 清理字符串，移除控制字符
func sanitizeString(s string) string {
	// 移除控制字符，保留换行和制表符
	re := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	return re.ReplaceAllString(s, "")
}
