package handlers

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

type DeviceStopHandler struct {
	db                *gorm.DB
	deviceStopService *services.DeviceStopService
}

const (
	maxTransitReverseGeocodeSamples = 180
	minTransitSampleInterval        = 45 * time.Minute
	minTransitSampleDistanceKM      = 80.0
	transitCacheForceRefreshAfter   = 6 * time.Hour
	minTransitCityNodeCount         = 2
	stopReanalyzeMinInterval        = 10 * time.Minute
	stopReanalyzeLookback           = 6 * time.Hour
)

var shipmentStopRefreshLock sync.Map

func NewDeviceStopHandler(db *gorm.DB) *DeviceStopHandler {
	return &DeviceStopHandler{
		db:                db,
		deviceStopService: services.NewDeviceStopService(db),
	}
}

// GetStopRecords 获取设备停留记录列表
// GET /api/device-stops?device_id=xxx&device_external_id=xxx&page=1&page_size=20
func (h *DeviceStopHandler) GetStopRecords(c *gin.Context) {
	deviceID := c.Query("device_id")
	deviceExternalID := c.Query("device_external_id")
	shipmentID := c.Query("shipment_id")
	status := c.Query("status")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	req := models.DeviceStopRecordListRequest{
		DeviceID:         deviceID,
		DeviceExternalID: deviceExternalID,
		ShipmentID:       shipmentID,
		Status:           status,
		StartTime:        startTime,
		EndTime:          endTime,
		Page:             page,
		PageSize:         pageSize,
	}

	records, total, err := h.deviceStopService.GetStopRecords(req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 转换为响应格式
	response := make([]models.DeviceStopRecordResponse, len(records))
	for i, record := range records {
		address := normalizeStopAddressForResponse(record.Address, record.Latitude, record.Longitude)
		response[i] = models.DeviceStopRecordResponse{
			ID:               record.ID,
			DeviceID:         record.DeviceID,
			DeviceExternalID: record.DeviceExternalID,
			ShipmentID:       record.ShipmentID,
			StartTime:        record.StartTime,
			EndTime:          record.EndTime,
			DurationSeconds:  record.DurationSeconds,
			DurationText:     record.DurationText,
			Latitude:         record.Latitude,
			Longitude:        record.Longitude,
			Address:          address,
			Status:           record.Status,
			AlertSent:        record.AlertSent,
			CreatedAt:        record.CreatedAt,
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"records":   response,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetDeviceStopStats 获取设备停留统计
// GET /api/device-stops/stats/:device_external_id
func (h *DeviceStopHandler) GetDeviceStopStats(c *gin.Context) {
	deviceExternalID := c.Param("device_external_id")
	deviceID := c.Query("device_id")

	stats, err := h.deviceStopService.GetDeviceStopStats(deviceID, deviceExternalID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetCurrentStop 获取设备当前停留记录
// GET /api/device-stops/current/:device_external_id
func (h *DeviceStopHandler) GetCurrentStop(c *gin.Context) {
	deviceExternalID := c.Param("device_external_id")

	record, err := h.deviceStopService.GetCurrentStop(deviceExternalID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.NotFound(c, "未找到当前停留记录")
		} else {
			utils.InternalError(c, err.Error())
		}
		return
	}

	address := normalizeStopAddressForResponse(record.Address, record.Latitude, record.Longitude)
	response := models.DeviceStopRecordResponse{
		ID:               record.ID,
		DeviceID:         record.DeviceID,
		DeviceExternalID: record.DeviceExternalID,
		ShipmentID:       record.ShipmentID,
		StartTime:        record.StartTime,
		EndTime:          record.EndTime,
		DurationSeconds:  record.DurationSeconds,
		DurationText:     record.DurationText,
		Latitude:         record.Latitude,
		Longitude:        record.Longitude,
		Address:          address,
		Status:           record.Status,
		AlertSent:        record.AlertSent,
		CreatedAt:        record.CreatedAt,
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetShipmentStopRecords 获取运单的停留记录
// GET /api/shipments/:id/stops
// 只返回设备绑定到解绑期间的停留记录
func (h *DeviceStopHandler) GetShipmentStopRecords(c *gin.Context) {
	// 兼容两种参数命名：当前路由使用 :id，部分旧代码使用 :shipment_id
	shipmentID := c.Param("id")
	if shipmentID == "" {
		shipmentID = c.Param("shipment_id")
	}
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		utils.InternalError(c, "查询运单信息失败: "+err.Error())
		return
	}

	// 规则优化: 查询停留记录前，按轨迹增量自动补齐停留分析，保障节点自动更新
	if err := h.ensureShipmentStopsFresh(&shipment); err != nil {
		log.Printf("[DeviceStop] ensureShipmentStopsFresh failed shipment=%s: %v", shipmentID, err)
	}

	// 构建时间过滤条件
	var startTime, endTime string
	if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() {
		startTime = shipment.DeviceBoundAt.Format("2006-01-02 15:04:05")
	}
	if shipment.DeviceUnboundAt != nil && !shipment.DeviceUnboundAt.IsZero() {
		endTime = shipment.DeviceUnboundAt.Format("2006-01-02 15:04:05")
	}

	req := models.DeviceStopRecordListRequest{
		ShipmentID: shipmentID,
		Page:       page,
		PageSize:   pageSize,
		StartTime:  startTime,
		EndTime:    endTime,
	}

	records, total, err := h.deviceStopService.GetStopRecords(req)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 转换为响应格式
	response := make([]models.DeviceStopRecordResponse, len(records))
	for i, record := range records {
		address := normalizeStopAddressForResponse(record.Address, record.Latitude, record.Longitude)
		response[i] = models.DeviceStopRecordResponse{
			ID:               record.ID,
			DeviceID:         record.DeviceID,
			DeviceExternalID: record.DeviceExternalID,
			ShipmentID:       record.ShipmentID,
			StartTime:        record.StartTime,
			EndTime:          record.EndTime,
			DurationSeconds:  record.DurationSeconds,
			DurationText:     record.DurationText,
			Latitude:         record.Latitude,
			Longitude:        record.Longitude,
			Address:          address,
			Status:           record.Status,
			AlertSent:        record.AlertSent,
			CreatedAt:        record.CreatedAt,
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"records":   response,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetShipmentTransitCities 获取运单途经国家/城市节点
// GET /api/shipments/:id/transit-cities?refresh=true|false
func (h *DeviceStopHandler) GetShipmentTransitCities(c *gin.Context) {
	shipmentID := c.Param("id")
	if shipmentID == "" {
		shipmentID = c.Param("shipment_id")
	}
	if shipmentID == "" {
		utils.BadRequest(c, "缺少运单ID")
		return
	}

	refresh := strings.EqualFold(c.DefaultQuery("refresh", "false"), "true")

	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.NotFound(c, "运单不存在")
		} else {
			utils.InternalError(c, "查询运单失败: "+err.Error())
		}
		return
	}

	if !refresh {
		cached, err := h.loadTransitCityCache(shipmentID)
		if err != nil {
			utils.InternalError(c, "查询途经城市缓存失败: "+err.Error())
			return
		}
		if len(cached) > 0 {
			needRefresh, err := h.shouldRefreshTransitCityCache(&shipment, cached)
			if err != nil {
				utils.InternalError(c, "检查途经城市缓存失败: "+err.Error())
				return
			}
			if !needRefresh {
				utils.SuccessResponse(c, cached)
				return
			}
		}
	}

	cities, err := h.rebuildTransitCitiesFromTracks(&shipment)
	if err != nil {
		utils.InternalError(c, "生成途经城市失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, cities)
}

type transitGeoLookup struct {
	Country  string
	Province string
	City     string
}

type shipmentTrackSnapshot struct {
	LocateTime time.Time
	Latitude   float64
	Longitude  float64
}

func normalizeStopAddressForResponse(address string, lat, lng *float64) string {
	text := strings.TrimSpace(address)
	if lat == nil || lng == nil {
		return text
	}
	normalized := strings.TrimSpace(services.EnsureBilingualNodeAddress(text, *lat, *lng))
	if normalized == "" {
		return text
	}
	return normalized
}

func (h *DeviceStopHandler) getBindingTrackWindow(shipment *models.Shipment, binding models.ShipmentDeviceBinding) (time.Time, time.Time, bool) {
	segmentStart := resolveBindingStartTime(shipment, binding)

	segmentEnd := time.Now()
	if binding.UnboundAt != nil && !binding.UnboundAt.IsZero() {
		segmentEnd = *binding.UnboundAt
	}
	if shipment.TrackEndAt != nil && !shipment.TrackEndAt.IsZero() && shipment.TrackEndAt.Before(segmentEnd) {
		segmentEnd = *shipment.TrackEndAt
	}

	if segmentStart.After(segmentEnd) {
		return time.Time{}, time.Time{}, false
	}
	return segmentStart, segmentEnd, true
}

func (h *DeviceStopHandler) getLatestShipmentTrackSnapshot(shipment *models.Shipment, bindings []models.ShipmentDeviceBinding) (*shipmentTrackSnapshot, error) {
	var latest *shipmentTrackSnapshot

	for _, binding := range bindings {
		segmentStart, segmentEnd, ok := h.getBindingTrackWindow(shipment, binding)
		if !ok {
			continue
		}

		var track models.DeviceTrack
		err := h.db.
			Where("device_id = ? AND locate_time >= ? AND locate_time <= ?", binding.DeviceID, segmentStart, segmentEnd).
			Order("locate_time DESC").
			First(&track).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return nil, err
		}

		if latest == nil || track.LocateTime.After(latest.LocateTime) {
			latest = &shipmentTrackSnapshot{
				LocateTime: track.LocateTime,
				Latitude:   track.Latitude,
				Longitude:  track.Longitude,
			}
		}
	}

	return latest, nil
}

func (h *DeviceStopHandler) ensureShipmentStopsFresh(shipment *models.Shipment) error {
	if shipment == nil || shipment.ID == "" {
		return nil
	}

	lockValue, _ := shipmentStopRefreshLock.LoadOrStore(shipment.ID, &sync.Mutex{})
	refreshLock := lockValue.(*sync.Mutex)
	refreshLock.Lock()
	defer refreshLock.Unlock()

	bindings := h.resolveShipmentBindings(shipment)
	if len(bindings) == 0 {
		return nil
	}

	latestTrack, err := h.getLatestShipmentTrackSnapshot(shipment, bindings)
	if err != nil || latestTrack == nil {
		return err
	}

	var latestStop models.DeviceStopRecord
	err = h.db.Where("shipment_id = ?", shipment.ID).Order("updated_at DESC").First(&latestStop).Error
	hasStop := err == nil
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if hasStop &&
		!services.IsCoordinateAddress(latestStop.Address) &&
		!latestTrack.LocateTime.After(latestStop.UpdatedAt.Add(stopReanalyzeMinInterval)) {
		return nil
	}

	analysisEnd := latestTrack.LocateTime
	analysisStart := analysisEnd.Add(-stopReanalyzeLookback)
	if hasStop {
		// 回看一点窗口，避免边界段落漏判
		candidateStart := latestStop.StartTime.Add(-30 * time.Minute)
		if candidateStart.After(analysisStart) {
			analysisStart = candidateStart
		}
	}
	if shipment.CreatedAt.After(analysisStart) {
		analysisStart = shipment.CreatedAt
	}
	if shipment.LeftOriginAt != nil && !shipment.LeftOriginAt.IsZero() && shipment.LeftOriginAt.After(analysisStart) {
		analysisStart = *shipment.LeftOriginAt
	}
	if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() && shipment.DeviceBoundAt.After(analysisStart) {
		analysisStart = *shipment.DeviceBoundAt
	}

	stopService := services.NewDeviceStopService(h.db)

	for _, binding := range bindings {
		segmentStart, segmentEnd, ok := h.getBindingTrackWindow(shipment, binding)
		if !ok {
			continue
		}
		if segmentStart.Before(analysisStart) {
			segmentStart = analysisStart
		}
		if segmentEnd.After(analysisEnd) {
			segmentEnd = analysisEnd
		}
		if segmentStart.After(segmentEnd) {
			continue
		}

		var device models.Device
		if err := h.db.Select("id, external_device_id").First(&device, "id = ?", binding.DeviceID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		if device.ExternalDeviceID == nil || *device.ExternalDeviceID == "" {
			continue
		}

		if err := stopService.AnalyzeDeviceTracksAndCreateStops(
			device.ID,
			*device.ExternalDeviceID,
			shipment.ID,
			segmentStart,
			segmentEnd,
		); err != nil {
			return err
		}
	}

	return nil
}

func (h *DeviceStopHandler) shouldRefreshTransitCityCache(shipment *models.Shipment, cached []models.TransitCityResponse) (bool, error) {
	if len(cached) == 0 {
		return true, nil
	}
	if hasTransitCacheQualityIssue(cached) {
		return true, nil
	}

	bindings := h.resolveShipmentBindings(shipment)
	if len(bindings) == 0 {
		return false, nil
	}

	latestTrack, err := h.getLatestShipmentTrackSnapshot(shipment, bindings)
	if err != nil || latestTrack == nil {
		return false, err
	}

	lastCity := cached[len(cached)-1]
	if latestTrack.LocateTime.After(lastCity.EnteredAt.Add(transitCacheForceRefreshAfter)) {
		return true, nil
	}

	if !latestTrack.LocateTime.After(lastCity.EnteredAt.Add(minTransitSampleInterval)) {
		return false, nil
	}

	distance := services.CalculateDistance(
		lastCity.Latitude,
		lastCity.Longitude,
		latestTrack.Latitude,
		latestTrack.Longitude,
	)

	return distance >= minTransitSampleDistanceKM, nil
}

func hasTransitCacheQualityIssue(cached []models.TransitCityResponse) bool {
	for _, city := range cached {
		rawCity := strings.TrimSpace(city.City)
		if rawCity == "" {
			return true
		}
		normalizedCity := normalizeTransitCity(rawCity)
		if normalizedCity == "" {
			return true
		}
		if strings.TrimSpace(city.Province) == "" {
			normalizedCountry := normalizeTransitCountry(city.Country)
			if isChinaCountry(normalizedCountry) && inferTransitProvinceFromCity(normalizedCountry, normalizedCity) == "" {
				return true
			}
		}
		if hasChineseCharacters(rawCity) {
			if normalizedCity != rawCity {
				return true
			}
			if isCompositeTransitCity(rawCity) {
				return true
			}
		}
	}
	return false
}

func isCompositeTransitCity(city string) bool {
	value := strings.TrimSpace(city)
	if !hasChineseCharacters(value) {
		return false
	}
	cityIdx := strings.LastIndex(value, "市")
	if cityIdx <= 0 {
		return false
	}
	if districtIdx := strings.IndexAny(value, "区县旗"); districtIdx >= 0 && districtIdx < cityIdx {
		return true
	}
	return false
}

func (h *DeviceStopHandler) rebuildTransitCitiesFromTracks(shipment *models.Shipment) ([]models.TransitCityResponse, error) {
	bindings := h.resolveShipmentBindings(shipment)
	if len(bindings) == 0 {
		if err := h.replaceTransitCityCache(shipment.ID, nil); err != nil {
			return nil, err
		}
		return []models.TransitCityResponse{}, nil
	}

	sort.Slice(bindings, func(i, j int) bool {
		return bindings[i].BoundAt.Before(bindings[j].BoundAt)
	})

	if err := h.ensureShipmentStopsFresh(shipment); err != nil {
		log.Printf("[DeviceStop] ensureShipmentStopsFresh failed when rebuilding transit cities shipment=%s: %v", shipment.ID, err)
	}

	allTracks := make([]models.DeviceTrack, 0, 512)
	analysisStart := getTransitAnalysisStart(shipment)

	for _, binding := range bindings {
		segmentStart, segmentEnd, ok := h.getBindingTrackWindow(shipment, binding)
		if !ok {
			continue
		}
		if segmentStart.Before(analysisStart) {
			segmentStart = analysisStart
		}
		if segmentStart.After(segmentEnd) {
			continue
		}

		var segmentTracks []models.DeviceTrack
		if err := h.db.
			Where("device_id = ? AND locate_time >= ? AND locate_time <= ?", binding.DeviceID, segmentStart, segmentEnd).
			Order("locate_time ASC").
			Find(&segmentTracks).Error; err != nil {
			return nil, fmt.Errorf("查询轨迹失败(device=%s): %w", binding.DeviceID, err)
		}

		allTracks = append(allTracks, segmentTracks...)
	}

	if len(allTracks) == 0 {
		if err := h.replaceTransitCityCache(shipment.ID, nil); err != nil {
			return nil, err
		}
		return []models.TransitCityResponse{}, nil
	}

	sort.Slice(allTracks, func(i, j int) bool {
		return allTracks[i].LocateTime.Before(allTracks[j].LocateTime)
	})

	sampledTracks := selectTransitTrackSamples(allTracks, maxTransitReverseGeocodeSamples)
	if len(sampledTracks) == 0 {
		if err := h.replaceTransitCityCache(shipment.ID, nil); err != nil {
			return nil, err
		}
		return []models.TransitCityResponse{}, nil
	}

	geoService := services.NewGeocodingService()
	geoCache := make(map[string]transitGeoLookup, len(sampledTracks))
	visited := make(map[string]struct{}, 16)
	records := make([]models.TransitCityRecord, 0, 16)
	nominatimBudget := computeTransitNominatimBudget(len(sampledTracks))

	for _, track := range sampledTracks {
		if !isValidTransitCoordinate(track.Latitude, track.Longitude) {
			continue
		}

		country, province, city := h.resolveCountryProvinceCity(track.Latitude, track.Longitude, geoService, geoCache, &nominatimBudget)
		if country == "" {
			country = inferTransitCountryFromCoordinate(track.Latitude, track.Longitude)
		}
		country = normalizeTransitCountry(country)
		if country == "" {
			country = "国际区域"
		}
		province = normalizeTransitProvince(province)
		city = normalizeTransitCity(city)
		if city == "" {
			city = fallbackTransitCityFromCoordinate(track.Latitude, track.Longitude)
		}
		if province == "" {
			province = inferTransitProvinceFromCity(country, city)
		}
		if country == "" || city == "" {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(country)) + "|" + strings.ToLower(strings.TrimSpace(province)) + "|" + strings.ToLower(strings.TrimSpace(city))
		if _, exists := visited[key]; exists {
			continue
		}
		visited[key] = struct{}{}

		records = append(records, models.TransitCityRecord{
			ShipmentID: shipment.ID,
			DeviceID:   track.DeviceID,
			Country:    country,
			Province:   province,
			City:       city,
			Latitude:   track.Latitude,
			Longitude:  track.Longitude,
			EnteredAt:  track.LocateTime,
			IsOversea:  !isChinaCountry(country),
		})
	}

	if len(records) < minTransitCityNodeCount {
		stopRecords, err := h.buildTransitCitiesFromStops(shipment.ID, visited)
		if err != nil {
			log.Printf("[DeviceStop] buildTransitCitiesFromStops failed shipment=%s: %v", shipment.ID, err)
		} else if len(stopRecords) > 0 {
			records = append(records, stopRecords...)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].EnteredAt.Before(records[j].EnteredAt)
	})

	if err := h.replaceTransitCityCache(shipment.ID, records); err != nil {
		return nil, err
	}

	return h.loadTransitCityCache(shipment.ID)
}

func getTransitAnalysisStart(shipment *models.Shipment) time.Time {
	start := shipment.CreatedAt
	if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() && shipment.DeviceBoundAt.After(start) {
		start = *shipment.DeviceBoundAt
	}
	if shipment.LeftOriginAt != nil && !shipment.LeftOriginAt.IsZero() && shipment.LeftOriginAt.After(start) {
		start = *shipment.LeftOriginAt
	}
	return start
}

func computeTransitNominatimBudget(sampleCount int) int {
	if sampleCount <= 0 {
		return 0
	}
	budget := sampleCount / 3
	if budget < 12 {
		budget = 12
	}
	if budget > 80 {
		budget = 80
	}
	return budget
}

func (h *DeviceStopHandler) buildTransitCitiesFromStops(shipmentID string, visited map[string]struct{}) ([]models.TransitCityRecord, error) {
	var stops []models.DeviceStopRecord
	if err := h.db.
		Where("shipment_id = ?", shipmentID).
		Order("start_time ASC").
		Find(&stops).Error; err != nil {
		return nil, err
	}

	geoService := services.NewGeocodingService()
	geoCache := make(map[string]transitGeoLookup, len(stops))
	nominatimBudget := computeTransitNominatimBudget(len(stops))

	records := make([]models.TransitCityRecord, 0, 12)
	for _, stop := range stops {
		if stop.Latitude == nil || stop.Longitude == nil {
			continue
		}
		if !isValidTransitCoordinate(*stop.Latitude, *stop.Longitude) {
			continue
		}

		country, province, city := parseCountryProvinceCityFromStopAddress(stop.Address, *stop.Latitude, *stop.Longitude)
		countryHint := country
		if countryHint == "" {
			countryHint = inferTransitCountryFromCoordinate(*stop.Latitude, *stop.Longitude)
		}
		cityNeedsPromotion := city == "" || isTransitDistrictLike(city)
		provinceNeedsFill := strings.TrimSpace(province) == ""
		if !cityNeedsPromotion && isChinaCountry(countryHint) && city != "" && !hasChineseCharacters(city) {
			cityNeedsPromotion = true
		}
		if country == "" || cityNeedsPromotion || provinceNeedsFill {
			resolvedCountry, resolvedProvince, resolvedCity := h.resolveCountryProvinceCity(*stop.Latitude, *stop.Longitude, geoService, geoCache, &nominatimBudget)
			if country == "" {
				country = resolvedCountry
			}
			if strings.TrimSpace(province) == "" {
				province = resolvedProvince
			}
			if city == "" {
				city = resolvedCity
			} else if cityNeedsPromotion && resolvedCity != "" && !isTransitDistrictLike(resolvedCity) {
				city = resolvedCity
			}
		}
		if country == "" {
			country = inferTransitCountryFromCoordinate(*stop.Latitude, *stop.Longitude)
		}
		country = normalizeTransitCountry(country)
		if country == "" {
			country = "国际区域"
		}
		province = normalizeTransitProvince(province)
		city = normalizeTransitCity(city)
		if city == "" {
			city = fallbackTransitCityFromCoordinate(*stop.Latitude, *stop.Longitude)
		}
		if province == "" {
			province = inferTransitProvinceFromCity(country, city)
		}
		if country == "" || city == "" {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(country)) + "|" + strings.ToLower(strings.TrimSpace(province)) + "|" + strings.ToLower(strings.TrimSpace(city))
		if _, exists := visited[key]; exists {
			continue
		}
		visited[key] = struct{}{}

		records = append(records, models.TransitCityRecord{
			ShipmentID: shipmentID,
			DeviceID:   stop.DeviceID,
			Country:    country,
			Province:   province,
			City:       city,
			Latitude:   *stop.Latitude,
			Longitude:  *stop.Longitude,
			EnteredAt:  stop.StartTime,
			IsOversea:  !isChinaCountry(country),
		})
	}

	return records, nil
}

func fallbackTransitCityFromCoordinate(lat, lng float64) string {
	const step = 0.5 // 约50km~55km网格，避免无城市名时节点过密
	latGrid := math.Round(lat/step) * step
	lngGrid := math.Round(lng/step) * step
	return fmt.Sprintf("坐标区域(%.1f,%.1f)", latGrid, lngGrid)
}

var (
	chineseAdminPattern          = regexp.MustCompile(`[\p{Han}]{2,12}?(自治州|地区|盟|市|县|区|旗)`)
	chineseCityPattern           = regexp.MustCompile(`[\p{Han}]{2,16}?(自治州|地区|盟|市)`)
	chineseProvincePattern       = regexp.MustCompile(`[\p{Han}]{2,12}?(省|自治区|特别行政区)`)
	chineseMunicipalityPattern   = regexp.MustCompile(`(北京市|天津市|上海市|重庆市)`)
	englishProvinceSuffixPattern = regexp.MustCompile(`(?i)\s+(state|province)$`)
	englishAdminKeywords         = []string{"city", "county", "district", "prefecture", "region", "state", "province"}
	englishNoiseTokens           = map[string]struct{}{
		"town": {}, "township": {}, "village": {}, "road": {}, "street": {}, "avenue": {},
		"highway": {}, "expressway": {}, "bridge": {}, "service": {}, "area": {}, "parking": {},
	}
	chineseNoiseSuffixes = []string{"小区", "园区", "社区", "校区", "景区", "开发区", "服务区", "园", "站", "高速"}
)

func parseCountryProvinceCityFromStopAddress(address string, lat, lng float64) (string, string, string) {
	text := strings.TrimSpace(address)
	if text == "" {
		return inferTransitCountryFromCoordinate(lat, lng), "", ""
	}

	parts := strings.Split(text, "/")
	zhPart := strings.TrimSpace(parts[0])
	enPart := ""
	if len(parts) > 1 {
		enPart = strings.TrimSpace(parts[1])
	}

	country := inferTransitCountryFromText(strings.Join([]string{zhPart, enPart}, " "))
	if country == "" {
		country = inferTransitCountryFromCoordinate(lat, lng)
	}

	province := extractTransitProvinceFromChinese(zhPart)
	if province == "" {
		province = extractTransitProvinceFromEnglish(enPart)
	}
	if province == "" {
		province = extractTransitProvinceFromEnglish(zhPart)
	}

	city := extractTransitCityFromChinese(zhPart)
	if candidate := extractTransitCityFromEnglish(enPart); candidate != "" {
		city = pickBetterTransitCityCandidate(city, candidate)
	}
	if candidate := extractTransitCityFromEnglish(zhPart); candidate != "" {
		city = pickBetterTransitCityCandidate(city, candidate)
	}

	return country, province, city
}

func extractTransitProvinceFromChinese(text string) string {
	value := strings.TrimSpace(text)
	if value == "" {
		return ""
	}
	if match := chineseMunicipalityPattern.FindString(value); match != "" {
		return strings.TrimSpace(match)
	}
	if match := chineseProvincePattern.FindString(value); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

func extractTransitProvinceFromEnglish(text string) string {
	value := strings.TrimSpace(text)
	if value == "" {
		return ""
	}
	segments := splitEnglishAddressSegments(value)
	for _, segment := range segments {
		if candidate := extractEnglishAdminByKeyword(segment, "province"); candidate != "" {
			return candidate
		}
		if candidate := extractEnglishAdminByKeyword(segment, "state"); candidate != "" {
			return candidate
		}
	}
	return ""
}

func pickBetterTransitCityCandidate(current, candidate string) string {
	current = strings.TrimSpace(current)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	currentDistrictLike := isTransitDistrictLike(current)
	candidateDistrictLike := isTransitDistrictLike(candidate)
	if currentDistrictLike && !candidateDistrictLike {
		return candidate
	}
	if !hasChineseCharacters(current) && hasChineseCharacters(candidate) {
		return candidate
	}
	return current
}

func inferTransitCountryFromText(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return ""
	}

	aliases := map[string]string{
		"china": "中国", "prc": "中国", "中国": "中国", "中华人民共和国": "中国",
		"hong kong": "中国", "hongkong": "中国", "香港": "中国",
		"macau": "中国", "macao": "中国", "澳门": "中国",
		"taiwan": "中国", "台湾": "中国",
		"russia": "俄罗斯", "俄罗斯": "俄罗斯",
		"mongolia": "蒙古", "蒙古": "蒙古",
		"kazakhstan": "哈萨克斯坦", "哈萨克斯坦": "哈萨克斯坦",
		"united states": "美国", "usa": "美国", "america": "美国", "美国": "美国",
		"canada": "加拿大", "加拿大": "加拿大",
		"germany": "德国", "德国": "德国",
		"france": "法国", "法国": "法国",
		"spain": "西班牙", "西班牙": "西班牙",
		"italy": "意大利", "意大利": "意大利",
		"netherlands": "荷兰", "荷兰": "荷兰",
		"belgium": "比利时", "比利时": "比利时",
		"japan": "日本", "日本": "日本",
		"korea": "韩国", "south korea": "韩国", "韩国": "韩国",
		"thailand": "泰国", "泰国": "泰国",
		"vietnam": "越南", "越南": "越南",
		"singapore": "新加坡", "新加坡": "新加坡",
		"malaysia": "马来西亚", "马来西亚": "马来西亚",
		"indonesia": "印度尼西亚", "印尼": "印度尼西亚", "印度尼西亚": "印度尼西亚",
	}

	for key, value := range aliases {
		if strings.Contains(lower, key) {
			return value
		}
	}
	return ""
}

func inferTransitCountryFromCoordinate(lat, lng float64) string {
	// 中国陆地区域粗略包围盒
	if lat >= 18.0 && lat <= 54.0 && lng >= 73.0 && lng <= 136.0 {
		return "中国"
	}
	return ""
}

func extractTransitCityFromChinese(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	matches := chineseAdminPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}

	cityLevel := make([]string, 0, len(matches))
	districtLevel := make([]string, 0, len(matches))

	for _, match := range matches {
		candidate := strings.TrimSpace(match)
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, "中华人民共和国") || candidate == "中国" {
			continue
		}

		skip := false
		for _, suffix := range chineseNoiseSuffixes {
			if strings.HasSuffix(candidate, suffix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		switch {
		case strings.HasSuffix(candidate, "市"),
			strings.HasSuffix(candidate, "自治州"),
			strings.HasSuffix(candidate, "地区"),
			strings.HasSuffix(candidate, "盟"):
			cityLevel = append(cityLevel, candidate)
		case strings.HasSuffix(candidate, "县"),
			strings.HasSuffix(candidate, "区"),
			strings.HasSuffix(candidate, "旗"):
			districtLevel = append(districtLevel, candidate)
		}
	}

	if len(cityLevel) > 0 {
		return cityLevel[0]
	}
	if len(districtLevel) > 0 {
		return districtLevel[0]
	}
	return ""
}

func extractTransitCityFromEnglish(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	segments := splitEnglishAddressSegments(text)
	for _, keyword := range englishAdminKeywords {
		for _, segment := range segments {
			if candidate := extractEnglishAdminByKeyword(segment, keyword); candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func splitEnglishAddressSegments(text string) []string {
	replacer := strings.NewReplacer("，", ",", ";", ",", "；", ",", "/", ",", "(", " ", ")", " ", "（", " ", "）", " ")
	parts := strings.Split(replacer.Replace(text), ",")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	if len(segments) == 0 {
		return []string{text}
	}
	return segments
}

func extractEnglishAdminByKeyword(segment string, keyword string) string {
	words := strings.FieldsFunc(segment, func(r rune) bool {
		return !(unicode.IsLetter(r) || r == '\'' || r == '-')
	})
	for i, word := range words {
		if !strings.EqualFold(word, keyword) || i == 0 || i != len(words)-1 {
			continue
		}

		start := i - 1
		for lookback := i - 2; lookback >= 0 && i-lookback <= 3; lookback-- {
			if _, noise := englishNoiseTokens[strings.ToLower(words[lookback])]; noise {
				break
			}
			start = lookback
		}

		candidate := strings.Join(words[start:i+1], " ")
		return strings.TrimSpace(candidate)
	}
	return ""
}

func normalizeTransitCity(city string) string {
	city = strings.TrimSpace(city)
	city = strings.Trim(city, ",;")
	switch strings.ToLower(city) {
	case "", "市辖区":
		return ""
	}
	if strings.HasPrefix(city, "坐标区域(") {
		return city
	}

	if hasChineseCharacters(city) {
		normalized := strings.Join(strings.Fields(city), "")
		if normalized == "" {
			return ""
		}
		if preferred := extractChineseTransitCity(normalized); preferred != "" {
			return preferred
		}
		if hasTransitCitySuffix(normalized) {
			return normalized
		}
		return normalized + "市"
	}

	cleaned := strings.NewReplacer("，", " ", ",", " ", ";", " ", "；", " ").Replace(city)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return ""
	}

	return toTransitTitleCase(cleaned)
}

func hasTransitCitySuffix(city string) bool {
	for _, suffix := range []string{"自治州", "地区", "盟", "市", "县", "区", "旗", "省", "特别行政区"} {
		if strings.HasSuffix(city, suffix) {
			return true
		}
	}
	return false
}

func extractChineseTransitCity(text string) string {
	matches := chineseCityPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}
	for i := len(matches) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(matches[i])
		if candidate == "" || candidate == "市辖区" {
			continue
		}
		return candidate
	}
	return ""
}

func isTransitDistrictLike(city string) bool {
	value := strings.TrimSpace(city)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "坐标区域(") {
		return false
	}
	if hasChineseCharacters(value) {
		return strings.HasSuffix(value, "区") || strings.HasSuffix(value, "县") || strings.HasSuffix(value, "旗")
	}
	lower := strings.ToLower(value)
	return strings.HasSuffix(lower, " district") || strings.HasSuffix(lower, " county")
}

func choosePreferredTransitCity(city, province, district string) string {
	city = strings.TrimSpace(city)
	province = strings.TrimSpace(province)
	district = strings.TrimSpace(district)

	preferred := city
	if preferred == "" {
		preferred = province
	}
	if preferred == "" {
		preferred = district
	}
	if isTransitDistrictLike(preferred) && province != "" && !isTransitDistrictLike(province) {
		preferred = province
	}
	return preferred
}

func hasChineseCharacters(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func toTransitTitleCase(text string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	for i, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func (h *DeviceStopHandler) resolveShipmentBindings(shipment *models.Shipment) []models.ShipmentDeviceBinding {
	bindings := make([]models.ShipmentDeviceBinding, 0, 2)
	if services.DeviceBinding != nil {
		bindings = services.DeviceBinding.GetBindingHistory(shipment.ID)
	}

	// 兼容旧数据：无绑定历史时根据运单当前/历史设备补一条
	if len(bindings) == 0 {
		if shipment.DeviceID != nil && *shipment.DeviceID != "" {
			boundAt := shipment.CreatedAt
			if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() {
				boundAt = *shipment.DeviceBoundAt
			}
			bindings = append(bindings, models.ShipmentDeviceBinding{
				ShipmentID: shipment.ID,
				DeviceID:   *shipment.DeviceID,
				BoundAt:    boundAt,
			})
		} else if shipment.UnboundDeviceID != nil && *shipment.UnboundDeviceID != "" {
			bindings = append(bindings, models.ShipmentDeviceBinding{
				ShipmentID: shipment.ID,
				DeviceID:   *shipment.UnboundDeviceID,
				BoundAt:    shipment.CreatedAt,
				UnboundAt:  shipment.DeviceUnboundAt,
			})
		}
	}

	return bindings
}

func (h *DeviceStopHandler) loadTransitCityCache(shipmentID string) ([]models.TransitCityResponse, error) {
	var records []models.TransitCityRecord
	if err := h.db.
		Where("shipment_id = ?", shipmentID).
		Order("entered_at ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	return h.toTransitCityResponses(records), nil
}

func (h *DeviceStopHandler) replaceTransitCityCache(shipmentID string, records []models.TransitCityRecord) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("shipment_id = ?", shipmentID).Delete(&models.TransitCityRecord{}).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		return tx.Create(&records).Error
	})
}

type normalizedTransitEntry struct {
	record   models.TransitCityRecord
	country  string
	province string
	city     string
}

func (h *DeviceStopHandler) toTransitCityResponses(records []models.TransitCityRecord) []models.TransitCityResponse {
	normalizedEntries := make([]normalizedTransitEntry, 0, len(records))
	geoService := services.NewGeocodingService()
	geoCache := make(map[string]transitGeoLookup, len(records))
	nominatimBudget := computeTransitNominatimBudget(len(records))

	for _, record := range records {
		rawCity := strings.TrimSpace(record.City)
		rawProvince := strings.TrimSpace(record.Province)
		country := normalizeTransitCountry(record.Country)
		province := normalizeTransitProvince(rawProvince)
		city := normalizeTransitCity(rawCity)
		needResolve := shouldPromoteTransitCity(country, rawCity, city) || province == ""

		if needResolve {
			resolvedCountry, resolvedProvince, resolvedCity := h.resolveCountryProvinceCity(record.Latitude, record.Longitude, geoService, geoCache, &nominatimBudget)
			if country == "" {
				country = normalizeTransitCountry(resolvedCountry)
			}
			if province == "" {
				province = normalizeTransitProvince(resolvedProvince)
			}
			normalizedResolvedCity := normalizeTransitCity(resolvedCity)
			if normalizedResolvedCity != "" {
				switch {
				case city == "":
					city = normalizedResolvedCity
				case isTransitDistrictLike(city) && !isTransitDistrictLike(normalizedResolvedCity):
					city = normalizedResolvedCity
				case isChinaCountry(country) && !hasChineseCharacters(city) && hasChineseCharacters(normalizedResolvedCity):
					city = normalizedResolvedCity
				case hasChineseCharacters(rawCity) && !hasTransitCitySuffix(rawCity):
					city = normalizedResolvedCity
				}
			}
		}

		if country == "" {
			country = "国际区域"
		}
		if city == "" {
			city = fallbackTransitCityFromCoordinate(record.Latitude, record.Longitude)
		}
		if province == "" {
			province = inferTransitProvinceFromCity(country, city)
		}

		normalizedEntries = append(normalizedEntries, normalizedTransitEntry{
			record:   record,
			country:  country,
			province: province,
			city:     city,
		})
	}

	preferredCityByGrid := make(map[string]string, len(normalizedEntries))
	for _, entry := range normalizedEntries {
		if !isUsableTransitCityLabel(entry.city) || isTransitDistrictLike(entry.city) {
			continue
		}
		gridKey := transitGridKey(entry.country, entry.record.Latitude, entry.record.Longitude)
		preferredCityByGrid[gridKey] = chooseBetterTransitGridCity(preferredCityByGrid[gridKey], entry.city)
	}

	resp := make([]models.TransitCityResponse, 0, len(normalizedEntries))
	visited := make(map[string]struct{}, len(normalizedEntries))
	for _, entry := range normalizedEntries {
		city := entry.city
		if isTransitDistrictLike(city) {
			gridKey := transitGridKey(entry.country, entry.record.Latitude, entry.record.Longitude)
			if preferred := strings.TrimSpace(preferredCityByGrid[gridKey]); preferred != "" {
				city = preferred
			}
		}

		key := strings.ToLower(strings.TrimSpace(entry.country)) + "|" + strings.ToLower(strings.TrimSpace(entry.province)) + "|" + strings.ToLower(strings.TrimSpace(city))
		if _, exists := visited[key]; exists {
			continue
		}
		visited[key] = struct{}{}

		resp = append(resp, models.TransitCityResponse{
			ID:        entry.record.ID,
			Country:   entry.country,
			Province:  entry.province,
			City:      city,
			Latitude:  entry.record.Latitude,
			Longitude: entry.record.Longitude,
			EnteredAt: entry.record.EnteredAt,
			IsOversea: entry.record.IsOversea,
		})
	}
	return resp
}

func shouldPromoteTransitCity(country, rawCity, normalizedCity string) bool {
	if strings.TrimSpace(normalizedCity) == "" {
		return true
	}
	rawCity = strings.TrimSpace(rawCity)
	if rawCity == "" {
		return true
	}
	if isCompositeTransitCity(rawCity) || isTransitDistrictLike(rawCity) {
		return true
	}
	if hasChineseCharacters(rawCity) && !hasTransitCitySuffix(rawCity) {
		return true
	}
	if isChinaCountry(country) && !hasChineseCharacters(normalizedCity) {
		return true
	}
	return false
}

func transitGridKey(country string, lat, lng float64) string {
	const step = 0.25
	latGrid := math.Round(lat/step) * step
	lngGrid := math.Round(lng/step) * step
	return fmt.Sprintf("%s|%.2f|%.2f", strings.ToLower(strings.TrimSpace(country)), latGrid, lngGrid)
}

func isUsableTransitCityLabel(city string) bool {
	value := strings.TrimSpace(city)
	if value == "" {
		return false
	}
	return !strings.HasPrefix(value, "坐标区域(")
}

func chooseBetterTransitGridCity(current, candidate string) string {
	current = strings.TrimSpace(current)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	if scoreTransitCityLabel(candidate) > scoreTransitCityLabel(current) {
		return candidate
	}
	return current
}

func scoreTransitCityLabel(city string) int {
	value := strings.TrimSpace(city)
	if value == "" {
		return -100
	}
	score := 0
	if hasChineseCharacters(value) {
		score += 4
	}
	if !isTransitDistrictLike(value) {
		score += 3
	}
	for _, suffix := range []string{"市", "自治州", "地区", "盟"} {
		if strings.HasSuffix(value, suffix) {
			score += 2
			break
		}
	}
	if strings.HasPrefix(value, "坐标区域(") {
		score -= 6
	}
	return score
}

func selectTransitTrackSamples(tracks []models.DeviceTrack, maxSamples int) []models.DeviceTrack {
	if len(tracks) == 0 {
		return nil
	}
	if maxSamples <= 1 {
		return []models.DeviceTrack{tracks[len(tracks)-1]}
	}
	if len(tracks) <= maxSamples {
		return tracks
	}

	// 先在全时间轴上基于时间/距离抽取候选点，避免采样偏向早期轨迹
	samples := make([]models.DeviceTrack, 0, len(tracks))
	last := tracks[0]
	samples = append(samples, last)

	for i := 1; i < len(tracks)-1; i++ {
		curr := tracks[i]
		distance := services.CalculateDistance(last.Latitude, last.Longitude, curr.Latitude, curr.Longitude)
		if curr.LocateTime.Sub(last.LocateTime) >= minTransitSampleInterval || distance >= minTransitSampleDistanceKM {
			samples = append(samples, curr)
			last = curr
		}
	}

	lastTrack := tracks[len(tracks)-1]
	if !samples[len(samples)-1].LocateTime.Equal(lastTrack.LocateTime) {
		samples = append(samples, lastTrack)
	}

	if len(samples) <= maxSamples {
		return samples
	}

	return downsampleTransitTracks(samples, maxSamples)
}

func downsampleTransitTracks(tracks []models.DeviceTrack, maxSamples int) []models.DeviceTrack {
	if len(tracks) == 0 {
		return nil
	}
	if len(tracks) <= maxSamples {
		return tracks
	}
	if maxSamples <= 1 {
		return []models.DeviceTrack{tracks[len(tracks)-1]}
	}

	lastIndex := len(tracks) - 1
	trimmed := make([]models.DeviceTrack, 0, maxSamples)
	trimmed = append(trimmed, tracks[0])

	if maxSamples > 2 {
		interval := float64(lastIndex) / float64(maxSamples-1)
		prev := 0
		for i := 1; i < maxSamples-1; i++ {
			idx := int(math.Round(float64(i) * interval))
			if idx <= prev {
				idx = prev + 1
			}
			if idx >= lastIndex {
				idx = lastIndex - 1
			}
			trimmed = append(trimmed, tracks[idx])
			prev = idx
		}
	}

	trimmed = append(trimmed, tracks[lastIndex])
	return trimmed
}

func (h *DeviceStopHandler) resolveCountryProvinceCity(lat, lng float64, geoService *services.GeocodingService, cache map[string]transitGeoLookup, nominatimBudget *int) (string, string, string) {
	cacheKey := fmt.Sprintf("%.3f,%.3f", lat, lng)
	if val, ok := cache[cacheKey]; ok {
		return val.Country, val.Province, val.City
	}

	var country, province, city string

	if services.TencentMap != nil {
		detail, err := services.TencentMap.ReverseGeocodeDetail(lat, lng)
		if err == nil && detail != nil {
			country = strings.TrimSpace(detail.Country)
			province = strings.TrimSpace(detail.Province)
			city = choosePreferredTransitCity(
				strings.TrimSpace(detail.City),
				province,
				strings.TrimSpace(detail.District),
			)
		}
	}

	if (country == "" || province == "" || city == "") && geoService != nil && nominatimBudget != nil && *nominatimBudget > 0 {
		*nominatimBudget--
		result, err := geoService.ReverseGeocode(lat, lng)
		if err == nil && result != nil {
			if country == "" {
				country = strings.TrimSpace(result.Country)
			}
			if province == "" {
				province = strings.TrimSpace(result.State)
			}
			if city == "" {
				city = strings.TrimSpace(result.City)
			}
		}
	}

	cache[cacheKey] = transitGeoLookup{Country: country, Province: province, City: city}
	return country, province, city
}

func isValidTransitCoordinate(lat, lng float64) bool {
	if math.Abs(lat) < 0.000001 && math.Abs(lng) < 0.000001 {
		return false
	}
	if lat < -90 || lat > 90 {
		return false
	}
	if lng < -180 || lng > 180 {
		return false
	}
	return true
}

func normalizeTransitCountry(country string) string {
	country = strings.TrimSpace(country)
	switch strings.ToLower(country) {
	case "中华人民共和国":
		return "中国"
	case "china", "people's republic of china", "prc":
		return "中国"
	default:
		return country
	}
}

func normalizeTransitProvince(province string) string {
	province = strings.TrimSpace(strings.Trim(province, ",;"))
	if province == "" {
		return ""
	}

	if hasChineseCharacters(province) {
		normalized := strings.Join(strings.Fields(province), "")
		if normalized == "" {
			return ""
		}
		return normalized
	}

	cleaned := strings.NewReplacer("，", " ", ",", " ", ";", " ", "；", " ").Replace(province)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return ""
	}
	cleaned = englishProvinceSuffixPattern.ReplaceAllString(cleaned, "")
	return toTransitTitleCase(cleaned)
}

func inferTransitProvinceFromCity(country, city string) string {
	if !isChinaCountry(country) {
		return ""
	}
	switch strings.TrimSpace(city) {
	case "北京市", "天津市", "上海市", "重庆市":
		return strings.TrimSpace(city)
	default:
		return ""
	}
}

func isChinaCountry(country string) bool {
	normalized := strings.ToLower(strings.TrimSpace(country))
	switch normalized {
	case "中国", "中华人民共和国", "china", "people's republic of china", "prc", "cn":
		return true
	default:
		return false
	}
}

// GetStopByID 获取停留记录详情
// GET /api/device-stops/record/:id
func (h *DeviceStopHandler) GetStopByID(c *gin.Context) {
	id := c.Param("id")

	record, err := h.deviceStopService.GetStopByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.NotFound(c, "停留记录不存在")
		} else {
			utils.InternalError(c, err.Error())
		}
		return
	}

	response := models.DeviceStopRecordResponse{
		ID:               record.ID,
		DeviceID:         record.DeviceID,
		DeviceExternalID: record.DeviceExternalID,
		ShipmentID:       record.ShipmentID,
		StartTime:        record.StartTime,
		EndTime:          record.EndTime,
		DurationSeconds:  record.DurationSeconds,
		DurationText:     record.DurationText,
		Latitude:         record.Latitude,
		Longitude:        record.Longitude,
		Address:          record.Address,
		Status:           record.Status,
		AlertSent:        record.AlertSent,
		CreatedAt:        record.CreatedAt,
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    response,
	})
}

// DeleteStop 删除停留记录
// DELETE /api/device-stops/record/:id
func (h *DeviceStopHandler) DeleteStop(c *gin.Context) {
	id := c.Param("id")

	err := h.deviceStopService.DeleteStop(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"message": "删除成功",
		},
	})
}

// BatchDeleteStops 批量删除停留记录
// DELETE /api/device-stops/batch
func (h *DeviceStopHandler) BatchDeleteStops(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: ids是必填项")
		return
	}

	err := h.deviceStopService.BatchDeleteStops(req.IDs)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"message": "批量删除成功",
			"count":   len(req.IDs),
		},
	})
}

