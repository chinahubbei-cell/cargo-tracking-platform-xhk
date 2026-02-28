package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Define a minimal Shipment struct for the update
type Shipment struct {
	ID           string
	CurrentStage string
}

func main() {
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting system-wide fix for legacy 'first_mile' pointers...")

	// Find count of affected shipments
	var count int64
	db.Model(&Shipment{}).Where("current_stage = ?", "first_mile").Count(&count)
	fmt.Printf("Found %d shipments with legacy 'first_mile' stage.\n", count)

	if count > 0 {
		// Perform Update
		result := db.Model(&Shipment{}).Where("current_stage = ?", "first_mile").Update("current_stage", "pre_transit")
		if result.Error != nil {
			log.Fatal("Error updating records:", result.Error)
		}
		fmt.Printf("Successfully updated %d records to 'pre_transit'.\n", result.RowsAffected)
	} else {
		fmt.Println("No records needed fixing.")
	}
}
