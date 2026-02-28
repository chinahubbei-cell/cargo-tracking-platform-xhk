package services

import (
	"errors"
	"math"
	"sort"
	"sync"

	"gorm.io/gorm"

	"trackcard-server/models"
)

// 全局GeocodingService单例，复用缓存
var (
	globalGeocodingService *GeocodingService
	geocodingOnce          sync.Once
)

// getGeocodingService 获取全局单例
func getGeocodingService() *GeocodingService {
	geocodingOnce.Do(func() {
		globalGeocodingService = NewGeocodingService()
	})
	return globalGeocodingService
}

// RoutePlannerService 路径规划服务
type RoutePlannerService struct {
	db *gorm.DB
}

// NewRoutePlannerService 创建路径规划服务
func NewRoutePlannerService(db *gorm.DB) *RoutePlannerService {
	return &RoutePlannerService{db: db}
}

// CalculateRouteRequest 计算路径请求
type CalculateRouteRequest struct {
	Origin        string  `json:"origin"`         // 发货地址
	Destination   string  `json:"destination"`    // 收货地址
	TransportType string  `json:"transport_type"` // 运输类型: sea(海运) / air(空运) / land(陆运) / multimodal(多式联运)
	TransportMode string  `json:"transport_mode"` // 运输模式: fcl(整柜) / lcl(零担) / ftl(整车)
	ContainerType string  `json:"container_type"` // 柜型: 20GP / 40GP / 40HQ (FCL模式)
	WeightKG      float64 `json:"weight_kg"`      // 货物重量(KG)
	VolumeCBM     float64 `json:"volume_cbm"`     // 货物体积(CBM，LCL模式)
	Quantity      int     `json:"quantity"`       // 件数(LCL模式)
	CargoType     string  `json:"cargo_type"`     // 货物类型: general/dangerous/cold_chain
	Currency      string  `json:"currency"`       // 货币单位: CNY(默认) / USD
}

// 运费计算常量
const (
	USD_TO_CNY_RATE = 7.25 // 美元对人民币汇率

	// FCL 固定费用 (USD)
	FCLFirstMileCostUSD = 300.0
	FCLLastMileCostUSD  = 250.0

	// LCL 费率 (USD/CBM)
	LCLFirstMileRatePerCBM = 30.0
	LCLLastMileRatePerCBM  = 25.0

	// 最小计费值
	MinChargeableCBM      = 0.1   // 最小计费体积
	MinChargeableWeightKG = 1.0   // 最小计费重量
	WeightToCBMDivisor    = 500.0 // 重量转体积除数
)

// convertToCurrency 货币转换
// 内部计算使用USD，根据请求的货币单位转换返回值
func convertToCurrency(usdAmount float64, currency string) float64 {
	if currency == "USD" {
		return usdAmount
	}
	// 默认转换为人民币(CNY)
	return usdAmount * USD_TO_CNY_RATE
}

// CalculateRouteResponse 计算路径响应
type CalculateRouteResponse struct {
	Origin      string                       `json:"origin"`
	Destination string                       `json:"destination"`
	Routes      []models.RouteRecommendation `json:"routes"`
	Currency    string                       `json:"currency"` // 返回的货币单位
}

// CalculateRoutes 计算推荐路径
// 根据运输类型分发到不同的规划算法
func (s *RoutePlannerService) CalculateRoutes(req CalculateRouteRequest) (*CalculateRouteResponse, error) {
	// 默认运输类型为海运
	if req.TransportType == "" {
		req.TransportType = "sea"
	}

	var routes []models.RouteRecommendation
	var err error

	// 根据运输类型分发到不同的规划算法
	switch req.TransportType {
	case "air":
		routes, err = s.calculateAirRoutes(req)
	case "land":
		routes, err = s.calculateLandRoutes(req)
	case "multimodal":
		routes, err = s.calculateMultimodalRoutes(req)
	default: // "sea" 或默认
		routes, err = s.calculateSeaRoutes(req)
	}

	if err != nil {
		return nil, err
	}

	return &CalculateRouteResponse{
		Origin:      req.Origin,
		Destination: req.Destination,
		Routes:      routes,
		Currency:    req.Currency,
	}, nil
}

// calculateSeaRoutes 海运路线规划
func (s *RoutePlannerService) calculateSeaRoutes(req CalculateRouteRequest) ([]models.RouteRecommendation, error) {
	// Step 1: 根据地址检测国家
	originCountry := s.detectCountryFromAddress(req.Origin)
	destCountry := s.detectCountryFromAddress(req.Destination)

	// Step 2: 查找起运港和目的港
	polPorts, err := s.findNearestPorts(req.Origin, "seaport", 3)
	if err != nil || len(polPorts) == 0 {
		polPorts = s.getDefaultPorts(originCountry)
	}

	podPorts, err := s.findNearestPorts(req.Destination, "seaport", 3)
	if err != nil || len(podPorts) == 0 {
		podPorts = s.getDefaultPorts(destCountry)
	}

	var routes []models.RouteRecommendation

	// 生成最快路径
	fastestRoute := s.generateFastestRoute(polPorts, podPorts, req)
	if fastestRoute != nil {
		routes = append(routes, *fastestRoute)
	}

	// 生成经济路径
	cheapestRoute := s.generateCheapestRoute(polPorts, podPorts, req)
	if cheapestRoute != nil {
		routes = append(routes, *cheapestRoute)
	}

	// 生成安全路径
	safestRoute := s.generateSafestRoute(polPorts, podPorts, req)
	if safestRoute != nil {
		routes = append(routes, *safestRoute)
	}

	return routes, nil
}

// calculateAirRoutes 空运路线规划
func (s *RoutePlannerService) calculateAirRoutes(req CalculateRouteRequest) ([]models.RouteRecommendation, error) {
	// Step 1: 查找起飞机场 (优先货运枢纽)
	originAirports, err := s.findNearestAirports(req.Origin, 3)
	if err != nil || len(originAirports) == 0 {
		// 使用默认机场
		originAirports = s.getDefaultAirports(req.Origin)
	}

	// Step 2: 查找目的机场
	destAirports, err := s.findNearestAirports(req.Destination, 3)
	if err != nil || len(destAirports) == 0 {
		destAirports = s.getDefaultAirports(req.Destination)
	}

	var routes []models.RouteRecommendation

	// 生成空运线路
	if len(originAirports) > 0 && len(destAirports) > 0 {
		oa := originAirports[0]
		da := destAirports[0]

		airRoute := s.buildAirRoute(req, oa, da)
		if airRoute != nil {
			routes = append(routes, *airRoute)
		}
	}

	// 如果没有找到合适机场，生成默认空运路线
	if len(routes) == 0 {
		defaultRoute := s.generateAirRoute(req)
		if defaultRoute != nil {
			routes = append(routes, *defaultRoute)
		}
	}

	return routes, nil
}

// calculateLandRoutes 陆运路线规划
func (s *RoutePlannerService) calculateLandRoutes(req CalculateRouteRequest) ([]models.RouteRecommendation, error) {
	var routes []models.RouteRecommendation

	landRoute := s.generateLandRoute(req)
	if landRoute != nil {
		routes = append(routes, *landRoute)
	}

	return routes, nil
}

// calculateMultimodalRoutes 多式联运规划
func (s *RoutePlannerService) calculateMultimodalRoutes(req CalculateRouteRequest) ([]models.RouteRecommendation, error) {
	var routes []models.RouteRecommendation

	// 根据起止点智能推荐组合
	originCountry := s.detectCountryFromAddress(req.Origin)
	destCountry := s.detectCountryFromAddress(req.Destination)

	// 组合1: 陆运 + 海运 (适合内陆城市出海)
	landSeaRoute := s.buildLandSeaRoute(req, originCountry, destCountry)
	if landSeaRoute != nil {
		routes = append(routes, *landSeaRoute)
	}

	// 组合2: 陆运 + 空运 (适合紧急货物)
	landAirRoute := s.buildLandAirRoute(req, originCountry, destCountry)
	if landAirRoute != nil {
		routes = append(routes, *landAirRoute)
	}

	// 组合3: 海运 + 铁路 (适合中欧班列)
	seaRailRoute := s.buildSeaRailRoute(req, originCountry, destCountry)
	if seaRailRoute != nil {
		routes = append(routes, *seaRailRoute)
	}

	// 如果没有生成任何组合，降级使用海运
	if len(routes) == 0 {
		seaRoutes, _ := s.calculateSeaRoutes(req)
		routes = seaRoutes
	}

	return routes, nil
}

// findNearestPorts 根据地址查找最近的港口
func (s *RoutePlannerService) findNearestPorts(address string, portType string, limit int) ([]models.Port, error) {
	var ports []models.Port

	// 首先检测地址所在国家
	countryCode := s.detectCountryFromAddress(address)

	// 按国家查找港口
	query := s.db.Where("type = ? AND country = ?", portType, countryCode).
		Order("is_transit_hub DESC, customs_efficiency DESC").
		Limit(limit)

	if err := query.Find(&ports).Error; err != nil {
		return nil, err
	}

	// 如果该国家没有港口，查找同区域的港口
	if len(ports) == 0 {
		region := s.getRegionByCountry(countryCode)
		regionCountries := s.getCountriesByRegion(region)

		query = s.db.Where("type = ? AND country IN ?", portType, regionCountries).
			Order("is_transit_hub DESC, customs_efficiency DESC").
			Limit(limit)
		query.Find(&ports)
	}

	// 如果还是没有，返回全球主要港口
	if len(ports) == 0 {
		s.db.Where("type = ? AND is_transit_hub = ?", portType, true).
			Order("customs_efficiency DESC").
			Limit(limit).
			Find(&ports)
	}

	return ports, nil
}

