package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// SyncScheduler 设备数据同步调度器
type SyncScheduler struct {
	db         *gorm.DB
	kuaihuoyun *KuaihuoyunService
	running    bool
	stopChan   chan struct{}
	mu         sync.Mutex
}

const (
	stopAnalyzeLookbackWindow = 6 * time.Hour
	stopAnalyzeBackstepWindow = 30 * time.Minute
)

// NewSyncScheduler 创建同步调度器
func NewSyncScheduler(db *gorm.DB) *SyncScheduler {
	return &SyncScheduler{
		db:         db,
		kuaihuoyun: Kuaihuoyun,
		stopChan:   make(chan struct{}),
	}
}

// Start 启动同步调度器
func (s *SyncScheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("[SyncScheduler] 启动设备数据同步调度器...")

	// 高频同步：每1分钟（活跃设备）
	go s.runInterval(1*time.Minute, "active", s.syncActiveDevices)

	// 中频同步：每5分钟（静止设备）
	go s.runInterval(5*time.Minute, "idle", s.syncIdleDevices)

	// 低频同步：每30分钟（离线设备）
	go s.runInterval(30*time.Minute, "offline", s.syncOfflineDevices)

	// 船司数据同步：每15分钟（海运运单）
	go s.runInterval(15*time.Minute, "carrier", s.syncCarrierTracks)

	// 全局预警检查：每1小时（检查ETD/ETA/免租期等）
	go s.runInterval(1*time.Hour, "alerts", s.checkAllShipmentAlerts)

	// 停留时长刷新：每1分钟（active 停留记录时长自动增长）
	go s.runInterval(1*time.Minute, "device-stops", s.refreshActiveDeviceStopDurations)

	// 数据清理：每天凌晨3点（清理超过365天的数据）
	go s.runDaily("03:00", s.cleanupOldData)

	// 服务期限预警检查：每天凌晨2点
	go s.runDaily("02:00", s.checkServiceExpirations)
}

// Stop 停止同步调度器
func (s *SyncScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopChan)
		s.running = false
		log.Println("[SyncScheduler] 同步调度器已停止")
	}
}

// runInterval 定时执行任务
func (s *SyncScheduler) runInterval(interval time.Duration, name string, task func() error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行一次
	if err := task(); err != nil {
		log.Printf("[SyncScheduler] %s 同步失败: %v", name, err)
	}

	for {
		select {
		case <-ticker.C:
			if err := task(); err != nil {
				log.Printf("[SyncScheduler] %s 同步失败: %v", name, err)
			}
		case <-s.stopChan:
			return
		}
	}
}

// runDaily 每天定时执行任务
func (s *SyncScheduler) runDaily(timeStr string, task func() error) {
	for {
		now := time.Now()
		targetTime, _ := time.Parse("15:04", timeStr)
		next := time.Date(now.Year(), now.Month(), now.Day(),
			targetTime.Hour(), targetTime.Minute(), 0, 0, now.Location())

		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}

		timer := time.NewTimer(next.Sub(now))
		select {
		case <-timer.C:
			if err := task(); err != nil {
				log.Printf("[SyncScheduler] 每日任务失败: %v", err)
			}
		case <-s.stopChan:
			timer.Stop()
			return
		}
	}
}

// syncActiveDevices 同步活跃设备（速度>0的在线设备）
func (s *SyncScheduler) syncActiveDevices() error {
	return s.syncDevicesByStatus("active")
}

// syncIdleDevices 同步静止设备（在线但速度=0）
func (s *SyncScheduler) syncIdleDevices() error {
	return s.syncDevicesByStatus("idle")
}

// syncOfflineDevices 同步离线设备
func (s *SyncScheduler) syncOfflineDevices() error {
	return s.syncDevicesByStatus("offline")
}

