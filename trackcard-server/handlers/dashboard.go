package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	// 检查缓存
	cacheKey := services.CacheKeyDashboardStats
	if cached, ok := services.Cache.Get(cacheKey); ok {
		utils.SuccessResponse(c, cached)
		return
	}

	var totalDevices, onlineDevices, offlineDevices int64
	var totalShipments, pendingShipments, inTransitShipments, deliveredShipments int64
	var pendingAlerts, criticalAlerts int64

	// 设备统计
	h.db.Model(&models.Device{}).Count(&totalDevices)
	h.db.Model(&models.Device{}).Where("status = ?", "online").Count(&onlineDevices)
	h.db.Model(&models.Device{}).Where("status = ?", "offline").Count(&offlineDevices)

	// 运单统计
	h.db.Model(&models.Shipment{}).Count(&totalShipments)
	h.db.Model(&models.Shipment{}).Where("status = ?", "pending").Count(&pendingShipments)
	h.db.Model(&models.Shipment{}).Where("status = ?", "in_transit").Count(&inTransitShipments)
	h.db.Model(&models.Shipment{}).Where("status = ?", "delivered").Count(&deliveredShipments)

	// 预警统计
	h.db.Model(&models.Alert{}).Where("status = ?", "pending").Count(&pendingAlerts)
	h.db.Model(&models.Alert{}).Where("status = ? AND severity = ?", "pending", "critical").Count(&criticalAlerts)

	stats := gin.H{
		"devices": gin.H{
			"total":   totalDevices,
			"online":  onlineDevices,
			"offline": offlineDevices,
		},
		"shipments": gin.H{
			"total":      totalShipments,
			"pending":    pendingShipments,
			"in_transit": inTransitShipments,
			"delivered":  deliveredShipments,
		},
		"alerts": gin.H{
			"pending":  pendingAlerts,
			"critical": criticalAlerts,
		},
	}

	// 写入缓存 (30秒 TTL)
	services.Cache.Set(cacheKey, stats, services.CacheTTLShort)

	utils.SuccessResponse(c, stats)
}

func (h *DashboardHandler) GetLocations(c *gin.Context) {
	var devices []models.Device
	h.db.Where("latitude IS NOT NULL AND longitude IS NOT NULL").Find(&devices)

	locations := make([]gin.H, 0, len(devices))
	for _, d := range devices {
		if d.Latitude != nil && d.Longitude != nil {
			locations = append(locations, gin.H{
				"id":          d.ID,
				"name":        d.Name,
				"type":        d.Type,
				"status":      d.Status,
				"latitude":    *d.Latitude,
				"longitude":   *d.Longitude,
				"battery":     d.Battery,
				"temperature": d.Temperature,
				"humidity":    d.Humidity,
				"last_update": d.LastUpdate,
			})
		}
	}

	utils.SuccessResponse(c, locations)
}

func (h *DashboardHandler) GetRecentAlerts(c *gin.Context) {
	var alerts []models.Alert
	h.db.Preload("Device").
		Where("status = ?", "pending").
		Order("created_at DESC").
		Limit(10).
		Find(&alerts)

	utils.SuccessResponse(c, alerts)
}

func (h *DashboardHandler) GetRecentShipments(c *gin.Context) {
	var shipments []models.Shipment
	h.db.Preload("Device").
		Order("created_at DESC").
		Limit(10).
		Find(&shipments)

	utils.SuccessResponse(c, shipments)
}

// GetSyncStats 获取数据同步统计信息
func (h *DashboardHandler) GetSyncStats(c *gin.Context) {
	var totalTracks int64
	var todayTracks int64
	var deviceCount int64
	var onlineDevices int64

	h.db.Model(&models.DeviceTrack{}).Count(&totalTracks)
	h.db.Model(&models.DeviceTrack{}).Where("synced_at >= CURRENT_DATE").Count(&todayTracks)
	h.db.Model(&models.Device{}).Count(&deviceCount)
	h.db.Model(&models.Device{}).Where("status = ?", "online").Count(&onlineDevices)

	utils.SuccessResponse(c, gin.H{
		"total_tracks":   totalTracks,
		"today_tracks":   todayTracks,
		"device_count":   deviceCount,
		"online_devices": onlineDevices,
		"sync_interval": gin.H{
			"active":  "1分钟",
			"idle":    "5分钟",
			"offline": "30分钟",
		},
		"retention_days": 365,
	})
}
