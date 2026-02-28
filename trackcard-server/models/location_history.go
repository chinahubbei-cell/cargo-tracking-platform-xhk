package models

import (
	"time"
)

type LocationHistory struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID    string    `gorm:"type:varchar(50);not null;index;column:device_id" json:"device_id"`
	ShipmentID  *string   `gorm:"type:varchar(50);column:shipment_id" json:"shipment_id"`
	Latitude    float64   `gorm:"type:decimal(10,6);not null" json:"latitude"`
	Longitude   float64   `gorm:"type:decimal(10,6);not null" json:"longitude"`
	Speed       float64   `gorm:"default:0" json:"speed"`
	Direction   float64   `gorm:"default:0" json:"direction"`
	Temperature *float64  `gorm:"type:decimal(5,2)" json:"temperature"`
	Humidity    *float64  `gorm:"type:decimal(5,2)" json:"humidity"`
	LocateType  *int      `gorm:"column:locate_type" json:"locate_type"`
	Timestamp   time.Time `gorm:"default:CURRENT_TIMESTAMP;index" json:"timestamp"`
}

func (LocationHistory) TableName() string {
	return "location_history"
}

type TrackPoint struct {
	Device      string   `json:"device"`
	Speed       float64  `json:"speed"`
	Direction   float64  `json:"direction"`
	LocateTime  int64    `json:"locateTime"`
	Longitude   float64  `json:"longitude"`
	Latitude    float64  `json:"latitude"`
	LocateType  int      `json:"locateType"`
	RunStatus   int      `json:"runStatus"`
	Temperature *float64 `json:"temperature"`
	Humidity    *float64 `json:"humidity"`
}
