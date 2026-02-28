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

// EarthRadiusMeters Earth Radius in meters
const EarthRadiusMeters = 6371000

// HaversineDistance Calculate distance between two points
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

	shipmentID := "260123000001"

	// Destination Info from previous step
	destLat := 37.885094
	destLng := 112.561383
	radius := 1000.0

	// Get Device ID
	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		fmt.Printf("Shipment not found: %v\n", err)
		return
	}

	deviceID := "GC-a36a619f" // Known from previous log analysis

	fmt.Printf("Analyzing tracks for device %s to find arrival at (%.6f, %.6f) within %.0fm...\n", deviceID, destLat, destLng, radius)

	var tracks []models.DeviceTrack
	// Fetch all tracks ordered by time
	if err := db.Where("device_id = ?", deviceID).Order("locate_time asc").Find(&tracks).Error; err != nil {
		fmt.Printf("Failed to fetch tracks: %v\n", err)
		return
	}

	fmt.Printf("Total tracks found: %d\n", len(tracks))

	consecutiveInside := 0
	var firstInsideTime time.Time
	found := false

	for i, t := range tracks {
		dist := HaversineDistance(t.Latitude, t.Longitude, destLat, destLng)
		isInside := dist <= radius

		if isInside {
			consecutiveInside++
			if consecutiveInside == 1 {
				firstInsideTime = t.LocateTime
			}
			if consecutiveInside >= 3 {
				fmt.Printf("Check: Point %d [%s] dist=%.0fm (Inside) - Consecutive: %d -> TRIGGERED!\n", i, t.LocateTime.Format("2006-01-02 15:04:05"), dist, consecutiveInside)
				found = true
				break
			} else {
				// fmt.Printf("Check: Point %d [%s] dist=%.0fm (Inside) - Consecutive: %d\n", i, t.LocateTime.Format("2006-01-02 15:04:05"), dist, consecutiveInside)
			}
		} else {
			consecutiveInside = 0
		}
	}

	if found {
		fmt.Printf("\n>>> TRUE ARRIVAL TIME FOUND: %s <<<\n", firstInsideTime.Format("2006-01-02 15:04:05"))

		// Apply Update
		updates := map[string]interface{}{
			"status":            "delivered",
			"progress":          100,
			"arrived_dest_at":   firstInsideTime,
			"track_end_at":      firstInsideTime,
			"status_updated_at": firstInsideTime,
			"current_milestone": "arrived",
			"ata":               firstInsideTime,
		}

		if err := db.Model(&shipment).Updates(updates).Error; err != nil {
			fmt.Printf("Failed to update shipment: %v\n", err)
		} else {
			fmt.Println("Shipment updated successfully with True Arrival Time.")

			// Log it
			logEntry := models.ShipmentLog{
				ShipmentID:  shipmentID,
				Action:      "status_changed",
				Field:       "status",
				OldValue:    "in_transit",
				NewValue:    "delivered",
				Description: fmt.Sprintf("系统重新计算：根据真实轨迹判定到达 (三点连续入围栏)，截断于 %s", firstInsideTime.Format("15:04:05")),
				OperatorID:  "system_recalc",
				CreatedAt:   time.Now(),
			}
			db.Create(&logEntry)
		}
	} else {
		fmt.Println("No valid arrival trigger (3 consecutive points inside fence) found in history.")
	}
}
