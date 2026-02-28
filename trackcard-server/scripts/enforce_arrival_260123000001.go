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

	// 2026/1/27 14:54:38 CST is 06:54:38 UTC
	arrivalTime, _ := time.Parse("2006-01-02 15:04:05", "2026-01-27 06:54:38")

	updates := map[string]interface{}{
		"status":            "delivered",
		"progress":          100,
		"arrived_dest_at":   arrivalTime,
		"track_end_at":      arrivalTime,
		"status_updated_at": arrivalTime,
		"current_milestone": "arrived",
		"ata":               arrivalTime,
		// Explicitly NOT setting 'current_stage' blindly here to avoid conflicts
		// if 'delivered' isn't a valid stage code in some lookup tables,
		// but typically status=delivered implies completion.
	}

	if err := db.Model(&shipment).Updates(updates).Error; err != nil {
		fmt.Printf("Failed to update shipment: %v\n", err)
		return
	}

	// Add log entry
	logEntry := models.ShipmentLog{
		ShipmentID:  shipmentID,
		Action:      "status_changed",
		Field:       "status",
		OldValue:    shipment.Status,
		NewValue:    "delivered",
		Description: "人工确认：恢复由于过境被系统回退的到达状态，截断轨迹于 14:54:38",
		OperatorID:  "admin_fix",
		CreatedAt:   time.Now(),
	}
	db.Create(&logEntry)

	fmt.Printf("Successfully enforced arrival for %s at %v\n", shipmentID, arrivalTime)
}
