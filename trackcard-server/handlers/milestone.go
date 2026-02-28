package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
)

// MilestoneHandler 节点配置处理器
type MilestoneHandler struct {
	db *gorm.DB
}

// NewMilestoneHandler 创建节点配置处理器
func NewMilestoneHandler(db *gorm.DB) *MilestoneHandler {
	return &MilestoneHandler{db: db}
}

// ==================== 物流产品 API ====================

// ListProducts 获取物流产品列表
// GET /api/milestones/products
func (h *MilestoneHandler) ListProducts(c *gin.Context) {
	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	products, err := engine.GetProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

// GetProduct 获取单个物流产品
// GET /api/milestones/products/:id
func (h *MilestoneHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	product, err := engine.GetProduct(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "产品不存在"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// CreateProduct 创建物流产品
// POST /api/milestones/products
func (h *MilestoneHandler) CreateProduct(c *gin.Context) {
	var req models.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	product, err := engine.CreateProduct(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// ==================== 模板 API ====================

// ListTemplates 获取模板列表
// GET /api/milestones/templates
func (h *MilestoneHandler) ListTemplates(c *gin.Context) {
	productID := c.Query("product_id")

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	templates, err := engine.GetTemplates(productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GetTemplate 获取单个模板
// GET /api/milestones/templates/:id
func (h *MilestoneHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	template, err := engine.GetTemplate(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "模板不存在"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// CreateTemplate 创建模板
// POST /api/milestones/templates
func (h *MilestoneHandler) CreateTemplate(c *gin.Context) {
	var req models.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	template, err := engine.CreateTemplate(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// ==================== 节点 API ====================

// ListNodes 获取模板的节点列表
// GET /api/milestones/templates/:id/nodes
func (h *MilestoneHandler) ListNodes(c *gin.Context) {
	templateID := c.Param("id")

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	nodes, err := engine.GetNodes(templateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, nodes)
}

// CreateNode 创建节点
// POST /api/milestones/nodes
func (h *MilestoneHandler) CreateNode(c *gin.Context) {
	var req models.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	node, err := engine.CreateNode(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, node)
}

// UpdateNode 更新节点
// PUT /api/milestones/nodes/:id
func (h *MilestoneHandler) UpdateNode(c *gin.Context) {
	nodeID := c.Param("id")

	var req models.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	if err := engine.UpdateNode(nodeID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteNode 删除节点
// DELETE /api/milestones/nodes/:id
func (h *MilestoneHandler) DeleteNode(c *gin.Context) {
	nodeID := c.Param("id")

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	if err := engine.DeleteNode(nodeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ==================== 运单节点实例 API ====================

// GetShipmentMilestones 获取运单的节点实例
// GET /api/shipments/:id/milestones
func (h *MilestoneHandler) GetShipmentMilestones(c *gin.Context) {
	shipmentID := c.Param("id")

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	milestones, err := engine.GetShipmentMilestones(shipmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, milestones)
}

// CreateShipmentMilestones 为运单创建节点实例
// POST /api/shipments/:id/milestones
func (h *MilestoneHandler) CreateShipmentMilestones(c *gin.Context) {
	shipmentID := c.Param("id")

	var req struct {
		TemplateID string `json:"template_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定模板ID"})
		return
	}

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	if err := engine.CreateMilestonesForShipment(shipmentID, req.TemplateID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "节点实例创建成功"})
}

// TransitionMilestone 推进到下一节点
// POST /api/shipments/:id/milestones/transition
func (h *MilestoneHandler) TransitionMilestone(c *gin.Context) {
	shipmentID := c.Param("id")

	var req struct {
		Note string `json:"note"`
	}
	c.ShouldBindJSON(&req)

	engine := services.GetMilestoneEngine()
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	if err := engine.TransitionToNextMilestone(shipmentID, models.TriggerManual, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回更新后的节点列表
	milestones, _ := engine.GetShipmentMilestones(shipmentID)
	c.JSON(http.StatusOK, gin.H{
		"message":    "推进成功",
		"milestones": milestones,
	})
}
