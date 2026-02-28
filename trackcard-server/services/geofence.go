package services

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// ====================
// 常量定义（修复：魔法数字提取）
// ====================

const (
	// DefaultMinConsecutivePoints 触发状态变更所需的连续轨迹点数量
	DefaultMinConsecutivePoints = 3

	// DepartureFallbackPoints 离开发货地兜底判定所需的连续围栏外点数
	DepartureFallbackPoints = 2

	// DepartureFarOutsideMultiplier 离开发货地兜底判定的距离倍数（相对围栏半径）
	DepartureFarOutsideMultiplier = 1.5

	// DepartureFallbackMinSpan 离开发货地兜底判定所需最小时间跨度
	DepartureFallbackMinSpan = 5 * time.Minute

	// DefaultMinFenceRadius 最小围栏半径（米），防止配置错误
	DefaultMinFenceRadius = 100

	// DefaultGlobalFenceRadius 默认全局围栏半径（米）
	DefaultGlobalFenceRadius = 1000

	// TrackHistoryLimit 历史轨迹检查上限
	TrackHistoryLimit = 100

	// ConfigCacheTTL 配置缓存过期时间
	ConfigCacheTTL = 5 * time.Minute

	// TrackCacheTTL 轨迹缓存过期时间
	TrackCacheTTL = 30 * time.Second

	// MaxProgressRetreat 允许的最大进度倒退（防止GPS漂移）
	MaxProgressRetreat = 5

	// EarthRadiusMeters 地球半径（米）
	EarthRadiusMeters = 6371000
)

// ====================
// 缓存结构（修复：配置和轨迹缓存）
// ====================

type configCacheEntry struct {
	value    int
	cachedAt time.Time
}

type trackCacheEntry struct {
	tracks   []models.DeviceTrack
	cachedAt time.Time
}

var (
	configCache sync.Map
	trackCache  sync.Map
)

// ====================
// GeofenceService 地理围栏服务（优化版）
// ====================

// GeofenceService 地理围栏服务
type GeofenceService struct {
	db        *gorm.DB
	debugMode bool
}

// NewGeofenceService 创建地理围栏服务
func NewGeofenceService(db *gorm.DB) *GeofenceService {
	return &GeofenceService{db: db, debugMode: false}
}

// SetDebugMode 设置调试模式
func (s *GeofenceService) SetDebugMode(enabled bool) {
	s.debugMode = enabled
}

// logDebug 条件日志输出
func (s *GeofenceService) logDebug(format string, args ...interface{}) {
	if s.debugMode {
		log.Printf("[Geofence-Debug] "+format, args...)
	}
}

// logInfo 普通日志
func (s *GeofenceService) logInfo(format string, args ...interface{}) {
	log.Printf("[Geofence] "+format, args...)
}

// logError 错误日志
func (s *GeofenceService) logError(format string, args ...interface{}) {
	log.Printf("[Geofence-Error] "+format, args...)
}

// ====================
// 配置读取（修复：带缓存）
// ====================

// getGlobalGeofenceRadius 从系统配置表读取全局围栏半径（带缓存）
func (s *GeofenceService) getGlobalGeofenceRadius(key string, defaultValue int) int {
	// 检查缓存
	if cached, ok := configCache.Load(key); ok {
		if entry, ok := cached.(*configCacheEntry); ok {
			if time.Since(entry.cachedAt) < ConfigCacheTTL {
				return entry.value
			}
		}
	}

	var config models.SystemConfig
	if err := s.db.Where("key = ?", key).First(&config).Error; err != nil {
		return defaultValue
	}

	val, err := strconv.Atoi(config.Value)
	if err != nil || val <= 0 {
		val = defaultValue
	}

	// 写入缓存
	configCache.Store(key, &configCacheEntry{value: val, cachedAt: time.Now()})
	return val
}

// getConfigBool 从系统配置获取布尔值（带缓存）
func (s *GeofenceService) getConfigBool(key string, defaultValue bool) bool {
	cacheKey := "bool:" + key
	if cached, ok := configCache.Load(cacheKey); ok {
		if entry, ok := cached.(*configCacheEntry); ok {
			if time.Since(entry.cachedAt) < ConfigCacheTTL {
				return entry.value == 1
			}
		}
	}

	var config models.SystemConfig
	if err := s.db.Where("key = ?", key).First(&config).Error; err != nil {
		return defaultValue
	}

	result := config.Value == "true" || config.Value == "1"
	val := 0
	if result {
		val = 1
	}
	configCache.Store(cacheKey, &configCacheEntry{value: val, cachedAt: time.Now()})
	return result
}

// getConfigInt 从系统配置获取整数值（带缓存）
func (s *GeofenceService) getConfigInt(key string, defaultValue int) int {
	return s.getGlobalGeofenceRadius(key, defaultValue)
}

// ====================
// 距离计算
// ====================

// HaversineDistance 计算两点间的距离(米)
// 使用Haversine公式计算地球表面两点之间的距离
func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusMeters * c
}

