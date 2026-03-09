package handlers

import (
	"net/http"
	"time"

	"trackcard-server/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TradeHandler struct {
	db *gorm.DB
}

func NewTradeHandler(db *gorm.DB) *TradeHandler {
	return &TradeHandler{db: db}
}

// addLog 记录操作日志
func (h *TradeHandler) addLog(orderID, action, fromStatus, toStatus, operatorID, operatorName, note string) {
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

// ==================== 创建新购订单 ====================

type CreatePurchaseRequest struct {
	ContactName     string                   `json:"contact_name" binding:"required"`
	Phone           string                   `json:"phone" binding:"required"`
	ShippingAddress string                   `json:"shipping_address"`
	ReceiverName    string                   `json:"receiver_name"`
	ReceiverPhone   string                   `json:"receiver_phone"`
	ServiceYears    int                      `json:"service_years"`
	Remark          string                   `json:"remark"`
	Items           []CreateTradeItemRequest `json:"items" binding:"required,min=1"`
}

type CreateTradeItemRequest struct {
	ProductName  string  `json:"product_name" binding:"required"`
	SkuID        string  `json:"sku_id"`
	Qty          int     `json:"qty" binding:"min=1"`
	UnitPrice    float64 `json:"unit_price"`
	ServicePrice float64 `json:"service_price"`
}

func (h *TradeHandler) CreatePurchaseOrder(c *gin.Context) {
	var req CreatePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误: " + err.Error()})
		return
	}

	userID := c.GetString("user_id")
	userName := c.GetString("user_name")
	orgID := c.GetString("org_id")

	// 获取组织名称
	var org models.Organization
	orgName := ""
	if err := h.db.First(&org, "id = ?", orgID).Error; err == nil {
		orgName = org.Name
	}

	// 计算总金额
	var amountTotal float64
	var items []models.OrderItem
	serviceYears := req.ServiceYears
	if serviceYears < 1 {
		serviceYears = 1
	}

	for _, item := range req.Items {
		qty := item.Qty
		if qty < 1 {
			qty = 1
		}
		amount := float64(qty) * (item.UnitPrice + item.ServicePrice*float64(serviceYears))
		amountTotal += amount

		items = append(items, models.OrderItem{
			ItemType:     "device",
			SkuID:        item.SkuID,
			ProductName:  item.ProductName,
			Qty:          qty,
			UnitPrice:    item.UnitPrice,
			ServicePrice: item.ServicePrice,
		})
	}

	order := models.Order{
		OrderType:       "purchase",
		Source:          "tracking",
		OrgID:           orgID,
		OrgName:         orgName,
		ContactName:     req.ContactName,
		Phone:           req.Phone,
		AmountTotal:     amountTotal,
		Currency:        "CNY",
		OrderStatus:     models.OrderStatusPendingReview, // 追踪平台提交直接进入待审核
		ContractStatus:  models.ContractStatusNotGenerated,
		PaymentStatus:   models.PaymentStatusPending,
		InvoiceStatus:   models.InvoiceStatusNotRequested,
		ShippingAddress: req.ShippingAddress,
		ReceiverName:    req.ReceiverName,
		ReceiverPhone:   req.ReceiverPhone,
		ServiceYears:    serviceYears,
		Remark:          req.Remark,
		CreatedBy:       userID,
		Items:           items,
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建订单失败: " + err.Error()})
		return
	}

	h.addLog(order.ID, "create", "", models.OrderStatusPendingReview, userID, userName, "追踪平台提交新购订单")

	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// ==================== 创建续费订单 ====================

type CreateRenewalRequest struct {
	ContactName  string                     `json:"contact_name" binding:"required"`
	Phone        string                     `json:"phone" binding:"required"`
	ServiceYears int                        `json:"service_years"`
	Remark       string                     `json:"remark"`
	Items        []CreateRenewalItemRequest `json:"items" binding:"required,min=1"`
}

type CreateRenewalItemRequest struct {
	DeviceID    string  `json:"device_id" binding:"required"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"` // 续费单价/年
}

func (h *TradeHandler) CreateRenewalOrder(c *gin.Context) {
	var req CreateRenewalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误: " + err.Error()})
		return
	}

	userID := c.GetString("user_id")
	userName := c.GetString("user_name")
	orgID := c.GetString("org_id")

	var org models.Organization
	orgName := ""
	if err := h.db.First(&org, "id = ?", orgID).Error; err == nil {
		orgName = org.Name
	}

	serviceYears := req.ServiceYears
	if serviceYears < 1 {
		serviceYears = 1
	}

	var amountTotal float64
	var items []models.OrderItem

	for _, item := range req.Items {
		amount := item.UnitPrice * float64(serviceYears)
		amountTotal += amount

		items = append(items, models.OrderItem{
			ItemType:     "service_renewal",
			DeviceID:     item.DeviceID,
			ProductName:  item.ProductName,
			Qty:          1,
			UnitPrice:    0,
			ServicePrice: item.UnitPrice,
		})
	}

	order := models.Order{
		OrderType:      "renewal",
		Source:         "tracking",
		OrgID:          orgID,
		OrgName:        orgName,
		ContactName:    req.ContactName,
		Phone:          req.Phone,
		AmountTotal:    amountTotal,
		Currency:       "CNY",
		OrderStatus:    models.OrderStatusPendingReview,
		ContractStatus: models.ContractStatusNotGenerated,
		PaymentStatus:  models.PaymentStatusPending,
		InvoiceStatus:  models.InvoiceStatusNotRequested,
		ServiceYears:   serviceYears,
		Remark:         req.Remark,
		CreatedBy:      userID,
		Items:          items,
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建订单失败: " + err.Error()})
		return
	}

	h.addLog(order.ID, "create", "", models.OrderStatusPendingReview, userID, userName, "追踪平台提交续费订单")

	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// ==================== 我的订单列表 ====================

func (h *TradeHandler) ListMyOrders(c *gin.Context) {
	orgID := c.GetString("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未选择组织"})
		return
	}

	var orders []models.Order
	query := h.db.Where("org_id = ?", orgID)

	if orderType := c.Query("order_type"); orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if status := c.Query("order_status"); status != "" {
		query = query.Where("order_status = ?", status)
	}

	query.Order("created_at DESC").Preload("Items").Find(&orders)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": orders})
}

