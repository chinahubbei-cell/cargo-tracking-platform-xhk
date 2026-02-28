// 诊断脚本：检查运单的围栏触发条件
// 用法: go run scripts/diagnose_geofence.go 运单号1 运单号2 ...
package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 简化的模型定义
type Shipment struct {
	ID                string    `gorm:"primaryKey"`
	DeviceID          *string   `gorm:"column:device_id"`
	Status            string    `gorm:"column:status"`
	OriginLat         *float64  `gorm:"column:origin_lat"`
	OriginLng         *float64  `gorm:"column:origin_lng"`
	DestLat           *float64  `gorm:"column:dest_lat"`
	DestLng           *float64  `gorm:"column:dest_lng"`
	OriginFenceRadius *int      `gorm:"column:origin_fence_radius"`
	DestFenceRadius   *int      `gorm:"column:dest_fence_radius"`
	AutoStatusEnabled bool      `gorm:"column:auto_status_enabled"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

type Device struct {
	ID        string    `gorm:"primaryKey"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type DeviceTrack struct {
	ID         uint      `gorm:"primaryKey"`
	DeviceID   string    `gorm:"column:device_id"`
	Latitude   float64   `gorm:"column:latitude"`
	Longitude  float64   `gorm:"column:longitude"`
	Speed      float64   `gorm:"column:speed"`
	LocateTime time.Time `gorm:"column:locate_time"`
}

type ShipmentLog struct {
	ID          uint      `gorm:"primaryKey"`
	ShipmentID  string    `gorm:"column:shipment_id"`
	Action      string    `gorm:"column:action"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

// Haversine 距离计算
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func printHeader(title string) {
	fmt.Println("\n" + "=" + title + " " + "=")
}

func printCheck(ok bool, msg string) {
	if ok {
		fmt.Printf("  ✅ %s\n", msg)
	} else {
		fmt.Printf("  ❌ %s\n", msg)
	}
}

func printInfo(msg string) {
	fmt.Printf("  ℹ️  %s\n", msg)
}

func diagnoseShipment(db *gorm.DB, shipmentID string) {
	fmt.Printf("\n%s\n", "══════════════════════════════════════════════════════════════")
	fmt.Printf("🔍 诊断运单: %s\n", shipmentID)
	fmt.Printf("%s\n", "══════════════════════════════════════════════════════════════")

	// 1. 查询运单基本信息
	var shipment Shipment
	if err := db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		fmt.Printf("❌ 运单不存在: %v\n", err)
		return
	}

	printHeader("基础配置检查")

	// 检查自动状态开关
	printCheck(shipment.AutoStatusEnabled, fmt.Sprintf("自动状态开关: %v", shipment.AutoStatusEnabled))

	// 检查运单状态
	validStatus := shipment.Status == "pending" || shipment.Status == "in_transit"
	printCheck(validStatus, fmt.Sprintf("运单状态: %s (需要pending或in_transit)", shipment.Status))

	// 检查设备绑定
	hasDevice := shipment.DeviceID != nil && *shipment.DeviceID != ""
	if hasDevice {
		printCheck(true, fmt.Sprintf("绑定设备: %s", *shipment.DeviceID))
	} else {
		printCheck(false, "设备绑定: 未绑定设备")
		return
	}

	// 2. 检查坐标配置
	printHeader("坐标配置检查")

	hasOrigin := shipment.OriginLat != nil && shipment.OriginLng != nil
	if hasOrigin {
		printCheck(true, fmt.Sprintf("发货地坐标: %.6f, %.6f", *shipment.OriginLat, *shipment.OriginLng))
	} else {
		printCheck(false, "发货地坐标: 未配置")
	}

	hasDest := shipment.DestLat != nil && shipment.DestLng != nil
	if hasDest {
		printCheck(true, fmt.Sprintf("目的地坐标: %.6f, %.6f", *shipment.DestLat, *shipment.DestLng))
	} else {
		printCheck(false, "目的地坐标: 未配置")
	}

	originRadius := 500
	if shipment.OriginFenceRadius != nil && *shipment.OriginFenceRadius > 100 {
		originRadius = *shipment.OriginFenceRadius
	}
	printInfo(fmt.Sprintf("发货地围栏半径: %d 米", originRadius))

	destRadius := 500
	if shipment.DestFenceRadius != nil && *shipment.DestFenceRadius > 100 {
		destRadius = *shipment.DestFenceRadius
	}
	printInfo(fmt.Sprintf("目的地围栏半径: %d 米", destRadius))

	// 3. 检查设备状态
	printHeader("设备状态检查")
	var device Device
	if err := db.First(&device, "id = ?", *shipment.DeviceID).Error; err != nil {
		printCheck(false, fmt.Sprintf("设备不存在: %v", err))
	} else {
		printCheck(device.Status == "active" || device.Status == "bound" || device.Status == "online", fmt.Sprintf("设备状态: %s", device.Status))
	}

	// 4. 检查轨迹数据
	printHeader("轨迹数据检查")
	var trackCount int64
	db.Model(&DeviceTrack{}).Where("device_id = ?", *shipment.DeviceID).Count(&trackCount)
	printCheck(trackCount > 0, fmt.Sprintf("轨迹总数: %d 条", trackCount))

	// 最近轨迹
	var recentTracks []DeviceTrack
	db.Where("device_id = ?", *shipment.DeviceID).Order("locate_time DESC").Limit(5).Find(&recentTracks)

	if len(recentTracks) > 0 {
		printInfo("最近5条轨迹:")
		for i, t := range recentTracks {
			distToOrigin := "-"
			distToDest := "-"
			originStatus := ""
			destStatus := ""

			if hasOrigin {
				d := haversineDistance(t.Latitude, t.Longitude, *shipment.OriginLat, *shipment.OriginLng)
				distToOrigin = fmt.Sprintf("%.0f米", d)
				if d <= float64(originRadius) {
					originStatus = " [在发货地围栏内]"
				} else {
					originStatus = " [在发货地围栏外]"
				}
			}

			if hasDest {
				d := haversineDistance(t.Latitude, t.Longitude, *shipment.DestLat, *shipment.DestLng)
				distToDest = fmt.Sprintf("%.0f米", d)
				if d <= float64(destRadius) {
					destStatus = " [在目的地围栏内]"
				} else {
					destStatus = " [在目的地围栏外]"
				}
			}

			fmt.Printf("    [%d] %s | 位置: %.4f,%.4f | 距发货地: %s%s | 距目的地: %s%s\n",
				i+1, t.LocateTime.Format("01-02 15:04:05"), t.Latitude, t.Longitude, distToOrigin, originStatus, distToDest, destStatus)
		}

		// 检查最近3个点是否满足触发条件
		if len(recentTracks) >= 3 && hasOrigin && hasDest {
			printInfo("")
			// 检查发货地离开条件
			if shipment.Status == "pending" {
				outsideCount := 0
				for _, t := range recentTracks[:3] {
					d := haversineDistance(t.Latitude, t.Longitude, *shipment.OriginLat, *shipment.OriginLng)
					if d > float64(originRadius) {
						outsideCount++
					}
				}
				if outsideCount >= 3 {
					printCheck(true, "发货地离开条件: 满足 (最近3个点都在围栏外)")
				} else {
					printCheck(false, fmt.Sprintf("发货地离开条件: 不满足 (仅%d/3个点在围栏外)", outsideCount))
				}
			}

			// 检查目的地到达条件
			if shipment.Status == "in_transit" {
				insideCount := 0
				for _, t := range recentTracks[:3] {
					d := haversineDistance(t.Latitude, t.Longitude, *shipment.DestLat, *shipment.DestLng)
					if d <= float64(destRadius) {
						insideCount++
					}
				}
				if insideCount >= 3 {
					printCheck(true, "目的地到达条件: 满足 (最近3个点都在围栏内)")
				} else {
					printCheck(false, fmt.Sprintf("目的地到达条件: 不满足 (仅%d/3个点在围栏内)", insideCount))
				}
			}
		}
	} else {
		printCheck(false, "没有轨迹数据")
	}

	// 5. 检查围栏触发日志
	printHeader("围栏触发日志检查")
	var logs []ShipmentLog
	db.Where("shipment_id = ? AND action = 'geofence_trigger'", shipmentID).Order("created_at DESC").Limit(5).Find(&logs)

	if len(logs) > 0 {
		printCheck(true, fmt.Sprintf("找到 %d 条围栏触发日志:", len(logs)))
		for _, l := range logs {
			fmt.Printf("    📍 %s: %s\n", l.CreatedAt.Format("01-02 15:04:05"), l.Description)
		}
	} else {
		printCheck(false, "没有围栏触发日志")
	}

	// 6. 诊断结论
	printHeader("诊断结论")
	issues := []string{}

	if !shipment.AutoStatusEnabled {
		issues = append(issues, "自动状态开关已关闭，需要开启")
	}
	if !validStatus {
		issues = append(issues, fmt.Sprintf("运单状态为 %s，只有pending/in_transit状态才会触发围栏", shipment.Status))
	}
	if !hasOrigin {
		issues = append(issues, "发货地坐标未配置")
	}
	if !hasDest {
		issues = append(issues, "目的地坐标未配置")
	}
	if trackCount == 0 {
		issues = append(issues, "设备没有上报任何轨迹，请检查设备是否正常工作")
	} else if len(recentTracks) < 3 {
		issues = append(issues, "轨迹点少于3条，围栏触发需要至少连续3个轨迹点确认")
	}

	if len(issues) == 0 {
		fmt.Println("  ✅ 所有条件正常，等待设备上报满足条件的轨迹点即可触发")
	} else {
		fmt.Println("  ⚠️  发现以下问题需要解决:")
		for i, issue := range issues {
			fmt.Printf("    %d. %s\n", i+1, issue)
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run scripts/diagnose_geofence.go 运单号1 [运单号2] ...")
		fmt.Println("示例: go run scripts/diagnose_geofence.go 260123000001 260123000002")
		os.Exit(1)
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open("trackcard.db"), &gorm.Config{})
	if err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔧 围栏触发诊断工具")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 诊断每个运单
	for _, shipmentID := range os.Args[1:] {
		diagnoseShipment(db, shipmentID)
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("诊断完成")
}
