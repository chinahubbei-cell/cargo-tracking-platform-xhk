package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
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

	var address string

	// 优先尝试腾讯地图（国内稳定）
	// 注意：前端传入 WGS-84 坐标，腾讯地图 reverseGeocodeDetail 内部不做坐标转换
	// 腾讯地图 API 默认接受 GCJ-02 但对于逆编码（只是查地名），WGS-84 偏差仅几百米，
	// 对于粗略地址通常仍能返回正确结果
	if services.TencentMap != nil {
		addr, err := services.TencentMap.ReverseGeocode(lat, lng)
		if err != nil {
			log.Printf("[ReverseGeocode] 腾讯地图失败: lat=%f, lng=%f, err=%v", lat, lng, err)
		} else if addr != "" {
			address = addr
		}
	}

	// 腾讯地图失败或不可用时，回退到 Nominatim（接受 WGS-84）
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
			"success":      true,
			"data":         gin.H{"display_name": fallbackAddress},
			"display_name": fallbackAddress,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"data":         gin.H{"display_name": address},
		"display_name": address,
	})
}
