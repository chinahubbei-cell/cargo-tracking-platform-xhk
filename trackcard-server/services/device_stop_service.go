package services

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
)

type DeviceStopService struct {
	db *gorm.DB
}

var stopAddressCache sync.Map

func NewDeviceStopService(db *gorm.DB) *DeviceStopService {
	return &DeviceStopService{db: db}
}

func calculateActiveStopDuration(start time.Time, now time.Time) (int, string) {
	duration := int(now.Sub(start).Seconds())
	if duration < 0 {
		duration = 0
	}
	return duration, formatDuration(duration)
}

func refreshActiveRecordDuration(record *models.DeviceStopRecord, now time.Time) bool {
	if record.Status != "active" {
		return false
	}

	duration, durationText := calculateActiveStopDuration(record.StartTime, now)
	changed := record.DurationSeconds != duration || record.DurationText != durationText
	record.DurationSeconds = duration
	record.DurationText = durationText
	record.UpdatedAt = now
	return changed
}

func persistActiveDurationIfChanged(db *gorm.DB, record *models.DeviceStopRecord, now time.Time) error {
	if record == nil {
		return nil
	}
	changed := refreshActiveRecordDuration(record, now)
	if !changed {
		return nil
	}

	return db.Model(&models.DeviceStopRecord{}).
		Where("id = ?", record.ID).
		Updates(map[string]interface{}{
			"duration_seconds": record.DurationSeconds,
			"duration_text":    record.DurationText,
			"updated_at":       record.UpdatedAt,
		}).Error
}

// normalizeStaleActiveStops closes stale active records that are older than newer stop records.
// This prevents outdated active records from being highlighted as the latest stop point.
func (s *DeviceStopService) normalizeStaleActiveStops(shipmentID, deviceID string, now time.Time) error {
	shipmentID = strings.TrimSpace(shipmentID)
	if shipmentID == "" {
		return nil
	}

	activeQuery := s.db.Where("shipment_id = ? AND status = ?", shipmentID, "active")
	if strings.TrimSpace(deviceID) != "" {
		activeQuery = activeQuery.Where("device_id = ?", deviceID)
	}

	var activeRecords []models.DeviceStopRecord
	if err := activeQuery.Order("start_time ASC").Find(&activeRecords).Error; err != nil {
		return err
	}

	for _, active := range activeRecords {
		newerQuery := s.db.Model(&models.DeviceStopRecord{}).
			Where("shipment_id = ? AND start_time > ?", active.ShipmentID, active.StartTime)
		if strings.TrimSpace(active.DeviceID) != "" {
			newerQuery = newerQuery.Where("device_id = ?", active.DeviceID)
		}

		var newer models.DeviceStopRecord
		err := newerQuery.
			Order("COALESCE(end_time, start_time) DESC").
			Order("start_time DESC").
			First(&newer).Error
		if err == gorm.ErrRecordNotFound {
			continue
		}
		if err != nil {
			return err
		}

		endAt := newer.StartTime
		if newer.EndTime != nil && newer.EndTime.After(endAt) {
			endAt = *newer.EndTime
		}
		if endAt.Before(active.StartTime) {
			endAt = active.StartTime
		}

		durationSeconds := int(endAt.Sub(active.StartTime).Seconds())
		if durationSeconds < 0 {
			durationSeconds = 0
		}

		if err := s.db.Model(&models.DeviceStopRecord{}).
			Where("id = ?", active.ID).
			Updates(map[string]interface{}{
				"status":           "completed",
				"end_time":         endAt,
				"duration_seconds": durationSeconds,
				"duration_text":    formatDuration(durationSeconds),
				"updated_at":       now,
			}).Error; err != nil {
			return err
		}
	}

	return nil
}

// formatDuration 格式化持续时间为文本
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	} else if seconds < 3600 {
		minutes := seconds / 60
		return fmt.Sprintf("%d分钟", minutes)
	} else if seconds < 86400 {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		if minutes > 0 {
			return fmt.Sprintf("%d小时%d分钟", hours, minutes)
		}
		return fmt.Sprintf("%d小时", hours)
	} else {
		days := seconds / 86400
		hours := (seconds % 86400) / 3600
		if hours > 0 {
			return fmt.Sprintf("%d天%d小时", days, hours)
		}
		return fmt.Sprintf("%d天", days)
	}
}