// UpdateActiveStops 更新所有活跃停留记录的持续时间
// POST /api/device-stops/update-active
// 这是一个管理接口,可以定期调用以更新活跃停留记录的持续时间
func (h *DeviceStopHandler) UpdateActiveStops(c *gin.Context) {
	err := h.deviceStopService.UpdateActiveStopDurations()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"message": "更新成功",
			"time":    time.Now(),
		},
	})
}

// CheckAlerts 检查并发送停留超时预警
// POST /api/device-stops/check-alerts
// 这是一个管理接口,可以定期调用以检查停留超时并发送预警
func (h *DeviceStopHandler) CheckAlerts(c *gin.Context) {
	err := h.deviceStopService.CheckAndSendAlert()
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"message": "检查完成",
			"time":    time.Now(),
		},
	})
}

// AnalyzeDeviceStops 手动触发设备停留分析
// POST /api/device-stops/analyze
// 用于在轨迹同步后手动触发停留检测，或重新分析历史数据
func (h *DeviceStopHandler) AnalyzeDeviceStops(c *gin.Context) {
	var req struct {
		DeviceID   string `json:"device_id" binding:"required"`
		StartTime  string `json:"start_time"` // 可选，格式: 2006-01-02 15:04:05
		EndTime    string `json:"end_time"`   // 可选
		ShipmentID string `json:"shipment_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: device_id是必填项")
		return
	}

	// 获取设备外部ID
	var device struct {
		ExternalID string `json:"external_id"`
	}
	if err := h.db.Table("devices").Select("external_id").Where("id = ?", req.DeviceID).First(&device).Error; err != nil {
		utils.InternalError(c, "查询设备信息失败: "+err.Error())
		return
	}

	// 解析时间参数
	var startTime, endTime time.Time
	var err error
	if req.StartTime != "" {
		startTime, err = time.Parse("2006-01-02 15:04:05", req.StartTime)
		if err != nil {
			utils.BadRequest(c, "开始时间格式错误，应为: 2006-01-02 15:04:05")
			return
		}
	}
	if req.EndTime != "" {
		endTime, err = time.Parse("2006-01-02 15:04:05", req.EndTime)
		if err != nil {
			utils.BadRequest(c, "结束时间格式错误，应为: 2006-01-02 15:04:05")
			return
		}
	}

	// 调用停留分析服务
	err = h.deviceStopService.AnalyzeDeviceTracksAndCreateStops(
		req.DeviceID,
		device.ExternalID,
		req.ShipmentID,
		startTime,
		endTime,
	)
	if err != nil {
		utils.InternalError(c, "停留分析失败: "+err.Error())
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"message": "停留分析完成",
			"time":    time.Now(),
		},
	})
}
