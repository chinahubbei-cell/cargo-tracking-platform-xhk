package models

import (
	"time"
)

// ShipmentSequence 运单序号表，用于生成唯一运单号
type ShipmentSequence struct {
	DatePrefix   string    `gorm:"primaryKey;type:varchar(6)" json:"date_prefix"` // 日期前缀，如 "260116"
	LastSequence int       `gorm:"default:0" json:"last_sequence"`                // 当日最后使用的序号
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ShipmentSequence) TableName() string {
	return "shipment_sequences"
}
