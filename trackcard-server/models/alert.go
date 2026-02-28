package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Alert struct {
	ID         string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	DeviceID   *string        `gorm:"type:varchar(50);column:device_id" json:"device_id"`
	ShipmentID *string        `gorm:"type:varchar(50);column:shipment_id" json:"shipment_id"`
	Type       string         `gorm:"type:varchar(50);not null" json:"type"`
	Severity   string         `gorm:"type:varchar(20);default:'warning'" json:"severity"`
	Title      string         `gorm:"type:varchar(255);not null" json:"title"`
	Message    *string        `gorm:"type:text" json:"message"`
	Location   *string        `gorm:"type:varchar(255)" json:"location"`
	Status     string         `gorm:"type:varchar(20);default:'pending'" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	ResolvedAt *time.Time     `gorm:"column:resolved_at" json:"resolved_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Category string    `gorm:"type:varchar(50);default:'system'" json:"category"` // 预警类别: physical, node, operation
	Device   *Device   `gorm:"foreignKey:DeviceID;references:ID" json:"device,omitempty"`
	Shipment *Shipment `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
}

func (a *Alert) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = "alert-" + uuid.New().String()[:8]
	}
	return nil
}

type AlertCreateRequest struct {
	DeviceID   *string `json:"device_id"`
	ShipmentID *string `json:"shipment_id"`
	Type       string  `json:"type" binding:"required"`
	Severity   string  `json:"severity"`
	Title      string  `json:"title" binding:"required"`
	Message    *string `json:"message"`
	Location   *string `json:"location"`
}

type AlertUpdateRequest struct {
	Status   *string `json:"status"`
	Severity *string `json:"severity"`
	Message  *string `json:"message"`
}
