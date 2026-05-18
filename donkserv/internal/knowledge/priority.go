// knowledge 知识库模块
package knowledge

import (
	"sort"
	"time"
)

// PriorityQueue 优先级队列
// 根据文档的修改时间、大小、访问次数计算优先级
type PriorityQueue struct {
	items []*PriorityItem
}

// PriorityItem 优先级队列项
type PriorityItem struct {
	Document *ScanResult
	Score    float64
}

// NewPriorityQueue 创建优先级队列
// 返回队列实例
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items: make([]*PriorityItem, 0),
	}
}

// Add 添加文档到队列
// doc: 扫描结果
// accessCount: 访问次数（从数据库获取，新文件为0）
func (pq *PriorityQueue) Add(doc *ScanResult, accessCount int) {
	score := calculatePriority(doc, accessCount)
	pq.items = append(pq.items, &PriorityItem{
		Document: doc,
		Score:    score,
	})
}

// Pop 取出优先级最高的文档
// 返回文档和是否成功
func (pq *PriorityQueue) Pop() (*ScanResult, bool) {
	if len(pq.items) == 0 {
		return nil, false
	}

	// 按分数排序（降序）
	pq.sort()

	// 取出第一个（分数最高）
	item := pq.items[0]
	pq.items = pq.items[1:]

	return item.Document, true
}

// Len 返回队列长度
func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

// IsEmpty 检查队列是否为空
func (pq *PriorityQueue) IsEmpty() bool {
	return len(pq.items) == 0
}

// GetAll 获取所有文档（按优先级排序）
// 返回排序后的文档列表
func (pq *PriorityQueue) GetAll() []*ScanResult {
	pq.sort()

	results := make([]*ScanResult, len(pq.items))
	for i, item := range pq.items {
		results[i] = item.Document
	}

	return results
}

// sort 按分数降序排序
func (pq *PriorityQueue) sort() {
	sort.Slice(pq.items, func(i, j int) bool {
		return pq.items[i].Score > pq.items[j].Score
	})
}

// calculatePriority 计算文档优先级分数
// 分数越高优先级越高
// doc: 扫描结果
// accessCount: 访问次数
// 返回优先级分数
func calculatePriority(doc *ScanResult, accessCount int) float64 {
	now := time.Now()

	// 时间分：越新的文件分数越高
	// 使用指数衰减，24小时内为满分，之后逐渐降低
	hoursSinceModified := now.Sub(doc.ModifiedTime).Hours()
	timeScore := 100.0 * exponentialDecay(hoursSinceModified, 24)

	// 热度分：访问次数越多分数越高
	// 每次访问加10分，最高100分
	hotScore := float64(accessCount) * 10.0
	if hotScore > 100.0 {
		hotScore = 100.0
	}

	// 大小分：小文件优先处理（快速完成）
	// 小于1KB：+20分
	// 1KB-100KB：+10分
	// 100KB-1MB：+5分
	// 大于1MB：0分
	var sizeScore float64
	switch {
	case doc.FileSize < 1024:
		sizeScore = 20.0
	case doc.FileSize < 100*1024:
		sizeScore = 10.0
	case doc.FileSize < 1024*1024:
		sizeScore = 5.0
	default:
		sizeScore = 0.0
	}

	// 扩展名分：优先处理文本文件
	// .md: +5分
	// .txt: +5分
	// .docx: +3分
	// .pdf: +2分（PDF解析较慢）
	var extScore float64
	switch doc.Extension {
	case ".md", ".txt":
		extScore = 5.0
	case ".docx":
		extScore = 3.0
	case ".pdf":
		extScore = 2.0
	default:
		extScore = 0.0
	}

	// 总分 = 时间分 * 0.5 + 热度分 * 0.3 + 大小分 * 0.1 + 扩展名分 * 0.1
	totalScore := timeScore*0.5 + hotScore*0.3 + sizeScore*0.1 + extScore*0.1

	return totalScore
}

// exponentialDecay 指数衰减函数
// value: 当前值
// halfLife: 半衰期
// 返回衰减后的值（0-1之间）
func exponentialDecay(value, halfLife float64) float64 {
	if value <= 0 {
		return 1.0
	}
	if halfLife <= 0 {
		return 0.0
	}
	return 1.0 / (1.0 + value/halfLife)
}

// BuildPriorityQueue 从扫描结果构建优先级队列
// results: 扫描结果列表
// store: 知识库存储（用于查询访问次数）
// 返回优先级队列
func BuildPriorityQueue(results []*ScanResult, store Store) (*PriorityQueue, error) {
	queue := NewPriorityQueue()

	for _, result := range results {
		// 查询访问次数
		accessCount := 0
		doc, err := store.GetDocumentByPath(result.FilePath)
		if err == nil && doc != nil {
			accessCount = doc.AccessCount
		}

		queue.Add(result, accessCount)
	}

	return queue, nil
}

// PriorityBatch 优先级批次
// 用于分批处理高优先级文件
type PriorityBatch struct {
	Documents []*ScanResult
	Priority  float64 // 批次平均优先级
}

// BatchByPriority 按优先级分批
// queue: 优先级队列
// batchSize: 每批大小
// 返回批次列表
func BatchByPriority(queue *PriorityQueue, batchSize int) []*PriorityBatch {
	if queue.IsEmpty() {
		return nil
	}

	var batches []*PriorityBatch
	var currentBatch []*ScanResult
	var totalPriority float64

	for !queue.IsEmpty() {
		doc, _ := queue.Pop()
		currentBatch = append(currentBatch, doc)

		// 计算当前批次的优先级
		if len(currentBatch) >= batchSize {
			avgPriority := totalPriority / float64(len(currentBatch))
			batches = append(batches, &PriorityBatch{
				Documents: currentBatch,
				Priority:  avgPriority,
			})
			currentBatch = nil
			totalPriority = 0
		}
	}

	// 处理剩余文档
	if len(currentBatch) > 0 {
		avgPriority := totalPriority / float64(len(currentBatch))
		batches = append(batches, &PriorityBatch{
			Documents: currentBatch,
			Priority:  avgPriority,
		})
	}

	return batches
}
