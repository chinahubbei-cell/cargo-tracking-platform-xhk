package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// CountryBounds 国家/地区边界判断
type CountryBounds struct {
	Name   string
	NameEn string
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
}

var countries = []CountryBounds{
	{"中国", "China", 18.2, 53.6, 73.5, 135.1},
	{"俄罗斯", "Russia", 41.2, 81.9, 19.6, 169.0},
	{"哈萨克斯坦", "Kazakhstan", 40.7, 55.4, 46.5, 87.3},
	{"巴基斯坦", "Pakistan", 23.7, 37.1, 60.9, 77.8},
	{"印度", "India", 6.8, 35.5, 68.7, 97.3},
	{"蒙古", "Mongolia", 41.6, 52.1, 87.7, 119.9},
	{"吉尔吉斯斯坦", "Kyrgyzstan", 39.2, 43.3, 69.3, 80.3},
	{"乌兹别克斯坦", "Uzbekistan", 37.2, 45.6, 55.9, 73.2},
	{"德国", "Germany", 47.3, 55.1, 5.9, 15.0},
	{"法国", "France", 41.3, 51.1, -4.8, 9.6},
	{"英国", "United Kingdom", 49.9, 60.9, -7.7, 1.8},
	{"意大利", "Italy", 36.6, 47.1, 6.8, 18.5},
	{"西班牙", "Spain", 36.0, 43.8, -9.3, 3.7},
	{"荷兰", "Netherlands", 50.8, 53.6, 3.4, 7.2},
	{"比利时", "Belgium", 49.5, 51.5, 2.5, 6.4},
	{"波兰", "Poland", 49.0, 54.9, 14.1, 24.1},
	{"捷克", "Czech Republic", 48.6, 51.1, 12.1, 18.9},
	{"土耳其", "Turkey", 35.8, 42.1, 25.6, 44.8},
	{"乌克兰", "Ukraine", 44.4, 52.4, 22.1, 40.2},
	{"白俄罗斯", "Belarus", 51.3, 56.2, 23.2, 32.7},
}

func main() {
	db, err := sql.Open("sqlite3", "trackcard.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	shipmentID := "260128000004"
	fmt.Printf("=== 开始修复运单 %s 的空地址记录 ===\n", shipmentID)

	// 查询所有地址为空的记录
	rows, err := db.Query(`
		SELECT id, latitude, longitude
		FROM device_stop_records
		WHERE shipment_id = ?
		AND (address IS NULL OR address = '')
		ORDER BY start_time
	`, shipmentID)
	if err != nil {
		log.Fatalf("查询停留记录失败: %v", err)
	}
	defer rows.Close()

	type Record struct {
		ID        string
		Latitude  float64
		Longitude float64
	}

	var records []Record
	for rows.Next() {
		var r Record
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
	for i, record := range records {
		// 生成地址
		address := generateAddress(record.Latitude, record.Longitude)

		// 更新数据库
		_, err = updateStmt.Exec(address, time.Now(), record.ID)
		if err != nil {
			log.Printf("[%d/%d] 更新数据库失败 %s: %v", i+1, len(records), record.ID, err)
			continue
		}

		updatedCount++
		fmt.Printf("[%d/%d] %s: %s (%.4f, %.4f)\n", i+1, len(records), record.ID, address, record.Latitude, record.Longitude)
	}

	fmt.Printf("\n=== 地址更新完成 ===\n")
	fmt.Printf("成功更新 %d 条记录\n", updatedCount)
}

// generateAddress 根据经纬度生成地址描述
func generateAddress(lat, lng float64) string {
	// 判断国家
	country := findCountry(lat, lng)

	// 格式化经纬度
	latDir := "北纬"
	if lat < 0 {
		latDir = "南纬"
	}
	lngDir := "东经"
	if lng < 0 {
		lngDir = "西经"
	}

	coords := fmt.Sprintf("%s%.4f°, %s%.4f°", latDir, math.Abs(lat), lngDir, math.Abs(lng))

	// 组合地址
	if country != nil {
		return fmt.Sprintf("%s %s", country.Name, coords)
	}
	return fmt.Sprintf("国际区域 %s", coords)
}

// findCountry 根据经纬度查找所属国家
func findCountry(lat, lng float64) *CountryBounds {
	for i := range countries {
		c := &countries[i]
		if lat >= c.MinLat && lat <= c.MaxLat && lng >= c.MinLng && lng <= c.MaxLng {
			return c
		}
	}
	return nil
}
