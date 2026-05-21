package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/longstageai/donk/donk/internal/tool"
)

// PDFParser PDF解析工具
// 用于解析完整PDF文档并提取文本内容，支持文件大小限制、路径安全检查和内容长度限制
type PDFParser struct {
	maxFileSize int64 // 最大文件大小（字节）
}

// PDFParserOption PDF解析器配置选项
type PDFParserOption func(*PDFParser)

// WithPDFMaxFileSize 设置最大PDF文件大小
func WithPDFMaxFileSize(size int64) PDFParserOption {
	return func(p *PDFParser) {
		p.maxFileSize = size
	}
}

// NewPDFParser 创建PDF解析工具
func NewPDFParser(opts ...PDFParserOption) *PDFParser {
	p := &PDFParser{
		maxFileSize: 20 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name 返回工具名称
func (p *PDFParser) Name() string {
	return "pdf_parser"
}

// Description 返回工具描述
func (p *PDFParser) Description() string {
	return "解析完整PDF文档并提取文本内容，支持内容长度限制。"
}

// Version 返回版本
func (p *PDFParser) Version() string {
	return "1.0.0"
}

// Category 返回分类
func (p *PDFParser) Category() string {
	return string(tool.CategoryFile)
}

// Parameters 返回参数定义
func (p *PDFParser) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"path": {
			Type:        "string",
			Description: "要解析的PDF文件绝对路径",
		},
		"max_chars": {
			Type:        "integer",
			Description: "返回文本最大字符数，0表示不限制，默认0",
			Default:     0,
		},
		"include_page_breaks": {
			Type:        "boolean",
			Description: "是否在结果中包含页码分隔标记，默认false",
			Default:     false,
		},
	}
	schema.Required = []string{"path"}
	return schema
}

// Execute 执行PDF解析
func (p *PDFParser) Execute(ctx *tool.Context) (*tool.Result, error) {
	startTime := time.Now()

	path, ok := ctx.Params["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "PDF文件路径不能为空"), nil
	}

	if !filepath.IsAbs(path) {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "PDF文件路径必须是绝对路径"), nil
	}
	resolvedPath := filepath.Clean(path)

	if err := p.validatePath(resolvedPath); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("路径安全检查失败: %v", err)), nil
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "PDF文件不存在"), nil
		}
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("访问PDF文件失败: %v", err)), nil
	}

	if info.IsDir() {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "路径是目录，不是PDF文件"), nil
	}

	if strings.ToLower(filepath.Ext(resolvedPath)) != ".pdf" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "仅支持解析.pdf文件"), nil
	}

	if info.Size() > p.maxFileSize {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("PDF文件大小 %d 字节超过限制 %d 字节", info.Size(), p.maxFileSize)), nil
	}

	maxChars := parseIntParam(ctx.Params["max_chars"], 0)
	includePageBreaks := false
	if v, ok := ctx.Params["include_page_breaks"].(bool); ok {
		includePageBreaks = v
	}

	if maxChars < 0 {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "max_chars不能小于0"), nil
	}

	content, totalPages, parsedPages, truncated, err := p.parsePDF(resolvedPath, maxChars, includePageBreaks)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
	}

	result := tool.NewResult(map[string]any{
		"path":         resolvedPath,
		"size":         info.Size(),
		"total_pages":  totalPages,
		"parsed_pages": parsedPages,
		"content":      content,
		"char_count":   len([]rune(content)),
		"truncated":    truncated,
		"modified_at":  info.ModTime().Format("2006-01-02 15:04:05"),
		"duration_ms":  time.Since(startTime).Milliseconds(),
	})
	result.SetExecutionTime(time.Since(startTime))
	return result, nil
}

// parsePDF 解析完整PDF文档并提取文本
func (p *PDFParser) parsePDF(path string, maxChars int, includePageBreaks bool) (string, int, int, bool, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", 0, 0, false, fmt.Errorf("打开PDF文件失败: %w", err)
	}
	defer file.Close()

	totalPages := reader.NumPage()
	if totalPages == 0 {
		return "", totalPages, 0, false, nil
	}

	var builder strings.Builder
	parsedPages := 0
	truncated := false

	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		page := reader.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		if includePageBreaks {
			builder.WriteString(fmt.Sprintf("\n--- Page %d ---\n", pageIndex))
		}
		builder.WriteString(text)
		builder.WriteString("\n")
		parsedPages++

		if maxChars > 0 && len([]rune(builder.String())) >= maxChars {
			truncated = true
			break
		}
	}

	content := strings.TrimSpace(builder.String())
	if maxChars > 0 {
		runes := []rune(content)
		if len(runes) > maxChars {
			content = string(runes[:maxChars])
			truncated = true
		}
	}

	return content, totalPages, parsedPages, truncated, nil
}

// validatePath 验证路径安全
func (p *PDFParser) validatePath(path string) error {
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

// parseIntParam 解析整数参数
func parseIntParam(value any, defaultValue int) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return defaultValue
}
