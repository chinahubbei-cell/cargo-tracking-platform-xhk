package seeds

import (
	"log"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// SeedAirports 种子机场数据 - 全球主要货运机场
func SeedAirports(db *gorm.DB) error {
	airports := []models.Airport{
		// 中国大陆
		{IATACode: "PVG", ICAOCode: "ZSPD", Name: "上海浦东国际机场", NameEn: "Shanghai Pudong", City: "上海", Country: "CN", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 31.1443, Longitude: 121.8083, Timezone: "Asia/Shanghai", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 3, RunwayCount: 5, CargoTerminals: 3, AnnualCargoTons: 384},
		{IATACode: "PEK", ICAOCode: "ZBAA", Name: "北京首都国际机场", NameEn: "Beijing Capital", City: "北京", Country: "CN", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 40.0799, Longitude: 116.6031, Timezone: "Asia/Shanghai", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 3, RunwayCount: 3, CargoTerminals: 2, AnnualCargoTons: 195},
		{IATACode: "CAN", ICAOCode: "ZGGG", Name: "广州白云国际机场", NameEn: "Guangzhou Baiyun", City: "广州", Country: "CN", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 23.3924, Longitude: 113.2988, Timezone: "Asia/Shanghai", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 2, RunwayCount: 3, CargoTerminals: 2, AnnualCargoTons: 203},
		{IATACode: "SZX", ICAOCode: "ZGSZ", Name: "深圳宝安国际机场", NameEn: "Shenzhen Baoan", City: "深圳", Country: "CN", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 22.6393, Longitude: 113.8106, Timezone: "Asia/Shanghai", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 156},
		{IATACode: "CTU", ICAOCode: "ZUUU", Name: "成都双流国际机场", NameEn: "Chengdu Shuangliu", City: "成都", Country: "CN", Region: "East Asia", Type: "mixed", Tier: 2, Latitude: 30.5785, Longitude: 103.9471, Timezone: "Asia/Shanghai", GeofenceKM: 10, IsCargoHub: false, CustomsEfficiency: 4, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 68},
		{IATACode: "XIY", ICAOCode: "ZLXY", Name: "西安咸阳国际机场", NameEn: "Xi'an Xianyang", City: "西安", Country: "CN", Region: "East Asia", Type: "cargo", Tier: 2, Latitude: 34.4471, Longitude: 108.7516, Timezone: "Asia/Shanghai", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 1, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 42},
		{IATACode: "NKG", ICAOCode: "ZSNJ", Name: "南京禄口国际机场", NameEn: "Nanjing Lukou", City: "南京", Country: "CN", Region: "East Asia", Type: "mixed", Tier: 2, Latitude: 31.7420, Longitude: 118.8620, Timezone: "Asia/Shanghai", GeofenceKM: 10, IsCargoHub: false, CustomsEfficiency: 4, CongestionLevel: 1, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 45},
		{IATACode: "CGO", ICAOCode: "ZHCC", Name: "郑州新郑国际机场", NameEn: "Zhengzhou Xinzheng", City: "郑州", Country: "CN", Region: "East Asia", Type: "cargo", Tier: 2, Latitude: 34.5197, Longitude: 113.8408, Timezone: "Asia/Shanghai", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 1, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 70},

		// 香港/台湾
		{IATACode: "HKG", ICAOCode: "VHHH", Name: "香港国际机场", NameEn: "Hong Kong International", City: "香港", Country: "HK", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 22.3080, Longitude: 113.9185, Timezone: "Asia/Hong_Kong", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 3, AnnualCargoTons: 420},
		{IATACode: "TPE", ICAOCode: "RCTP", Name: "台北桃园国际机场", NameEn: "Taipei Taoyuan", City: "台北", Country: "TW", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 25.0797, Longitude: 121.2342, Timezone: "Asia/Taipei", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 232},

		// 日本/韩国
		{IATACode: "NRT", ICAOCode: "RJAA", Name: "东京成田国际机场", NameEn: "Narita International", City: "东京", Country: "JP", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 35.7720, Longitude: 140.3929, Timezone: "Asia/Tokyo", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 215},
		{IATACode: "HND", ICAOCode: "RJTT", Name: "东京羽田机场", NameEn: "Tokyo Haneda", City: "东京", Country: "JP", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 35.5494, Longitude: 139.7798, Timezone: "Asia/Tokyo", GeofenceKM: 10, IsCargoHub: false, CustomsEfficiency: 5, CongestionLevel: 3, RunwayCount: 4, CargoTerminals: 1, AnnualCargoTons: 95},
		{IATACode: "KIX", ICAOCode: "RJBB", Name: "大阪关西国际机场", NameEn: "Osaka Kansai", City: "大阪", Country: "JP", Region: "East Asia", Type: "mixed", Tier: 2, Latitude: 34.4272, Longitude: 135.2440, Timezone: "Asia/Tokyo", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 85},
		{IATACode: "ICN", ICAOCode: "RKSI", Name: "首尔仁川国际机场", NameEn: "Incheon International", City: "首尔", Country: "KR", Region: "East Asia", Type: "mixed", Tier: 1, Latitude: 37.4602, Longitude: 126.4407, Timezone: "Asia/Seoul", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 4, CargoTerminals: 2, AnnualCargoTons: 295},

		// 东南亚
		{IATACode: "SIN", ICAOCode: "WSSS", Name: "新加坡樟宜机场", NameEn: "Singapore Changi", City: "新加坡", Country: "SG", Region: "Southeast Asia", Type: "mixed", Tier: 1, Latitude: 1.3644, Longitude: 103.9915, Timezone: "Asia/Singapore", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 3, CargoTerminals: 2, AnnualCargoTons: 200},
		{IATACode: "BKK", ICAOCode: "VTBS", Name: "曼谷素万那普机场", NameEn: "Bangkok Suvarnabhumi", City: "曼谷", Country: "TH", Region: "Southeast Asia", Type: "mixed", Tier: 1, Latitude: 13.6900, Longitude: 100.7501, Timezone: "Asia/Bangkok", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 3, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 140},
		{IATACode: "KUL", ICAOCode: "WMKK", Name: "吉隆坡国际机场", NameEn: "Kuala Lumpur International", City: "吉隆坡", Country: "MY", Region: "Southeast Asia", Type: "mixed", Tier: 2, Latitude: 2.7456, Longitude: 101.7099, Timezone: "Asia/Kuala_Lumpur", GeofenceKM: 12, IsCargoHub: false, CustomsEfficiency: 4, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 75},
		{IATACode: "CGK", ICAOCode: "WIII", Name: "雅加达苏加诺-哈达机场", NameEn: "Jakarta Soekarno-Hatta", City: "雅加达", Country: "ID", Region: "Southeast Asia", Type: "mixed", Tier: 2, Latitude: -6.1256, Longitude: 106.6558, Timezone: "Asia/Jakarta", GeofenceKM: 12, IsCargoHub: false, CustomsEfficiency: 3, CongestionLevel: 3, RunwayCount: 3, CargoTerminals: 1, AnnualCargoTons: 65},

		// 中东
		{IATACode: "DXB", ICAOCode: "OMDB", Name: "迪拜国际机场", NameEn: "Dubai International", City: "迪拜", Country: "AE", Region: "Middle East", Type: "mixed", Tier: 1, Latitude: 25.2532, Longitude: 55.3657, Timezone: "Asia/Dubai", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 3, RunwayCount: 2, CargoTerminals: 3, AnnualCargoTons: 260},
		{IATACode: "DWC", ICAOCode: "OMDW", Name: "迪拜世界中心机场", NameEn: "Dubai World Central", City: "迪拜", Country: "AE", Region: "Middle East", Type: "cargo", Tier: 2, Latitude: 24.8966, Longitude: 55.1614, Timezone: "Asia/Dubai", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 1, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 95},
		{IATACode: "DOH", ICAOCode: "OTHH", Name: "多哈哈马德机场", NameEn: "Doha Hamad", City: "多哈", Country: "QA", Region: "Middle East", Type: "mixed", Tier: 1, Latitude: 25.2731, Longitude: 51.6081, Timezone: "Asia/Qatar", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 210},

		// 欧洲
		{IATACode: "FRA", ICAOCode: "EDDF", Name: "法兰克福机场", NameEn: "Frankfurt Airport", City: "法兰克福", Country: "DE", Region: "Europe", Type: "mixed", Tier: 1, Latitude: 50.0379, Longitude: 8.5622, Timezone: "Europe/Berlin", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 3, RunwayCount: 4, CargoTerminals: 2, AnnualCargoTons: 210},
		{IATACode: "AMS", ICAOCode: "EHAM", Name: "阿姆斯特丹史基浦机场", NameEn: "Amsterdam Schiphol", City: "阿姆斯特丹", Country: "NL", Region: "Europe", Type: "mixed", Tier: 1, Latitude: 52.3105, Longitude: 4.7683, Timezone: "Europe/Amsterdam", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 3, RunwayCount: 6, CargoTerminals: 2, AnnualCargoTons: 175},
		{IATACode: "CDG", ICAOCode: "LFPG", Name: "巴黎戴高乐机场", NameEn: "Paris Charles de Gaulle", City: "巴黎", Country: "FR", Region: "Europe", Type: "mixed", Tier: 1, Latitude: 49.0097, Longitude: 2.5479, Timezone: "Europe/Paris", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 3, RunwayCount: 4, CargoTerminals: 2, AnnualCargoTons: 205},
		{IATACode: "LHR", ICAOCode: "EGLL", Name: "伦敦希思罗机场", NameEn: "London Heathrow", City: "伦敦", Country: "GB", Region: "Europe", Type: "mixed", Tier: 1, Latitude: 51.4700, Longitude: -0.4543, Timezone: "Europe/London", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 4, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 155},
		{IATACode: "LEJ", ICAOCode: "EDDP", Name: "莱比锡/哈雷机场", NameEn: "Leipzig/Halle", City: "莱比锡", Country: "DE", Region: "Europe", Type: "cargo", Tier: 2, Latitude: 51.4324, Longitude: 12.2416, Timezone: "Europe/Berlin", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 1, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 130},
		{IATACode: "LGG", ICAOCode: "EBLG", Name: "列日机场", NameEn: "Liege Airport", City: "列日", Country: "BE", Region: "Europe", Type: "cargo", Tier: 2, Latitude: 50.6374, Longitude: 5.4432, Timezone: "Europe/Brussels", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 1, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 110},

		// 北美
		{IATACode: "MEM", ICAOCode: "KMEM", Name: "孟菲斯国际机场", NameEn: "Memphis International", City: "孟菲斯", Country: "US", Region: "North America", Type: "cargo", Tier: 1, Latitude: 35.0421, Longitude: -89.9792, Timezone: "America/Chicago", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 1, RunwayCount: 4, CargoTerminals: 3, AnnualCargoTons: 450},
		{IATACode: "LAX", ICAOCode: "KLAX", Name: "洛杉矶国际机场", NameEn: "Los Angeles International", City: "洛杉矶", Country: "US", Region: "North America", Type: "mixed", Tier: 1, Latitude: 33.9416, Longitude: -118.4085, Timezone: "America/Los_Angeles", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 4, RunwayCount: 4, CargoTerminals: 2, AnnualCargoTons: 245},
		{IATACode: "JFK", ICAOCode: "KJFK", Name: "纽约肯尼迪机场", NameEn: "New York JFK", City: "纽约", Country: "US", Region: "North America", Type: "mixed", Tier: 1, Latitude: 40.6413, Longitude: -73.7781, Timezone: "America/New_York", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 4, RunwayCount: 4, CargoTerminals: 2, AnnualCargoTons: 140},
		{IATACode: "ORD", ICAOCode: "KORD", Name: "芝加哥奥黑尔机场", NameEn: "Chicago O'Hare", City: "芝加哥", Country: "US", Region: "North America", Type: "mixed", Tier: 1, Latitude: 41.9742, Longitude: -87.9073, Timezone: "America/Chicago", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 3, RunwayCount: 8, CargoTerminals: 2, AnnualCargoTons: 185},
		{IATACode: "MIA", ICAOCode: "KMIA", Name: "迈阿密国际机场", NameEn: "Miami International", City: "迈阿密", Country: "US", Region: "North America", Type: "mixed", Tier: 1, Latitude: 25.7959, Longitude: -80.2870, Timezone: "America/New_York", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 3, RunwayCount: 4, CargoTerminals: 2, AnnualCargoTons: 210},
		{IATACode: "ANC", ICAOCode: "PANC", Name: "安克雷奇机场", NameEn: "Anchorage Ted Stevens", City: "安克雷奇", Country: "US", Region: "North America", Type: "cargo", Tier: 1, Latitude: 61.1743, Longitude: -149.9962, Timezone: "America/Anchorage", GeofenceKM: 15, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 1, RunwayCount: 3, CargoTerminals: 2, AnnualCargoTons: 280},
		{IATACode: "CVG", ICAOCode: "KCVG", Name: "辛辛那提/北肯塔基机场", NameEn: "Cincinnati/Northern Kentucky", City: "辛辛那提", Country: "US", Region: "North America", Type: "cargo", Tier: 1, Latitude: 39.0489, Longitude: -84.6678, Timezone: "America/New_York", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 1, RunwayCount: 4, CargoTerminals: 2, AnnualCargoTons: 120},
		{IATACode: "SDF", ICAOCode: "KSDF", Name: "路易斯维尔机场", NameEn: "Louisville Muhammad Ali International", City: "路易斯维尔", Country: "US", Region: "North America", Type: "cargo", Tier: 1, Latitude: 38.1744, Longitude: -85.7360, Timezone: "America/Kentucky/Louisville", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 1, RunwayCount: 3, CargoTerminals: 2, AnnualCargoTons: 280},
		{IATACode: "YYZ", ICAOCode: "CYYZ", Name: "多伦多皮尔逊机场", NameEn: "Toronto Pearson", City: "多伦多", Country: "CA", Region: "North America", Type: "mixed", Tier: 1, Latitude: 43.6777, Longitude: -79.6248, Timezone: "America/Toronto", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 3, RunwayCount: 5, CargoTerminals: 2, AnnualCargoTons: 55},

		// 南美
		{IATACode: "GRU", ICAOCode: "SBGR", Name: "圣保罗瓜鲁柳斯机场", NameEn: "Sao Paulo Guarulhos", City: "圣保罗", Country: "BR", Region: "South America", Type: "mixed", Tier: 1, Latitude: -23.4356, Longitude: -46.4731, Timezone: "America/Sao_Paulo", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 3, CongestionLevel: 3, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 52},
		{IATACode: "VCP", ICAOCode: "SBKP", Name: "坎皮纳斯维拉科波斯机场", NameEn: "Campinas Viracopos", City: "坎皮纳斯", Country: "BR", Region: "South America", Type: "cargo", Tier: 2, Latitude: -23.0074, Longitude: -47.1345, Timezone: "America/Sao_Paulo", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 1, RunwayCount: 1, CargoTerminals: 1, AnnualCargoTons: 35},
		{IATACode: "BOG", ICAOCode: "SKBO", Name: "波哥大埃尔多拉多机场", NameEn: "Bogota El Dorado", City: "波哥大", Country: "CO", Region: "South America", Type: "mixed", Tier: 2, Latitude: 4.7016, Longitude: -74.1469, Timezone: "America/Bogota", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 75},

		// 澳洲
		{IATACode: "SYD", ICAOCode: "YSSY", Name: "悉尼金斯福德史密斯机场", NameEn: "Sydney Kingsford Smith", City: "悉尼", Country: "AU", Region: "Oceania", Type: "mixed", Tier: 1, Latitude: -33.9399, Longitude: 151.1753, Timezone: "Australia/Sydney", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 5, CongestionLevel: 3, RunwayCount: 3, CargoTerminals: 2, AnnualCargoTons: 52},
		{IATACode: "MEL", ICAOCode: "YMML", Name: "墨尔本图拉马林机场", NameEn: "Melbourne Tullamarine", City: "墨尔本", Country: "AU", Region: "Oceania", Type: "mixed", Tier: 2, Latitude: -37.6690, Longitude: 144.8410, Timezone: "Australia/Melbourne", GeofenceKM: 10, IsCargoHub: false, CustomsEfficiency: 5, CongestionLevel: 2, RunwayCount: 2, CargoTerminals: 1, AnnualCargoTons: 35},

		// 印度
		{IATACode: "DEL", ICAOCode: "VIDP", Name: "德里英迪拉甘地机场", NameEn: "Delhi Indira Gandhi", City: "德里", Country: "IN", Region: "South Asia", Type: "mixed", Tier: 1, Latitude: 28.5562, Longitude: 77.1000, Timezone: "Asia/Kolkata", GeofenceKM: 12, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 3, RunwayCount: 3, CargoTerminals: 2, AnnualCargoTons: 105},
		{IATACode: "BOM", ICAOCode: "VABB", Name: "孟买贾特拉帕蒂希瓦吉机场", NameEn: "Mumbai Chhatrapati Shivaji", City: "孟买", Country: "IN", Region: "South Asia", Type: "mixed", Tier: 1, Latitude: 19.0896, Longitude: 72.8656, Timezone: "Asia/Kolkata", GeofenceKM: 10, IsCargoHub: true, CustomsEfficiency: 4, CongestionLevel: 3, RunwayCount: 2, CargoTerminals: 2, AnnualCargoTons: 90},
	}

	for _, airport := range airports {
		var existing models.Airport
		if err := db.Where("iata_code = ?", airport.IATACode).First(&existing).Error; err == nil {
			continue // 已存在，跳过
		}

		if err := db.Create(&airport).Error; err != nil {
			log.Printf("创建机场失败 %s: %v", airport.IATACode, err)
		} else {
			log.Printf("已创建机场: %s (%s)", airport.Name, airport.IATACode)
		}
	}

	return nil
}
