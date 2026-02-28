package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Airport 机场模型 (与 models/airport.go 保持一致)
type Airport struct {
	ID                string `gorm:"primaryKey;type:varchar(36)"`
	IATACode          string `gorm:"type:varchar(3);uniqueIndex"`
	ICAOCode          string `gorm:"type:varchar(4)"`
	Name              string `gorm:"type:varchar(200);not null"`
	NameEn            string `gorm:"type:varchar(200)"`
	City              string `gorm:"type:varchar(100)"`
	Country           string `gorm:"type:varchar(10)"`
	Region            string `gorm:"type:varchar(100)"`
	Type              string `gorm:"type:varchar(20);default:'cargo'"`
	Tier              int    `gorm:"default:2"`
	Latitude          float64
	Longitude         float64
	Timezone          string  `gorm:"type:varchar(50)"`
	GeofenceKM        float64 `gorm:"default:10"`
	IsCargoHub        bool    `gorm:"default:false"`
	CustomsEfficiency int     `gorm:"default:3"`
	CongestionLevel   int     `gorm:"default:1"`
	RunwayCount       int     `gorm:"default:1"`
	CargoTerminals    int     `gorm:"default:1"`
	AnnualCargoTons   float64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

func (Airport) TableName() string {
	return "airports"
}

// 主要货运机场列表 (用于标记 is_cargo_hub)
var majorCargoHubs = map[string]bool{
	// 亚洲
	"HKG": true, "PVG": true, "ICN": true, "NRT": true, "SIN": true,
	"TPE": true, "CAN": true, "PEK": true, "BKK": true, "KUL": true,
	"DEL": true, "BOM": true, "SGN": true, "HAN": true, "CGK": true,
	// 欧洲
	"FRA": true, "CDG": true, "AMS": true, "LHR": true, "LGG": true,
	"LEJ": true, "MXP": true, "MAD": true, "IST": true, "SVO": true,
	"DME": true, "ZRH": true, "BRU": true, "LUX": true, "VIE": true,
	// 北美
	"MEM": true, "LAX": true, "ORD": true, "MIA": true, "JFK": true,
	"SDF": true, "ANC": true, "CVG": true, "ATL": true, "DFW": true,
	// 中东
	"DXB": true, "DOH": true, "AUH": true, "JED": true, "RUH": true,
	// 大洋洲
	"SYD": true, "MEL": true, "AKL": true,
	// 拉美
	"GRU": true, "MEX": true, "BOG": true, "SCL": true, "EZE": true,
	// 非洲
	"JNB": true, "CAI": true, "NBO": true, "ADD": true, "LOS": true,
}

// 区域映射
var regionMap = map[string]string{
	"AF": "Africa", "AS": "Asia", "EU": "Europe",
	"NA": "North America", "OC": "Oceania", "SA": "South America",
}

func main() {
	fmt.Println("=== 全球机场数据导入工具 ===")
	fmt.Printf("开始时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 连接数据库
	db, err := gorm.Open(sqlite.Open("../../trackcard.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 读取 CSV 文件
	file, err := os.Open("data_sources/airports.csv")
	if err != nil {
		log.Fatalf("无法打开 CSV 文件: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("读取 CSV 失败: %v", err)
	}

	fmt.Printf("CSV 文件总记录数: %d\n", len(records)-1)

	// 统计
	var imported, updated, skipped int
	var largeAirports, mediumAirports int

	// 跳过表头
	for i, record := range records[1:] {
		if i > 0 && i%10000 == 0 {
			fmt.Printf("处理进度: %d/%d\n", i, len(records)-1)
		}

		// CSV 字段索引 (基于表头)
		// 0:id, 1:ident, 2:type, 3:name, 4:latitude_deg, 5:longitude_deg,
		// 6:elevation_ft, 7:continent, 8:iso_country, 9:iso_region,
		// 10:municipality, 11:scheduled_service, 12:icao_code, 13:iata_code, ...

		airportType := record[2]
		iataCode := strings.TrimSpace(record[13])
		icaoCode := strings.TrimSpace(record[12])

		// 只导入有 IATA 代码的大中型机场
		if iataCode == "" || len(iataCode) != 3 {
			skipped++
			continue
		}

		// 只导入 large_airport 和 medium_airport
		if airportType != "large_airport" && airportType != "medium_airport" {
			skipped++
			continue
		}

		if airportType == "large_airport" {
			largeAirports++
		} else {
			mediumAirports++
		}

		// 解析坐标
		lat, _ := strconv.ParseFloat(record[4], 64)
		lng, _ := strconv.ParseFloat(record[5], 64)

		// 获取国家代码 (ISO 2字母)
		countryCode := ""
		if len(record[8]) >= 2 {
			countryCode = record[8][:2]
		}

		// 获取区域
		continent := record[7]
		region := regionMap[continent]
		if region == "" {
			region = "Other"
		}

		// 确定是否货运枢纽
		isCargoHub := majorCargoHubs[iataCode]

		// 确定等级
		tier := 3
		if airportType == "large_airport" {
			if isCargoHub {
				tier = 1
			} else {
				tier = 2
			}
		}

		// 构建机场对象
		airport := Airport{
			IATACode:          iataCode,
			ICAOCode:          icaoCode,
			Name:              translateName(record[3], countryCode),
			NameEn:            record[3],
			City:              record[10],
			Country:           countryCode,
			Region:            region,
			Type:              "cargo",
			Tier:              tier,
			Latitude:          lat,
			Longitude:         lng,
			GeofenceKM:        10,
			IsCargoHub:        isCargoHub,
			CustomsEfficiency: 3,
			CongestionLevel:   1,
		}

		// Upsert: 按 IATA 代码查找或创建
		var existing Airport
		result := db.Where("iata_code = ?", iataCode).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// 新记录
			airport.ID = uuid.New().String()
			airport.CreatedAt = time.Now()
			airport.UpdatedAt = time.Now()

			if err := db.Create(&airport).Error; err != nil {
				log.Printf("创建失败 [%s]: %v", iataCode, err)
				continue
			}
			imported++
		} else if result.Error == nil {
			// 更新现有记录 (保留 is_cargo_hub 如果已标记)
			updates := map[string]interface{}{
				"icao_code":  icaoCode,
				"name_en":    record[3],
				"city":       record[10],
				"country":    countryCode,
				"region":     region,
				"latitude":   lat,
				"longitude":  lng,
				"tier":       tier,
				"updated_at": time.Now(),
			}

			// 如果当前数据标记为货运枢纽，不覆盖
			if isCargoHub && !existing.IsCargoHub {
				updates["is_cargo_hub"] = true
			}

			db.Model(&existing).Updates(updates)
			updated++
		}
	}

	// 输出统计
	fmt.Println("\n=== 导入完成 ===")
	fmt.Printf("完成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("新增记录: %d\n", imported)
	fmt.Printf("更新记录: %d\n", updated)
	fmt.Printf("跳过记录: %d\n", skipped)
	fmt.Printf("大型机场: %d\n", largeAirports)
	fmt.Printf("中型机场: %d\n", mediumAirports)

	// 验证导入结果
	var total int64
	var countryCount int64
	db.Model(&Airport{}).Count(&total)
	db.Model(&Airport{}).Distinct("country").Count(&countryCount)

	fmt.Printf("\n数据库统计:\n")
	fmt.Printf("  机场总数: %d\n", total)
	fmt.Printf("  覆盖国家: %d\n", countryCount)
}

// translateName 为常见机场添加中文名称
func translateName(englishName, countryCode string) string {
	// 中国机场中文名映射
	chineseNames := map[string]string{
		"Shanghai Pudong International Airport":   "上海浦东国际机场",
		"Beijing Capital International Airport":   "北京首都国际机场",
		"Hong Kong International Airport":         "香港国际机场",
		"Guangzhou Baiyun International Airport":  "广州白云国际机场",
		"Shenzhen Bao'an International Airport":   "深圳宝安国际机场",
		"Chengdu Shuangliu International Airport": "成都双流国际机场",
		"Shanghai Hongqiao International Airport": "上海虹桥国际机场",
		"Hangzhou Xiaoshan International Airport": "杭州萧山国际机场",
		"Taipei Taoyuan International Airport":    "台北桃园国际机场",
		// 可继续添加...
	}

	if name, ok := chineseNames[englishName]; ok {
		return name
	}

	return englishName
}
