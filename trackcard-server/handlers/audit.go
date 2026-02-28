package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
)

// AuditHandler 审计日志处理器
type AuditHandler struct {
	db *gorm.DB
}

// NewAuditHandler 创建审计日志处理器
func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// ListAuditLogs 获取审计日志列表
// GET /api/audit/logs
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	var query models.AuditLogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的查询参数"})
		return
	}

	// 默认分页
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	// 构建查询
	tx := h.db.Model(&models.AuditLog{})

	if query.UserID != "" {
		tx = tx.Where("user_id = ?", query.UserID)
	}
	if query.Action != "" {
		tx = tx.Where("action = ?", query.Action)
	}
	if query.ResourceType != "" {
		tx = tx.Where("resource_type = ?", query.ResourceType)
	}
	if query.ResourceID != "" {
		tx = tx.Where("resource_id = ?", query.ResourceID)
	}
	if query.IsShadowMode != nil {
		tx = tx.Where("is_shadow_mode = ?", *query.IsShadowMode)
	}
	if query.StartDate != "" {
		if t, err := time.Parse("2006-01-02", query.StartDate); err == nil {
			tx = tx.Where("created_at >= ?", t)
		}
	}
	if query.EndDate != "" {
		if t, err := time.Parse("2006-01-02", query.EndDate); err == nil {
			tx = tx.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	// 统计总数
	var total int64
	tx.Count(&total)

	// 分页查询
	var logs []models.AuditLog
	offset := (query.Page - 1) * query.PageSize
	if err := tx.Order("created_at DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 转换响应
	responses := make([]models.AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, log.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      query.Page,
		"page_size": query.PageSize,
		"items":     responses,
	})
}

// ListShadowLogs 获取影子操作日志列表
// GET /api/audit/shadow-logs
func (h *AuditHandler) ListShadowLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	operatorID := c.Query("operator_id")
	shadowForID := c.Query("shadow_for_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tx := h.db.Model(&models.ShadowOperationLog{})

	if operatorID != "" {
		tx = tx.Where("operator_id = ?", operatorID)
	}
	if shadowForID != "" {
		tx = tx.Where("shadow_for_id = ?", shadowForID)
	}

	var total int64
	tx.Count(&total)

	var logs []models.ShadowOperationLog
	offset := (page - 1) * pageSize
	if err := tx.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	responses := make([]models.ShadowOperationResponse, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, log.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     responses,
	})
}

// GetResourceAuditLogs 获取特定资源的审计日志
// GET /api/shipments/:id/audit
func (h *AuditHandler) GetResourceAuditLogs(c *gin.Context) {
	resourceType := "shipments" // 从路径推断
	resourceID := c.Param("id")

	var logs []models.AuditLog
	if err := h.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Limit(50).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	responses := make([]models.AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, log.ToResponse())
	}

	c.JSON(http.StatusOK, responses)
}

// GetAuditStats 获取审计统计信息
// GET /api/audit/stats
func (h *AuditHandler) GetAuditStats(c *gin.Context) {
	// 今日操作统计
	today := time.Now().Truncate(24 * time.Hour)

	var todayCount int64
	h.db.Model(&models.AuditLog{}).
		Where("created_at >= ?", today).
		Count(&todayCount)

	var todayShadowCount int64
	h.db.Model(&models.ShadowOperationLog{}).
		Where("created_at >= ?", today).
		Count(&todayShadowCount)

	// 按操作类型统计
	type ActionStat struct {
		Action models.AuditActionType `json:"action"`
		Count  int64                  `json:"count"`
	}
	var actionStats []ActionStat
	h.db.Model(&models.AuditLog{}).
		Select("action, count(*) as count").
		Where("created_at >= ?", today).
		Group("action").
		Find(&actionStats)

	// 最活跃操作者
	type OperatorStat struct {
		UserName string `json:"user_name"`
		Count    int64  `json:"count"`
	}
	var operatorStats []OperatorStat
	h.db.Model(&models.AuditLog{}).
		Select("user_name, count(*) as count").
		Where("created_at >= ?", today).
		Group("user_name").
		Order("count DESC").
		Limit(5).
		Find(&operatorStats)

	c.JSON(http.StatusOK, gin.H{
		"today_operations":        todayCount,
		"today_shadow_operations": todayShadowCount,
		"action_stats":            actionStats,
		"top_operators":           operatorStats,
	})
}

// GetShadowModeStatus 获取当前用户的影子模式状态
// GET /api/audit/shadow-mode/status
func (h *AuditHandler) GetShadowModeStatus(c *gin.Context) {
	// 检查用户是否可以使用影子模式
	role, _ := c.Get("role")
	canUseShadowMode := role == "admin" || role == "operator"

	c.JSON(http.StatusOK, gin.H{
		"can_use_shadow_mode": canUseShadowMode,
		"current_role":        role,
	})
}

// GetShadowTargets 获取可以代操作的目标列表
// GET /api/audit/shadow-mode/targets
func (h *AuditHandler) GetShadowTargets(c *gin.Context) {
	targetType := c.Query("type") // user/partner/customer

	var targets []gin.H

	switch targetType {
	case "partner":
		var partners []models.Partner
		h.db.Where("status = 'active'").Find(&partners)
		for _, p := range partners {
			targets = append(targets, gin.H{
				"id":   p.ID,
				"name": p.Name,
				"type": p.Type,
			})
		}
	default:
		// 返回普通用户列表
		var users []models.User
		h.db.Where("status = 'active' AND role != 'admin'").Find(&users)
		for _, u := range users {
			targets = append(targets, gin.H{
				"id":   u.ID,
				"name": u.Name,
				"role": u.Role,
			})
		}
	}

	c.JSON(http.StatusOK, targets)
}
