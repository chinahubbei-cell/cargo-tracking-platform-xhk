package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"trackcard-server/models"
	"trackcard-server/services"
)

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	shipmentID := "260128000004"
	fmt.Printf("=== 重新生成运单 %s 的环节 ===\n", shipmentID)

	// 1. 删除现有环节
	fmt.Println("步骤1: 删除现有环节数据...")
	result := db.Where("shipment_id = ?", shipmentID).Delete(&models.ShipmentStage{})
	if result.Error != nil {
		log.Fatalf("删除环节失败: %v", result.Error)
	}
	fmt.Printf("✅ 已删除 %d 条旧环节记录\n", result.RowsAffected)

	// 2. 初始化服务
	fmt.Println("步骤2: 初始化服务...")
	services.InitShipmentStageService(db)
	stageService := services.GetShipmentStageService()
	if stageService == nil {
		log.Fatal("环节服务初始化失败")
	}

	// 3. 重新创建环节
	fmt.Println("步骤3: 创建新环节...")
	if err := stageService.CreateStagesForShipment(shipmentID, "", ""); err != nil {
		log.Fatalf("创建环节失败: %v", err)
	}

	// 4. 验证结果
	fmt.Println("步骤4: 验证新环节...")
	var stages []models.ShipmentStage
	if err := db.Where("shipment_id = ?", shipmentID).Order("stage_order ASC").Find(&stages).Error; err != nil {
		log.Fatalf("查询环节失败: %v", err)
	}

	fmt.Printf("\n✅ 成功！运单 %s 现在有 %d 个环节:\n\n", shipmentID, len(stages))
	for _, stage := range stages {
		statusIcon := "⏳"
		if stage.Status == "completed" {
			statusIcon = "✅"
		} else if stage.Status == "in_progress" {
			statusIcon = "🔄"
		}
		fmt.Printf("  [%d] %s %s (order: %d)\n", stage.StageOrder, statusIcon, stage.StageCode, stage.StageOrder)
	}
	fmt.Println()
}
