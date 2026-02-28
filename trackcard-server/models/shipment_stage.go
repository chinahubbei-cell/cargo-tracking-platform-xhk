package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StageCode 运输环节代码
type StageCode string

const (
	// 主运输环节7个（含中转）或6个（无中转）
	SSPreTransit  StageCode = "pre_transit"  // 前程运输
	SSOriginPort  StageCode = "origin_port"  // 起运港
	SSMainLine    StageCode = "main_line"    // 干线运输
	SSTransitPort StageCode = "transit_port" // 中转港（可选）
	SSDestPort    StageCode = "dest_port"    // 目的港
	SSLastMile    StageCode = "last_mile"    // 末端配送
	SSDelivered   StageCode = "delivered"    // 签收

	// 硬件自动触发子事件6个（记录日志用）
	SSOriginArrival    StageCode = "origin_arrival"    // 起运港到港
	SSOriginDeparture  StageCode = "origin_departure"  // 起运港离港
	SSTransitArrival   StageCode = "transit_arrival"   // 中转港到港
	SSTransitDeparture StageCode = "transit_departure" // 中转港离港
	SSDestArrival      StageCode = "dest_arrival"      // 目的港到港
	SSDestDeparture    StageCode = "dest_departure"    // 目的港离港

	// 兼容旧版环节代码
	SSPickup       StageCode = "pickup"        // 旧版：揽收 -> pre_transit
	SSFirstMile    StageCode = "first_mile"    // 旧版：前程运输 -> pre_transit
	SSMainCarriage StageCode = "main_carriage" // 旧版：干线运输 -> main_line
	SSDelivery     StageCode = "delivery"      // 旧版：末端派送 -> last_mile
)

// StageStatus 环节状态
type StageStatus string

const (
	StageStatusPending    StageStatus = "pending"     // 待开始
	StageStatusInProgress StageStatus = "in_progress" // 进行中
	StageStatusCompleted  StageStatus = "completed"   // 已完成
	StageStatusSkipped    StageStatus = "skipped"     // 跳过
)

// TriggerType 触发方式
type TriggerType string

const (
	TriggerManual   TriggerType = "manual"   // 手工操作
	TriggerGeofence TriggerType = "geofence" // 电子围栏自动触发
	TriggerAPI      TriggerType = "api"      // API回传
)

// ShipmentStage 运单环节详情
type ShipmentStage struct {
	ID         string      `gorm:"primaryKey;type:varchar(50)" json:"id"`
	ShipmentID string      `gorm:"type:varchar(50);index;not null" json:"shipment_id"`
	StageCode  StageCode   `gorm:"type:varchar(20);not null" json:"stage_code"`
	StageOrder int         `gorm:"not null" json:"stage_order"` // 环节顺序: 1-5
	Status     StageStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	// 环节负责方（合作伙伴）
	PartnerID   *string `gorm:"type:varchar(50)" json:"partner_id"`
	PartnerName string  `gorm:"type:varchar(100)" json:"partner_name"`
	PartnerType string  `gorm:"type:varchar(50)" json:"partner_type"` // 合作伙伴类型

	// 前程运输特有字段
	VehiclePlate string     `gorm:"type:varchar(20)" json:"vehicle_plate"` // 拖车车牌
	PickupTime   *time.Time `json:"pickup_time"`                           // 提货时间
	WarehouseID  *string    `gorm:"type:varchar(50)" json:"warehouse_id"`  // 集货仓ID

	// 起运港/干线特有字段
	VesselName string `gorm:"type:varchar(100)" json:"vessel_name"` // 船名
	VoyageNo   string `gorm:"type:varchar(50)" json:"voyage_no"`    // 航次
	Carrier    string `gorm:"type:varchar(100)" json:"carrier"`     // 船司/航司
	PortCode   string `gorm:"type:varchar(20)" json:"port_code"`    // 港口代码

	// 末端配送特有字段
	DeliveryType   string     `gorm:"type:varchar(20)" json:"delivery_type"` // 配送类型
	ReceiverName   string     `gorm:"type:varchar(100)" json:"receiver_name"`
	ReceivedAt     *time.Time `json:"received_at"`
	SignatureImage string     `gorm:"type:varchar(500)" json:"signature_image"`

	// 时间节点
	PlannedStart *time.Time `json:"planned_start"` // 计划开始时间
	PlannedEnd   *time.Time `json:"planned_end"`   // 计划结束时间
	ActualStart  *time.Time `json:"actual_start"`  // 实际开始时间
	ActualEnd    *time.Time `json:"actual_end"`    // 实际结束时间

	// 触发信息
	TriggerType TriggerType `gorm:"type:varchar(20)" json:"trigger_type"`  // 触发方式
	GeofenceID  *string     `gorm:"type:varchar(50)" json:"geofence_id"`   // 触发的围栏ID
	TriggerNote string      `gorm:"type:varchar(500)" json:"trigger_note"` // 触发备注

	// 费用信息
	CostName string  `gorm:"type:varchar(100)" json:"cost_name"` // 费用名称（如：干线运输费）
	Cost     float64 `gorm:"type:decimal(12,2);default:0" json:"cost"`
	Currency string  `gorm:"type:varchar(3);default:'CNY'" json:"currency"`
	CostNote string  `gorm:"type:varchar(500)" json:"cost_note"` // 费用备注

	// 附加数据（JSON格式存储环节特有数据）
	ExtraData string `gorm:"type:text" json:"extra_data"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Shipment *Shipment `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
	Partner  *Partner  `gorm:"foreignKey:PartnerID;references:ID" json:"partner,omitempty"`
}

