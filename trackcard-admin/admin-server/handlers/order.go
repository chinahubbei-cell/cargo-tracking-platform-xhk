package handlers

import (
	"net/http"
	"time"

	"trackcard-admin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 订单状态常量
const (
	OrderStatusPending    = "pending"
	OrderStatusConfirmed  = "confirmed"
	OrderStatusProcessing = "processing"
	OrderStatusShipped    = "shipped"
	OrderStatusCompleted  = "completed"
	OrderStatusCancelled  = "cancelled"
)

type OrderHandler struct {
	db *gorm.DB
}

func NewOrderHandler(db *gorm.DB) *OrderHandler {
	return &OrderHandler{db: db}
}

// List 获取订单列表（带分页）
func (h *OrderHandler) List(c *gin.Context) {
	var page PaginationParams
	c.ShouldBindQuery(&page)
	page.Normalize()

	var total int64
	var orders []models.HardwareOrder

	query := h.db.Model(&models.HardwareOrder{})

	if status := c.Query("status"); status != "" {
		query = query.Where("order_status = ?", status)
	}
	if orderType := c.Query("type"); orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if orgID := c.Query("org_id"); orgID != "" {
		query = query.Where("org_id = ?", orgID)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("order_no LIKE ? OR org_name LIKE ? OR contact_name LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	query.Count(&total)

	offset := (page.Page - 1) * page.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(page.PageSize).Find(&orders)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orders,
		"pagination": gin.H{
			"page":        page.Page,
			"page_size":   page.PageSize,
			"total":       total,
			"total_pages": (total + int64(page.PageSize) - 1) / int64(page.PageSize),
		},
	})
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	OrgID           string  `json:"org_id" binding:"required"`
	OrgName         string  `json:"org_name"`
	OrderType       string  `json:"order_type" binding:"required,oneof=purchase renew upgrade"`
	Quantity        int     `json:"quantity" binding:"min=1"`
	UnitPrice       float64 `json:"unit_price" binding:"min=0"`
	TotalAmount     float64 `json:"total_amount" binding:"min=0"`
	ContactName     string  `json:"contact_name"`
	Phone           string  `json:"phone"`
	ShippingAddress string  `json:"shipping_address"`
	Remark          string  `json:"remark"`
}

// Create 创建订单
func (h *OrderHandler) Create(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "请求参数错误，请检查必填项"})
		return
	}

	// 验证组织存在
	var org models.Organization
	if err := h.db.First(&org, "id = ?", req.OrgID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "ORG_NOT_FOUND", "message": "组织不存在"})
		return
	}

	order := models.HardwareOrder{
		OrgID:           req.OrgID,
		OrgName:         org.Name,
		OrderType:       req.OrderType,
		Quantity:        req.Quantity,
		UnitPrice:       req.UnitPrice,
		TotalAmount:     req.TotalAmount,
		ContactName:     req.ContactName,
		Phone:           req.Phone,
		ShippingAddress: req.ShippingAddress,
		Remark:          req.Remark,
		OrderStatus:     OrderStatusPending,
		CreatedBy:       c.GetString("user_id"),
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "CREATE_FAILED", "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// Get 获取订单详情
func (h *OrderHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_ID", "message": "ID不能为空"})
		return
	}

	var order models.HardwareOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "订单不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// UpdateOrderRequest 更新订单请求（白名单）
type UpdateOrderRequest struct {
	ContactName     *string  `json:"contact_name"`
	Phone           *string  `json:"phone"`
	ShippingAddress *string  `json:"shipping_address"`
	Quantity        *int     `json:"quantity"`
	UnitPrice       *float64 `json:"unit_price"`
	TotalAmount     *float64 `json:"total_amount"`
	Remark          *string  `json:"remark"`
}

// Update 更新订单（白名单限制，仅限pending状态）
func (h *OrderHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var order models.HardwareOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "订单不存在"})
		return
	}

	// 只有pending状态可以修改
	if order.OrderStatus != OrderStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "STATUS_NOT_ALLOWED", "message": "只有待确认状态的订单可以修改"})
		return
	}

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "请求参数错误"})
		return
	}

	updates := make(map[string]interface{})
	if req.ContactName != nil {
		updates["contact_name"] = *req.ContactName
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.ShippingAddress != nil {
		updates["shipping_address"] = *req.ShippingAddress
	}
	if req.Quantity != nil && *req.Quantity > 0 {
		updates["quantity"] = *req.Quantity
	}
	if req.UnitPrice != nil {
		updates["unit_price"] = *req.UnitPrice
	}
	if req.TotalAmount != nil {
		updates["total_amount"] = *req.TotalAmount
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		h.db.Model(&order).Updates(updates)
	}

	h.db.First(&order, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// Confirm 确认订单
func (h *OrderHandler) Confirm(c *gin.Context) {
	id := c.Param("id")
	adminID := c.GetString("user_id")

	var order models.HardwareOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "订单不存在"})
		return
	}

	if order.OrderStatus != OrderStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "STATUS_NOT_ALLOWED", "message": "只有待确认状态的订单可以确认"})
		return
	}

	now := time.Now()
	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status": OrderStatusConfirmed,
		"confirmed_at": now,
		"confirmed_by": adminID,
		"updated_at":   now,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订单已确认"})
}

type ShipRequest struct {
	TrackingNo      string `json:"tracking_no"`
	TrackingCompany string `json:"tracking_company"`
}

// Ship 发货
func (h *OrderHandler) Ship(c *gin.Context) {
	id := c.Param("id")
	adminID := c.GetString("user_id")

	var order models.HardwareOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "订单不存在"})
		return
	}

	if order.OrderStatus != OrderStatusConfirmed && order.OrderStatus != OrderStatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "STATUS_NOT_ALLOWED", "message": "只有已确认或处理中的订单可以发货"})
		return
	}

	var req ShipRequest
	c.ShouldBindJSON(&req)

	now := time.Now()
	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status":     OrderStatusShipped,
		"shipped_at":       now,
		"shipped_by":       adminID,
		"tracking_no":      req.TrackingNo,
		"tracking_company": req.TrackingCompany,
		"updated_at":       now,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已发货"})
}

// Complete 完成订单
func (h *OrderHandler) Complete(c *gin.Context) {
	id := c.Param("id")

	var order models.HardwareOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "订单不存在"})
		return
	}

	// 只有shipped状态可以完成
	if order.OrderStatus != OrderStatusShipped {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "STATUS_NOT_ALLOWED", "message": "只有已发货的订单可以标记完成"})
		return
	}

	now := time.Now()
	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status": OrderStatusCompleted,
		"received_at":  now,
		"updated_at":   now,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订单已完成"})
}

type CancelRequest struct {
	Reason string `json:"reason"`
}

// Cancel 取消订单
func (h *OrderHandler) Cancel(c *gin.Context) {
	id := c.Param("id")

	var order models.HardwareOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "订单不存在"})
		return
	}

	// 检查状态
	if order.OrderStatus == OrderStatusShipped || order.OrderStatus == OrderStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "STATUS_NOT_ALLOWED", "message": "已发货或已完成的订单不能取消"})
		return
	}
	if order.OrderStatus == OrderStatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "ALREADY_CANCELLED", "message": "订单已取消"})
		return
	}

	var req CancelRequest
	c.ShouldBindJSON(&req)

	now := time.Now()
	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status":  OrderStatusCancelled,
		"cancel_reason": req.Reason,
		"cancelled_at":  now,
		"updated_at":    now,
	})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订单已取消"})
}