// findNearestAirports 根据地址查找最近的机场
func (s *RoutePlannerService) findNearestAirports(address string, limit int) ([]models.Airport, error) {
	var airports []models.Airport
	countryCode := s.detectCountryFromAddress(address)

	// 1. 本国机场 (优先查找货运枢纽)
	query := s.db.Where("country = ?", countryCode).
		Order("is_cargo_hub DESC, tier ASC, annual_cargo_tons DESC").
		Limit(limit)

	if err := query.Find(&airports).Error; err != nil {
		return nil, err
	}

	// 2. 区域机场 (如果本国没有)
	if len(airports) == 0 {
		region := s.getRegionByCountry(countryCode)
		regionCountries := s.getCountriesByRegion(region)

		s.db.Where("country IN ?", regionCountries).
			Order("is_cargo_hub DESC, tier ASC").
			Limit(limit).Find(&airports)
	}

	// 3. 全球主要枢纽 (如果不幸没有)
	if len(airports) == 0 {
		s.db.Where("is_cargo_hub = ?", true).
			Order("annual_cargo_tons DESC").
			Limit(limit).Find(&airports)
	}

	return airports, nil
}

// getRegionByCountry 根据国家代码获取区域
func (s *RoutePlannerService) getRegionByCountry(countryCode string) string {
	regionMap := map[string]string{
		"CN": "East Asia", "JP": "East Asia", "KR": "East Asia", "TW": "East Asia", "HK": "East Asia",
		"SG": "Southeast Asia", "MY": "Southeast Asia", "TH": "Southeast Asia", "VN": "Southeast Asia",
		"ID": "Southeast Asia", "PH": "Southeast Asia",
		"IN": "South Asia", "PK": "South Asia", "BD": "South Asia", "LK": "South Asia",
		"US": "North America", "CA": "North America", "MX": "North America",
		"DE": "Europe", "FR": "Europe", "GB": "Europe", "NL": "Europe", "BE": "Europe",
		"ES": "Europe", "IT": "Europe", "PL": "Europe", "SE": "Europe", "GR": "Europe", "RU": "Europe",
		"AE": "Middle East", "SA": "Middle East", "EG": "Middle East", "OM": "Middle East", "IL": "Middle East", "TR": "Middle East",
		"BR": "South America", "AR": "South America", "CL": "South America", "PE": "South America", "PA": "South America",
		"ZA": "Africa", "MA": "Africa", "KE": "Africa", "NG": "Africa",
		"AU": "Oceania", "NZ": "Oceania",
	}
	if region, ok := regionMap[countryCode]; ok {
		return region
	}
	return "Other"
}

// getCountriesByRegion 根据区域获取国家列表
func (s *RoutePlannerService) getCountriesByRegion(region string) []string {
	regionCountries := map[string][]string{
		"East Asia":      {"CN", "JP", "KR", "TW", "HK"},
		"Southeast Asia": {"SG", "MY", "TH", "VN", "ID", "PH"},
		"South Asia":     {"IN", "PK", "BD", "LK"},
		"North America":  {"US", "CA", "MX"},
		"Europe":         {"DE", "FR", "GB", "NL", "BE", "ES", "IT", "PL", "SE", "GR"},
		"Middle East":    {"AE", "SA", "EG", "OM", "IL"},
		"South America":  {"BR", "AR", "CL", "PE", "PA"},
		"Africa":         {"ZA", "MA", "KE", "NG"},
		"Oceania":        {"AU", "NZ"},
	}
	if countries, ok := regionCountries[region]; ok {
		return countries
	}
	return []string{}
}

