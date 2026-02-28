package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"trackcard-server/models"

	// "trackcard-server/services"  // Avoid service import to prevent conflicts, duplicate logic if needed
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	shipmentID := "260127000025"
	if len(os.Args) > 1 {
		shipmentID = os.Args[1]
	}

	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Fixing Shipment %s...\n", shipmentID)

	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Route: %s -> %s\n", shipment.Origin, shipment.Destination)

	// 1. Check if Transit Port should exist
	// Manual logic copy from DetermineHasTransitPort
	originLower := strings.ToLower(shipment.Origin)
	destLower := strings.ToLower(shipment.Destination)
	// Simplified check for CN -> US West
	isOriginCN := strings.Contains(originLower, "china") || strings.Contains(originLower, "cn") || strings.Contains(originLower, "shenzhen")
	isDestUSWest := (strings.Contains(destLower, "usa") || strings.Contains(destLower, "us")) && (strings.Contains(destLower, "los angeles") || strings.Contains(destLower, "long beach"))

	shouldHaveTransit := true
	if isOriginCN && isDestUSWest {
		shouldHaveTransit = false
		fmt.Println("Detected Direct Route: Removing Transit Port")
	}

	tx := db.Begin()

	// 2. Remove Transit Port if needed
	if !shouldHaveTransit {
		if err := tx.Where("shipment_id = ? AND stage_code = ?", shipmentID, models.SSTransitPort).Delete(&models.ShipmentStage{}).Error; err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}

	// 3. Reset Stages
	// Completed: PreTransit(1), OriginPort(2)
	// InProgress: MainLine(3)
	// Pending: Rest

	// Reset all to Pending first to be safe
	if err := tx.Model(&models.ShipmentStage{}).Where("shipment_id = ?", shipmentID).Updates(map[string]interface{}{"status": "pending"}).Error; err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	// Set 1 & 2 to Completed
	if err := tx.Model(&models.ShipmentStage{}).
		Where("shipment_id = ? AND stage_order <= 2", shipmentID).
		Updates(map[string]interface{}{"status": "completed"}).Error; err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	// Set 3 (MainLine) to InProgress
	if err := tx.Model(&models.ShipmentStage{}).
		Where("shipment_id = ? AND stage_code = ?", shipmentID, models.SSMainLine).
		Updates(map[string]interface{}{"status": "in_progress"}).Error; err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	// 4. Update Shipment Pointer
	updates := map[string]interface{}{
		"current_stage": string(models.SSMainLine),
		"status":        "in_transit",
	}
	if err := tx.Model(&models.Shipment{}).Where("id = ?", shipmentID).Updates(updates).Error; err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	// 5. Cleanup "Delivered" if it was completed erroneously
	// Already reset to pending above (Stage Order 7 > 2)

	tx.Commit()
	fmt.Println("Shipment Fixed: Set to Main Line (In Progress).")
}
