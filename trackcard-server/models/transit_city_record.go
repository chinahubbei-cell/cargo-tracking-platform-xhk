package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransitCityRecord 途经城市记录表
// 基于设备轨迹经纬度反向地理编码后去重，持久化存储运单途经的城市
type TransitCityRecord struct {
	ID         string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	ShipmentID string         `gorm:"type:varchar(50);not null;index:idx_transit_city_shipment" json:"shipment_id"`
	DeviceID   string         `gorm:"type:varchar(50);index:idx_transit_city_device" json:"device_id"`
	Country    string         `gorm:"type:varchar(100);not null" json:"country"`                 // 国家名称
	Province   string         `gorm:"type:varchar(100);default:''" json:"province"`              // 省/州名称
	City       string         `gorm:"type:varchar(100);not null" json:"city"`                    // 城市名称
	Latitude   float64        `gorm:"type:decimal(10,7)" json:"latitude"`                        // 首次进入时的纬度
	Longitude  float64        `gorm:"type:decimal(10,7)" json:"longitude"`                       // 首次进入时的经度
	EnteredAt  time.Time      `gorm:"not null;index:idx_transit_city_entered" json:"entered_at"` // 首次进入时间
	IsOversea  bool           `gorm:"default:false" json:"is_oversea"`                           // 是否海外城市
	CreatedAt  time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *TransitCityRecord) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = "TC-" + uuid.New().String()[:8]
	}
	return nil
}

// TransitCityResponse 途经城市响应
type TransitCityResponse struct {
	ID        string    `json:"id"`
	Country   string    `json:"country"`
	Province  string    `json:"province"`
	City      string    `json:"city"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	EnteredAt time.Time `json:"entered_at"`
	IsOversea bool      `json:"is_oversea"`
}
