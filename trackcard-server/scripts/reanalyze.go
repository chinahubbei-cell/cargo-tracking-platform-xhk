package main

import (
	"log"
	"time"

	"trackcard-server/config"
	"trackcard-server/models"
	"trackcard-server/services"
)

func main() {
	cfg := config.Load()
	db, err := config.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}

	shipmentID := "260128000003"
	deviceID := "GC-83fd51e3"

	// Find the device
	var device models.Device
	if err := db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		log.Fatalf("device not found: %v", err)
	}

	stopService := services.NewDeviceStopService(db)

	startTime := time.Date(2026, 3, 4, 12, 0, 0, 0, time.Local)
	endTime := time.Now()

	log.Printf("Reanalyzing stops for shipment %s, device %s", shipmentID, deviceID)
	err = stopService.AnalyzeDeviceTracksAndCreateStops(deviceID, *device.ExternalDeviceID, shipmentID, startTime, endTime)
	if err != nil {
		log.Fatalf("failed to analyze: %v", err)
	}

	log.Println("Done")
}
