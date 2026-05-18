package token

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler Token统计HTTP处理器
type Handler struct {
	stats *TokenStats
}

// NewHandler 创建Token统计处理器
// 参数:
//   - stats: Token统计实例
//
// 返回:
//   - *Handler: HTTP处理器实例
func NewHandler(stats *TokenStats) *Handler {
	return &Handler{stats: stats}
}

// RegisterRoutes 注册API路由
// 参数:
//   - engine: Gin引擎
func (h *Handler) RegisterRoutes(engine *gin.Engine) {
	api := engine.Group("/api/v1/tokens")
	{
		api.GET("/usage", h.GetUsageList)
		api.GET("/budget", h.GetBudgetStatus)
	}
}

// UsageItem Token使用记录项
type UsageItem struct {
	Date         string `json:"date"`          // 日期，格式 20060102
	TotalTokens  int    `json:"total_tokens"`  // 总Token数
	InputTokens  int    `json:"input_tokens"`  // 输入Token数
	OutputTokens int    `json:"output_tokens"` // 输出Token数
	UpdatedAt    string `json:"updated_at"`    // 更新时间，ISO8601格式
}

// UsageListResponse Token使用记录列表响应
type UsageListResponse struct {
	Items    []*UsageItem `json:"items"`     // 使用记录列表
	Total    int          `json:"total"`     // 总条数
	Page     int          `json:"page"`      // 当前页码
	PageSize int          `json:"page_size"` // 每页条数
}

// GetUsageList 获取Token使用记录列表（分页、倒序）
// 方法: GET /api/v1/tokens/usage
// 查询参数:
//   - page: 页码，从1开始，默认1
//   - page_size: 每页条数，默认20，最大100
func (h *Handler) GetUsageList(c *gin.Context) {
	// 解析分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	// 限制最大页大小
	if pageSize > 100 {
		pageSize = 100
	}

	// 获取分页数据
	usages, total, err := h.stats.GetUsageList(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取Token使用记录失败: " + err.Error(),
		})
		return
	}

	// 转换为响应格式
	items := make([]*UsageItem, 0, len(usages))
	for _, usage := range usages {
		items = append(items, &UsageItem{
			Date:         usage.Date,
			TotalTokens:  usage.TotalTokens,
			InputTokens:  usage.InputTokens,
			OutputTokens: usage.OutputTokens,
			UpdatedAt:    usage.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": UsageListResponse{
			Items:    items,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// BudgetStatusResponse Token预算状态响应
type BudgetStatusResponse struct {
	Date         string  `json:"date"`          // 当前日期，格式 20060102
	Limit        int     `json:"limit"`         // 每日限额，-1表示无限制
	Used         int     `json:"used"`          // 今日已使用Token数
	Remaining    int     `json:"remaining"`     // 剩余可用Token数，-1表示无限制
	UsagePercent float64 `json:"usage_percent"` // 使用百分比
	IsLimited    bool    `json:"is_limited"`    // 是否设置了限额
	IsExceeded   bool    `json:"is_exceeded"`   // 是否已超限
}

// GetBudgetStatus 获取今日Token预算状态
// 方法: GET /api/v1/tokens/budget
// 返回: 剩余额度、是否超限、使用量等信息
func (h *Handler) GetBudgetStatus(c *gin.Context) {
	// 获取今日日期
	date := time.Now().Format("20060102")

	// 获取每日限额
	limit := h.stats.GetDailyLimit()

	// 获取今日使用量
	used := h.stats.GetTodayUsage()

	// 获取剩余预算
	remaining := h.stats.GetRemainingBudget()

	// 检查是否已超限
	isExceeded := h.stats.IsBudgetExceeded()

	// 计算使用百分比
	var usagePercent float64
	isLimited := limit > 0
	if isLimited && limit > 0 {
		usagePercent = float64(used) / float64(limit) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": BudgetStatusResponse{
			Date:         date,
			Limit:        limit,
			Used:         used,
			Remaining:    remaining,
			UsagePercent: usagePercent,
			IsLimited:    isLimited,
			IsExceeded:   isExceeded,
		},
	})
}