// ==================== 订单详情 ====================

func (h *TradeHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")

	var order models.Order
	if err := h.db.Preload("Items").Preload("Contract").Preload("Payment").
		Preload("Invoice").Preload("Logs", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Where("org_id = ?", orgID).First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

// ==================== 在线签约 ====================

type OnlineSignRequest struct {
	CompanyName    string `json:"company_name" binding:"required"`
	CreditCode     string `json:"credit_code"`
	LegalPerson    string `json:"legal_person"`
	ContactName    string `json:"contact_name"`
	ContactPhone   string `json:"contact_phone"`
	ContactAddress string `json:"contact_address"`
	ReceiverName   string `json:"receiver_name"`
	ReceiverPhone  string `json:"receiver_phone"`
	ReceiverAddr   string `json:"receiver_addr"`
}

func (h *TradeHandler) SignOnline(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	userID := c.GetString("user_id")
	userName := c.GetString("user_name")

	var req OnlineSignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误: " + err.Error()})
		return
	}

	var order models.Order
	if err := h.db.Preload("Contract").Where("org_id = ?", orgID).First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.OrderStatus != models.OrderStatusContractPending && order.OrderStatus != models.OrderStatusApproved {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单状态不允许签约"})
		return
	}

	now := time.Now()

	if order.Contract == nil {
		// 创建合同记录
		contract := models.Contract{
			OrderID:        order.ID,
			SignMode:       "online",
			ContractStatus: models.ContractStatusSignedOnline,
			CompanyName:    req.CompanyName,
			CreditCode:     req.CreditCode,
			LegalPerson:    req.LegalPerson,
			ContactName:    req.ContactName,
			ContactPhone:   req.ContactPhone,
			ContactAddress: req.ContactAddress,
			ReceiverName:   req.ReceiverName,
			ReceiverPhone:  req.ReceiverPhone,
			ReceiverAddr:   req.ReceiverAddr,
			SignedAt:       &now,
		}
		h.db.Create(&contract)
	} else {
		h.db.Model(order.Contract).Updates(map[string]interface{}{
			"sign_mode":       "online",
			"contract_status": models.ContractStatusSignedOnline,
			"company_name":    req.CompanyName,
			"credit_code":     req.CreditCode,
			"legal_person":    req.LegalPerson,
			"contact_name":    req.ContactName,
			"contact_phone":   req.ContactPhone,
			"contact_address": req.ContactAddress,
			"receiver_name":   req.ReceiverName,
			"receiver_phone":  req.ReceiverPhone,
			"receiver_addr":   req.ReceiverAddr,
			"signed_at":       now,
			"updated_at":      now,
		})
	}

	oldStatus := order.OrderStatus
	h.db.Model(&order).Updates(map[string]interface{}{
		"order_status":    models.OrderStatusPaymentPending,
		"contract_status": models.ContractStatusSignedOnline,
		"updated_at":      now,
	})

	h.addLog(order.ID, "sign_online", oldStatus, models.OrderStatusPaymentPending, userID, userName, "在线签约完成")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "签约成功"})
}

// ==================== 线下签约上传 ====================

type OfflineUploadRequest struct {
	FileURL string `json:"file_url" binding:"required"`
}

