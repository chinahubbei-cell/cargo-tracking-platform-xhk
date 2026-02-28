package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LogisticsStage 物流环节
type LogisticsStage string

const (
	StageFirstMile  LogisticsStage = "first_mile"  // 前程运输
	StageOriginPort LogisticsStage = "origin_port" // 起运港
	StageMainLeg    LogisticsStage = "main_leg"    // 干线运输
	StageDestPort   LogisticsStage = "dest_port"   // 目的港
	StageLastMile   LogisticsStage = "last_mile"   // 末端配送
)

// PartnerType 合作伙伴类型 (扩展到16种)
type PartnerType string

const (
	// 前程运输 First Mile
	PartnerTypeDrayageOrigin PartnerType = "drayage_origin" // 首程拖车
	PartnerTypeConsolidator  PartnerType = "consolidator"   // 集运/拼箱公司 CFS
	PartnerTypeInspector     PartnerType = "inspector"      // 质检机构 SGS/BV

	// 起运港 Origin Port
	PartnerTypeBookingAgent  PartnerType = "booking_agent"  // 订舱代理
	PartnerTypeCustomsExport PartnerType = "customs_export" // 出口报关行
	PartnerTypeTerminal      PartnerType = "terminal"       // 码头/堆场运营商

	// 干线运输 Main Leg
	PartnerTypeVOCC    PartnerType = "vocc"    // 船公司 (实际承运人)
	PartnerTypeNVOCC   PartnerType = "nvocc"   // 无船承运人
	PartnerTypeAirline PartnerType = "airline" // 航空公司

	// 目的港 Destination Port
	PartnerTypeCustomsImport   PartnerType = "customs_import"   // 进口清关行
	PartnerTypeDrayageDest     PartnerType = "drayage_dest"     // 目的港拖车
	PartnerTypeChassisProvider PartnerType = "chassis_provider" // 底盘车队
	PartnerTypeBondedWarehouse PartnerType = "bonded_warehouse" // 保税仓

	// 末端配送 Last Mile
	PartnerTypeOverseas3PL       PartnerType = "overseas_3pl"       // 海外仓服务商
	PartnerTypeCourier           PartnerType = "courier"            // 快递公司 UPS/FedEx
	PartnerTypePlatformWarehouse PartnerType = "platform_warehouse" // 电商平台仓 FBA/WFS

	// 兼容旧类型
	PartnerTypeForwarder PartnerType = "forwarder" // 货代(通用)
	PartnerTypeBroker    PartnerType = "broker"    // 报关行(通用)
	PartnerTypeTrucker   PartnerType = "trucker"   // 拖车行(通用)
	PartnerTypeWarehouse PartnerType = "warehouse" // 仓库(通用)
)

// Partner 合作伙伴模型 - 外部协作方
type Partner struct {
	ID   string      `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name string      `gorm:"type:varchar(100);not null" json:"name"`
	Code string      `gorm:"uniqueIndex;type:varchar(50)" json:"code"`
	Type PartnerType `gorm:"type:varchar(30)" json:"type"`

	// 物流环节与细分
	Stage   LogisticsStage `gorm:"type:varchar(20);index" json:"stage"` // 所属物流环节
	SubType string         `gorm:"type:varchar(50)" json:"sub_type"`    // 细分类型

	// 联系信息
	ContactName string `gorm:"type:varchar(100)" json:"contact_name"`
	Phone       string `gorm:"type:varchar(50)" json:"phone"`
	Email       string `gorm:"type:varchar(100)" json:"email"`
	Address     string `gorm:"type:varchar(500)" json:"address"`

	// 服务区域
	Country string `gorm:"type:varchar(100)" json:"country"` // 服务国家
	Region  string `gorm:"type:varchar(200)" json:"region"`  // 服务区域

	// 业务信息
	ServicePorts   string   `gorm:"type:varchar(500)" json:"service_ports"`      // 服务港口(逗号分隔)
	ServiceRoutes  string   `gorm:"type:varchar(500)" json:"service_routes"`     // 服务航线(逗号分隔)
	Rating         float64  `gorm:"type:decimal(3,2);default:5.0" json:"rating"` // 评分 0-5
	TotalShipments int      `gorm:"default:0" json:"total_shipments"`            // 合作运单数
	OnTimeRate     *float64 `gorm:"type:decimal(5,2)" json:"on_time_rate"`       // 准时率

	// 资质与能力 (JSON存储)
	Certifications      string `gorm:"type:text" json:"certifications"`       // 资质证书JSON
	APIConfig           string `gorm:"type:text" json:"api_config"`           // API集成配置JSON
	ServiceCapabilities string `gorm:"type:text" json:"service_capabilities"` // 服务能力JSON

	// 合同与财务
	ContractInfo      string   `gorm:"type:text" json:"contract_info"`               // 合同信息JSON
	PaymentTerms      string   `gorm:"type:varchar(100)" json:"payment_terms"`       // 账期 如 NET30
	InsuranceCoverage *float64 `gorm:"type:decimal(15,2)" json:"insurance_coverage"` // 保险额度

	// 状态
	Status string `gorm:"type:varchar(20);default:'active'" json:"status"` // active/inactive

	// 关联到创建者组织 (数据隔离)
	OwnerOrgID string `gorm:"type:varchar(50);index" json:"owner_org_id"`

	// 关联用户 (可选，如果Partner需要登录系统)
	UserID *string `gorm:"type:varchar(50)" json:"user_id"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (p *Partner) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = "partner-" + uuid.New().String()[:8]
	}
	return nil
}

