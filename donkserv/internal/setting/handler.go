package setting

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler HTTP请求处理结构体
// 封装了业务逻辑层，处理HTTP请求
type Handler struct {
	service *Service // 业务逻辑层实例
}

// NewHandler 创建HTTP请求处理实例
// 参数:
//   - service: 业务逻辑层实例
//
// 返回:
//   - *Handler: HTTP处理实例
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetConfig 获取完整配置
// 方法: GET /api/v1/config
func (h *Handler) GetConfig(c *gin.Context) {
	cfg, err := h.service.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// UpdateConfig 更新完整配置（支持部分更新）
// 方法: PUT /api/v1/config
// 只更新传入的字段，未传字段保持数据库原有值
func (h *Handler) UpdateConfig(c *gin.Context) {
	var req ConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateConfigPartial(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// GetLLMConfig 获取LLM配置
// 方法: GET /api/v1/config/llm
func (h *Handler) GetLLMConfig(c *gin.Context) {
	cfg, err := h.service.GetLLMConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// UpdateLLMConfig 更新LLM配置
// 方法: PUT /api/v1/config/llm
func (h *Handler) UpdateLLMConfig(c *gin.Context) {
	var req LLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateLLMConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// GetEmbeddingConfig 获取Embedding配置
// 方法: GET /api/v1/config/embedding
func (h *Handler) GetEmbeddingConfig(c *gin.Context) {
	cfg, err := h.service.GetEmbeddingConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// UpdateEmbeddingConfig 更新Embedding配置
// 方法: PUT /api/v1/config/embedding
func (h *Handler) UpdateEmbeddingConfig(c *gin.Context) {
	var req EmbeddingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateEmbeddingConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// GetAgentConfig 获取Agent配置
// 方法: GET /api/v1/config/agent
func (h *Handler) GetAgentConfig(c *gin.Context) {
	cfg, err := h.service.GetAgentConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// UpdateAgentConfig 更新Agent配置
// 方法: PUT /api/v1/config/agent
func (h *Handler) UpdateAgentConfig(c *gin.Context) {
	var req AgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateAgentConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// HealthCheck 健康检查（无需认证）
// 方法: GET /health
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Response JSON响应结构
type Response struct {
	Code    int         `json:"code"`           // 状态码
	Message string      `json:"message"`        // 消息
	Data    interface{} `json:"data,omitempty"` // 数据
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ToJSON 转换为JSON
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ==================== 睡眠管理接口 ====================

// GetSleepStatus 获取睡眠管理状态
// 方法: GET /api/v1/system/sleep
func (h *Handler) GetSleepStatus(c *gin.Context) {
	sm := GetSleepManager()
	status := sm.Status()
	SuccessResponse(c, status)
}

// PreventSleepRequest 阻止睡眠请求
type PreventSleepRequest struct {
	KeepDisplay bool `json:"keep_display"` // 是否保持显示器开启
}

// PreventSleepHandler 阻止系统睡眠
// 方法: POST /api/v1/system/sleep/prevent
func (h *Handler) PreventSleepHandler(c *gin.Context) {
	var req PreventSleepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sm := GetSleepManager()
	if err := sm.Prevent(req.KeepDisplay); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "阻止睡眠失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "已阻止系统睡眠",
		"data":    sm.Status(),
	})
}

// AllowSleepHandler 允许系统睡眠
// 方法: POST /api/v1/system/sleep/allow
func (h *Handler) AllowSleepHandler(c *gin.Context) {
	sm := GetSleepManager()
	if err := sm.Allow(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "恢复睡眠失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "已恢复系统睡眠",
		"data":    sm.Status(),
	})
}

// ==================== 知识库管理接口 ====================

// KnowledgeStatusResponse 知识库状态响应
type KnowledgeStatusResponse struct {
	Enabled   bool   `json:"enabled"`    // 是否启用
	Running   bool   `json:"running"`    // 是否运行中
	Message   string `json:"message"`    // 状态消息
	LastError string `json:"last_error"` // 最后错误信息
}

// knowledgeInitializer 知识库初始化器接口（由外部注入）
var knowledgeInitializer KnowledgeController

// KnowledgeController 知识库控制器接口
type KnowledgeController interface {
	Start() error
	Stop() error
	IsRunning() bool
	GetLastError() string
}

// SetKnowledgeController 设置知识库控制器
// 在应用启动时由 main 包注入
func SetKnowledgeController(controller KnowledgeController) {
	knowledgeInitializer = controller
}

// GetKnowledgeConfig 获取知识库配置
// 方法: GET /api/v1/config/knowledge
func (h *Handler) GetKnowledgeConfig(c *gin.Context) {
	cfg, err := h.service.GetKnowledgeConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}

	// 获取运行状态
	response := KnowledgeStatusResponse{
		Enabled: cfg.Enabled,
	}
	if knowledgeInitializer != nil {
		response.Running = knowledgeInitializer.IsRunning()
		response.LastError = knowledgeInitializer.GetLastError()
	}

	c.JSON(http.StatusOK, response)
}

// UpdateKnowledgeConfig 更新知识库配置（启用/禁用）
// 方法: PUT /api/v1/config/knowledge
func (h *Handler) UpdateKnowledgeConfig(c *gin.Context) {
	var req KnowledgeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新配置
	if err := h.service.UpdateKnowledgeConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 注意：配置只控制是否处理文档，不控制定时器
	// 定时器始终运行，每次执行前检查配置
	c.JSON(http.StatusOK, gin.H{
		"message": "知识库配置更新成功",
		"enabled": req.Enabled,
		"note":    "配置仅控制是否处理文档，定时器始终运行",
	})
}

// StartKnowledge 启动知识库
// 方法: POST /api/v1/knowledge/start
// 注意：定时器始终运行，此API仅用于手动触发一次构建
func (h *Handler) StartKnowledge(c *gin.Context) {
	if knowledgeInitializer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "知识库控制器未初始化"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "定时器始终运行，如需立即执行请使用手动触发接口",
		"running": knowledgeInitializer.IsRunning(),
	})
}

// StopKnowledge 停止知识库
// 方法: POST /api/v1/knowledge/stop
// 注意：定时器始终运行，无法停止。如需禁用文档处理，请更新配置
func (h *Handler) StopKnowledge(c *gin.Context) {
	if knowledgeInitializer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "知识库控制器未初始化"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "定时器始终运行，如需禁用文档处理请 PUT /api/v1/config/knowledge {enabled: false}",
		"running": knowledgeInitializer.IsRunning(),
	})
}

// GetKnowledgeStatus 获取知识库状态
// 方法: GET /api/v1/knowledge/status
func (h *Handler) GetKnowledgeStatus(c *gin.Context) {
	if knowledgeInitializer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "知识库控制器未初始化"})
		return
	}

	response := KnowledgeStatusResponse{
		Running:   knowledgeInitializer.IsRunning(),
		LastError: knowledgeInitializer.GetLastError(),
	}

	// 获取配置中的启用状态
	cfg, err := h.service.GetKnowledgeConfig()
	if err == nil && cfg != nil {
		response.Enabled = cfg.Enabled
	}

	c.JSON(http.StatusOK, response)
}
