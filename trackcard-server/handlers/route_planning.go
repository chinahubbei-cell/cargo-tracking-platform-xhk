package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

// RoutePlanningHandler 路径规划处理器
type RoutePlanningHandler struct {
	db      *gorm.DB
	planner *services.RoutePlannerService
}

// NewRoutePlanningHandler 创建路径规划处理器
func NewRoutePlanningHandler(db *gorm.DB) *RoutePlanningHandler {
	return &RoutePlanningHandler{
		db:      db,
		planner: services.NewRoutePlannerService(db),
	}
}

// CalculateRoutes 计算推荐路径
func (h *RoutePlanningHandler) CalculateRoutes(c *gin.Context) {
	var req services.CalculateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Origin == "" || req.Destination == "" {
		utils.BadRequest(c, "起点和终点不能为空")
		return
	}

	result, err := h.planner.CalculateRoutes(req)
	if err != nil {
		utils.InternalError(c, "计算路径失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, result)
}

// ListPorts 获取港口列表
func (h *RoutePlanningHandler) ListPorts(c *gin.Context) {
	var ports []models.Port

	query := h.db.Order("country ASC, name ASC")

	// 国家筛选
	if country := c.Query("country"); country != "" {
		query = query.Where("country = ?", country)
	}

	// 类型筛选
	if portType := c.Query("type"); portType != "" {
		query = query.Where("type = ?", portType)
	}

	// 中转枢纽筛选
	if hub := c.Query("is_transit_hub"); hub == "true" {
		query = query.Where("is_transit_hub = true")
	}

	// 搜索 - 转义 LIKE 特殊字符
	if search := c.Query("search"); search != "" {
		escapedSearch := strings.ReplaceAll(search, "%", "\\%")
		escapedSearch = strings.ReplaceAll(escapedSearch, "_", "\\_")
		query = query.Where("name LIKE ? OR name_en LIKE ? OR code LIKE ?",
			"%"+escapedSearch+"%", "%"+escapedSearch+"%", "%"+escapedSearch+"%")
	}

	if err := query.Find(&ports).Error; err != nil {
		utils.InternalError(c, "获取港口列表失败")
		return
	}

	utils.SuccessResponse(c, ports)
}

// GetPort 获取港口详情
func (h *RoutePlanningHandler) GetPort(c *gin.Context) {
	id := c.Param("id")

	var port models.Port
	if err := h.db.First(&port, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "港口不存在")
		return
	}

	utils.SuccessResponse(c, port)
}

// CreatePort 创建港口
func (h *RoutePlanningHandler) CreatePort(c *gin.Context) {
	var port models.Port
	if err := c.ShouldBindJSON(&port); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	if err := h.db.Create(&port).Error; err != nil {
		utils.InternalError(c, "创建港口失败")
		return
	}

	utils.CreatedResponse(c, port)
}

// UpdatePortRequest 港口更新请求 - 白名单限制可更新字段
type UpdatePortRequest struct {
	Name              *string  `json:"name"`
	NameEn            *string  `json:"name_en"`
	Country           *string  `json:"country"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	Type              *string  `json:"type"`
	IsTransitHub      *bool    `json:"is_transit_hub"`
	CustomsEfficiency *float64 `json:"customs_efficiency"`
}

// UpdatePort 更新港口
func (h *RoutePlanningHandler) UpdatePort(c *gin.Context) {
	id := c.Param("id")

	var port models.Port
	if err := h.db.First(&port, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "港口不存在")
		return
	}

	var req UpdatePortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	// 使用白名单构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.NameEn != nil {
		updates["name_en"] = *req.NameEn
	}
	if req.Country != nil {
		updates["country"] = *req.Country
	}
	if req.Latitude != nil {
		updates["latitude"] = *req.Latitude
	}
	if req.Longitude != nil {
		updates["longitude"] = *req.Longitude
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.IsTransitHub != nil {
		updates["is_transit_hub"] = *req.IsTransitHub
	}
	if req.CustomsEfficiency != nil {
		updates["customs_efficiency"] = *req.CustomsEfficiency
	}

	if len(updates) == 0 {
		utils.BadRequest(c, "没有要更新的字段")
		return
	}

	if err := h.db.Model(&port).Updates(updates).Error; err != nil {
		utils.InternalError(c, "更新失败")
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "更新成功"})
}

// DeletePort 删除港口
func (h *RoutePlanningHandler) DeletePort(c *gin.Context) {
	id := c.Param("id")

	// 检查是否有航线使用该港口
	var lineCount int64
	h.db.Model(&models.ShippingLine{}).Where("pol_port_id = ? OR pod_port_id = ?", id, id).Count(&lineCount)
	if lineCount > 0 {
		utils.ErrorResponse(c, 409, fmt.Sprintf("该港口正被 %d 条航线使用，无法删除", lineCount))
		return
	}

	if err := h.db.Delete(&models.Port{}, "id = ?", id).Error; err != nil {
		utils.InternalError(c, "删除失败")
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// ListShippingLines 获取航线列表
func (h *RoutePlanningHandler) ListShippingLines(c *gin.Context) {
	var lines []models.ShippingLine

	query := h.db.Preload("POLPort").Preload("PODPort").Where("active = true").Order("transit_days ASC")

	// 运输方式筛选
	if mode := c.Query("transport_mode"); mode != "" {
		query = query.Where("transport_mode = ?", mode)
	}

	// 承运商筛选
	if carrier := c.Query("carrier"); carrier != "" {
		query = query.Where("carrier = ?", carrier)
	}

	if err := query.Find(&lines).Error; err != nil {
		utils.InternalError(c, "获取航线列表失败")
		return
	}

	utils.SuccessResponse(c, lines)
}

// CreateShippingLine 创建航线
func (h *RoutePlanningHandler) CreateShippingLine(c *gin.Context) {
	var line models.ShippingLine
	if err := c.ShouldBindJSON(&line); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	if err := h.db.Create(&line).Error; err != nil {
		utils.InternalError(c, "创建航线失败")
		return
	}

	utils.CreatedResponse(c, line)
}

// SaveRoutePlan 保存规划结果
func (h *RoutePlanningHandler) SaveRoutePlan(c *gin.Context) {
	var req struct {
		Origin          string                     `json:"origin"`
		Destination     string                     `json:"destination"`
		CargoType       string                     `json:"cargo_type"`
		RecommendedType string                     `json:"recommended_type"`
		Route           models.RouteRecommendation `json:"route"`
		ShipmentID      *string                    `json:"shipment_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	planData, _ := json.Marshal(req.Route)

	plan := models.RoutePlan{
		OriginAddress:   req.Origin,
		DestAddress:     req.Destination,
		CargoType:       req.CargoType,
		RecommendedType: req.RecommendedType,
		PlanData:        string(planData),
		TotalDays:       req.Route.TotalDays,
		TotalCost:       req.Route.TotalCost,
		ShipmentID:      req.ShipmentID,
	}

	if err := h.db.Create(&plan).Error; err != nil {
		utils.InternalError(c, "保存规划失败")
		return
	}

	utils.CreatedResponse(c, plan)
}

// ListRoutePlans 获取规划历史
func (h *RoutePlanningHandler) ListRoutePlans(c *gin.Context) {
	var plans []models.RoutePlan

	query := h.db.Order("created_at DESC")

	if shipmentID := c.Query("shipment_id"); shipmentID != "" {
		query = query.Where("shipment_id = ?", shipmentID)
	}

	if err := query.Limit(50).Find(&plans).Error; err != nil {
		utils.InternalError(c, "获取规划历史失败")
		return
	}

	utils.SuccessResponse(c, plans)
}