// getDefaultPorts 获取默认港口
func (s *RoutePlannerService) getDefaultPorts(country string) []models.Port {
	var ports []models.Port
	s.db.Where("country = ?", country).Order("is_transit_hub DESC").Limit(3).Find(&ports)

	// 如果数据库没有数据，返回硬编码的默认值
	if len(ports) == 0 {
		switch country {
		case "CN":
			ports = []models.Port{
				{ID: "default-cn-1", Code: "CNSZX", Name: "深圳盐田港", NameEn: "Shenzhen Yantian", Country: "CN", Latitude: 22.5655, Longitude: 114.2689, Type: "seaport"},
				{ID: "default-cn-2", Code: "CNSHA", Name: "上海港", NameEn: "Shanghai Port", Country: "CN", Latitude: 31.2304, Longitude: 121.4737, Type: "seaport"},
			}
		case "US":
			ports = []models.Port{
				{ID: "default-us-1", Code: "USLAX", Name: "洛杉矶港", NameEn: "Los Angeles", Country: "US", Latitude: 33.7361, Longitude: -118.2628, Type: "seaport"},
				{ID: "default-us-2", Code: "USNYC", Name: "纽约港", NameEn: "New York", Country: "US", Latitude: 40.6892, Longitude: -74.0445, Type: "seaport"},
				{ID: "default-us-3", Code: "USMIA", Name: "迈阿密港", NameEn: "Miami", Country: "US", Latitude: 25.7617, Longitude: -80.1918, Type: "seaport"},
			}
		case "DE":
			ports = []models.Port{
				{ID: "default-de-1", Code: "DEHAM", Name: "汉堡港", NameEn: "Hamburg", Country: "DE", Latitude: 53.5511, Longitude: 9.9937, Type: "seaport"},
			}
		case "NL", "EU":
			ports = []models.Port{
				{ID: "default-nl-1", Code: "NLRTM", Name: "鹿特丹港", NameEn: "Rotterdam", Country: "NL", Latitude: 51.9244, Longitude: 4.4777, Type: "seaport"},
				{ID: "default-nl-2", Code: "NLAMS", Name: "阿姆斯特丹港", NameEn: "Amsterdam", Country: "NL", Latitude: 52.3676, Longitude: 4.9041, Type: "seaport"},
			}
		case "BE":
			ports = []models.Port{
				{ID: "default-be-1", Code: "BEANR", Name: "安特卫普港", NameEn: "Antwerp", Country: "BE", Latitude: 51.2194, Longitude: 4.4025, Type: "seaport"},
			}
		case "FR":
			ports = []models.Port{
				{ID: "default-fr-1", Code: "FRLEH", Name: "勒阿弗尔港", NameEn: "Le Havre", Country: "FR", Latitude: 49.4938, Longitude: 0.1077, Type: "seaport"},
				{ID: "default-fr-2", Code: "FRMRS", Name: "马赛港", NameEn: "Marseille", Country: "FR", Latitude: 43.2965, Longitude: 5.3698, Type: "seaport"},
			}
		case "JP":
			ports = []models.Port{
				{ID: "default-jp-1", Code: "JPTYO", Name: "东京港", NameEn: "Tokyo", Country: "JP", Latitude: 35.6762, Longitude: 139.6503, Type: "seaport"},
			}
		case "KR":
			ports = []models.Port{
				{ID: "default-kr-1", Code: "KRPUS", Name: "釜山港", NameEn: "Busan", Country: "KR", Latitude: 35.1796, Longitude: 129.0756, Type: "seaport"},
			}
		case "SG":
			ports = []models.Port{
				{ID: "default-sg-1", Code: "SGSIN", Name: "新加坡港", NameEn: "Singapore", Country: "SG", Latitude: 1.2644, Longitude: 103.8201, Type: "seaport"},
			}
		case "MY":
			ports = []models.Port{
				{ID: "default-my-1", Code: "MYPKG", Name: "巴生港", NameEn: "Port Klang", Country: "MY", Latitude: 3.0319, Longitude: 101.3685, Type: "seaport"},
				{ID: "default-my-2", Code: "MYPEN", Name: "槟城港", NameEn: "Penang", Country: "MY", Latitude: 5.4164, Longitude: 100.3467, Type: "seaport"},
			}
		case "TH":
			ports = []models.Port{
				{ID: "default-th-1", Code: "THLCH", Name: "林查班港", NameEn: "Laem Chabang", Country: "TH", Latitude: 13.0827, Longitude: 100.8833, Type: "seaport"},
			}
		case "VN":
			ports = []models.Port{
				{ID: "default-vn-1", Code: "VNSGN", Name: "胡志明港", NameEn: "Ho Chi Minh", Country: "VN", Latitude: 10.7769, Longitude: 106.7009, Type: "seaport"},
			}
		case "PH":
			ports = []models.Port{
				{ID: "default-ph-1", Code: "PHMNL", Name: "马尼拉港", NameEn: "Manila", Country: "PH", Latitude: 14.5995, Longitude: 120.9842, Type: "seaport"},
			}
		case "AU":
			ports = []models.Port{
				{ID: "default-au-1", Code: "AUSYD", Name: "悉尼港", NameEn: "Sydney", Country: "AU", Latitude: -33.8688, Longitude: 151.2093, Type: "seaport"},
			}
		case "GB", "UK":
			ports = []models.Port{
				{ID: "default-gb-1", Code: "GBFXT", Name: "费利克斯托港", NameEn: "Felixstowe", Country: "GB", Latitude: 51.9632, Longitude: 1.3514, Type: "seaport"},
			}
		case "ID":
			ports = []models.Port{
				{ID: "default-id-1", Code: "IDJKT", Name: "雅加达港", NameEn: "Jakarta", Country: "ID", Latitude: -6.0884, Longitude: 106.8836, Type: "seaport"},
			}
		case "PK":
			ports = []models.Port{
				{ID: "default-pk-1", Code: "PKKHI", Name: "卡拉奇港", NameEn: "Karachi", Country: "PK", Latitude: 24.8607, Longitude: 67.0011, Type: "seaport"},
				{ID: "default-pk-2", Code: "PKGWD", Name: "瓜达尔港", NameEn: "Gwadar", Country: "PK", Latitude: 25.1264, Longitude: 62.3225, Type: "seaport"},
			}
		case "IN":
			ports = []models.Port{
				{ID: "default-in-1", Code: "INNSA", Name: "孟买港", NameEn: "Mumbai", Country: "IN", Latitude: 18.9400, Longitude: 72.8347, Type: "seaport"},
				{ID: "default-in-2", Code: "INMAA", Name: "金奈港", NameEn: "Chennai", Country: "IN", Latitude: 13.0827, Longitude: 80.2707, Type: "seaport"},
			}
		case "AE":
			ports = []models.Port{
				{ID: "default-ae-1", Code: "AEJEA", Name: "迪拜杰贝勒阿里港", NameEn: "Jebel Ali", Country: "AE", Latitude: 25.0657, Longitude: 55.0272, Type: "seaport"},
			}
		case "SA":
			ports = []models.Port{
				{ID: "default-sa-1", Code: "SAJED", Name: "吉达港", NameEn: "Jeddah", Country: "SA", Latitude: 21.4858, Longitude: 39.1925, Type: "seaport"},
			}
		case "RU":
			ports = []models.Port{
				{ID: "default-ru-1", Code: "RULED", Name: "圣彼得堡港", NameEn: "St. Petersburg", Country: "RU", Latitude: 59.9343, Longitude: 30.3351, Type: "seaport"},
				{ID: "default-ru-2", Code: "RUVVO", Name: "符拉迪沃斯托克港", NameEn: "Vladivostok", Country: "RU", Latitude: 43.1198, Longitude: 131.8869, Type: "seaport"},
			}
		case "BR":
			ports = []models.Port{
				{ID: "default-br-1", Code: "BRSSZ", Name: "桑托斯港", NameEn: "Santos", Country: "BR", Latitude: -23.9608, Longitude: -46.3336, Type: "seaport"},
				{ID: "default-br-2", Code: "BRRIG", Name: "里约港", NameEn: "Rio de Janeiro", Country: "BR", Latitude: -22.9068, Longitude: -43.1729, Type: "seaport"},
			}
		case "EG":
			ports = []models.Port{
				{ID: "default-eg-1", Code: "EGPSD", Name: "塞得港", NameEn: "Port Said", Country: "EG", Latitude: 31.2653, Longitude: 32.3019, Type: "seaport"},
				{ID: "default-eg-2", Code: "EGALX", Name: "亚历山大港", NameEn: "Alexandria", Country: "EG", Latitude: 31.2001, Longitude: 29.9187, Type: "seaport"},
			}
		case "ES":
			ports = []models.Port{
				{ID: "default-es-1", Code: "ESALG", Name: "阿尔赫西拉斯港", NameEn: "Algeciras", Country: "ES", Latitude: 36.1408, Longitude: -5.4534, Type: "seaport"},
				{ID: "default-es-2", Code: "ESBCN", Name: "巴塞罗那港", NameEn: "Barcelona", Country: "ES", Latitude: 41.3851, Longitude: 2.1734, Type: "seaport"},
			}
		case "IT":
			ports = []models.Port{
				{ID: "default-it-1", Code: "ITGOA", Name: "热那亚港", NameEn: "Genoa", Country: "IT", Latitude: 44.4056, Longitude: 8.9463, Type: "seaport"},
			}
		case "ZA":
			ports = []models.Port{
				{ID: "default-za-1", Code: "ZADUR", Name: "德班港", NameEn: "Durban", Country: "ZA", Latitude: -29.8587, Longitude: 31.0218, Type: "seaport"},
			}
		case "MX":
			ports = []models.Port{
				{ID: "default-mx-1", Code: "MXMAN", Name: "曼萨尼约港", NameEn: "Manzanillo", Country: "MX", Latitude: 19.0522, Longitude: -104.3118, Type: "seaport"},
			}
		case "TR":
			ports = []models.Port{
				{ID: "default-tr-1", Code: "TRIST", Name: "伊斯坦布尔港", NameEn: "Istanbul", Country: "TR", Latitude: 41.0082, Longitude: 28.9784, Type: "seaport"},
			}
		// 新西兰
		case "NZ":
			ports = []models.Port{
				{ID: "default-nz-1", Code: "NZAKL", Name: "奥克兰港", NameEn: "Auckland", Country: "NZ", Latitude: -36.8485, Longitude: 174.7633, Type: "seaport"},
			}
		// 加拿大
		case "CA":
			ports = []models.Port{
				{ID: "default-ca-1", Code: "CAVAN", Name: "温哥华港", NameEn: "Vancouver", Country: "CA", Latitude: 49.2827, Longitude: -123.1207, Type: "seaport"},
			}
		// 阿根廷
		case "AR":
			ports = []models.Port{
				{ID: "default-ar-1", Code: "ARBUE", Name: "布宜诺斯艾利斯港", NameEn: "Buenos Aires", Country: "AR", Latitude: -34.6037, Longitude: -58.3816, Type: "seaport"},
			}
		// 智利
		case "CL":
			ports = []models.Port{
				{ID: "default-cl-1", Code: "CLSAI", Name: "圣安东尼奥港", NameEn: "San Antonio", Country: "CL", Latitude: -33.5950, Longitude: -71.6211, Type: "seaport"},
			}
		// 希腊
		case "GR":
			ports = []models.Port{
				{ID: "default-gr-1", Code: "GRPIR", Name: "比雷埃夫斯港", NameEn: "Piraeus", Country: "GR", Latitude: 37.9475, Longitude: 23.6422, Type: "seaport"},
			}
		// 波兰
		case "PL":
			ports = []models.Port{
				{ID: "default-pl-1", Code: "PLGDN", Name: "格但斯克港", NameEn: "Gdansk", Country: "PL", Latitude: 54.3520, Longitude: 18.6466, Type: "seaport"},
			}
		default:
			// 使用新加坡作为全球中转港（世界最大中转港之一），为未配置国家提供合理回退
			ports = []models.Port{
				{ID: "default-global-1", Code: "SGSIN", Name: "新加坡港", NameEn: "Singapore", Country: "SG", Latitude: 1.2644, Longitude: 103.8201, Type: "seaport"},
			}
		}
	}

	return ports
}

