package services

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
)

// TencentMapService 腾讯地图服务
type TencentMapService struct {
	apiKey    string
	secretKey string // 用于签名验证
}

// generateSignature 生成请求签名
func (s *TencentMapService) generateSignature(requestPath string, params url.Values) string {
	if s.secretKey == "" {
		return ""
	}

	// 参数按key排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 拼接参数
	var paramStr strings.Builder
	for i, k := range keys {
		if i > 0 {
			paramStr.WriteString("&")
		}
		paramStr.WriteString(k)
		paramStr.WriteString("=")
		paramStr.WriteString(params.Get(k))
	}

	// 生成签名字符串: 请求路径?参数字符串+SK
	signStr := requestPath + "?" + paramStr.String() + s.secretKey

	// MD5加密
	hash := md5.Sum([]byte(signStr))
	sig := hex.EncodeToString(hash[:])
	return sig
}

// GeocodeResult 地理编码结果
type GeocodeResult struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Result    *struct {
		Title    string `json:"title"`
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		Address           string `json:"address"`
		AddressComponents struct {
			Province     string `json:"province"`
			City         string `json:"city"`
			District     string `json:"district"`
			Street       string `json:"street"`
			StreetNumber string `json:"street_number"`
			// 海外地址字段
			Nation   string `json:"nation"`
			AdLevel1 string `json:"ad_level_1"` // 一级行政区划(州/县)
			AdLevel2 string `json:"ad_level_2"` // 二级区划(郡/县)
			AdLevel3 string `json:"ad_level_3"` // 三级区划(市)
		} `json:"address_components"`
		AdInfo struct {
			Nation     string `json:"nation"`
			NationCode string `json:"nation_code"`
			Adcode     string `json:"adcode"`
		} `json:"ad_info"`
		Reliability int `json:"reliability"`
		Level       int `json:"level"`
	} `json:"result"`
}

