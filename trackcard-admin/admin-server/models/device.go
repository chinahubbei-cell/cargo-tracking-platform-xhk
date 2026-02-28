package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HardwareDevice 硬件设备（库存管理）
type HardwareDevice struct {
	ID          string `gorm:"primaryKey" json:"id"`
	DeviceType  string `gorm:"size:50;not null" json:"device_type"` // tracker, sensor, gateway
	DeviceModel string `gorm:"size:50" json:"device_model"`         // 型号
	IMEI        string `gorm:"uniqueIndex;size:20" json:"imei"`     // IMEI号
	SN          string `gorm:"uniqueIndex;size:50" json:"sn"`       // 序列号
	ICCID       string `gorm:"size:30" json:"iccid"`                // SIM卡号
	SimStatus   string `gorm:"size:20" json:"sim_status"`           // active, suspended, cancelled

	// 状态
	Status string `gorm:"size:20;default:in_stock" json:"status"` // in_stock, allocated, activated, returned, damaged

	// 设备归属层级（支持多层级）
	// 第一层：所属组织
	OrgID   string `gorm:"size:50;index" json:"org_id"`
	OrgName string `gorm:"size:200" json:"org_name"`

	// 第二层：分配给客户的子账号（可选）
	SubAccountID   string `gorm:"size:50;index" json:"sub_account_id"`
	SubAccountName string `gorm:"size:100" json:"sub_account_name"`

	// 第三层：绑定的运单（动态）
	BoundShipmentID string `gorm:"size:50" json:"bound_shipment_id"`

	// 分配信息
	AllocatedAt *time.Time `json:"allocated_at"`
	AllocatedBy string     `gorm:"size:50" json:"allocated_by"`
	OrderID     string     `gorm:"size:50" json:"order_id"` // 关联订单

	// 激活信息
	ActivatedAt *time.Time `json:"activated_at"`
	LastOnline  *time.Time `json:"last_online"`

	// 退回信息
	ReturnedAt   *time.Time `json:"returned_at"`
	ReturnReason string     `gorm:"size:200" json:"return_reason"`

	// 入库信息
	BatchNo     string    `gorm:"size:50" json:"batch_no"` // 入库批次
	PurchaseAt  time.Time `json:"purchase_at"`             // 采购日期
	WarrantyEnd time.Time `json:"warranty_end"`            // 保修截止

	Remark    string    `gorm:"size:500" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *HardwareDevice) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// IsAvailable 设备是否可分配
func (d *HardwareDevice) IsAvailable() bool {
	return d.Status == "in_stock"
}

// IsActive 设备是否激活
func (d *HardwareDevice) IsActive() bool {
	return d.Status == "activated"
}

// DeviceAllocationLog 设备分配日志
type DeviceAllocationLog struct {
	ID       string `gorm:"primaryKey" json:"id"`
	DeviceID string `gorm:"size:50;not null;index" json:"device_id"`
	IMEI     string `gorm:"size:20" json:"imei"`
	Action   string `gorm:"size:20;not null" json:"action"` // allocate, return, transfer, bind, unbind

	FromOrgID string `gorm:"size:50" json:"from_org_id"`
	ToOrgID   string `gorm:"size:50" json:"to_org_id"`
	OrgName   string `gorm:"size:200" json:"org_name"`

	OrderID    string `gorm:"size:50" json:"order_id"`
	ShipmentID string `gorm:"size:50" json:"shipment_id"`

	Remark    string    `gorm:"size:500" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `gorm:"size:50" json:"created_by"`
}

func (l *DeviceAllocationLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

// DeviceStats 设备统计
type DeviceStats struct {
	Total     int `json:"total"`
	InStock   int `json:"in_stock"`
	Allocated int `json:"allocated"`
	Activated int `json:"activated"`
	Returned  int `json:"returned"`
	Damaged   int `json:"damaged"`
}
