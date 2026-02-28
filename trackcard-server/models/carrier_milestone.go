package models

import (
	"time"

	"gorm.io/gorm"
)

// CarrierMilestone 船司追踪里程碑 (Phase 2 专用)
// 用于存储船司API返回的里程碑事件，与Phase 7的节点配置引擎分离
type CarrierMilestone struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ShipmentID string `gorm:"type:varchar(50);index" json:"shipment_id"`

	// 里程碑信息
	Code     string `gorm:"type:varchar(50)" json:"code"`  // 标准化事件代码
	Name     string `gorm:"type:varchar(100)" json:"name"` // 中文名称
	Sequence int    `gorm:"default:0" json:"sequence"`     // 顺序 (用于排序显示)

	// 状态与时间
	Status      string     `gorm:"type:varchar(20);default:'planned'" json:"status"` // planned, actual, skipped
	PlannedTime *time.Time `json:"planned_time"`
	ActualTime  *time.Time `json:"actual_time"`

	// 来源与位置
	Source   string `gorm:"type:varchar(50)" json:"source"` // iot, carrier, manual, geofence
	Location string `gorm:"type:varchar(200)" json:"location"`
	LoCode   string `gorm:"type:varchar(20)" json:"lo_code"`

	// 关联数据源ID
	CarrierTrackID *uint `json:"carrier_track_id"` // 关联的船司事件
	DeviceTrackID  *uint `json:"device_track_id"`  // 关联的设备轨迹

	// 备注
	Remark    string         `gorm:"type:varchar(500)" json:"remark"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Shipment *Shipment `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
}

// TableName 指定表名
func (CarrierMilestone) TableName() string {
	return "carrier_milestones"
}

// Note: EventCodeToName and Event* constants are defined in carrier_track.go
// to avoid duplicate declarations
