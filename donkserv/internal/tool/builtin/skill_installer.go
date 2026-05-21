package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

const (
	defaultSkillInstallMaxArchiveSize = 50 * 1024 * 1024
	defaultSkillInstallMaxFileCount   = 1000
)

var officialSkillInstallCommandRegex = regexp.MustCompile(`npx\s+skills\s+add\s+["']?(https://github\.com/[^\s"'<>]+)["']?\s+--skill\s+["']?([A-Za-z0-9._-]+)["']?`)

// SkillInstaller Skill安装工具
// 用于从 officialskills.sh 详情页解析安装来源，并把远程GitHub仓库中的指定Skill安装到本地Skill目录
type SkillInstaller struct {
	client         *http.Client // HTTP客户端
	skillDir       string       // 本地Skill根目录
	maxArchiveSize int64        // 最大zip包大小（字节）
	maxFileCount   int          // 最大解压文件数量
}

// SkillInstallerOption Skill安装工具配置选项
type SkillInstallerOption func(*SkillInstaller)

// WithSkillInstallerMaxArchiveSize 设置最大zip包大小
func WithSkillInstallerMaxArchiveSize(size int64) SkillInstallerOption {
	return func(i *SkillInstaller) {
		if size > 0 {
			i.maxArchiveSize = size
		}
	}
}

// WithSkillInstallerMaxFileCount 设置最大解压文件数量
func WithSkillInstallerMaxFileCount(count int) SkillInstallerOption {
	return func(i *SkillInstaller) {
		if count > 0 {
			i.maxFileCount = count
		}
	}
}

// NewSkillInstaller 创建Skill安装工具
func NewSkillInstaller(skillDir string, opts ...SkillInstallerOption) *SkillInstaller {
	i := &SkillInstaller{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		skillDir:       skillDir,
		maxArchiveSize: defaultSkillInstallMaxArchiveSize,
		maxFileCount:   defaultSkillInstallMaxFileCount,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Name 返回工具名称
func (i *SkillInstaller) Name() string {
	return "skill_installer"
}

// Description 返回工具描述
func (i *SkillInstaller) Description() string {
	return "从 officialskills.sh 详情页或GitHub仓库安装Skill到本地Skill目录，支持覆盖控制和安全解压。"
}

// Version 返回版本
func (i *SkillInstaller) Version() string {
	return "1.0.0"
}

// Category 返回分类
func (i *SkillInstaller) Category() string {
	return string(tool.CategoryUtility)
}

// Parameters 返回参数定义
func (i *SkillInstaller) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"url": {
			Type:        "string",
			Description: "officialskills.sh详情页URL，例如 https://officialskills.sh/anthropics/skills/pdf",
		},
		"repo_url": {
			Type:        "string",
			Description: "GitHub仓库URL，例如 https://github.com/anthropics/skills；提供url时可省略",
		},
		"skill": {
			Type:        "string",
			Description: "要安装的Skill名称；提供officialskills.sh详情页url时可省略",
		},
		"overwrite": {
			Type:        "boolean",
			Description: "本地Skill已存在时是否覆盖，默认false",
			Default:     false,
		},
	}
	return schema
}

// Execute 执行Skill安装
func (i *SkillInstaller) Execute(ctx *tool.Context) (*tool.Result, error) {
	startTime := time.Now()

	installURL, _ := ctx.Params["url"].(string)
	repoURL, _ := ctx.Params["repo_url"].(string)
	skillName, _ := ctx.Params["skill"].(string)
	overwrite, _ := ctx.Params["overwrite"].(bool)

	installURL = cleanInstallerParam(installURL)
	repoURL = cleanInstallerParam(repoURL)
	skillName = cleanInstallerParam(skillName)

	if installURL == "" && repoURL == "" && skillName == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "必须提供officialskills.sh详情页url、skill名称，或同时提供repo_url和skill"), nil
	}

	var err error
	if installURL == "" && repoURL == "" && skillName != "" {
		installURL, err = i.findOfficialSkillURL(ctx.Values, skillName)
		if err != nil {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
		}
	}
	if installURL != "" {
		repoURL, skillName, err = i.resolveOfficialSkill(ctx.Values, installURL)
		if err != nil {
			return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
		}
	}

	if repoURL == "" || skillName == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "repo_url和skill不能为空"), nil
	}
	if err := validateGitHubRepoURL(repoURL); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
	}
	if err := validateSkillInstallName(skillName); err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, err.Error()), nil
	}

	installedPath, fileCount, branch, err := i.installFromGitHub(ctx.Values, repoURL, skillName, overwrite)
	installSource := "github_archive"
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
	}

	result := tool.NewResult(map[string]any{
		"name":           skillName,
		"detail_url":     installURL,
		"repo_url":       repoURL,
		"branch":         branch,
		"install_source": installSource,
		"installed_path": installedPath,
		"files":          fileCount,
		"overwritten":    overwrite,
		"duration_ms":    time.Since(startTime).Milliseconds(),
	})
	result.SetExecutionTime(time.Since(startTime))
	return result, nil
}

