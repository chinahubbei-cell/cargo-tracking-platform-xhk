package models

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 审计日志类型 ====================

// AuditActionType 审计动作类型
type AuditActionType string

const (
	AuditActionCreate  AuditActionType = "create"
	AuditActionUpdate  AuditActionType = "update"
	AuditActionDelete  AuditActionType = "delete"
	AuditActionView    AuditActionType = "view"
	AuditActionExport  AuditActionType = "export"
	AuditActionLogin   AuditActionType = "login"
	AuditActionLogout  AuditActionType = "logout"
	AuditActionApprove AuditActionType = "approve"
	AuditActionReject  AuditActionType = "reject"
)

// ==================== 影子模式日志 ====================

// ShadowOperationLog 影子操作日志
type ShadowOperationLog struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// 操作者信息
	OperatorID   string `gorm:"type:varchar(50);index;not null" json:"operator_id"` // 实际操作人
	OperatorName string `gorm:"type:varchar(100)" json:"operator_name"`
	OperatorRole string `gorm:"type:varchar(50)" json:"operator_role"`

	// 被代操作对象
	ShadowForType string `gorm:"type:varchar(50);index" json:"shadow_for_type"` // user/partner/customer
	ShadowForID   string `gorm:"type:varchar(50);index" json:"shadow_for_id"`
	ShadowForName string `gorm:"type:varchar(100)" json:"shadow_for_name"`

	// 操作详情
	Action   string `gorm:"type:varchar(50);index" json:"action"`    // HTTP方法
	Resource string `gorm:"type:varchar(200);index" json:"resource"` // 资源路径
	Method   string `gorm:"type:varchar(10)" json:"method"`

	// 关联资源
	ShipmentID *string `gorm:"type:varchar(50);index" json:"shipment_id,omitempty"`
	DocumentID *uint   `gorm:"index" json:"document_id,omitempty"`

	// 请求/响应
	RequestBody  string `gorm:"type:text" json:"-"`
	ResponseCode int    `json:"response_code"`

	// 元数据
	ClientIP  string `gorm:"type:varchar(50)" json:"client_ip"`
	UserAgent string `gorm:"type:varchar(500)" json:"user_agent"`
	Reason    string `gorm:"type:varchar(500)" json:"reason"` // 代操作原因
	Remark    string `gorm:"type:varchar(1000)" json:"remark"`

	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (ShadowOperationLog) TableName() string {
	return "shadow_operation_logs"
}

// ==================== 通用审计日志 ====================

// AuditLog 审计日志
type AuditLog struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// 操作者
	UserID   string `gorm:"type:varchar(50);index;not null" json:"user_id"`
	UserName string `gorm:"type:varchar(100)" json:"user_name"`
	UserRole string `gorm:"type:varchar(50)" json:"role"`

	// 是否影子模式
	IsShadowMode  bool    `gorm:"default:false" json:"is_shadow_mode"`
	ShadowForID   *string `gorm:"type:varchar(50)" json:"shadow_for_id,omitempty"`
	ShadowForName *string `gorm:"type:varchar(100)" json:"shadow_for_name,omitempty"`

	// 操作
	Action       AuditActionType `gorm:"type:varchar(30);index" json:"action"`
	ResourceType string          `gorm:"type:varchar(50);index" json:"resource_type"` // shipment/document/user等
	ResourceID   string          `gorm:"type:varchar(50);index" json:"resource_id"`
	Description  string          `gorm:"type:varchar(500)" json:"description"`

	// 变更详情
	OldValue string `gorm:"type:text" json:"old_value,omitempty"` // JSON格式的旧值
	NewValue string `gorm:"type:text" json:"new_value,omitempty"` // JSON格式的新值

	// 请求信息
	Method     string `gorm:"type:varchar(10)" json:"method"`
	Path       string `gorm:"type:varchar(200)" json:"path"`
	StatusCode int    `json:"status_code"`
	ClientIP   string `gorm:"type:varchar(50)" json:"client_ip"`
	UserAgent  string `gorm:"type:varchar(500)" json:"user_agent,omitempty"`

	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// ==================== API请求/响应结构 ====================

// ShadowModeRequest 启用影子模式请求
type ShadowModeRequest struct {
	ShadowForType string `json:"shadow_for_type" binding:"required"` // user/partner/customer
	ShadowForID   string `json:"shadow_for_id" binding:"required"`
	Reason        string `json:"reason" binding:"required"` // 代操作原因
}

// ShadowOperationResponse 影子操作日志响应
type ShadowOperationResponse struct {
	ID            uint      `json:"id"`
	OperatorName  string    `json:"operator_name"`
	ShadowForName string    `json:"shadow_for_name"`
	Action        string    `json:"action"`
	Resource      string    `json:"resource"`
	Reason        string    `json:"reason"`
	ClientIP      string    `json:"client_ip"`
	CreatedAt     time.Time `json:"created_at"`
}

func (l *ShadowOperationLog) ToResponse() ShadowOperationResponse {
	return ShadowOperationResponse{
		ID:            l.ID,
		OperatorName:  l.OperatorName,
		ShadowForName: l.ShadowForName,
		Action:        l.Action,
		Resource:      l.Resource,
		Reason:        l.Reason,
		ClientIP:      l.ClientIP,
		CreatedAt:     l.CreatedAt,
	}
}

// AuditLogResponse 审计日志响应
type AuditLogResponse struct {
	ID           uint            `json:"id"`
	UserName     string          `json:"user_name"`
	IsShadowMode bool            `json:"is_shadow_mode"`
	ShadowFor    string          `json:"shadow_for,omitempty"`
	Action       AuditActionType `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Description  string          `json:"description"`
	Method       string          `json:"method"`
	Path         string          `json:"path"`
	ClientIP     string          `json:"client_ip"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (l *AuditLog) ToResponse() AuditLogResponse {
	resp := AuditLogResponse{
		ID:           l.ID,
		UserName:     l.UserName,
		IsShadowMode: l.IsShadowMode,
		Action:       l.Action,
		ResourceType: l.ResourceType,
		ResourceID:   l.ResourceID,
		Description:  l.Description,
		Method:       l.Method,
		Path:         l.Path,
		ClientIP:     l.ClientIP,
		CreatedAt:    l.CreatedAt,
	}
	if l.ShadowForName != nil {
		resp.ShadowFor = *l.ShadowForName
	}
	return resp
}

// AuditLogQuery 审计日志查询参数
type AuditLogQuery struct {
	UserID       string `form:"user_id"`
	Action       string `form:"action"`
	ResourceType string `form:"resource_type"`
	ResourceID   string `form:"resource_id"`
	IsShadowMode *bool  `form:"is_shadow_mode"`
	StartDate    string `form:"start_date"`
	EndDate      string `form:"end_date"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}