// getAddressCoordinates 根据地址获取坐标 (简化实现)
func (s *RoutePlannerService) getAddressCoordinates(address string) (lat, lng float64) {
	// 简化的地址-坐标映射，实际应调用地理编码API
	addressCoords := map[string][2]float64{
		// 中国城市 - 一二线城市
		"深圳": {22.5431, 114.0579}, "广东省深圳市": {22.5431, 114.0579},
		"东莞": {23.0430, 113.7633}, "广东省东莞市": {23.0430, 113.7633},
		"广州": {23.1291, 113.2644}, "广东省广州市": {23.1291, 113.2644},
		"上海": {31.2304, 121.4737}, "上海市": {31.2304, 121.4737},
		"北京": {39.9042, 116.4074}, "北京市": {39.9042, 116.4074},
		"杭州": {30.2741, 120.1551}, "浙江省杭州市": {30.2741, 120.1551},
		"宁波": {29.8683, 121.5440}, "浙江省宁波市": {29.8683, 121.5440},
		"武汉": {30.5928, 114.3055}, "湖北省武汉市": {30.5928, 114.3055}, "湖北": {30.5928, 114.3055},
		"成都": {30.5728, 104.0668}, "四川省成都市": {30.5728, 104.0668}, "四川": {30.5728, 104.0668},
		"重庆": {29.5630, 106.5516}, "重庆市": {29.5630, 106.5516},
		"西安": {34.3416, 108.9398}, "陕西省西安市": {34.3416, 108.9398}, "陕西": {34.3416, 108.9398},
		"长沙": {28.2282, 112.9388}, "湖南省长沙市": {28.2282, 112.9388}, "湖南": {28.2282, 112.9388},
		"郑州": {34.7466, 113.6254}, "河南省郑州市": {34.7466, 113.6254}, "河南": {34.7466, 113.6254},
		"济南": {36.6512, 117.1201}, "山东省济南市": {36.6512, 117.1201},
		"青岛": {36.0671, 120.3826}, "山东省青岛市": {36.0671, 120.3826}, "山东": {36.0671, 120.3826},
		"南京": {32.0603, 118.7969}, "江苏省南京市": {32.0603, 118.7969}, "江苏": {32.0603, 118.7969},
		"苏州": {31.2989, 120.5853}, "江苏省苏州市": {31.2989, 120.5853},
		"天津": {39.0842, 117.2010}, "天津市": {39.0842, 117.2010},
		"厦门": {24.4798, 118.0894}, "福建省厦门市": {24.4798, 118.0894}, "福建": {24.4798, 118.0894},
		"福州": {26.0745, 119.2965}, "福建省福州市": {26.0745, 119.2965},
		// 中国城市 - 三四线城市
		"贵阳": {26.6470, 106.6302}, "贵州省贵阳市": {26.6470, 106.6302}, "贵州": {26.6470, 106.6302},
		"铜仁": {27.7183, 109.1850}, "贵州省铜仁市": {27.7183, 109.1850}, "碧江": {27.7183, 109.1850},
		"昆明": {25.0389, 102.7183}, "云南省昆明市": {25.0389, 102.7183}, "云南": {25.0389, 102.7183},
		"南宁": {22.8170, 108.3665}, "广西南宁市": {22.8170, 108.3665}, "广西": {22.8170, 108.3665},
		"兰州": {36.0611, 103.8343}, "甘肃省兰州市": {36.0611, 103.8343}, "甘肃": {36.0611, 103.8343},
		"乌鲁木齐": {43.8256, 87.6168}, "新疆乌鲁木齐": {43.8256, 87.6168}, "新疆": {43.8256, 87.6168},
		"拉萨": {29.6500, 91.1000}, "西藏拉萨市": {29.6500, 91.1000}, "西藏": {29.6500, 91.1000},
		"海口": {20.0440, 110.1999}, "海南省海口市": {20.0440, 110.1999}, "海南": {20.0440, 110.1999},
		"沈阳": {41.8057, 123.4315}, "辽宁省沈阳市": {41.8057, 123.4315}, "辽宁": {41.8057, 123.4315},
		"大连": {38.9140, 121.6147}, "辽宁省大连市": {38.9140, 121.6147},
		"长春": {43.8171, 125.3235}, "吉林省长春市": {43.8171, 125.3235}, "吉林": {43.8171, 125.3235},
		"哈尔滨": {45.8038, 126.5350}, "黑龙江哈尔滨": {45.8038, 126.5350}, "黑龙江": {45.8038, 126.5350},
		"太原": {37.8706, 112.5489}, "山西省太原市": {37.8706, 112.5489}, "山西": {37.8706, 112.5489},
		"石家庄": {38.0428, 114.5149}, "河北省石家庄市": {38.0428, 114.5149}, "河北": {38.0428, 114.5149},
		"合肥": {31.8206, 117.2272}, "安徽省合肥市": {31.8206, 117.2272}, "安徽": {31.8206, 117.2272},
		"南昌": {28.6820, 115.8579}, "江西省南昌市": {28.6820, 115.8579}, "江西": {28.6820, 115.8579},
		"呼和浩特": {40.8420, 111.7500}, "内蒙古呼和浩特": {40.8420, 111.7500}, "内蒙古": {40.8420, 111.7500},
		"银川": {38.4872, 106.2309}, "宁夏银川市": {38.4872, 106.2309}, "宁夏": {38.4872, 106.2309},
		"西宁": {36.6171, 101.7782}, "青海省西宁市": {36.6171, 101.7782}, "青海": {36.6171, 101.7782},
		// 美国城市
		"洛杉矶": {34.0522, -118.2437}, "美国洛杉矶": {34.0522, -118.2437}, "Los Angeles": {34.0522, -118.2437},
		"纽约": {40.7128, -74.0060}, "美国纽约": {40.7128, -74.0060}, "New York": {40.7128, -74.0060},
		"芝加哥": {41.8781, -87.6298}, "Chicago": {41.8781, -87.6298},
		"迈阿密": {25.7617, -80.1918}, "美国迈阿密": {25.7617, -80.1918}, "Miami": {25.7617, -80.1918},
		"旧金山": {37.7749, -122.4194}, "美国旧金山": {37.7749, -122.4194}, "San Francisco": {37.7749, -122.4194},
		"西雅图": {47.6062, -122.3321}, "美国西雅图": {47.6062, -122.3321}, "Seattle": {47.6062, -122.3321},
		"休斯敦": {29.7604, -95.3698}, "Houston": {29.7604, -95.3698},
		"达拉斯": {32.7767, -96.7970}, "Dallas": {32.7767, -96.7970},
		// 欧洲城市
		"汉堡": {53.5511, 9.9937}, "德国汉堡": {53.5511, 9.9937}, "Hamburg": {53.5511, 9.9937},
		"鹿特丹": {51.9244, 4.4777}, "荷兰鹿特丹": {51.9244, 4.4777}, "Rotterdam": {51.9244, 4.4777},
		"伦敦": {51.5074, -0.1278}, "英国伦敦": {51.5074, -0.1278}, "London": {51.5074, -0.1278},
		"巴黎": {48.8566, 2.3522}, "法国巴黎": {48.8566, 2.3522}, "Paris": {48.8566, 2.3522},
		"马赛": {43.2965, 5.3698}, "法国马赛": {43.2965, 5.3698}, "Marseille": {43.2965, 5.3698},
		"勒阿弗尔": {49.4938, 0.1077}, "Le Havre": {49.4938, 0.1077},
		"马德里": {40.4168, -3.7038}, "西班牙马德里": {40.4168, -3.7038}, "Madrid": {40.4168, -3.7038},
		"罗马": {41.9028, 12.4964}, "意大利罗马": {41.9028, 12.4964}, "Rome": {41.9028, 12.4964},
		// 亚洲城市
		"东京": {35.6762, 139.6503}, "日本东京": {35.6762, 139.6503},
		"釜山": {35.1796, 129.0756}, "韩国釜山": {35.1796, 129.0756},
		"新加坡": {1.3521, 103.8198},
		"雅加达": {-6.2088, 106.8456}, "印尼雅加达": {-6.2088, 106.8456},
		// 南亚城市
		"巴基斯坦": {30.3753, 69.3451}, "卡拉奇": {24.8607, 67.0011}, "伊斯兰堡": {33.6844, 73.0479},
		"孟买": {19.0760, 72.8777}, "印度孟买": {19.0760, 72.8777}, "新德里": {28.6139, 77.2090},
		"迪拜": {25.2048, 55.2708}, "阿联酋迪拜": {25.2048, 55.2708},
		"吉达": {21.4858, 39.1925}, "沙特吉达": {21.4858, 39.1925},
		// 澳洲城市
		"悉尼": {-33.8688, 151.2093}, "澳大利亚悉尼": {-33.8688, 151.2093},
		"墨尔本": {-37.8136, 144.9631},
	}

	// 模糊匹配
	for key, coords := range addressCoords {
		if contains(address, key) {
			return coords[0], coords[1]
		}
	}

	// 本地缓存未命中，尝试调用地理编码API
	geocodingService := getGeocodingService()
	result, err := geocodingService.Geocode(address)
	if err == nil && result != nil {
		return result.Lat, result.Lng
	}

	// API也失败，返回默认深圳坐标
	return 22.5431, 114.0579
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// detectCountryFromAddress 根据地址检测国家代码
func (s *RoutePlannerService) detectCountryFromAddress(address string) string {
	countryPatterns := map[string][]string{
		"CN": {"中国", "广东", "深圳", "上海", "北京", "杭州", "宁波", "东莞", "广州", "天津", "青岛", "厦门"},
		"US": {"美国", "USA", "洛杉矶", "纽约", "芝加哥", "迈阿密", "旧金山", "西雅图", "休斯敦", "达拉斯", "Los Angeles", "New York", "Chicago", "Miami", "San Francisco", "Seattle", "Houston", "Dallas"},
		"DE": {"德国", "Germany", "汉堡", "Hamburg", "柏林", "Berlin", "法兰克福", "Frankfurt"},
		"FR": {"法国", "France", "巴黎", "Paris", "马赛", "Marseille", "勒阿弗尔", "Le Havre"},
		"JP": {"日本", "Japan", "东京", "Tokyo", "大阪", "Osaka"},
		"KR": {"韩国", "Korea", "釜山", "Busan", "首尔", "Seoul"},
		"SG": {"新加坡", "Singapore"},
		"AU": {"澳大利亚", "Australia", "悉尼", "Sydney", "墨尔本"},
		"GB": {"英国", "UK", "Britain", "伦敦", "London"},
		"ID": {"印尼", "印度尼西亚", "Indonesia", "雅加达", "Jakarta"},
		"NL": {"荷兰", "Netherlands", "鹿特丹", "Rotterdam", "阿姆斯特丹", "Amsterdam"},
		"PK": {"巴基斯坦", "Pakistan", "卡拉奇", "Karachi", "伊斯兰堡", "Islamabad", "瓜达尔", "Gwadar"},
		"IN": {"印度", "India", "孟买", "Mumbai", "新德里", "New Delhi", "金奈", "Chennai"},
		"AE": {"阿联酋", "UAE", "迪拜", "Dubai", "杰贝勒阿里", "Jebel Ali"},
		"SA": {"沙特阿拉伯", "Saudi", "吉达", "Jeddah"},
		// 新增国家
		"RU": {"俄罗斯", "Russia", "莫斯科", "Moscow", "圣彼得堡", "St. Petersburg", "海参崴", "符拉迪沃斯托克", "Vladivostok"},
		"IT": {"意大利", "Italy", "罗马", "Rome", "米兰", "Milan"},
		"ES": {"西班牙", "Spain", "马德里", "Madrid", "巴塞罗那", "Barcelona"},
		"CA": {"加拿大", "Canada", "多伦多", "Toronto", "温哥华", "Vancouver"},
		"BR": {"巴西", "Brazil", "圣保罗", "São Paulo", "Sao Paulo"},
		"TH": {"泰国", "Thailand", "曼谷", "Bangkok"},
		"VN": {"越南", "Vietnam", "胡志明", "Ho Chi Minh", "河内", "Hanoi"},
		"MY": {"马来西亚", "Malaysia", "吉隆坡", "Kuala Lumpur"},
		"TR": {"土耳其", "Turkey", "伊斯坦布尔", "Istanbul"},
		"PL": {"波兰", "Poland", "华沙", "Warsaw"},
	}

	for code, patterns := range countryPatterns {
		for _, pattern := range patterns {
			if contains(address, pattern) {
				return code
			}
		}
	}

	// 本地匹配失败，尝试调用地理编码API
	geocodingService := getGeocodingService()
	result, err := geocodingService.Geocode(address)
	if err == nil && result != nil && result.CountryCode != "" {
		return result.CountryCode
	}

	return "CN" // 默认中国
}

// 航线区域定义
const (
	RouteRegionAsia    = "asia"
	RouteRegionNorthAm = "north_america"
	RouteRegionEurope  = "europe"
	RouteRegionOther   = "other"
)

// detectRouteRegion 根据目的港确定航线区域
func detectRouteRegion(countryCode string) string {
	asiaCountries := []string{"CN", "JP", "KR", "SG", "MY", "TH", "VN", "PH", "ID", "HK", "TW"}
	northAmCountries := []string{"US", "CA", "MX"}
	europeCountries := []string{"DE", "FR", "NL", "BE", "GB", "ES", "IT", "GR", "PL", "RU", "TR"}

	for _, c := range asiaCountries {
		if c == countryCode {
			return RouteRegionAsia
		}
	}
	for _, c := range northAmCountries {
		if c == countryCode {
			return RouteRegionNorthAm
		}
	}
	for _, c := range europeCountries {
		if c == countryCode {
			return RouteRegionEurope
		}
	}
	return RouteRegionOther
}

// calculateFCLPrice FCL整柜运费计算 (返回美元价格)
func calculateFCLPrice(containerType, routeRegion, cargoType string) float64 {
	// 基础价格表 (USD) - 按柜型和航线
	priceTable := map[string]map[string]float64{
		"20GP": {
			RouteRegionAsia:    1500,
			RouteRegionNorthAm: 2500,
			RouteRegionEurope:  2000,
			RouteRegionOther:   1800,
		},
		"40GP": {
			RouteRegionAsia:    2500,
			RouteRegionNorthAm: 4000,
			RouteRegionEurope:  3200,
			RouteRegionOther:   3000,
		},
		"40HQ": {
			RouteRegionAsia:    2800,
			RouteRegionNorthAm: 4500,
			RouteRegionEurope:  3600,
			RouteRegionOther:   3400,
		},
	}

	container := containerType
	if container == "" {
		container = "20GP"
	}

	basePrice := priceTable[container][routeRegion]
	if basePrice == 0 {
		basePrice = priceTable["20GP"][RouteRegionOther]
	}

	// 货物类型附加费
	switch cargoType {
	case "dangerous":
		basePrice *= 1.5 // 危险品加价50%
	case "cold_chain":
		basePrice *= 1.3 // 冷链加价30%
	}

	return basePrice
}

// calculateLCLPrice LCL零担运费计算 (返回美元价格)
func calculateLCLPrice(weightKG, volumeCBM float64, routeRegion, cargoType string) float64 {
	// 每CBM价格表 (USD)
	pricePerCBM := map[string]float64{
		RouteRegionAsia:    80,
		RouteRegionNorthAm: 150,
		RouteRegionEurope:  120,
		RouteRegionOther:   100,
	}

	// 最低收费
	minCharge := map[string]float64{
		RouteRegionAsia:    150,
		RouteRegionNorthAm: 300,
		RouteRegionEurope:  250,
		RouteRegionOther:   200,
	}

	// 计费重量: max(实际体积, 重量/WeightToCBMDivisor)，确保最小计费值
	chargeableCBM := math.Max(volumeCBM, MinChargeableCBM)
	weightCBM := weightKG / WeightToCBMDivisor
	if weightCBM > chargeableCBM {
		chargeableCBM = weightCBM
	}

	price := chargeableCBM * pricePerCBM[routeRegion]

	// 确保不低于最低收费
	if price < minCharge[routeRegion] {
		price = minCharge[routeRegion]
	}

	// 货物类型附加费
	switch cargoType {
	case "dangerous":
		price *= 1.5
	case "cold_chain":
		price *= 1.3
	}

	return price
}

// generateFastestRoute 生成最快路径
func (s *RoutePlannerService) generateFastestRoute(polPorts, podPorts []models.Port, req CalculateRouteRequest) *models.RouteRecommendation {
	if len(polPorts) == 0 || len(podPorts) == 0 {
		return nil
	}

	pol := polPorts[0]
	pod := podPorts[0]

	// 获取发货地和目的地坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 查找直航航线
	var line models.ShippingLine
	err := s.db.Where("pol_port_id = ? AND pod_port_id = ? AND active = true", pol.ID, pod.ID).
		Order("transit_days ASC").First(&line).Error

	transitDays := 14 // 默认天数
	carrier := "COSCO"

	// 获取航线区域
	destCountry := s.detectCountryFromAddress(req.Destination)
	routeRegion := detectRouteRegion(destCountry)

	// 根据运输模式计算干线费用
	var baseCost float64
	if req.TransportMode == "fcl" {
		// FCL整柜模式
		baseCost = calculateFCLPrice(req.ContainerType, routeRegion, req.CargoType)
	} else {
		// LCL零担模式（默认）
		baseCost = calculateLCLPrice(req.WeightKG, req.VolumeCBM, routeRegion, req.CargoType)
	}

	if err == nil {
		transitDays = line.TransitDays
		carrier = line.Carrier
		// 如果数据库有配置，可选择使用数据库价格
		// baseCost = line.BaseCost
	}

	// 计算头程和尾程费用（按运输模式）
	var firstMileCost, lastMileCost float64
	if req.TransportMode == "fcl" {
		firstMileCost = FCLFirstMileCostUSD
		lastMileCost = FCLLastMileCostUSD
	} else {
		// LCL按体积计费头程尾程
		chargeableCBM := math.Max(req.VolumeCBM, MinChargeableCBM)
		if req.WeightKG/WeightToCBMDivisor > chargeableCBM {
			chargeableCBM = req.WeightKG / WeightToCBMDivisor
		}
		if chargeableCBM < 1 {
			chargeableCBM = 1 // 最小1CBM
		}
		firstMileCost = chargeableCBM * LCLFirstMileRatePerCBM
		lastMileCost = chargeableCBM * LCLLastMileRatePerCBM
	}
	firstMileDays := 1
	lastMileDays := 3

	// 应用货币转换（默认CNY）
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	firstMileCost = convertToCurrency(firstMileCost, currency)
	baseCost = convertToCurrency(baseCost, currency)
	lastMileCost = convertToCurrency(lastMileCost, currency)

	return &models.RouteRecommendation{
		Type:      "fastest",
		Label:     "最快路径",
		TotalDays: firstMileDays + transitDays + lastMileDays,
		TotalCost: firstMileCost + baseCost + lastMileCost,
		Segments: []models.RouteSegment{
			{
				Type:       "first_mile",
				From:       req.Origin,
				To:         pol.Name,
				FromLat:    originLat,
				FromLng:    originLng,
				ToLat:      pol.Latitude,
				ToLng:      pol.Longitude,
				Mode:       "truck",
				Days:       firstMileDays,
				Cost:       firstMileCost,
				DistanceKM: 65,
			},
			{
				Type:    "line_haul",
				From:    pol.Code + " (" + pol.Name + ")",
				To:      pod.Code + " (" + pod.Name + ")",
				FromLat: pol.Latitude,
				FromLng: pol.Longitude,
				ToLat:   pod.Latitude,
				ToLng:   pod.Longitude,
				Mode:    "ocean",
				Carrier: carrier,
				Days:    transitDays,
				Cost:    baseCost,
			},
			{
				Type:       "last_mile",
				From:       pod.Name,
				To:         req.Destination,
				FromLat:    pod.Latitude,
				FromLng:    pod.Longitude,
				ToLat:      destLat,
				ToLng:      destLng,
				Mode:       "truck",
				Days:       lastMileDays,
				Cost:       lastMileCost,
				DistanceKM: 50,
			},
		},
	}
}

// generateCheapestRoute 生成经济路径
func (s *RoutePlannerService) generateCheapestRoute(polPorts, podPorts []models.Port, req CalculateRouteRequest) *models.RouteRecommendation {
	if len(polPorts) == 0 || len(podPorts) == 0 {
		return nil
	}

	pol := polPorts[0]
	pod := podPorts[0]

	// 获取发货地和目的地坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 查找中转航线 (通过中转枢纽)
	var transitHub models.Port
	s.db.Where("is_transit_hub = true").First(&transitHub)

	transitHubName := "釜山"
	if transitHub.ID != "" {
		transitHubName = transitHub.Name
	}

	// 应用货币转换
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	firstMileCost := convertToCurrency(600, currency)
	lineHaulCost := convertToCurrency(2200, currency)
	lastMileCost := convertToCurrency(400, currency)
	totalCost := firstMileCost + lineHaulCost + lastMileCost

	return &models.RouteRecommendation{
		Type:      "cheapest",
		Label:     "经济路径",
		TotalDays: 25,
		TotalCost: totalCost,
		Segments: []models.RouteSegment{
			{
				Type:       "first_mile",
				From:       req.Origin,
				To:         pol.Name,
				FromLat:    originLat,
				FromLng:    originLng,
				ToLat:      pol.Latitude,
				ToLng:      pol.Longitude,
				Mode:       "truck",
				Days:       1,
				Cost:       firstMileCost,
				DistanceKM: 65,
			},
			{
				Type:         "line_haul",
				From:         pol.Code + " (" + pol.Name + ")",
				To:           pod.Code + " (" + pod.Name + ")",
				FromLat:      pol.Latitude,
				FromLng:      pol.Longitude,
				ToLat:        pod.Latitude,
				ToLng:        pod.Longitude,
				Mode:         "ocean",
				Carrier:      "MSC",
				Days:         21,
				Cost:         lineHaulCost,
				TransitPorts: []string{transitHubName},
			},
			{
				Type:       "last_mile",
				From:       pod.Name,
				To:         req.Destination,
				FromLat:    pod.Latitude,
				FromLng:    pod.Longitude,
				ToLat:      destLat,
				ToLng:      destLng,
				Mode:       "truck",
				Days:       3,
				Cost:       lastMileCost,
				DistanceKM: 50,
			},
		},
	}
}

// generateSafestRoute 生成安全路径 (优先设备覆盖好的港口)
func (s *RoutePlannerService) generateSafestRoute(polPorts, podPorts []models.Port, req CalculateRouteRequest) *models.RouteRecommendation {
	if len(polPorts) == 0 || len(podPorts) == 0 {
		return nil
	}

	// 按清关效率排序选择最佳港口
	sort.Slice(polPorts, func(i, j int) bool {
		return polPorts[i].CustomsEfficiency > polPorts[j].CustomsEfficiency
	})

	pol := polPorts[0]
	pod := podPorts[0]

	// 获取发货地和目的地坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 应用货币转换
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	firstMileCost := convertToCurrency(800, currency)
	lineHaulCost := convertToCurrency(2800, currency)
	lastMileCost := convertToCurrency(400, currency)
	totalCost := firstMileCost + lineHaulCost + lastMileCost

	return &models.RouteRecommendation{
		Type:           "safest",
		Label:          "安全路径",
		TotalDays:      20,
		TotalCost:      totalCost,
		DeviceCoverage: "95%",
		Segments: []models.RouteSegment{
			{
				Type:       "first_mile",
				From:       req.Origin,
				To:         pol.Name,
				FromLat:    originLat,
				FromLng:    originLng,
				ToLat:      pol.Latitude,
				ToLng:      pol.Longitude,
				Mode:       "truck",
				Days:       1,
				Cost:       firstMileCost,
				DistanceKM: 65,
			},
			{
				Type:    "line_haul",
				From:    pol.Code + " (" + pol.Name + ")",
				To:      pod.Code + " (" + pod.Name + ")",
				FromLat: pol.Latitude,
				FromLng: pol.Longitude,
				ToLat:   pod.Latitude,
				ToLng:   pod.Longitude,
				Mode:    "ocean",
				Carrier: "COSCO",
				Days:    16,
				Cost:    lineHaulCost,
			},
			{
				Type:       "last_mile",
				From:       pod.Name,
				To:         req.Destination,
				FromLat:    pod.Latitude,
				FromLng:    pod.Longitude,
				ToLat:      destLat,
				ToLng:      destLng,
				Mode:       "truck",
				Days:       3,
				Cost:       lastMileCost,
				DistanceKM: 50,
			},
		},
	}
}

// generateAirRoute 生成空运路径
func (s *RoutePlannerService) generateAirRoute(req CalculateRouteRequest) *models.RouteRecommendation {
	// 1. 查找机场
	originAirports, _ := s.findNearestAirports(req.Origin, 1)
	destAirports, _ := s.findNearestAirports(req.Destination, 1)

	if len(originAirports) == 0 || len(destAirports) == 0 {
		return nil
	}

	oa := originAirports[0]
	da := destAirports[0]

	// 坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 2. 计算费用 (简单模型: 基础费 + 重量 * 单价)
	// 假设空运单价: $5/kg (亚洲), $8/kg (跨洲)
	destCountry := s.detectCountryFromAddress(req.Destination)
	routeRegion := detectRouteRegion(destCountry)

	ratePerKG := 5.0
	if routeRegion == RouteRegionNorthAm || routeRegion == RouteRegionEurope {
		ratePerKG = 8.0
	}

	lineHaulCost := 100.0 + req.WeightKG*ratePerKG
	firstMileCost := 150.0 // 卡车去机场
	lastMileCost := 150.0  // 卡车送货

	// 3. 计算时间
	totalDays := 3 // 默认
	if routeRegion == RouteRegionAsia {
		totalDays = 2
	}

	// 4. 货币转换
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}

	totalCost := convertToCurrency(firstMileCost+lineHaulCost+lastMileCost, currency)
	firstMileCost = convertToCurrency(firstMileCost, currency)
	lineHaulCost = convertToCurrency(lineHaulCost, currency)
	lastMileCost = convertToCurrency(lastMileCost, currency)

	return &models.RouteRecommendation{
		Type:      "fastest", // 空运通常是最快的
		Label:     "空运直达",
		TotalDays: totalDays,
		TotalCost: totalCost,
		Segments: []models.RouteSegment{
			{
				Type:       "first_mile",
				From:       req.Origin,
				To:         oa.IATACode,
				FromLat:    originLat,
				FromLng:    originLng,
				ToLat:      oa.Latitude,
				ToLng:      oa.Longitude,
				Mode:       "truck",
				Days:       1,
				Cost:       firstMileCost,
				DistanceKM: 45,
			},
			{
				Type:    "line_haul",
				From:    oa.IATACode + " (" + oa.Name + ")",
				To:      da.IATACode + " (" + da.Name + ")",
				FromLat: oa.Latitude,
				FromLng: oa.Longitude,
				ToLat:   da.Latitude,
				ToLng:   da.Longitude,
				Mode:    "air",
				Carrier: "AirChina",
				Days:    1, // 飞行时间通常<1天，算1天
				Cost:    lineHaulCost,
			},
			{
				Type:       "last_mile",
				From:       da.IATACode,
				To:         req.Destination,
				FromLat:    da.Latitude,
				FromLng:    da.Longitude,
				ToLat:      destLat,
				ToLng:      destLng,
				Mode:       "truck",
				Days:       1,
				Cost:       lastMileCost,
				DistanceKM: 40,
			},
		},
	}
}

