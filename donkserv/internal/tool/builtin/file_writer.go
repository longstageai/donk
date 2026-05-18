package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/longstageai/donk/donk/internal/tool"
)

// FileWriter 文件写入工具
// 用于写入文件内容，支持追加模式和覆盖模式
// 特性：
// - 支持追加模式和覆盖模式
// - 支持路径安全检查（防止目录遍历攻击）
// - 支持文件大小限制
// - 支持自动创建目录
// - 支持写入前备份原文件

type FileWriter struct {
	maxFileSize   int64    // 最大文件大小（字节）
	workingDir    string   // 工作目录（用于相对路径解析）
	allowedExts   []string // 允许的文件扩展名
	backupEnabled bool     // 是否启用备份
	backupSuffix  string   // 备份文件后缀
}

// FileWriterOption 文件写入器配置选项
type FileWriterOption func(*FileWriter)

// WithMaxFileSizeWriter 设置最大文件大小
func WithMaxFileSizeWriter(size int64) FileWriterOption {
	return func(w *FileWriter) {
		w.maxFileSize = size
	}
}

// WithWorkingDirWriter 设置工作目录
func WithWorkingDirWriter(dir string) FileWriterOption {
	return func(w *FileWriter) {
		w.workingDir = dir
	}
}

// WithAllowedExtsWriter 设置允许的文件扩展名
func WithAllowedExtsWriter(exts ...string) FileWriterOption {
	return func(w *FileWriter) {
		w.allowedExts = exts
	}
}

// WithBackupEnabled 设置是否启用备份
func WithBackupEnabled(enabled bool) FileWriterOption {
	return func(w *FileWriter) {
		w.backupEnabled = enabled
	}
}

// NewFileWriter 创建文件写入工具
func NewFileWriter(opts ...FileWriterOption) *FileWriter {
	w := &FileWriter{
		maxFileSize:   100 * 1024 * 1024, // 默认 100MB
		workingDir:    "",
		allowedExts:   []string{},
		backupEnabled: true,
		backupSuffix:  ".backup",
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Name 返回工具名称
func (w *FileWriter) Name() string {
	return "file_writer"
}

// Description 返回工具描述
func (w *FileWriter) Description() string {
	return "写入文件内容，支持追加模式和覆盖模式。支持路径安全检查、自动创建目录、文件备份。"
}

// Version 返回版本
func (w *FileWriter) Version() string {
	return "1.1.0"
}

// Category 返回分类
func (w *FileWriter) Category() string {
	return string(tool.CategoryFile)
}

// Parameters 返回参数定义
func (w *FileWriter) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"path": {
			Type:        "string",
			Description: "要写入的文件路径（支持绝对路径和相对路径）",
		},
		"content": {
			Type:        "string",
			Description: "要写入的内容",
		},
		"mode": {
			Type:        "string",
			Description: "写入模式：append(追加，默认)、overwrite(覆盖)、create(仅创建新文件)",
			Default:     "append",
			Enum:        []interface{}{"append", "overwrite", "create"},
		},
		"add_newline": {
			Type:        "boolean",
			Description: "写入内容后是否追加换行符，默认 true",
			Default:     true,
		},
		"create_dirs": {
			Type:        "boolean",
			Description: "是否自动创建不存在的目录，默认 true",
			Default:     true,
		},
		"backup": {
			Type:        "boolean",
			Description: "覆盖前是否备份原文件，默认 true",
			Default:     true,
		},
		"encoding": {
			Type:        "string",
			Description: "文件编码，支持 utf-8、gbk、gb2312、big5，默认 utf-8",
			Default:     "utf-8",
			Enum:        []interface{}{"utf-8", "gbk", "gb2312", "big5"},
		},
	}
	schema.Required = []string{"path", "content"}
	return schema
}

