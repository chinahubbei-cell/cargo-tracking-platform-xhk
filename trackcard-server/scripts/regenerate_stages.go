package main

import (
	"fmt"
	"log"

	"trackcard-server/models"
	"trackcard-server/services"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("./trackcard.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	shipmentID := "260131000001"

	// 初始化服务
	services.InitShipmentStageService(db)
	svc := services.GetShipmentStageService()

	// 检查运单信息
	var shipment models.Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		log.Fatalf("Failed to find shipment: %v", err)
	}
	fmt.Printf("Shipment: %s\n", shipmentID)
	fmt.Printf("  Transport Type: %s\n", shipment.TransportType)
	fmt.Printf("  Origin: %s\n", shipment.OriginAddress)
	fmt.Printf("  Destination: %s\n", shipment.DestAddress)

	// 删除现有 stages
	db.Where("shipment_id = ?", shipmentID).Delete(&models.ShipmentStage{})
	fmt.Println("Deleted existing stages")

	// 重新生成
	if err := svc.CreateStagesForShipment(shipmentID, "", ""); err != nil {
		log.Fatalf("Failed to regenerate stages: %v", err)
	}

	fmt.Println("Successfully regenerated stages!")

	// 查询新生成的 stages
	var stages []models.ShipmentStage
	db.Where("shipment_id = ?", shipmentID).Order("stage_order").Find(&stages)
	fmt.Printf("\nNew stages (%d):\n", len(stages))
	for _, s := range stages {
		fmt.Printf("  %d. %s - %s\n", s.StageOrder, s.StageCode, s.Carrier)
	}
}
