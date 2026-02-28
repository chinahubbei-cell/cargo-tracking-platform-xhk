package main

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trackcard-server/config"
	"trackcard-server/models"
	"trackcard-server/services"
)

func main() {
	// 初始化数据库
	dbPath := "trackcard.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// 尝试在上级目录找
		dbPath = "../trackcard-server/trackcard.db"
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			// 尝试绝对路径
			dbPath = "/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk/trackcard-server/trackcard.db"
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				log.Fatal("找不到数据库文件")
			}
		}
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 加载配置并初始化服务
	config.Load()
	services.InitKuaihuoyun()
	services.InitScheduler(db)

	if !services.Kuaihuoyun.IsConfigured() {
		log.Fatal("快货运API未配置，无法同步数据")
	}

	// 获取所有运单（包括历史运单）
	var shipments []models.Shipment
	if err := db.Find(&shipments).Error; err != nil {
		log.Fatalf("查询运单失败: %v", err)
	}

	log.Printf("开始同步 %d 个运单的轨迹数据...", len(shipments))

	successCount := 0
	skipCount := 0

	for _, shipment := range shipments {
		// 确定要同步的设备ID
		deviceID := ""
		if shipment.DeviceID != nil && *shipment.DeviceID != "" {
			deviceID = *shipment.DeviceID
		} else if shipment.UnboundDeviceID != nil && *shipment.UnboundDeviceID != "" {
			deviceID = *shipment.UnboundDeviceID
		}

		if deviceID == "" {
			log.Printf("运单 %s 无关联设备，跳过", shipment.ID)
			skipCount++
			continue
		}

		// 确定查询时间范围
		var startTime, endTime time.Time

		// 开始时间：绑定时间 > 发车时间 > 创建时间
		if shipment.DeviceBoundAt != nil {
			startTime = *shipment.DeviceBoundAt
		} else if shipment.LeftOriginAt != nil {
			startTime = *shipment.LeftOriginAt
		} else {
			startTime = shipment.CreatedAt
		}

		// 结束时间：解绑时间 > 到达时间 > 当前时间
		if shipment.DeviceUnboundAt != nil {
			endTime = *shipment.DeviceUnboundAt
		} else if shipment.ArrivedDestAt != nil {
			endTime = *shipment.ArrivedDestAt
		} else {
			endTime = time.Now()
		}

		// 确保时间有效
		if startTime.IsZero() {
			startTime = time.Now().Add(-24 * time.Hour)
		}
		if endTime.IsZero() {
			endTime = time.Now()
		}
		if endTime.Before(startTime) {
			endTime = startTime.Add(1 * time.Hour)
		}

		log.Printf("正在同步运单 %s (设备: %s, 时间: %s - %s)",
			shipment.ID, deviceID, startTime.Format(time.DateTime), endTime.Format(time.DateTime))

		// 调用同步服务（切分时间段以避免单次请求过大，如果需要的话，但这里简单起见直接调）
		// SyncDeviceTrack是我们之前看到的services中的方法
		tracks, err := services.Scheduler.SyncDeviceTrack(deviceID, startTime, endTime)
		if err != nil {
			log.Printf("❌ 运单 %s 同步失败: %v", shipment.ID, err)
		} else {
			log.Printf("✅ 运单 %s 同步成功，获取 %d 个轨迹点", shipment.ID, len(tracks))
			successCount++

			// 【闭环逻辑】轨迹同步后，自动触发停留检测分析
			// 获取设备外部ID（如果有）
			deviceExternalID := ""
			if shipment.DeviceID != nil && *shipment.DeviceID == deviceID {
				// 从设备记录获取外部ID
				var device struct {
					ExternalID string `json:"external_id"`
				}
				if dbErr := db.Table("devices").Select("external_id").Where("id = ?", deviceID).First(&device).Error; dbErr == nil {
					deviceExternalID = device.ExternalID
				}
			}

			// 调用停留分析服务
			if deviceExternalID != "" {
				deviceStopService := services.NewDeviceStopService(db)
				analyzeErr := deviceStopService.AnalyzeDeviceTracksAndCreateStops(
					deviceID,
					deviceExternalID,
					shipment.ID,
					startTime,
					endTime,
				)
				if analyzeErr != nil {
					log.Printf("⚠️  运单 %s 停留分析失败: %v", shipment.ID, analyzeErr)
				} else {
					log.Printf("✅ 运单 %s 停留分析完成", shipment.ID)
				}
			}
		}

		// 避免API速率限制
		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("同步完成: 成功 %d, 跳过 %d, 总计 %d", successCount, skipCount, len(shipments))
}