// IsInsideGeofence 判断点是否在围栏内
func IsInsideGeofence(lat, lng, fenceLat, fenceLng float64, radius int) bool {
	distance := HaversineDistance(lat, lng, fenceLat, fenceLng)
	return distance <= float64(radius)
}

// isValidCoordinate 验证坐标有效性（修复：边界条件）
func isValidCoordinate(lat, lng float64) bool {
	// 检查经纬度范围
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return false
	}
	// 排除无效的 (0,0) 坐标（通常表示GPS未定位）
	if lat == 0 && lng == 0 {
		return false
	}
	// 检查NaN
	if math.IsNaN(lat) || math.IsNaN(lng) {
		return false
	}
	return true
}

// clamp 限制值在指定范围内
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ====================
// 轨迹查询（修复：带缓存）
// ====================

// getRecentTracks 获取设备最近的轨迹点（带缓存）
func (s *GeofenceService) getRecentTracks(deviceID string, limit int) []models.DeviceTrack {
	cacheKey := fmt.Sprintf("tracks:%s:%d", deviceID, limit)

	// 检查缓存
	if cached, ok := trackCache.Load(cacheKey); ok {
		if entry, ok := cached.(*trackCacheEntry); ok {
			if time.Since(entry.cachedAt) < TrackCacheTTL {
				return entry.tracks
			}
		}
	}

	var tracks []models.DeviceTrack
	s.db.Where("device_id = ?", deviceID).
		Order("locate_time DESC").
		Limit(limit).
		Find(&tracks)

	// 写入缓存
	trackCache.Store(cacheKey, &trackCacheEntry{tracks: tracks, cachedAt: time.Now()})
	return tracks
}

// invalidateTrackCache 使轨迹缓存失效
func (s *GeofenceService) invalidateTrackCache(deviceID string) {
	trackCache.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok && strings.HasPrefix(k, "tracks:"+deviceID) {
			trackCache.Delete(key)
		}
		return true
	})
}

// ====================
// 核心围栏检测（修复：事务+乐观锁+函数拆分）
// ====================

// CheckAndUpdateStatus 检测设备位置并自动更新运单状态
func (s *GeofenceService) CheckAndUpdateStatus(deviceID string, lat, lng float64) {
	// 修复：验证坐标有效性
	if !isValidCoordinate(lat, lng) {
		s.logDebug("无效坐标被忽略: device=%s, lat=%.6f, lng=%.6f", deviceID, lat, lng)
		return
	}

	// 查找设备关联的所有活跃运单
	var shipments []models.Shipment
	err := s.db.Where("device_id = ? AND status IN ('pending', 'in_transit')", deviceID).Find(&shipments).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.logError("查询设备运单失败: device=%s, error=%v", deviceID, err)
		}
		return
	}

	if len(shipments) == 0 {
		return
	}

	// 循环处理每个运单
	for _, shipment := range shipments {
		s.processShipmentGeofence(deviceID, lat, lng, shipment)
	}
}

// processShipmentGeofence 处理单个运单的围栏检测
func (s *GeofenceService) processShipmentGeofence(deviceID string, lat, lng float64, shipment models.Shipment) {
	// 1. 验证运单配置
	if !s.validateShipmentForGeofence(&shipment) {
		return
	}

	// 2. 计算围栏半径
	originRadius, destRadius := s.calculateFenceRadii(&shipment)

	// 3. 根据状态执行相应检测
	switch shipment.Status {
	case "pending":
		s.checkDeparture(&shipment, deviceID, originRadius)
	case "in_transit":
		// 优先检测到达目的地
		s.checkArrivalOrUpdateProgress(&shipment, deviceID, lat, lng, destRadius)
		// 同时检测是否进入了中间环节的围栏 (新增逻辑)
		s.checkIntermediateStages(&shipment, deviceID, lat, lng)
	}
}

// ====================
// 中间环节检测（新增：方案 A）
// ====================