// StartDeviceStop 开始设备停留记录
// 当设备进入 runStatus=1(静止)状态时调用
func (s *DeviceStopService) StartDeviceStop(deviceExternalID, deviceID, shipmentID string, latitude, longitude float64, address string) (*models.DeviceStopRecord, error) {
	resolvedAddress := strings.TrimSpace(address)
	if resolvedAddress == "" || IsCoordinateAddress(resolvedAddress) {
		resolvedAddress = ResolveNodeAddress(latitude, longitude)
	}
	resolvedAddress = EnsureBilingualNodeAddress(resolvedAddress, latitude, longitude)
	if resolvedAddress == "" {
		resolvedAddress = formatCoordinateFallbackAddress(latitude, longitude)
	}

	// 检查是否已有活跃的停留记录
	var existingRecord models.DeviceStopRecord
	err := s.db.Where("device_external_id = ? AND status = ?", deviceExternalID, "active").
		Order("created_at DESC").
		First(&existingRecord).Error

	if err == nil {
		// 已存在活跃记录,更新位置信息
		existingRecord.Latitude = &latitude
		existingRecord.Longitude = &longitude
		existingRecord.Address = resolvedAddress
		existingRecord.UpdatedAt = time.Now()
		err = s.db.Save(&existingRecord).Error
		return &existingRecord, err
	}

	// 创建新的停留记录
	record := &models.DeviceStopRecord{
		DeviceExternalID: deviceExternalID,
		DeviceID:         deviceID,
		ShipmentID:       shipmentID,
		StartTime:        time.Now(),
		Latitude:         &latitude,
		Longitude:        &longitude,
		Address:          resolvedAddress,
		Status:           "active",
		DurationSeconds:  0,
		DurationText:     "0秒",
	}

	err = s.db.Create(record).Error
	return record, err
}

// EndDeviceStop 结束设备停留记录
// 当设备从 runStatus=1(静止)变为 runStatus=2(运动)时调用
func (s *DeviceStopService) EndDeviceStop(deviceExternalID string) error {
	var record models.DeviceStopRecord
	err := s.db.Where("device_external_id = ? AND status = ?", deviceExternalID, "active").
		Order("created_at DESC").
		First(&record).Error

	if err != nil {
		// 没有活跃的停留记录
		return nil
	}

	// 更新结束时间和时长
	now := time.Now()
	record.EndTime = &now
	duration, durationText := calculateActiveStopDuration(record.StartTime, now)
	record.DurationSeconds = duration
	record.DurationText = durationText
	record.Status = "completed"
	record.UpdatedAt = now

	return s.db.Save(&record).Error
}

// UpdateActiveStopDurations 更新所有活跃停留记录的持续时间
// 应该定期调用(例如每分钟)
func (s *DeviceStopService) UpdateActiveStopDurations() error {
	var records []models.DeviceStopRecord
	err := s.db.Where("status = ?", "active").Find(&records).Error
	if err != nil {
		return err
	}

	now := time.Now()
	for _, record := range records {
		if err := persistActiveDurationIfChanged(s.db, &record, now); err != nil {
			return err
		}
	}

	return nil
}

// GetStopRecords 获取设备停留记录列表
func (s *DeviceStopService) GetStopRecords(req models.DeviceStopRecordListRequest) ([]models.DeviceStopRecord, int64, error) {
	now := time.Now()
	if err := s.normalizeStaleActiveStops(req.ShipmentID, req.DeviceID, now); err != nil {
		return nil, 0, err
	}

	query := s.db.Model(&models.DeviceStopRecord{})

	if req.DeviceID != "" {
		query = query.Where("device_id = ?", req.DeviceID)
	}
	if req.DeviceExternalID != "" {
		query = query.Where("device_external_id = ?", req.DeviceExternalID)
	}
	if req.ShipmentID != "" {
		query = query.Where("shipment_id = ?", req.ShipmentID)
	}
	if req.Status != "" && req.Status != "all" {
		query = query.Where("status = ?", req.Status)
	}
	if req.StartTime != "" {
		if startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime); err == nil {
			query = query.Where("start_time >= ?", startTime)
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime); err == nil {
			query = query.Where("start_time <= ?", endTime)
		}
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 分页
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var records []models.DeviceStopRecord
	err := query.Order("start_time DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return records, total, err
	}

	for i := range records {
		if err := persistActiveDurationIfChanged(s.db, &records[i], now); err != nil {
			return records, total, err
		}
	}

	return records, total, err
}

