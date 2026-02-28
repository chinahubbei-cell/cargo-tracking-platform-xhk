package services

import (
	"fmt"
	"log"
	"sync"
	"time"
	"trackcard-server/models"

	"gorm.io/gorm"
)

// CarrierEvent 船司事件 (统一格式)
type CarrierEvent struct {
	EventCode  string     `json:"event_code"`
	EventName  string     `json:"event_name"`
	Location   string     `json:"location"`
	LoCode     string     `json:"lo_code"`
	Latitude   *float64   `json:"latitude"`
	Longitude  *float64   `json:"longitude"`
	VesselName string     `json:"vessel_name"`
	VoyageNo   string     `json:"voyage_no"`
	Carrier    string     `json:"carrier"`
	EventTime  time.Time  `json:"event_time"`
	ETAUpdate  *time.Time `json:"eta_update"`
	IsActual   bool       `json:"is_actual"`
}

// CarrierProvider 统一船司API接口
type CarrierProvider interface {
	Name() string
	IsConfigured() bool
	TrackByBL(billOfLading string) ([]CarrierEvent, error)
}

// CarrierTrackingService 船司追踪服务
type CarrierTrackingService struct {
	db       *gorm.DB
	provider CarrierProvider
	mu       sync.RWMutex
}

var CarrierTracking *CarrierTrackingService

// InitCarrierTracking 初始化船司追踪服务
func InitCarrierTracking(db *gorm.DB, provider CarrierProvider) {
	CarrierTracking = &CarrierTrackingService{
		db:       db,
		provider: provider,
	}
	log.Printf("[CarrierTracking] 初始化完成, Provider: %s, Configured: %v",
		provider.Name(), provider.IsConfigured())
}

// IsConfigured 检查是否已配置
func (s *CarrierTrackingService) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.provider != nil && s.provider.IsConfigured()
}

// ProviderName 获取提供商名称
func (s *CarrierTrackingService) ProviderName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.provider == nil {
		return "none"
	}
	return s.provider.Name()
}

// TrackShipment 追踪单个运单
func (s *CarrierTrackingService) TrackShipment(shipment *models.Shipment) ([]models.CarrierTrack, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("船司追踪服务未配置")
	}

	if shipment.BillOfLading == "" {
		return nil, fmt.Errorf("运单 %s 无提单号", shipment.ID)
	}

	// 调用API获取事件
	events, err := s.provider.TrackByBL(shipment.BillOfLading)
	if err != nil {
		return nil, fmt.Errorf("API查询失败: %v", err)
	}

	// 转换并保存
	var tracks []models.CarrierTrack
	now := time.Now()

	for _, e := range events {
		track := models.CarrierTrack{
			ShipmentID:   shipment.ID,
			BillOfLading: shipment.BillOfLading,
			EventCode:    e.EventCode,
			EventName:    e.EventName,
			Location:     e.Location,
			LoCode:       e.LoCode,
			Latitude:     e.Latitude,
			Longitude:    e.Longitude,
			VesselName:   e.VesselName,
			VoyageNo:     e.VoyageNo,
			Carrier:      e.Carrier,
			EventTime:    e.EventTime,
			ETAUpdate:    e.ETAUpdate,
			IsActual:     e.IsActual,
			Source:       s.provider.Name(),
			SyncedAt:     now,
		}

		// 避免重复插入 (同一运单+同一事件代码+同一时间)
		var existing models.CarrierTrack
		result := s.db.Where("shipment_id = ? AND event_code = ? AND event_time = ?",
			track.ShipmentID, track.EventCode, track.EventTime).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			if err := s.db.Create(&track).Error; err != nil {
				log.Printf("[CarrierTracking] 保存事件失败: %v", err)
				continue
			}
			tracks = append(tracks, track)

			// 同步创建/更新里程碑
			s.updateMilestone(shipment, &track)
		} else {
			// 更新现有记录
			s.db.Model(&existing).Updates(map[string]interface{}{
				"eta_update": e.ETAUpdate,
				"synced_at":  now,
			})
		}
	}

	// 更新运单船务信息
	s.updateShipmentFromEvents(shipment, events)

	log.Printf("[CarrierTracking] 运单 %s 同步 %d 个事件", shipment.ID, len(tracks))
	return tracks, nil
}

// updateMilestone 更新里程碑
func (s *CarrierTrackingService) updateMilestone(shipment *models.Shipment, track *models.CarrierTrack) {
	// 查找或创建里程碑 (使用 CarrierMilestone)
	var milestone models.CarrierMilestone
	result := s.db.Where("shipment_id = ? AND code = ?", shipment.ID, track.EventCode).First(&milestone)

	sequence := models.SeaMilestoneSequence[track.EventCode]
	if sequence == 0 {
		sequence = 99 // 未知事件放最后
	}

	name := models.EventCodeToName[track.EventCode]
	if name == "" {
		name = track.EventName
	}

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新里程碑
		milestone = models.CarrierMilestone{
			ShipmentID:     shipment.ID,
			Code:           track.EventCode,
			Name:           name,
			Sequence:       sequence,
			Status:         models.MilestoneStatusActual,
			ActualTime:     &track.EventTime,
			Source:         models.MilestoneSourceCarrier,
			Location:       track.Location,
			LoCode:         track.LoCode,
			CarrierTrackID: &track.ID,
		}
		s.db.Create(&milestone)
	} else {
		// 更新现有里程碑
		if track.IsActual {
			s.db.Model(&milestone).Updates(map[string]interface{}{
				"status":           models.MilestoneStatusActual,
				"actual_time":      track.EventTime,
				"location":         track.Location,
				"carrier_track_id": track.ID,
			})
		}
	}
}

// updateShipmentFromEvents 根据事件更新运单信息
func (s *CarrierTrackingService) updateShipmentFromEvents(shipment *models.Shipment, events []CarrierEvent) {
	updates := make(map[string]interface{})

	for _, e := range events {
		// 更新船务信息
		if e.VesselName != "" && shipment.VesselName == "" {
			updates["vessel_name"] = e.VesselName
		}
		if e.VoyageNo != "" && shipment.VoyageNo == "" {
			updates["voyage_no"] = e.VoyageNo
		}
		if e.Carrier != "" && shipment.Carrier == "" {
			updates["carrier"] = e.Carrier
		}

		// 更新ATA (实际到达)
		if e.EventCode == models.EventVesselArrival && e.IsActual {
			updates["ata"] = e.EventTime
		}

		// 更新ETA (如有变更)
		if e.ETAUpdate != nil {
			updates["eta"] = *e.ETAUpdate
		}
	}

	if len(updates) > 0 {
		s.db.Model(shipment).Updates(updates)
	}
}

// GetShipmentTracks 获取运单的船司事件列表
func (s *CarrierTrackingService) GetShipmentTracks(shipmentID string) ([]models.CarrierTrack, error) {
	var tracks []models.CarrierTrack
	err := s.db.Where("shipment_id = ?", shipmentID).
		Order("event_time ASC").
		Find(&tracks).Error
	return tracks, err
}

// GetShipmentMilestones 获取运单的船司里程碑
func (s *CarrierTrackingService) GetShipmentMilestones(shipmentID string) ([]models.CarrierMilestone, error) {
	var milestones []models.CarrierMilestone
	err := s.db.Where("shipment_id = ?", shipmentID).
		Order("sequence ASC, actual_time ASC").
		Find(&milestones).Error
	return milestones, err
}
