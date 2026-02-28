// 补录脚本：为缺少围栏触发日志的运单补录日志
// 用法: go run scripts/backfill_geofence_logs.go
package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 简化的模型定义
type Shipment struct {
	ID            string     `gorm:"primaryKey"`
	Status        string     `gorm:"column:status"`
	LeftOriginAt  *time.Time `gorm:"column:left_origin_at"`
	ArrivedDestAt *time.Time `gorm:"column:arrived_dest_at"`
	ATD           *time.Time `gorm:"column:atd"`
	ATA           *time.Time `gorm:"column:ata"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

type ShipmentLog struct {
	ID          uint      `gorm:"primaryKey"`
	ShipmentID  string    `gorm:"column:shipment_id"`
	Action      string    `gorm:"column:action"`
	Field       string    `gorm:"column:field"`
	OldValue    string    `gorm:"column:old_value"`
	NewValue    string    `gorm:"column:new_value"`
	Description string    `gorm:"column:description"`
	OperatorID  string    `gorm:"column:operator_id"`
	OperatorIP  string    `gorm:"column:operator_ip"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	fmt.Println("🔧 围栏触发日志补录工具")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 查找缺少围栏触发日志的运单
	var shipments []Shipment
	subQuery := db.Model(&ShipmentLog{}).
		Select("shipment_id").
		Where("action = 'geofence_trigger'")

	err = db.Where("status IN ('in_transit', 'delivered') AND id NOT IN (?)", subQuery).
		Find(&shipments).Error
	if err != nil {
		log.Fatalf("❌ 查询失败: %v", err)
	}

	if len(shipments) == 0 {
		fmt.Println("✅ 没有需要补录的运单")
		return
	}

	fmt.Printf("📋 发现 %d 个运单需要补录\n\n", len(shipments))

	backfilledCount := 0
	for _, shipment := range shipments {
		fmt.Printf("处理运单: %s (状态: %s)\n", shipment.ID, shipment.Status)

		// 为 in_transit 和 delivered 状态的运单补录"离开发货地"日志
		if shipment.Status == "in_transit" || shipment.Status == "delivered" {
			departTime := shipment.LeftOriginAt
			if departTime == nil && shipment.ATD != nil {
				departTime = shipment.ATD
			}
			if departTime == nil {
				now := shipment.UpdatedAt
				departTime = &now
			}

			logEntry := ShipmentLog{
				ShipmentID:  shipment.ID,
				Action:      "geofence_trigger",
				Field:       "status",
				OldValue:    "pending",
				NewValue:    "in_transit",
				Description: "[补录] 设备自动触发离开发货地",
				OperatorID:  "system",
				OperatorIP:  "geofence_backfill",
				CreatedAt:   *departTime,
			}
			if err := db.Create(&logEntry).Error; err == nil {
				backfilledCount++
				fmt.Printf("  ✅ 补录离开发货地日志 (时间: %s)\n", departTime.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  ❌ 补录失败: %v\n", err)
			}
		}

		// 为 delivered 状态的运单补录"到达目的地"日志
		if shipment.Status == "delivered" {
			arriveTime := shipment.ArrivedDestAt
			if arriveTime == nil && shipment.ATA != nil {
				arriveTime = shipment.ATA
			}
			if arriveTime == nil {
				now := shipment.UpdatedAt
				arriveTime = &now
			}

			logEntry := ShipmentLog{
				ShipmentID:  shipment.ID,
				Action:      "geofence_trigger",
				Field:       "status",
				OldValue:    "in_transit",
				NewValue:    "delivered",
				Description: "[补录] 设备自动触发到达目的地",
				OperatorID:  "system",
				OperatorIP:  "geofence_backfill",
				CreatedAt:   *arriveTime,
			}
			if err := db.Create(&logEntry).Error; err == nil {
				backfilledCount++
				fmt.Printf("  ✅ 补录到达目的地日志 (时间: %s)\n", arriveTime.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  ❌ 补录失败: %v\n", err)
			}
		}
		fmt.Println()
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ 补录完成，共补录 %d 条日志\n", backfilledCount)
}
