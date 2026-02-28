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

	shipmentID := "260123000003"

	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		fmt.Printf("Shipment not found: %v\n", err)
		return
	}

	// 2026/1/27 14:54:38 CST is 06:54:38 UTC
	// Go's time.Parse usually assumes UTC if no timezone info, but let's be explicit about the requested CST time
	// Actually, just parsing as 2026-01-27 14:54:38 in +0800 context directly:
	loc, _ := time.LoadLocation("Asia/Shanghai")
	arrivalTime, err := time.ParseInLocation("2006/1/2 15:04:05", "2026/1/27 14:54:38", loc)
	if err != nil {
		// Fallback to manual offset if location load fails (common in minimal containers)
		t, _ := time.Parse("2006/1/2 15:04:05", "2026/1/27 14:54:38")
		arrivalTime = t.Add(-8 * time.Hour) // Adjust to UTC
	} else {
		arrivalTime = arrivalTime.UTC()
	}

	updates := map[string]interface{}{
		"status":            "delivered",
		"progress":          100,
		"arrived_dest_at":   arrivalTime,
		"track_end_at":      arrivalTime,
		"status_updated_at": arrivalTime,
		"current_milestone": "arrived",
		"ata":               arrivalTime,
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
		Description: "人工确认：由于真实轨迹未触发围栏，强制设置到达时间为 2026/1/27 14:54:38",
		OperatorID:  "admin_fix",
		CreatedAt:   time.Now(),
	}
	db.Create(&logEntry)

	fmt.Printf("Successfully enforced arrival for %s at %v (UTC)\n", shipmentID, arrivalTime)
}
