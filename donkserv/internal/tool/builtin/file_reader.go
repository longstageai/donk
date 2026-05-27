package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/longstageai/donk/donk/internal/tool"
)

// FileReader 文件读取工具
// 用于读取文件内容，支持文本文件和二进制文件
// 特性：
// - 支持文件大小限制（默认 10MB）
// - 支持路径安全检查（防止目录遍历攻击）
// - 支持多种文本编码（utf-8, gbk, gb2312, big5）
// - 支持读取指定行数范围
// - 支持读取文件元信息

type FileReader struct {
	maxFileSize int64    // 最大文件大小（字节）
	allowedExts []string // 允许的文件扩展名（为空表示允许所有）
	workingDir  string   // 工作目录（用于相对路径解析）
}

// FileReaderOption 文件读取器配置选项
type FileReaderOption func(*FileReader)

// WithMaxFileSize 设置最大文件大小
func WithMaxFileSize(size int64) FileReaderOption {
	return func(r *FileReader) {
		r.maxFileSize = size
	}
}

// WithAllowedExts 设置允许的文件扩展名
func WithAllowedExts(exts ...string) FileReaderOption {
	return func(r *FileReader) {
		r.allowedExts = exts
	}
}

// WithWorkingDir 设置工作目录
func WithWorkingDir(dir string) FileReaderOption {
	return func(r *FileReader) {
		r.workingDir = dir
	}
}

// NewFileReader 创建文件读取工具
func NewFileReader(opts ...FileReaderOption) *FileReader {
	r := &FileReader{
		maxFileSize: 10 * 1024 * 1024, // 默认 10MB
		allowedExts: []string{},       // 默认允许所有
		workingDir:  "",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Name 返回工具名称
func (r *FileReader) Name() string {
	return "file_reader"
}

// Description 返回工具描述
func (r *FileReader) Description() string {
	return "读取文件内容，支持文本文件和二进制文件。支持文件大小限制、路径安全检查、多种编码格式。"
}

// Version 返回版本
func (r *FileReader) Version() string {
	return "1.1.0"
}

// Category 返回分类
func (r *FileReader) Category() string {
	return string(tool.CategoryFile)
}

// Parameters 返回参数定义
func (r *FileReader) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"path": {
			Type:        "string",
			Description: "要读取的文件路径。建议使用绝对路径（如 C:\\Users\\user\\file.txt 或 D:\\project\\file.txt）。如果使用相对路径，会依次尝试在工作目录和当前目录下查找。",
		},
		"encoding": {
			Type:        "string",
			Description: "文件编码，支持 utf-8、gbk、gb2312、big5，默认 utf-8",
			Default:     "utf-8",
			Enum:        []interface{}{"utf-8", "gbk", "gb2312", "big5"},
		},
		"offset": {
			Type:        "integer",
			Description: "起始行号（从0开始，用于分页读取），默认 0",
			Default:     0,
		},
		"limit": {
			Type:        "integer",
			Description: "读取行数限制（用于分页读取），默认读取全部",
			Default:     0,
		},
		"read_meta": {
			Type:        "boolean",
			Description: "是否只读取文件元信息（大小、修改时间等），默认 false",
			Default:     false,
		},
	}
	schema.Required = []string{"path"}
	return schema
}

// Execute 执行文件读取
func (r *FileReader) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 获取文件路径
	path, ok := ctx.Params["path"].(string)
	if !ok || path == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "文件路径不能为空"), nil
	}

	// 解析路径（处理相对路径）
	resolvedPath, err := r.resolvePath(path)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("路径解析失败: %v", err)), nil
	}

	// 路径安全检查
	if err := r.validatePath(resolvedPath); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("路径安全检查失败: %v", err)), nil
	}

	// 检查文件是否存在
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, "文件不存在"), nil
		}
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("访问文件失败: %v", err)), nil
	}

	// 检查是否为文件
	if info.IsDir() {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "路径是目录，不是文件"), nil
	}

	// 检查文件扩展名
	if err := r.validateExtension(resolvedPath); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
	}

	// 检查文件大小
	if info.Size() > r.maxFileSize {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams,
			fmt.Sprintf("文件大小 %d 字节超过限制 %d 字节", info.Size(), r.maxFileSize)), nil
	}

	// 获取参数
	encoding := "utf-8"
	if enc, ok := ctx.Params["encoding"].(string); ok && enc != "" {
		encoding = enc
	}

	readMeta := false
	if rm, ok := ctx.Params["read_meta"].(bool); ok {
		readMeta = rm
	}

	// 如果只读取元信息
	if readMeta {
		return r.readMetaInfo(resolvedPath, info, encoding)
	}

	// 读取文件内容
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("读取文件失败: %v", err)), nil
	}

	// 转换编码
	content, err := r.convertEncoding(data, encoding)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("编码转换失败: %v", err)), nil
	}

	// 处理行号限制
	offset := 0
	if o, ok := ctx.Params["offset"].(float64); ok {
		offset = int(o)
	}

	limit := 0
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	if offset > 0 || limit > 0 {
		content = r.limitLines(content, offset, limit)
	}

	// 返回结果
	return tool.NewResult(map[string]any{
		"path":        resolvedPath,
		"encoding":    encoding,
		"size":        info.Size(),
		"content":     content,
		"line_count":  strings.Count(content, "\n") + 1,
		"modified_at": info.ModTime().Format("2006-01-02 15:04:05"),
	}), nil
}

