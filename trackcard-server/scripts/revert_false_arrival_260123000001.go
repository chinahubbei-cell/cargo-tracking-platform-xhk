package main

import (
	"fmt"
	"os"
	"time"
	"trackcard-server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	dbPath := "trackcard.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	shipmentID := "260123000001"

	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		fmt.Printf("Shipment not found: %v\n", err)
		return
	}

	fmt.Printf("Current Status: %s\n", shipment.Status)

	// Revert status and clear arrival fields
	updates := map[string]interface{}{
		"status":            "in_transit",
		"current_stage":     "in_transit",
		"track_end_at":      nil,
		"arrived_dest_at":   nil,
		"ata":               nil,
		"current_milestone": "in_transit",
		"progress":          99, // Reset to high progress but not 100
	}

	if err := db.Model(&shipment).Updates(updates).Error; err != nil {
		fmt.Printf("Failed to update shipment: %v\n", err)
		return
	}

	// Add a log entry for the manual reversion
	logEntry := models.ShipmentLog{
		ShipmentID:  shipmentID,
		Action:      "status_changed",
		Field:       "status",
		OldValue:    "delivered",
		NewValue:    "in_transit",
		Description: "系统自动回退：检测到到达后仍有大量轨迹，判定为过境误触围栏，恢复运输状态",
		OperatorID:  "system_fix",
		CreatedAt:   time.Now(),
	}
	db.Create(&logEntry)

	fmt.Printf("Successfully reverted shipment %s to in_transit and cleared TrackEndAt.\n", shipmentID)
}