// SuggestionResult 输入联想结果
type SuggestionResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Data    []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Address  string `json:"address"`
		Province string `json:"province"`
		City     string `json:"city"`
		District string `json:"district"`
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"data"`
}

// AddressInfo 解析后的地址信息
type AddressInfo struct {
	Address     string  `json:"address"`     // 完整地址
	ShortName   string  `json:"short_name"`  // 简称(省市区或州/郡/市)
	Lat         float64 `json:"lat"`         // 纬度
	Lng         float64 `json:"lng"`         // 经度
	Province    string  `json:"province"`    // 省/州
	City        string  `json:"city"`        // 市
	District    string  `json:"district"`    // 区/县
	Nation      string  `json:"nation"`      // 国家
	IsOversea   bool    `json:"is_oversea"`  // 是否海外地址
	Reliability int     `json:"reliability"` // 可信度(1-10)
}

// SuggestionItem 联想项
type SuggestionItem struct {
	Title    string  `json:"title"`    // 地址标题
	Address  string  `json:"address"`  // 详细地址
	Province string  `json:"province"` // 省
	City     string  `json:"city"`     // 市
	District string  `json:"district"` // 区
	Lat      float64 `json:"lat"`      // 纬度
	Lng      float64 `json:"lng"`      // 经度
	Nation   string  `json:"nation"`   // 国家
}

// isChineseAddress 判断是否为中文地址
func isChineseAddress(address string) bool {
	for _, r := range address {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// extractCountry 从地址中提取国家名称
// 用于海外地址解析时的country参数
func extractCountry(address string) string {
	// 国家名称映射表 (地址中可能出现的名称 -> API参数值)
	countryMap := map[string]string{
		// 东南亚
		"indonesia":   "indonesia",
		"印尼":          "indonesia",
		"印度尼西亚":       "indonesia",
		"singapore":   "singapore",
		"新加坡":         "singapore",
		"malaysia":    "malaysia",
		"马来西亚":        "malaysia",
		"thailand":    "thailand",
		"泰国":          "thailand",
		"vietnam":     "vietnam",
		"越南":          "vietnam",
		"philippines": "philippines",
		"菲律宾":         "philippines",
		// 东亚
		"japan":       "japan",
		"日本":          "japan",
		"korea":       "korea",
		"south korea": "korea",
		"韩国":          "korea",
		// 南亚
		"india": "india",
		"印度":    "india",
		// 中东
		"uae":          "uae",
		"dubai":        "uae",
		"阿联酋":          "uae",
		"迪拜":           "uae",
		"saudi arabia": "saudi",
		"沙特":           "saudi",
		"saudi":        "saudi",
		// 欧洲
		"uk":             "uk",
		"united kingdom": "uk",
		"england":        "uk",
		"britain":        "uk",
		"英国":             "uk",
		"germany":        "germany",
		"德国":             "germany",
		"france":         "france",
		"法国":             "france",
		"italy":          "italy",
		"意大利":            "italy",
		"spain":          "spain",
		"西班牙":            "spain",
		"netherlands":    "netherlands",
		"荷兰":             "netherlands",
		"belgium":        "belgium",
		"比利时":            "belgium",
		// 俄罗斯 & 独联体
		"russia": "russia",
		"俄罗斯":    "russia",
		"moscow": "russia",
		"莫斯科":    "russia",
		// 北美
		"usa":           "usa",
		"united states": "usa",
		"america":       "usa",
		"美国":            "usa",
		"canada":        "canada",
		"加拿大":           "canada",
		"mexico":        "mexico",
		"墨西哥":           "mexico",
		// 南美
		"brazil":    "brazil",
		"巴西":        "brazil",
		"chile":     "chile",
		"智利":        "chile",
		"argentina": "argentina",
		"阿根廷":       "argentina",
		"peru":      "peru",
		"秘鲁":        "peru",
		// 大洋洲
		"australia":   "australia",
		"澳大利亚":        "australia",
		"澳洲":          "australia",
		"new zealand": "newzealand",
		"新西兰":         "newzealand",
		// 非洲
		"south africa": "southafrica",
		"南非":           "southafrica",
		"egypt":        "egypt",
		"埃及":           "egypt",
	}

	// 将地址转为小写进行匹配
	lowerAddr := strings.ToLower(address)

	for keyword, country := range countryMap {
		if strings.Contains(lowerAddr, strings.ToLower(keyword)) {
			return country
		}
	}

	return ""
}

// Geocode 地址地理编码
func (s *TencentMapService) Geocode(address string, oversea bool) (*AddressInfo, error) {
	requestPath := "/ws/geocoder/v1"

	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("address", address)
	params.Set("output", "json")

	// 尝试从地址中提取国家名称
	country := extractCountry(address)

	// 如果是海外地址，添加海外参数
	// 1. 显式指定oversea=true
	// 2. 地址不含中文
	// 3. 地址包含海外国家关键词(如"印尼")
	if oversea || !isChineseAddress(address) || country != "" {
		params.Set("oversea", "1")
		// 海外地址需要language参数
		// 如果是包含中文的海外地址(如"印尼雅加达")，保持中文返回可能更友好，但腾讯地图API文档建议海外用英文
		// 实测发现腾讯地图对"印尼"等关键词配合oversea=1能返回中文结果
		// 这里暂不强制en，除非完全无中文
		if !isChineseAddress(address) {
			params.Set("language", "en")
		} else {
			// 中文海外搜索，尝试不设language或设为zh-CN，腾讯API默认依据Header或自动
			// 此时主要依赖 country 参数来定位
		}

		if country != "" {
			params.Set("country", country)
		}
	}

	// 添加签名
	if s.secretKey != "" {
		sig := s.generateSignature(requestPath, params)
		params.Set("sig", sig)
	}

	requestURL := "https://apis.map.qq.com" + requestPath + "?" + params.Encode()
	log.Printf("[TencentMap] Geocode请求: %s", requestURL)

	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("请求腾讯地图API失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result GeocodeResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	log.Printf("[TencentMap] API响应: status=%d, message=%s", result.Status, result.Message)

	if result.Status != 0 {
		return nil, fmt.Errorf("地理编码失败: %s", result.Message)
	}

	if result.Result == nil {
		return nil, fmt.Errorf("未找到地址")
	}

	// 构建返回结果
	info := &AddressInfo{
		Address:     result.Result.Address,
		Lat:         result.Result.Location.Lat,
		Lng:         result.Result.Location.Lng,
		Province:    result.Result.AddressComponents.Province,
		City:        result.Result.AddressComponents.City,
		District:    result.Result.AddressComponents.District,
		Nation:      result.Result.AdInfo.Nation,
		Reliability: result.Result.Reliability,
	}

	// 判断是否海外地址并生成简称
	isChinese := isChineseAddress(address)
	if isChinese && (info.Nation == "" || info.Nation == "中国" || strings.Contains(info.Nation, "中国")) {
		// 中文地址: 省+市+区
		info.IsOversea = false
		parts := []string{}
		if info.Province != "" {
			parts = append(parts, info.Province)
		}
		if info.City != "" && info.City != info.Province {
			parts = append(parts, info.City)
		}
		// ShortName only contains Province + City (Level 2)
		info.ShortName = strings.Join(parts, "")
	} else {
		// 海外地址: ad_level_1 + ad_level_2 + ad_level_3
		info.IsOversea = true
		ac := result.Result.AddressComponents
		parts := []string{}
		if ac.AdLevel1 != "" {
			parts = append(parts, ac.AdLevel1)
		}
		if ac.AdLevel2 != "" {
			parts = append(parts, ac.AdLevel2)
		}
		if ac.AdLevel3 != "" {
			parts = append(parts, ac.AdLevel3)
		}
		if len(parts) > 0 {
			info.ShortName = strings.Join(parts, ", ")
		} else {
			// 备用：使用province/city/district
			parts := []string{}
			if info.Province != "" {
				parts = append(parts, info.Province)
			}
			if info.City != "" {
				parts = append(parts, info.City)
			}
			if info.District != "" {
				parts = append(parts, info.District)
			}
			info.ShortName = strings.Join(parts, ", ")
		}
	}

	log.Printf("[TencentMap] 解析结果: address=%s, short=%s, lat=%f, lng=%f", info.Address, info.ShortName, info.Lat, info.Lng)

	return info, nil
}

// Suggestion 地址输入联想
// oversea: 是否搜索海外地址
func (s *TencentMapService) Suggestion(keyword string, region string, oversea bool) ([]SuggestionItem, error) {
	requestPath := "/ws/place/v1/suggestion"

	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("keyword", keyword)
	params.Set("output", "json")

	if region != "" {
		params.Set("region", region)
	}

	// 尝试提取国家
	country := extractCountry(keyword)

	// 如果是海外地址或关键词不含中文，或包含国家关键词，添加海外参数
	if oversea || !isChineseAddress(keyword) || country != "" {
		params.Set("oversea", "1")
		// 海外地址需要language参数
		if !isChineseAddress(keyword) {
			params.Set("language", "en")
		}
	}

	// 添加签名
	if s.secretKey != "" {
		sig := s.generateSignature(requestPath, params)
		params.Set("sig", sig)
	}

	requestURL := "https://apis.map.qq.com" + requestPath + "?" + params.Encode()

	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("请求腾讯地图API失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result SuggestionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Status != 0 {
		return nil, fmt.Errorf("输入联想失败: %s", result.Message)
	}

	items := make([]SuggestionItem, 0, len(result.Data))
	for _, d := range result.Data {
		// 腾讯联想API不直接返回国家字段，我们简单推断
		// 如果不是海外模式，默认为中国
		nation := ""
		if !oversea && isChineseAddress(d.Title) {
			nation = "中国"
		}

		items = append(items, SuggestionItem{
			Title:    d.Title,
			Address:  d.Address,
			Province: d.Province,
			City:     d.City,
			District: d.District,
			Lat:      d.Location.Lat,
			Lng:      d.Location.Lng,
			Nation:   nation,
		})
	}

	return items, nil
}

// ReverseGeocodeResult 逆地理编码结果
type ReverseGeocodeResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		Address            string `json:"address"`
		FormattedAddresses struct {
			Recommend string `json:"recommend"` // 推荐解析地址
			Rough     string `json:"rough"`     // 粗略地址
		} `json:"formatted_addresses"`
		AddressComponent struct {
			Country      string `json:"country"`       // 国家
			Province     string `json:"province"`      // 省
			City         string `json:"city"`          // 市
			District     string `json:"district"`      // 区
			Street       string `json:"street"`        // 街道
			StreetNumber string `json:"street_number"` // 门牌
		} `json:"address_component"`
		AdInfo struct {
			Nation     string `json:"nation"`      // 国家/地区
			NationCode string `json:"nation_code"` // 国家代码
			Adcode     string `json:"adcode"`      // 行政区划代码
		} `json:"ad_info"`
	} `json:"result"`
}

// ReverseGeocodeDetail 结构化逆地理结果
type ReverseGeocodeDetail struct {
	Address    string `json:"address"`
	Country    string `json:"country"`
	Province   string `json:"province"`
	City       string `json:"city"`
	District   string `json:"district"`
	NationCode string `json:"nation_code"`
}

func (s *TencentMapService) reverseGeocodeDetail(lat, lng float64, language string) (*ReverseGeocodeDetail, error) {
	requestPath := "/ws/geocoder/v1/"
	params := url.Values{}
	params.Set("key", s.apiKey)
	params.Set("location", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("get_poi", "0")
	if strings.TrimSpace(language) != "" {
		params.Set("language", strings.TrimSpace(language))
	}

	// 服务端 Key 启用签名校验时必须带 sig
	if s.secretKey != "" {
		sig := s.generateSignature(requestPath, params)
		params.Set("sig", sig)
	}

	requestURL := "https://apis.map.qq.com" + requestPath + "?" + params.Encode()
	log.Printf("[TencentMap] ReverseGeocode请求: %s", requestURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求腾讯地图API失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result ReverseGeocodeResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Status != 0 {
		return nil, fmt.Errorf("腾讯地图API错误: %s", result.Message)
	}

	log.Printf("[TencentMap] 逆地理编码结果: (%f, %f) -> %s", lat, lng, result.Result.Address)

	// 使用推荐解析地址，如果没有则使用标准地址
	address := result.Result.FormattedAddresses.Recommend
	if address == "" {
		address = result.Result.FormattedAddresses.Rough
	}
	if address == "" {
		address = result.Result.Address
	}

	country := strings.TrimSpace(result.Result.AddressComponent.Country)
	if country == "" {
		country = strings.TrimSpace(result.Result.AdInfo.Nation)
	}

	city := strings.TrimSpace(result.Result.AddressComponent.City)
	if city == "" {
		city = strings.TrimSpace(result.Result.AddressComponent.Province)
	}
	if city == "" {
		city = strings.TrimSpace(result.Result.AddressComponent.District)
	}

	return &ReverseGeocodeDetail{
		Address:    address,
		Country:    country,
		Province:   strings.TrimSpace(result.Result.AddressComponent.Province),
		City:       city,
		District:   strings.TrimSpace(result.Result.AddressComponent.District),
		NationCode: strings.TrimSpace(result.Result.AdInfo.NationCode),
	}, nil
}

// ReverseGeocodeDetailWithLanguage 逆地理编码（可指定语言）
func (s *TencentMapService) ReverseGeocodeDetailWithLanguage(lat, lng float64, language string) (*ReverseGeocodeDetail, error) {
	return s.reverseGeocodeDetail(lat, lng, language)
}

// ReverseGeocodeDetail 逆地理编码：返回结构化国家/城市信息
func (s *TencentMapService) ReverseGeocodeDetail(lat, lng float64) (*ReverseGeocodeDetail, error) {
	return s.reverseGeocodeDetail(lat, lng, "")
}

// ReverseGeocode 逆地理编码：将经纬度转换为地址
func (s *TencentMapService) ReverseGeocode(lat, lng float64) (string, error) {
	detail, err := s.ReverseGeocodeDetail(lat, lng)
	if err != nil {
		return "", err
	}
	return detail.Address, nil
}

// 全局实例
var TencentMap *TencentMapService

// NewTencentMapService 创建腾讯地图服务
func NewTencentMapService(apiKey, secretKey string) *TencentMapService {
	return &TencentMapService{apiKey: apiKey, secretKey: secretKey}
}

// InitTencentMap 初始化腾讯地图服务
func InitTencentMap(apiKey, secretKey string) {
	TencentMap = NewTencentMapService(apiKey, secretKey)
	log.Println("[腾讯地图] 服务初始化完成")
}
