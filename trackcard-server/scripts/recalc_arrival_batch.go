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

	targetShipments := []string{"260123000002", "260123000003"}

	for _, shipmentID := range targetShipments {
		fmt.Printf("\n--------------------------------------------------\n")
		fmt.Printf("Processing Shipment: %s\n", shipmentID)

		var shipment models.Shipment
		if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
			fmt.Printf("Shipment not found: %v\n", err)
			continue
		}

		if shipment.DestLat == nil || shipment.DestLng == nil {
			fmt.Println("Skipping: Destination coordinates missing.")
			continue
		}

		destLat := *shipment.DestLat
		destLng := *shipment.DestLng
		radius := float64(shipment.DestRadius)
		if radius < 100 {
			radius = 1000 // Default fallback
		}

		// Determine Device ID
		var deviceID string
		if shipment.DeviceID != nil && *shipment.DeviceID != "" {
			deviceID = *shipment.DeviceID
		} else if shipment.UnboundDeviceID != nil && *shipment.UnboundDeviceID != "" {
			deviceID = *shipment.UnboundDeviceID
		} else {
			// Try to find from logs if not in columns
			var log models.ShipmentLog
			db.Where("shipment_id = ? AND action IN ('device_bound', 'device_unbound')", shipmentID).First(&log)
			// This is a weak fallback, assuming description contains it or log structure allows extraction,
			// but usually UnboundDeviceID is reliable for finished shipments.
			if shipment.UnboundDeviceID == nil {
				fmt.Println("Skipping: Device ID not found (unbound device id is nil).")
				continue
			}
		}

		fmt.Printf("Device: %s | Dest: (%.6f, %.6f) | Radius: %.0fm\n", deviceID, destLat, destLng, radius)

		var tracks []models.DeviceTrack
		if err := db.Where("device_id = ?", deviceID).Order("locate_time asc").Find(&tracks).Error; err != nil {
			fmt.Printf("Failed to fetch tracks: %v\n", err)
			continue
		}
		fmt.Printf("Total tracks: %d\n", len(tracks))

		consecutiveInside := 0
		var firstInsideTime time.Time
		found := false

		// Logic: Find FIRST occurrence of 3 consecutive points inside fence
		for _, t := range tracks {
			dist := HaversineDistance(t.Latitude, t.Longitude, destLat, destLng)
			isInside := dist <= radius

			if isInside {
				consecutiveInside++
				if consecutiveInside == 1 {
					firstInsideTime = t.LocateTime
				}
				if consecutiveInside >= 3 {
					found = true
					fmt.Printf(">>> Trigger Found! Time: %s | Dist: %.0fm\n", firstInsideTime.Format("2006-01-02 15:04:05"), dist)
					break
				}
			} else {
				consecutiveInside = 0
			}
		}

		if found {
			// Update Shipment
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
				fmt.Printf("SUCCESS: Shipment %s updated to Arrival Time: %s\n", shipmentID, firstInsideTime.Format("2006-01-02 15:04:05"))

				// Log
				logEntry := models.ShipmentLog{
					ShipmentID:  shipmentID,
					Action:      "status_changed",
					Field:       "status",
					OldValue:    shipment.Status,
					NewValue:    "delivered",
					Description: fmt.Sprintf("系统重新计算：根据真实轨迹判定到达 (三点连续入围栏)，截断于 %s", firstInsideTime.Format("15:04:05")),
					OperatorID:  "system_recalc",
					CreatedAt:   time.Now(),
				}
				db.Create(&logEntry)
			}
		} else {
			fmt.Println("NO TRIGGER FOUND: Tracks never satisfied 3-consecutive-points-inside condition.")
		}
	}
}
