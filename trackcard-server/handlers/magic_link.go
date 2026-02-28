package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
)

// MagicLinkHandler 魔术链接处理器
type MagicLinkHandler struct {
	db *gorm.DB
}

// NewMagicLinkHandler 创建魔术链接处理器
func NewMagicLinkHandler(db *gorm.DB) *MagicLinkHandler {
	return &MagicLinkHandler{db: db}
}

// CreateLink 创建魔术链接
// POST /api/magic-links
func (h *MagicLinkHandler) CreateLink(c *gin.Context) {
	var req models.CreateMagicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	svc := services.GetMagicLinkService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	resp, err := svc.CreateLink(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetLinksByShipment 获取运单的所有魔术链接
// GET /api/shipments/:id/magic-links
func (h *MagicLinkHandler) GetLinksByShipment(c *gin.Context) {
	shipmentID := c.Param("id")

	svc := services.GetMagicLinkService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	links, err := svc.GetLinksByShipment(shipmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, links)
}

// ==================== 公开接口（无需认证） ====================

// HandleMagicLink 处理魔术链接访问（返回H5页面数据）
// GET /m/:token
func (h *MagicLinkHandler) HandleMagicLink(c *gin.Context) {
	token := c.Param("token")

	svc := services.GetMagicLinkService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	page, err := svc.GetActionPage(token)
	if err != nil {
		// 返回错误页面数据
		c.JSON(http.StatusOK, gin.H{
			"valid":   false,
			"error":   err.Error(),
			"expired": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"data":  page,
	})
}

// SubmitMagicLink 提交魔术链接操作
// POST /m/:token/submit
func (h *MagicLinkHandler) SubmitMagicLink(c *gin.Context) {
	token := c.Param("token")

	var req models.SubmitMagicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	svc := services.GetMagicLinkService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	if err := svc.SubmitAction(token, &req, c.ClientIP()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "提交成功，感谢您的配合！",
	})
}

// GetActionTypes 获取所有动作类型（管理端用）
// GET /api/magic-links/action-types
func (h *MagicLinkHandler) GetActionTypes(c *gin.Context) {
	types := make([]gin.H, 0)
	for code, info := range models.ActionTypeInfo {
		types = append(types, gin.H{
			"code":        code,
			"title":       info.Title,
			"description": info.Description,
			"need_photo":  info.NeedPhoto,
			"need_gps":    info.NeedGPS,
		})
	}
	c.JSON(http.StatusOK, types)
}
