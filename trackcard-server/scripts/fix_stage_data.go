package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"trackcard-server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dbPath := "trackcard.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting stage data fix...")

	// 1. Get all shipments
	var shipments []models.Shipment
	if err := db.Find(&shipments).Error; err != nil {
		log.Fatal(err)
	}

	for _, s := range shipments {
		fmt.Printf("Processing Shipment %s...\n", s.ID)

		var stages []models.ShipmentStage
		if err := db.Where("shipment_id = ?", s.ID).Find(&stages).Error; err != nil {
			log.Printf("  Error getting stages: %v\n", err)
			continue
		}

		// Define newCurrent early to avoid goto jump error
		var newCurrent models.StageCode

		// Mapping Logic
		// Key: NewCode, Value: Best Stage to keep
		stagesToKeep := make(map[models.StageCode]*models.ShipmentStage)

		// New Code Order Map
		codeOrder := map[models.StageCode]int{
			models.SSPreTransit:  1,
			models.SSOriginPort:  2,
			models.SSMainLine:    3,
			models.SSTransitPort: 4,
			models.SSDestPort:    5,
			models.SSLastMile:    6,
			models.SSDelivered:   7,
		}

		for i := range stages {
			stage := &stages[i]
			var newCode models.StageCode

			switch string(stage.StageCode) {
			case "pickup", "first_mile", "pre_transit":
				newCode = models.SSPreTransit
			case "origin_arrival", "origin_departure", "origin_port":
				newCode = models.SSOriginPort
			case "main_carriage", "main_line":
				newCode = models.SSMainLine
			case "transit_arrival", "transit_departure", "transit_port":
				newCode = models.SSTransitPort
			case "dest_arrival", "dest_departure", "dest_port":
				newCode = models.SSDestPort
			case "delivery", "last_mile":
				newCode = models.SSLastMile
			case "delivered", "sign_off":
				newCode = models.SSDelivered
			default:
				// Keep explicit unknown codes as is, or map to closest?
				// Assuming current data is one of above.
				newCode = stage.StageCode
			}

			// Duplicate Resolution Strategy
			if existing, ok := stagesToKeep[newCode]; ok {
				// If we already have a candidate for this newCode
				// Keep the one that is "more complete" or "latest"
				if stage.Status == models.StageStatusCompleted && existing.Status != models.StageStatusCompleted {
					// Replace with completed one
					// Note: We need to MARK valid/invalid for DB updates
					stagesToKeep[newCode] = stage
				} else if stage.Status == models.StageStatusInProgress {
					stagesToKeep[newCode] = stage
				} else {
					// Keep existing
				}
				// Mark 'stage' as to-delete?
				// We will delete ALL stages for this shipment first, then re-insert correct ones?
				// Safer: Update in place if IDs match, else delete.
			} else {
				stagesToKeep[newCode] = stage
			}
		}

		// Apply Changes
		// Delete all existing stages for this shipment (Transaction safe)
		tx := db.Begin()
		if err := tx.Where("shipment_id = ?", s.ID).Delete(&models.ShipmentStage{}).Error; err != nil {
			tx.Rollback()
			log.Printf("  Error deleting stages: %v\n", err)
			continue
		}

		// Re-insert valid ones with updated Code and Order
		for code, stage := range stagesToKeep {
			order := codeOrder[code]
			if order == 0 {
				order = 99 // Fallback
			}

			// Construct new stage record (preserving ID if possible, but GORM create might gen new ID if we deleted)
			// Wait, if we delete, we lose ID references in Logs?
			// Ideally we UPDATE.
			// But duplicate removal requires deletion.

			// Let's copy data to a new struct to insert
			newStage := models.ShipmentStage{
				ShipmentID: s.ID,
				StageCode:  code,
				StageOrder: order,
				Status:     stage.Status,
				// Copy other fields
				PartnerID:    stage.PartnerID,
				PartnerName:  stage.PartnerName,
				VehiclePlate: stage.VehiclePlate,
				VesselName:   stage.VesselName,
				VoyageNo:     stage.VoyageNo,
				Carrier:      stage.Carrier,
				PlannedStart: stage.PlannedStart,
				PlannedEnd:   stage.PlannedEnd,
				ActualStart:  stage.ActualStart,
				ActualEnd:    stage.ActualEnd,
				Cost:         stage.Cost,
				Currency:     stage.Currency,
				TriggerType:  stage.TriggerType,
				TriggerNote:  stage.TriggerNote,
				ExtraData:    stage.ExtraData,
				CreatedAt:    stage.CreatedAt,
				UpdatedAt:    time.Now(),
			}
			newStage.ID = stage.ID // Try to keep ID

			if err := tx.Create(&newStage).Error; err != nil {
				// If ID conflict (shouldn't be, we deleted), handle it?
				// We deleted rows, so ID reuse depends on DB engine. SQLite allows logic reuse if TEXT PK?
				// ShipmentStage ID is UUID format usually.
				// If conflict, try clearing ID to generate new one
				newStage.ID = ""
				if err2 := tx.Create(&newStage).Error; err2 != nil {
					tx.Rollback()
					log.Printf("  Error re-creating stage %s: %v\n", code, err2)
					goto NextShipment
				}
			}
		}

		// Update Shipment CurrentStage
		// Map old current_stage to new code
		switch s.CurrentStage {
		case "pickup", "first_mile", "pre_transit":
			newCurrent = models.SSPreTransit
		case "origin_arrival", "origin_departure", "origin_port":
			newCurrent = models.SSOriginPort
		case "main_carriage", "main_line":
			newCurrent = models.SSMainLine
		case "transit_arrival", "transit_departure", "transit_port":
			newCurrent = models.SSTransitPort
		case "dest_arrival", "dest_departure", "dest_port":
			newCurrent = models.SSDestPort
		case "delivery", "last_mile":
			newCurrent = models.SSLastMile
		case "delivered", "sign_off":
			newCurrent = models.SSDelivered
		default:
			newCurrent = models.StageCode(s.CurrentStage)
		}

		if err := tx.Model(&models.Shipment{}).Where("id = ?", s.ID).Update("current_stage", newCurrent).Error; err != nil {
			tx.Rollback()
			log.Printf("  Error updating shipment current_stage: %v\n", err)
			continue
		}

		tx.Commit()
		fmt.Printf("  Fixed Shipment %s\n", s.ID)

	NextShipment:
	}
	fmt.Println("Done.")
}
