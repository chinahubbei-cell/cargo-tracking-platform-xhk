package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LogisticsProduct 物流产品定义
type LogisticsProduct struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`            // 产品名称：国际空运、整箱海运FCL
	Code        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"` // 产品代码：air_freight, fcl_sea
	Description string    `gorm:"type:text" json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	Templates []MilestoneTemplate `gorm:"foreignKey:ProductID" json:"templates,omitempty"`
}

func (p *LogisticsProduct) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = "prod-" + uuid.New().String()[:8]
	}
	return nil
}

// MilestoneTemplate 节点模板
type MilestoneTemplate struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	ProductID   string    `gorm:"type:varchar(50);index;not null" json:"product_id"` // 关联物流产品
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`            // 模板名称：标准空运流程
	Description string    `gorm:"type:text" json:"description"`
	Version     int       `gorm:"default:1" json:"version"`        // 版本号
	IsActive    bool      `gorm:"default:true" json:"is_active"`   // 是否启用
	IsDefault   bool      `gorm:"default:false" json:"is_default"` // 是否默认模板
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	Product LogisticsProduct `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Nodes   []MilestoneNode  `gorm:"foreignKey:TemplateID" json:"nodes,omitempty"`
}

func (t *MilestoneTemplate) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = "tpl-" + uuid.New().String()[:8]
	}
	return nil
}

// MilestoneNodeType 节点类型
type MilestoneNodeType string

const (
	NodeTypeStandard    MilestoneNodeType = "standard"    // 标准节点
	NodeTypeParallel    MilestoneNodeType = "parallel"    // 并行节点（同时执行）
	NodeTypeConditional MilestoneNodeType = "conditional" // 条件节点（根据条件选择分支）
	NodeTypeGateway     MilestoneNodeType = "gateway"     // 网关节点（多分支汇合）
)

// MilestoneNode 节点定义
type MilestoneNode struct {
	ID           string            `gorm:"primaryKey;type:varchar(50)" json:"id"`
	TemplateID   string            `gorm:"type:varchar(50);index;not null" json:"template_id"`     // 关联模板
	NodeCode     string            `gorm:"type:varchar(50);not null;default:''" json:"node_code"`  // 节点代码：pickup, customs_export
	NodeName     string            `gorm:"type:varchar(100);not null;default:''" json:"node_name"` // 节点名称：提货、出口报关
	NodeNameEn   string            `gorm:"type:varchar(100)" json:"node_name_en"`                  // 英文名称
	NodeType     MilestoneNodeType `gorm:"type:varchar(20);default:'standard'" json:"node_type"`
	NodeOrder    int               `gorm:"not null" json:"node_order"`             // 顺序
	ParentNodeID *string           `gorm:"type:varchar(50)" json:"parent_node_id"` // 父节点（支持并行）
	GroupCode    *string           `gorm:"type:varchar(50)" json:"group_code"`     // 分组代码（如：first_mile, origin_port）
	GroupName    *string           `gorm:"type:varchar(100)" json:"group_name"`    // 分组名称
	IsMandatory  bool              `gorm:"default:true" json:"is_mandatory"`       // 是否必须
	IsVisible    bool              `gorm:"default:true" json:"is_visible"`         // 是否对客户可见
	Icon         string            `gorm:"type:varchar(50)" json:"icon"`           // 图标

	// 触发条件
	TriggerType       string         `gorm:"type:varchar(50)" json:"trigger_type"` // manual, api, geofence, event, time
	TriggerConditions datatypes.JSON `gorm:"type:json" json:"trigger_conditions"`  // 触发条件JSON

	// 超时配置
	TimeoutHours  int    `gorm:"default:0" json:"timeout_hours"`         // 超时小时数，0表示不设超时
	TimeoutAction string `gorm:"type:varchar(50)" json:"timeout_action"` // 超时动作：alert, escalate, auto_complete

	// 关联角色/任务
	ResponsibleRole string `gorm:"type:varchar(50)" json:"responsible_role"` // 负责角色：shipper, carrier, customs_broker

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Template   MilestoneTemplate `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	ParentNode *MilestoneNode    `gorm:"foreignKey:ParentNodeID" json:"parent_node,omitempty"`
}

func (n *MilestoneNode) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = "node-" + uuid.New().String()[:8]
	}
	return nil
}

// ----------------------
// 运单节点实例（运行时数据）
// ----------------------