// BeforeCreate 创建前自动生成UUID
func (s *ShipmentStage) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// GetStageOrder 根据环节代码获取顺序
func GetStageOrder(code StageCode) int {
	switch code {
	case SSPreTransit, SSPickup, SSFirstMile:
		return 1
	case SSOriginArrival:
		return 2
	case SSOriginDeparture, SSOriginPort:
		return 3
	case SSMainLine, SSMainCarriage:
		return 4
	case SSTransitArrival:
		return 5
	case SSTransitDeparture, SSTransitPort:
		return 6
	case SSDestArrival, SSDestPort:
		return 7
	case SSDestDeparture:
		return 8
	case SSDelivery, SSLastMile:
		return 9
	case SSDelivered:
		return 10
	default:
		return 0
	}
}

// GetStageName 获取环节中文名称
func GetStageName(code StageCode) string {
	switch code {
	case SSPreTransit, SSPickup, SSFirstMile:
		return "前程运输"
	case SSOriginPort:
		return "起运港"
	case SSOriginArrival:
		return "起运港到港"
	case SSOriginDeparture:
		return "起运港离港"
	case SSMainLine, SSMainCarriage:
		return "干线运输"
	case SSTransitPort:
		return "中转港"
	case SSTransitArrival:
		return "中转港到港"
	case SSTransitDeparture:
		return "中转港离港"
	case SSDestPort:
		return "目的港"
	case SSDestArrival:
		return "目的港到港"
	case SSDestDeparture:
		return "目的港离港"
	case SSLastMile, SSDelivery:
		return "末端配送"
	case SSDelivered:
		return "签收"
	default:
		return "未知环节"
	}
}

// GetStageCostName 获取环节默认费用名称
func GetStageCostName(code StageCode) string {
	switch code {
	case SSPreTransit, SSPickup, SSFirstMile:
		return "前程运输费"
	case SSOriginPort:
		return "起运港操作费"
	case SSOriginArrival:
		return "起运港到港费"
	case SSOriginDeparture:
		return "起运港离港费"
	case SSMainLine, SSMainCarriage:
		return "干线运输费"
	case SSTransitPort:
		return "中转港操作费"
	case SSTransitArrival:
		return "中转港到港费"
	case SSTransitDeparture:
		return "中转港离港费"
	case SSDestPort:
		return "目的港操作费"
	case SSDestArrival:
		return "目的港到港费"
	case SSDestDeparture:
		return "目的港离港费"
	case SSLastMile, SSDelivery:
		return "末端配送费"
	case SSDelivered:
		return "签收服务费"
	default:
		return "其他费用"
	}
}

// GetStageIcon 获取环节图标
func GetStageIcon(code StageCode) string {
	switch code {
	case SSPreTransit, SSPickup, SSFirstMile:
		return "🚚"
	case SSOriginPort:
		return "🚢"
	case SSOriginArrival:
		return "🚢⬇️"
	case SSOriginDeparture:
		return "🚢⬆️"
	case SSMainLine, SSMainCarriage:
		return "🌊"
	case SSTransitPort:
		return "⚓"
	case SSTransitArrival:
		return "⚓⬇️"
	case SSTransitDeparture:
		return "⚓⬆️"
	case SSDestPort:
		return "🏁"
	case SSDestArrival:
		return "🏁⬇️"
	case SSDestDeparture:
		return "🏁⬆️"
	case SSDelivery, SSLastMile:
		return "🚚"
	case SSDelivered:
		return "✅"
	default:
		return "❓"
	}
}

// AllStageCodes 返回跨境运输环节代码（包含中转港，7个主环节）
func AllStageCodes() []StageCode {
	return []StageCode{
		SSPreTransit,  // 1. 前程运输
		SSOriginPort,  // 2. 起运港
		SSMainLine,    // 3. 干线运输
		SSTransitPort, // 4. 中转港
		SSDestPort,    // 5. 目的港
		SSLastMile,    // 6. 末端配送
		SSDelivered,   // 7. 签收
	}
}

// AllStageCodesWithoutTransit 返回无中转的环节代码（直航，6个主环节）
func AllStageCodesWithoutTransit() []StageCode {
	return []StageCode{
		SSPreTransit, // 1. 前程运输
		SSOriginPort, // 2. 起运港
		SSMainLine,   // 3. 干线运输
		SSDestPort,   // 4. 目的港
		SSLastMile,   // 5. 末端配送
		SSDelivered,  // 6. 签收
	}
}