// GetDeviceStopStats 获取设备停留统计信息
func (s *DeviceStopService) GetDeviceStopStats(deviceID, deviceExternalID string) (*models.DeviceStopStats, error) {
	query := s.db.Model(&models.DeviceStopRecord{})
	if deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if deviceExternalID != "" {
		query = query.Where("device_external_id = ?", deviceExternalID)
	}

	// 统计总数和总时长
	type Result struct {
		TotalStops    int64
		TotalDuration int64
	}
	var result Result
	err := query.Select("COUNT(*) as total_stops, COALESCE(SUM(duration_seconds), 0) as total_duration").
		Where("status = ?", "completed").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	// 计算平均时长
	averageDuration := 0
	if result.TotalStops > 0 {
		averageDuration = int(result.TotalDuration / result.TotalStops)
	}

	// 获取当前活跃的停留记录
	var currentStop *models.DeviceStopRecordResponse
	var activeRecord models.DeviceStopRecord
	err = s.db.Where("device_external_id = ? AND status = ?", deviceExternalID, "active").
		Order("created_at DESC").
		First(&activeRecord).Error
	if err == nil {
		_ = persistActiveDurationIfChanged(s.db, &activeRecord, time.Now())
		// 更新当前停留时长（实时计算）
		duration, durationText := calculateActiveStopDuration(activeRecord.StartTime, time.Now())
		currentStop = &models.DeviceStopRecordResponse{
			ID:               activeRecord.ID,
			DeviceID:         activeRecord.DeviceID,
			DeviceExternalID: activeRecord.DeviceExternalID,
			ShipmentID:       activeRecord.ShipmentID,
			StartTime:        activeRecord.StartTime,
			EndTime:          activeRecord.EndTime,
			DurationSeconds:  duration,
			DurationText:     durationText,
			Latitude:         activeRecord.Latitude,
			Longitude:        activeRecord.Longitude,
			Address:          activeRecord.Address,
			Status:           activeRecord.Status,
			AlertSent:        activeRecord.AlertSent,
			CreatedAt:        activeRecord.CreatedAt,
		}
	}

	stats := &models.DeviceStopStats{
		DeviceID:         deviceID,
		DeviceExternalID: deviceExternalID,
		TotalStops:       int(result.TotalStops),
		TotalDuration:    int(result.TotalDuration),
		AverageDuration:  averageDuration,
		CurrentStop:      currentStop,
	}

	return stats, nil
}

// CheckAndSendAlert 检查并发送停留超时预警
// 应该定期调用(例如每小时)
func (s *DeviceStopService) CheckAndSendAlert() error {
	// 查找所有活跃的停留记录
	var records []models.DeviceStopRecord
	err := s.db.Where("status = ? AND alert_sent = ?", "active", false).Find(&records).Error
	if err != nil {
		return err
	}

	now := time.Now()
	for _, record := range records {
		duration := int(now.Sub(record.StartTime).Seconds())
		threshold := record.AlertThresholdHours * 3600

		if duration >= threshold {
			// TODO: 这里应该调用消息服务发送预警
			// 例如发送邮件、短信或推送通知
			// services.AlertService.SendStopAlert(record)

			// 标记为已发送
			record.AlertSent = true
			s.db.Save(&record)
		}
	}

	return nil
}

// GetCurrentStop 获取设备当前停留记录
func (s *DeviceStopService) GetCurrentStop(deviceExternalID string) (*models.DeviceStopRecord, error) {
	var record models.DeviceStopRecord
	err := s.db.Where("device_external_id = ? AND status = ?", deviceExternalID, "active").
		Order("created_at DESC").
		First(&record).Error

	if err != nil {
		return nil, err
	}

	// 更新当前停留时长（实时计算）并回写数据库，避免调度异常导致库值长期不变
	_ = persistActiveDurationIfChanged(s.db, &record, time.Now())

	return &record, nil
}