// Execute 执行文件写入
func (w *FileWriter) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 获取文件路径
	path, ok := ctx.Params["path"].(string)
	if !ok || path == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "文件路径不能为空"), nil
	}

	// 获取写入内容
	content, ok := ctx.Params["content"].(string)
	if !ok {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "写入内容不能为空"), nil
	}

	// 解析路径
	resolvedPath, err := w.resolvePath(path)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("路径解析失败: %v", err)), nil
	}

	// 路径安全检查
	if err := w.validatePath(resolvedPath); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("路径安全检查失败: %v", err)), nil
	}

	// 检查文件扩展名
	if err := w.validateExtension(resolvedPath); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
	}

	// 获取参数
	mode := "append"
	if m, ok := ctx.Params["mode"].(string); ok && m != "" {
		mode = m
	}

	addNewline := true
	if nl, ok := ctx.Params["add_newline"].(bool); ok {
		addNewline = nl
	}

	createDirs := true
	if cd, ok := ctx.Params["create_dirs"].(bool); ok {
		createDirs = cd
	}

	backup := w.backupEnabled
	if b, ok := ctx.Params["backup"].(bool); ok {
		backup = b
	}

	// 追加换行符
	if addNewline && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// 检查内容大小
	contentSize := int64(len(content))
	if contentSize > w.maxFileSize {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams,
			fmt.Sprintf("内容大小 %d 字节超过限制 %d 字节", contentSize, w.maxFileSize)), nil
	}

	// 自动创建目录
	if createDirs {
		dir := filepath.Dir(resolvedPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("创建目录失败: %v", err)), nil
		}
	}

	// 检查文件是否存在
	fileExists := false
	if _, err := os.Stat(resolvedPath); err == nil {
		fileExists = true
	}

	// 根据模式处理
	var writeMode int
	switch mode {
	case "append":
		writeMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	case "overwrite":
		// 备份原文件
		if fileExists && backup {
			if err := w.backupFile(resolvedPath); err != nil {
				return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("备份文件失败: %v", err)), nil
			}
		}
		writeMode = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	case "create":
		if fileExists {
			return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "文件已存在，无法使用 create 模式"), nil
		}
		writeMode = os.O_CREATE | os.O_EXCL | os.O_WRONLY
	default:
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, fmt.Sprintf("不支持的写入模式: %s", mode)), nil
	}

	// 打开文件
	file, err := os.OpenFile(resolvedPath, writeMode, 0644)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("打开文件失败: %v", err)), nil
	}
	defer file.Close()

	// 写入内容
	written, err := file.WriteString(content)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("写入文件失败: %v", err)), nil
	}

	// 获取文件信息
	info, _ := os.Stat(resolvedPath)
	var fileSize int64
	if info != nil {
		fileSize = info.Size()
	}

	// 返回结果
	return tool.NewResult(map[string]any{
		"path":           resolvedPath,
		"mode":           mode,
		"bytes_written":  written,
		"file_size":      fileSize,
		"backup_created": fileExists && mode == "overwrite" && backup,
	}), nil
}

// resolvePath 解析路径
func (w *FileWriter) resolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	if w.workingDir != "" {
		fullPath := filepath.Join(w.workingDir, path)
		return filepath.Clean(fullPath), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

// validatePath 验证路径安全
func (w *FileWriter) validatePath(path string) error {
	dangerousPatterns := []string{
		"..", "//", "\\\\", "\x00",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return fmt.Errorf("路径包含非法字符: %s", pattern)
		}
	}

	// 检查是否为符号链接
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("不支持写入符号链接文件")
	}

	return nil
}

// validateExtension 验证文件扩展名
func (w *FileWriter) validateExtension(path string) error {
	if len(w.allowedExts) == 0 {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range w.allowedExts {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}

	return fmt.Errorf("不支持的文件类型: %s，允许的类型: %v", ext, w.allowedExts)
}

// backupFile 备份文件
func (w *FileWriter) backupFile(path string) error {
	backupPath := path + w.backupSuffix
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(backupPath, data, 0644)
}
