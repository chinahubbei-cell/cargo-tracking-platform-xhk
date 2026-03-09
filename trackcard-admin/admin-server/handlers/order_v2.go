package handlers

import (
	"net/http"
	"time"

	"trackcard-admin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderV2Handler struct {
	db *gorm.DB
}

func NewOrderV2Handler(db *gorm.DB) *OrderV2Handler {
	return &OrderV2Handler{db: db}
}

// ==================== 辅助函数 ====================

func (h *OrderV2Handler) addLog(orderID, action, fromStatus, toStatus, operatorID, operatorName, note string) {
	log := models.OrderLog{
		OrderID:      orderID,
		Action:       action,
		FromStatus:   fromStatus,
		ToStatus:     toStatus,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Note:         note,
	}
	h.db.Create(&log)
}

// ==================== 订单列表 ====================

func (h *OrderV2Handler) List(c *gin.Context) {
	var page PaginationParams
	c.ShouldBindQuery(&page)
	page.Normalize()

	var total int64
	var orders []models.Order

	query := h.db.Model(&models.Order{})

	if orderType := c.Query("order_type"); orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}
	if status := c.Query("order_status"); status != "" {
		query = query.Where("order_status = ?", status)
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
	query.Order("created_at DESC").Offset(offset).Limit(page.PageSize).
		Preload("Items").Find(&orders)

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

// ==================== 订单详情 ====================

func (h *OrderV2Handler) Get(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.Preload("Items").Preload("Contract").Preload("Payment").
		Preload("Invoice").Preload("Logs", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// ==================== 创建订单（管理后台手工） ====================

type CreateOrderV2Request struct {
	OrgID       string `json:"org_id" binding:"required"`
	OrderType   string `json:"order_type" binding:"required,oneof=purchase renewal"`
	ContactName string `json:"contact_name"`
	Phone       string `json:"phone"`
	Remark      string `json:"remark"`

	// 收货信息（新购）
	ShippingAddress string `json:"shipping_address"`
	ReceiverName    string `json:"receiver_name"`
	ReceiverPhone   string `json:"receiver_phone"`

	// 有效期
	ServiceYears int `json:"service_years"`

	// 商品明细
	Items []CreateOrderItemRequest `json:"items" binding:"required,min=1"`
}

type CreateOrderItemRequest struct {
	ItemType     string  `json:"item_type" binding:"required"`
	DeviceID     string  `json:"device_id"`
	SkuID        string  `json:"sku_id"`
	ProductName  string  `json:"product_name"`
	Qty          int     `json:"qty" binding:"min=1"`
	UnitPrice    float64 `json:"unit_price"`
	ServicePrice float64 `json:"service_price"`
}

func (h *OrderV2Handler) Create(c *gin.Context) {
	var req CreateOrderV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误: " + err.Error()})
		return
	}

	// 获取组织名称
	var org models.Organization
	if err := h.db.First(&org, "id = ?", req.OrgID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "组织不存在"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")

	// 计算总金额
	var amountTotal float64
	var items []models.OrderItem
	for _, item := range req.Items {
		amount := float64(item.Qty) * (item.UnitPrice + item.ServicePrice)
		if req.ServiceYears > 1 {
			amount = float64(item.Qty) * (item.UnitPrice + item.ServicePrice*float64(req.ServiceYears))
		}
		amountTotal += amount

		items = append(items, models.OrderItem{
			ItemType:     item.ItemType,
			DeviceID:     item.DeviceID,
			SkuID:        item.SkuID,
			ProductName:  item.ProductName,
			Qty:          item.Qty,
			UnitPrice:    item.UnitPrice,
			ServicePrice: item.ServicePrice,
		})
	}

	order := models.Order{
		OrderType:       req.OrderType,
		Source:          "admin",
		OrgID:           req.OrgID,
		OrgName:         org.Name,
		ContactName:     req.ContactName,
		Phone:           req.Phone,
		AmountTotal:     amountTotal,
		Currency:        "CNY",
		OrderStatus:     models.OrderStatusDraft,
		ContractStatus:  models.ContractStatusNotGenerated,
		PaymentStatus:   models.PaymentStatusPending,
		InvoiceStatus:   models.InvoiceStatusNotRequested,
		ShippingAddress: req.ShippingAddress,
		ReceiverName:    req.ReceiverName,
		ReceiverPhone:   req.ReceiverPhone,
		ServiceYears:    req.ServiceYears,
		Remark:          req.Remark,
		CreatedBy:       adminID,
		Items:           items,
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建订单失败: " + err.Error()})
		return
	}

	h.addLog(order.ID, "create", "", models.OrderStatusDraft, adminID, adminName, "管理后台创建订单")

	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// ==================== 提交审核 ====================

func (h *OrderV2Handler) SubmitReview(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.OrderStatus != models.OrderStatusDraft {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "只有草稿状态可以提交审核"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")

	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status": models.OrderStatusPendingReview,
		"updated_at":   time.Now(),
	})

	h.addLog(order.ID, "submit_review", models.OrderStatusDraft, models.OrderStatusPendingReview, adminID, adminName, "提交审核")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已提交审核"})
}

// ==================== 审核（通过/驳回） ====================

type ReviewRequest struct {
	Action  string `json:"action" binding:"required,oneof=approve reject"`
	Comment string `json:"comment"`
}

func (h *OrderV2Handler) Review(c *gin.Context) {
	id := c.Param("id")
	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	var order models.Order
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.OrderStatus != models.OrderStatusPendingReview {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "只有待审核状态可以审核"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")
	now := time.Now()

	var newStatus string
	if req.Action == "approve" {
		newStatus = models.OrderStatusApproved
	} else {
		newStatus = models.OrderStatusRejected
	}

	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status":   newStatus,
		"reviewer_id":    adminID,
		"reviewer_name":  adminName,
		"reviewed_at":    now,
		"review_comment": req.Comment,
		"updated_at":     now,
	})

	actionLabel := "审核通过"
	if req.Action == "reject" {
		actionLabel = "审核驳回"
	}
	h.addLog(order.ID, req.Action, models.OrderStatusPendingReview, newStatus, adminID, adminName, actionLabel+": "+req.Comment)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": actionLabel})
}

// ==================== 生成合同 ====================

func (h *OrderV2Handler) GenerateContract(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.OrderStatus != models.OrderStatusApproved && order.OrderStatus != models.OrderStatusContractPending {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单状态不允许生成合同"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")

	// 创建合同记录
	contract := models.Contract{
		OrderID:        order.ID,
		ContractStatus: models.ContractStatusGenerated,
	}
	h.db.Create(&contract)

	// 更新订单状态
	oldStatus := order.OrderStatus
	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status":    models.OrderStatusContractPending,
		"contract_status": models.ContractStatusGenerated,
		"updated_at":      time.Now(),
	})

	h.addLog(order.ID, "generate_contract", oldStatus, models.OrderStatusContractPending, adminID, adminName, "生成合同: "+contract.ContractNo)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "合同已生成", "data": contract})
}

