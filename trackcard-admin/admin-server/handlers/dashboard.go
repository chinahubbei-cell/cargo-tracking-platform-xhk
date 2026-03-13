package handlers

import (
	"net/http"
	"time"

	"trackcard-admin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GetStats 获取仪表盘统计
func (h *DashboardHandler) GetStats(c *gin.Context) {
	var orgTotal, orgActive, orgExpiring, orgExpired int64
	var deviceTotal, deviceInStock, deviceAllocated, deviceActivated int64

	// 组织统计
	h.db.Model(&models.Organization{}).Count(&orgTotal)
	h.db.Model(&models.Organization{}).Where("service_status = ?", "active").Count(&orgActive)

	expireDate := time.Now().AddDate(0, 0, 30)
	h.db.Model(&models.Organization{}).
		Where("service_end IS NOT NULL AND service_end <= ? AND service_status = ?", expireDate, "active").
		Count(&orgExpiring)
	h.db.Model(&models.Organization{}).Where("service_status = ?", "expired").Count(&orgExpired)

	// 设备统计
	h.db.Model(&models.HardwareDevice{}).Count(&deviceTotal)
	h.db.Model(&models.HardwareDevice{}).Where("status = ?", "in_stock").Count(&deviceInStock)
	h.db.Model(&models.HardwareDevice{}).Where("status = ?", "allocated").Count(&deviceAllocated)
	h.db.Model(&models.HardwareDevice{}).Where("status = ?", "activated").Count(&deviceActivated)

	// 即将到期组织
	var expiringOrgs []models.Organization
	h.db.Where("service_end IS NOT NULL AND service_end <= ? AND service_status = ?",
		expireDate, "active").Order("service_end ASC").Limit(5).Find(&expiringOrgs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"organizations": gin.H{
				"total":    orgTotal,
				"active":   orgActive,
				"expiring": orgExpiring,
				"expired":  orgExpired,
			},
			"devices": gin.H{
				"total":     deviceTotal,
				"in_stock":  deviceInStock,
				"allocated": deviceAllocated,
				"activated": deviceActivated,
			},
			"expiring_orgs": expiringOrgs,
		},
	})
}
