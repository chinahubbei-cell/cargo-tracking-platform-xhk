package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ============================================================
// 航空数据服务 - 外部API集成
// ============================================================
// 支持的API服务商:
// 1. AviationStack (性价比高，适合中小型TMS)
// 2. FlightAware AeroAPI (企业级，数据最权威)
// 3. Amadeus for Developers (适合订舱与规划)
// ============================================================

// AviationAPIProvider 航空API服务商类型
type AviationAPIProvider string

const (
	ProviderAviationStack AviationAPIProvider = "aviationstack"
	ProviderFlightAware   AviationAPIProvider = "flightaware"
	ProviderAmadeus       AviationAPIProvider = "amadeus"
)

// IATA代码验证正则 (3位大写字母)
var iataCodeRegex = regexp.MustCompile(`^[A-Z]{3}$`)

// 机场数据缓存
var aviationCache sync.Map

type aviationCacheEntry struct {
	airport  *AviationStackAirport
	cachedAt time.Time
}

// AviationDataService 航空数据服务
type AviationDataService struct {
	provider      AviationAPIProvider
	apiKey        string
	baseURL       string
	httpClient    *http.Client
	cacheEnabled  bool
	cacheDuration time.Duration
}

// AviationStackConfig AviationStack 配置
type AviationStackConfig struct {
	APIKey  string
	BaseURL string // 默认: http://api.aviationstack.com/v1/
}

// FlightAwareConfig FlightAware AeroAPI 配置
type FlightAwareConfig struct {
	APIKey  string
	BaseURL string // 默认: https://aeroapi.flightaware.com/aeroapi/
}

// NewAviationDataService 创建航空数据服务
func NewAviationDataService(provider AviationAPIProvider) *AviationDataService {
	service := &AviationDataService{
		provider:      provider,
		cacheEnabled:  true,
		cacheDuration: 24 * time.Hour, // 机场数据缓存24小时
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	// 根据提供商配置
	switch provider {
	case ProviderAviationStack:
		service.apiKey = os.Getenv("AVIATIONSTACK_API_KEY")
		service.baseURL = "http://api.aviationstack.com/v1/"
	case ProviderFlightAware:
		service.apiKey = os.Getenv("FLIGHTAWARE_API_KEY")
		service.baseURL = "https://aeroapi.flightaware.com/aeroapi/"
	case ProviderAmadeus:
		service.apiKey = os.Getenv("AMADEUS_API_KEY")
		service.baseURL = "https://test.api.amadeus.com/v1/"
	}

	return service
}

// ============================================================
// AviationStack API 响应结构
// ============================================================

// AviationStackAirportResponse AviationStack 机场查询响应
type AviationStackAirportResponse struct {
	Pagination struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Count  int `json:"count"`
		Total  int `json:"total"`
	} `json:"pagination"`
	Data []AviationStackAirport `json:"data"`
}