// findOfficialSkillURL 根据Skill名称从 officialskills.sh 首页索引中查找详情页URL
func (i *SkillInstaller) findOfficialSkillURL(ctx context.Context, skillName string) (string, error) {
	body, err := i.fetchText(ctx, officialSkillsIndexURL, 2*1024*1024)
	if err != nil {
		return "", err
	}
	entries := parseOfficialSkillsIndex(body)
	matches := filterOfficialSkillEntries(entries, skillName, 5)
	for _, entry := range matches {
		name, _ := entry["name"].(string)
		entryURL, _ := entry["url"].(string)
		if name == skillName && entryURL != "" {
			return entryURL, nil
		}
	}
	return "", fmt.Errorf("officialskills.sh 中未找到Skill: %s", skillName)
}

// resolveOfficialSkill 从 officialskills.sh 详情页解析GitHub仓库地址和Skill名称
func (i *SkillInstaller) resolveOfficialSkill(ctx context.Context, detailURL string) (string, string, error) {
	parsed, err := url.Parse(detailURL)
	if err != nil {
		return "", "", fmt.Errorf("无效的详情页URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() != "officialskills.sh" {
		return "", "", fmt.Errorf("仅支持officialskills.sh的https详情页URL")
	}

	body, err := i.fetchText(ctx, detailURL, 2*1024*1024)
	if err != nil {
		return "", "", err
	}

	match := officialSkillInstallCommandRegex.FindStringSubmatch(body)
	if len(match) == 3 {
		return strings.TrimSpace(match[1]), strings.TrimSpace(match[2]), nil
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) >= 3 && parts[1] == "skills" {
		return fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1]), parts[2], nil
	}

	return "", "", fmt.Errorf("未能从officialskills.sh详情页解析安装命令")
}

// installFromGitHub 从GitHub仓库下载zip并安装指定Skill目录
func (i *SkillInstaller) installFromGitHub(ctx context.Context, repoURL, skillName string, overwrite bool) (string, int, string, error) {
	branches := []string{"main", "master"}
	var lastErr error
	for _, branch := range branches {
		archiveURL := strings.TrimRight(repoURL, "/") + "/archive/refs/heads/" + branch + ".zip"
		archive, err := i.fetchBytes(ctx, archiveURL, i.maxArchiveSize)
		if err != nil {
			lastErr = err
			continue
		}

		installedPath, fileCount, err := i.extractSkillArchive(archive, skillName, overwrite)
		if err != nil {
			lastErr = err
			continue
		}
		return installedPath, fileCount, branch, nil
	}
	return "", 0, "", fmt.Errorf("无法从GitHub仓库 '%s' 安装Skill '%s'：%w\n\n"+
		"请检查：\n"+
		"- 仓库地址是否正确\n"+
		"- 仓库是否为公开仓库\n"+
		"- Skill是否存在于该仓库的skills/目录下", repoURL, skillName, lastErr)
}