// GetStopByID 根据ID获取停留记录详情
func (s *DeviceStopService) GetStopByID(id string) (*models.DeviceStopRecord, error) {
	var record models.DeviceStopRecord
	err := s.db.Where("id = ?", id).First(&record).Error
	if err != nil {
		return &record, err
	}
	// 详情查询时也同步刷新并回写 active 时长
	_ = persistActiveDurationIfChanged(s.db, &record, time.Now())
	return &record, err
}

// DeleteStop 删除停留记录
func (s *DeviceStopService) DeleteStop(id string) error {
	return s.db.Delete(&models.DeviceStopRecord{}, "id = ?", id).Error
}

// BatchDeleteStops 批量删除停留记录
func (s *DeviceStopService) BatchDeleteStops(ids []string) error {
	return s.db.Delete(&models.DeviceStopRecord{}, ids).Error
}

// GetStopsByTimeRange 获取指定时间范围内的停留记录
func (s *DeviceStopService) GetStopsByTimeRange(deviceExternalID string, startTime, endTime time.Time) ([]models.DeviceStopRecord, error) {
	var records []models.DeviceStopRecord
	query := s.db.Where("device_external_id = ?", deviceExternalID)

	if !startTime.IsZero() {
		query = query.Where("start_time >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("start_time <= ?", endTime)
	}

	err := query.Order("start_time DESC").Find(&records).Error
	return records, err
}

// CalculateDistance 计算两个经纬度之间的距离(公里)
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // 地球半径,单位公里

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

// AnalyzeDeviceTracksAndCreateStops 分析设备轨迹并自动生成/更新停留记录
// 这是停留检测闭环的核心方法，应该在轨迹同步后自动调用
func (s *DeviceStopService) AnalyzeDeviceTracksAndCreateStops(deviceID, deviceExternalID, shipmentID string, startTime, endTime time.Time) error {
	// 停留检测参数
	const (
		minStopDuration          = 5 * time.Minute // 最短停留时长
		locationChangeThreshold  = 0.1             // 位置变化阈值(公里)，用于判断是否为同一次停留
		stationarySpeedThreshold = 2.0             // 低速抖动阈值(<=2 视为可判定停留)
	)

	// 查询设备轨迹数据
	var tracks []models.DeviceTrack
	query := s.db.Where("device_id = ?", deviceID)

	if !startTime.IsZero() {
		query = query.Where("locate_time >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("locate_time <= ?", endTime)
	}

	err := query.Order("locate_time ASC").Find(&tracks).Error
	if err != nil {
		return fmt.Errorf("查询设备轨迹失败: %v", err)
	}

	if len(tracks) == 0 {
		return nil // 没有轨迹数据，无需分析
	}

	// 分析停留段
	type StopSegment struct {
		StartTime    time.Time
		EndTime      *time.Time
		Latitude     float64
		Longitude    float64
		TrackCount   int
		HasZeroSpeed bool
	}

	var stopSegments []StopSegment
	var currentStop *StopSegment

	for i, track := range tracks {
		// 优先使用设备自身上报的真实运行状态(RunStatus: 2=静止)。如果历史数据中无此字段(=0)，则降级使用速度判定
		trackIsStationary := track.RunStatus == 2 || (track.RunStatus == 0 && track.Speed <= stationarySpeedThreshold)
		trackIsStrictStop := track.RunStatus == 2 || (track.RunStatus == 0 && track.Speed == 0)

		// 同样，如果是历史兼容降级，双重校验漂移
		if !trackIsStationary && track.RunStatus == 0 && i > 0 {
			prevTrack := tracks[i-1]
			dist := CalculateDistance(prevTrack.Latitude, prevTrack.Longitude, track.Latitude, track.Longitude)
			timeDiffHours := track.LocateTime.Sub(prevTrack.LocateTime).Hours()
			if timeDiffHours > 0 {
				actualSpeed := dist / timeDiffHours
				if actualSpeed <= stationarySpeedThreshold {
					trackIsStationary = true
					if actualSpeed == 0 {
						trackIsStrictStop = true
					}
				}
			} else if dist == 0 {
				trackIsStationary = true
				trackIsStrictStop = true
			}
		}

		if currentStop == nil {
			if !trackIsStationary {
				continue
			}
			currentStop = &StopSegment{
				StartTime:    track.LocateTime,
				Latitude:     track.Latitude,
				Longitude:    track.Longitude,
				TrackCount:   1,
				HasZeroSpeed: trackIsStrictStop,
			}
			continue
		}

		distance := CalculateDistance(
			currentStop.Latitude, currentStop.Longitude,
			track.Latitude, track.Longitude,
		)

		if trackIsStationary && distance <= locationChangeThreshold {
			// 低速且位置变化小，视为同一次停留（兼容设备低速抖动）
			currentStop.TrackCount++
			currentStop.Latitude = track.Latitude
			currentStop.Longitude = track.Longitude
			if trackIsStrictStop {
				currentStop.HasZeroSpeed = true
			}
			continue
		}

		// 出现明确移动或位置漂移，结束当前停留段
		currentStop.EndTime = &track.LocateTime
		stopSegments = append(stopSegments, *currentStop)
		currentStop = nil

		// 若当前点仍为低速，启动新的候选停留段
		if trackIsStationary {
			currentStop = &StopSegment{
				StartTime:    track.LocateTime,
				Latitude:     track.Latitude,
				Longitude:    track.Longitude,
				TrackCount:   1,
				HasZeroSpeed: trackIsStrictStop,
			}
		}
	}

	// 处理最后一个未结束的停留段
	if currentStop != nil {
		stopSegments = append(stopSegments, *currentStop)
	}

	// 过滤掉停留时长不足的段，并生成/更新停留记录
	now := time.Now()
	for _, segment := range stopSegments {
		segmentEnd := segment.EndTime
		segmentStatus := "completed"
		segmentDurationEnd := now
		if segmentEnd != nil {
			segmentDurationEnd = *segmentEnd
		} else {
			segmentStatus = "active"
		}

		duration := segmentDurationEnd.Sub(segment.StartTime)
		// 纯低速(无 speed=0)且仅单点的片段不入库，避免把慢速行驶误识别成停留
		if !segment.HasZeroSpeed && segment.TrackCount < 2 {
			continue
		}
		if duration < minStopDuration {
			continue // 停留时长不足，跳过
		}

		// 生成地址
		address := generateAddressFromCoordinates(segment.Latitude, segment.Longitude)

		// 检查是否已存在可复用的停留记录
		// active 段优先复用“当前活跃记录”，避免每次增量回扫都生成新的 active 记录导致时长被重置
		var existingRecord models.DeviceStopRecord
		var err error
		if segmentStatus == "active" {
			err = s.db.Where(
				"shipment_id = ? AND device_id = ? AND status = ?",
				shipmentID, deviceID, "active",
			).Order("start_time ASC").First(&existingRecord).Error
		} else {
			err = s.db.Where(
				"shipment_id = ? AND device_id = ? AND start_time = ?",
				shipmentID, deviceID, segment.StartTime,
			).First(&existingRecord).Error
		}

		if err == nil {
			// 更新现有记录
			// active 记录沿用历史最早开始时间，避免增量窗口导致停留时长被反复归零
			if segmentStatus == "active" {
				if segment.StartTime.Before(existingRecord.StartTime) {
					existingRecord.StartTime = segment.StartTime
				}
				existingRecord.EndTime = nil
				duration = now.Sub(existingRecord.StartTime)
			} else {
				actualDuration := segmentEnd.Sub(existingRecord.StartTime)
				if actualDuration < minStopDuration {
					// 该记录曾经是 active 被展示，但最终确认生命周期结束且总时长不足 15 分钟，判定为无效停留，予以删除
					s.db.Unscoped().Delete(&existingRecord)
					continue
				}
				existingRecord.EndTime = segmentEnd
				duration = actualDuration
			}
			existingRecord.Latitude = &segment.Latitude
			existingRecord.Longitude = &segment.Longitude
			existingRecord.Address = address
			existingRecord.Status = segmentStatus
			existingRecord.DurationSeconds = int(duration.Seconds())
			existingRecord.DurationText = formatDuration(existingRecord.DurationSeconds)
			existingRecord.UpdatedAt = now
			if err := s.db.Save(&existingRecord).Error; err != nil {
				return fmt.Errorf("更新停留记录失败(shipment=%s,start=%s): %w", shipmentID, segment.StartTime.Format(time.RFC3339), err)
			}
		} else if err == gorm.ErrRecordNotFound {
			durationSeconds := int(duration.Seconds())
			// 创建新记录
			record := &models.DeviceStopRecord{
				DeviceID:         deviceID,
				DeviceExternalID: deviceExternalID,
				ShipmentID:       shipmentID,
				StartTime:        segment.StartTime,
				EndTime:          segmentEnd,
				Latitude:         &segment.Latitude,
				Longitude:        &segment.Longitude,
				Address:          address,
				Status:           segmentStatus,
				DurationSeconds:  durationSeconds,
				DurationText:     formatDuration(durationSeconds),
			}
			if err := s.db.Create(record).Error; err != nil {
				return fmt.Errorf("创建停留记录失败(shipment=%s,start=%s): %w", shipmentID, segment.StartTime.Format(time.RFC3339), err)
			}
		} else {
			return fmt.Errorf("查询停留记录失败(shipment=%s,start=%s): %w", shipmentID, segment.StartTime.Format(time.RFC3339), err)
		}
	}

	// 如果当前窗口最新轨迹点速度明显高于低速阈值，补齐该运单活跃停留记录的结束状态
	lastTrack := tracks[len(tracks)-1]
	lastTrackIsStationary := lastTrack.Speed <= stationarySpeedThreshold
	if !lastTrackIsStationary && len(tracks) > 1 {
		prevTrack := tracks[len(tracks)-2]
		dist := CalculateDistance(prevTrack.Latitude, prevTrack.Longitude, lastTrack.Latitude, lastTrack.Longitude)
		timeDiffHours := lastTrack.LocateTime.Sub(prevTrack.LocateTime).Hours()
		if timeDiffHours > 0 && (dist/timeDiffHours) <= stationarySpeedThreshold {
			lastTrackIsStationary = true
		} else if timeDiffHours <= 0 && dist == 0 {
			lastTrackIsStationary = true
		}
	}

	if !lastTrackIsStationary {
		var activeRecords []models.DeviceStopRecord
		if err := s.db.
			Where("shipment_id = ? AND device_id = ? AND status = ?", shipmentID, deviceID, "active").
			Find(&activeRecords).Error; err != nil {
			return fmt.Errorf("查询活跃停留记录失败(shipment=%s): %w", shipmentID, err)
		}

		for _, active := range activeRecords {
			endAt := lastTrack.LocateTime
			durationSeconds := int(endAt.Sub(active.StartTime).Seconds())
			if durationSeconds < 0 {
				durationSeconds = 0
			}

			updates := map[string]interface{}{
				"status":           "completed",
				"end_time":         endAt,
				"duration_seconds": durationSeconds,
				"duration_text":    formatDuration(durationSeconds),
				"updated_at":       now,
			}
			if err := s.db.Model(&models.DeviceStopRecord{}).
				Where("id = ?", active.ID).
				Updates(updates).Error; err != nil {
				return fmt.Errorf("关闭活跃停留记录失败(id=%s): %w", active.ID, err)
			}
		}
	}

	if err := s.deduplicateActiveStops(shipmentID, deviceID, now); err != nil {
		return fmt.Errorf("清理重复活跃停留记录失败(shipment=%s,device=%s): %w", shipmentID, deviceID, err)
	}

	return nil
}

func (s *DeviceStopService) deduplicateActiveStops(shipmentID, deviceID string, now time.Time) error {
	var activeRecords []models.DeviceStopRecord
	if err := s.db.Where("shipment_id = ? AND device_id = ? AND status = ?", shipmentID, deviceID, "active").
		Order("start_time ASC").
		Find(&activeRecords).Error; err != nil {
		return err
	}

	if len(activeRecords) <= 1 {
		return nil
	}

	keeper := activeRecords[0]
	for _, r := range activeRecords[1:] {
		durationSeconds := int(now.Sub(r.StartTime).Seconds())
		if durationSeconds < 0 {
			durationSeconds = 0
		}
		if err := s.db.Model(&models.DeviceStopRecord{}).
			Where("id = ?", r.ID).
			Updates(map[string]interface{}{
				"status":           "completed",
				"end_time":         now,
				"duration_seconds": durationSeconds,
				"duration_text":    formatDuration(durationSeconds),
				"updated_at":       now,
			}).Error; err != nil {
			return err
		}
	}

	// keeper 重新按当前时间刷新一次时长，确保展示稳定
	keeperDuration := int(now.Sub(keeper.StartTime).Seconds())
	if keeperDuration < 0 {
		keeperDuration = 0
	}
	return s.db.Model(&models.DeviceStopRecord{}).
		Where("id = ?", keeper.ID).
		Updates(map[string]interface{}{
			"duration_seconds": keeperDuration,
			"duration_text":    formatDuration(keeperDuration),
			"updated_at":       now,
		}).Error
}

// generateAddressFromCoordinates 根据经纬度生成简单地址描述
func generateAddressFromCoordinates(lat, lng float64) string {
	return ResolveNodeAddress(lat, lng)
}

// ResolveNodeAddress 将停留点经纬度解析为可读地址（优先返回中英文双语）
func ResolveNodeAddress(lat, lng float64) string {
	if !isValidStopCoordinate(lat, lng) {
		return ""
	}

	cacheKey := fmt.Sprintf("%.4f,%.4f", lat, lng)
	if cached, ok := stopAddressCache.Load(cacheKey); ok {
		if text, okCast := cached.(string); okCast && strings.TrimSpace(text) != "" {
			return text
		}
	}

	address := strings.TrimSpace(resolveBilingualAddressFromGeo(lat, lng))
	address = EnsureBilingualNodeAddress(address, lat, lng)

	stopAddressCache.Store(cacheKey, address)
	return address
}

// EnsureBilingualNodeAddress 统一将地址规范为“中文 / English”格式
func EnsureBilingualNodeAddress(address string, lat, lng float64) string {
	zhAddress, enAddress := splitBilingualAddress(address)

	// 若“中文侧”并不含中文字符，视为缺失，后续使用兜底补齐。
	if zhAddress != "" && !containsChinese(zhAddress) {
		zhAddress = ""
	}
	// 若“英文侧”仍含中文字符，也视为缺失，后续使用兜底补齐。
	if enAddress != "" && containsChinese(enAddress) {
		enAddress = ""
	}

	if zhAddress == "" || enAddress == "" {
		if isValidStopCoordinate(lat, lng) {
			fallbackZH, fallbackEN := splitBilingualAddress(formatCoordinateFallbackAddress(lat, lng))
			if zhAddress == "" {
				zhAddress = fallbackZH
			}
			if enAddress == "" {
				enAddress = fallbackEN
			}
		}
	}

	return composeBilingualAddress(zhAddress, enAddress)
}

func splitBilingualAddress(address string) (string, string) {
	text := strings.TrimSpace(strings.ReplaceAll(address, "／", "/"))
	if text == "" {
		return "", ""
	}

	parts := strings.SplitN(text, "/", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}

	if containsChinese(text) {
		return text, ""
	}
	return "", text
}

// IsCoordinateAddress 判断地址是否仍是经纬度兜底文本
func IsCoordinateAddress(address string) bool {
	text := strings.TrimSpace(strings.ToLower(address))
	if text == "" {
		return false
	}

	markers := []string{
		"北纬", "南纬", "东经", "西经",
		"latitude", "longitude", "lat", "lng",
		"国际区域",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}

	parts := strings.Split(text, ",")
	if len(parts) == 2 {
		var lat, lng float64
		if _, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%f", &lat); err == nil {
			if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &lng); err == nil {
				return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
			}
		}
	}

	return false
}

