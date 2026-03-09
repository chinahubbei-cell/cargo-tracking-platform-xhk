package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// GeocodingResult 地理编码结果
type GeocodingResult struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DisplayName string  `json:"display_name"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	State       string  `json:"state"`
	City        string  `json:"city"`
}

// GeocodingService 地理编码服务
type GeocodingService struct {
	cache      map[string]*GeocodingResult
	cacheMutex sync.RWMutex
	httpClient *http.Client
}

// NewGeocodingService 创建地理编码服务实例
func NewGeocodingService() *GeocodingService {
	return &GeocodingService{
		cache: make(map[string]*GeocodingResult),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// nominatimResponse Nominatim API响应结构
type nominatimResponse struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
	Address     struct {
		City        string `json:"city"`
		Town        string `json:"town"`
		Village     string `json:"village"`
		State       string `json:"state"`
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
	} `json:"address"`
}

// Geocode 地址转坐标
func (s *GeocodingService) Geocode(address string) (*GeocodingResult, error) {
	if address == "" {
		return nil, fmt.Errorf("地址不能为空")
	}

	// 查询缓存
	cacheKey := strings.ToLower(strings.TrimSpace(address))
	s.cacheMutex.RLock()
	if result, ok := s.cache[cacheKey]; ok {
		s.cacheMutex.RUnlock()
		return result, nil
	}
	s.cacheMutex.RUnlock()

	// 调用Nominatim API
	baseURL := "https://nominatim.openstreetmap.org/search"
	params := url.Values{}
	params.Set("q", address)
	params.Set("format", "json")
	params.Set("addressdetails", "1")
	params.Set("limit", "1")
	params.Set("accept-language", "zh,en")

	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "TrackCard-Cargo-Platform/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var results []nominatimResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到地址: %s", address)
	}

	// 解析结果
	first := results[0]
	var lat, lng float64
	fmt.Sscanf(first.Lat, "%f", &lat)
	fmt.Sscanf(first.Lon, "%f", &lng)

	// 获取城市名
	city := first.Address.City
	if city == "" {
		city = first.Address.Town
	}
	if city == "" {
		city = first.Address.Village
	}
	if city == "" {
		city = first.Address.State
	}

	result := &GeocodingResult{
		Lat:         lat,
		Lng:         lng,
		DisplayName: first.DisplayName,
		Country:     first.Address.Country,
		CountryCode: strings.ToUpper(first.Address.CountryCode),
		State:       first.Address.State,
		City:        city,
	}

	// 存入缓存
	s.cacheMutex.Lock()
	s.cache[cacheKey] = result
	s.cacheMutex.Unlock()

	return result, nil
}

// GeocodeWithFallback 带回退的地理编码
func (s *GeocodingService) GeocodeWithFallback(address string, defaultLat, defaultLng float64, defaultCountry string) *GeocodingResult {
	result, err := s.Geocode(address)
	if err != nil {
		// 返回默认值
		return &GeocodingResult{
			Lat:         defaultLat,
			Lng:         defaultLng,
			CountryCode: defaultCountry,
		}
	}
	return result
}

// ReverseGeocode 坐标转地址
func (s *GeocodingService) ReverseGeocode(lat, lng float64) (*GeocodingResult, error) {
	return s.ReverseGeocodeWithLanguage(lat, lng, "zh,en")
}

// ReverseGeocodeWithLanguage 坐标转地址（支持指定返回语言）
func (s *GeocodingService) ReverseGeocodeWithLanguage(lat, lng float64, acceptLanguage string) (*GeocodingResult, error) {
	if strings.TrimSpace(acceptLanguage) == "" {
		acceptLanguage = "zh,en"
	}

	// 查询缓存
	cacheKey := fmt.Sprintf("%.4f,%.4f|%s", lat, lng, strings.ToLower(strings.TrimSpace(acceptLanguage)))
	s.cacheMutex.RLock()
	if result, ok := s.cache[cacheKey]; ok {
		s.cacheMutex.RUnlock()
		return result, nil
	}
	s.cacheMutex.RUnlock()

	// 调用Nominatim反向地理编码API
	baseURL := "https://nominatim.openstreetmap.org/reverse"
	params := url.Values{}
	params.Set("lat", fmt.Sprintf("%f", lat))
	params.Set("lon", fmt.Sprintf("%f", lng))
	params.Set("format", "json")
	params.Set("addressdetails", "1")
	params.Set("accept-language", acceptLanguage)

	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "Cargo-Tracking-Platform/1.1 (admin@cargotrack.test)")

	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second) // 指数退避
		}
		resp, err = s.httpClient.Do(req)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("请求失败(重试后): %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result nominatimResponse
	if parseErr := json.Unmarshal(body, &result); parseErr != nil {
		return nil, fmt.Errorf("解析响应失败: %v - body: %s", parseErr, string(body))
	}

	if result.DisplayName == "" {
		return nil, fmt.Errorf("无法通过服务返回有效的逆解析结果: body: %s", string(body))
	}

	// 获取城市名
	city := result.Address.City
	if city == "" {
		city = result.Address.Town
	}
	if city == "" {
		city = result.Address.Village
	}
	if city == "" {
		city = result.Address.State
	}

	geocodeResult := &GeocodingResult{
		Lat:         lat,
		Lng:         lng,
		DisplayName: result.DisplayName,
		Country:     result.Address.Country,
		CountryCode: strings.ToUpper(result.Address.CountryCode),
		State:       result.Address.State,
		City:        city,
	}

	// 存入缓存
	s.cacheMutex.Lock()
	s.cache[cacheKey] = geocodeResult
	s.cacheMutex.Unlock()

	return geocodeResult, nil
}

// GetCountryCodeFromAddress 从地址获取国家代码
func (s *GeocodingService) GetCountryCodeFromAddress(address string) string {
	result, err := s.Geocode(address)
	if err != nil {
		return ""
	}
	return result.CountryCode
}

// ClearCache 清空缓存
func (s *GeocodingService) ClearCache() {
	s.cacheMutex.Lock()
	s.cache = make(map[string]*GeocodingResult)
	s.cacheMutex.Unlock()
}

// CacheSize 获取缓存大小
func (s *GeocodingService) CacheSize() int {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	return len(s.cache)
}