// checkIntermediateStages 检测是否进入中间环节围栏
// 遍历所有待开始(pending)的中间环节，检查设备是否进入其围栏
func (s *GeofenceService) checkIntermediateStages(shipment *models.Shipment, deviceID string, lat, lng float64) {
	// 获取运单所有环节
	stageService := GetShipmentStageService()
	if stageService == nil {
		return
	}

	stages, err := stageService.GetStagesByShipmentID(shipment.ID)
	if err != nil {
		s.logError("获取运单环节失败: shipment=%s, error=%v", shipment.ID, err)
		return
	}

	// 遍历环节
	for _, stage := range stages {
		// 只检查 pending 状态的环节（in_progress 也可以检查，视业务需不需要自动完成）
		// 这里假设进入围栏即触发完成前置 + 激活当前 + (可选)完成当前
		// 根据设计，进入港口围栏通常意味着“到达港口”，即前一环节(如Main Line)结束，本环节(Dest Port)开始
		// 或者本环节(Origin Port)开始。
		//简化逻辑：只要是 pending 的环节，且有坐标，就检测
		if stage.Status != "pending" {
			continue
		}

		// 获取环节坐标
		stageLat, stageLng, radius := s.getStageCoordinates(&stage)
		if stageLat == 0 && stageLng == 0 {
			continue
		}

		// 检测是否在围栏内
		// 这里可以使用简单的当前点检测，也可以用连续点。为了响应快，先用单点 + 历史点辅助
		insideCount := s.countConsecutivePointsInside(deviceID, stageLat, stageLng, radius)

		if insideCount >= DefaultMinConsecutivePoints {
			s.logInfo("运单 %s 设备进入环节 %s (%s) 围栏", shipment.ID, stage.StageCode, models.GetStageName(stage.StageCode))

			// 触发环节流转
			if stage.StageOrder > 1 {
				// 自动流转：将当前环节设为进行中，并完成所有前置环节
				// 注意：CompleteStagesUpTo(target=CurrentStage) 会将 Target 设为 InProgress，Previous 设为 Completed
				if err := stageService.CompleteStagesUpTo(shipment.ID, stage.StageCode, models.TriggerGeofence, "自动触发：进入下一环节围栏"); err != nil {
					s.logError("自动流转环节失败: %v", err)
				} else {
					s.logInfo("成功触发环节流转: %s -> InProgress", stage.StageCode)
				}
			}

			// 如果是第一个 Pending 环节（例如跳过了中间很多），这个逻辑也适用（Complete X-1）。

			// 特殊情况：如果是 Dest Port，进入后 Main Line 完成，Dest Port In Progress。没问题。
			// 特殊情况：如果是 Origin Port (Pending)，进入后...
			// 前一个是 Pre-Transit。
			// Pre-Transit 完成 -> Origin Port In Progress。没问题。

			// 增加防抖：避免反复触发？
			// CompleteStagesUpTo 是幂等的，如果已经是 Completed 再次调用没问题。
			// 但为了性能，我们可以在上面 `if stage.Status != "pending"` 挡住。
			// 如果 Stage X 已经是 In Progress，就不需要再触发“完成 X-1”了（因为已经完成了）。

			// 完美。
		}
	}
}

// getStageCoordinates 解析环节的地理位置
func (s *GeofenceService) getStageCoordinates(stage *models.ShipmentStage) (lat, lng float64, radius int) {
	radius = 500 // 默认半径 500米

	// 1. 优先从 ExtraData 解析 (如果存储了 lat/lng json)
	// 这里假设 ExtraData 格式: {"lat": 12.34, "lng": 56.78}
	// 简单的字符串包含检查（为了性能，不做完整JSON解析，除非必要）
	// 或者直接解析。

	// 2. 从环节代码或元数据解析
	// 如果是 Origin Port 或 Dest Port，通常在 creates_stages 时会保存 PortCode

	portCode := stage.PortCode
	if portCode != "" {
		// 查询 Port 表
		// 注意：Port 模型在 models 包中，但我们需要在这个 Service 直接查库
		type PortLoc struct {
			Latitude  float64
			Longitude float64
		}
		var p PortLoc
		// 尝试查询 Port 表 (需确保表名正确，通常是 ports)
		if err := s.db.Table("ports").Select("latitude, longitude").Where("code = ?", portCode).Scan(&p).Error; err == nil {
			if isValidCoordinate(p.Latitude, p.Longitude) {
				return p.Latitude, p.Longitude, radius
			}
		}
		// 尝试查询 Airport 表
		var a PortLoc
		if err := s.db.Table("airports").Select("latitude, longitude").Where("iata_code = ?", portCode).Scan(&a).Error; err == nil {
			if isValidCoordinate(a.Latitude, a.Longitude) {
				return a.Latitude, a.Longitude, radius
			}
		}
	}

	return 0, 0, 0
}

// validateShipmentForGeofence 验证运单是否满足围栏检测条件
func (s *GeofenceService) validateShipmentForGeofence(shipment *models.Shipment) bool {
	// 检查自动状态开关
	if !shipment.AutoStatusEnabled {
		return false
	}

	// 检查坐标完整性
	if shipment.OriginLat == nil || shipment.OriginLng == nil ||
		shipment.DestLat == nil || shipment.DestLng == nil {
		s.reportMissingCoordinates(shipment)
		return false
	}

	// 验证坐标有效性
	if !isValidCoordinate(*shipment.OriginLat, *shipment.OriginLng) ||
		!isValidCoordinate(*shipment.DestLat, *shipment.DestLng) {
		s.logDebug("运单坐标无效: shipment=%s", shipment.ID)
		return false
	}

	return true
}

// calculateFenceRadii 计算围栏半径
func (s *GeofenceService) calculateFenceRadii(shipment *models.Shipment) (originRadius, destRadius int) {
	globalRadius := s.getGlobalGeofenceRadius("geofence_radius", DefaultGlobalFenceRadius)

	originRadius = shipment.OriginRadius
	if originRadius < DefaultMinFenceRadius {
		originRadius = globalRadius
		s.logDebug("运单 %s 发货地围栏使用全局配置 %d 米", shipment.ID, globalRadius)
	}

	destRadius = shipment.DestRadius
	if destRadius < DefaultMinFenceRadius {
		destRadius = globalRadius
		s.logDebug("运单 %s 目的地围栏使用全局配置 %d 米", shipment.ID, globalRadius)
	}

	return originRadius, destRadius
}