func isValidStopCoordinate(lat, lng float64) bool {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return false
	}
	if math.Abs(lat) < 0.000001 && math.Abs(lng) < 0.000001 {
		return false
	}
	return true
}

func resolveBilingualAddressFromGeo(lat, lng float64) string {
	var zhAddress string
	var enAddress string

	if TencentMap != nil {
		if detail, err := TencentMap.ReverseGeocodeDetailWithLanguage(lat, lng, "zh-CN"); err == nil && detail != nil {
			zhAddress = formatDetailAddress(detail, true)
		}
		if detail, err := TencentMap.ReverseGeocodeDetailWithLanguage(lat, lng, "en"); err == nil && detail != nil {
			enAddress = formatDetailAddress(detail, false)
		}
	}

	if zhAddress == "" || enAddress == "" {
		geoService := NewGeocodingService()

		if zhAddress == "" {
			if result, err := geoService.ReverseGeocodeWithLanguage(lat, lng, "zh-CN"); err == nil && result != nil {
				zhAddress = strings.TrimSpace(result.DisplayName)
			}
		}

		if enAddress == "" {
			if result, err := geoService.ReverseGeocodeWithLanguage(lat, lng, "en"); err == nil && result != nil {
				enAddress = strings.TrimSpace(result.DisplayName)
			}
		}

		if zhAddress == "" || enAddress == "" {
			if geocodeResult, err := geoService.ReverseGeocode(lat, lng); err == nil && geocodeResult != nil {
				displayName := strings.TrimSpace(geocodeResult.DisplayName)
				if zhAddress == "" {
					zhAddress = displayName
				}
				if enAddress == "" && !containsChinese(displayName) {
					enAddress = displayName
				}
			}
		}
	}

	return composeBilingualAddress(zhAddress, enAddress)
}

