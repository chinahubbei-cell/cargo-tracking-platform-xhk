package handlers

import (
	"log"
	"math"
	"net/http"
	"strings"

	"trackcard-server/models"
	"trackcard-server/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PortHandler struct {
	db *gorm.DB
}

func NewPortHandler(db *gorm.DB) *PortHandler {
	return &PortHandler{db: db}
}

// GetPorts 获取港口列表
// @Summary 获取港口列表
// @Tags Ports
// @Param country query string false "国家代码筛选"
// @Param region query string false "区域筛选"
// @Param tier query int false "港口等级筛选"
// @Param type query string false "类型筛选"
// @Param search query string false "搜索关键词"
// @Success 200 {object} gin.H
// @Router /api/ports [get]
func (h *PortHandler) GetPorts(c *gin.Context) {
	country := c.Query("country")
	region := c.Query("region")
	tier := c.Query("tier")
	portType := c.Query("type")
	search := c.Query("search")

	// 无筛选条件时使用缓存
	if country == "" && region == "" && tier == "" && portType == "" && search == "" {
		cacheKey := services.CacheKeyPortsAll
		if cached, ok := services.Cache.Get(cacheKey); ok {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	var ports []models.Port
	query := h.db.Model(&models.Port{})

	// 国家筛选
	if country != "" {
		query = query.Where("country = ?", country)
	}

	// 区域筛选
	if region != "" {
		query = query.Where("region = ?", region)
	}

	// 等级筛选
	if tier != "" {
		query = query.Where("tier = ?", tier)
	}

	// 类型筛选
	if portType != "" {
		query = query.Where("type = ?", portType)
	}

	// 搜索
	if search != "" {
		query = query.Where("name LIKE ? OR name_en LIKE ? OR code LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 按等级和名称排序
	query = query.Order("tier ASC, name ASC")

	if err := query.Find(&ports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取港口列表失败"})
		return
	}

	result := gin.H{
		"code":  200,
		"data":  ports,
		"total": len(ports),
	}

	// 无筛选条件时写入缓存 (1小时TTL)
	if country == "" && region == "" && tier == "" && portType == "" && search == "" {
		services.Cache.Set(services.CacheKeyPortsAll, result, services.CacheTTLLong)
	}

	c.JSON(http.StatusOK, result)
}

// GetPort 获取单个港口详情
// @Summary 获取单个港口
// @Tags Ports
// @Param code path string true "港口代码"
// @Success 200 {object} gin.H
// @Router /api/ports/{code} [get]
func (h *PortHandler) GetPort(c *gin.Context) {
	code := c.Param("code")
	var port models.Port

	if err := h.db.Where("code = ?", code).First(&port).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "港口不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": port,
	})
}

// CreatePort 创建港口
// @Summary 创建港口
// @Tags Ports
// @Accept json
// @Param port body models.Port true "港口信息"
// @Success 200 {object} gin.H
// @Router /api/ports [post]
func (h *PortHandler) CreatePort(c *gin.Context) {
	var port models.Port
	if err := c.ShouldBindJSON(&port); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	if err := h.db.Create(&port).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建港口失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建成功",
		"data":    port,
	})
}

// UpdatePort 更新港口
// @Summary 更新港口
// @Tags Ports
// @Accept json
// @Param code path string true "港口代码"
// @Param port body models.Port true "港口信息"
// @Success 200 {object} gin.H
// @Router /api/ports/{code} [put]
func (h *PortHandler) UpdatePort(c *gin.Context) {
	code := c.Param("code")
	var port models.Port

	if err := h.db.Where("code = ?", code).First(&port).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "港口不存在"})
		return
	}

	if err := c.ShouldBindJSON(&port); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数错误"})
		return
	}

	if err := h.db.Save(&port).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新港口失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新成功",
		"data":    port,
	})
}

// DeletePort 删除港口
// @Summary 删除港口
// @Tags Ports
// @Param code path string true "港口代码"
// @Success 200 {object} gin.H
// @Router /api/ports/{code} [delete]
func (h *PortHandler) DeletePort(c *gin.Context) {
	code := c.Param("code")

	if err := h.db.Where("code = ?", code).Delete(&models.Port{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除港口失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// CalculateDistance 计算两港口距离
// @Summary 计算两港口距离(海里)
// @Tags Ports
// @Param from query string true "起始港口代码"
// @Param to query string true "目的港口代码"
// @Success 200 {object} gin.H
// @Router /api/ports/distance [get]
func (h *PortHandler) CalculateDistance(c *gin.Context) {
	fromCode := c.Query("from")
	toCode := c.Query("to")

	if fromCode == "" || toCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请提供起始和目的港口代码"})
		return
	}

	var fromPort, toPort models.Port

	if err := h.db.Where("code = ?", fromCode).First(&fromPort).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "起始港口不存在"})
		return
	}

	if err := h.db.Where("code = ?", toCode).First(&toPort).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "目的港口不存在"})
		return
	}

	// Haversine公式计算距离
	distanceKM := portHaversineDistance(fromPort.Latitude, fromPort.Longitude, toPort.Latitude, toPort.Longitude)
	distanceNM := distanceKM / 1.852 // 转换为海里

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"from":        fromPort,
			"to":          toPort,
			"distance_km": math.Round(distanceKM*100) / 100,
			"distance_nm": math.Round(distanceNM*100) / 100, // 海里
		},
	})
}

