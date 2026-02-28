package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceStopRecord 设备停留记录表
// 用于记录设备基于 runStatus 的停留信息
type DeviceStopRecord struct {
	ID         string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	// 设备和运单关联
	DeviceExternalID string         `gorm:"type:varchar(50);not null;index:idx_device_stop_external" json:"device_external_id"`
	DeviceID         string         `gorm:"type:varchar(50);index:idx_device_stop_id" json:"device_id"`
	ShipmentID       string         `gorm:"type:varchar(50);index:idx_device_stop_shipment" json:"shipment_id"`
	// 停留时间信息
	StartTime      time.Time      `gorm:"not null" json:"start_time"`
	EndTime        *time.Time     `json:"end_time"`
	DurationSeconds int           `gorm:"default:0" json:"duration_seconds"`
	DurationText    string        `gorm:"type:varchar(50)" json:"duration_text"`
	// 停留位置信息
	Latitude  *float64 `gorm:"type:decimal(10,7)" json:"latitude"`
	Longitude *float64 `gorm:"type:decimal(10,7)" json:"longitude"`
	Address   string   `gorm:"type:varchar(500)" json:"address"`
	// 停留状态: active=停留中, completed=已结束
	Status string `gorm:"type:varchar(20);default:'active';index:idx_device_stop_status_time" json:"status"`
	// 预警状态
	AlertSent         bool `gorm:"default:false" json:"alert_sent"`
	AlertThresholdHours int  `gorm:"default:24" json:"alert_threshold_hours"`
	// 元数据
	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (d *DeviceStopRecord) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = "DSR-" + uuid.New().String()[:8]
	}
	return nil
}

// DeviceStopRecordListRequest 停留记录列表查询请求
type DeviceStopRecordListRequest struct {
	DeviceID         string `json:"device_id"`
	DeviceExternalID string `json:"device_external_id"`
	ShipmentID       string `json:"shipment_id"`
	Status           string `json:"status"` // active, completed, all
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	Page             int    `json:"page"`
	PageSize         int    `json:"page_size"`
}

// DeviceStopRecordResponse 停留记录响应
type DeviceStopRecordResponse struct {
	ID               string    `json:"id"`
	DeviceID         string    `json:"device_id"`
	DeviceExternalID string    `json:"device_external_id"`
	ShipmentID       string    `json:"shipment_id"`
	StartTime        time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	DurationSeconds  int       `json:"duration_seconds"`
	DurationText     string    `json:"duration_text"`
	Latitude         *float64  `json:"latitude"`
	Longitude        *float64  `json:"longitude"`
	Address          string    `json:"address"`
	Status           string    `json:"status"`
	AlertSent        bool      `json:"alert_sent"`
	CreatedAt        time.Time `json:"created_at"`
}

// DeviceStopStats 设备停留统计
type DeviceStopStats struct {
	DeviceID         string  `json:"device_id"`
	DeviceExternalID string  `json:"device_external_id"`
	TotalStops       int     `json:"total_stops"`        // 总停留次数
	TotalDuration    int     `json:"total_duration"`     // 总停留时长(秒)
	AverageDuration  int     `json:"average_duration"`   // 平均停留时长(秒)
	CurrentStop      *DeviceStopRecordResponse `json:"current_stop,omitempty"` // 当前停留记录(如果有)
}
