package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/longstageai/donk/donk/internal/tool"
)

const officialSkillsIndexURL = "https://officialskills.sh/"

var (
	// 匹配 officialskills.sh 首页中的 Skill 详情链接和链接内部HTML
	officialSkillLinkRegex = regexp.MustCompile(`(?is)<a\s+[^>]*href="(/[^"<>]+/skills/[^"<>]+)"[^>]*>(.*?)</a>`)
	// 从单个 Skill 条目中提取 Skill 名称
	officialSkillNameRegex = regexp.MustCompile(`(?is)<span[^>]*font-semibold[^>]*>(.*?)</span>`)
	// 从单个 Skill 条目中提取来源信息，例如 Anthropic/skills
	officialSkillSourceRegex = regexp.MustCompile(`(?is)<span[^>]*hidden\s+sm:inline[^>]*>(.*?)</span>`)
	// 从单个 Skill 条目中提取简介文本
	officialSkillDescriptionRegex = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
)

// OfficialSkillsSearch officialskills.sh 搜索工具
// 用于拉取 officialskills.sh 首页索引，并在本地按关键词搜索可用的 Skill
// 返回结果包含 Skill 名称、来源、描述、详情URL和路径信息，后续安装工具可以复用这些结果
type OfficialSkillsSearch struct {
	client       *http.Client // HTTP客户端
	indexURL     string       // officialskills.sh 索引地址
	maxIndexSize int64        // 最大索引响应大小（字节）
}

// OfficialSkillsSearchOption officialskills.sh 搜索工具配置选项
type OfficialSkillsSearchOption func(*OfficialSkillsSearch)

// WithOfficialSkillsIndexURL 设置 officialskills.sh 索引地址
func WithOfficialSkillsIndexURL(indexURL string) OfficialSkillsSearchOption {
	return func(s *OfficialSkillsSearch) {
		if strings.TrimSpace(indexURL) != "" {
			s.indexURL = strings.TrimSpace(indexURL)
		}
	}
}

// WithOfficialSkillsMaxIndexSize 设置最大索引响应大小
func WithOfficialSkillsMaxIndexSize(size int64) OfficialSkillsSearchOption {
	return func(s *OfficialSkillsSearch) {
		if size > 0 {
			s.maxIndexSize = size
		}
	}
}

// NewOfficialSkillsSearch 创建 officialskills.sh 搜索工具
func NewOfficialSkillsSearch(opts ...OfficialSkillsSearchOption) *OfficialSkillsSearch {
	s := &OfficialSkillsSearch{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		indexURL:     officialSkillsIndexURL,
		maxIndexSize: 2 * 1024 * 1024,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name 返回工具名称
func (s *OfficialSkillsSearch) Name() string {
	return "official_skills_search"
}

// Description 返回工具描述
func (s *OfficialSkillsSearch) Description() string {
	return "搜索 officialskills.sh 官方 Skill 索引，返回匹配的 Skill 名称、来源、描述和详情 URL。"
}

// Version 返回版本
func (s *OfficialSkillsSearch) Version() string {
	return "1.0.0"
}

// Category 返回分类
func (s *OfficialSkillsSearch) Category() string {
	return string(tool.CategorySearch)
}

// Parameters 返回参数定义
func (s *OfficialSkillsSearch) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"query": {
			Type:        "string",
			Description: "搜索关键词，会匹配 Skill 名称、来源和描述；为空时返回首页前若干条",
		},
		"limit": {
			Type:        "integer",
			Description: "返回结果数量，默认10，最大50",
			Default:     10,
		},
	}
	return schema
}

// Execute 执行 officialskills.sh 搜索
func (s *OfficialSkillsSearch) Execute(ctx *tool.Context) (*tool.Result, error) {
	startTime := time.Now()

	// 兼容 query 和 q 两种参数名，方便后续不同调用方复用
	query := ""
	if q, ok := ctx.Params["query"].(string); ok {
		query = strings.TrimSpace(q)
	}
	if query == "" {
		if q, ok := ctx.Params["q"].(string); ok {
			query = strings.TrimSpace(q)
		}
	}

	// 限制返回数量，避免一次性把官方索引全部塞进工具结果
	limit := parseIntParam(ctx.Params["limit"], 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// 每次执行时实时拉取首页索引，确保搜索结果与 officialskills.sh 当前内容一致
	body, err := s.fetchIndex(ctx.Values)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, err.Error()), nil
	}

	// 先解析完整索引，再按关键词做本地过滤和排序
	entries := parseOfficialSkillsIndex(body)
	matches := filterOfficialSkillEntries(entries, query, limit)

	result := tool.NewResult(map[string]any{
		"query":       query,
		"source":      s.indexURL,
		"total":       len(entries),
		"count":       len(matches),
		"results":     matches,
		"duration_ms": time.Since(startTime).Milliseconds(),
	})
	result.SetExecutionTime(time.Since(startTime))
	return result, nil
}

