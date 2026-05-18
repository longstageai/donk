package profile

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/longstageai/donk/donk/internal/message"
	"github.com/longstageai/donk/donk/internal/model"
	"github.com/longstageai/donk/donk/internal/setting"
	"github.com/longstageai/donk/donk/internal/token"
	"github.com/longstageai/donk/donk/pkg/logger"
)

// ProfileManager 用户画像管理器
// 负责协调触发器、提取器、更新器，管理画像生命周期
// 使用 ProviderCache 模式支持 LLM 动态配置
type ProfileManager struct {
	userID        string
	trigger       *Trigger
	updater       *Updater
	storage       Storage
	providerCache *model.ProviderCache // Provider 缓存，用于动态配置
	tokenStats    *token.TokenStats    // Token 统计器

	mu         sync.RWMutex
	profile    *UserProfile
	isRunning  bool
	stopCh     chan struct{}
	extracting atomic.Int32 // 正在提取的计数，用于优雅关闭
}

// NewProfileManager 创建画像管理器
//
// 参数:
//   - userID: 用户ID
//   - providerCache: Provider 缓存
//   - storage: 存储接口
//   - tokenStats: Token 统计器（可为 nil，表示不统计）
//
// 返回:
//   - *ProfileManager: 管理器实例
func NewProfileManager(userID string, providerCache *model.ProviderCache, storage Storage, tokenStats *token.TokenStats) *ProfileManager {
	return &ProfileManager{
		userID:        userID,
		trigger:       NewTrigger(),
		updater:       NewUpdater(storage),
		storage:       storage,
		providerCache: providerCache,
		tokenStats:    tokenStats,
		stopCh:        make(chan struct{}),
	}
}

// Start 启动管理器
// 加载现有画像并启动定时任务
func (pm *ProfileManager) Start() error {
	// 加载现有画像
	ctx := context.Background()
	profile, err := pm.storage.Load(ctx, pm.userID)
	if err != nil {
		logger.Warn("加载用户画像失败，创建新画像", map[string]interface{}{
			"error": err.Error(),
		})
		profile = NewEmptyProfile(pm.userID)
	}

	pm.mu.Lock()
	pm.profile = profile
	pm.isRunning = true
	pm.mu.Unlock()

	// 启动定时任务
	go pm.run()

	return nil
}

// Stop 停止管理器
// 优雅关闭，等待正在进行的提取完成
func (pm *ProfileManager) Stop() {
	pm.mu.Lock()
	pm.isRunning = false
	pm.mu.Unlock()

	// 关闭定时触发
	close(pm.stopCh)

	// 清空触发器缓冲区，避免消息丢失
	pm.trigger.Clear()

	// 等待正在进行的提取完成（最多5秒）
	waitStart := time.Now()
	for pm.extracting.Load() > 0 && time.Since(waitStart) < 5*time.Second {
		time.Sleep(100 * time.Millisecond)
	}

	if pm.extracting.Load() > 0 {
		logger.Warn("停止画像管理器时仍有提取任务未完成", map[string]interface{}{
			"pending": pm.extracting.Load(),
		})
	} else {
		logger.Info("画像管理器已优雅停止", map[string]interface{}{
			"user_id": pm.userID,
		})
	}
}

// run 后台运行循环
func (pm *ProfileManager) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopCh:
			return
		case <-ticker.C:
			pm.checkAndExtract()
		}
	}
}

// checkAndExtract 检查并执行提取
func (pm *ProfileManager) checkAndExtract() {
	// 检查 Token 预算是否充足
	if pm.tokenStats != nil {
		if hasBudget, remaining := pm.tokenStats.CheckBudget(); !hasBudget {
			logger.Warn("Token 预算已耗尽，跳过画像提取", map[string]interface{}{
				"user_id":   pm.userID,
				"remaining": remaining,
			})
			return
		}
	}

	msgs := pm.trigger.GetPendingMessages()
	if len(msgs) == 0 {
		return
	}

	pm.extractAndUpdate(msgs)
}

// AddMessage 添加消息
// 由Agent调用，添加新消息并可能触发提取
//
// 参数:
//   - msg: 消息
func (pm *ProfileManager) AddMessage(msg message.Message) {
	triggered, msgs := pm.trigger.OnMessage(msg)
	if triggered && len(msgs) > 0 {
		go pm.extractAndUpdate(msgs)
	}
}

