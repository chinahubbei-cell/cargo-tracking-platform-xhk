package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Device struct {
	ID               string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	HighValueWarning bool           `gorm:"default:false" json:"high_value_warning"` // 高价值预警
	Name             string         `gorm:"type:varchar(100);not null" json:"name"`
	Type             string         `gorm:"type:varchar(50);default:'container'" json:"type"`
	Status           string         `gorm:"type:varchar(20);default:'online'" json:"status"`
	Battery          int            `gorm:"default:100" json:"battery"`
	Latitude         *float64       `gorm:"type:decimal(10,6)" json:"latitude"`
	Longitude        *float64       `gorm:"type:decimal(10,6)" json:"longitude"`
	ExternalDeviceID *string        `gorm:"type:varchar(50);column:external_device_id" json:"external_device_id"`
	Provider         string         `gorm:"type:varchar(50);default:'kuaihuoyun'" json:"provider"`
	Speed            float64        `gorm:"default:0" json:"speed"`
	Direction        float64        `gorm:"default:0" json:"direction"`
	Temperature      *float64       `gorm:"type:decimal(5,2)" json:"temperature"`
	Humidity         *float64       `gorm:"type:decimal(5,2)" json:"humidity"`
	Shock            *float64       `gorm:"type:decimal(5,2)" json:"shock"` // 震动 g值
	Tilt             *float64       `gorm:"type:decimal(5,2)" json:"tilt"`  // 倾斜 角度
	Light            *float64       `gorm:"type:decimal(5,2)" json:"light"` // 光照 lux
	LocateType       *int           `gorm:"column:locate_type" json:"locate_type"`
	OrgID            *string        `gorm:"type:varchar(50);index" json:"org_id"` // 所属组织ID
	SubAccountID     *string        `gorm:"type:varchar(50);index" json:"sub_account_id"`
	ServiceStatus    string         `gorm:"type:varchar(20);default:'active'" json:"service_status"`
	ServiceStartAt   *time.Time     `json:"service_start_at"`
	ServiceEndAt     *time.Time     `json:"service_end_at"`
	LastUpdate       time.Time      `gorm:"column:last_update;default:CURRENT_TIMESTAMP" json:"last_update"`
	CreatedAt        time.Time      `json:"created_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = "GC-" + uuid.New().String()[:8]
	}
	return nil
}

type DeviceCreateRequest struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"` // 可选，不再是必填
	Type             string   `json:"type"`
	Provider         string   `json:"provider"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	ExternalDeviceID *string  `json:"external_device_id"`
	OrgID            *string  `json:"org_id"`         // 所属组织ID
	SubAccountID     *string  `json:"sub_account_id"` // 分机构/子账号ID
	ServiceStartAt   *string  `json:"service_start_at,omitempty"`
	ServiceEndAt     *string  `json:"service_end_at,omitempty"`
}

type DeviceUpdateRequest struct {
	Name             *string  `json:"name"`
	Type             *string  `json:"type"`
	Provider         *string  `json:"provider"`
	Status           *string  `json:"status"`
	Battery          *int     `json:"battery"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	ExternalDeviceID *string  `json:"external_device_id"`
	OrgID            *string  `json:"org_id"`         // 所属组织ID
	SubAccountID     *string  `json:"sub_account_id"` // 分机构/子账号ID
	ServiceStatus    *string  `json:"service_status"`
	ServiceStartAt   *string  `json:"service_start_at,omitempty"`
	ServiceEndAt     *string  `json:"service_end_at,omitempty"`
}