// fetchIndex 拉取 officialskills.sh 首页索引HTML
func (s *OfficialSkillsSearch) fetchIndex(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.indexURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建 officialskills.sh 请求失败: %w", err)
	}
	// 使用浏览器UA，避免部分站点对默认Go客户端返回差异化内容
	req.Header.Set("User-Agent", DefaultHTTPConfig.DefaultHeaders["User-Agent"])
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 officialskills.sh 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("officialskills.sh 返回异常状态: %s", resp.Status)
	}

	// 使用LimitReader限制读取量，避免异常响应占用过多内存
	data, err := io.ReadAll(io.LimitReader(resp.Body, s.maxIndexSize+1))
	if err != nil {
		return "", fmt.Errorf("读取 officialskills.sh 响应失败: %w", err)
	}
	if int64(len(data)) > s.maxIndexSize {
		return "", fmt.Errorf("officialskills.sh 响应超过大小限制 %d 字节", s.maxIndexSize)
	}
	return string(data), nil
}

// parseOfficialSkillsIndex 解析 officialskills.sh 首页HTML中的 Skill 条目
func parseOfficialSkillsIndex(html string) []map[string]any {
	matches := officialSkillLinkRegex.FindAllStringSubmatch(html, -1)
	results := make([]map[string]any, 0, len(matches))
	// 首页可能存在重复链接，使用path去重
	seen := make(map[string]struct{})

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		path := strings.TrimSpace(match[1])
		content := match[2]
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}

		// 从条目HTML中提取展示字段，同时从路径中拆分 owner/repo/skill
		name := firstOfficialSkillText(officialSkillNameRegex, content)
		source := strings.ReplaceAll(firstOfficialSkillText(officialSkillSourceRegex, content), " /", "/")
		description := firstOfficialSkillText(officialSkillDescriptionRegex, content)
		owner, repo, slug := parseOfficialSkillPath(path)
		if name == "" {
			name = slug
		}
		if source == "" && owner != "" && repo != "" {
			source = owner + "/" + repo
		}

		results = append(results, map[string]any{
			"name":        name,
			"owner":       owner,
			"repo":        repo,
			"source":      source,
			"description": description,
			"url":         "https://officialskills.sh" + path,
			"path":        path,
		})
	}

	return results
}

// firstOfficialSkillText 用指定正则提取第一个文本字段，并清理HTML标签
func firstOfficialSkillText(pattern *regexp.Regexp, content string) string {
	match := pattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	text := strings.ReplaceAll(match[1], "<!-- -->", "")
	return stripHTML(text)
}

// parseOfficialSkillPath 从 /owner/repo/skill 路径中拆分 owner、repo 和 skill slug
func parseOfficialSkillPath(path string) (string, string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}

// filterOfficialSkillEntries 根据关键词过滤并排序 Skill 条目
func filterOfficialSkillEntries(entries []map[string]any, query string, limit int) []map[string]any {
	// 空关键词时直接返回首页顺序的前limit条，便于浏览官方推荐列表
	if query == "" {
		if len(entries) > limit {
			return entries[:limit]
		}
		return entries
	}

	terms := strings.Fields(strings.ToLower(query))
	type scoredEntry struct {
		entry map[string]any
		score int
		index int
	}

	scored := make([]scoredEntry, 0)
	for i, entry := range entries {
		score := scoreOfficialSkillEntry(entry, terms)
		if score > 0 {
			scored = append(scored, scoredEntry{entry: entry, score: score, index: i})
		}
	}

	// 分数越高越靠前；同分时保留首页原始顺序，避免结果抖动
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].index < scored[j].index
		}
		return scored[i].score > scored[j].score
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	results := make([]map[string]any, 0, len(scored))
	for _, item := range scored {
		results = append(results, item.entry)
	}
	return results
}

// scoreOfficialSkillEntry 计算单个 Skill 条目与搜索词的匹配分数
func scoreOfficialSkillEntry(entry map[string]any, terms []string) int {
	name := strings.ToLower(fmt.Sprint(entry["name"]))
	source := strings.ToLower(fmt.Sprint(entry["source"]))
	description := strings.ToLower(fmt.Sprint(entry["description"]))
	path := strings.ToLower(fmt.Sprint(entry["path"]))

	score := 0
	matchedTerms := 0
	for _, term := range terms {
		matched := false
		if name == term {
			score += 100
			matched = true
		} else if strings.Contains(name, term) {
			score += 50
			matched = true
		}
		if strings.Contains(source, term) {
			score += 20
			matched = true
		}
		if strings.Contains(path, term) {
			score += 15
			matched = true
		}
		if strings.Contains(description, term) {
			score += 10
			matched = true
		}
		if matched {
			matchedTerms++
		}
	}
	if matchedTerms == 0 {
		return 0
	}
	if matchedTerms == len(terms) {
		score += 30
	}
	return score
}
