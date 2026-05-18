package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// CRLF Windows 风格换行符 \r\n
	CRLF = "\r\n"
	// LF Unix/Linux 风格换行符 \n
	LF = "\n"
	// CR 旧 Mac 风格换行符 \r
	CR = "\r"
)

// FileLoader 文件加载器
// 负责从工作目录加载提示词文件
// 支持从多个路径搜索文件，并提供缓存功能
type FileLoader struct {
	workspace string            // 工作目录路径
	cache     map[string]string // 内容缓存
}

// NewFileLoader 创建文件加载器
// workspace: 工作目录路径
func NewFileLoader(workspace string) *FileLoader {
	return &FileLoader{
		workspace: workspace,
		cache:     make(map[string]string),
	}
}

// Load 加载文件内容
// name: 文件名（不含扩展名）
// 返回文件内容，如果文件不存在则返回错误
func (l *FileLoader) Load(name string) (string, error) {
	// 先检查缓存
	if cached, ok := l.cache[name]; ok {
		return cached, nil
	}

	// 搜索可能的文件路径
	paths := []string{
		filepath.Join(l.workspace, "prompt", name+".md"),
		filepath.Join(l.workspace, name+".md"),
		filepath.Join(l.workspace, "prompt", name),
	}

	// 遍历搜索路径
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil {
			result := string(content)
			result = NormalizeLineBreaks(result)
			// 存入缓存
			l.cache[name] = result
			return result, nil
		}
	}

	return "", fmt.Errorf("文件不存在: %s", name)
}

// Save 保存文件内容
// name: 文件名（不含扩展名）
// content: 文件内容
// 返回保存过程中发生的错误
func (l *FileLoader) Save(name, content string) error {
	// 确保目录存在
	promptDir := filepath.Join(l.workspace, "prompt")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	path := filepath.Join(promptDir, name+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	// 更新缓存
	l.cache[name] = content
	return nil
}

// Exists 检查文件是否存在
// name: 文件名（不含扩展名）
// 返回是否存在
func (l *FileLoader) Exists(name string) bool {
	paths := []string{
		filepath.Join(l.workspace, "prompt", name+".md"),
		filepath.Join(l.workspace, name+".md"),
		filepath.Join(l.workspace, "prompt", name),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// ListFiles 列出所有提示词文件
// 返回文件名列表（不含扩展名）
func (l *FileLoader) ListFiles() ([]string, error) {
	promptDir := filepath.Join(l.workspace, "prompt")
	entries, err := os.ReadDir(promptDir)
	if err != nil {
		// 目录不存在时返回空列表
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// 收集所有 .md 文件
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}
	return files, nil
}

// ClearCache 清除所有缓存
func (l *FileLoader) ClearCache() {
	l.cache = make(map[string]string)
}

// NormalizeLineBreaks 规范化换行符
// 将不同平台的换行符统一转换为 Unix 风格 (\n)
// 处理 Windows (\r\n)、Unix (\n)、旧 Mac (\r) 的换行符差异
func NormalizeLineBreaks(content string) string {
	content = strings.ReplaceAll(content, CRLF, LF)
	content = strings.ReplaceAll(content, CR, LF)
	return content
}

// EnsureParagraphBreaks 确保段落之间有双换行符
// Markdown 格式中，段落之间需要用双换行符分隔
// 此函数将单换行转换为双换行，确保 Markdown 正确渲染
func EnsureParagraphBreaks(content string) string {
	content = NormalizeLineBreaks(content)
	content = strings.ReplaceAll(content, LF+LF, "\n\n")
	re := regexp.MustCompile(`([^\n])\n([^\n])`)
	return re.ReplaceAllString(content, "$1\n\n$2")
}