// reportMissingCoordinates 报告缺失坐标
func (s *GeofenceService) reportMissingCoordinates(shipment *models.Shipment) {
	if AlertChecker == nil {
		return
	}

	missingCoords := []string{}
	if shipment.OriginLat == nil || shipment.OriginLng == nil {
		missingCoords = append(missingCoords, "发货地坐标")
	}
	if shipment.DestLat == nil || shipment.DestLng == nil {
		missingCoords = append(missingCoords, "目的地坐标")
	}

	AlertChecker.createOrUpdateAlert(shipment, "missing_coordinates", "warning", "坐标信息缺失",
		fmt.Sprintf("运单缺少%s，无法进行围栏检测，请补充完整坐标信息", strings.Join(missingCoords, "、")), "operation")
}

// ====================
// 离开发货地检测（修复：事务+乐观锁）
// ====================

// checkDeparture 检测是否离开发货地
// 增强版：同时完成前程运输环节
func (s *GeofenceService) checkDeparture(shipment *models.Shipment, deviceID string, originRadius int) {
	// 检查最近N个轨迹点是否都在发货地围栏外
	outsideCount := s.countConsecutivePointsOutside(
		deviceID,
		*shipment.OriginLat, *shipment.OriginLng,
		originRadius,
	)

	if outsideCount < DefaultMinConsecutivePoints {
		if !s.hasStrongDepartureEvidence(deviceID, *shipment.OriginLat, *shipment.OriginLng, originRadius) {
			return
		}
		s.logInfo("运单 %s 使用兜底规则触发离开发货地（连续2个点显著在围栏外）", shipment.ID)
	}

	// 修复：使用事务和乐观锁更新状态，但日志在事务外部记录
	var statusUpdated bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 使用 WHERE 条件确保状态未被其他进程修改（乐观锁）
		result := tx.Model(&models.Shipment{}).
			Where("id = ? AND status = ?", shipment.ID, "pending").
			Updates(map[string]interface{}{
				"status":            "in_transit",
				"left_origin_at":    now,
				"status_updated_at": now,
				"current_milestone": "departed",
				"current_stage":     "origin_port", // 更新当前环节
				"atd":               now,
			})

		if result.RowsAffected == 0 {
			// 状态已被其他进程更新，跳过
			return nil
		}

		statusUpdated = true
		s.logInfo("运单 %s 已确认离开发货地(连续%d个轨迹点)", shipment.ID, outsideCount)

		return nil
	})

	if err != nil {
		s.logError("更新运单状态失败: shipment=%s, error=%v", shipment.ID, err)
		return
	}

	// 修复：日志记录移至事务外部，避免SQLite死锁
	if statusUpdated && ShipmentLog != nil {
		ShipmentLog.LogStatusChanged(shipment.ID, "pending", "in_transit", "system", "geofence")
		ShipmentLog.Log(shipment.ID, "geofence_trigger", "status", "pending", "in_transit",
			fmt.Sprintf("设备自动触发离开发货地（连续%d个点在围栏外）", outsideCount), "system", "geofence")
	}

	// 新增：自动完成前程运输环节，并激活起运港环节
	if statusUpdated {
		stageService := GetShipmentStageService()
		if stageService != nil {
			if err := stageService.CompleteStagesUpTo(shipment.ID, models.SSOriginPort, models.TriggerGeofence,
				"离开发货地自动触发"); err != nil {
				s.logError("自动补全环节失败: shipment=%s, error=%v", shipment.ID, err)
			}
		}
	}

	// 自动关闭相关预警
	s.closeRelatedAlerts(shipment.ID, []string{"etd_not_departed"})

	// 检查设备是否从未在围栏内出现
	s.checkFirstTrackOutsideFence(shipment, deviceID, originRadius)
}

// hasStrongDepartureEvidence 判断是否满足离开发货地的兜底触发条件
// 条件：最近2个点都在围栏外，且都超过围栏半径*1.5，并且两点时间间隔>=5分钟
func (s *GeofenceService) hasStrongDepartureEvidence(deviceID string, fenceLat, fenceLng float64, radius int) bool {
	tracks := s.getRecentTracks(deviceID, DepartureFallbackPoints+1)
	if len(tracks) < DepartureFallbackPoints {
		return false
	}

	latest := tracks[0]
	prev := tracks[1]

	if IsInsideGeofence(latest.Latitude, latest.Longitude, fenceLat, fenceLng, radius) ||
		IsInsideGeofence(prev.Latitude, prev.Longitude, fenceLat, fenceLng, radius) {
		return false
	}

	farThreshold := float64(radius) * DepartureFarOutsideMultiplier
	latestDistance := HaversineDistance(latest.Latitude, latest.Longitude, fenceLat, fenceLng)
	prevDistance := HaversineDistance(prev.Latitude, prev.Longitude, fenceLat, fenceLng)
	if latestDistance < farThreshold || prevDistance < farThreshold {
		return false
	}

	span := latest.LocateTime.Sub(prev.LocateTime)
	if span < 0 {
		span = -span
	}
	if span < DepartureFallbackMinSpan {
		return false
	}

	return true
}