// portHaversineDistance 使用Haversine公式计算两点间距离(km)
func portHaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // 地球半径(km)

	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// GetPortRegions 获取港口区域列表
// @Summary 获取所有港口区域
// @Tags Ports
// @Success 200 {object} gin.H
// @Router /api/ports/regions [get]
func (h *PortHandler) GetPortRegions(c *gin.Context) {
	var regions []string
	h.db.Model(&models.Port{}).Distinct("region").Where("region != ''").Pluck("region", &regions)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": regions,
	})
}

// GetPortCountries 获取港口国家列表
// @Summary 获取所有港口国家
// @Tags Ports
// @Success 200 {object} gin.H
// @Router /api/ports/countries [get]
func (h *PortHandler) GetPortCountries(c *gin.Context) {
	var countries []string
	h.db.Model(&models.Port{}).Distinct("country").Where("country != ''").Pluck("country", &countries)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": countries,
	})
}

// GetPortGeofences 获取所有港口围栏数据
// GET /api/port-geofences
func (h *PortHandler) GetPortGeofences(c *gin.Context) {
	// 获取港口围栏
	var geofences []models.PortGeofence
	if err := h.db.Where("is_active = ?", true).Order("code ASC").Find(&geofences).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "获取港口围栏失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": geofences, "total": len(geofences)})
}

// GetPortGeofence 获取单个港口围栏
// GET /api/port-geofences/:code
func (h *PortHandler) GetPortGeofence(c *gin.Context) {
	code := c.Param("code")
	var geofence models.PortGeofence
	if err := h.db.Where("code = ?", code).First(&geofence).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "港口围栏不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": geofence})
}

// SeedPortGeofences 为所有港口创建围栏数据
func SeedPortGeofences(db *gorm.DB) error {
	// 清空现有围栏数据并重新创建
	db.Exec("DELETE FROM port_geofences")

	// 获取所有港口
	var ports []models.Port
	if err := db.Find(&ports).Error; err != nil {
		log.Printf("❌ 获取港口列表失败: %v", err)
		return err
	}
	log.Printf("📍 发现 %d 个港口需要创建围栏", len(ports))

	// 区域颜色映射
	regionColors := map[string]string{
		"East Asia":      "#1890ff",
		"Southeast Asia": "#52c41a",
		"South Asia":     "#faad14",
		"Middle East":    "#fa541c",
		"Europe":         "#13c2c2",
		"North America":  "#722ed1",
		"South America":  "#eb2f96",
		"Africa":         "#a0d911",
		"Oceania":        "#2f54eb",
	}

	// 根据港口等级设置默认半径
	tierRadius := map[int]int{
		1: 10000, // 一级港口 10km
		2: 7000,  // 二级港口 7km
		3: 5000,  // 三级港口 5km
		4: 3000,  // 四级港口 3km
	}

	for _, port := range ports {
		// 检查是否已存在围栏
		var count int64
		db.Model(&models.PortGeofence{}).Where("code = ?", port.Code).Count(&count)
		if count > 0 {
			continue
		}

		// 获取颜色
		color := regionColors[port.Region]
		if color == "" {
			color = "#1890ff"
		}

		// 获取半径
		radius := tierRadius[port.Tier]
		if radius == 0 {
			radius = 5000
		}

		geofence := models.PortGeofence{
			ID:           "port-" + strings.ToLower(port.Code),
			Code:         port.Code,
			Name:         port.Name,
			NameCN:       port.Name, // 使用Name作为NameCN
			Country:      port.Country,
			CountryCN:    port.Country, // 使用Country作为CountryCN
			GeofenceType: "circle",
			CenterLat:    port.Latitude,
			CenterLng:    port.Longitude,
			Radius:       radius,
			Color:        color,
			IsActive:     true,
		}
		db.Create(&geofence)
	}

	log.Printf("✅ 已为 %d 个港口创建围栏", len(ports))
	return nil
}