// PartnerResponse 响应结构
type PartnerResponse struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Code                string         `json:"code"`
	Type                PartnerType    `json:"type"`
	Stage               LogisticsStage `json:"stage"`
	SubType             string         `json:"sub_type"`
	ContactName         string         `json:"contact_name"`
	Phone               string         `json:"phone"`
	Email               string         `json:"email"`
	Address             string         `json:"address"`
	Country             string         `json:"country"`
	Region              string         `json:"region"`
	ServicePorts        string         `json:"service_ports"`
	ServiceRoutes       string         `json:"service_routes"`
	Rating              float64        `json:"rating"`
	TotalShipments      int            `json:"total_shipments"`
	OnTimeRate          *float64       `json:"on_time_rate"`
	Certifications      string         `json:"certifications"`
	APIConfig           string         `json:"api_config"`
	ServiceCapabilities string         `json:"service_capabilities"`
	ContractInfo        string         `json:"contract_info"`
	PaymentTerms        string         `json:"payment_terms"`
	InsuranceCoverage   *float64       `json:"insurance_coverage"`
	Status              string         `json:"status"`
	OwnerOrgID          string         `json:"owner_org_id"`
	CreatedAt           time.Time      `json:"created_at"`
}

func (p *Partner) ToResponse() PartnerResponse {
	return PartnerResponse{
		ID:                  p.ID,
		Name:                p.Name,
		Code:                p.Code,
		Type:                p.Type,
		Stage:               p.Stage,
		SubType:             p.SubType,
		ContactName:         p.ContactName,
		Phone:               p.Phone,
		Email:               p.Email,
		Address:             p.Address,
		Country:             p.Country,
		Region:              p.Region,
		ServicePorts:        p.ServicePorts,
		ServiceRoutes:       p.ServiceRoutes,
		Rating:              p.Rating,
		TotalShipments:      p.TotalShipments,
		OnTimeRate:          p.OnTimeRate,
		Certifications:      p.Certifications,
		APIConfig:           p.APIConfig,
		ServiceCapabilities: p.ServiceCapabilities,
		ContractInfo:        p.ContractInfo,
		PaymentTerms:        p.PaymentTerms,
		InsuranceCoverage:   p.InsuranceCoverage,
		Status:              p.Status,
		OwnerOrgID:          p.OwnerOrgID,
		CreatedAt:           p.CreatedAt,
	}
}

// PartnerCreateRequest 创建请求
type PartnerCreateRequest struct {
	Name                string         `json:"name" binding:"required"`
	Code                string         `json:"code" binding:"required"`
	Type                PartnerType    `json:"type" binding:"required"`
	Stage               LogisticsStage `json:"stage"`
	SubType             string         `json:"sub_type"`
	ContactName         string         `json:"contact_name"`
	Phone               string         `json:"phone"`
	Email               string         `json:"email"`
	Address             string         `json:"address"`
	Country             string         `json:"country"`
	Region              string         `json:"region"`
	ServicePorts        string         `json:"service_ports"`
	ServiceRoutes       string         `json:"service_routes"`
	Certifications      string         `json:"certifications"`
	APIConfig           string         `json:"api_config"`
	ServiceCapabilities string         `json:"service_capabilities"`
	ContractInfo        string         `json:"contract_info"`
	PaymentTerms        string         `json:"payment_terms"`
	InsuranceCoverage   *float64       `json:"insurance_coverage"`
}

// PartnerUpdateRequest 更新请求
type PartnerUpdateRequest struct {
	Name                *string  `json:"name"`
	SubType             *string  `json:"sub_type"`
	ContactName         *string  `json:"contact_name"`
	Phone               *string  `json:"phone"`
	Email               *string  `json:"email"`
	Address             *string  `json:"address"`
	Country             *string  `json:"country"`
	Region              *string  `json:"region"`
	ServicePorts        *string  `json:"service_ports"`
	ServiceRoutes       *string  `json:"service_routes"`
	Rating              *float64 `json:"rating"`
	Certifications      *string  `json:"certifications"`
	APIConfig           *string  `json:"api_config"`
	ServiceCapabilities *string  `json:"service_capabilities"`
	ContractInfo        *string  `json:"contract_info"`
	PaymentTerms        *string  `json:"payment_terms"`
	InsuranceCoverage   *float64 `json:"insurance_coverage"`
	Status              *string  `json:"status"`
}
