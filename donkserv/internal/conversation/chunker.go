package conversation

import "strings"

// TextSplitterConfig 文本切片器配置
type TextSplitterConfig struct {
	MaxChunkSize int // 最大块长度（字符数）
	OverlapSize  int // 相邻块重叠长度
}

// DefaultTextSplitterConfig 默认配置
var DefaultTextSplitterConfig = TextSplitterConfig{
	MaxChunkSize: 500, // 默认500字符
	OverlapSize:  50,  // 默认重叠50字符
}

// TextSplitter 文本切片器
// 将长文本按规则切分为多个小块
type TextSplitter struct {
	config TextSplitterConfig
}

// NewTextSplitter 创建文本切片器
//
// 参数:
//
//	config: 切片配置
//
// 返回:
//
//	*TextSplitter: 切片器实例
func NewTextSplitter(config TextSplitterConfig) *TextSplitter {
	if config.MaxChunkSize == 0 {
		config.MaxChunkSize = DefaultTextSplitterConfig.MaxChunkSize
	}
	if config.OverlapSize == 0 {
		config.OverlapSize = DefaultTextSplitterConfig.OverlapSize
	}

	return &TextSplitter{
		config: config,
	}
}

// Split 将文本切分为多个块
// 按段落分割，然后合并小段落，限制每块长度
//
// 参数:
//
//	text: 待分割文本
//
// 返回:
//
//	[]string: 切分后的文本块列表
func (t *TextSplitter) Split(text string) []string {
	if text == "" {
		return nil
	}

	// 按换行符分割段落
	paragraphs := splitByLines(text)
	if len(paragraphs) == 0 {
		return []string{text}
	}

	// 合并小段落，限制每块长度
	chunks := mergeParagraphs(paragraphs, t.config.MaxChunkSize)

	return chunks
}

// splitByLines 按行分割文本
//
// 参数:
//
//	text: 文本
//
// 返回:
//
//	[]string: 行列表
func splitByLines(text string) []string {
	var lines []string
	var current string

	for _, r := range text {
		if r == '\n' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// mergeParagraphs 合并段落，限制每块长度
//
// 参数:
//
//	paragraphs: 段落列表
//	maxSize: 最大块长度
//
// 返回:
//
//	[]string: 合并后的块列表
func mergeParagraphs(paragraphs []string, maxSize int) []string {
	if len(paragraphs) == 0 {
		return nil
	}

	var chunks []string
	var current strings.Builder

	for i, para := range paragraphs {
		// 如果当前块加上新段落超过最大长度，先保存当前块
		if current.Len()+len(para) > maxSize && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}

		// 添加段落
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(para)

		// 如果是最后一个段落，直接保存
		if i == len(paragraphs)-1 {
			chunks = append(chunks, current.String())
		}
	}

	// 处理单独一个段落超过最大长度的情况
	if len(chunks) == 0 && len(paragraphs) > 0 {
		chunks = append(chunks, paragraphs[0])
	}

	return chunks
}
