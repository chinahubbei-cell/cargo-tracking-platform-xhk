package handlers

import (
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

	// lang := c.DefaultQuery("lang", "zh-CN")
	var address string

	// 优先尝试腾讯地图（国内稳定）
	if services.TencentMap != nil {
		addr, err := services.TencentMap.ReverseGeocode(lat, lng)
		if err == nil && addr != "" {
			address = addr
		}
	}

	// 腾讯地图失败或不可用时，回退到 Nominatim
	if address == "" && services.Nominatim != nil {
		addr, err := services.Nominatim.ReverseGeocode(lat, lng)
		if err == nil && addr != "" {
			address = addr
		}
	}

	if address == "" {
		c.JSON(http.StatusOK, gin.H{
			"success":      true,
			"data":         gin.H{"display_name": ""},
			"display_name": "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"data":         gin.H{"display_name": address},
		"display_name": address,
	})
}
