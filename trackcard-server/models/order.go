package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ========== 订单状态常量 (与 admin-server 保持一致) ==========

const (
	OrderStatusDraft           = "draft"
	OrderStatusPendingReview   = "pending_review"
	OrderStatusRejected        = "rejected"
	OrderStatusApproved        = "approved"
	OrderStatusContractPending = "contract_pending"
	OrderStatusContractSigned  = "contract_signed"
	OrderStatusPaymentPending  = "payment_pending"
	OrderStatusPaid            = "paid"
	OrderStatusInvoicePending  = "invoice_pending"
	OrderStatusInvoiced        = "invoiced"
	OrderStatusFulfilling      = "fulfilling"
	OrderStatusCompleted       = "completed"
	OrderStatusCancelled       = "cancelled"
)

const (
	ContractStatusNotGenerated  = "not_generated"
	ContractStatusGenerated     = "generated"
	ContractStatusSigning       = "signing"
	ContractStatusSignedOnline  = "signed_online"
	ContractStatusSignedOffline = "signed_offline"
	ContractStatusInvalid       = "invalid"
)

const (
	PaymentStatusPending = "pending_payment"
	PaymentStatusPaid    = "paid"
)

const (
	InvoiceStatusNotRequested = "not_requested"
	InvoiceStatusRequested    = "requested"
	InvoiceStatusApproved     = "approved"
	InvoiceStatusIssued       = "issued"
	InvoiceStatusDelivered    = "delivered"
	InvoiceStatusRejected     = "rejected"
)

// ========== Order 主订单表 ==========

type Order struct {
	ID      string `gorm:"primaryKey;size:50" json:"id"`
	OrderNo string `gorm:"uniqueIndex;size:50" json:"order_no"`

	OrderType string `gorm:"size:20;not null" json:"order_type"`
	Source    string `gorm:"size:20;not null" json:"source"`

	OrgID        string `gorm:"size:50" json:"org_id"`
	OrgName      string `gorm:"size:200" json:"org_name"`
	SubAccountID string `gorm:"size:50" json:"sub_account_id"`
	ContactName  string `gorm:"size:100" json:"contact_name"`
	Phone        string `gorm:"size:30" json:"phone"`

	AmountTotal float64 `json:"amount_total"`
	Currency    string  `gorm:"size:10;default:CNY" json:"currency"`

	OrderStatus    string `gorm:"size:30;default:draft" json:"order_status"`
	ContractStatus string `gorm:"size:30;default:not_generated" json:"contract_status"`
	PaymentStatus  string `gorm:"size:30;default:pending_payment" json:"payment_status"`
	InvoiceStatus  string `gorm:"size:30;default:not_requested" json:"invoice_status"`

	ReviewerID    string     `gorm:"size:50" json:"reviewer_id"`
	ReviewerName  string     `gorm:"size:100" json:"reviewer_name"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	ReviewComment string     `gorm:"size:500" json:"review_comment"`

	ShippingAddress string `gorm:"size:500" json:"shipping_address"`
	ReceiverName    string `gorm:"size:100" json:"receiver_name"`
	ReceiverPhone   string `gorm:"size:30" json:"receiver_phone"`

	ServiceYears int `json:"service_years"`

	Remark    string    `gorm:"size:500" json:"remark"`
	CreatedBy string    `gorm:"size:50" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Items    []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Contract *Contract   `gorm:"foreignKey:OrderID" json:"contract,omitempty"`
	Payment  *Payment    `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
	Invoice  *Invoice    `gorm:"foreignKey:OrderID" json:"invoice,omitempty"`
	Logs     []OrderLog  `gorm:"foreignKey:OrderID" json:"logs,omitempty"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	if o.OrderNo == "" {
		now := time.Now()
		prefix := "PO"
		if o.OrderType == "renewal" {
			prefix = "RN"
		}
		o.OrderNo = prefix + "-" + now.Format("060102") + "-" + uuid.New().String()[:6]
	}
	return nil
}

// ========== OrderItem ==========

