package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
)

// GeofenceHandler 围栏管理处理器
type GeofenceHandler struct {
	db *gorm.DB
}

// NewGeofenceHandler 创建围栏管理处理器
func NewGeofenceHandler(db *gorm.DB) *GeofenceHandler {
	return &GeofenceHandler{db: db}
}

// DiagnoseShipmentRequest 诊断请求
// 修复：添加数组长度限制防止滥用
type DiagnoseShipmentRequest struct {
	ShipmentIDs []string `json:"shipment_ids" binding:"required,max=100"`
}

// DiagnoseShipment 诊断运单围栏触发状态
// POST /api/admin/geofence/diagnose
func (h *GeofenceHandler) DiagnoseShipment(c *gin.Context) {
	var req DiagnoseShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供运单ID列表"})
		return
	}

	results := make([]gin.H, 0)

	for _, shipmentID := range req.ShipmentIDs {
		result := h.diagnoseOne(shipmentID)
		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

// diagnoseOne 诊断单个运单
func (h *GeofenceHandler) diagnoseOne(shipmentID string) gin.H {
	result := gin.H{
		"shipment_id": shipmentID,
		"checks":      []gin.H{},
		"issues":      []string{},
	}

	checks := make([]gin.H, 0)
	issues := make([]string, 0)

	// 1. 查询运单
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		result["error"] = "运单不存在"
		return result
	}

	// 2. 检查自动状态
	checks = append(checks, gin.H{"name": "自动状态开关", "value": shipment.AutoStatusEnabled, "ok": shipment.AutoStatusEnabled})
	if !shipment.AutoStatusEnabled {
		issues = append(issues, "自动状态开关未启用")
	}

	// 3. 检查运单状态
	validStatus := shipment.Status == "pending" || shipment.Status == "in_transit"
	checks = append(checks, gin.H{"name": "运单状态", "value": shipment.Status, "ok": validStatus})
	if !validStatus {
		issues = append(issues, "运单状态不适合围栏触发 (需要pending或in_transit)")
	}

	// 4. 检查设备绑定
	hasDevice := shipment.DeviceID != nil && *shipment.DeviceID != ""
	deviceID := ""
	if hasDevice {
		deviceID = *shipment.DeviceID
	}
	checks = append(checks, gin.H{"name": "设备绑定", "value": deviceID, "ok": hasDevice})
	if !hasDevice {
		issues = append(issues, "未绑定设备")
		result["checks"] = checks
		result["issues"] = issues
		return result
	}

	// 5. 检查坐标
	hasOrigin := shipment.OriginLat != nil && shipment.OriginLng != nil
	hasDest := shipment.DestLat != nil && shipment.DestLng != nil
	checks = append(checks, gin.H{"name": "发货地坐标", "ok": hasOrigin})
	checks = append(checks, gin.H{"name": "目的地坐标", "ok": hasDest})
	if !hasOrigin {
		issues = append(issues, "发货地坐标未配置")
	}
	if !hasDest {
		issues = append(issues, "目的地坐标未配置")
	}

	// 6. 检查设备状态
	var device models.Device
	if err := h.db.First(&device, "id = ?", deviceID).Error; err == nil {
		validDeviceStatus := device.Status == "online" || device.Status == "active" || device.Status == "bound"
		checks = append(checks, gin.H{"name": "设备状态", "value": device.Status, "ok": validDeviceStatus})
	}

	// 7. 检查轨迹数据
	var trackCount int64
	h.db.Model(&models.DeviceTrack{}).Where("device_id = ?", deviceID).Count(&trackCount)
	hasEnoughTracks := trackCount >= 3
	checks = append(checks, gin.H{"name": "轨迹数据", "value": trackCount, "ok": hasEnoughTracks})
	if !hasEnoughTracks {
		issues = append(issues, "轨迹点不足3条")
	}

	// 8. 检查围栏触发日志
	var logCount int64
	h.db.Model(&models.ShipmentLog{}).Where("shipment_id = ? AND action = 'geofence_trigger'", shipmentID).Count(&logCount)
	checks = append(checks, gin.H{"name": "围栏触发日志", "value": logCount, "ok": logCount > 0})

	result["checks"] = checks
	result["issues"] = issues
	result["status"] = map[bool]string{true: "正常", false: "有问题"}[len(issues) == 0]

	return result
}