// ====================
// 到达目的地检测（修复：事务+乐观锁）
// ====================

// checkArrivalOrUpdateProgress 检测到达或更新进度
func (s *GeofenceService) checkArrivalOrUpdateProgress(shipment *models.Shipment, deviceID string, lat, lng float64, destRadius int) {
	// 检查最近N个轨迹点是否都在目的地围栏内
	insideCount := s.countConsecutivePointsInside(
		deviceID,
		*shipment.DestLat, *shipment.DestLng,
		destRadius,
	)

	if insideCount >= DefaultMinConsecutivePoints {
		s.handleArrival(shipment, insideCount)
	} else {
		s.updateProgress(shipment, lat, lng)
	}
}

// handleArrival 处理到达目的地
// 增强版：自动完成所有环节
func (s *GeofenceService) handleArrival(shipment *models.Shipment, insideCount int) {
	// 修复：使用事务和乐观锁更新状态，但日志在事务外部记录
	var statusUpdated bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 使用 WHERE 条件确保状态未被其他进程修改（乐观锁）
		result := tx.Model(&models.Shipment{}).
			Where("id = ? AND status = ?", shipment.ID, "in_transit").
			Updates(map[string]interface{}{
				"status":            "delivered",
				"progress":          100,
				"arrived_dest_at":   now,
				"status_updated_at": now,
				"current_milestone": "arrived",
				"current_stage":     "delivered", // 更新当前环节
				"ata":               now,
			})

		if result.RowsAffected == 0 {
			// 状态已被其他进程更新，跳过
			return nil
		}

		statusUpdated = true
		s.logInfo("运单 %s 已确认到达目的地(连续%d个轨迹点)", shipment.ID, insideCount)

		return nil
	})

	if err != nil {
		s.logError("更新运单状态失败: shipment=%s, error=%v", shipment.ID, err)
		return
	}

	// 修复：日志记录移至事务外部，避免SQLite死锁
	if statusUpdated && ShipmentLog != nil {
		ShipmentLog.LogStatusChanged(shipment.ID, "in_transit", "delivered", "system", "geofence")
		ShipmentLog.Log(shipment.ID, "geofence_trigger", "status", "in_transit", "delivered",
			fmt.Sprintf("设备自动触发到达目的地（连续%d个点在围栏内）", insideCount), "system", "geofence")
	}

	// 新增：自动完成所有环节（包括可能跳过的中间环节）
	if statusUpdated {
		stageService := GetShipmentStageService()
		if stageService != nil {
			if err := stageService.CompleteAllStages(shipment.ID, models.TriggerGeofence,
				"到达目的地自动触发"); err != nil {
				s.logError("自动补全所有环节失败: shipment=%s, error=%v", shipment.ID, err)
			}
		}
	}

	// 处理到达后续操作
	s.handleArrivalActions(shipment, time.Now())

	// 自动关闭到达后不再需要的预警
	s.closeRelatedAlerts(shipment.ID, []string{"eta_delay", "vessel_delay", "carrier_stale"})
}

// ====================
// 轨迹点统计（修复：使用缓存）
// ====================

// countConsecutivePointsOutside 统计连续在围栏外的轨迹点数量
func (s *GeofenceService) countConsecutivePointsOutside(deviceID string, fenceLat, fenceLng float64, radius int) int {
	tracks := s.getRecentTracks(deviceID, DefaultMinConsecutivePoints)

	if len(tracks) < DefaultMinConsecutivePoints {
		return 0
	}

	count := 0
	for _, track := range tracks {
		if !IsInsideGeofence(track.Latitude, track.Longitude, fenceLat, fenceLng, radius) {
			count++
		} else {
			break // 要求连续
		}
	}
	return count
}

// countConsecutivePointsInside 统计连续在围栏内的轨迹点数量
func (s *GeofenceService) countConsecutivePointsInside(deviceID string, fenceLat, fenceLng float64, radius int) int {
	tracks := s.getRecentTracks(deviceID, DefaultMinConsecutivePoints)

	if len(tracks) < DefaultMinConsecutivePoints {
		return 0
	}

	count := 0
	for _, track := range tracks {
		if IsInsideGeofence(track.Latitude, track.Longitude, fenceLat, fenceLng, radius) {
			count++
		} else {
			break // 要求连续
		}
	}
	return count
}

