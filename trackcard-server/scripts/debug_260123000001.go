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

	// Query Shipment
	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		fmt.Printf("Shipment not found: %v\n", err)
		return
	}

	// Query Logs
	var logs []models.ShipmentLog
	if err := db.Where("shipment_id = ?", shipmentID).Order("created_at asc").Find(&logs).Error; err != nil {
		fmt.Printf("Failed to fetch logs: %v\n", err)
	}

	// Print Details
	fmt.Printf("\n=== Shipment %s ===\n", shipment.ID)
	fmt.Printf("Status:             %s\n", shipment.Status)
	fmt.Printf("Arrived At:         %v\n", shipment.ArrivedDestAt)
	fmt.Printf("Track End At:       %v\n", shipment.TrackEndAt)

	// Check tracks after arrival
	var trackCount int64
	var laterTracks []models.DeviceTrack
	arrivalTime, _ := time.Parse("2006-01-02 15:04:05", "2026-01-27 06:54:38")

	// Assuming DeviceID is stored in shipment and matches logs
	var deviceID string
	if shipment.DeviceID != nil {
		deviceID = *shipment.DeviceID
	} else if shipment.UnboundDeviceID != nil {
		deviceID = *shipment.UnboundDeviceID
	} else {
		deviceID = "GC-a36a619f" // Fallback from log inspection
	}

	if err := db.Model(&models.DeviceTrack{}).
		Where("device_id = ? AND locate_time > ?", deviceID, arrivalTime).
		Count(&trackCount).Error; err != nil {
		fmt.Printf("Failed to count tracks: %v\n", err)
	}

	fmt.Printf("\nTrack points after %s: %d\n", arrivalTime.Format("2006-01-02 15:04:05"), trackCount)

	if trackCount > 0 {
		db.Where("device_id = ? AND locate_time > ?", deviceID, arrivalTime).
			Limit(5).
			Find(&laterTracks)
		for _, t := range laterTracks {
			fmt.Printf(" - [%s] %.6f, %.6f\n", t.LocateTime.Format("15:04:05"), t.Latitude, t.Longitude)
		}
	}
}
