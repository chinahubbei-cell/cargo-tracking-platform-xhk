package main

import (
	"fmt"
	"os"
	"trackcard-server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. Connect to DB
	dbPath := "trackcard.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	shipmentID := "260123000003"

	// 2. Query Shipment
	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		fmt.Printf("Shipment not found: %v\n", err)
		return
	}

	// 3. Query Logs
	var logs []models.ShipmentLog
	if err := db.Where("shipment_id = ?", shipmentID).Order("created_at asc").Find(&logs).Error; err != nil {
		fmt.Printf("Failed to fetch logs: %v\n", err)
	}

	// 4. Print Details
	fmt.Printf("\n=== Shipment %s ===\n", shipment.ID)
	fmt.Printf("Status:             %s\n", shipment.Status)
	fmt.Printf("Current Stage:      %s\n", shipment.CurrentStage)
	fmt.Printf("AutoStatusEnabled:  %v\n", shipment.AutoStatusEnabled)
	fmt.Printf("Device ID:          %v\n", shipment.DeviceID)
	if shipment.DeviceID != nil {
		fmt.Printf("Device ID (Value):  %s\n", *shipment.DeviceID)
	}
	fmt.Printf("Track End At:       %v\n", shipment.TrackEndAt)

	fmt.Println("\n--- Shipment Logs (Ordered) ---")
	for _, log := range logs {
		fmt.Printf("[%s] %s | %s | Field: %s | Old: %s -> New: %s\n",
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.Action,
			log.Description,
			log.Field,
			log.OldValue,
			log.NewValue)
	}
}