func formatDetailAddress(detail *ReverseGeocodeDetail, isChinese bool) string {
	if detail == nil {
		return ""
	}

	address := strings.TrimSpace(detail.Address)
	if address != "" {
		return address
	}

	parts := make([]string, 0, 4)
	for _, value := range []string{detail.Country, detail.Province, detail.City, detail.District} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(parts) > 0 && strings.EqualFold(parts[len(parts)-1], value) {
			continue
		}
		parts = append(parts, value)
	}

	if len(parts) == 0 {
		return ""
	}
	if isChinese {
		return strings.Join(parts, "")
	}
	return strings.Join(parts, ", ")
}

func composeBilingualAddress(zhAddress, enAddress string) string {
	zhAddress = strings.TrimSpace(zhAddress)
	enAddress = strings.TrimSpace(enAddress)

	if zhAddress != "" && enAddress != "" {
		if strings.EqualFold(zhAddress, enAddress) {
			return zhAddress
		}
		return zhAddress + " / " + enAddress
	}
	if zhAddress != "" {
		return zhAddress
	}
	return enAddress
}

func containsChinese(text string) bool {
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

func formatCoordinateFallbackAddress(lat, lng float64) string {
	// 简单的国家判断
	country := findCountry(lat, lng)
	countryEN := translateCountryToEnglish(country)

	latDir := "北纬"
	if lat < 0 {
		latDir = "南纬"
	}
	lngDir := "东经"
	if lng < 0 {
		lngDir = "西经"
	}

	coordsZH := fmt.Sprintf("%s%.4f°, %s%.4f°", latDir, math.Abs(lat), lngDir, math.Abs(lng))

	latEN := "N"
	if lat < 0 {
		latEN = "S"
	}
	lngEN := "E"
	if lng < 0 {
		lngEN = "W"
	}
	coordsEN := fmt.Sprintf("Lat %.4f°%s, Lng %.4f°%s", math.Abs(lat), latEN, math.Abs(lng), lngEN)

	if country != "" {
		if countryEN == "" {
			countryEN = "Unknown Region"
		}
		return fmt.Sprintf("%s %s / %s %s", country, coordsZH, countryEN, coordsEN)
	}
	return fmt.Sprintf("国际区域 %s / International Waters %s", coordsZH, coordsEN)
}

func translateCountryToEnglish(country string) string {
	switch strings.TrimSpace(country) {
	case "中国":
		return "China"
	case "俄罗斯":
		return "Russia"
	case "哈萨克斯坦":
		return "Kazakhstan"
	case "蒙古":
		return "Mongolia"
	case "德国":
		return "Germany"
	case "法国":
		return "France"
	default:
		return ""
	}
}

// findCountry 根据经纬度查找所属国家(简化版)
func findCountry(lat, lng float64) string {
	// 国家边界定义
	countries := []struct {
		Name   string
		MinLat float64
		MaxLat float64
		MinLng float64
		MaxLng float64
	}{
		{"中国", 18.2, 53.6, 73.5, 135.1},
		{"俄罗斯", 41.2, 81.9, 19.6, 169.0},
		{"哈萨克斯坦", 40.7, 55.4, 46.5, 87.3},
		{"蒙古", 41.6, 52.1, 87.7, 119.9},
		{"德国", 47.3, 55.1, 5.9, 15.0},
		{"法国", 41.3, 51.1, -4.8, 9.6},
	}

	for _, c := range countries {
		if lat >= c.MinLat && lat <= c.MaxLat && lng >= c.MinLng && lng <= c.MaxLng {
			return c.Name
		}
	}
	return ""
}