// AviationStackAirport AviationStack 机场数据
type AviationStackAirport struct {
	AirportName  string  `json:"airport_name"`
	IATACode     string  `json:"iata_code"`
	ICAOCode     string  `json:"icao_code"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	GeoNameID    string  `json:"geoname_id"`
	Timezone     string  `json:"timezone"`
	GMT          string  `json:"gmt"`
	PhoneNumber  string  `json:"phone_number"`
	CountryName  string  `json:"country_name"`
	CountryISO2  string  `json:"country_iso2"`
	CityIATACode string  `json:"city_iata_code"`
}

// AviationStackFlightResponse AviationStack 航班查询响应
type AviationStackFlightResponse struct {
	Pagination struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Count  int `json:"count"`
		Total  int `json:"total"`
	} `json:"pagination"`
	Data []AviationStackFlight `json:"data"`
}

// AviationStackFlight AviationStack 航班数据
type AviationStackFlight struct {
	FlightDate   string `json:"flight_date"`
	FlightStatus string `json:"flight_status"`
	Departure    struct {
		Airport   string `json:"airport"`
		Timezone  string `json:"timezone"`
		IATA      string `json:"iata"`
		ICAO      string `json:"icao"`
		Terminal  string `json:"terminal"`
		Gate      string `json:"gate"`
		Delay     int    `json:"delay"`
		Scheduled string `json:"scheduled"`
		Estimated string `json:"estimated"`
		Actual    string `json:"actual"`
	} `json:"departure"`
	Arrival struct {
		Airport   string `json:"airport"`
		Timezone  string `json:"timezone"`
		IATA      string `json:"iata"`
		ICAO      string `json:"icao"`
		Terminal  string `json:"terminal"`
		Gate      string `json:"gate"`
		Baggage   string `json:"baggage"`
		Delay     int    `json:"delay"`
		Scheduled string `json:"scheduled"`
		Estimated string `json:"estimated"`
		Actual    string `json:"actual"`
	} `json:"arrival"`
	Airline struct {
		Name string `json:"name"`
		IATA string `json:"iata"`
		ICAO string `json:"icao"`
	} `json:"airline"`
	Flight struct {
		Number     string `json:"number"`
		IATA       string `json:"iata"`
		ICAO       string `json:"icao"`
		Codeshared struct {
			AirlineName  string `json:"airline_name"`
			AirlineIATA  string `json:"airline_iata"`
			FlightNumber string `json:"flight_number"`
		} `json:"codeshared"`
	} `json:"flight"`
}

// ============================================================
// API 调用方法
// ============================================================

// buildSecureURL 构建 URL，避免 API Key 出现在错误日志中
func (s *AviationDataService) buildSecureURL(endpoint string, params map[string]string) (string, error) {
	u, err := url.Parse(s.baseURL + endpoint)
	if err != nil {
		return "", fmt.Errorf("无效的 URL: %w", err)
	}
	q := u.Query()
	q.Set("access_key", s.apiKey)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// GetAirportByIATA 通过IATA代码获取机场信息 (AviationStack)
func (s *AviationDataService) GetAirportByIATA(iataCode string) (*AviationStackAirport, error) {
	// 1. 验证提供商
	if s.provider != ProviderAviationStack {
		return nil, fmt.Errorf("此方法仅支持 AviationStack 提供商")
	}

	// 2. 验证 API Key
	if s.apiKey == "" {
		return nil, fmt.Errorf("未配置 AVIATIONSTACK_API_KEY 环境变量")
	}

	// 3. 输入验证：IATA代码格式检查
	iataCode = strings.ToUpper(strings.TrimSpace(iataCode))
	if !iataCodeRegex.MatchString(iataCode) {
		return nil, fmt.Errorf("无效的 IATA 代码格式: %s (应为3位大写字母)", iataCode)
	}

	// 4. 检查缓存
	if cached, ok := aviationCache.Load("airport:" + iataCode); ok {
		if entry, ok := cached.(*aviationCacheEntry); ok {
			if time.Since(entry.cachedAt) < 24*time.Hour {
				return entry.airport, nil
			}
		}
	}

	// 5. 构建安全 URL
	apiURL, err := s.buildSecureURL("airports", map[string]string{"iata_code": iataCode})
	if err != nil {
		return nil, err
	}

	// 6. 发送请求
	resp, err := s.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode)
	}

	// 7. 解析响应
	var result AviationStackAirportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("未找到机场: %s", iataCode)
	}

	// 8. 写入缓存
	aviationCache.Store("airport:"+iataCode, &aviationCacheEntry{
		airport:  &result.Data[0],
		cachedAt: time.Now(),
	})

	return &result.Data[0], nil
}

// GetFlightStatus 获取航班状态 (AviationStack)
func (s *AviationDataService) GetFlightStatus(flightIATA string) (*AviationStackFlight, error) {
	// 1. 验证提供商
	if s.provider != ProviderAviationStack {
		return nil, fmt.Errorf("此方法仅支持 AviationStack 提供商")
	}

	// 2. 验证 API Key
	if s.apiKey == "" {
		return nil, fmt.Errorf("未配置 AVIATIONSTACK_API_KEY 环境变量")
	}

	// 3. 输入验证：航班号格式检查
	flightIATA = strings.ToUpper(strings.TrimSpace(flightIATA))
	if flightIATA == "" {
		return nil, fmt.Errorf("航班号不能为空")
	}

	// 4. 构建安全 URL
	apiURL, err := s.buildSecureURL("flights", map[string]string{"flight_iata": flightIATA})
	if err != nil {
		return nil, err
	}

	// 5. 发送请求
	resp, err := s.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode)
	}

	// 6. 解析响应
	var result AviationStackFlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("未找到航班: %s", flightIATA)
	}

	return &result.Data[0], nil
}

// ============================================================
// FlightAware AeroAPI 方法 (企业级)
// ============================================================

// FlightAwareFlightInfo FlightAware 航班信息
type FlightAwareFlightInfo struct {
	FlightID            string    `json:"fa_flight_id"`
	OperatorICAO        string    `json:"operator_icao"`
	FlightNumber        string    `json:"flight_number"`
	Origin              string    `json:"origin"`
	Destination         string    `json:"destination"`
	DepartureTimeUTC    time.Time `json:"scheduled_off"`
	ArrivalTimeUTC      time.Time `json:"scheduled_on"`
	EstimatedArrival    time.Time `json:"estimated_on"`
	ActualDepartureTime time.Time `json:"actual_off"`
	ActualArrivalTime   time.Time `json:"actual_on"`
	Status              string    `json:"status"`
}

// GetFlightInfoFlightAware 获取航班信息 (FlightAware)
// 需要配置 FLIGHTAWARE_API_KEY 环境变量
func (s *AviationDataService) GetFlightInfoFlightAware(flightID string) (*FlightAwareFlightInfo, error) {
	if s.provider != ProviderFlightAware {
		return nil, fmt.Errorf("此方法仅支持 FlightAware 提供商")
	}
	if s.apiKey == "" {
		return nil, fmt.Errorf("未配置 FLIGHTAWARE_API_KEY 环境变量")
	}

	url := fmt.Sprintf("%sflights/%s", s.baseURL, flightID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("x-apikey", s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode)
	}

	var result FlightAwareFlightInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// ============================================================
// 使用示例 (代码注释)
// ============================================================
/*
使用方式:

1. 配置环境变量:
   export AVIATIONSTACK_API_KEY=your_api_key

2. 创建服务并调用:
   service := NewAviationDataService(ProviderAviationStack)
   airport, err := service.GetAirportByIATA("HKG")
   if err != nil {
       log.Printf("获取机场信息失败: %v", err)
   }
   fmt.Printf("机场: %s, 时区: %s\n", airport.AirportName, airport.Timezone)

3. 获取航班状态:
   flight, err := service.GetFlightStatus("CX888")
   if err == nil {
       fmt.Printf("航班状态: %s, 延误: %d分钟\n", flight.FlightStatus, flight.Arrival.Delay)
   }
*/