// IsTransitStage 判断是否为中转环节
func IsTransitStage(code StageCode) bool {
	return code == SSTransitPort || code == SSTransitArrival || code == SSTransitDeparture
}

// RouteType 线路类型
type RouteType string

const (
	RouteTypeDomestic    RouteType = "domestic"     // 境内运输
	RouteTypeCrossBorder RouteType = "cross_border" // 跨境运输
)

// DomesticStageCodes 返回境内运输环节代码（简化版3个环节）
func DomesticStageCodes() []StageCode {
	return []StageCode{
		SSPreTransit, // 前程运输
		SSLastMile,   // 末端配送
		SSDelivered,  // 签收
	}
}

// GetStageCodesForRoute 根据路由配置获取环节列表
// hasTransitPort: 是否有中转港（基于线路规划数据判断）
func GetStageCodesForRoute(routeType RouteType, hasTransitPort bool) []StageCode {
	if routeType == RouteTypeDomestic {
		return DomesticStageCodes()
	}
	if hasTransitPort {
		return AllStageCodes() // 包含中转港
	}
	return AllStageCodesWithoutTransit() // 直航，无中转港
}

// ShipmentStageUpdateRequest 环节更新请求
type ShipmentStageUpdateRequest struct {
	Status       *StageStatus `json:"status"`
	PartnerID    *string      `json:"partner_id"`
	PartnerName  *string      `json:"partner_name"`
	VehiclePlate *string      `json:"vehicle_plate"`
	VesselName   *string      `json:"vessel_name"`
	VoyageNo     *string      `json:"voyage_no"`
	Carrier      *string      `json:"carrier"`
	ActualStart  *time.Time   `json:"actual_start"`
	ActualEnd    *time.Time   `json:"actual_end"`
	CostName     *string      `json:"cost_name"`
	Cost         *float64     `json:"cost"`
	Currency     *string      `json:"currency"`
	TriggerType  *TriggerType `json:"trigger_type"`
	TriggerNote  *string      `json:"trigger_note"`
	ExtraData    *string      `json:"extra_data"`
}

// ShipmentStageResponse 环节响应结构
type ShipmentStageResponse struct {
	ID          string      `json:"id"`
	ShipmentID  string      `json:"shipment_id"`
	StageCode   StageCode   `json:"stage_code"`
	StageName   string      `json:"stage_name"`
	StageIcon   string      `json:"stage_icon"`
	StageOrder  int         `json:"stage_order"`
	Status      StageStatus `json:"status"`
	PartnerName string      `json:"partner_name"`

	// 关键数据
	VehiclePlate string `json:"vehicle_plate,omitempty"`
	VesselName   string `json:"vessel_name,omitempty"`
	VoyageNo     string `json:"voyage_no,omitempty"`
	Carrier      string `json:"carrier,omitempty"`
	PortCode     string `json:"port_code,omitempty"` // 港口代码

	// 港口坐标（用于地图渲染）
	PortLat  float64 `json:"port_lat,omitempty"`
	PortLng  float64 `json:"port_lng,omitempty"`
	PortName string  `json:"port_name,omitempty"`

	// 时间
	PlannedStart *time.Time `json:"planned_start,omitempty"`
	PlannedEnd   *time.Time `json:"planned_end,omitempty"`
	ActualStart  *time.Time `json:"actual_start,omitempty"`
	ActualEnd    *time.Time `json:"actual_end,omitempty"`

	// 费用
	CostName string  `json:"cost_name"` // 费用名称（如：干线运输费）
	Cost     float64 `json:"cost"`
	Currency string  `json:"currency"`

	// 触发信息
	TriggerType TriggerType `json:"trigger_type,omitempty"`
	TriggerNote string      `json:"trigger_note,omitempty"`
}

// ToResponse 转换为响应结构
func (s *ShipmentStage) ToResponse() ShipmentStageResponse {
	// 费用名称：优先使用存储的名称，否则使用默认名称
	costName := s.CostName
	if costName == "" {
		costName = GetStageCostName(s.StageCode)
	}

	return ShipmentStageResponse{
		ID:           s.ID,
		ShipmentID:   s.ShipmentID,
		StageCode:    s.StageCode,
		StageName:    GetStageName(s.StageCode),
		StageIcon:    GetStageIcon(s.StageCode),
		StageOrder:   s.StageOrder,
		Status:       s.Status,
		PartnerName:  s.PartnerName,
		VehiclePlate: s.VehiclePlate,
		VesselName:   s.VesselName,
		VoyageNo:     s.VoyageNo,
		Carrier:      s.Carrier,
		PortCode:     s.PortCode,
		PlannedStart: s.PlannedStart,
		PlannedEnd:   s.PlannedEnd,
		ActualStart:  s.ActualStart,
		ActualEnd:    s.ActualEnd,
		CostName:     costName,
		Cost:         s.Cost,
		Currency:     s.Currency,
		TriggerType:  s.TriggerType,
		TriggerNote:  s.TriggerNote,
	}
}
