package models

import (
	"time"

	"gorm.io/gorm"
)

// CarrierTrack 船司追踪记录 - 记录船务系统推送的事件数据
type CarrierTrack struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	ShipmentID   string `gorm:"type:varchar(50);index" json:"shipment_id"`
	BillOfLading string `gorm:"type:varchar(100);index" json:"bill_of_lading"` // B/L号 - 查询锚点

	// 船务状态事件
	EventCode string   `gorm:"type:varchar(50)" json:"event_code"`  // DEPARTURE, ARRIVAL, TRANSSHIP, GATE_OUT, GATE_IN...
	EventName string   `gorm:"type:varchar(100)" json:"event_name"` // 离港、到港、中转、提柜、还柜...
	Location  string   `gorm:"type:varchar(200)" json:"location"`   // 港口/位置名称
	LoCode    string   `gorm:"type:varchar(20)" json:"lo_code"`     // UN/LOCODE (如 CNSHA, USLAX)
	Latitude  *float64 `gorm:"type:decimal(10,6)" json:"latitude"`
	Longitude *float64 `gorm:"type:decimal(10,6)" json:"longitude"`

	// 船务信息
	VesselName string `gorm:"type:varchar(100)" json:"vessel_name"`
	VoyageNo   string `gorm:"type:varchar(50)" json:"voyage_no"`
	Carrier    string `gorm:"type:varchar(100)" json:"carrier"`

	// 时间信息
	EventTime time.Time  `gorm:"index" json:"event_time"`       // 事件发生时间
	ETAUpdate *time.Time `json:"eta_update"`                    // 最新ETA更新 (如有)
	IsActual  bool       `gorm:"default:true" json:"is_actual"` // 是否实际事件(vs 计划)

	// 元数据
	Source    string         `gorm:"type:varchar(50)" json:"source"` // vizion, p44, maersk, mock...
	RawData   string         `gorm:"type:text" json:"-"`             // 原始JSON (调试用)
	SyncedAt  time.Time      `gorm:"index" json:"synced_at"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Shipment *Shipment `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
}

// 标准事件代码常量
const (
	EventGateOut         = "GATE_OUT"         // 提柜出场
	EventLoadedOnVessel  = "LOADED"           // 装船
	EventVesselDeparture = "VESSEL_DEPARTURE" // 起运港离港
	EventTransshipment   = "TRANSSHIPMENT"    // 中转港
	EventVesselArrival   = "VESSEL_ARRIVAL"   // 目的港到港
	EventDischarge       = "DISCHARGE"        // 卸船
	EventGateIn          = "GATE_IN"          // 还柜进场
	EventCustomsHold     = "CUSTOMS_HOLD"     // 海关扣货
	EventCustomsRelease  = "CUSTOMS_RELEASE"  // 海关放行
	EventDelivery        = "DELIVERY"         // 签收
)

// EventCodeToName 事件代码转中文名称
var EventCodeToName = map[string]string{
	EventGateOut:         "提柜出场",
	EventLoadedOnVessel:  "装船完成",
	EventVesselDeparture: "船舶离港",
	EventTransshipment:   "中转换船",
	EventVesselArrival:   "船舶到港",
	EventDischarge:       "卸船完成",
	EventGateIn:          "还柜进场",
	EventCustomsHold:     "海关查验",
	EventCustomsRelease:  "海关放行",
	EventDelivery:        "签收完成",
}
