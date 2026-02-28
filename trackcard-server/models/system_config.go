package models

import (
	"time"
)

type SystemConfig struct {
	Key       string    `gorm:"primaryKey;type:varchar(100)" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (SystemConfig) TableName() string {
	return "system_config"
}
