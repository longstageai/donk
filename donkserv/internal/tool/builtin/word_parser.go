package builtin

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/fumiama/go-docx"
	"github.com/longstageai/donk/donk/internal/tool"
)

// WordParser Word文档解析工具
// 用于解析完整DOC/DOCX文档并提取文本内容，支持文件大小限制、路径安全检查和内容长度限制
type WordParser struct {
	maxFileSize int64 // 最大文件大小（字节）
}

// WordParserOption Word解析器配置选项
type WordParserOption func(*WordParser)

// WithWordMaxFileSize 设置最大Word文件大小
func WithWordMaxFileSize(size int64) WordParserOption {
	return func(p *WordParser) {
		p.maxFileSize = size
	}
}

// NewWordParser 创建Word解析工具
func NewWordParser(opts ...WordParserOption) *WordParser {
	p := &WordParser{
		maxFileSize: 20 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name 返回工具名称
func (p *WordParser) Name() string {
	return "word_parser"
}

// Description 返回工具描述
func (p *WordParser) Description() string {
	return "解析完整DOC/DOCX文档并提取文本内容，支持内容长度限制。"
}

// Version 返回版本
func (p *WordParser) Version() string {
	return "1.0.0"
}

// Category 返回分类
func (p *WordParser) Category() string {
	return string(tool.CategoryFile)
}

// Parameters 返回参数定义
func (p *WordParser) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"path": {
			Type:        "string",
			Description: "要解析的DOC或DOCX文件绝对路径",
		},
		"max_chars": {
			Type:        "integer",
			Description: "返回文本最大字符数，0表示不限制，默认0",
			Default:     0,
		},
	}
	schema.Required = []string{"path"}
	return schema
}

// Execute 执行Word文档解析
func (p *WordParser) Execute(ctx *tool.Context) (*tool.Result, error) {
	startTime := time.Now()

	path, ok := ctx.Params["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "Word文件路径不能为空"), nil
	}
	if !filepath.IsAbs(path) {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "Word文件路径必须是绝对路径"), nil
	}
	resolvedPath := filepath.Clean(path)

	if err := p.validatePath(resolvedPath); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("路径安全检查失败: %v", err)), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "Word文件不存在"), nil
		}
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("访问Word文件失败: %v", err)), nil
	}
	if info.IsDir() {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "路径是目录，不是Word文件"), nil
	}
	if info.Size() > p.maxFileSize {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("Word文件大小 %d 字节超过限制 %d 字节", info.Size(), p.maxFileSize)), nil
	}

	ext := strings.ToLower(filepath.Ext(resolvedPath))
	if ext != ".doc" && ext != ".docx" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "仅支持解析.doc和.docx文件"), nil
	}

	maxChars := parseIntParam(ctx.Params["max_chars"], 0)
	if maxChars < 0 {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "max_chars不能小于0"), nil
	}

	content, parserType, truncated, err := p.parseWord(resolvedPath, ext, maxChars)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
	}

	result := tool.NewResult(map[string]any{
		"path":        resolvedPath,
		"size":        info.Size(),
		"file_type":   ext,
		"parser_type": parserType,
		"content":     content,
		"char_count":  len([]rune(content)),
		"truncated":   truncated,
		"modified_at": info.ModTime().Format("2006-01-02 15:04:05"),
		"duration_ms": time.Since(startTime).Milliseconds(),
	})
	result.SetExecutionTime(time.Since(startTime))
	return result, nil
}

// parseWord 根据文件扩展名解析Word文档
func (p *WordParser) parseWord(path, ext string, maxChars int) (string, string, bool, error) {
	if ext == ".docx" {
		return p.parseDocx(path, maxChars)
	}
	return p.parseDoc(path, maxChars)
}

