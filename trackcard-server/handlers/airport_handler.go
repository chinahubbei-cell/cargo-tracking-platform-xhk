package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"trackcard-server/models"
	"trackcard-server/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AirportHandler struct {
	db *gorm.DB
}

func NewAirportHandler(db *gorm.DB) *AirportHandler {
	return &AirportHandler{db: db}
}

// GetAirports 获取机场列表
// @Summary 获取机场列表
// @Tags Airports
// @Param country query string false "国家代码筛选"
// @Param region query string false "区域筛选"
// @Param tier query int false "机场等级筛选"
// @Param type query string false "类型筛选"
// @Param search query string false "搜索关键词"
// @Param cargo_hub query bool false "仅显示货运枢纽"
// @Param page query int false "页码 (默认1)"
// @Param page_size query int false "每页数量 (默认50, 最大200)"
// @Success 200 {object} gin.H
// @Router /api/airports [get]
func (h *AirportHandler) GetAirports(c *gin.Context) {
	country := c.Query("country")
	region := c.Query("region")
	tier := c.Query("tier")
	airportType := c.Query("type")
	search := c.Query("search")
	cargoHub := c.Query("cargo_hub")

	// 分页参数
	page := 1
	pageSize := 50 // 默认每页50条
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 {
		if ps > 200 {
			ps = 200 // 最大200条
		}
		pageSize = ps
	}
	offset := (page - 1) * pageSize

	// 无筛选条件且第一页时使用缓存
	isFirstPage := page == 1 && pageSize == 50
	noFilters := country == "" && region == "" && tier == "" && airportType == "" && search == "" && cargoHub == ""
	if isFirstPage && noFilters {
		cacheKey := services.CacheKeyAirportsAll
		if cached, ok := services.Cache.Get(cacheKey); ok {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	var airports []models.Airport
	var total int64
	query := h.db.Model(&models.Airport{})

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
	if airportType != "" {
		query = query.Where("type = ?", airportType)
	}

	// 货运枢纽筛选
	if cargoHub == "true" || cargoHub == "1" {
		query = query.Where("is_cargo_hub = ?", true)
	}

	// 搜索
	if search != "" {
		searchLike := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(name_en) LIKE ? OR LOWER(iata_code) LIKE ? OR LOWER(city) LIKE ?",
			searchLike, searchLike, searchLike, searchLike)
	}

	// 获取总数 (count before pagination)
	query.Count(&total)

	// 排序：优先货运枢纽，然后按等级
	query = query.Order("is_cargo_hub DESC, tier ASC, annual_cargo_tons DESC")

	// 分页
	query = query.Offset(offset).Limit(pageSize)

	if err := query.Find(&airports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "获取机场列表失败"})
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	result := gin.H{
		"success":     true,
		"data":        airports,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	}

	// 无筛选条件且第一页时写入缓存 (1小时TTL)
	if isFirstPage && noFilters {
		services.Cache.Set(services.CacheKeyAirportsAll, result, services.CacheTTLLong)
	}

	c.JSON(http.StatusOK, result)
}

// GetAirport 获取单个机场详情
// @Summary 获取单个机场
// @Tags Airports
// @Param code path string true "机场IATA代码"
// @Success 200 {object} gin.H
// @Router /api/airports/{code} [get]
func (h *AirportHandler) GetAirport(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))

	var airport models.Airport
	if err := h.db.Where("iata_code = ?", code).First(&airport).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "机场不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    airport,
	})
}

// CreateAirport 创建机场
// @Summary 创建机场
// @Tags Airports
// @Accept json
// @Param airport body models.Airport true "机场信息"
// @Success 200 {object} gin.H
// @Router /api/airports [post]
func (h *AirportHandler) CreateAirport(c *gin.Context) {
	var airport models.Airport
	if err := c.ShouldBindJSON(&airport); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求数据"})
		return
	}

	// 标准化代码为大写
	airport.IATACode = strings.ToUpper(airport.IATACode)
	airport.ICAOCode = strings.ToUpper(airport.ICAOCode)

	// 检查重复
	var existing models.Airport
	if err := h.db.Where("iata_code = ?", airport.IATACode).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": "机场代码已存在"})
		return
	}

	if err := h.db.Create(&airport).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "创建机场失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    airport,
	})
}

// UpdateAirport 更新机场
// @Summary 更新机场
// @Tags Airports
// @Accept json
// @Param code path string true "机场IATA代码"
// @Param airport body models.Airport true "机场信息"
// @Success 200 {object} gin.H
// @Router /api/airports/{code} [put]
func (h *AirportHandler) UpdateAirport(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))

	var airport models.Airport
	if err := h.db.Where("iata_code = ?", code).First(&airport).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "机场不存在"})
		return
	}

	var updates models.Airport
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求数据"})
		return
	}

	// 更新字段
	if err := h.db.Model(&airport).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "更新机场失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    airport,
	})
}