func (h *TradeHandler) UploadOfflineContract(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	userID := c.GetString("user_id")
	userName := c.GetString("user_name")

	var req OfflineUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	var order models.Order
	if err := h.db.Preload("Contract").Where("org_id = ?", orgID).First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	now := time.Now()

	if order.Contract != nil {
		h.db.Model(order.Contract).Updates(map[string]interface{}{
			"sign_mode":       "offline",
			"contract_status": models.ContractStatusSigning,
			"file_url":        req.FileURL,
			"updated_at":      now,
		})
	} else {
		contract := models.Contract{
			OrderID:        order.ID,
			SignMode:       "offline",
			ContractStatus: models.ContractStatusSigning,
			FileURL:        req.FileURL,
		}
		h.db.Create(&contract)
	}

	// 更新合同状态但不改变订单状态（需管理后台确认）
	h.db.Model(&order).Updates(map[string]interface{}{
		"contract_status": models.ContractStatusSigning,
		"updated_at":      now,
	})

	h.addLog(order.ID, "upload_offline_contract", "", "", userID, userName, "上传线下签约合同")

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "合同已上传，等待管理员确认"})
}

// ==================== 创建支付单 ====================

func (h *TradeHandler) CreatePayment(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	userID := c.GetString("user_id")
	userName := c.GetString("user_name")

	var order models.Order
	if err := h.db.Preload("Payment").Where("org_id = ?", orgID).First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	if order.OrderStatus != models.OrderStatusPaymentPending && order.OrderStatus != models.OrderStatusContractSigned {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单状态不允许支付"})
		return
	}

	if order.Payment != nil && order.Payment.PaymentStatus == models.PaymentStatusPaid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订单已支付"})
		return
	}

	// Phase 1: 创建支付占位单
	if order.Payment == nil {
		payment := models.Payment{
			OrderID:       order.ID,
			Channel:       "placeholder",
			Amount:        order.AmountTotal,
			PaymentStatus: models.PaymentStatusPending,
		}
		h.db.Create(&payment)

		h.addLog(order.ID, "create_payment", "", "", userID, userName, "创建支付单（待线下打款）")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "支付单已创建，请按以下信息完成银行转账",
		"data": gin.H{
			"amount":       order.AmountTotal,
			"currency":     order.Currency,
			"bank_name":    "中国建设银行股份有限公司铜仁火车站支行",
			"account_name": "贵州小快科技有限公司",
			"account_no":   "52050168364100000305",
			"bank_code":    "105705081888",
			"note":         "支付完成后请联系管理员确认到账",
		},
	})
}

// ==================== 开票申请 ====================

type InvoiceApplyRequest struct {
	InvoiceType string  `json:"invoice_type"` // normal
	Title       string  `json:"title" binding:"required"`
	TaxNo       string  `json:"tax_no" binding:"required"`
	Amount      float64 `json:"amount"`
	Email       string  `json:"email"`
	Address     string  `json:"address"`
}

func (h *TradeHandler) ApplyInvoice(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")
	userID := c.GetString("user_id")
	userName := c.GetString("user_name")

	var req InvoiceApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误: " + err.Error()})
		return
	}

	var order models.Order
	if err := h.db.Preload("Invoice").Where("org_id = ?", orgID).First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订单不存在"})
		return
	}

	// 仅已支付订单可申请开票
	if order.PaymentStatus != models.PaymentStatusPaid {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "仅已支付订单可申请开票"})
		return
	}

	if order.Invoice != nil && order.Invoice.InvoiceStatus != models.InvoiceStatusRejected {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "已有开票申请"})
		return
	}

	invoiceType := req.InvoiceType
	if invoiceType == "" {
		invoiceType = "normal"
	}
	amount := req.Amount
	if amount <= 0 {
		amount = order.AmountTotal
	}

	if order.Invoice != nil {
		// 重新申请
		h.db.Model(order.Invoice).Updates(map[string]interface{}{
			"invoice_type":   invoiceType,
			"title":          req.Title,
			"tax_no":         req.TaxNo,
			"amount":         amount,
			"email":          req.Email,
			"address":        req.Address,
			"invoice_status": models.InvoiceStatusRequested,
			"review_comment": "",
			"updated_at":     time.Now(),
		})
	} else {
		invoice := models.Invoice{
			ID:            uuid.New().String(),
			OrderID:       order.ID,
			InvoiceType:   invoiceType,
			Title:         req.Title,
			TaxNo:         req.TaxNo,
			Amount:        amount,
			Email:         req.Email,
			Address:       req.Address,
			InvoiceStatus: models.InvoiceStatusRequested,
		}
		h.db.Create(&invoice)
	}

	h.db.Model(&order).Updates(map[string]interface{}{
		"invoice_status": models.InvoiceStatusRequested,
		"updated_at":     time.Now(),
	})

	h.addLog(order.ID, "apply_invoice", "", models.InvoiceStatusRequested, userID, userName, "申请开票: "+req.Title)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "开票申请已提交"})
}