// checkFirstTrackOutsideFence 检查设备是否从未在指定围栏内出现过
func (s *GeofenceService) checkFirstTrackOutsideFence(shipment *models.Shipment, deviceID string, radius int) {
	var tracks []models.DeviceTrack
	s.db.Where("device_id = ?", deviceID).
		Order("locate_time ASC").
		Limit(TrackHistoryLimit).
		Find(&tracks)

	for _, track := range tracks {
		if IsInsideGeofence(track.Latitude, track.Longitude, *shipment.OriginLat, *shipment.OriginLng, radius) {
			return // 找到了在围栏内的轨迹点
		}
	}

	// 从未在围栏内出现
	if AlertChecker != nil {
		AlertChecker.createOrUpdateAlert(shipment, "no_origin_track", "info", "设备首次上报即在围栏外",
			"设备从未在发货地围栏内出现轨迹点，可能存在设备开机延迟或数据缺失", "node")
	}
	s.logInfo("警告: 运单 %s 设备从未在发货地围栏内出现", shipment.ID)
}

// ====================
// 进度更新（修复：边界条件）
// ====================

// updateProgress 根据当前位置更新运输进度
func (s *GeofenceService) updateProgress(shipment *models.Shipment, currentLat, currentLng float64) {
	// 验证坐标
	if !isValidCoordinate(currentLat, currentLng) {
		return
	}
	if shipment.OriginLat == nil || shipment.DestLat == nil {
		return
	}

	// 计算总距离
	totalDistance := HaversineDistance(
		*shipment.OriginLat, *shipment.OriginLng,
		*shipment.DestLat, *shipment.DestLng,
	)

	// 修复：起止距离过近时跳过
	if totalDistance < 100 {
		return
	}

	// 计算已行驶距离
	traveledDistance := HaversineDistance(
		*shipment.OriginLat, *shipment.OriginLng,
		currentLat, currentLng,
	)

	// 计算剩余距离
	remainingDistance := HaversineDistance(
		currentLat, currentLng,
		*shipment.DestLat, *shipment.DestLng,
	)

	// 修复：使用两种方式计算进度，取较小值（避免反向问题）
	progressByTraveled := traveledDistance / totalDistance * 100
	progressByRemaining := (totalDistance - remainingDistance) / totalDistance * 100
	progress := int(math.Min(progressByTraveled, progressByRemaining))

	// 限制范围
	progress = clamp(progress, 0, 99)

	// 修复：防止进度倒退（GPS漂移导致）
	if progress < shipment.Progress-MaxProgressRetreat {
		s.logDebug("运单 %s 进度倒退被忽略: current=%d, new=%d", shipment.ID, shipment.Progress, progress)
		return
	}

	// 只有进度增加时才更新
	if progress > shipment.Progress {
		s.db.Model(shipment).Update("progress", progress)
		s.logDebug("运单 %s 进度更新: %d%%", shipment.ID, progress)
	}
}

// ====================
// 辅助功能
// ====================

// closeRelatedAlerts 关闭运单相关的指定类型预警
func (s *GeofenceService) closeRelatedAlerts(shipmentID string, alertTypes []string) {
	if len(alertTypes) == 0 {
		return
	}
	result := s.db.Model(&models.Alert{}).
		Where("shipment_id = ? AND type IN ? AND status = 'pending'", shipmentID, alertTypes).
		Updates(map[string]interface{}{
			"status":      "resolved",
			"resolved_at": time.Now(),
		})
	if result.RowsAffected > 0 {
		s.logInfo("运单 %s 自动关闭了 %d 个相关预警", shipmentID, result.RowsAffected)
	}
}

// handleArrivalActions 处理到达目的地后的后续操作
func (s *GeofenceService) handleArrivalActions(shipment *models.Shipment, arrivalTime time.Time) {
	// 1. 记录轨迹截止时间
	s.db.Model(shipment).Update("track_end_at", arrivalTime)
	s.logInfo("运单 %s 轨迹截止时间已设置", shipment.ID)

	// 2. 检查是否启用自动解绑
	if !s.getConfigBool("auto_unbind_on_arrival", true) {
		return
	}

	// 3. 检查运单级别是否启用自动解绑
	if !shipment.AutoUnbindEnabled {
		return
	}

	// 4. 检查解绑延迟时间
	delayHours := s.getConfigInt("unbind_delay_hours", 0)
	if delayHours > 0 {
		s.logInfo("运单 %s 配置了 %d 小时延迟解绑", shipment.ID, delayHours)
		return
	}

	// 5. 立即解绑
	s.unbindDevice(shipment)
}

// unbindDevice 解除运单与设备的绑定
func (s *GeofenceService) unbindDevice(shipment *models.Shipment) {
	if shipment.DeviceID == nil || *shipment.DeviceID == "" {
		return
	}

	deviceID := *shipment.DeviceID
	now := time.Now()

	// 1. 调用 DeviceBinding 服务解除绑定记录 (更新 shipment_device_bindings 表)
	if DeviceBinding != nil {
		if err := DeviceBinding.UnbindDevice(shipment.ID, "auto_arrival"); err != nil {
			s.logError("运单 %s 解绑设备记录失败: %v", shipment.ID, err)
		}
	}

	// 2. 更新运单表 (更新 shipments 表)
	s.db.Model(shipment).Updates(map[string]interface{}{
		"unbound_device_id": deviceID,
		"device_id":         nil,
		"device_unbound_at": now,
	})

	// 记录日志
	if ShipmentLog != nil {
		ShipmentLog.Log(shipment.ID, "device_unbound", "device_id",
			deviceID, "", "到达目的地后自动解除设备绑定", "system", "geofence")
	}

	// 清除轨迹缓存
	s.invalidateTrackCache(deviceID)

	s.logInfo("运单 %s 已自动解除设备 %s 绑定", shipment.ID, deviceID)
}