// parseDocx 解析DOCX文档并提取文本
func (p *WordParser) parseDocx(path string, maxChars int) (string, string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "docx", false, fmt.Errorf("打开DOCX文件失败: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", "docx", false, fmt.Errorf("获取DOCX文件信息失败: %w", err)
	}

	document, err := docx.Parse(file, info.Size())
	if err != nil {
		return "", "docx", false, fmt.Errorf("解析DOCX文件失败: %w", err)
	}

	var builder strings.Builder
	truncated := false
	for _, item := range document.Document.Body.Items {
		itemText := strings.TrimSpace(fmt.Sprintf("%v", item))
		if itemText == "" || itemText == "<nil>" {
			continue
		}
		builder.WriteString(itemText)
		builder.WriteString("\n")
		if maxChars > 0 && len([]rune(builder.String())) >= maxChars {
			truncated = true
			break
		}
	}

	content, truncatedByLimit := limitText(cleanWordContent(builder.String()), maxChars)
	return content, "docx", truncated || truncatedByLimit, nil
}

// parseDoc 从DOC二进制文件中兜底提取可读文本
func (p *WordParser) parseDoc(path string, maxChars int) (string, string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "doc", false, fmt.Errorf("读取DOC文件失败: %w", err)
	}

	content := extractReadableText(data)
	content, truncated := limitText(cleanWordContent(content), maxChars)
	return content, "doc_text_fallback", truncated, nil
}

// validatePath 验证路径安全
func (p *WordParser) validatePath(path string) error {
	dangerousPatterns := []string{"..", "//", "\\\\", "\x00"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return fmt.Errorf("路径包含非法字符: %s", pattern)
		}
	}

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("不支持解析符号链接文件")
	}
	return nil
}

// extractReadableText 从二进制内容中提取可读文本
func extractReadableText(data []byte) string {
	var parts []string
	parts = append(parts, extractASCIIText(data)...)
	parts = append(parts, extractUTF16LEText(data)...)
	return strings.Join(parts, "\n")
}

// extractASCIIText 提取ASCII可读文本片段
func extractASCIIText(data []byte) []string {
	var parts []string
	var builder strings.Builder
	for _, b := range data {
		r := rune(b)
		if r == '\r' || r == '\n' || r == '\t' || (r >= 32 && r <= 126) {
			builder.WriteRune(r)
			continue
		}
		appendReadablePart(&parts, &builder)
	}
	appendReadablePart(&parts, &builder)
	return parts
}

// extractUTF16LEText 提取UTF-16LE可读文本片段
func extractUTF16LEText(data []byte) []string {
	if len(data) < 2 {
		return nil
	}
	var parts []string
	var units []uint16
	flush := func() {
		if len(units) < 4 {
			units = units[:0]
			return
		}
		runes := utf16.Decode(units)
		text := strings.TrimSpace(string(runes))
		if isReadableText(text) {
			parts = append(parts, text)
		}
		units = units[:0]
	}

	for i := 0; i+1 < len(data); i += 2 {
		unit := uint16(data[i]) | uint16(data[i+1])<<8
		if unit == 9 || unit == 10 || unit == 13 || (unit >= 32 && unit != 0xffff && unicode.IsPrint(rune(unit))) {
			units = append(units, unit)
			continue
		}
		flush()
	}
	flush()
	return parts
}

// appendReadablePart 添加可读文本片段
func appendReadablePart(parts *[]string, builder *strings.Builder) {
	text := strings.TrimSpace(builder.String())
	builder.Reset()
	if isReadableText(text) {
		*parts = append(*parts, text)
	}
}

// isReadableText 判断文本片段是否具备可读性
func isReadableText(text string) bool {
	if len([]rune(text)) < 4 || !utf8.ValidString(text) {
		return false
	}
	letters := 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			letters++
		}
	}
	return letters >= 2 && !bytes.Contains([]byte(text), []byte("\x00"))
}

// cleanWordContent 清理Word解析结果
func cleanWordContent(content string) string {
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")
	content = strings.ReplaceAll(content, "\x00", " ")
	content = strings.Join(strings.Fields(content), " ")
	return strings.TrimSpace(content)
}

// limitText 限制文本最大字符数
func limitText(content string, maxChars int) (string, bool) {
	if maxChars <= 0 {
		return content, false
	}
	runes := []rune(content)
	if len(runes) <= maxChars {
		return content, false
	}
	return string(runes[:maxChars]), true
}
