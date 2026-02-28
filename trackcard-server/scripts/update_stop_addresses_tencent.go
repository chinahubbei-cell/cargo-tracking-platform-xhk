package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"trackcard-server/services"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 检查腾讯地图服务是否已初始化
	if services.TencentMap == nil {
		// 如果未初始化，使用默认配置
		services.InitTencentMap("C42BZ-YNQKV-VV5PV-5A2IY-TRSWQ-7XFR5", "")
		log.Println("[腾讯地图] 使用默认配置初始化")
	}

	// 打开数据库
	db, err := sql.Open("sqlite3", "trackcard.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== 开始更新停留记录地址（使用腾讯地图） ===")

	// 查询所有没有地址的停留记录（只查询最近的20条）
	rows, err := db.Query(`
		SELECT id, latitude, longitude
		FROM device_stop_records
		WHERE (address IS NULL OR address = '')
		AND latitude IS NOT NULL
		AND longitude IS NOT NULL
		ORDER BY start_time DESC
		LIMIT 20
	`)
	if err != nil {
		log.Fatalf("查询停留记录失败: %v", err)
	}
	defer rows.Close()

	var records []struct {
		ID        string
		Latitude  float64
		Longitude float64
	}

	for rows.Next() {
		var r struct {
			ID        string
			Latitude  float64
			Longitude float64
		}
		err := rows.Scan(&r.ID, &r.Latitude, &r.Longitude)
		if err != nil {
			log.Printf("扫描记录失败: %v", err)
			continue
		}
		records = append(records, r)
	}

	if len(records) == 0 {
		fmt.Println("没有需要更新地址的记录")
		return
	}

	fmt.Printf("找到 %d 条需要更新地址的记录\n", len(records))

	// 准备更新语句
	updateStmt, err := db.Prepare(`
		UPDATE device_stop_records
		SET address = ?, updated_at = ?
		WHERE id = ?
	`)
	if err != nil {
		log.Fatalf("准备更新语句失败: %v", err)
	}
	defer updateStmt.Close()

	updatedCount := 0
	failedCount := 0

	for i, record := range records {
		fmt.Printf("[%d/%d] 正在获取 %s 的地址...", i+1, len(records), record.ID)

		// 调用腾讯地图逆地理编码
		address, err := services.TencentMap.ReverseGeocode(record.Latitude, record.Longitude)
		if err != nil {
			log.Printf(" 失败: %v\n", err)
			// 使用经纬度作为备用地址
			address = fmt.Sprintf("%.6f, %.6f", record.Latitude, record.Longitude)
			failedCount++
		} else {
			fmt.Printf(" 成功\n")
			updatedCount++
		}

		// 更新数据库
		_, err = updateStmt.Exec(address, time.Now(), record.ID)
		if err != nil {
			log.Printf("更新数据库失败 %s: %v", record.ID, err)
			continue
		}

		fmt.Printf("  📍 %s\n", address)

		// 短暂延迟，避免请求过快
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\n=== 地址更新完成 ===\n")
	fmt.Printf("成功: %d, 失败: %d, 总计: %d\n", updatedCount, failedCount, len(records))
}
