package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
)

// RouteHandler 线路处理器
type RouteHandler struct {
	db *gorm.DB
}

// NewRouteHandler 创建线路处理器
func NewRouteHandler(db *gorm.DB) *RouteHandler {
	return &RouteHandler{db: db}
}

// List 获取线路列表
func (h *RouteHandler) List(c *gin.Context) {
	var routes []models.Route

	query := h.db.Order("created_at DESC")

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 类型筛选
	if routeType := c.Query("type"); routeType != "" {
		query = query.Where("type = ?", routeType)
	}

	// 搜索
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&routes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取线路列表失败"})
		return
	}

	// 解析节点JSON
	type RouteWithNodes struct {
		models.Route
		ParsedNodes []models.RouteNode `json:"parsed_nodes"`
	}

	var result []RouteWithNodes
	for _, route := range routes {
		rn := RouteWithNodes{Route: route}
		if route.Nodes != "" {
			if err := json.Unmarshal([]byte(route.Nodes), &rn.ParsedNodes); err != nil {
				log.Printf("警告: 无法解析线路节点 JSON (route_id=%s): %v", route.ID, err)
				rn.ParsedNodes = []models.RouteNode{}
			}
		}
		result = append(result, rn)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": result})
}

// Get 获取线路详情
func (h *RouteHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var route models.Route
	if err := h.db.First(&route, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "线路不存在"})
		return
	}

	// 解析节点
	var nodes []models.RouteNode
	if route.Nodes != "" {
		if err := json.Unmarshal([]byte(route.Nodes), &nodes); err != nil {
			log.Printf("警告: 无法解析线路节点 JSON (route_id=%s): %v", route.ID, err)
			nodes = []models.RouteNode{}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"route":        route,
			"parsed_nodes": nodes,
		},
	})
}

// Create 创建线路
func (h *RouteHandler) Create(c *gin.Context) {
	var req models.CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 序列化节点
	nodesJSON, _ := json.Marshal(req.Nodes)

	route := models.Route{
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		Status:         "active",
		Nodes:          string(nodesJSON),
		TotalDistance:  req.TotalDistance,
		EstimatedHours: req.EstimatedHours,
	}

	if route.Type == "" {
		route.Type = "road"
	}

	if err := h.db.Create(&route).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建线路失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "创建成功", "data": route})
}

// Update 更新线路
func (h *RouteHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var route models.Route
	if err := h.db.First(&route, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "线路不存在"})
		return
	}

	var req models.UpdateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if len(req.Nodes) > 0 {
		nodesJSON, _ := json.Marshal(req.Nodes)
		updates["nodes"] = string(nodesJSON)
	}
	if req.TotalDistance > 0 {
		updates["total_distance"] = req.TotalDistance
	}
	if req.EstimatedHours > 0 {
		updates["estimated_hours"] = req.EstimatedHours
	}

	if err := h.db.Model(&route).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功"})
}

// Delete 删除线路
func (h *RouteHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// 检查是否有运单使用该线路
	var shipmentCount int64
	h.db.Model(&models.Shipment{}).Where("route_id = ?", id).Count(&shipmentCount)
	if shipmentCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": fmt.Sprintf("该线路正被 %d 个运单使用，无法删除", shipmentCount),
		})
		return
	}

	if err := h.db.Delete(&models.Route{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}
