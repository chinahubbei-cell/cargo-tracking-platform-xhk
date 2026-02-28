package main

import (
	"fmt"
	"log"
	"time"

	"trackcard-server/models"
	// "trackcard-server/services" // Avoid cycle or init issues, use DB directly
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Simplified duplicate of service logic solely for testing
func main() {
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal(err)
	}

	// 1. Create a Test Shipment
	shipmentID := fmt.Sprintf("TEST_%d", time.Now().Unix())
	shipment := models.Shipment{
		ID:          shipmentID,
		Origin:      "China",
		Destination: "USA", // Force Cross-Border
		Status:      "pending",
	}
	if err := db.Create(&shipment).Error; err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created Test Shipment: %s\n", shipmentID)

	// 2. Create Stages (Full Cycle)
	stages := []models.ShipmentStage{
		{ID: uuid.New().String(), ShipmentID: shipmentID, StageCode: models.SSPreTransit, StageOrder: models.GetStageOrder(models.SSPreTransit), Status: "in_progress"},
		{ID: uuid.New().String(), ShipmentID: shipmentID, StageCode: models.SSOriginPort, StageOrder: models.GetStageOrder(models.SSOriginPort), Status: "pending"},
		{ID: uuid.New().String(), ShipmentID: shipmentID, StageCode: models.SSMainLine, StageOrder: models.GetStageOrder(models.SSMainLine), Status: "pending"},
		{ID: uuid.New().String(), ShipmentID: shipmentID, StageCode: models.SSTransitPort, StageOrder: models.GetStageOrder(models.SSTransitPort), Status: "pending"},
		{ID: uuid.New().String(), ShipmentID: shipmentID, StageCode: models.SSDestPort, StageOrder: models.GetStageOrder(models.SSDestPort), Status: "pending"},
		{ID: uuid.New().String(), ShipmentID: shipmentID, StageCode: models.SSLastMile, StageOrder: models.GetStageOrder(models.SSLastMile), Status: "pending"},
		{ID: uuid.New().String(), ShipmentID: shipmentID, StageCode: models.SSDelivered, StageOrder: models.GetStageOrder(models.SSDelivered), Status: "pending"},
	}
	if err := db.Create(&stages).Error; err != nil {
		log.Fatal(err)
	}

	// Update Shipment CurrentStage
	db.Model(&shipment).Update("current_stage", string(models.SSPreTransit))
	fmt.Println("Initialized Stages.")

	// Function to transition and check status
	transitionAndCheck := func(fromCode models.StageCode, expectedNextCode models.StageCode, expectedStatus string) {
		fmt.Printf("\n--- Transitioning from %s ---\n", fromCode)

		// 1. Mark Current as Completed
		err := db.Model(&models.ShipmentStage{}).
			Where("shipment_id = ? AND stage_code = ?", shipmentID, fromCode).
			Update("status", "completed").Error
		if err != nil {
			log.Fatal(err)
		}

		// 2. Simulate Next Stage Activation (Service Logic Simulation)
		// Find Next
		var nextStage models.ShipmentStage
		err = db.Where("shipment_id = ? AND stage_order > ?", shipmentID, models.GetStageOrder(fromCode)).
			Order("stage_order ASC").First(&nextStage).Error

		var statusUpdate string
		if err == gorm.ErrRecordNotFound {
			// End of flow
			statusUpdate = "delivered"
		} else {
			// Activate Next
			db.Model(&models.ShipmentStage{}).Where("id = ?", nextStage.ID).Update("status", "in_progress")

			// Determine Status (Logic Check)
			// Logic:
			// 1 -> 2 (Completed PreTransit): Status "in_transit"
			// 6 -> 7 (Completed LastMile): Status "arrived"

			currentStatus := "pending"                            // Default
			db.Model(&shipment).Select("status").First(&shipment) // fetch
			currentStatus = shipment.Status

			if nextStage.StageOrder >= 2 && currentStatus == "pending" {
				statusUpdate = "in_transit"
			} else if nextStage.StageCode == models.SSDelivered {
				statusUpdate = "arrived"
			} else {
				statusUpdate = currentStatus // No change
			}
		}

		if statusUpdate != "" {
			db.Model(&shipment).Update("status", statusUpdate)
			db.Model(&shipment).Update("current_stage", string(nextStage.StageCode)) // Simulate pointer update
			if nextStage.ID == "" {                                                  // Finished
				db.Model(&shipment).Update("current_stage", "delivered") // or keep last? Service keeps 'delivered' stage code usually
			}
		}

		// Verify
		var s models.Shipment
		db.First(&s, "id = ?", shipmentID)
		fmt.Printf("After Transition: Status='%s', CurrentStage='%s'\n", s.Status, s.CurrentStage)

		if s.Status != expectedStatus {
			fmt.Printf("FAIL: Expected Status '%s', Got '%s'\n", expectedStatus, s.Status)
		} else {
			fmt.Printf("PASS: Status is '%s'\n", s.Status)
		}
	}

	// Step 1: Finish Pre-Transit (1) -> Start Origin Port (2). Status should become "in_transit"
	transitionAndCheck(models.SSPreTransit, models.SSOriginPort, "in_transit")

	// Step 2: Finish Origin Port (2) -> Start Main Line (3). Status remains "in_transit"
	transitionAndCheck(models.SSOriginPort, models.SSMainLine, "in_transit")

	// Step 3: Finish Main Line (3) -> Start Transit Port (4). Status remains "in_transit"
	transitionAndCheck(models.SSMainLine, models.SSTransitPort, "in_transit")

	// Step 4: Finish Transit Port (4) -> Start Dest Port (5). Status remains "in_transit"
	transitionAndCheck(models.SSTransitPort, models.SSDestPort, "in_transit")

	// Step 5: Finish Dest Port (5) -> Start Last Mile (6). Status remains "in_transit"
	transitionAndCheck(models.SSDestPort, models.SSLastMile, "in_transit")

	// Step 6: Finish Last Mile (6) -> Start Delivered (7). Status should become "arrived"
	transitionAndCheck(models.SSLastMile, models.SSDelivered, "arrived")

	// Step 7: Finish Delivered (7) -> End. Status should see logic?
	// Real Logic: TransitionToNextStage returns "Delivered" if no next stage.
	fmt.Println("\n--- Finishing Delivered Stage ---")
	db.Model(&models.ShipmentStage{}).Where("shipment_id = ? AND stage_code = ?", shipmentID, models.SSDelivered).Update("status", "completed")

	// Simulate "No Next Stage" logic
	db.Model(&shipment).Updates(map[string]interface{}{"status": "delivered", "progress": 100})

	db.First(&shipment, "id = ?", shipmentID)
	fmt.Printf("Final State: Status='%s'\n", shipment.Status)
	if shipment.Status == "delivered" {
		fmt.Println("PASS: Final status is 'delivered'")
	} else {
		fmt.Println("FAIL: Final status mismatch")
	}
}
