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

	var shipments []models.Shipment
	// Find delivered shipments with missing TrackEndAt
	if err := db.Where("status = ? AND track_end_at IS NULL", "delivered").Find(&shipments).Error; err != nil {
		fmt.Printf("Failed to query shipments: %v\n", err)
		return
	}

	fmt.Printf("Found %d delivered shipments with missing TrackEndAt\n", len(shipments))

	for _, s := range shipments {
		fmt.Printf("Fixing shipment %s...\n", s.ID)

		var endTime *time.Time
		if s.ArrivedDestAt != nil {
			endTime = s.ArrivedDestAt
		} else if s.StatusUpdatedAt != nil {
			endTime = s.StatusUpdatedAt
		} else {
			// Fallback to now if no time available (shouldn't happen for delivered)
			now := time.Now()
			endTime = &now
		}

		if err := db.Model(&s).Update("track_end_at", endTime).Error; err != nil {
			fmt.Printf("  Error fixing shipment %s: %v\n", s.ID, err)
		} else {
			fmt.Printf("  Success! TrackEndAt set to %v\n", endTime)
		}
	}
}