// ShipmentMilestone 运单节点实例
type ShipmentMilestone struct {
	ID         string      `gorm:"primaryKey;type:varchar(50)" json:"id"`
	ShipmentID string      `gorm:"type:varchar(50);index;not null" json:"shipment_id"`
	NodeID     string      `gorm:"type:varchar(50);index" json:"node_id"` // 关联节点定义
	NodeCode   string      `gorm:"type:varchar(50);not null;default:''" json:"node_code"`
	NodeName   string      `gorm:"type:varchar(100);not null;default:''" json:"node_name"`
	NodeOrder  int         `gorm:"not null;default:0" json:"node_order"`
	GroupCode  *string     `gorm:"type:varchar(50)" json:"group_code"`
	GroupName  *string     `gorm:"type:varchar(100)" json:"group_name"`
	Status     StageStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	// 时间
	PlannedStart *time.Time `json:"planned_start"`
	PlannedEnd   *time.Time `json:"planned_end"`
	ActualStart  *time.Time `json:"actual_start"`
	ActualEnd    *time.Time `json:"actual_end"`

	// 触发信息
	TriggerType TriggerType    `gorm:"type:varchar(20)" json:"trigger_type"`
	TriggerNote string         `gorm:"type:text" json:"trigger_note"`
	TriggerData datatypes.JSON `gorm:"type:json" json:"trigger_data"` // 触发时的原始数据

	// 负责方
	PartnerID   *string `gorm:"type:varchar(50)" json:"partner_id"`
	PartnerName string  `gorm:"type:varchar(100)" json:"partner_name"`

	// 地点
	LocationName string  `gorm:"type:varchar(200)" json:"location_name"`
	Latitude     float64 `gorm:"type:decimal(10,7)" json:"latitude"`
	Longitude    float64 `gorm:"type:decimal(10,7)" json:"longitude"`

	// 备注和附件
	Remarks     string         `gorm:"type:text" json:"remarks"`
	Attachments datatypes.JSON `gorm:"type:json" json:"attachments"` // [{name, url, type}]

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Node *MilestoneNode `gorm:"foreignKey:NodeID" json:"node,omitempty"`
}

func (m *ShipmentMilestone) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = "sm-" + uuid.New().String()[:8]
	}
	return nil
}

// ----------------------
// API 请求/响应类型
// ----------------------

// CreateProductRequest 创建物流产品请求
type CreateProductRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	ProductID   string `json:"product_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

// CreateNodeRequest 创建节点请求
type CreateNodeRequest struct {
	TemplateID        string            `json:"template_id" binding:"required"`
	NodeCode          string            `json:"node_code" binding:"required"`
	NodeName          string            `json:"node_name" binding:"required"`
	NodeNameEn        string            `json:"node_name_en"`
	NodeType          MilestoneNodeType `json:"node_type"`
	NodeOrder         int               `json:"node_order" binding:"required"`
	ParentNodeID      *string           `json:"parent_node_id"`
	GroupCode         *string           `json:"group_code"`
	GroupName         *string           `json:"group_name"`
	IsMandatory       bool              `json:"is_mandatory"`
	IsVisible         bool              `json:"is_visible"`
	Icon              string            `json:"icon"`
	TriggerType       string            `json:"trigger_type"`
	TriggerConditions map[string]any    `json:"trigger_conditions"`
	TimeoutHours      int               `json:"timeout_hours"`
	TimeoutAction     string            `json:"timeout_action"`
	ResponsibleRole   string            `json:"responsible_role"`
}

// ProductResponse 物流产品响应
type ProductResponse struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Code          string             `json:"code"`
	Description   string             `json:"description"`
	IsActive      bool               `json:"is_active"`
	TemplateCount int                `json:"template_count"`
	Templates     []TemplateResponse `json:"templates,omitempty"`
}

// TemplateResponse 模板响应
type TemplateResponse struct {
	ID          string         `json:"id"`
	ProductID   string         `json:"product_id"`
	ProductName string         `json:"product_name"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     int            `json:"version"`
	IsActive    bool           `json:"is_active"`
	IsDefault   bool           `json:"is_default"`
	NodeCount   int            `json:"node_count"`
	Nodes       []NodeResponse `json:"nodes,omitempty"`
}

// NodeResponse 节点响应
type NodeResponse struct {
	ID              string            `json:"id"`
	NodeCode        string            `json:"node_code"`
	NodeName        string            `json:"node_name"`
	NodeNameEn      string            `json:"node_name_en"`
	NodeType        MilestoneNodeType `json:"node_type"`
	NodeOrder       int               `json:"node_order"`
	ParentNodeID    *string           `json:"parent_node_id"`
	GroupCode       *string           `json:"group_code"`
	GroupName       *string           `json:"group_name"`
	IsMandatory     bool              `json:"is_mandatory"`
	IsVisible       bool              `json:"is_visible"`
	Icon            string            `json:"icon"`
	TriggerType     string            `json:"trigger_type"`
	TimeoutHours    int               `json:"timeout_hours"`
	TimeoutAction   string            `json:"timeout_action"`
	ResponsibleRole string            `json:"responsible_role"`
}
