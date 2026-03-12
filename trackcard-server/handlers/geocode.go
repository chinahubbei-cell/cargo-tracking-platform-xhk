package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	"trackcard-server/services"
)

// GeocodeHandler 地理编码处理器
type GeocodeHandler struct{}

// NewGeocodeHandler 创建处理器
func NewGeocodeHandler() *GeocodeHandler {
	return &GeocodeHandler{}
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

// shouldUseNominatim 决定是否使用Nominatim服务
func (h *GeocodeHandler) shouldUseNominatim(input string, forceOversea bool) bool {
	// 如果明确指定海外，或者输入不包含中文，则使用Nominatim
	return forceOversea || !isChineseAddress(input)
}

// Geocode 地址地理编码
// GET /api/geocode?address=xxx&oversea=0|1
func (h *GeocodeHandler) Geocode(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "地址不能为空"})
		return
	}

	oversea := c.Query("oversea") == "1"

	if h.shouldUseNominatim(address, oversea) {
		// 检查Nominatim服务是否初始化
		if services.Nominatim == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "海外地图服务暂不可用"})
			return
		}

		result, err := services.Nominatim.Geocode(address)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    result,
		})
		return
	}

	// 中文地址使用腾讯地图
	if services.TencentMap == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "国内地图服务暂不可用"})
		return
	}

	result, err := services.TencentMap.Geocode(address, oversea)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// Suggestion 地址输入联想
// GET /api/geocode/suggestion?keyword=xxx&region=xxx&oversea=0|1
func (h *GeocodeHandler) Suggestion(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "关键词不能为空"})
		return
	}

	region := c.Query("region")
	oversea := c.Query("oversea") == "1"

	if h.shouldUseNominatim(keyword, oversea) {
		// 检查Nominatim服务是否初始化
		if services.Nominatim == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "海外地图服务暂不可用"})
			return
		}

		results, err := services.Nominatim.Search(keyword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    results,
		})
		return
	}

	// 中文地址使用腾讯地图
	if services.TencentMap == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "国内地图服务暂不可用"})
		return
	}

	results, err := services.TencentMap.Suggestion(keyword, region, oversea)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

// ReverseGeocode 逆地理编码（坐标转地址）
// GET /api/geocode/reverse?lat=xxx&lng=xxx&lang=zh-CN
// 返回结构化数据：display_name, province, city, district, country, short_name
func (h *GeocodeHandler) ReverseGeocode(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少经纬度参数"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "纬度格式错误"})
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "经度格式错误"})
		return
	}

	// 坐标有效性校验
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "经纬度超出有效范围"})
		return
	}

	// 结构化响应字段
	var address, province, city, district, country, shortName string

	// 优先尝试腾讯地图（国内稳定，且能返回结构化省市区）
	if services.TencentMap != nil {
		detail, err := services.TencentMap.ReverseGeocodeDetail(lat, lng)
		if err != nil {
			log.Printf("[ReverseGeocode] 腾讯地图失败: lat=%f, lng=%f, err=%v", lat, lng, err)
		} else if detail != nil {
			address = detail.Address
			province = detail.Province
			city = detail.City
			district = detail.District
			country = detail.Country

			// 生成 short_name，与 PC 端 AddressInput 的逻辑统一
			// 国内地址: 省+市 (如 "广东省深圳市")
			// 海外地址: 城市名
			if country == "中国" || country == "" || strings.Contains(country, "中国") {
				parts := []string{}
				if province != "" {
					parts = append(parts, province)
				}
				if city != "" && city != province {
					parts = append(parts, city)
				}
				shortName = strings.Join(parts, "")
			} else {
				// 海外地址
				if city != "" {
					shortName = city
				} else if province != "" {
					shortName = province
				} else {
					shortName = country
				}
			}
		}
	}

	// 腾讯地图失败或不可用时，回退到 Nominatim（接受 WGS-84），仅获取 display_name
	if address == "" && services.Nominatim != nil {
		addr, err := services.Nominatim.ReverseGeocode(lat, lng)
		if err != nil {
			log.Printf("[ReverseGeocode] Nominatim失败: lat=%f, lng=%f, err=%v", lat, lng, err)
		} else if addr != "" {
			address = addr
		}
	}

	// 两个服务都失败 - 回退到 geocoding.go 中的 GeocodingService
	if address == "" {
		geocodingSvc := services.NewGeocodingService()
		result, err := geocodingSvc.ReverseGeocode(lat, lng)
		if err != nil {
			log.Printf("[ReverseGeocode] GeocodingService也失败: lat=%f, lng=%f, err=%v", lat, lng, err)
		} else if result != nil && result.DisplayName != "" {
			address = result.DisplayName
		}
	}

	if address == "" {
		log.Printf("[ReverseGeocode] 所有服务均失败: lat=%f, lng=%f", lat, lng)
		// 返回格式化坐标作为 fallback 地址
		latDir := "N"
		if lat < 0 {
			latDir = "S"
		}
		lngDir := "E"
		if lng < 0 {
			lngDir = "W"
		}
		fallbackAddress := fmt.Sprintf("%.4f°%s, %.4f°%s", lat, latDir, lng, lngDir)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"display_name": fallbackAddress,
				"short_name":   fallbackAddress,
			},
		})
		return
	}

	// 如果 shortName 为空（回退到 Nominatim 或 GeocodingService 的场景），使用 address
	if shortName == "" {
		shortName = address
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"display_name": address,
			"province":     province,
			"city":         city,
			"district":     district,
			"country":      country,
			"short_name":   shortName,
		},
	})
}