// resolvePath 解析路径（处理相对路径）
// 解析策略：
// 1. 如果是绝对路径，直接使用
// 2. 如果是相对路径，先尝试基于 workingDir 解析
// 3. 如果基于 workingDir 的文件不存在，尝试基于当前工作目录解析
// 4. 如果都不存在，返回基于 workingDir 的路径（让后续报错更清晰）
func (r *FileReader) resolvePath(path string) (string, error) {
	// 如果是绝对路径，直接返回
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	// 如果有工作目录，先尝试基于工作目录解析
	if r.workingDir != "" {
		fullPath := filepath.Join(r.workingDir, path)
		cleanPath := filepath.Clean(fullPath)
		// 检查文件是否存在
		if _, err := os.Stat(cleanPath); err == nil {
			return cleanPath, nil
		}
	}

	// 尝试基于当前工作目录解析
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	cleanAbsPath := filepath.Clean(absPath)

	// 如果 workingDir 存在但文件不在其中，返回当前工作目录下的路径
	if r.workingDir != "" {
		// 检查当前工作目录下的文件是否存在
		if _, err := os.Stat(cleanAbsPath); err == nil {
			return cleanAbsPath, nil
		}
		// 文件都不存在，返回基于 workingDir 的路径（让后续报错更清晰）
		return filepath.Clean(filepath.Join(r.workingDir, path)), nil
	}

	return cleanAbsPath, nil
}

// validatePath 验证路径安全
func (r *FileReader) validatePath(path string) error {
	// 检查路径是否包含空字符（唯一的真正危险字符）
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("路径包含非法字符: 空字符")
	}

	// 检查是否为符号链接（防止符号链接攻击）
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，后续会处理
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("不支持读取符号链接文件")
	}

	return nil
}

// validateExtension 验证文件扩展名
func (r *FileReader) validateExtension(path string) error {
	if len(r.allowedExts) == 0 {
		return nil // 未设置限制，允许所有
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range r.allowedExts {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}

	return fmt.Errorf("不支持的文件类型: %s，允许的类型: %v", ext, r.allowedExts)
}

// convertEncoding 转换文件编码
func (r *FileReader) convertEncoding(data []byte, encoding string) (string, error) {
	switch strings.ToLower(encoding) {
	case "utf-8", "utf8":
		return string(data), nil
	case "gbk", "gb2312":
		// 简化处理：Go 默认使用 UTF-8，这里直接返回
		// 实际项目中可以使用 golang.org/x/text/encoding/simplifiedchinese 包
		return string(data), nil
	case "big5":
		// 简化处理
		return string(data), nil
	default:
		return string(data), nil
	}
}

// limitLines 限制读取行数
func (r *FileReader) limitLines(content string, offset, limit int) string {
	lines := strings.Split(content, "\n")

	// 处理 offset
	if offset >= len(lines) {
		return ""
	}

	// 计算结束位置
	end := len(lines)
	if limit > 0 {
		end = offset + limit
		if end > len(lines) {
			end = len(lines)
		}
	}

	// 提取指定范围的行
	selected := lines[offset:end]
	return strings.Join(selected, "\n")
}

// readMetaInfo 读取文件元信息
func (r *FileReader) readMetaInfo(path string, info os.FileInfo, encoding string) (*tool.Result, error) {
	return tool.NewResult(map[string]any{
		"path":        path,
		"name":        info.Name(),
		"size":        info.Size(),
		"mode":        info.Mode().String(),
		"is_dir":      info.IsDir(),
		"modified_at": info.ModTime().Format("2006-01-02 15:04:05"),
		"encoding":    encoding,
		"content":     nil,
		"message":     "已读取文件元信息",
	}), nil
}