// syncDevicesByStatus 按状态同步设备
func (s *SyncScheduler) syncDevicesByStatus(activityStatus string) error {
	if !s.kuaihuoyun.IsConfigured() {
		return nil // API未配置，跳过
	}

	// 获取需要同步的设备ID列表
	var devices []models.Device
	query := s.db.Model(&models.Device{})

	switch activityStatus {
	case "active":
		// 优化：活跃设备范围扩大，包括：
		// 1. 当前速度>0的在线设备
		// 2. 最近30分钟内有速度>0记录的在线设备（临时停车不降级）
		// 3. 绑定了pending/in_transit状态运单的设备（确保货物追踪实时性）
		recentActiveThreshold := time.Now().Add(-30 * time.Minute)

		// 子查询：最近30分钟有速度>0轨迹的设备ID
		recentActiveSubquery := s.db.Model(&models.DeviceTrack{}).
			Select("DISTINCT device_id").
			Where("locate_time >= ? AND speed > 0", recentActiveThreshold)

		// 子查询：绑定活跃运单的设备ID
		shipmentBoundSubquery := s.db.Model(&models.Shipment{}).
			Select("device_id").
			Where("device_id IS NOT NULL AND device_id != '' AND status IN ('pending', 'in_transit') AND deleted_at IS NULL")

		query = query.Where(
			"status = ? AND (speed > 0 OR id IN (?) OR id IN (?))",
			"online", recentActiveSubquery, shipmentBoundSubquery,
		)
	case "idle":
		// 静止设备：在线但不在活跃列表中
		// 排除绑定活跃运单的设备（它们应该按活跃频率同步）
		recentActiveThreshold := time.Now().Add(-30 * time.Minute)

		recentActiveSubquery := s.db.Model(&models.DeviceTrack{}).
			Select("DISTINCT device_id").
			Where("locate_time >= ? AND speed > 0", recentActiveThreshold)

		shipmentBoundSubquery := s.db.Model(&models.Shipment{}).
			Select("device_id").
			Where("device_id IS NOT NULL AND device_id != '' AND status IN ('pending', 'in_transit') AND deleted_at IS NULL")

		query = query.Where(
			"status = ? AND (speed = 0 OR speed IS NULL) AND id NOT IN (?) AND id NOT IN (?)",
			"online", recentActiveSubquery, shipmentBoundSubquery,
		)
	case "offline":
		query = query.Where("status = ?", "offline")
	}

	if err := query.Find(&devices).Error; err != nil {
		return err
	}

	if len(devices) == 0 {
		return nil
	}

	// 提取设备ID
	var deviceIDs []string
	for _, d := range devices {
		if d.ExternalDeviceID != nil && *d.ExternalDeviceID != "" {
			deviceIDs = append(deviceIDs, *d.ExternalDeviceID)
		}
	}

	if len(deviceIDs) == 0 {
		return nil
	}

	// 批量获取设备信息
	infos, err := s.kuaihuoyun.GetDeviceInfoList(deviceIDs)
	if err != nil {
		return err
	}

	// 保存轨迹数据
	now := time.Now()
	for _, info := range infos {
		// 查找对应的设备
		var device models.Device
		for _, d := range devices {
			if d.ExternalDeviceID != nil && *d.ExternalDeviceID == info.Device {
				device = d
				break
			}
		}

		// 创建轨迹记录
		track := models.DeviceTrack{
			DeviceID:    device.ID,
			Latitude:    info.Latitude,
			Longitude:   info.Longitude,
			Speed:       info.Speed,
			Direction:   info.Direction,
			Temperature: info.Temperature,
			Humidity:    info.Humidity,
			LocateType:  info.LocateType,
			LocateTime:  time.Unix(info.LocateTime, 0),
			SyncedAt:    now,
		}

		// 避免重复插入（同一设备同一时间点）
		result := s.db.Where("device_id = ? AND locate_time = ?",
			track.DeviceID, track.LocateTime).FirstOrCreate(&track)
		if result.Error != nil {
			log.Printf("[SyncScheduler] 保存轨迹失败 %s: %v", device.ID, result.Error)
		} else if result.RowsAffected > 0 {
			s.refreshBoundShipmentStopsForDevice(&device, track.LocateTime)
		}

		// 更新设备状态
		s.updateDeviceFromInfo(&device, &info)
	}

	log.Printf("[SyncScheduler] %s 同步完成: %d 设备, %d 轨迹点",
		activityStatus, len(devices), len(infos))
	return nil
}

