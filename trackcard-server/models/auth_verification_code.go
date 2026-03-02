package models

import (
	"time"

	"gorm.io/gorm"
)

type AuthVerificationCode struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Scene            string         `gorm:"type:varchar(30);index;not null" json:"scene"`
	PhoneCountryCode string         `gorm:"type:varchar(8);default:'+86';index" json:"phone_country_code"`
	PhoneNumber      string         `gorm:"type:varchar(20);index;not null" json:"phone_number"`
	CodeHash         string         `gorm:"type:varchar(255);not null" json:"-"`
	ExpiresAt        time.Time      `gorm:"index" json:"expires_at"`
	UsedAt           *time.Time     `json:"used_at"`
	RequestIP        string         `gorm:"type:varchar(64)" json:"request_ip"`
	AttemptCount     int            `gorm:"default:0" json:"attempt_count"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

type SMSSendLog struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Provider         string         `gorm:"type:varchar(20);index" json:"provider"`
	PhoneCountryCode string         `gorm:"type:varchar(8);default:'+86'" json:"phone_country_code"`
	PhoneNumber      string         `gorm:"type:varchar(20);index" json:"phone_number"`
	TemplateCode     string         `gorm:"type:varchar(100)" json:"template_code"`
	BizID            string         `gorm:"type:varchar(100)" json:"biz_id"`
	Status           string         `gorm:"type:varchar(20);index" json:"status"`
	ErrorCode        string         `gorm:"type:varchar(50)" json:"error_code"`
	ErrorMessage     string         `gorm:"type:varchar(500)" json:"error_message"`
	SentAt           *time.Time     `json:"sent_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}
