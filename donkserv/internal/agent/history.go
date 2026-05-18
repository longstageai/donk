package agent

// HistoryLoader 动态历史加载器
// 根据Token预算动态决定加载多少历史消息
type HistoryLoader struct {
	maxHistoryMessages int     // 最大历史消息数
	tokenBudget        int     // Token预算
	avgTokensPerChar   float64 // 平均每字符Token数（估算）
}

// NewHistoryLoader 创建动态历史加载器
// maxMessages: 最大历史消息数上限
// tokenBudget: Token预算
func NewHistoryLoader(maxMessages int, tokenBudget int) *HistoryLoader {
	return &HistoryLoader{
		maxHistoryMessages: maxMessages,
		tokenBudget:        tokenBudget,
		avgTokensPerChar:   0.25,
	}
}

// EstimateTokens 估算文本的Token数量
// 基于平均每字符Token数进行估算
func (h *HistoryLoader) EstimateTokens(text string) int {
	return int(float64(len(text)) * h.avgTokensPerChar)
}

// CalculateLoadCount 根据剩余预算计算应加载的历史消息数
// input: 用户输入
// remainingBudget: 剩余Token预算
// 返回: 应加载的历史消息数量
func (h *HistoryLoader) CalculateLoadCount(input string, remainingBudget int) int {
	// 估算输入占用的Token
	inputTokens := h.EstimateTokens(input)
	// 预留200Token给系统和其他开销
	availableForHistory := remainingBudget - inputTokens - 200

	if availableForHistory <= 0 {
		return 0
	}

	// 根据可用Token计算可加载的最大消息数
	// 假设每条消息平均50个字符
	maxByToken := float64(availableForHistory) / h.avgTokensPerChar

	maxMessages := h.maxHistoryMessages
	// 如果Token预算不支持这么多消息，则按比例减少
	if maxByToken < float64(maxMessages*50) {
		maxMessages = int(maxByToken / 50)
		if maxMessages < 1 {
			maxMessages = 1
		}
	}

	return maxMessages
}
