package main

import (
	"fmt"
	"math"
	"os"
	"time"
	"trackcard-server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const EarthRadiusMeters = 6371000

func HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusMeters * c
}

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
		fmt.Printf("Shipment not found\n")
		return
	}

	destLat := *shipment.DestLat
	destLng := *shipment.DestLng

	// Determine Device ID
	var deviceID string
	if shipment.DeviceID != nil && *shipment.DeviceID != "" {
		deviceID = *shipment.DeviceID
	} else if shipment.UnboundDeviceID != nil && *shipment.UnboundDeviceID != "" {
		deviceID = *shipment.UnboundDeviceID
	} else {
		// Log fallback
		deviceID = "GC-22a57303" // From previous logs
	}

	fmt.Printf("Shipment: %s | Device: %s | Dest: (%.6f, %.6f)\n", shipmentID, deviceID, destLat, destLng)

	var tracks []models.DeviceTrack
	db.Where("device_id = ?", deviceID).Order("locate_time asc").Find(&tracks) // Find ALL to be sure

	minDist := 999999.0
	var minTime time.Time

	for _, t := range tracks {
		dist := HaversineDistance(t.Latitude, t.Longitude, destLat, destLng)
		if dist < minDist {
			minDist = dist
			minTime = t.LocateTime
		}
	}

	fmt.Printf("Minimum Distance Reached: %.2fm at %s\n", minDist, minTime.Format("2006-01-02 15:04:05"))
}