// ====================
// 补录日志
// ====================

// BackfillMissingLogs 为缺少围栏触发日志的运单补录日志
// 修复：分别检查 Departure 和 Arrival 日志，不再跳过已有部分日志的运单
func (s *GeofenceService) BackfillMissingLogs() (int, error) {
	// 查找所有 in_transit 或 delivered 状态的运单
	var shipments []models.Shipment
	err := s.db.Where("status IN ('in_transit', 'delivered')").Find(&shipments).Error
	if err != nil {
		return 0, err
	}

	if len(shipments) == 0 {
		s.logInfo("没有需要检查的运单")
		return 0, nil
	}

	s.logInfo("开始检查 %d 个运单的围栏触发日志", len(shipments))

	backfilledCount := 0
	for _, shipment := range shipments {
		count := s.backfillShipmentLogs(&shipment)
		backfilledCount += count
	}

	if backfilledCount > 0 {
		s.logInfo("补录完成，共补录 %d 条日志", backfilledCount)
	} else {
		s.logInfo("所有运单日志完整，无需补录")
	}
	return backfilledCount, nil
}

// backfillShipmentLogs 补录单个运单的日志
// 修复：分别检查 Departure 和 Arrival 日志是否存在
func (s *GeofenceService) backfillShipmentLogs(shipment *models.Shipment) int {
	count := 0

	// 检查是否有离开发货地日志 (pending -> in_transit)
	hasDepartureLog := s.hasGeofenceLog(shipment.ID, "pending", "in_transit")

	// 检查是否有到达目的地日志 (in_transit -> delivered)
	hasArrivalLog := s.hasGeofenceLog(shipment.ID, "in_transit", "delivered")

	// 补录离开发货地日志（in_transit 或 delivered 状态但缺少离开日志）
	if (shipment.Status == "in_transit" || shipment.Status == "delivered") && !hasDepartureLog {
		departTime := s.getShipmentDepartTime(shipment)

		// 修复：如果运单中没有发车时间，更新它
		if shipment.LeftOriginAt == nil {
			s.db.Model(shipment).Update("left_origin_at", departTime)
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
			CreatedAt:   departTime,
		}
		if err := s.db.Create(&logEntry).Error; err == nil {
			s.logInfo("运单 %s 补录离开发货地日志", shipment.ID)
			count++
		}
	}

	// 补录到达目的地日志（delivered 状态但缺少到达日志）
	if shipment.Status == "delivered" && !hasArrivalLog {
		arriveTime := s.getShipmentArriveTime(shipment)

		// 修复：如果运单中没有到达时间，更新它
		// 但要防止将到达时间设置为与出发时间相同（数据错误）
		if shipment.ArrivedDestAt == nil {
			if shipment.LeftOriginAt == nil || !arriveTime.Equal(*shipment.LeftOriginAt) {
				s.db.Model(shipment).Update("arrived_dest_at", arriveTime)
			} else {
				s.logInfo("运单 %s 跳过补录 arrived_dest_at：到达时间与出发时间相同", shipment.ID)
			}
		}

		// 修复：补录到达时，确保 TrackEndAt 被设置
		if shipment.TrackEndAt == nil {
			s.db.Model(shipment).Update("track_end_at", arriveTime)
			s.logInfo("运单 %s 补录轨迹截止时间", shipment.ID)
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
			CreatedAt:   arriveTime,
		}
		if err := s.db.Create(&logEntry).Error; err == nil {
			s.logInfo("运单 %s 补录到达目的地日志", shipment.ID)
			count++
		}

	}

	return count
}

// hasGeofenceLog 检查运单是否已有指定类型的围栏触发日志
func (s *GeofenceService) hasGeofenceLog(shipmentID, oldValue, newValue string) bool {
	var count int64
	s.db.Model(&models.ShipmentLog{}).
		Where("shipment_id = ? AND action = 'geofence_trigger' AND old_value = ? AND new_value = ?",
			shipmentID, oldValue, newValue).
		Count(&count)
	return count > 0
}

func (s *GeofenceService) getShipmentDepartTime(shipment *models.Shipment) time.Time {
	if shipment.LeftOriginAt != nil {
		return *shipment.LeftOriginAt
	}
	if shipment.ATD != nil {
		return *shipment.ATD
	}
	if shipment.StatusUpdatedAt != nil {
		return *shipment.StatusUpdatedAt
	}
	return time.Now()
}