// refreshBoundShipmentStopsForDevice 当设备轨迹新增时，增量刷新该设备绑定活跃运单的停留记录
func (s *SyncScheduler) refreshBoundShipmentStopsForDevice(device *models.Device, analysisEnd time.Time) {
	if device == nil || device.ID == "" || device.ExternalDeviceID == nil || *device.ExternalDeviceID == "" {
		return
	}
	if analysisEnd.IsZero() {
		analysisEnd = time.Now()
	}

	var shipments []models.Shipment
	if err := s.db.
		Select("id, device_bound_at, left_origin_at, created_at").
		Where("device_id = ? AND status IN ('pending', 'in_transit') AND deleted_at IS NULL", device.ID).
		Find(&shipments).Error; err != nil {
		log.Printf("[SyncScheduler] 查询设备关联运单失败 device=%s: %v", device.ID, err)
		return
	}
	if len(shipments) == 0 {
		return
	}

	stopService := NewDeviceStopService(s.db)
	for _, shipment := range shipments {
		analysisStart := analysisEnd.Add(-stopAnalyzeLookbackWindow)

		var latestStop models.DeviceStopRecord
		err := s.db.
			Select("start_time").
			Where("shipment_id = ? AND device_id = ?", shipment.ID, device.ID).
			Order("start_time DESC").
			First(&latestStop).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("[SyncScheduler] 查询停留记录失败 shipment=%s: %v", shipment.ID, err)
			continue
		}
		if err == nil {
			candidateStart := latestStop.StartTime.Add(-stopAnalyzeBackstepWindow)
			// 收敛到最近一次停留附近，既避免边界漏判，也避免反复回扫过长历史窗口
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
		if analysisStart.After(analysisEnd) {
			continue
		}

		if err := stopService.AnalyzeDeviceTracksAndCreateStops(
			device.ID,
			*device.ExternalDeviceID,
			shipment.ID,
			analysisStart,
			analysisEnd,
		); err != nil {
			log.Printf("[SyncScheduler] 增量刷新停留记录失败 shipment=%s device=%s: %v", shipment.ID, device.ID, err)
		}
	}
}

// updateDeviceFromInfo 从API信息更新设备状态
func (s *SyncScheduler) updateDeviceFromInfo(device *models.Device, info *DeviceInfo) {
	lastLocateTime := time.Unix(info.LocateTime, 0)
	if lastLocateTime.IsZero() || info.LocateTime <= 0 {
		lastLocateTime = time.Now()
	}

	updates := map[string]interface{}{
		"latitude":  info.Latitude,
		"longitude": info.Longitude,
		"speed":     info.Speed,
		"direction": info.Direction,
		"battery":   info.PowerRate,
		// 使用设备真实定位时间，避免“看起来在线但轨迹长期不更新”的假象
		"last_update": lastLocateTime,
	}

	if info.Temperature != nil {
		updates["temperature"] = *info.Temperature
	}
	if info.Humidity != nil {
		updates["humidity"] = *info.Humidity
	}

	// 更新在线状态
	if info.Status == 1 {
		updates["status"] = "online"
	} else {
		updates["status"] = "offline"
	}

	s.db.Model(device).Updates(updates)

	// 触发地理围栏检测，自动更新运单状态
	// 注意：运单表device_id存储的是Device.ID，不是ExternalDeviceID
	if Geofence != nil {
		Geofence.CheckAndUpdateStatus(device.ID, info.Latitude, info.Longitude)
	}

	// 触发预警检测
	if AlertChecker != nil {
		var shipment models.Shipment
		// 查找该设备关联的活跃运单
		if err := s.db.Where("device_id = ? AND status NOT IN ('completed', 'cancelled')", device.ID).First(&shipment).Error; err == nil {
			// 将设备最新数据关联到运单对象以便检查
			shipment.Device = device
			AlertChecker.CheckAllAlerts(&shipment)
		}
	}
}

