package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContainerType 集装箱类型
type ContainerType string

const (
	Container20GP ContainerType = "20GP"
	Container40GP ContainerType = "40GP"
	Container40HQ ContainerType = "40HQ"
	Container45HQ ContainerType = "45HQ"
	ContainerLCL  ContainerType = "LCL" // 散货
)

// ContainerTypeNames 集装箱类型名称
var ContainerTypeNames = map[ContainerType]string{
	Container20GP: "20尺普柜",
	Container40GP: "40尺普柜",
	Container40HQ: "40尺高柜",
	Container45HQ: "45尺高柜",
	ContainerLCL:  "散货/拼箱",
}

// FreightRate 运价表
type FreightRate struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	PartnerID string `gorm:"type:varchar(50);index" json:"partner_id"` // 货代ID

	// 航线信息
	Origin          string `gorm:"type:varchar(100);index" json:"origin"`      // 起运港 (CNSHA)
	OriginName      string `gorm:"type:varchar(200)" json:"origin_name"`       // 起运港名称
	Destination     string `gorm:"type:varchar(100);index" json:"destination"` // 目的港 (USLAX)
	DestinationName string `gorm:"type:varchar(200)" json:"destination_name"`  // 目的港名称
	TransitDays     int    `gorm:"type:int" json:"transit_days"`               // 航程天数
	Carrier         string `gorm:"type:varchar(100)" json:"carrier"`           // 船司

	// 柜型与费用
	ContainerType ContainerType `gorm:"type:varchar(20)" json:"container_type"`
	Currency      string        `gorm:"type:varchar(10);default:'USD'" json:"currency"`

	// 费用明细
	OceanFreight float64 `gorm:"type:decimal(12,2)" json:"ocean_freight"` // 海运费
	BAF          float64 `gorm:"type:decimal(12,2)" json:"baf"`           // 燃油附加费
	CAF          float64 `gorm:"type:decimal(12,2)" json:"caf"`           // 汇率附加费
	PSS          float64 `gorm:"type:decimal(12,2)" json:"pss"`           // 旺季附加费
	GRI          float64 `gorm:"type:decimal(12,2)" json:"gri"`           // 综合费率上涨
	THC          float64 `gorm:"type:decimal(12,2)" json:"thc"`           // 码头操作费
	DocFee       float64 `gorm:"type:decimal(12,2)" json:"doc_fee"`       // 文件费
	SealFee      float64 `gorm:"type:decimal(12,2)" json:"seal_fee"`      // 铅封费
	OtherFee     float64 `gorm:"type:decimal(12,2)" json:"other_fee"`     // 其他费用
	TotalFee     float64 `gorm:"type:decimal(12,2)" json:"total_fee"`     // 总费用

	// 有效期
	ValidFrom time.Time `gorm:"index" json:"valid_from"`
	ValidTo   time.Time `gorm:"index" json:"valid_to"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`

	// 备注
	Remarks string `gorm:"type:varchar(500)" json:"remarks"`

	// 关联到创建者组织
	OwnerOrgID string `gorm:"type:varchar(50);index" json:"owner_org_id"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Partner *Partner `gorm:"foreignKey:PartnerID;references:ID" json:"partner,omitempty"`
}

func (r *FreightRate) BeforeCreate(tx *gorm.DB) error {
	// 自动计算总费用
	r.TotalFee = r.OceanFreight + r.BAF + r.CAF + r.PSS + r.GRI + r.THC + r.DocFee + r.SealFee + r.OtherFee
	return nil
}

func (r *FreightRate) BeforeUpdate(tx *gorm.DB) error {
	// 自动重新计算总费用
	r.TotalFee = r.OceanFreight + r.BAF + r.CAF + r.PSS + r.GRI + r.THC + r.DocFee + r.SealFee + r.OtherFee
	return nil
}

// TableName 设置表名
func (FreightRate) TableName() string {
	return "freight_rates"
}

// RateResponse 响应结构
type RateResponse struct {
	ID              uint          `json:"id"`
	PartnerID       string        `json:"partner_id"`
	PartnerName     string        `json:"partner_name,omitempty"`
	Origin          string        `json:"origin"`
	OriginName      string        `json:"origin_name"`
	Destination     string        `json:"destination"`
	DestinationName string        `json:"destination_name"`
	TransitDays     int           `json:"transit_days"`
	Carrier         string        `json:"carrier"`
	ContainerType   ContainerType `json:"container_type"`
	Currency        string        `json:"currency"`
	OceanFreight    float64       `json:"ocean_freight"`
	BAF             float64       `json:"baf"`
	CAF             float64       `json:"caf"`
	PSS             float64       `json:"pss"`
	GRI             float64       `json:"gri"`
	THC             float64       `json:"thc"`
	DocFee          float64       `json:"doc_fee"`
	SealFee         float64       `json:"seal_fee"`
	OtherFee        float64       `json:"other_fee"`
	TotalFee        float64       `json:"total_fee"`
	ValidFrom       time.Time     `json:"valid_from"`
	ValidTo         time.Time     `json:"valid_to"`
	IsActive        bool          `json:"is_active"`
	Remarks         string        `json:"remarks"`
}

func (r *FreightRate) ToResponse() RateResponse {
	resp := RateResponse{
		ID:              r.ID,
		PartnerID:       r.PartnerID,
		Origin:          r.Origin,
		OriginName:      r.OriginName,
		Destination:     r.Destination,
		DestinationName: r.DestinationName,
		TransitDays:     r.TransitDays,
		Carrier:         r.Carrier,
		ContainerType:   r.ContainerType,
		Currency:        r.Currency,
		OceanFreight:    r.OceanFreight,
		BAF:             r.BAF,
		CAF:             r.CAF,
		PSS:             r.PSS,
		GRI:             r.GRI,
		THC:             r.THC,
		DocFee:          r.DocFee,
		SealFee:         r.SealFee,
		OtherFee:        r.OtherFee,
		TotalFee:        r.TotalFee,
		ValidFrom:       r.ValidFrom,
		ValidTo:         r.ValidTo,
		IsActive:        r.IsActive,
		Remarks:         r.Remarks,
	}
	if r.Partner != nil {
		resp.PartnerName = r.Partner.Name
	}
	return resp
}

// RateCreateRequest 创建请求
type RateCreateRequest struct {
	PartnerID       string        `json:"partner_id" binding:"required"`
	Origin          string        `json:"origin" binding:"required"`
	OriginName      string        `json:"origin_name"`
	Destination     string        `json:"destination" binding:"required"`
	DestinationName string        `json:"destination_name"`
	TransitDays     int           `json:"transit_days"`
	Carrier         string        `json:"carrier"`
	ContainerType   ContainerType `json:"container_type" binding:"required"`
	Currency        string        `json:"currency"`
	OceanFreight    float64       `json:"ocean_freight"`
	BAF             float64       `json:"baf"`
	CAF             float64       `json:"caf"`
	PSS             float64       `json:"pss"`
	GRI             float64       `json:"gri"`
	THC             float64       `json:"thc"`
	DocFee          float64       `json:"doc_fee"`
	SealFee         float64       `json:"seal_fee"`
	OtherFee        float64       `json:"other_fee"`
	ValidFrom       time.Time     `json:"valid_from" binding:"required"`
	ValidTo         time.Time     `json:"valid_to" binding:"required"`
	Remarks         string        `json:"remarks"`
}

// RateQueryRequest 查询请求 - 智能比价
type RateQueryRequest struct {
	Origin        string        `json:"origin" binding:"required"`
	Destination   string        `json:"destination" binding:"required"`
	ContainerType ContainerType `json:"container_type"`
	ShipDate      *time.Time    `json:"ship_date"`
}

// PartnerPerformance 货代绩效统计
type PartnerPerformance struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	PartnerID string `gorm:"type:varchar(50);uniqueIndex:idx_partner_route" json:"partner_id"`
	RouteLane string `gorm:"type:varchar(100);uniqueIndex:idx_partner_route" json:"route_lane"` // 航线 (CNSHA-USLAX)

	// 统计数据
	TotalShipments   int     `gorm:"default:0" json:"total_shipments"`            // 总运单数
	OnTimeShipments  int     `gorm:"default:0" json:"on_time_shipments"`          // 准时数
	DelayedShipments int     `gorm:"default:0" json:"delayed_shipments"`          // 延误数
	OnTimeRate       float64 `gorm:"type:decimal(5,2)" json:"on_time_rate"`       // 准时率
	AvgTransitDays   float64 `gorm:"type:decimal(5,1)" json:"avg_transit_days"`   // 平均航程
	AvgDelayDays     float64 `gorm:"type:decimal(5,1)" json:"avg_delay_days"`     // 平均延误
	TotalClaim       float64 `gorm:"type:decimal(12,2)" json:"total_claim"`       // 理赔总额
	Rating           float64 `gorm:"type:decimal(3,2);default:5.0" json:"rating"` // 综合评分

	// 时间段
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Partner *Partner `gorm:"foreignKey:PartnerID;references:ID" json:"partner,omitempty"`
}

// TableName 设置表名
func (PartnerPerformance) TableName() string {
	return "partner_performances"
}

// GenerateID 生成唯一ID
func GenerateRateID() string {
	return "rate-" + uuid.New().String()[:8]
}
