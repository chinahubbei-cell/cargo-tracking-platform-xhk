package handlers

import (
	"net/http"
	"strings"

	"trackcard-server/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SearchHandler 搜索处理器
type SearchHandler struct {
	db *gorm.DB
}

// NewSearchHandler 创建搜索处理器
func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

// SuggestionItem 搜索建议项
type SuggestionItem struct {
	Value    string `json:"value"`    // 显示的值
	Type     string `json:"type"`     // 类型: shipment, device
	ID       string `json:"id"`       // 实际ID
	Label    string `json:"label"`    // 标签描述
	SubLabel string `json:"subLabel"` // 副标签
}

// Suggestions 获取搜索建议
// GET /api/search/suggestions?keyword=xxx
func (h *SearchHandler) Suggestions(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" || len(keyword) < 2 {
		c.JSON(http.StatusOK, gin.H{"suggestions": []SuggestionItem{}})
		return
	}

	var suggestions []SuggestionItem
	limit := 5 // 每种类型最多返回5条

	// 搜索运单
	var shipments []models.Shipment
	h.db.Where("id LIKE ? OR bill_of_lading LIKE ? OR cargo_name LIKE ? OR container_no LIKE ?",
		"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Order("created_at DESC").
		Limit(limit).
		Find(&shipments)

	for _, s := range shipments {
		label := s.ID
		if s.BillOfLading != "" {
			label = s.BillOfLading
		}
		subLabel := ""
		if s.Origin != "" && s.Destination != "" {
			subLabel = s.Origin + " → " + s.Destination
		}
		suggestions = append(suggestions, SuggestionItem{
			Value:    s.ID,
			Type:     "shipment",
			ID:       s.ID,
			Label:    label,
			SubLabel: subLabel,
		})
	}

	// 搜索设备
	var devices []models.Device
	h.db.Where("id LIKE ? OR external_device_id LIKE ? OR name LIKE ?",
		"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Order("created_at DESC").
		Limit(limit).
		Find(&devices)

	for _, d := range devices {
		label := d.ID
		if d.ExternalDeviceID != nil && *d.ExternalDeviceID != "" {
			label = *d.ExternalDeviceID
		}
		subLabel := d.Name
		suggestions = append(suggestions, SuggestionItem{
			Value:    d.ID,
			Type:     "device",
			ID:       d.ID,
			Label:    label,
			SubLabel: subLabel,
		})
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}