// extractAndUpdate 提取并更新画像
// 使用 ProviderCache 和 setting 获取最新 LLM 配置
func (pm *ProfileManager) extractAndUpdate(msgs []message.Message) {
	// 增加正在提取计数
	pm.extracting.Add(1)
	defer pm.extracting.Add(-1)

	ctx := context.Background()

	// 检查 Token 预算是否充足
	if pm.tokenStats != nil {
		if hasBudget, remaining := pm.tokenStats.CheckBudget(); !hasBudget {
			logger.Warn("Token 预算已耗尽，跳过画像提取", map[string]interface{}{
				"user_id":   pm.userID,
				"remaining": remaining,
			})
			return
		}
	}

	// 从 setting 获取 LLM 配置
	provider := setting.GetProvider()
	if provider == nil {
		logger.Error("ConfigProvider 未初始化", nil)
		return
	}

	llmCfg, err := provider.GetLLMConfig()
	if err != nil {
		logger.Error("获取 LLM 配置失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	if llmCfg == nil {
		logger.Error("LLM 配置不存在", nil)
		return
	}

	// 从 ProviderCache 获取 Adapter
	adapter, exists := pm.providerCache.Get(llmCfg.Provider)
	if !exists {
		logger.Error("Provider 不存在", map[string]interface{}{
			"provider": llmCfg.Provider,
		})
		return
	}

	// 动态设置配置（复用 Adapter，只更新参数）
	adapter.SetConfig(llmCfg.Model, llmCfg.APIKey, llmCfg.BaseURL)

	// 创建提取器
	extractor := NewExtractor(adapter)

	// 提取信息
	result, err := extractor.Extract(ctx, msgs)
	if err != nil {
		logger.Error("提取画像信息失败", map[string]interface{}{
			"error":     err.Error(),
			"msg_count": len(msgs),
		})
		return
	}

	// 记录 Token 消耗
	if pm.tokenStats != nil && result.Usage.TotalTokens > 0 {
		if err := pm.tokenStats.Record(result.Usage.PromptTokens, result.Usage.CompletionTokens); err != nil {
			logger.Warn("记录 Token 消耗失败", map[string]interface{}{
				"error": err.Error(),
			})
		}
		logger.Info("画像提取 Token 消耗", map[string]interface{}{
			"user_id":           pm.userID,
			"prompt_tokens":     result.Usage.PromptTokens,
			"completion_tokens": result.Usage.CompletionTokens,
			"total_tokens":      result.Usage.TotalTokens,
		})
	}

	// 如果没有提取到信息，跳过
	if len(result.Tags) == 0 && len(result.Preferences) == 0 {
		logger.Debug("未提取到画像信息", map[string]interface{}{
			"msg_count": len(msgs),
		})
		return
	}

	pm.mu.Lock()
	profile := pm.profile
	pm.mu.Unlock()

	// 更新画像
	if err := pm.updater.Update(ctx, profile, result); err != nil {
		logger.Error("更新画像失败", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.Info("画像更新完成", map[string]interface{}{
		"user_id":     pm.userID,
		"tags_added":  len(result.Tags),
		"prefs_added": len(result.Preferences),
		"confidence":  result.Confidence,
	})
}

// GetProfile 获取当前画像
//
// 返回:
//   - *UserProfile: 用户画像
func (pm *ProfileManager) GetProfile() *UserProfile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.profile
}

// GetProfilePrompt 获取画像Prompt文本
//
// 返回:
//   - string: Prompt文本
func (pm *ProfileManager) GetProfilePrompt() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.profile == nil {
		return ""
	}
	return pm.profile.ToPrompt()
}

// BuildPrompt 构建带画像的Prompt
//
// 参数:
//   - userInput: 用户输入
//
// 返回:
//   - string: 完整Prompt
func (pm *ProfileManager) BuildPrompt(userInput string) string {
	profilePrompt := pm.GetProfilePrompt()
	if profilePrompt == "" {
		return userInput
	}

	return profilePrompt + "\n\n## 当前输入\n" + userInput
}