// installFromOfficialPage 从 officialskills.sh 详情页内容生成本地Skill快照
func (i *SkillInstaller) installFromOfficialPage(ctx context.Context, detailURL, skillName string, overwrite bool, archiveErr error) (string, int, error) {
	body, err := i.fetchText(ctx, detailURL, 4*1024*1024)
	if err != nil {
		return "", 0, fmt.Errorf("GitHub仓库安装失败: %w；同时读取officialskills.sh详情页失败: %w", archiveErr, err)
	}

	content := stripHTML(body)
	content = strings.TrimSpace(content)
	if content == "" {
		return "", 0, fmt.Errorf("GitHub仓库安装失败: %w；officialskills.sh详情页内容为空", archiveErr)
	}

	// 检查内容质量：如果内容看起来像是网页抓取失败的混乱文本（没有换行、过长的一行）
	if isLowQualityContent(content) {
		return "", 0, fmt.Errorf("GitHub仓库安装失败: %w\n\n"+
			"从 officialskills.sh 抓取的内容质量不佳，可能是：\n"+
			"1. 该 skill 在 officialskills.sh 上不存在或已被移除\n"+
			"2. 页面结构发生变化，无法正确解析\n"+
			"3. 请检查页面 %s 是否可以正常访问\n\n"+
			"建议：尝试直接从 GitHub 仓库安装，或联系 skill 维护者", archiveErr, detailURL)
	}

	metadata := buildSkillSnapshotMetadata(skillName, content)
	targetDir := filepath.Join(i.skillDir, skillName)
	if _, err := os.Stat(targetDir); err == nil && !overwrite {
		return "", 0, fmt.Errorf("Skill已存在: %s，如需覆盖请设置overwrite=true", skillName)
	}
	if err := os.MkdirAll(i.skillDir, 0755); err != nil {
		return "", 0, fmt.Errorf("创建Skill根目录失败 '%s': %w\n\n"+
			"可能的原因：\n"+
			"1. 没有目录写入权限，请检查 '%s' 是否可写\n"+
			"2. 磁盘空间不足\n"+
			"3. 路径包含非法字符\n\n"+
			"建议：尝试以管理员身份运行程序，或手动创建该目录", i.skillDir, err, i.skillDir)
	}

	tempDir, err := os.MkdirTemp(i.skillDir, ".install-"+skillName+"-")
	if err != nil {
		return "", 0, fmt.Errorf("创建临时安装目录失败 '%s': %w\n\n"+
			"可能的原因：\n"+
			"1. 没有写入权限，请检查 '%s' 目录是否可写\n"+
			"2. 磁盘空间不足\n"+
			"3. 临时目录名包含非法字符\n\n"+
			"建议：尝试以管理员身份运行程序", i.skillDir, err, i.skillDir)
	}
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.RemoveAll(tempDir)
		}
	}()

	content = fmt.Sprintf("---\nname: %s\ndescription: %s\nversion: 1.0.0\nhomepage: %s\n---\n\n%s\n", skillName, yamlQuote(metadata), yamlQuote(detailURL), content)
	if err := os.WriteFile(filepath.Join(tempDir, "SKILL.md"), []byte(content), 0644); err != nil {
		return "", 0, fmt.Errorf("写入SKILL.md失败: %w\n\n"+
			"可能的原因：\n"+
			"1. 没有文件写入权限\n"+
			"2. 磁盘空间不足\n"+
			"3. 文件被其他程序占用\n\n"+
			"建议：尝试以管理员身份运行程序", err)
	}

	if overwrite {
		if err := os.RemoveAll(targetDir); err != nil {
			return "", 0, fmt.Errorf("删除旧Skill目录失败 '%s': %w\n\n"+
				"可能的原因：\n"+
				"1. 目录或文件被其他程序占用\n"+
				"2. 没有删除权限\n"+
				"3. 目录为只读\n\n"+
				"建议：关闭可能正在使用该目录的程序，或以管理员身份运行", targetDir, err)
		}
	}
	if err := os.Rename(tempDir, targetDir); err != nil {
		return "", 0, fmt.Errorf("移动Skill目录失败 (从 '%s' 到 '%s'): %w\n\n"+
			"可能的原因：\n"+
			"1. 目标目录已存在且被占用\n"+
			"2. 没有写入权限\n"+
			"3. 跨磁盘移动失败\n\n"+
			"建议：尝试以管理员身份运行程序，或手动移动目录", tempDir, targetDir, err)
	}
	cleanupTemp = false
	return targetDir, 1, nil
}

// fetchText 拉取文本内容并限制响应大小
func (i *SkillInstaller) fetchText(ctx context.Context, reqURL string, maxSize int64) (string, error) {
	data, err := i.fetchBytes(ctx, reqURL, maxSize)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// fetchBytes 拉取二进制内容并限制响应大小
func (i *SkillInstaller) fetchBytes(ctx context.Context, reqURL string, maxSize int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", DefaultHTTPConfig.DefaultHeaders["User-Agent"])

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败 %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求 %s 返回异常状态: %s", reqURL, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("响应超过大小限制 %d 字节", maxSize)
	}
	return data, nil
}