// SyncDeviceImmediate 立即同步指定设备的轨迹数据
// 用于设备绑定后立即获取轨迹，不等待定时器
func (s *SyncScheduler) SyncDeviceImmediate(deviceID string) error {
	if !s.kuaihuoyun.IsConfigured() {
		log.Printf("[SyncScheduler] 快货运API未配置，跳过设备 %s 即时同步", deviceID)
		return nil
	}

	// 获取设备信息
	var device models.Device
	if err := s.db.First(&device, "id = ?", deviceID).Error; err != nil {
		return err
	}

	if device.ExternalDeviceID == nil || *device.ExternalDeviceID == "" {
		log.Printf("[SyncScheduler] 设备 %s 无外部设备号，跳过同步", deviceID)
		return nil
	}

	// 从API获取设备信息
	info, err := s.kuaihuoyun.GetDeviceInfo(*device.ExternalDeviceID)
	if err != nil {
		log.Printf("[SyncScheduler] 设备 %s 即时同步失败: %v", deviceID, err)
		return err
	}

	// 保存轨迹数据
	now := time.Now()
	track := models.DeviceTrack{
		DeviceID:    device.ID,
		Latitude:    info.Latitude,
		Longitude:   info.Longitude,
		Speed:       info.Speed,
		Direction:   info.Direction,
		Temperature: info.Temperature,
		Humidity:    info.Humidity,
		LocateType:  info.LocateType,
		LocateTime:  time.Unix(info.LocateTime, 0),
		SyncedAt:    now,
	}

	// 避免重复插入
	result := s.db.Where("device_id = ? AND locate_time = ?", track.DeviceID, track.LocateTime).FirstOrCreate(&track)
	if result.Error != nil {
		log.Printf("[SyncScheduler] 设备 %s 即时同步保存轨迹失败: %v", deviceID, result.Error)
	} else if result.RowsAffected > 0 {
		s.refreshBoundShipmentStopsForDevice(&device, track.LocateTime)
	}

	// 更新设备状态
	s.updateDeviceFromInfo(&device, info)

	log.Printf("[SyncScheduler] 设备 %s 即时同步完成，位置: (%.6f, %.6f)", deviceID, info.Latitude, info.Longitude)
	return nil
}

// checkAllShipmentAlerts 定时检查所有活跃运单的非物理类预警（如ETA、免租期）
func (s *SyncScheduler) checkAllShipmentAlerts() error {
	var shipments []models.Shipment
	if err := s.db.Preload("Device").Where("status NOT IN ('completed', 'cancelled')").Find(&shipments).Error; err != nil {
		return err
	}

	if AlertChecker != nil {
		for _, shipment := range shipments {
			// 这里主要检查时间相关的预警
			AlertChecker.CheckAllAlerts(&shipment)
		}
	}
	return nil
}

func (s *SyncScheduler) refreshActiveDeviceStopDurations() error {
	deviceStopSvc := NewDeviceStopService(s.db)
	return deviceStopSvc.UpdateActiveStopDurations()
}

// syncCarrierTracks 同步船司追踪数据
func (s *SyncScheduler) syncCarrierTracks() error {
	if CarrierTracking == nil || !CarrierTracking.IsConfigured() {
		return nil // 船司追踪未配置，跳过
	}

	// 获取需要同步的海运运单 (有B/L号且在途)
	var shipments []models.Shipment
	err := s.db.Where("transport_type = 'sea' AND bill_of_lading != '' AND status = 'in_transit'").
		Find(&shipments).Error
	if err != nil {
		return err
	}

	if len(shipments) == 0 {
		return nil
	}

	log.Printf("[SyncScheduler] 开始同步 %d 个海运运单的船司数据", len(shipments))

	syncedCount := 0
	for i := range shipments {
		tracks, err := CarrierTracking.TrackShipment(&shipments[i])
		if err != nil {
			log.Printf("[SyncScheduler] 运单 %s 船司同步失败: %v", shipments[i].ID, err)
			continue
		}
		if len(tracks) > 0 {
			syncedCount++
		}
	}

	log.Printf("[SyncScheduler] 船司数据同步完成: %d/%d 运单有更新", syncedCount, len(shipments))
	return nil
}

// cleanupOldData 清理超过1095天的旧数据
func (s *SyncScheduler) cleanupOldData() error {
	cutoffDate := time.Now().AddDate(-3, 0, 0) // 1095天前（3年）

	result := s.db.Where("locate_time < ?", cutoffDate).Delete(&models.DeviceTrack{})
	if result.Error != nil {
		return result.Error
	}

	log.Printf("[SyncScheduler] 已清理 %d 条超过1095天的轨迹数据", result.RowsAffected)
	return nil
}

