package main

import (
	"flag"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trackcard-server/config"
	"trackcard-server/models"
	"trackcard-server/services"
)

func main() {
	shipmentID := flag.String("shipment", "", "仅修复指定运单")
	limit := flag.Int("limit", 0, "最多处理条数，0=不限制")
	dryRun := flag.Bool("dry-run", false, "仅预览不写库")
	force := flag.Bool("force", false, "强制重算（不只限经纬度样式地址）")
	sleepMS := flag.Int("sleep-ms", 80, "每条更新后的休眠毫秒，防止外部API突发请求")
	flag.Parse()

	dbPath, err := resolveDBPath()
	if err != nil {
		log.Fatalf("定位数据库失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	config.Load()
	tencentMapSK := os.Getenv("TENCENT_MAP_SK")
	if tencentMapSK == "" {
		tencentMapSK = "1oJ0xgyshoOMEoTzQvGxfNl80V85oKWS"
	}
	services.InitTencentMap("C42BZ-YNQKV-VV5PV-5A2IY-TRSWQ-7XFR5", tencentMapSK)
	services.InitNominatimService("admin@trackcard.com")

	query := db.Model(&models.DeviceStopRecord{})
	if strings.TrimSpace(*shipmentID) != "" {
		query = query.Where("shipment_id = ?", strings.TrimSpace(*shipmentID))
	}
	if !*force {
		query = query.Where(
			"address IS NULL OR address = '' OR address LIKE ? OR address LIKE ? OR address LIKE ? OR address LIKE ? OR address LIKE ?",
			"%北纬%", "%南纬%", "%东经%", "%西经%", "%国际区域%",
		)
	}
	if *limit > 0 {
		query = query.Limit(*limit)
	}

	var records []models.DeviceStopRecord
	if err := query.Order("start_time ASC").Find(&records).Error; err != nil {
		log.Fatalf("查询停留记录失败: %v", err)
	}

	log.Printf("待处理停留记录: %d 条 (shipment=%s, force=%v, dryRun=%v)",
		len(records), strings.TrimSpace(*shipmentID), *force, *dryRun)

	var updated, skipped, failed int
	for _, record := range records {
		if record.Latitude == nil || record.Longitude == nil || !isValidCoordinate(*record.Latitude, *record.Longitude) {
			skipped++
			continue
		}

		oldAddress := strings.TrimSpace(record.Address)
		if !*force && oldAddress != "" && !services.IsCoordinateAddress(oldAddress) {
			skipped++
			continue
		}

		newAddress := strings.TrimSpace(services.ResolveNodeAddress(*record.Latitude, *record.Longitude))
		if newAddress == "" {
			failed++
			log.Printf("⚠️  解析地址失败: id=%s shipment=%s lat=%.6f lng=%.6f",
				record.ID, record.ShipmentID, *record.Latitude, *record.Longitude)
			continue
		}
		if newAddress == oldAddress {
			skipped++
			continue
		}

		log.Printf("🔧 更新停留地址: id=%s shipment=%s\n    old=%s\n    new=%s",
			record.ID, record.ShipmentID, oldAddress, newAddress)

		if !*dryRun {
			if err := db.Model(&models.DeviceStopRecord{}).
				Where("id = ?", record.ID).
				Updates(map[string]interface{}{
					"address":    newAddress,
					"updated_at": time.Now(),
				}).Error; err != nil {
				failed++
				log.Printf("❌ 写库失败: id=%s err=%v", record.ID, err)
				continue
			}
		}

		updated++
		if *sleepMS > 0 {
			time.Sleep(time.Duration(*sleepMS) * time.Millisecond)
		}
	}

	log.Printf("完成: updated=%d skipped=%d failed=%d", updated, skipped, failed)
}

func resolveDBPath() (string, error) {
	candidates := []string{
		"trackcard.db",
		"../trackcard-server/trackcard.db",
		"/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk/trackcard-server/trackcard.db",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

func isValidCoordinate(lat, lng float64) bool {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return false
	}
	if math.Abs(lat) < 0.000001 && math.Abs(lng) < 0.000001 {
		return false
	}
	return true
}
