package main

import (
	"fmt"
	"log"
	"time"

	"trackcard-server/models"
	"trackcard-server/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. Initialize DB
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Initialize Services
	services.InitShipmentLog(db)
	services.InitDeviceBinding(db)
	services.InitShipmentStageService(db)
	services.InitGeofence(db)
	services.Geofence.SetDebugMode(true)

	shipmentID := "260205000001"
	deviceID := "TEST-DEV-001"

	// Coordinates for Ningbo Port (CNNBG) - typically around 29.8, 121.5
	// I will read them from CLI args or just hardcode if I know them.
	// The query result from previous step will give me exact values.
	// For now, I'll use placeholders and update file if needed, or query dynamically.

	// Query Port Coordinates
	// var port models.Port

	lat := 29.916667
	lng := 121.666667

	// Setup Test Data
	log.Println("Setting up test data...")

	// Create/Update Device
	db.Exec("INSERT OR REPLACE INTO devices (id, name, status, latitude, longitude, last_update) VALUES (?, ?, ?, ?, ?, ?)",
		deviceID, "Test Device", "online", lat, lng, time.Now())

	// Update Shipment
	db.Exec("UPDATE shipments SET device_id=?, status='in_transit', auto_status_enabled=1, origin_radius=1000, dest_radius=1000 WHERE id=?",
		deviceID, shipmentID)

	// Ensure stages exist
	// (Assuming they exist from previous `curl` output)

	// 3. Trigger Geofence Check
	// Simulate 3 consecutive points to satisfy the count requirement
	log.Println("Simulating 3 consecutive points in Geofence...")

	// Insert 3 track points
	for i := 0; i < 3; i++ {
		db.Create(&models.DeviceTrack{
			DeviceID:   deviceID,
			Latitude:   lat,
			Longitude:  lng,
			LocateTime: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	log.Printf("Triggering CheckAndUpdateStatus for device %s at (%.6f, %.6f)...", deviceID, lat, lng)
	services.Geofence.CheckAndUpdateStatus(deviceID, lat, lng)

	// 4. Verify Result
	var stages []models.ShipmentStage
	db.Where("shipment_id = ?", shipmentID).Order("stage_order ASC").Find(&stages)

	fmt.Printf("\n=== FULL STAGE STATUS ===\n")
	for _, s := range stages {
		fmt.Printf("[%d] %s (%s): %s\n", s.StageOrder, s.StageCode, models.GetStageName(s.StageCode), s.Status)
	}

	// Check logs
	var logs []models.ShipmentLog
	db.Where("shipment_id = ?", shipmentID).Order("created_at desc").Limit(5).Find(&logs)
	fmt.Printf("\n=== LOGS ===\n")
	for _, l := range logs {
		fmt.Printf("[%s] %s -> %s : %s\n", l.Action, l.OldValue, l.NewValue, l.Description)
	}
}