// checkServiceExpirations 检查组织和设备的到期情况
func (s *SyncScheduler) checkServiceExpirations() error {
	log.Println("[SyncScheduler] 开始检查组织和设备服务到期情况...")

	now := time.Now()
	windows := []int{30, 7, 1}

	// 1. 检查组织到期 (ORG_EXPIRE)
	var orgs []models.Organization
	if err := s.db.Find(&orgs).Error; err == nil {
		for _, org := range orgs {
			if org.ServiceEnd == nil || org.ServiceEnd.IsZero() {
				continue
			}

			daysLeft := int(org.ServiceEnd.Sub(now).Hours() / 24)

			for _, w := range windows {
				if daysLeft == w {
					s.createExpirationAlert(nil, "ORG_EXPIRE", "warning",
						"组织服务即将到期",
						fmt.Sprintf("您的组织 [%s] 服务还有 %d 天到期（%s）", org.Name, daysLeft, org.ServiceEnd.Format("2006-01-02")),
						"system")
					break
				}
			}

			if daysLeft < 0 {
				s.createExpirationAlert(nil, "ORG_EXPIRE", "critical",
					"组织服务已过期",
					fmt.Sprintf("您的组织 [%s] 服务已过期 %d 天，请联系管理员续费", org.Name, -daysLeft),
					"system")
			}
		}
	}

	// 2. 检查设备到期 (DEVICE_EXPIRE)
	var devices []models.Device
	if err := s.db.Find(&devices).Error; err == nil {
		for _, device := range devices {
			if device.ServiceEndAt == nil || device.ServiceEndAt.IsZero() {
				continue
			}

			daysLeft := int(device.ServiceEndAt.Sub(now).Hours() / 24)

			for _, w := range windows {
				if daysLeft == w {
					s.createExpirationAlert(&device.ID, "DEVICE_EXPIRE", "warning",
						"设备服务即将到期",
						fmt.Sprintf("设备 [%s] 服务还有 %d 天到期（%s）", device.Name, daysLeft, device.ServiceEndAt.Format("2006-01-02")),
						"physical")

					s.notifyDeviceExpiration(device, daysLeft)
					break
				}
			}

			if daysLeft < 0 {
				s.createExpirationAlert(&device.ID, "DEVICE_EXPIRE", "critical",
					"设备服务已过期",
					fmt.Sprintf("设备 [%s] 服务已过期 %d 天，系统将停止服务", device.Name, -daysLeft),
					"physical")

				s.notifyDeviceExpiration(device, daysLeft)
			}
		}
	}

	log.Println("[SyncScheduler] 组织和设备到期检查完成")
	return nil
}

func (s *SyncScheduler) notifyDeviceExpiration(device models.Device, daysLeft int) {
	if device.SubAccountID != nil && *device.SubAccountID != "" {
		log.Printf("[SyncScheduler] [通知优先推送分机构 %s] 设备 %s 剩余 %d 天到期", *device.SubAccountID, device.ID, daysLeft)
	} else if device.OrgID != nil && *device.OrgID != "" {
		log.Printf("[SyncScheduler] [回退并通知主组织管理员 %s] 设备 %s 剩余 %d 天到期", *device.OrgID, device.ID, daysLeft)
	}
}

func (s *SyncScheduler) createExpirationAlert(deviceID *string, alertType, severity, title, message, category string) {
	var count int64
	query := s.db.Model(&models.Alert{}).Where("type = ? AND title = ? AND created_at > ?", alertType, title, time.Now().Add(-24*time.Hour))
	if deviceID != nil {
		query = query.Where("device_id = ?", *deviceID)
	}
	query.Count(&count)
	if count > 0 {
		return
	}

	alert := models.Alert{
		DeviceID:  deviceID,
		Type:      alertType,
		Severity:  severity,
		Title:     title,
		Message:   &message,
		Status:    "pending",
		Category:  category,
		CreatedAt: time.Now(),
	}
	s.db.Create(&alert)
}

// GetSyncStats 获取同步统计信息
func (s *SyncScheduler) GetSyncStats() map[string]interface{} {
	var totalTracks int64
	var todayTracks int64
	var deviceCount int64

	today := time.Now().Truncate(24 * time.Hour)

	s.db.Model(&models.DeviceTrack{}).Count(&totalTracks)
	s.db.Model(&models.DeviceTrack{}).Where("synced_at >= ?", today).Count(&todayTracks)
	s.db.Model(&models.Device{}).Count(&deviceCount)

	return map[string]interface{}{
		"total_tracks": totalTracks,
		"today_tracks": todayTracks,
		"device_count": deviceCount,
		"running":      s.running,
	}
}

