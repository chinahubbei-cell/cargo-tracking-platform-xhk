package main

import (
	"fmt"
	"trackcard-server/config"
	"trackcard-server/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	config.Load()
	db, _ := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})

	var shipment models.Shipment
	db.First(&shipment, "id = ?", "260203000003")

	fmt.Printf("Shipment ID: %v\n", shipment.ID)
	fmt.Printf("DeviceID: %v\n", shipment.DeviceID)
	if shipment.DeviceID != nil {
		fmt.Printf("DeviceID Val: %s\n", *shipment.DeviceID)
	}

	fmt.Printf("UnboundDeviceID: %v\n", shipment.UnboundDeviceID)
	if shipment.UnboundDeviceID != nil {
		fmt.Printf("UnboundDeviceID Val: %s\n", *shipment.UnboundDeviceID)
	}

	fmt.Printf("DeviceBoundAt: %v\n", shipment.DeviceBoundAt)
	fmt.Printf("DeviceUnboundAt: %v\n", shipment.DeviceUnboundAt)
}
