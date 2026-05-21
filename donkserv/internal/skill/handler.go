package skill

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

// Handler Skill HTTP 处理器
type Handler struct {
	service *Service
}

// NewHandler 创建 Skill 处理器
// 参数:
//   - service: Skill 服务实例
//
// 返回:
//   - *Handler: 处理器实例
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes 注册路由
// 参数:
//   - r: Gin 引擎
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/skills")
	{
		api.GET("", h.List)
		api.POST("/rescan", h.Rescan)

		skill := api.Group("/:name")
		{
			skill.GET("", h.Get)
			skill.DELETE("", h.Delete)
			skill.POST("/enable", h.Enable)
			skill.POST("/disable", h.Disable)
			skill.GET("/instructions", h.GetInstructions)
			skill.GET("/scripts", h.ListScripts)
			skill.GET("/scripts/:script", h.GetScriptContent)
		}
	}
}

// List 获取 Skill 列表
// GET /api/v1/skills
// 排序规则：先按启用状态（启用的在前），再按创建时间倒序（新的在前）
func (h *Handler) List(c *gin.Context) {
	skills, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 排序：先按启用状态（启用的在前），再按创建时间倒序（新的在前）
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Enabled != skills[j].Enabled {
			return skills[i].Enabled // 启用的排在前面
		}
		return skills[i].CreatedAt.After(skills[j].CreatedAt) // 创建时间倒序
	})

	c.JSON(http.StatusOK, gin.H{
		"data":  skills,
		"total": len(skills),
	})
}

// Get 获取 Skill 详情
// GET /api/v1/skills/:name
func (h *Handler) Get(c *gin.Context) {
	name := c.Param("name")

	skill, err := h.service.Get(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, skill)
}

// Enable 启用 Skill
// POST /api/v1/skills/:name/enable
func (h *Handler) Enable(c *gin.Context) {
	name := c.Param("name")

	if err := h.service.Enable(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Skill 已启用",
	})
}

// Disable 禁用 Skill
// POST /api/v1/skills/:name/disable
func (h *Handler) Disable(c *gin.Context) {
	name := c.Param("name")

	if err := h.service.Disable(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Skill 已禁用",
	})
}

// Delete 删除 Skill
// DELETE /api/v1/skills/:name
func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("name")

	if err := h.service.Delete(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Skill 已删除",
	})
}

// Rescan 重新扫描文件系统
// POST /api/v1/skills/rescan
func (h *Handler) Rescan(c *gin.Context) {
	if err := h.service.Rescan(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "扫描完成",
	})
}

// GetInstructions 获取 Skill 指令
// GET /api/v1/skills/:name/instructions
func (h *Handler) GetInstructions(c *gin.Context) {
	name := c.Param("name")

	instructions, err := h.service.GetInstructions(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instructions": instructions,
	})
}

// ListScripts 获取脚本列表
// GET /api/v1/skills/:name/scripts
func (h *Handler) ListScripts(c *gin.Context) {
	name := c.Param("name")

	scripts, err := h.service.ListScripts(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scripts": scripts,
	})
}

// GetScriptContent 获取脚本内容
// GET /api/v1/skills/:name/scripts/:script
func (h *Handler) GetScriptContent(c *gin.Context) {
	skillName := c.Param("name")
	scriptName := c.Param("script")

	content, err := h.service.GetScriptContent(skillName, scriptName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":    scriptName,
		"content": content,
	})
}
