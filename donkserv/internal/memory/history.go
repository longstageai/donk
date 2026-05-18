package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// HistoryStore 历史记录存储
// 按日期分文件存储，支持关键词、标签、角色筛选
// 启动时加载最近 N 天的数据，自动清理过期文件
type HistoryStore struct {
	mu            sync.RWMutex  // 读写锁，保证并发安全
	dir           string        // 存储目录
	maxEntries    int           // 最大条目数（内存中）
	maxAgeDays    int           // 最大保留天数
	recentEntries []MemoryEntry // 最近的历史记录（内存索引）
}

// NewHistoryStore 创建历史记录存储实例
// dir: 存储目录路径
// maxEntries: 最大条目数（内存中缓存的条目数）
// maxAgeDays: 最大保留天数（超过此天数的文件会被清理）
func NewHistoryStore(dir string, maxEntries int, maxAgeDays int) (*HistoryStore, error) {
	dir = filepath.Join(dir, "history")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	store := &HistoryStore{
		dir:        dir,
		maxEntries: maxEntries,
		maxAgeDays: maxAgeDays,
	}

	// 加载最近的历史记录到内存
	if err := store.loadRecentEntries(); err != nil {
		return nil, err
	}

	return store, nil
}

// Add 添加历史记录
// 自动设置时间戳和类型，保存到对应日期的文件中
func (h *HistoryStore) Add(entry *MemoryEntry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry.Type = MemoryTypeHistory
	entry.Timestamp = time.Now()

	// 自动生成唯一标识
	if entry.Key == "" {
		entry.Key = strings.Join([]string{
			"session",
			entry.Timestamp.Format("20060102150405"),
		}, "_")
	}

	// 添加到内存索引
	h.recentEntries = append(h.recentEntries, *entry)

	// 保存到当天的文件
	return h.saveToFile(entry)
}

// GetRecent 获取最近的历史记录
// limit: 返回的条目数量限制
func (h *HistoryStore) GetRecent(limit int) ([]MemoryEntry, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 处理边界情况
	if limit <= 0 || limit > len(h.recentEntries) {
		limit = len(h.recentEntries)
	}

	if limit == 0 {
		return []MemoryEntry{}, nil
	}

	start := len(h.recentEntries) - limit
	result := make([]MemoryEntry, limit)
	copy(result, h.recentEntries[start:])

	return result, nil
}

// Search 搜索历史记录
// 支持按类型、角色、标签、关键词筛选
func (h *HistoryStore) Search(req SearchRequest) (*SearchResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var results []MemoryEntry

	// 遍历所有条目进行筛选
	for _, entry := range h.recentEntries {
		// 类型筛选
		if req.MemoryType != "" && entry.Type != req.MemoryType {
			continue
		}

		// 角色筛选（user/assistant/tool）
		if req.Role != "" && entry.Role != req.Role {
			continue
		}

		// 标签筛选
		if len(req.Tags) > 0 && !hasAnyTag(entry.Tags, req.Tags) {
			continue
		}

		// 关键词匹配（不区分大小写）
		if len(req.Keywords) > 0 && !matchKeywords(entry.Content, req.Keywords) {
			continue
		}

		results = append(results, entry)
	}

	// 应用返回数量限制
	limit := req.Limit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}

	return &SearchResult{
		Entries: results[:limit],
		Total:   len(results),
	}, nil
}

// GetBySession 获取指定会话的所有历史记录
// sessionID: 会话标识符
func (h *HistoryStore) GetBySession(sessionID string) ([]MemoryEntry, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var results []MemoryEntry
	for _, entry := range h.recentEntries {
		if entry.Metadata.SessionID == sessionID {
			results = append(results, entry)
		}
	}

	return results, nil
}

// Clear 清空所有历史记录
// 删除所有文件并清空内存索引
func (h *HistoryStore) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries, err := os.ReadDir(h.dir)
	if err != nil {
		return err
	}

	// 删除所有文件
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		os.Remove(filepath.Join(h.dir, entry.Name()))
	}

	// 清空内存索引
	h.recentEntries = make([]MemoryEntry, 0)
	return nil
}

// loadRecentEntries 加载最近的历史记录到内存
// 读取最近 maxAgeDays 天的文件，按时间排序后取最近的 maxEntries 条
func (h *HistoryStore) loadRecentEntries() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	files, err := os.ReadDir(h.dir)
	if err != nil {
		return err
	}

	var validFiles []string
	cutoffDate := time.Now().AddDate(0, 0, -h.maxAgeDays)

	// 筛选有效的历史文件（日期格式的 JSON 文件）
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		// 解析日期
		dateStr := strings.TrimSuffix(f.Name(), ".json")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// 如果文件日期早于截止日期，则删除（自动清理过期文件）
		if fileDate.Before(cutoffDate) {
			os.Remove(filepath.Join(h.dir, f.Name()))
			continue
		}

		validFiles = append(validFiles, f.Name())
	}

	// 按文件名排序（按日期排序）
	sort.Strings(validFiles)

	// 合并所有有效文件的数据
	var allEntries []MemoryEntry
	for _, filename := range validFiles {
		entries, err := h.loadFile(filepath.Join(h.dir, filename))
		if err != nil {
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	// 1. 按时间升序排序（旧的在前，新的在后）
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.Before(allEntries[j].Timestamp)
	})

	// 2. 截取最近的条目（保留最新的 N 条）
	limit := h.maxEntries
	if len(allEntries) > limit {
		allEntries = allEntries[len(allEntries)-limit:]
	}

	h.recentEntries = allEntries
	return nil
}

// loadFile 读取单个历史文件
func (h *HistoryStore) loadFile(filePath string) ([]MemoryEntry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var entries []MemoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// saveToFile 保存记录到当天的历史文件
// 文件名格式：2026-03-25.json
func (h *HistoryStore) saveToFile(entry *MemoryEntry) error {
	// 以日期作为文件名
	dateStr := entry.Timestamp.Format("2006-01-02")
	filename := dateStr + ".json"
	filePath := filepath.Join(h.dir, filename)

	var entries []MemoryEntry

	// 读取已有数据
	existingData, err := os.ReadFile(filePath)
	if err == nil {
		json.Unmarshal(existingData, &entries)
	}

	// 追加新记录
	entries = append(entries, *entry)

	// 写入文件
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func hasAnyTag(entryTags, requestTags []string) bool {
	for _, rt := range requestTags {
		for _, et := range entryTags {
			if strings.EqualFold(et, rt) {
				return true
			}
		}
	}
	return false
}

func matchKeywords(content string, keywords []string) bool {
	content = strings.ToLower(content)
	for _, kw := range keywords {
		if strings.Contains(content, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// LoadRecent7Days 加载最近7日的历史记录
// 从磁盘重新加载，包含今日，最多返回20条，按时间升序排序（旧的在前，新的在后）
func (h *HistoryStore) LoadRecent7Days() ([]MemoryEntry, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()
	var allEntries []MemoryEntry

	// 从今日向前7日，包含今日
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		filename := dateStr + ".json"
		filePath := filepath.Join(h.dir, filename)

		// 从磁盘加载文件
		entries, err := h.loadFile(filePath)
		if err != nil {
			continue
		}

		allEntries = append(allEntries, entries...)
	}

	// 按时间升序排序（旧的在前，新的在后）
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.Before(allEntries[j].Timestamp)
	})

	// 限制最多20条
	if len(allEntries) > 20 {
		allEntries = allEntries[len(allEntries)-20:]
	}

	return allEntries, nil
}