// ==================== 确认线下签约 ====================

type ConfirmOfflineRequest struct {
	FileURL string `json:"file_url"`
}

func (h *OrderV2Handler) ConfirmOfflineSign(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.Preload("Contract").First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.Contract == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "合同尚未生成"})
		return
	}

	var req ConfirmOfflineRequest
	c.ShouldBindJSON(&req)

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")
	now := time.Now()

	h.db.Model(order.Contract).Updates(map[string]interface{}{
		"sign_mode":       "offline",
		"contract_status": models.ContractStatusSignedOffline,
		"file_url":        req.FileURL,
		"signed_at":       now,
		"updated_at":      now,
	})

	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status":    models.OrderStatusPaymentPending,
		"contract_status": models.ContractStatusSignedOffline,
		"updated_at":      now,
	})

	h.addLog(order.ID, "confirm_offline_sign", order.OrderStatus, models.OrderStatusPaymentPending, adminID, adminName, "确认线下签约完成")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "签约已确认"})
}

// ==================== 确认到账（手工支付确认） ====================

type ConfirmPaymentRequest struct {
	Amount float64 `json:"amount"`
	Note   string  `json:"note"`
}

func (h *OrderV2Handler) ConfirmPayment(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.Preload("Payment").First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.OrderStatus != models.OrderStatusPaymentPending && order.OrderStatus != models.OrderStatusContractSigned {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单状态不允许确认支付"})
		return
	}

	var req ConfirmPaymentRequest
	c.ShouldBindJSON(&req)

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")
	now := time.Now()

	// 创建或更新支付记录
	if order.Payment == nil {
		payment := models.Payment{
			OrderID:       order.ID,
			Channel:       "placeholder",
			Amount:        order.AmountTotal,
			PaymentStatus: models.PaymentStatusPaid,
			PaidAt:        &now,
			ConfirmedBy:   adminID,
		}
		h.db.Create(&payment)
	} else {
		h.db.Model(order.Payment).Updates(map[string]interface{}{
			"payment_status": models.PaymentStatusPaid,
			"paid_at":        now,
			"confirmed_by":   adminID,
			"updated_at":     now,
		})
	}

	oldStatus := order.OrderStatus
	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status":   models.OrderStatusPaid,
		"payment_status": models.PaymentStatusPaid,
		"updated_at":     now,
	})

	note := "确认到账"
	if req.Note != "" {
		note += ": " + req.Note
	}
	h.addLog(order.ID, "confirm_payment", oldStatus, models.OrderStatusPaid, adminID, adminName, note)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "支付已确认"})
}