// DeleteAirport 删除机场
// @Summary 删除机场
// @Tags Airports
// @Param code path string true "机场IATA代码"
// @Success 200 {object} gin.H
// @Router /api/airports/{code} [delete]
func (h *AirportHandler) DeleteAirport(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))

	result := h.db.Where("iata_code = ?", code).Delete(&models.Airport{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "删除机场失败"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "机场不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "机场已删除",
	})
}

// CalculateDistance 计算两机场距离
// @Summary 计算两机场距离(公里)
// @Tags Airports
// @Param from query string true "起始机场IATA代码"
// @Param to query string true "目的机场IATA代码"
// @Success 200 {object} gin.H
// @Router /api/airports/distance [get]
func (h *AirportHandler) CalculateDistance(c *gin.Context) {
	fromCode := strings.ToUpper(c.Query("from"))
	toCode := strings.ToUpper(c.Query("to"))

	if fromCode == "" || toCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请提供起始和目的机场代码"})
		return
	}

	var fromAirport, toAirport models.Airport
	if err := h.db.Where("iata_code = ?", fromCode).First(&fromAirport).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "起始机场不存在"})
		return
	}
	if err := h.db.Where("iata_code = ?", toCode).First(&toAirport).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "目的机场不存在"})
		return
	}

	// 计算大圆距离
	distanceKM := airportHaversineDistance(fromAirport.Latitude, fromAirport.Longitude,
		toAirport.Latitude, toAirport.Longitude)

	// 估算飞行时间 (平均巡航速度 850km/h)
	flightHours := distanceKM / 850.0

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"from":               fromAirport,
			"to":                 toAirport,
			"distance_km":        math.Round(distanceKM*100) / 100,
			"estimated_hours":    math.Round(flightHours*100) / 100,
			"estimated_duration": formatDuration(flightHours),
		},
	})
}

// airportHaversineDistance 使用Haversine公式计算两点间距离(km)
func airportHaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // km

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// formatDuration 格式化飞行时间
func formatDuration(hours float64) string {
	h := int(hours)
	m := int((hours - float64(h)) * 60)
	if h > 0 {
		return fmt.Sprintf("%d小时%d分钟", h, m)
	}
	return fmt.Sprintf("%d分钟", m)
}

// GetAirportRegions 获取机场区域列表
// @Summary 获取所有机场区域
// @Tags Airports
// @Success 200 {object} gin.H
// @Router /api/airports/regions [get]
func (h *AirportHandler) GetAirportRegions(c *gin.Context) {
	var regions []string
	h.db.Model(&models.Airport{}).Distinct("region").Where("region != ''").Pluck("region", &regions)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    regions,
	})
}

// GetAirportCountries 获取机场国家列表
// @Summary 获取所有机场国家
// @Tags Airports
// @Success 200 {object} gin.H
// @Router /api/airports/countries [get]
func (h *AirportHandler) GetAirportCountries(c *gin.Context) {
	var countries []string
	h.db.Model(&models.Airport{}).Distinct("country").Where("country != ''").Pluck("country", &countries)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    countries,
	})
}

// GetAirportGeofences 获取所有机场围栏数据
// GET /api/airport-geofences
func (h *AirportHandler) GetAirportGeofences(c *gin.Context) {
	var geofences []models.AirportGeofence
	h.db.Where("is_active = ?", true).Find(&geofences)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    geofences,
	})
}

// GetAirportGeofence 获取单个机场围栏
// GET /api/airport-geofences/:code
func (h *AirportHandler) GetAirportGeofence(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))

	var geofence models.AirportGeofence
	if err := h.db.Where("airport_code = ?", code).First(&geofence).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "机场围栏不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    geofence,
	})
}

// SeedAirportGeofences 为所有机场创建围栏数据
func SeedAirportGeofences(db *gorm.DB) error {
	var airports []models.Airport
	if err := db.Find(&airports).Error; err != nil {
		return err
	}

	for _, airport := range airports {
		var existing models.AirportGeofence
		if err := db.Where("airport_code = ?", airport.IATACode).First(&existing).Error; err == nil {
			continue // 已存在，跳过
		}

		geofence := models.AirportGeofence{
			AirportCode: airport.IATACode,
			AirportName: airport.Name,
			City:        airport.City,
			Country:     airport.Country,
			Longitude:   airport.Longitude,
			Latitude:    airport.Latitude,
			RadiusKM:    airport.GeofenceKM,
			IsActive:    true,
		}

		if err := db.Create(&geofence).Error; err != nil {
			log.Printf("创建机场围栏失败 %s: %v", airport.IATACode, err)
		}
	}

	return nil
}
