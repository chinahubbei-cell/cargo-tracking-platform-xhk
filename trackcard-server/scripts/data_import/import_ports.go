package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Port 港口模型
type Port struct {
	ID              string `gorm:"primaryKey;type:varchar(36)"`
	Code            string `gorm:"type:varchar(20);uniqueIndex"`
	Name            string `gorm:"type:varchar(200);not null"`
	NameEn          string `gorm:"type:varchar(200)"`
	Country         string `gorm:"type:varchar(10)"`
	Region          string `gorm:"type:varchar(100)"`
	Type            string `gorm:"type:varchar(20);default:'seaport'"`
	Tier            int    `gorm:"default:2"`
	Latitude        float64
	Longitude       float64
	Timezone        string  `gorm:"type:varchar(50)"`
	GeofenceKM      float64 `gorm:"default:15"`
	IsTransitHub    bool    `gorm:"default:false"`
	CustomsEff      int     `gorm:"default:3"`
	CongestionLevel int     `gorm:"default:1"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (Port) TableName() string {
	return "ports"
}

// 全球主要海港列表 (用于标记 is_transit_hub)
var majorSeaports = map[string]bool{
	// 亚洲主要港口
	"CNSHA": true, "CNNBO": true, "CNSZX": true, "CNYTN": true, "CNQIN": true,
	"CNXMN": true, "CNDLC": true, "CNTAO": true, "HKHKG": true, "TWKHH": true,
	"SGSIN": true, "MYPEN": true, "MYPKG": true, "IDTPP": true, "IDJKT": true,
	"VNSGN": true, "VNHPH": true, "THBKK": true, "THLCH": true,
	"JPYOK": true, "JPTYO": true, "JPNGO": true, "JPOSA": true, "JPKOB": true,
	"KRPUS": true, "KRINC": true,
	// 欧洲主要港口
	"NLRTM": true, "NLAMS": true, "DEHAM": true, "DEBHV": true,
	"BEANR": true, "BEZEE": true,
	"GBFXT": true, "GBSOU": true, "GBLGP": true,
	"FRLEH": true, "FRMRS": true,
	"ESALG": true, "ESVLC": true, "ESBAR": true,
	"ITGOA": true, "ITGIT": true, "ITMIL": true,
	"GRTPK": true,
	// 北美主要港口
	"USLAX": true, "USLGB": true, "USNYC": true, "USSAV": true, "USHOU": true,
	"USSEA": true, "USOAK": true, "USCHI": true, "USMIA": true,
	"CAMTR": true, "CAVAN": true, "CAPRR": true,
	// 中东主要港口
	"AEJEA": true, "AEKLP": true, "AEAUH": true, "OMSLL": true,
	"SAJED": true, "SADMM": true,
	// 大洋洲主要港口
	"AUSYD": true, "AUMEL": true, "AUBNE": true, "AUFRE": true,
	"NZAKL": true, "NZTRG": true,
	// 非洲主要港口
	"ZADUR": true, "ZACPT": true, "EGPSD": true, "EGALY": true,
	"MAPTM": true, "MATNG": true,
	// 南美主要港口
	"BRSSZ": true, "BRPNG": true, "BRRIO": true,
	"ARBUE": true, "CLSAI": true, "PECLL": true, "COPBG": true,
}

// 区域映射
var regionMap = map[string]string{
	"AD": "Europe", "AE": "Middle East", "AF": "Asia", "AG": "Caribbean",
	"AL": "Europe", "AM": "Asia", "AO": "Africa", "AR": "South America",
	"AT": "Europe", "AU": "Oceania", "AZ": "Asia",
	"BA": "Europe", "BB": "Caribbean", "BD": "Asia", "BE": "Europe",
	"BG": "Europe", "BH": "Middle East", "BI": "Africa", "BJ": "Africa",
	"BN": "Asia", "BO": "South America", "BR": "South America", "BS": "Caribbean",
	"BW": "Africa", "BY": "Europe",
	"CA": "North America", "CD": "Africa", "CF": "Africa", "CG": "Africa",
	"CH": "Europe", "CI": "Africa", "CL": "South America", "CM": "Africa",
	"CN": "Asia", "CO": "South America", "CR": "Central America", "CU": "Caribbean",
	"CY": "Europe", "CZ": "Europe",
	"DE": "Europe", "DJ": "Africa", "DK": "Europe", "DO": "Caribbean",
	"DZ": "Africa",
	"EC": "South America", "EE": "Europe", "EG": "Africa", "ER": "Africa",
	"ES": "Europe", "ET": "Africa",
	"FI": "Europe", "FJ": "Oceania", "FR": "Europe",
	"GA": "Africa", "GB": "Europe", "GE": "Asia", "GH": "Africa",
	"GR": "Europe", "GT": "Central America", "GN": "Africa",
	"HK": "Asia", "HN": "Central America", "HR": "Europe", "HT": "Caribbean",
	"HU": "Europe",
	"ID": "Asia", "IE": "Europe", "IL": "Middle East", "IN": "Asia",
	"IQ": "Middle East", "IR": "Middle East", "IS": "Europe", "IT": "Europe",
	"JM": "Caribbean", "JO": "Middle East", "JP": "Asia",
	"KE": "Africa", "KG": "Asia", "KH": "Asia", "KR": "Asia", "KW": "Middle East",
	"KZ": "Asia",
	"LA": "Asia", "LB": "Middle East", "LK": "Asia", "LT": "Europe",
	"LU": "Europe", "LV": "Europe", "LY": "Africa",
	"MA": "Africa", "MC": "Europe", "MD": "Europe", "ME": "Europe",
	"MG": "Africa", "MK": "Europe", "ML": "Africa", "MM": "Asia",
	"MN": "Asia", "MO": "Asia", "MR": "Africa", "MT": "Europe",
	"MU": "Africa", "MV": "Asia", "MW": "Africa", "MX": "North America",
	"MY": "Asia", "MZ": "Africa",
	"NA": "Africa", "NE": "Africa", "NG": "Africa", "NI": "Central America",
	"NL": "Europe", "NO": "Europe", "NP": "Asia", "NZ": "Oceania",
	"OM": "Middle East",
	"PA": "Central America", "PE": "South America", "PG": "Oceania",
	"PH": "Asia", "PK": "Asia", "PL": "Europe", "PT": "Europe",
	"PY": "South America",
	"QA": "Middle East",
	"RO": "Europe", "RS": "Europe", "RU": "Europe", "RW": "Africa",
	"SA": "Middle East", "SD": "Africa", "SE": "Europe", "SG": "Asia",
	"SI": "Europe", "SK": "Europe", "SL": "Africa", "SN": "Africa",
	"SO": "Africa", "SR": "South America", "SS": "Africa", "SV": "Central America",
	"SY": "Middle East",
	"TD": "Africa", "TG": "Africa", "TH": "Asia", "TJ": "Asia",
	"TM": "Asia", "TN": "Africa", "TR": "Europe", "TT": "Caribbean",
	"TW": "Asia", "TZ": "Africa",
	"UA": "Europe", "UG": "Africa", "US": "North America", "UY": "South America",
	"UZ": "Asia",
	"VE": "South America", "VN": "Asia",
	"YE": "Middle East",
	"ZA": "Africa", "ZM": "Africa", "ZW": "Africa",
}

func main() {
	fmt.Println("=== 全球港口数据导入工具 ===")
	fmt.Printf("开始时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// 连接数据库
	db, err := gorm.Open(sqlite.Open("../../trackcard.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 读取 CSV 文件
	file, err := os.Open("data_sources/ports_unlocode.csv")
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
	var seaports, railports int

	// 跳过表头
	for i, record := range records[1:] {
		if i > 0 && i%20000 == 0 {
			fmt.Printf("处理进度: %d/%d\n", i, len(records)-1)
		}

		// CSV 字段: Change,Country,Location,Name,NameWoDiacritics,Subdivision,Status,Function,Date,IATA,Coordinates,Remarks
		// 索引:      0      1       2        3    4                5           6      7        8    9    10          11

		if len(record) < 11 {
			skipped++
			continue
		}

		countryCode := strings.TrimSpace(record[1])
		locationCode := strings.TrimSpace(record[2])
		name := strings.TrimSpace(record[3])
		nameWoDiacritics := strings.TrimSpace(record[4])
		function := strings.TrimSpace(record[7])
		coordinates := strings.TrimSpace(record[10])

		// 构建 UN/LOCODE
		unlocode := countryCode + locationCode
		if len(unlocode) != 5 {
			skipped++
			continue
		}

		// 解析功能代码 (Function)
		// 格式: 位置表示功能, 例如 "1-345---"
		// 位置0=港口(1), 位置1=铁路(2), 位置2=公路(3), 位置3=机场(4), 位置5=多式联运(6), 位置7=内河港(8)
		hasPort := len(function) > 0 && function[0] == '1'
		hasRail := len(function) > 1 && function[1] == '2'
		hasMultimodal := len(function) > 5 && function[5] == '6'
		hasRiver := len(function) > 7 && function[7] == '8'

		// 只导入有港口/铁路/多式联运/内河功能的地点
		if !hasPort && !hasRail && !hasMultimodal && !hasRiver {
			skipped++
			continue
		}

		// 确定类型
		portType := "seaport"
		if hasRail && !hasPort && !hasRiver {
			portType = "railway"
			railports++
		} else if hasRiver && !hasPort {
			portType = "river"
			seaports++
		} else {
			seaports++
		}

		// 解析坐标
		lat, lng := parseCoordinates(coordinates)

		// 对于主要港口，即使没有坐标也导入
		isKnownMajor := majorSeaports[unlocode]
		if lat == 0 && lng == 0 && !isKnownMajor {
			// 无坐标数据且不是主要港口，跳过
			skipped++
			continue
		}

		// 获取区域
		region := regionMap[countryCode]
		if region == "" {
			region = "Other"
		}

		// 确定是否主要港口
		isTransitHub := majorSeaports[unlocode]

		// 确定等级
		tier := 3
		if isTransitHub {
			tier = 1
		} else if hasMultimodal {
			tier = 2
		}

		// 使用无重音符号的名称作为英文名
		nameEn := nameWoDiacritics
		if nameEn == "" {
			nameEn = name
		}

		// 构建港口对象
		port := Port{
			Code:         unlocode,
			Name:         name,
			NameEn:       nameEn,
			Country:      countryCode,
			Region:       region,
			Type:         portType,
			Tier:         tier,
			Latitude:     lat,
			Longitude:    lng,
			GeofenceKM:   15,
			IsTransitHub: isTransitHub,
			CustomsEff:   3,
		}

		// Upsert: 按代码查找或创建
		var existing Port
		result := db.Where("code = ?", unlocode).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// 新记录
			port.ID = uuid.New().String()
			port.CreatedAt = time.Now()
			port.UpdatedAt = time.Now()

			if err := db.Create(&port).Error; err != nil {
				// 可能是唯一键冲突，尝试更新
				continue
			}
			imported++
		} else if result.Error == nil {
			// 更新现有记录
			updates := map[string]interface{}{
				"name_en":    nameEn,
				"country":    countryCode,
				"region":     region,
				"latitude":   lat,
				"longitude":  lng,
				"updated_at": time.Now(),
			}

			// 如果当前数据标记为枢纽，不覆盖
			if isTransitHub && !existing.IsTransitHub {
				updates["is_transit_hub"] = true
				updates["tier"] = 1
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
	fmt.Printf("海港数量: %d\n", seaports)
	fmt.Printf("铁路港数量: %d\n", railports)

	// 验证导入结果
	var total int64
	var countryCount int64
	db.Model(&Port{}).Count(&total)
	db.Model(&Port{}).Distinct("country").Count(&countryCount)

	fmt.Printf("\n数据库统计:\n")
	fmt.Printf("  港口总数: %d\n", total)
	fmt.Printf("  覆盖国家: %d\n", countryCount)
}

// parseCoordinates 解析 UN/LOCODE 坐标格式
// 格式: "4230N 00131E" -> (42.5, 1.517)
func parseCoordinates(coord string) (float64, float64) {
	if coord == "" {
		return 0, 0
	}

	// 匹配格式: DDMMN/S DDDMME/W
	re := regexp.MustCompile(`(\d{2})(\d{2})([NS])\s*(\d{3})(\d{2})([EW])`)
	matches := re.FindStringSubmatch(coord)
	if len(matches) != 7 {
		return 0, 0
	}

	latDeg, _ := strconv.ParseFloat(matches[1], 64)
	latMin, _ := strconv.ParseFloat(matches[2], 64)
	latDir := matches[3]

	lngDeg, _ := strconv.ParseFloat(matches[4], 64)
	lngMin, _ := strconv.ParseFloat(matches[5], 64)
	lngDir := matches[6]

	lat := latDeg + latMin/60.0
	lng := lngDeg + lngMin/60.0

	if latDir == "S" {
		lat = -lat
	}
	if lngDir == "W" {
		lng = -lng
	}

	return lat, lng
}