func (s *GeofenceService) getShipmentArriveTime(shipment *models.Shipment) time.Time {
	if shipment.ArrivedDestAt != nil {
		return *shipment.ArrivedDestAt
	}
	if shipment.ATA != nil {
		return *shipment.ATA
	}
	if shipment.StatusUpdatedAt != nil {
		return *shipment.StatusUpdatedAt
	}
	return time.Now()
}

// ====================
// 调试功能
// ====================

// CheckWithDebug 带调试日志的围栏检测
func (s *GeofenceService) CheckWithDebug(deviceID string, lat, lng float64) string {
	result := fmt.Sprintf("设备 %s 位置 (%.6f, %.6f) 围栏检测:\n", deviceID, lat, lng)

	if !isValidCoordinate(lat, lng) {
		result += "  ❌ 坐标无效\n"
		return result
	}
	result += "  ✓ 坐标有效\n"

	var shipment models.Shipment
	err := s.db.Where("device_id = ? AND status IN ('pending', 'in_transit')", deviceID).First(&shipment).Error
	if err != nil {
		result += "  ❌ 设备没有关联活跃运单\n"
		return result
	}
	result += fmt.Sprintf("  ✓ 关联运单: %s (状态: %s)\n", shipment.ID, shipment.Status)

	if !shipment.AutoStatusEnabled {
		result += "  ❌ 自动状态未启用\n"
		return result
	}
	result += "  ✓ 自动状态已启用\n"

	if shipment.OriginLat == nil || shipment.DestLat == nil {
		result += "  ❌ 坐标未配置\n"
		return result
	}

	originRadius, destRadius := s.calculateFenceRadii(&shipment)

	distToOrigin := HaversineDistance(lat, lng, *shipment.OriginLat, *shipment.OriginLng)
	distToDest := HaversineDistance(lat, lng, *shipment.DestLat, *shipment.DestLng)

	result += fmt.Sprintf("  距发货地: %.0f米 (围栏: %d米) - %s\n",
		distToOrigin, originRadius,
		map[bool]string{true: "围栏内", false: "围栏外"}[distToOrigin <= float64(originRadius)])
	result += fmt.Sprintf("  距目的地: %.0f米 (围栏: %d米) - %s\n",
		distToDest, destRadius,
		map[bool]string{true: "围栏内", false: "围栏外"}[distToDest <= float64(destRadius)])

	if shipment.Status == "pending" {
		outsideCount := s.countConsecutivePointsOutside(deviceID, *shipment.OriginLat, *shipment.OriginLng, originRadius)
		result += fmt.Sprintf("  离开发货地条件: %d/%d 个点在围栏外 - %s\n",
			outsideCount, DefaultMinConsecutivePoints,
			map[bool]string{true: "满足", false: "不满足"}[outsideCount >= DefaultMinConsecutivePoints])
	}

	if shipment.Status == "in_transit" {
		insideCount := s.countConsecutivePointsInside(deviceID, *shipment.DestLat, *shipment.DestLng, destRadius)
		result += fmt.Sprintf("  到达目的地条件: %d/%d 个点在围栏内 - %s\n",
			insideCount, DefaultMinConsecutivePoints,
			map[bool]string{true: "满足", false: "不满足"}[insideCount >= DefaultMinConsecutivePoints])
	}

	return result
}

// ====================
// 兼容旧代码的函数（已废弃，保留向后兼容）
// ====================

// countRecentPointsOutsideFence 向后兼容
func (s *GeofenceService) countRecentPointsOutsideFence(deviceID string, limit int, fenceLat, fenceLng float64, radius int) int {
	return s.countConsecutivePointsOutside(deviceID, fenceLat, fenceLng, radius)
}

// countRecentPointsInsideFence 向后兼容
func (s *GeofenceService) countRecentPointsInsideFence(deviceID string, limit int, fenceLat, fenceLng float64, radius int) int {
	return s.countConsecutivePointsInside(deviceID, fenceLat, fenceLng, radius)
}

// hasNoTracksInsideFence 向后兼容
func (s *GeofenceService) hasNoTracksInsideFence(deviceID string, fenceLat, fenceLng float64, radius int) bool {
	var tracks []models.DeviceTrack
	s.db.Where("device_id = ?", deviceID).
		Order("locate_time ASC").
		Limit(TrackHistoryLimit).
		Find(&tracks)

	for _, track := range tracks {
		if IsInsideGeofence(track.Latitude, track.Longitude, fenceLat, fenceLng, radius) {
			return false
		}
	}
	return true
}

// ====================
// 全局实例
// ====================

// Geofence 全局地理围栏服务实例
var Geofence *GeofenceService

// GeofenceDebugEnabled 围栏调试日志开关（向后兼容）
var GeofenceDebugEnabled = false

// InitGeofence 初始化地理围栏服务
func InitGeofence(db *gorm.DB) {
	Geofence = NewGeofenceService(db)
	Geofence.SetDebugMode(GeofenceDebugEnabled)
	log.Println("[Geofence] 地理围栏服务初始化完成（优化版）")
}