// generateLandRoute 生成陆运路径
func (s *RoutePlannerService) generateLandRoute(req CalculateRouteRequest) *models.RouteRecommendation {
	// 获取坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 计算距离 (Haversine)
	distanceKM := haversineDistance(originLat, originLng, destLat, destLng)
	// 修正系数 (直线距离 -> 实际路程)
	roadDistance := distanceKM * 1.4

	// 估算时间 (平均速度 60km/h + 休息时间)
	drivingHours := roadDistance / 60.0
	// 假设每天行驶 10 小时
	totalDays := int(math.Ceil(drivingHours / 10.0))
	if totalDays < 1 {
		totalDays = 1
	}

	// 估算费用
	// 假设每公里 1.5 USD (整车) 或 0.5 USD (零担/部分)
	ratePerKM := 1.5
	if req.TransportMode == "lcl" || req.WeightKG < 5000 {
		ratePerKM = 0.5
	}
	totalCostUSD := roadDistance * ratePerKM

	// 货币转换
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}
	totalCost := convertToCurrency(totalCostUSD, currency)
	oneThirdCost := totalCost / 3.0

	return &models.RouteRecommendation{
		Type:      "fastest", // 陆运作为最快/默认
		Label:     "陆运直达",
		TotalDays: totalDays,
		TotalCost: totalCost,
		Segments: []models.RouteSegment{
			{
				Type:       "first_mile",
				From:       req.Origin,
				To:         req.Origin + " 集散中心",
				FromLat:    originLat,
				FromLng:    originLng,
				ToLat:      originLat + 0.01,
				ToLng:      originLng + 0.01,
				Mode:       "truck",
				Days:       1,
				Cost:       oneThirdCost,
				DistanceKM: roadDistance * 0.1,
			},
			{
				Type:       "line_haul",
				From:       req.Origin + " 集散中心",
				To:         req.Destination + " 分拨中心",
				FromLat:    originLat + 0.01,
				FromLng:    originLng + 0.01,
				ToLat:      destLat - 0.01,
				ToLng:      destLng - 0.01,
				Mode:       "truck",
				Carrier:    "顺丰速运",
				Days:       totalDays,
				Cost:       oneThirdCost,
				DistanceKM: roadDistance * 0.8,
			},
			{
				Type:       "last_mile",
				From:       req.Destination + " 分拨中心",
				To:         req.Destination,
				FromLat:    destLat - 0.01,
				FromLng:    destLng - 0.01,
				ToLat:      destLat,
				ToLng:      destLng,
				Mode:       "truck",
				Days:       1,
				Cost:       oneThirdCost,
				DistanceKM: roadDistance * 0.1,
			},
		},
	}
}

