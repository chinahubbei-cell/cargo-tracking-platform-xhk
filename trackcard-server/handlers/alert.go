package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/utils"
)

type AlertHandler struct {
	db *gorm.DB
}

func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{db: db}
}

func (h *AlertHandler) List(c *gin.Context) {
	status := c.Query("status")
	severity := c.Query("severity")
	alertType := c.Query("type")

	query := h.db.Model(&models.Alert{}).Preload("Device").Preload("Shipment")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if alertType != "" {
		query = query.Where("type = ?", alertType)
	}

	var alerts []models.Alert
	if err := query.Order("created_at DESC").Find(&alerts).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, alerts)
}

func (h *AlertHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var alert models.Alert
	if err := h.db.Preload("Device").Preload("Shipment").First(&alert, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "预警不存在")
		return
	}

	utils.SuccessResponse(c, alert)
}

func (h *AlertHandler) Create(c *gin.Context) {
	var req models.AlertCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的预警信息")
		return
	}

	alert := models.Alert{
		DeviceID:   req.DeviceID,
		ShipmentID: req.ShipmentID,
		Type:       req.Type,
		Severity:   req.Severity,
		Title:      req.Title,
		Message:    req.Message,
		Location:   req.Location,
		Status:     "pending",
	}

	if alert.Severity == "" {
		alert.Severity = "warning"
	}

	if err := h.db.Create(&alert).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.CreatedResponse(c, alert)
}

func (h *AlertHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var alert models.Alert
	if err := h.db.First(&alert, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "预警不存在")
		return
	}

	var req models.AlertUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求数据")
		return
	}

	updates := make(map[string]interface{})
	if req.Status != nil {
		updates["status"] = *req.Status
		if *req.Status == "resolved" {
			now := time.Now()
			updates["resolved_at"] = now
		}
	}
	if req.Severity != nil {
		updates["severity"] = *req.Severity
	}
	if req.Message != nil {
		updates["message"] = *req.Message
	}

	if err := h.db.Model(&alert).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.First(&alert, "id = ?", id)
	utils.SuccessResponse(c, alert)
}

func (h *AlertHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&models.Alert{}, "id = ?", id).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"success": true})
}

func (h *AlertHandler) Resolve(c *gin.Context) {
	id := c.Param("id")

	var alert models.Alert
	if err := h.db.First(&alert, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "预警不存在")
		return
	}

	now := time.Now()
	h.db.Model(&alert).Updates(map[string]interface{}{
		"status":      "resolved",
		"resolved_at": now,
	})

	h.db.First(&alert, "id = ?", id)
	utils.SuccessResponse(c, alert)
}

func (h *AlertHandler) Stats(c *gin.Context) {
	var pending, warning, critical int64

	h.db.Model(&models.Alert{}).Where("status = ?", "pending").Count(&pending)
	h.db.Model(&models.Alert{}).Where("status = ? AND severity = ?", "pending", "warning").Count(&warning)
	h.db.Model(&models.Alert{}).Where("status = ? AND severity = ?", "pending", "critical").Count(&critical)

	utils.SuccessResponse(c, gin.H{
		"total":    pending,
		"warning":  warning,
		"critical": critical,
	})
}