// SyncDeviceTrack 手动同步单个设备的轨迹
func (s *SyncScheduler) SyncDeviceTrack(deviceID string, startTime, endTime time.Time) ([]models.DeviceTrack, error) {
	if !s.kuaihuoyun.IsConfigured() {
		return nil, nil
	}

	// 获取设备信息
	var device models.Device
	if err := s.db.Where("id = ? OR external_device_id = ?", deviceID, deviceID).First(&device).Error; err != nil {
		return nil, err
	}

	// 增量同步策略：
	// 1. 先查该时间窗口内本地最新点
	// 2. 仅拉取“本地最新点之后”的缺失区间，避免重复全量拉取
	var latestLocal models.DeviceTrack
	hasLatestLocal := false
	if err := s.db.Where("device_id = ? AND locate_time BETWEEN ? AND ?",
		device.ID, startTime, endTime).
		Order("locate_time DESC").
		First(&latestLocal).Error; err == nil {
		hasLatestLocal = true
	}

	// 检查ExternalDeviceID
	if device.ExternalDeviceID == nil || *device.ExternalDeviceID == "" {
		// 没有外部设备号时，直接返回本地结果
		var localTracks []models.DeviceTrack
		s.db.Where("device_id = ? AND locate_time BETWEEN ? AND ?",
			device.ID, startTime, endTime).
			Order("locate_time ASC").
			Find(&localTracks)
		return localTracks, nil
	}

	syncStart := startTime
	if hasLatestLocal {
		next := latestLocal.LocateTime.Add(1 * time.Second)
		if next.After(syncStart) {
			syncStart = next
		}
	}

	// 仅当存在待补齐时间窗口时才调用外部API
	if !syncStart.After(endTime) {
		startStr := syncStart.Format("2006-01-02 15:04:05")
		endStr := endTime.Format("2006-01-02 15:04:05")

		apiTracks, err := s.kuaihuoyun.GetTrack(*device.ExternalDeviceID, startStr, endStr)
		if err != nil {
			return nil, err
		}

		// 保存到本地（幂等去重：device_id + locate_time）
		var hasNewTracks bool
		var latestTrackTime time.Time
		for _, t := range apiTracks {
			track := models.DeviceTrack{
				DeviceID:    device.ID,
				Latitude:    t.Latitude,
				Longitude:   t.Longitude,
				Speed:       t.Speed,
				Direction:   t.Direction,
				RunStatus:   t.RunStatus,
				Temperature: t.Temperature,
				Humidity:    t.Humidity,
				LocateType:  t.LocateType,
				LocateTime:  time.Unix(t.LocateTime, 0),
				SyncedAt:    time.Now(),
			}

			result := s.db.Where("device_id = ? AND locate_time = ?", track.DeviceID, track.LocateTime).
				FirstOrCreate(&track)
			if result.Error != nil {
				// 兼容历史环境下的唯一冲突报错
				if !strings.Contains(strings.ToLower(result.Error.Error()), "duplicate") &&
					!strings.Contains(strings.ToLower(result.Error.Error()), "unique") {
					log.Printf("[SyncScheduler] 保存轨迹失败: %v", result.Error)
				}
			} else if result.RowsAffected > 0 {
				hasNewTracks = true
				if track.LocateTime.After(latestTrackTime) {
					latestTrackTime = track.LocateTime
				}
			}
		}

		if hasNewTracks {
			s.refreshBoundShipmentStopsForDevice(&device, latestTrackTime)
		}
	}

	// 返回最终本地窗口数据
	var localTracks []models.DeviceTrack
	s.db.Where("device_id = ? AND locate_time BETWEEN ? AND ?",
		device.ID, startTime, endTime).
		Order("locate_time ASC").
		Find(&localTracks)

	return localTracks, nil
}

// 全局同步调度器实例
var Scheduler *SyncScheduler

// InitScheduler 初始化同步调度器
func InitScheduler(db *gorm.DB) {
	Scheduler = NewSyncScheduler(db)
}