// SaveRoutePlan 保存规划结果
func (s *RoutePlannerService) SaveRoutePlan(plan *models.RoutePlan) error {
	return s.db.Create(plan).Error
}

// GetRoutePlan 获取规划结果
func (s *RoutePlannerService) GetRoutePlan(id string) (*models.RoutePlan, error) {
	var plan models.RoutePlan
	if err := s.db.First(&plan, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

// haversineDistance 计算两点间的球面距离 (km)
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // 地球半径 (km)

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// ============================================================
// 空运规划辅助方法
// ============================================================

// getDefaultAirports 获取默认机场 (当数据库没有匹配时)
func (s *RoutePlannerService) getDefaultAirports(address string) []models.Airport {
	countryCode := s.detectCountryFromAddress(address)

	// 主要国家的默认货运机场
	defaultAirports := map[string]models.Airport{
		"CN": {IATACode: "PVG", Name: "上海浦东国际机场", City: "上海", Country: "CN", Latitude: 31.1443, Longitude: 121.8083, IsCargoHub: true},
		"US": {IATACode: "LAX", Name: "洛杉矶国际机场", City: "洛杉矶", Country: "US", Latitude: 33.9425, Longitude: -118.4081, IsCargoHub: true},
		"DE": {IATACode: "FRA", Name: "法兰克福国际机场", City: "法兰克福", Country: "DE", Latitude: 50.0379, Longitude: 8.5622, IsCargoHub: true},
		"JP": {IATACode: "NRT", Name: "成田国际机场", City: "东京", Country: "JP", Latitude: 35.7720, Longitude: 140.3929, IsCargoHub: true},
		"KR": {IATACode: "ICN", Name: "仁川国际机场", City: "首尔", Country: "KR", Latitude: 37.4602, Longitude: 126.4407, IsCargoHub: true},
		"SG": {IATACode: "SIN", Name: "新加坡樟宜机场", City: "新加坡", Country: "SG", Latitude: 1.3644, Longitude: 103.9915, IsCargoHub: true},
		"AE": {IATACode: "DXB", Name: "迪拜国际机场", City: "迪拜", Country: "AE", Latitude: 25.2528, Longitude: 55.3644, IsCargoHub: true},
		"GB": {IATACode: "LHR", Name: "伦敦希思罗机场", City: "伦敦", Country: "GB", Latitude: 51.4700, Longitude: -0.4543, IsCargoHub: true},
		"HK": {IATACode: "HKG", Name: "香港国际机场", City: "香港", Country: "HK", Latitude: 22.3080, Longitude: 113.9185, IsCargoHub: true},
		"NL": {IATACode: "AMS", Name: "阿姆斯特丹史基浦机场", City: "阿姆斯特丹", Country: "NL", Latitude: 52.3086, Longitude: 4.7639, IsCargoHub: true},
		// 新增国家
		"RU": {IATACode: "SVO", Name: "莫斯科谢列梅捷沃机场", City: "莫斯科", Country: "RU", Latitude: 55.9726, Longitude: 37.4146, IsCargoHub: true},
		"FR": {IATACode: "CDG", Name: "巴黎戴高乐机场", City: "巴黎", Country: "FR", Latitude: 49.0097, Longitude: 2.5479, IsCargoHub: true},
		"IT": {IATACode: "MXP", Name: "米兰马尔彭萨机场", City: "米兰", Country: "IT", Latitude: 45.6306, Longitude: 8.7281, IsCargoHub: true},
		"ES": {IATACode: "MAD", Name: "马德里巴拉哈斯机场", City: "马德里", Country: "ES", Latitude: 40.4936, Longitude: -3.5668, IsCargoHub: true},
		"AU": {IATACode: "SYD", Name: "悉尼机场", City: "悉尼", Country: "AU", Latitude: -33.9399, Longitude: 151.1753, IsCargoHub: true},
		"CA": {IATACode: "YYZ", Name: "多伦多皮尔逊机场", City: "多伦多", Country: "CA", Latitude: 43.6777, Longitude: -79.6248, IsCargoHub: true},
		"BR": {IATACode: "GRU", Name: "圣保罗瓜鲁柳斯机场", City: "圣保罗", Country: "BR", Latitude: -23.4356, Longitude: -46.4731, IsCargoHub: true},
		"IN": {IATACode: "DEL", Name: "德里英迪拉甘地机场", City: "德里", Country: "IN", Latitude: 28.5562, Longitude: 77.1000, IsCargoHub: true},
		"TH": {IATACode: "BKK", Name: "曼谷素万那普机场", City: "曼谷", Country: "TH", Latitude: 13.6900, Longitude: 100.7501, IsCargoHub: true},
		"VN": {IATACode: "SGN", Name: "胡志明市新山一机场", City: "胡志明市", Country: "VN", Latitude: 10.8188, Longitude: 106.6520, IsCargoHub: true},
		"ID": {IATACode: "CGK", Name: "雅加达苏加诺机场", City: "雅加达", Country: "ID", Latitude: -6.1256, Longitude: 106.6558, IsCargoHub: true},
		"MY": {IATACode: "KUL", Name: "吉隆坡国际机场", City: "吉隆坡", Country: "MY", Latitude: 2.7456, Longitude: 101.7072, IsCargoHub: true},
		"TR": {IATACode: "IST", Name: "伊斯坦布尔机场", City: "伊斯坦布尔", Country: "TR", Latitude: 41.2608, Longitude: 28.7418, IsCargoHub: true},
		"PL": {IATACode: "WAW", Name: "华沙肖邦机场", City: "华沙", Country: "PL", Latitude: 52.1657, Longitude: 20.9671, IsCargoHub: true},
	}

	if airport, ok := defaultAirports[countryCode]; ok {
		return []models.Airport{airport}
	}

	// 返回上海作为默认
	return []models.Airport{defaultAirports["CN"]}
}

// buildAirRoute 构建空运路线
func (s *RoutePlannerService) buildAirRoute(req CalculateRouteRequest, originAirport, destAirport models.Airport) *models.RouteRecommendation {
	// 获取起点/终点坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 计算头程陆运距离
	firstMileKM := haversineDistance(originLat, originLng, originAirport.Latitude, originAirport.Longitude)
	// 计算空运干线距离
	airDistanceKM := haversineDistance(originAirport.Latitude, originAirport.Longitude, destAirport.Latitude, destAirport.Longitude)
	// 计算尾程陆运距离
	lastMileKM := haversineDistance(destAirport.Latitude, destAirport.Longitude, destLat, destLng)

	// 空运费用计算 (按重量/体积计费取大)
	chargeableWeight := req.WeightKG
	volumetricWeight := req.VolumeCBM * 167 // 空运体积重转换系数
	if volumetricWeight > chargeableWeight {
		chargeableWeight = volumetricWeight
	}

	// 空运费率 (USD/kg)
	airRatePerKG := 3.5 // 基础费率
	if req.CargoType == "dangerous" {
		airRatePerKG = 6.0
	} else if req.CargoType == "cold_chain" {
		airRatePerKG = 5.0
	}

	// 距离调整 (长距离略有优惠)
	if airDistanceKM > 8000 {
		airRatePerKG *= 1.2
	} else if airDistanceKM > 5000 {
		airRatePerKG *= 1.1
	}

	airFreightCost := chargeableWeight * airRatePerKG
	firstMileCost := firstMileKM * 0.5 // USD/km
	lastMileCost := lastMileKM * 0.5

	totalCostUSD := airFreightCost + firstMileCost + lastMileCost
	totalCost := convertToCurrency(totalCostUSD, req.Currency)

	// 时效计算
	firstMileDays := int(math.Ceil(firstMileKM / 500)) // 每天500km
	airDays := 1                                       // 空运通常1天
	if airDistanceKM > 10000 {
		airDays = 2
	}
	lastMileDays := int(math.Ceil(lastMileKM / 500))
	totalDays := firstMileDays + airDays + lastMileDays + 1 // +1 清关

	if totalDays < 3 {
		totalDays = 3 // 最少3天
	}

	return &models.RouteRecommendation{
		Type:      "fastest",
		Label:     "✈️ 空运直达",
		TotalDays: totalDays,
		TotalCost: totalCost,
		Segments: []models.RouteSegment{
			{
				Type:       "first_mile",
				From:       req.Origin,
				To:         originAirport.Name,
				FromLat:    originLat,
				FromLng:    originLng,
				ToLat:      originAirport.Latitude,
				ToLng:      originAirport.Longitude,
				Mode:       "truck",
				Days:       firstMileDays,
				Cost:       convertToCurrency(firstMileCost, req.Currency),
				DistanceKM: firstMileKM,
			},
			{
				Type:       "line_haul",
				From:       originAirport.Name,
				To:         destAirport.Name,
				FromLat:    originAirport.Latitude,
				FromLng:    originAirport.Longitude,
				ToLat:      destAirport.Latitude,
				ToLng:      destAirport.Longitude,
				Mode:       "air",
				Carrier:    "航空货运",
				Days:       airDays,
				Cost:       convertToCurrency(airFreightCost, req.Currency),
				DistanceKM: airDistanceKM,
			},
			{
				Type:       "last_mile",
				From:       destAirport.Name,
				To:         req.Destination,
				FromLat:    destAirport.Latitude,
				FromLng:    destAirport.Longitude,
				ToLat:      destLat,
				ToLng:      destLng,
				Mode:       "truck",
				Days:       lastMileDays,
				Cost:       convertToCurrency(lastMileCost, req.Currency),
				DistanceKM: lastMileKM,
			},
		},
	}
}

// ============================================================
// 多式联运规划辅助方法
// ============================================================

// buildLandSeaRoute 陆运+海运组合
func (s *RoutePlannerService) buildLandSeaRoute(req CalculateRouteRequest, originCountry, destCountry string) *models.RouteRecommendation {
	// 获取起止坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 查找最近港口
	polPorts, _ := s.findNearestPorts(req.Origin, "seaport", 1)
	podPorts, _ := s.findNearestPorts(req.Destination, "seaport", 1)

	if len(polPorts) == 0 || len(podPorts) == 0 {
		return nil
	}

	pol := polPorts[0]
	pod := podPorts[0]

	// 距离计算
	firstMileKM := haversineDistance(originLat, originLng, pol.Latitude, pol.Longitude)
	seaDistanceKM := haversineDistance(pol.Latitude, pol.Longitude, pod.Latitude, pod.Longitude)
	lastMileKM := haversineDistance(pod.Latitude, pod.Longitude, destLat, destLng)

	// 费用计算
	firstMileCostUSD := firstMileKM * 0.3
	seaFreightCostUSD := calculateLCLPrice(req.WeightKG, req.VolumeCBM, detectRouteRegion(destCountry), req.CargoType)
	lastMileCostUSD := lastMileKM * 0.3

	totalCostUSD := firstMileCostUSD + seaFreightCostUSD + lastMileCostUSD
	totalCost := convertToCurrency(totalCostUSD, req.Currency)

	// 时效
	firstMileDays := int(math.Ceil(firstMileKM / 600))
	seaDays := int(math.Ceil(seaDistanceKM / 600)) // 海运日均600km
	if seaDays < 7 {
		seaDays = 7
	}
	lastMileDays := int(math.Ceil(lastMileKM / 600))

	totalDays := firstMileDays + seaDays + lastMileDays + 3 // +3 港口操作

	return &models.RouteRecommendation{
		Type:      "balanced",
		Label:     "🚚🚢 陆运+海运",
		TotalDays: totalDays,
		TotalCost: totalCost,
		Segments: []models.RouteSegment{
			{Type: "first_mile", From: req.Origin, To: pol.Name, FromLat: originLat, FromLng: originLng, ToLat: pol.Latitude, ToLng: pol.Longitude, Mode: "truck", Days: firstMileDays, Cost: convertToCurrency(firstMileCostUSD, req.Currency), DistanceKM: firstMileKM},
			{Type: "line_haul", From: pol.Name, To: pod.Name, FromLat: pol.Latitude, FromLng: pol.Longitude, ToLat: pod.Latitude, ToLng: pod.Longitude, Mode: "ocean", Carrier: "船运", Days: seaDays, Cost: convertToCurrency(seaFreightCostUSD, req.Currency), DistanceKM: seaDistanceKM},
			{Type: "last_mile", From: pod.Name, To: req.Destination, FromLat: pod.Latitude, FromLng: pod.Longitude, ToLat: destLat, ToLng: destLng, Mode: "truck", Days: lastMileDays, Cost: convertToCurrency(lastMileCostUSD, req.Currency), DistanceKM: lastMileKM},
		},
	}
}

// buildLandAirRoute 陆运+空运组合 (紧急货物)
func (s *RoutePlannerService) buildLandAirRoute(req CalculateRouteRequest, originCountry, destCountry string) *models.RouteRecommendation {
	// 获取起止坐标
	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 查找最近机场
	originAirports := s.getDefaultAirports(req.Origin)
	destAirports := s.getDefaultAirports(req.Destination)

	if len(originAirports) == 0 || len(destAirports) == 0 {
		return nil
	}

	oa := originAirports[0]
	da := destAirports[0]

	// 距离计算
	firstMileKM := haversineDistance(originLat, originLng, oa.Latitude, oa.Longitude)
	airDistanceKM := haversineDistance(oa.Latitude, oa.Longitude, da.Latitude, da.Longitude)
	lastMileKM := haversineDistance(da.Latitude, da.Longitude, destLat, destLng)

	// 费用 (空运较贵)
	chargeableWeight := req.WeightKG
	if req.VolumeCBM*167 > chargeableWeight {
		chargeableWeight = req.VolumeCBM * 167
	}
	airFreightCostUSD := chargeableWeight * 4.0
	firstMileCostUSD := firstMileKM * 0.5
	lastMileCostUSD := lastMileKM * 0.5

	totalCostUSD := firstMileCostUSD + airFreightCostUSD + lastMileCostUSD
	totalCost := convertToCurrency(totalCostUSD, req.Currency)

	// 时效 (快速)
	firstMileDays := int(math.Ceil(firstMileKM / 500))
	airDays := 1
	if airDistanceKM > 10000 {
		airDays = 2
	}
	lastMileDays := int(math.Ceil(lastMileKM / 500))

	totalDays := firstMileDays + airDays + lastMileDays + 1

	return &models.RouteRecommendation{
		Type:      "fastest",
		Label:     "🚚✈️ 陆运+空运 (加急)",
		TotalDays: totalDays,
		TotalCost: totalCost,
		Segments: []models.RouteSegment{
			{Type: "first_mile", From: req.Origin, To: oa.Name, FromLat: originLat, FromLng: originLng, ToLat: oa.Latitude, ToLng: oa.Longitude, Mode: "truck", Days: firstMileDays, Cost: convertToCurrency(firstMileCostUSD, req.Currency), DistanceKM: firstMileKM},
			{Type: "line_haul", From: oa.Name, To: da.Name, FromLat: oa.Latitude, FromLng: oa.Longitude, ToLat: da.Latitude, ToLng: da.Longitude, Mode: "air", Carrier: "航空货运", Days: airDays, Cost: convertToCurrency(airFreightCostUSD, req.Currency), DistanceKM: airDistanceKM},
			{Type: "last_mile", From: da.Name, To: req.Destination, FromLat: da.Latitude, FromLng: da.Longitude, ToLat: destLat, ToLng: destLng, Mode: "truck", Days: lastMileDays, Cost: convertToCurrency(lastMileCostUSD, req.Currency), DistanceKM: lastMileKM},
		},
	}
}

// buildSeaRailRoute 海运+铁路组合 (中欧班列等)
func (s *RoutePlannerService) buildSeaRailRoute(req CalculateRouteRequest, originCountry, destCountry string) *models.RouteRecommendation {
	// 仅适用于中国到欧洲的路线
	if originCountry != "CN" {
		return nil
	}
	europeanCountries := []string{"DE", "FR", "GB", "NL", "BE", "ES", "IT", "PL", "SE", "GR", "AT", "CZ", "HU"}
	isEuropeDest := false
	for _, c := range europeanCountries {
		if destCountry == c {
			isEuropeDest = true
			break
		}
	}
	if !isEuropeDest {
		return nil
	}

	originLat, originLng := s.getAddressCoordinates(req.Origin)
	destLat, destLng := s.getAddressCoordinates(req.Destination)

	// 中欧班列主要节点
	xianLat, xianLng := 34.3416, 108.9398       // 西安
	duisburgLat, duisburgLng := 51.4344, 6.7623 // 杜伊斯堡

	// 距离计算
	firstMileKM := haversineDistance(originLat, originLng, xianLat, xianLng)
	railDistanceKM := haversineDistance(xianLat, xianLng, duisburgLat, duisburgLng)
	lastMileKM := haversineDistance(duisburgLat, duisburgLng, destLat, destLng)

	// 中欧班列费用 (比海运贵但比空运便宜)
	railRatePerCBM := 800.0 // USD/CBM
	railFreightCostUSD := req.VolumeCBM * railRatePerCBM
	if railFreightCostUSD < 500 {
		railFreightCostUSD = 500 // 最低收费
	}
	firstMileCostUSD := firstMileKM * 0.3
	lastMileCostUSD := lastMileKM * 0.4

	totalCostUSD := firstMileCostUSD + railFreightCostUSD + lastMileCostUSD
	totalCost := convertToCurrency(totalCostUSD, req.Currency)

	// 时效 (中欧班列约14-18天)
	firstMileDays := int(math.Ceil(firstMileKM / 600))
	railDays := 16
	lastMileDays := int(math.Ceil(lastMileKM / 500))

	totalDays := firstMileDays + railDays + lastMileDays

	return &models.RouteRecommendation{
		Type:      "balanced",
		Label:     "🚚🚂 中欧班列",
		TotalDays: totalDays,
		TotalCost: totalCost,
		Segments: []models.RouteSegment{
			{Type: "first_mile", From: req.Origin, To: "西安国际港站", FromLat: originLat, FromLng: originLng, ToLat: xianLat, ToLng: xianLng, Mode: "truck", Days: firstMileDays, Cost: convertToCurrency(firstMileCostUSD, req.Currency), DistanceKM: firstMileKM},
			{Type: "line_haul", From: "西安国际港站", To: "杜伊斯堡场站", FromLat: xianLat, FromLng: xianLng, ToLat: duisburgLat, ToLng: duisburgLng, Mode: "rail", Carrier: "中欧班列", Days: railDays, Cost: convertToCurrency(railFreightCostUSD, req.Currency), DistanceKM: railDistanceKM},
			{Type: "last_mile", From: "杜伊斯堡场站", To: req.Destination, FromLat: duisburgLat, FromLng: duisburgLng, ToLat: destLat, ToLng: destLng, Mode: "truck", Days: lastMileDays, Cost: convertToCurrency(lastMileCostUSD, req.Currency), DistanceKM: lastMileKM},
		},
	}
}

// 保留 errors 包导入供未来使用
var _ = errors.New
