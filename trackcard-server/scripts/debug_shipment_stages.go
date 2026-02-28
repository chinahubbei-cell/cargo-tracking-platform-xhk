package main

import (
	"fmt"
	"log"
	"os"

	"trackcard-server/models"

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

	var stages []models.ShipmentStage
	if err := db.Where("shipment_id = ?", shipmentID).Order("stage_order ASC").Find(&stages).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Stages for Shipment %s:\n", shipmentID)
	for _, s := range stages {
		fmt.Printf("Order: %d | Code: %s | Status: %s | ID: %s\n", s.StageOrder, s.StageCode, s.Status, s.ID)
	}

	var shipment models.Shipment
	db.First(&shipment, "id = ?", shipmentID)
	fmt.Printf("CurrentStage Pointer: %s\n", shipment.CurrentStage)
}
