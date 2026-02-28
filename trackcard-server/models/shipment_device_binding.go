package models

import (
	"time"

	"gorm.io/gorm"
)

// ShipmentDeviceBinding 运单设备绑定历史
// 记录运单与设备的完整绑定历史，支持中途更换设备和历史轨迹追溯
type ShipmentDeviceBinding struct {
	ID            uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	ShipmentID    string         `json:"shipment_id" gorm:"index;not null"` // 运单ID
	DeviceID      string         `json:"device_id" gorm:"index;not null"`   // 设备ID
	BoundAt       time.Time      `json:"bound_at" gorm:"not null"`          // 绑定时间
	UnboundAt     *time.Time     `json:"unbound_at"`                        // 解绑时间（null表示当前绑定）
	UnboundReason string         `json:"unbound_reason"`                    // 解绑原因: replaced(更换设备), completed(运单完成), manual(手动解绑)
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Shipment *Shipment `json:"shipment,omitempty" gorm:"foreignKey:ShipmentID"`
	Device   *Device   `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
}

// TableName 表名
func (ShipmentDeviceBinding) TableName() string {
	return "shipment_device_bindings"
}

// IsActive 是否当前有效绑定
func (b *ShipmentDeviceBinding) IsActive() bool {
	return b.UnboundAt == nil
}