// ==================== 订单作废 ====================

type VoidRequest struct {
	Reason string `json:"reason"`
}

func (h *OrderV2Handler) Void(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	// 仅在未支付前允许作废
	if order.PaymentStatus == models.PaymentStatusPaid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "已支付订单不允许作废，需走退款流程"})
		return
	}
	if order.OrderStatus == models.OrderStatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单已作废"})
		return
	}
	if order.OrderStatus == models.OrderStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "已完成订单不允许作废"})
		return
	}

	var req VoidRequest
	c.ShouldBindJSON(&req)

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")
	oldStatus := order.OrderStatus

	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status": models.OrderStatusCancelled,
		"updated_at":   time.Now(),
	})

	h.addLog(order.ID, "void", oldStatus, models.OrderStatusCancelled, adminID, adminName, "订单作废: "+req.Reason)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订单已作废"})
}

// ==================== 开票审核 ====================

type InvoiceReviewRequest struct {
	Action  string `json:"action" binding:"required,oneof=approve reject"`
	Comment string `json:"comment"`
}

func (h *OrderV2Handler) InvoiceReview(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.Preload("Invoice").First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.Invoice == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到开票申请"})
		return
	}
	if order.Invoice.InvoiceStatus != models.InvoiceStatusRequested {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "开票申请状态不允许审核"})
		return
	}

	var req InvoiceReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")

	var newStatus string
	if req.Action == "approve" {
		newStatus = models.InvoiceStatusApproved
	} else {
		newStatus = models.InvoiceStatusRejected
	}

	h.db.Model(order.Invoice).Updates(map[string]interface{}{
		"invoice_status": newStatus,
		"review_comment": req.Comment,
		"updated_at":     time.Now(),
	})

	h.db.Model(&order).Updates(map[string]interface{}{
		"invoice_status": newStatus,
		"updated_at":     time.Now(),
	})

	actionLabel := "开票审核通过"
	if req.Action == "reject" {
		actionLabel = "开票审核驳回"
	}
	h.addLog(order.ID, "invoice_"+req.Action, models.InvoiceStatusRequested, newStatus, adminID, adminName, actionLabel+": "+req.Comment)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": actionLabel})
}

// ==================== 开票 ====================

type IssueInvoiceRequest struct {
	InvoiceNo string `json:"invoice_no" binding:"required"`
	FileURL   string `json:"file_url"`
}

func (h *OrderV2Handler) IssueInvoice(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.Preload("Invoice").First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.Invoice == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到开票申请"})
		return
	}
	if order.Invoice.InvoiceStatus != models.InvoiceStatusApproved {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "开票申请未审核通过"})
		return
	}

	var req IssueInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")
	now := time.Now()

	h.db.Model(order.Invoice).Updates(map[string]interface{}{
		"invoice_status": models.InvoiceStatusIssued,
		"invoice_no":     req.InvoiceNo,
		"file_url":       req.FileURL,
		"issued_at":      now,
		"updated_at":     now,
	})

	h.db.Model(&order).Updates(map[string]interface{}{
		"invoice_status": models.InvoiceStatusIssued,
		"order_status":   models.OrderStatusInvoiced,
		"updated_at":     now,
	})

	h.addLog(order.ID, "issue_invoice", models.InvoiceStatusApproved, models.InvoiceStatusIssued, adminID, adminName, "开票完成: "+req.InvoiceNo)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "开票完成"})
}

// ==================== 完成订单 ====================

func (h *OrderV2Handler) Complete(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	// 已支付后即可完成
	if order.PaymentStatus != models.PaymentStatusPaid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单未支付，不能完成"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")
	oldStatus := order.OrderStatus

	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status": models.OrderStatusCompleted,
		"updated_at":   time.Now(),
	})

	h.addLog(order.ID, "complete", oldStatus, models.OrderStatusCompleted, adminID, adminName, "订单完成")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订单已完成"})
}

// ==================== 履约（标记为履约中） ====================

func (h *OrderV2Handler) StartFulfilling(c *gin.Context) {
	id := c.Param("id")
	var order models.Order
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.PaymentStatus != models.PaymentStatusPaid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单未支付"})
		return
	}

	adminID := c.GetString("user_id")
	adminName := c.GetString("user_name")
	oldStatus := order.OrderStatus

	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status": models.OrderStatusFulfilling,
		"updated_at":   time.Now(),
	})

	h.addLog(order.ID, "start_fulfilling", oldStatus, models.OrderStatusFulfilling, adminID, adminName, "开始履约")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已开始履约"})
}