type OrderItem struct {
	ID             string     `gorm:"primaryKey;size:50" json:"id"`
	OrderID        string     `gorm:"size:50;not null;index" json:"order_id"`
	ItemType       string     `gorm:"size:30;not null" json:"item_type"`
	DeviceID       string     `gorm:"size:50" json:"device_id"`
	SkuID          string     `gorm:"size:50" json:"sku_id"`
	ProductName    string     `gorm:"size:200" json:"product_name"`
	Qty            int        `gorm:"default:1" json:"qty"`
	UnitPrice      float64    `json:"unit_price"`
	ServicePrice   float64    `json:"service_price"`
	ServiceStartAt *time.Time `json:"service_start_at"`
	ServiceEndAt   *time.Time `json:"service_end_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (i *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

// ========== Contract ==========

type Contract struct {
	ID             string     `gorm:"primaryKey;size:50" json:"id"`
	OrderID        string     `gorm:"size:50;not null;uniqueIndex" json:"order_id"`
	ContractNo     string     `gorm:"size:50" json:"contract_no"`
	SignMode       string     `gorm:"size:20" json:"sign_mode"`
	ContractStatus string     `gorm:"size:30;default:not_generated" json:"contract_status"`
	CompanyName    string     `gorm:"size:200" json:"company_name"`
	CreditCode     string     `gorm:"size:50" json:"credit_code"`
	LegalPerson    string     `gorm:"size:100" json:"legal_person"`
	LegalPersonID  string     `gorm:"size:50" json:"legal_person_id"`
	ContactName    string     `gorm:"size:100" json:"contact_name"`
	ContactPhone   string     `gorm:"size:30" json:"contact_phone"`
	ContactAddress string     `gorm:"size:500" json:"contact_address"`
	ReceiverName   string     `gorm:"size:100" json:"receiver_name"`
	ReceiverPhone  string     `gorm:"size:30" json:"receiver_phone"`
	ReceiverAddr   string     `gorm:"size:500" json:"receiver_addr"`
	FileURL        string     `gorm:"size:500" json:"file_url"`
	SignedAt       *time.Time `json:"signed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (c *Contract) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.ContractNo == "" {
		c.ContractNo = "CT-" + time.Now().Format("060102") + "-" + uuid.New().String()[:6]
	}
	return nil
}

// ========== Payment ==========

type Payment struct {
	ID            string     `gorm:"primaryKey;size:50" json:"id"`
	OrderID       string     `gorm:"size:50;not null;uniqueIndex" json:"order_id"`
	PaymentNo     string     `gorm:"size:50" json:"payment_no"`
	Channel       string     `gorm:"size:30;default:placeholder" json:"channel"`
	Amount        float64    `json:"amount"`
	PaymentStatus string     `gorm:"size:30;default:pending_payment" json:"payment_status"`
	PaidAt        *time.Time `json:"paid_at"`
	RawCallback   string     `gorm:"type:text" json:"raw_callback"`
	ConfirmedBy   string     `gorm:"size:50" json:"confirmed_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.PaymentNo == "" {
		p.PaymentNo = "PAY-" + time.Now().Format("060102") + "-" + uuid.New().String()[:6]
	}
	return nil
}

// ========== Invoice ==========

type Invoice struct {
	ID            string     `gorm:"primaryKey;size:50" json:"id"`
	OrderID       string     `gorm:"size:50;not null;uniqueIndex" json:"order_id"`
	InvoiceType   string     `gorm:"size:30;default:normal" json:"invoice_type"`
	Title         string     `gorm:"size:200" json:"title"`
	TaxNo         string     `gorm:"size:50" json:"tax_no"`
	Amount        float64    `json:"amount"`
	Email         string     `gorm:"size:100" json:"email"`
	Address       string     `gorm:"size:500" json:"address"`
	InvoiceStatus string     `gorm:"size:30;default:not_requested" json:"invoice_status"`
	InvoiceNo     string     `gorm:"size:50" json:"invoice_no"`
	IssuedAt      *time.Time `json:"issued_at"`
	FileURL       string     `gorm:"size:500" json:"file_url"`
	ReviewComment string     `gorm:"size:500" json:"review_comment"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (inv *Invoice) BeforeCreate(tx *gorm.DB) error {
	if inv.ID == "" {
		inv.ID = uuid.New().String()
	}
	return nil
}

// ========== OrderLog ==========

type OrderLog struct {
	ID           string    `gorm:"primaryKey;size:50" json:"id"`
	OrderID      string    `gorm:"size:50;not null;index" json:"order_id"`
	Action       string    `gorm:"size:50;not null" json:"action"`
	FromStatus   string    `gorm:"size:30" json:"from_status"`
	ToStatus     string    `gorm:"size:30" json:"to_status"`
	OperatorID   string    `gorm:"size:50" json:"operator_id"`
	OperatorName string    `gorm:"size:100" json:"operator_name"`
	Note         string    `gorm:"size:500" json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

func (l *OrderLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}
