package models

import (
	"time"

	"gorm.io/gorm"
)

// ShipmentLog 运单操作日志
// 记录运单的所有变更历史，便于追溯查询
type ShipmentLog struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	ShipmentID  string         `json:"shipment_id" gorm:"index;not null"` // 运单ID
	Action      string         `json:"action" gorm:"not null"`            // 操作类型: created, updated, status_changed, device_bound, device_unbound, etc.
	Field       string         `json:"field"`                             // 变更字段（如 status, device_id 等）
	OldValue    string         `json:"old_value"`                         // 变更前的值
	NewValue    string         `json:"new_value"`                         // 变更后的值
	Description string         `json:"description"`                       // 操作描述
	OperatorID  string         `json:"operator_id"`                       // 操作人ID（如果有）
	OperatorIP  string         `json:"operator_ip"`                       // 操作人IP
	CreatedAt   time.Time      `json:"created_at" gorm:"index"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 表名
func (ShipmentLog) TableName() string {
	return "shipment_logs"
}

// 操作类型常量
const (
	LogActionCreated        = "created"         // 创建运单
	LogActionUpdated        = "updated"         // 更新运单
	LogActionStatusChanged  = "status_changed"  // 状态变更
	LogActionDeviceBound    = "device_bound"    // 绑定设备
	LogActionDeviceUnbound  = "device_unbound"  // 解绑设备
	LogActionDeviceReplaced = "device_replaced" // 更换设备
	LogActionDeleted        = "deleted"         // 删除运单
	LogActionLocationEnter  = "location_enter"  // 进入地理围栏
	LogActionLocationLeave  = "location_leave"  // 离开地理围栏
)
