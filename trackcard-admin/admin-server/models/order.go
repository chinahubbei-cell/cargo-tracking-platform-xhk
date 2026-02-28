package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HardwareOrder 硬件订单
type HardwareOrder struct {
	ID      string `gorm:"primaryKey" json:"id"`
	OrderNo string `gorm:"uniqueIndex;size:50" json:"order_no"` // 订单号

	// 客户信息
	OrgID       string `gorm:"size:50" json:"org_id"`
	OrgName     string `gorm:"size:200" json:"org_name"`
	ContactName string `gorm:"size:100" json:"contact_name"`
	Phone       string `gorm:"size:20" json:"phone"`

	// 订单类型
	OrderType string `gorm:"size:20;not null" json:"order_type"` // purchase, renew, upgrade

	// 商品信息
	Products    string  `gorm:"type:text" json:"products"` // JSON:商品列表
	Quantity    int     `gorm:"default:1" json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalAmount float64 `json:"total_amount"`

	// 支付信息（第一期不对接支付，由前台系统处理）
	PaymentStatus string     `gorm:"size:20;default:pending" json:"payment_status"` // pending, paid, refunded
	PaymentMethod string     `gorm:"size:20" json:"payment_method"`                 // wechat, alipay, bank, offline
	PaidAt        *time.Time `json:"paid_at"`
	PaymentRef    string     `gorm:"size:100" json:"payment_ref"` // 支付流水号

	// 订单状态
	OrderStatus string `gorm:"size:20;default:pending" json:"order_status"` // pending, confirmed, processing, shipped, completed, cancelled

	// 发货信息
	ShippingAddress string     `gorm:"size:500" json:"shipping_address"`
	ShippedAt       *time.Time `json:"shipped_at"`
	TrackingNo      string     `gorm:"size:50" json:"tracking_no"`      // 快递单号
	TrackingCompany string     `gorm:"size:50" json:"tracking_company"` // 快递公司
	ReceivedAt      *time.Time `json:"received_at"`

	// 来源
	Source    string `gorm:"size:50" json:"source"`      // web, api, manual
	SourceRef string `gorm:"size:100" json:"source_ref"` // 来源订单号

	Remark      string    `gorm:"size:500" json:"remark"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `gorm:"size:50" json:"created_by"`
	ConfirmedBy string    `gorm:"size:50" json:"confirmed_by"`
}

func (o *HardwareOrder) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	if o.OrderNo == "" {
		o.OrderNo = generateOrderNo()
	}
	return nil
}

// generateOrderNo 生成订单号 HW-YYMMDD-XXXXXX
func generateOrderNo() string {
	now := time.Now()
	return now.Format("HW-060102-") + uuid.New().String()[:6]
}

// OrderProduct 订单商品
type OrderProduct struct {
	ProductCode string  `json:"product_code"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

// GetProducts 获取商品列表
func (o *HardwareOrder) GetProducts() []OrderProduct {
	var products []OrderProduct
	if o.Products != "" {
		json.Unmarshal([]byte(o.Products), &products)
	}
	return products
}

// SetProducts 设置商品列表
func (o *HardwareOrder) SetProducts(products []OrderProduct) {
	data, _ := json.Marshal(products)
	o.Products = string(data)
}

// ServiceRenewal 服务续费记录
type ServiceRenewal struct {
	ID          string `gorm:"primaryKey" json:"id"`
	OrgID       string `gorm:"size:50;not null" json:"org_id"`
	OrderID     string `gorm:"size:50" json:"order_id"`
	RenewalType string `gorm:"size:20;not null" json:"renewal_type"` // manual, auto, upgrade

	PeriodMonths int     `json:"period_months"`
	Amount       float64 `json:"amount"`

	OldEndDate *time.Time `json:"old_end_date"`
	NewEndDate *time.Time `json:"new_end_date"`

	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `gorm:"size:50" json:"created_by"`
}

func (r *ServiceRenewal) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}
