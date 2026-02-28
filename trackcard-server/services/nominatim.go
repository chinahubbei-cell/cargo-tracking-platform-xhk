package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

// NominatimService OpenStreetMap Nominatim 地理编码服务
type NominatimService struct {
	userAgent   string
	client      *http.Client
	rateLimiter *time.Ticker
	mu          sync.Mutex
}

// NominatimSearchResult Nominatim搜索结果
type NominatimSearchResult struct {
	PlaceID     int     `json:"place_id"`
	Licence     string  `json:"licence"`
	OsmType     string  `json:"osm_type"`
	OsmID       int     `json:"osm_id"`
	Lat         string  `json:"lat"`
	Lon         string  `json:"lon"`
	DisplayName string  `json:"display_name"`
	Class       string  `json:"class"`
	Type        string  `json:"type"`
	Importance  float64 `json:"importance"`
	Address     struct {
		Road        string `json:"road"`
		Suburb      string `json:"suburb"`
		City        string `json:"city"`
		Town        string `json:"town"`
		Village     string `json:"village"`
		County      string `json:"county"`
		State       string `json:"state"`
		Postcode    string `json:"postcode"`
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
	} `json:"address"`
}

// NewNominatimService 创建Nominatim服务
// contactEmail: 用于Nominatim API的User-Agent，以便在问题发生时联系
func NewNominatimService(contactEmail string) *NominatimService {
	// Nominatim 使用政策要求：绝对最大值为 1 请求/秒
	// 我们设置为 1.2 秒以保持安全缓冲
	return &NominatimService{
		userAgent:   fmt.Sprintf("TrackCard/1.0 (%s)", contactEmail),
		client:      &http.Client{Timeout: 15 * time.Second},
		rateLimiter: time.NewTicker(1200 * time.Millisecond),
	}
}

// isEnglishAddress 判断是否为英文/海外地址（不含中文）
func isEnglishAddress(address string) bool {
	for _, r := range address {
		if unicode.Is(unicode.Han, r) {
			return false
		}
	}
	return true
}

// waitRateLimit 等待限流
func (s *NominatimService) waitRateLimit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	<-s.rateLimiter.C
}

// Search 地址搜索/联想
func (s *NominatimService) Search(query string) ([]SuggestionItem, error) {
	if query == "" {
		return nil, nil
	}
	if len(query) > 200 {
		return nil, fmt.Errorf("地址长度超过限制")
	}

	// 等待限流
	s.waitRateLimit()

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("addressdetails", "1")
	params.Set("limit", "10")
	params.Set("accept-language", "en")

	requestURL := "https://nominatim.openstreetmap.org/search?" + params.Encode()
	log.Printf("[Nominatim] Search请求: %s", requestURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求Nominatim失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim API错误: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var results []NominatimSearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	log.Printf("[Nominatim] 搜索返回 %d 条结果", len(results))

	items := make([]SuggestionItem, 0, len(results))
	for _, r := range results {
		lat, lng := parseLatLng(r.Lat, r.Lon)

		// 提取城市名
		city := r.Address.City
		if city == "" {
			city = r.Address.Town
		}
		if city == "" {
			city = r.Address.Village
		}

		items = append(items, SuggestionItem{
			Title:    r.DisplayName,
			Address:  r.DisplayName,
			Province: r.Address.State,
			City:     city,
			District: r.Address.County,
			Lat:      lat,
			Lng:      lng,
			Nation:   r.Address.Country,
		})
	}

	return items, nil
}

// Geocode 地址解析
func (s *NominatimService) Geocode(address string) (*AddressInfo, error) {
	if address == "" {
		return nil, fmt.Errorf("地址不能为空")
	}
	if len(address) > 200 {
		return nil, fmt.Errorf("地址长度超过限制")
	}

	// 等待限流
	s.waitRateLimit()

	params := url.Values{}
	params.Set("q", address)
	params.Set("format", "json")
	params.Set("addressdetails", "1")
	params.Set("limit", "1")
	params.Set("accept-language", "en")

	requestURL := "https://nominatim.openstreetmap.org/search?" + params.Encode()
	log.Printf("[Nominatim] Geocode请求: %s", requestURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求Nominatim失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim API错误: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var results []NominatimSearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到地址")
	}

	r := results[0]
	lat, lng := parseLatLng(r.Lat, r.Lon)

	// 提取城市名
	city := r.Address.City
	if city == "" {
		city = r.Address.Town
	}
	if city == "" {
		city = r.Address.Village
	}

	// 生成简称
	parts := []string{}
	if city != "" {
		parts = append(parts, city)
	}
	if r.Address.Country != "" {
		parts = append(parts, r.Address.Country)
	}
	shortName := strings.Join(parts, ", ")
	if shortName == "" {
		shortName = r.DisplayName
	}

	info := &AddressInfo{
		Address:     r.DisplayName,
		ShortName:   shortName,
		Lat:         lat,
		Lng:         lng,
		Province:    r.Address.State,
		City:        city,
		District:    r.Address.County,
		Nation:      r.Address.Country,
		IsOversea:   true,
		Reliability: 8,
	}

	log.Printf("[Nominatim] 解析结果: %s -> (%f, %f)", shortName, lat, lng)

	return info, nil
}

// parseLatLng 解析纬度经度
func parseLatLng(latStr, lngStr string) (float64, float64) {
	var lat, lng float64
	fmt.Sscanf(latStr, "%f", &lat)
	fmt.Sscanf(lngStr, "%f", &lng)
	return lat, lng
}

// ReverseGeocode 逆地理编码：将经纬度转换为地址
func (s *NominatimService) ReverseGeocode(lat, lng float64) (string, error) {
	// 等待限流
	s.waitRateLimit()

	params := url.Values{}
	params.Set("lat", fmt.Sprintf("%f", lat))
	params.Set("lon", fmt.Sprintf("%f", lng))
	params.Set("format", "json")
	params.Set("accept-language", "zh-CN")
	params.Set("zoom", "18")

	requestURL := "https://nominatim.openstreetmap.org/reverse?" + params.Encode()
	log.Printf("[Nominatim] ReverseGeocode请求: %s", requestURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求Nominatim失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Nominatim API错误: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		DisplayName string `json:"display_name"`
		Address     struct {
			Road     string `json:"road"`
			Suburb   string `json:"suburb"`
			City     string `json:"city"`
			Town     string `json:"town"`
			Village  string `json:"village"`
			County   string `json:"county"`
			State    string `json:"state"`
			Country  string `json:"country"`
		} `json:"address"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	log.Printf("[Nominatim] 逆地理编码结果: (%f, %f) -> %s", lat, lng, result.DisplayName)

	// 构建简化的地址显示
	parts := []string{}
	if result.Address.Road != "" {
		parts = append(parts, result.Address.Road)
	}
	if result.Address.City != "" {
		parts = append(parts, result.Address.City)
	} else if result.Address.Town != "" {
		parts = append(parts, result.Address.Town)
	} else if result.Address.Village != "" {
		parts = append(parts, result.Address.Village)
	}
	if result.Address.County != "" {
		parts = append(parts, result.Address.County)
	}
	if result.Address.State != "" {
		parts = append(parts, result.Address.State)
	}

	shortAddress := strings.Join(parts, "")
	if shortAddress == "" {
		shortAddress = result.DisplayName
	}

	return shortAddress, nil
}

// 全局实例
var Nominatim *NominatimService

// InitNominatimService 初始化Nominatim服务
func InitNominatimService(contactEmail string) {
	Nominatim = NewNominatimService(contactEmail)
}