// extractSkillArchive 从GitHub仓库zip中安全提取目标Skill目录
func (i *SkillInstaller) extractSkillArchive(archive []byte, skillName string, overwrite bool) (string, int, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return "", 0, fmt.Errorf("读取zip包失败: %w", err)
	}

	if err := os.MkdirAll(i.skillDir, 0755); err != nil {
		return "", 0, fmt.Errorf("创建Skill根目录失败 '%s': %w\n\n"+
			"可能的原因：\n"+
			"1. 没有目录写入权限，请检查 '%s' 是否可写\n"+
			"2. 磁盘空间不足\n"+
			"3. 路径包含非法字符\n\n"+
			"建议：尝试以管理员身份运行程序，或手动创建该目录", i.skillDir, err, i.skillDir)
	}

	targetDir := filepath.Join(i.skillDir, skillName)
	if _, err := os.Stat(targetDir); err == nil && !overwrite {
		return "", 0, fmt.Errorf("Skill已存在: %s，如需覆盖请设置overwrite=true", skillName)
	}

	prefix := findSkillArchivePrefix(reader.File, skillName)
	if prefix == "" {
		return "", 0, fmt.Errorf("在仓库中未找到Skill '%s'。可能的原因：\n"+
			"1. Skill名称拼写错误，请检查skill名称是否正确\n"+
			"2. 该仓库中不存在此Skill，请确认Skill是否存在于该仓库的skills目录中\n"+
			"3. Skill目录中缺少SKILL.md文件，请确保Skill目录包含有效的SKILL.md", skillName)
	}

	selectedFiles := make([]*zip.File, 0)
	for _, file := range reader.File {
		cleanName := path.Clean(file.Name)
		if strings.HasPrefix(cleanName+"/", prefix) || strings.HasPrefix(cleanName, prefix) {
			selectedFiles = append(selectedFiles, file)
		}
	}

	if len(selectedFiles) > i.maxFileCount {
		return "", 0, fmt.Errorf("Skill文件数量 %d 超过限制 %d", len(selectedFiles), i.maxFileCount)
	}

	tempDir, err := os.MkdirTemp(i.skillDir, ".install-"+skillName+"-")
	if err != nil {
		return "", 0, fmt.Errorf("创建临时安装目录失败 '%s': %w\n\n"+
			"可能的原因：\n"+
			"1. 没有写入权限，请检查 '%s' 目录是否可写\n"+
			"2. 磁盘空间不足\n"+
			"3. 临时目录名包含非法字符\n\n"+
			"建议：尝试以管理员身份运行程序", i.skillDir, err, i.skillDir)
	}
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			_ = os.RemoveAll(tempDir)
		}
	}()

	fileCount := 0
	for _, file := range selectedFiles {
		relPath := strings.TrimPrefix(path.Clean(file.Name), prefix)
		if relPath == "." || relPath == "" {
			continue
		}
		if err := i.extractZipFile(file, tempDir, relPath); err != nil {
			return "", 0, err
		}
		if !file.FileInfo().IsDir() {
			fileCount++
		}
	}

	if _, err := os.Stat(filepath.Join(tempDir, "SKILL.md")); err != nil {
		return "", 0, fmt.Errorf("安装内容缺少SKILL.md: %w", err)
	}

	if overwrite {
		if err := os.RemoveAll(targetDir); err != nil {
			return "", 0, fmt.Errorf("删除旧Skill目录失败: %w", err)
		}
	}
	if err := os.Rename(tempDir, targetDir); err != nil {
		return "", 0, fmt.Errorf("移动Skill目录失败: %w", err)
	}
	cleanupTemp = false

	return targetDir, fileCount, nil
}

