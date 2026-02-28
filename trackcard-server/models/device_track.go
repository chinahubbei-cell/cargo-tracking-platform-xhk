package models

import (
	"time"
)

// DeviceTrack 设备轨迹数据 - 存储从API同步的原始轨迹点
type DeviceTrack struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID string `gorm:"type:varchar(50);not null;index:idx_device_time" json:"device_id"`

	// 位置数据
	Latitude  float64 `gorm:"type:decimal(10,7);not null" json:"latitude"`
	Longitude float64 `gorm:"type:decimal(10,7);not null" json:"longitude"`
	Speed     float64 `gorm:"type:decimal(8,2);default:0" json:"speed"`
	Direction float64 `gorm:"type:decimal(5,2);default:0" json:"direction"`

	// 传感器数据
	Temperature *float64 `gorm:"type:decimal(5,2)" json:"temperature"`
	Humidity    *float64 `gorm:"type:decimal(5,2)" json:"humidity"`

	// 定位类型 (1=GPS, 2=基站, 3=WiFi)
	LocateType int `gorm:"default:1" json:"locate_type"`

	// 时间戳
	LocateTime time.Time `gorm:"not null;index:idx_device_time" json:"locate_time"` // 设备定位时间
	SyncedAt   time.Time `gorm:"autoCreateTime" json:"synced_at"`                   // 同步入库时间
}

// TableName 指定表名
func (DeviceTrack) TableName() string {
	return "device_tracks"
}

// DeviceSyncStatus 设备同步状态
type DeviceSyncStatus struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID string `gorm:"type:varchar(50);uniqueIndex;not null" json:"device_id"`

	// 同步状态
	LastSyncAt   time.Time `json:"last_sync_at"`                    // 最后同步时间
	LastLocateAt time.Time `json:"last_locate_at"`                  // 最后定位时间
	SyncInterval int       `gorm:"default:60" json:"sync_interval"` // 同步间隔（秒）

	// 设备活动状态 (active, idle, offline)
	ActivityStatus string `gorm:"type:varchar(20);default:'active'" json:"activity_status"`

	// 统计
	TotalPoints int64 `gorm:"default:0" json:"total_points"` // 总轨迹点数
	TodayPoints int   `gorm:"default:0" json:"today_points"` // 今日轨迹点数

	// 错误记录
	LastError  string `gorm:"type:text" json:"last_error"`
	ErrorCount int    `gorm:"default:0" json:"error_count"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (DeviceSyncStatus) TableName() string {
	return "device_sync_status"
}
