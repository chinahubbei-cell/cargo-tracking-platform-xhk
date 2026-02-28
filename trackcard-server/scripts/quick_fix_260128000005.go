package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Fixing Shipment 260128000005...")

	// Explicitly update CurrentStage to 'pre_transit'
	// This matches the actual 'in_progress' stage ID found in debug logs
	if err := db.Exec("UPDATE shipments SET current_stage = 'pre_transit' WHERE id = '260128000005'").Error; err != nil {
		log.Fatal(err)
	}

	fmt.Println("Fixed: Shipment 260128000005 set to pre_transit.")
}