// extractZipFile 安全解压单个zip文件到目标目录
func (i *SkillInstaller) extractZipFile(file *zip.File, targetDir, relPath string) error {
	if strings.Contains(relPath, "\\") || strings.HasPrefix(relPath, "../") || relPath == ".." || filepath.IsAbs(relPath) {
		return fmt.Errorf("zip文件包含不安全路径: %s", relPath)
	}

	destPath := filepath.Join(targetDir, filepath.FromSlash(relPath))
	cleanTarget := filepath.Clean(targetDir) + string(os.PathSeparator)
	cleanDest := filepath.Clean(destPath)
	if !strings.HasPrefix(cleanDest, cleanTarget) && cleanDest != filepath.Clean(targetDir) {
		return fmt.Errorf("zip文件路径越界: %s", relPath)
	}

	mode := file.FileInfo().Mode()
	if mode&os.ModeSymlink != 0 {
		return fmt.Errorf("不支持安装包含符号链接的Skill文件: %s", relPath)
	}

	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(destPath, 0755); err != nil {
			return fmt.Errorf("创建目录失败 '%s': %w\n\n"+
				"可能的原因：\n"+
				"1. 没有写入权限\n"+
				"2. 磁盘空间不足\n\n"+
				"建议：尝试以管理员身份运行程序", destPath, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("创建文件目录失败 '%s': %w\n\n"+
			"可能的原因：\n"+
			"1. 没有写入权限\n"+
			"2. 磁盘空间不足\n\n"+
			"建议：尝试以管理员身份运行程序", filepath.Dir(destPath), err)
	}

	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开zip文件失败: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("创建目标文件失败 '%s': %w\n\n"+
			"可能的原因：\n"+
			"1. 没有文件写入权限\n"+
			"2. 磁盘空间不足\n"+
			"3. 文件被其他程序占用\n\n"+
			"建议：尝试以管理员身份运行程序", destPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("写入目标文件失败 '%s': %w\n\n"+
			"可能的原因：\n"+
			"1. 磁盘空间不足\n"+
			"2. 文件被其他程序占用\n\n"+
			"建议：检查磁盘空间，或尝试以管理员身份运行程序", destPath, err)
	}
	return nil
}

// findSkillArchivePrefix 在zip包中递归查找目标Skill目录前缀
func findSkillArchivePrefix(files []*zip.File, skillName string) string {
	for _, file := range files {
		cleanName := path.Clean(file.Name)
		parts := strings.Split(cleanName, "/")
		if len(parts) < 3 || !strings.EqualFold(parts[len(parts)-1], "SKILL.md") {
			continue
		}
		if parts[len(parts)-2] == skillName {
			return strings.Join(parts[:len(parts)-1], "/") + "/"
		}
	}
	return ""
}

// cleanInstallerParam 清理模型传参中常见的Markdown包裹符号和多余空白
func cleanInstallerParam(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "` \t\r\n")
	value = strings.Trim(value, "\"'")
	value = strings.TrimSpace(value)
	return strings.Trim(value, "`")
}

// isLowQualityContent 检查从网页抓取的内容是否为低质量内容
// 当内容没有换行符且长度超过阈值时，认为是抓取失败的混乱文本
func isLowQualityContent(content string) bool {
	// 如果内容没有换行符（\n）且长度超过1000字符，可能是网页抓取失败的混乱文本
	if !strings.Contains(content, "\n") && len(content) > 1000 {
		return true
	}
	// 如果内容包含明显的网页UI元素关键词，说明是抓取了错误的页面部分
	lowQualityIndicators := []string{
		"AboutStar", "Sign in", "Back to skills", "Save skill",
		"Setup & Installation", "View on GitHub", "Show More",
		"officialskills.sh— Agent",
	}
	for _, indicator := range lowQualityIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

// buildSkillSnapshotMetadata 从详情页文本中提取适合作为Skill描述的摘要
func buildSkillSnapshotMetadata(skillName, content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "npx skills add") || strings.Contains(line, "github.com/") {
			continue
		}
		if len([]rune(line)) > 220 {
			return string([]rune(line)[:220])
		}
		return line
	}
	return "Installed skill snapshot for " + skillName
}

// yamlQuote 转义简单YAML字符串值
func yamlQuote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return "\"" + value + "\""
}

// validateGitHubRepoURL 校验GitHub仓库URL
func validateGitHubRepoURL(repoURL string) error {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("无效的GitHub仓库URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() != "github.com" {
		return fmt.Errorf("仅支持https://github.com/ 仓库URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("GitHub仓库URL必须是 https://github.com/{owner}/{repo} 格式")
	}
	return nil
}

// validateSkillInstallName 校验Skill名称，避免写入异常目录
func validateSkillInstallName(name string) error {
	if name == "" {
		return fmt.Errorf("Skill名称不能为空")
	}
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			continue
		}
		return fmt.Errorf("Skill名称包含非法字符: %c", ch)
	}
	if name == "." || name == ".." || strings.Contains(name, "..") {
		return fmt.Errorf("Skill名称不安全: %s", name)
	}
	return nil
}