// BackfillLogs 补录缺失的围栏触发日志
// POST /api/admin/geofence/backfill
func (h *GeofenceHandler) BackfillLogs(c *gin.Context) {
	// 查找缺少围栏触发日志的运单
	var shipments []models.Shipment
	subQuery := h.db.Model(&models.ShipmentLog{}).
		Select("shipment_id").
		Where("action = 'geofence_trigger'")

	err := h.db.Where("status IN ('in_transit', 'delivered') AND id NOT IN (?)", subQuery).
		Find(&shipments).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(shipments) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":          "没有需要补录的运单",
			"backfilled_count": 0,
		})
		return
	}

	backfilledCount := 0
	details := make([]gin.H, 0)

	for _, shipment := range shipments {
		shipmentDetails := gin.H{
			"shipment_id": shipment.ID,
			"status":      shipment.Status,
			"logs":        []string{},
		}
		logs := make([]string, 0)

		// 补录离开发货地日志
		if shipment.Status == "in_transit" || shipment.Status == "delivered" {
			departTime := shipment.LeftOriginAt
			if departTime == nil && shipment.ATD != nil {
				departTime = shipment.ATD
			}
			if departTime == nil {
				now := time.Now()
				departTime = &now
			}

			logEntry := models.ShipmentLog{
				ShipmentID:  shipment.ID,
				Action:      "geofence_trigger",
				Field:       "status",
				OldValue:    "pending",
				NewValue:    "in_transit",
				Description: "[补录] 设备自动触发离开发货地",
				OperatorID:  "system",
				OperatorIP:  "geofence_backfill",
				CreatedAt:   *departTime,
			}
			if err := h.db.Create(&logEntry).Error; err == nil {
				backfilledCount++
				logs = append(logs, "离开发货地")
			}
		}

		// 补录到达目的地日志
		if shipment.Status == "delivered" {
			arriveTime := shipment.ArrivedDestAt
			if arriveTime == nil && shipment.ATA != nil {
				arriveTime = shipment.ATA
			}
			if arriveTime == nil {
				now := time.Now()
				arriveTime = &now
			}

			// 修复：补录到达日志时，确保运单轨迹被截断
			if shipment.TrackEndAt == nil {
				h.db.Model(&shipment).Update("track_end_at", *arriveTime)
			}

			logEntry := models.ShipmentLog{
				ShipmentID:  shipment.ID,
				Action:      "geofence_trigger",
				Field:       "status",
				OldValue:    "in_transit",
				NewValue:    "delivered",
				Description: "[补录] 设备自动触发到达目的地",
				OperatorID:  "system",
				OperatorIP:  "geofence_backfill",
				CreatedAt:   *arriveTime,
			}
			if err := h.db.Create(&logEntry).Error; err == nil {
				backfilledCount++
				logs = append(logs, "到达目的地")
			}
		}

		shipmentDetails["logs"] = logs
		if len(logs) > 0 {
			details = append(details, shipmentDetails)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "补录完成",
		"shipments_found":  len(shipments),
		"backfilled_count": backfilledCount,
		"details":          details,
	})
}

// TriggerCheck 手动触发围栏检测
// POST /api/admin/geofence/trigger/:shipment_id
func (h *GeofenceHandler) TriggerCheck(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "运单不存在"})
		return
	}

	if shipment.DeviceID == nil || *shipment.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "运单未绑定设备"})
		return
	}

	var device models.Device
	if err := h.db.First(&device, "id = ?", *shipment.DeviceID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备不存在"})
		return
	}

	if device.Latitude == nil || device.Longitude == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备没有位置数据"})
		return
	}

	// 触发围栏检测
	if services.Geofence != nil {
		services.Geofence.CheckAndUpdateStatus(*shipment.DeviceID, *device.Latitude, *device.Longitude)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "围栏检测已触发",
		"shipment_id": shipmentID,
		"device_id":   *shipment.DeviceID,
		"location": gin.H{
			"latitude":  *device.Latitude,
			"longitude": *device.Longitude,
		},
	})
}
