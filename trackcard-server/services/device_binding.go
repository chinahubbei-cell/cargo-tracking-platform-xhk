package services

import (
	"log"
	"time"

	"gorm.io/gorm"

	"trackcard-server/models"
)

// DeviceBindingService 设备绑定服务
type DeviceBindingService struct {
	db *gorm.DB
}

// NewDeviceBindingService 创建设备绑定服务
func NewDeviceBindingService(db *gorm.DB) *DeviceBindingService {
	return &DeviceBindingService{db: db}
}

// BindDevice 绑定设备到运单
// 如果运单已有绑定设备，会先解绑旧设备（标记为replaced）
func (s *DeviceBindingService) BindDevice(shipmentID, deviceID string) error {
	if deviceID == "" {
		return nil // 空设备ID不处理
	}

	now := time.Now()

	// 查找当前有效绑定
	var existingBinding models.ShipmentDeviceBinding
	err := s.db.Where("shipment_id = ? AND unbound_at IS NULL", shipmentID).First(&existingBinding).Error

	if err != nil {
		// 如果绑定表无记录，检查shipments表是否有device_id（数据修复逻辑）
		var shipment models.Shipment
		if dbErr := s.db.Select("device_id").Where("id = ?", shipmentID).First(&shipment).Error; dbErr == nil {
			if shipment.DeviceID != nil && *shipment.DeviceID != "" && *shipment.DeviceID != deviceID {
				// shipments表有设备但绑定表没有记录，说明数据不一致
				// 创建一条需要解绑的记录（修复历史数据）
				log.Printf("[DeviceBinding] 检测到数据不一致: 运单 %s 在shipments表绑定设备 %s，但bindings表无记录，正在修复...", shipmentID, *shipment.DeviceID)
				existingBinding = models.ShipmentDeviceBinding{
					ShipmentID: shipmentID,
					DeviceID:   *shipment.DeviceID,
					BoundAt:    now.Add(-1 * time.Second), // 比当前早1秒
				}
				s.db.Create(&existingBinding)
				// 立即解绑
				s.db.Model(&existingBinding).Updates(map[string]interface{}{
					"unbound_at":     now,
					"unbound_reason": "replaced_data_repair",
				})
				log.Printf("[DeviceBinding] 数据修复完成: 运单 %s 解绑旧设备 %s", shipmentID, *shipment.DeviceID)
			}
		}
	} else {
		// 已有绑定，检查是否是同一设备
		if existingBinding.DeviceID == deviceID {
			return nil // 同一设备，无需操作
		}
		// 解绑旧设备
		s.db.Model(&existingBinding).Updates(map[string]interface{}{
			"unbound_at":     now,
			"unbound_reason": "replaced",
		})
		log.Printf("[DeviceBinding] 运单 %s 解绑设备 %s (更换设备)", shipmentID, existingBinding.DeviceID)
	}

	// 创建新绑定
	newBinding := models.ShipmentDeviceBinding{
		ShipmentID: shipmentID,
		DeviceID:   deviceID,
		BoundAt:    now,
	}
	if err := s.db.Create(&newBinding).Error; err != nil {
		return err
	}

	log.Printf("[DeviceBinding] 运单 %s 绑定设备 %s", shipmentID, deviceID)

	// 绑定成功后，立即触发设备数据同步（获取最新位置）
	if Scheduler != nil {
		go func() {
			if err := Scheduler.SyncDeviceImmediate(deviceID); err != nil {
				log.Printf("[DeviceBinding] 设备 %s 即时同步失败: %v", deviceID, err)
			}
		}()
	}

	// 立即触发围栏检测（如果设备已经在围栏外，可以马上触发状态变更）
	if Geofence != nil {
		go func() {
			// 获取设备最新位置
			var device models.Device
			if err := s.db.First(&device, "id = ?", deviceID).Error; err == nil {
				if device.Latitude != nil && device.Longitude != nil {
					log.Printf("[DeviceBinding] 运单 %s 设备绑定后立即执行围栏检测", shipmentID)
					Geofence.CheckAndUpdateStatus(deviceID, *device.Latitude, *device.Longitude)
				}
			}
		}()
	}

	return nil
}

// UnbindDevice 解绑运单的当前设备
func (s *DeviceBindingService) UnbindDevice(shipmentID, reason string) error {
	now := time.Now()

	result := s.db.Model(&models.ShipmentDeviceBinding{}).
		Where("shipment_id = ? AND unbound_at IS NULL", shipmentID).
		Updates(map[string]interface{}{
			"unbound_at":     now,
			"unbound_reason": reason,
		})

	if result.RowsAffected > 0 {
		log.Printf("[DeviceBinding] 运单 %s 解绑设备 (原因: %s)", shipmentID, reason)
	}
	return result.Error
}

// GetCurrentBinding 获取运单当前绑定的设备
func (s *DeviceBindingService) GetCurrentBinding(shipmentID string) *models.ShipmentDeviceBinding {
	var binding models.ShipmentDeviceBinding
	err := s.db.Where("shipment_id = ? AND unbound_at IS NULL", shipmentID).First(&binding).Error
	if err != nil {
		return nil
	}
	return &binding
}

// GetBindingHistory 获取运单的设备绑定历史
func (s *DeviceBindingService) GetBindingHistory(shipmentID string) []models.ShipmentDeviceBinding {
	var bindings []models.ShipmentDeviceBinding
	s.db.Where("shipment_id = ?", shipmentID).Order("bound_at ASC").Find(&bindings)
	return bindings
}

// GetDeviceActiveBinding 获取设备的当前有效绑定（用于判断设备是否可用）
func (s *DeviceBindingService) GetDeviceActiveBinding(deviceID string) *models.ShipmentDeviceBinding {
	var binding models.ShipmentDeviceBinding
	// 只查询非完成状态的运单绑定
	err := s.db.Joins("JOIN shipments ON shipment_device_bindings.shipment_id = shipments.id").
		Where("shipment_device_bindings.device_id = ? AND shipment_device_bindings.unbound_at IS NULL AND shipments.status NOT IN ('delivered', 'cancelled')", deviceID).
		First(&binding).Error
	if err != nil {
		return nil
	}
	return &binding
}

// IsDeviceAvailable 检查设备是否可用（未绑定到进行中的运单）
func (s *DeviceBindingService) IsDeviceAvailable(deviceID string) bool {
	return s.GetDeviceActiveBinding(deviceID) == nil
}

// DeviceBinding 全局设备绑定服务实例
var DeviceBinding *DeviceBindingService

// InitDeviceBinding 初始化设备绑定服务
func InitDeviceBinding(db *gorm.DB) {
	DeviceBinding = NewDeviceBindingService(db)
	log.Println("[DeviceBinding] 设备绑定服务初始化完成")
}
