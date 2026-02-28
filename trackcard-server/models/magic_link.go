package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MagicLink 免登录魔术链接
// 用于B端供应商（司机、报关行等）通过短信/邮件链接直接操作
type MagicLink struct {
	ID    string `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Token string `gorm:"uniqueIndex;type:varchar(64);not null" json:"token"`

	// 关联信息
	ShipmentID  string  `gorm:"type:varchar(50);index" json:"shipment_id"`
	TaskID      *string `gorm:"type:varchar(50);index" json:"task_id"`
	MilestoneID *string `gorm:"type:varchar(50);index" json:"milestone_id"`

	// 目标角色和动作
	TargetRole  string `gorm:"type:varchar(50);not null" json:"target_role"` // trucker, customs_broker, warehouse, carrier
	TargetName  string `gorm:"type:varchar(100)" json:"target_name"`         // 目标人/公司名称
	TargetPhone string `gorm:"type:varchar(20)" json:"target_phone"`         // 接收短信的手机号
	TargetEmail string `gorm:"type:varchar(100)" json:"target_email"`        // 接收邮件的邮箱
	ActionType  string `gorm:"type:varchar(50);not null" json:"action_type"` // 动作类型
	ActionTitle string `gorm:"type:varchar(200)" json:"action_title"`        // 动作标题（显示用）

	// 状态和时效
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	UsedIP    string     `gorm:"type:varchar(50)" json:"used_ip"`

	// 提交的数据
	SubmittedData string `gorm:"type:text" json:"submitted_data"` // JSON格式

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *MagicLink) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = "ml-" + uuid.New().String()[:8]
	}
	if m.Token == "" {
		m.Token = generateSecureToken(32)
	}
	return nil
}

// IsValid 检查链接是否有效
func (m *MagicLink) IsValid() bool {
	return m.UsedAt == nil && time.Now().Before(m.ExpiresAt)
}

// IsExpired 检查链接是否过期
func (m *MagicLink) IsExpired() bool {
	return time.Now().After(m.ExpiresAt)
}

// IsUsed 检查链接是否已使用
func (m *MagicLink) IsUsed() bool {
	return m.UsedAt != nil
}

// generateSecureToken 生成安全随机Token
func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ----------------------
// 动作类型常量
// ----------------------

const (
	// 司机相关
	ActionConfirmPickup   = "confirm_pickup"   // 确认提货
	ActionUploadPOD       = "upload_pod"       // 上传回单/签收单
	ActionConfirmDelivery = "confirm_delivery" // 确认送达
	ActionReportLocation  = "report_location"  // 上报位置

	// 报关行相关
	ActionUploadCustomsDoc = "upload_customs_doc" // 上传报关资料
	ActionConfirmCleared   = "confirm_cleared"    // 确认清关放行

	// 仓库相关
	ActionConfirmWarehouseIn  = "confirm_warehouse_in"  // 确认入仓
	ActionConfirmWarehouseOut = "confirm_warehouse_out" // 确认出仓

	// 通用
	ActionUploadDocument = "upload_document" // 上传文档
	ActionConfirmStatus  = "confirm_status"  // 确认状态
)

// ActionTypeInfo 动作类型信息
var ActionTypeInfo = map[string]struct {
	Title       string
	Description string
	NeedPhoto   bool
	NeedGPS     bool
}{
	ActionConfirmPickup:       {"确认提货", "请确认已提取货物", true, true},
	ActionUploadPOD:           {"上传回单", "请拍照上传签收回单", true, false},
	ActionConfirmDelivery:     {"确认送达", "请确认货物已送达收货人", true, true},
	ActionReportLocation:      {"上报位置", "请上报当前位置", false, true},
	ActionUploadCustomsDoc:    {"上传报关资料", "请上传报关相关文件", false, false},
	ActionConfirmCleared:      {"确认清关放行", "请确认货物已清关放行", false, false},
	ActionConfirmWarehouseIn:  {"确认入仓", "请确认货物已入仓", true, false},
	ActionConfirmWarehouseOut: {"确认出仓", "请确认货物已出仓", true, false},
	ActionUploadDocument:      {"上传文档", "请上传所需文档", false, false},
	ActionConfirmStatus:       {"确认状态", "请确认当前状态", false, false},
}

// ----------------------
// API 请求/响应类型
// ----------------------

// CreateMagicLinkRequest 创建魔术链接请求
type CreateMagicLinkRequest struct {
	ShipmentID  string  `json:"shipment_id" binding:"required"`
	TaskID      *string `json:"task_id"`
	MilestoneID *string `json:"milestone_id"`
	TargetRole  string  `json:"target_role" binding:"required"`
	TargetName  string  `json:"target_name"`
	TargetPhone string  `json:"target_phone"`
	TargetEmail string  `json:"target_email"`
	ActionType  string  `json:"action_type" binding:"required"`
	ExpiresIn   int     `json:"expires_in"` // 有效期（小时），默认24
}

// MagicLinkResponse 魔术链接响应
type MagicLinkResponse struct {
	ID          string    `json:"id"`
	ShortURL    string    `json:"short_url"` // 短链接 tms.com/m/xyz123
	FullURL     string    `json:"full_url"`  // 完整链接
	Token       string    `json:"token"`
	ActionType  string    `json:"action_type"`
	ActionTitle string    `json:"action_title"`
	ExpiresAt   time.Time `json:"expires_at"`
	TargetPhone string    `json:"target_phone"`
	TargetEmail string    `json:"target_email"`
}

// MagicLinkActionPage 操作页面数据
type MagicLinkActionPage struct {
	ShipmentID  string `json:"shipment_id"`
	ActionType  string `json:"action_type"`
	ActionTitle string `json:"action_title"`
	Description string `json:"description"`
	NeedPhoto   bool   `json:"need_photo"`
	NeedGPS     bool   `json:"need_gps"`
	TargetName  string `json:"target_name"`
}

// SubmitMagicLinkRequest 提交魔术链接操作请求
type SubmitMagicLinkRequest struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	PhotoURLs []string `json:"photo_urls"`
	Documents []string `json:"documents"`
	Remarks   string   `json:"remarks"`
	Confirmed bool     `json:"confirmed"`
}
