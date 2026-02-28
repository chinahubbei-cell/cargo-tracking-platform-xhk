package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Shipment struct {
	ID                string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	DeviceID          *string        `gorm:"type:varchar(50);column:device_id" json:"device_id"`
	TransportType     string         `gorm:"type:varchar(20);column:transport_type" json:"transport_type"` // 运输类型: sea/air/land/multimodal
	TransportMode     string         `gorm:"type:varchar(20);column:transport_mode" json:"transport_mode"` // 运输模式: lcl/fcl
	ContainerType     string         `gorm:"type:varchar(20);column:container_type" json:"container_type"` // 柜型: 20GP/40GP/40HQ
	CargoType         string         `gorm:"type:varchar(20);column:cargo_type" json:"cargo_type"`         // 货物类型: general/dangerous/cold_chain
	CargoName         string         `gorm:"type:varchar(255);column:cargo_name" json:"cargo_name"`        // 货物名称
	Origin            string         `gorm:"type:varchar(255);not null" json:"origin"`
	Destination       string         `gorm:"type:varchar(255);not null" json:"destination"`
	OriginLat         *float64       `gorm:"type:decimal(10,6);column:origin_lat" json:"origin_lat"`
	OriginLng         *float64       `gorm:"type:decimal(10,6);column:origin_lng" json:"origin_lng"`
	OriginAddress     string         `gorm:"type:varchar(500);column:origin_address" json:"origin_address"` // 发货详细地址
	DestLat           *float64       `gorm:"type:decimal(10,6);column:dest_lat" json:"dest_lat"`
	DestLng           *float64       `gorm:"type:decimal(10,6);column:dest_lng" json:"dest_lng"`
	DestAddress       string         `gorm:"type:varchar(500);column:dest_address" json:"dest_address"`                        // 收货详细地址
	OriginRadius      int            `gorm:"default:1000;column:origin_radius" json:"origin_radius"`                           // 发货地围栏半径(米)
	DestRadius        int            `gorm:"default:1000;column:dest_radius" json:"dest_radius"`                               // 目的地围栏半径(米)
	AutoStatusEnabled bool           `gorm:"default:true;column:auto_status_enabled" json:"auto_status_enabled"`               // 启用自动状态切换
	CurrentMilestone  string         `gorm:"type:varchar(50);column:current_milestone" json:"current_milestone"`               // 当前里程碑
	CurrentStage      string         `gorm:"type:varchar(20);column:current_stage;default:'pre_transit'" json:"current_stage"` // 当前运输环节
	Status            string         `gorm:"type:varchar(20);default:'pending';index:idx_shipments_status;index:idx_shipments_status_created" json:"status"`
	Progress          int            `gorm:"default:0" json:"progress"`
	DepartureTime     *time.Time     `gorm:"column:departure_time" json:"departure_time"`
	ETA               *time.Time     `json:"eta"`
	DeviceBoundAt     *time.Time     `gorm:"column:device_bound_at" json:"device_bound_at"`
	LeftOriginAt      *time.Time     `gorm:"column:left_origin_at" json:"left_origin_at"`
	ArrivedDestAt     *time.Time     `gorm:"column:arrived_dest_at" json:"arrived_dest_at"` // 到达目的地时间
	StatusUpdatedAt   *time.Time     `gorm:"column:status_updated_at" json:"status_updated_at"`
	TrackEndAt        *time.Time     `gorm:"column:track_end_at" json:"track_end_at"`                            // 轨迹截止时间点
	DeviceUnboundAt   *time.Time     `gorm:"column:device_unbound_at" json:"device_unbound_at"`                  // 设备解绑时间
	UnboundDeviceID   *string        `gorm:"type:varchar(50);column:unbound_device_id" json:"unbound_device_id"` // 解绑前的设备ID
	AutoUnbindEnabled bool           `gorm:"default:true;column:auto_unbind_enabled" json:"auto_unbind_enabled"` // 是否启用自动解绑
	OrgID             *string        `gorm:"type:varchar(50);index" json:"org_id"`                               // 所属组织ID
	CreatedAt         time.Time      `gorm:"index:idx_shipments_created_at;index:idx_shipments_status_created" json:"created_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// 发货人/收货人信息
	SenderName    string `gorm:"type:varchar(100);column:sender_name" json:"sender_name"`      // 发货人姓名
	SenderPhone   string `gorm:"type:varchar(50);column:sender_phone" json:"sender_phone"`     // 发货人电话
	ReceiverName  string `gorm:"type:varchar(100);column:receiver_name" json:"receiver_name"`  // 收货人姓名
	ReceiverPhone string `gorm:"type:varchar(50);column:receiver_phone" json:"receiver_phone"` // 收货人电话

	// 关键单证 (Key Documents) - 标准跨境型新增
	BillOfLading string `gorm:"type:varchar(100);column:bill_of_lading" json:"bill_of_lading"` // 提单号 MBL/HBL/AWB
	ContainerNo  string `gorm:"type:varchar(50);column:container_no" json:"container_no"`      // 箱号/车牌号

	// 详细路由 - ETD/ATD/ATA
	ETD *time.Time `gorm:"column:etd" json:"etd"` // 预计出发时间
	ATD *time.Time `gorm:"column:atd" json:"atd"` // 实际出发时间
	ATA *time.Time `gorm:"column:ata" json:"ata"` // 实际到达时间 (Phase 1新增)

	// 船务信息 (Vessel Info) - Phase 1新增
	VesselName string `gorm:"type:varchar(100);column:vessel_name" json:"vessel_name"` // 船名
	VoyageNo   string `gorm:"type:varchar(50);column:voyage_no" json:"voyage_no"`      // 航次
	Carrier    string `gorm:"type:varchar(100);column:carrier" json:"carrier"`         // 船司/航司
	SealNo     string `gorm:"type:varchar(50);column:seal_no" json:"seal_no"`          // 封条号

	// 订单关联 (Order Relation) - Phase 1新增
	PONumbers     string `gorm:"type:varchar(500);column:po_numbers" json:"po_numbers"`           // PO单号(可多个,逗号分隔)
	SKUIDs        string `gorm:"type:varchar(500);column:sku_ids" json:"sku_ids"`                 // SKU ID(可多个)
	FBAShipmentID string `gorm:"type:varchar(100);column:fba_shipment_id" json:"fba_shipment_id"` // FBA发货编号

	// 费用记录 (Cost Records) - Phase 1新增
	FreightCost *float64 `gorm:"type:decimal(12,2);column:freight_cost" json:"freight_cost"` // 运费 USD
	Surcharges  *float64 `gorm:"type:decimal(12,2);column:surcharges" json:"surcharges"`     // 附加费 USD
	CustomsFee  *float64 `gorm:"type:decimal(12,2);column:customs_fee" json:"customs_fee"`   // 关税 USD
	OtherCost   *float64 `gorm:"type:decimal(12,2);column:other_cost" json:"other_cost"`     // 其他费用 USD
	TotalCost   *float64 `gorm:"type:decimal(12,2);column:total_cost" json:"total_cost"`     // 总费用 USD

	// 货物量纲 (Cargo Dimensions)
	Pieces *int     `gorm:"column:pieces" json:"pieces"`                    // 件数
	Weight *float64 `gorm:"type:decimal(10,2);column:weight" json:"weight"` // 重量 kg
	Volume *float64 `gorm:"type:decimal(10,2);column:volume" json:"volume"` // 体积 m³

	// IoT 预警阈值
	MaxTemperature *float64 `gorm:"type:decimal(5,2);column:max_temperature" json:"max_temperature"`
	MinTemperature *float64 `gorm:"type:decimal(5,2);column:min_temperature" json:"min_temperature"`
	MaxHumidity    *float64 `gorm:"type:decimal(5,2);column:max_humidity" json:"max_humidity"`
	MaxShock       *float64 `gorm:"type:decimal(5,2);column:max_shock" json:"max_shock"` // 震动阈值 g
	MaxTilt        *float64 `gorm:"type:decimal(5,2);column:max_tilt" json:"max_tilt"`   // 倾斜阈值 度

	// 业务规则阈值 (Business Rule Thresholds)
	MaxETADelayHours     *int `gorm:"default:24;column:max_eta_delay_hours" json:"max_eta_delay_hours"`         // 允许最大ETA延误小时数
	MaxCustomsHoldHours  *int `gorm:"default:48;column:max_customs_hold_hours" json:"max_customs_hold_hours"`   // 允许最大查验滞留小时数
	FreeTimeWarningHours *int `gorm:"default:24;column:free_time_warning_hours" json:"free_time_warning_hours"` // 免租期预警提前小时数

	// 关务与末端作业状态
	CustomsStatus      string     `gorm:"type:varchar(50);default:'pending'" json:"customs_status"`  // pending, examination, cleared, hold
	CustomsHoldSince   *time.Time `json:"customs_hold_since"`                                        // 查验/扣货开始时间
	FreeTimeExpiration *time.Time `json:"free_time_expiration"`                                      // 免租期结束时间
	AppointmentStatus  string     `gorm:"type:varchar(50);default:'none'" json:"appointment_status"` // none, scheduled, failed, completed

	// 关联
	Device *Device `gorm:"foreignKey:DeviceID;references:ID" json:"device,omitempty"`
}

// BeforeCreate - ID由ShipmentIDGenerator服务生成，这里不再自动生成
func (s *Shipment) BeforeCreate(tx *gorm.DB) error {
	// ID应该在handler中由ShipmentIDGenerator生成
	// 这里仅作为备用，如果ID未设置则报错
	if s.ID == "" {
		return fmt.Errorf("运单ID未设置，请使用ShipmentIDGenerator生成")
	}
	return nil
}

// GetTotalDuration 计算总耗时（从开始运输到签收/当前时间）
func (s *Shipment) GetTotalDuration() string {
	if s.LeftOriginAt == nil {
		return "总耗时：未开始"
	}

	endTime := s.ArrivedDestAt
	if endTime == nil {
		// 运输中 - 计算从开始到当前的耗时
		now := time.Now()
		endTime = &now
	}

	// 防御性检查：如果 arrived_dest_at 等于 left_origin_at（数据异常），使用当前时间
	duration := endTime.Sub(*s.LeftOriginAt)
	if duration <= 0 && s.Status == "delivered" {
		// 已签收但耗时为0或负数，说明数据异常，尝试使用 StatusUpdatedAt 或当前时间
		if s.StatusUpdatedAt != nil && s.StatusUpdatedAt.After(*s.LeftOriginAt) {
			duration = s.StatusUpdatedAt.Sub(*s.LeftOriginAt)
		} else {
			now := time.Now()
			duration = now.Sub(*s.LeftOriginAt)
		}
	}

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("总耗时：%d天%d小时%d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("总耗时：%d小时%d分钟", hours, minutes)
	}
	return fmt.Sprintf("总耗时：%d分钟", minutes)
}

type ShipmentCreateRequest struct {
	OrgID         *string    `json:"org_id"` // 所属组织ID（可选，不传则使用用户主组织）
	DeviceID      *string    `json:"device_id"`
	TransportType string     `json:"transport_type"` // 运输类型: sea/air/land/multimodal
	TransportMode string     `json:"transport_mode"` // 运输模式: lcl/fcl
	ContainerType string     `json:"container_type"` // 柜型: 20GP/40GP/40HQ
	CargoType     string     `json:"cargo_type"`     // 货物类型: general/dangerous/cold_chain
	CargoName     string     `json:"cargo_name"`     // 货物名称
	Origin        string     `json:"origin" binding:"required"`
	Destination   string     `json:"destination" binding:"required"`
	OriginLat     *float64   `json:"origin_lat"`
	OriginLng     *float64   `json:"origin_lng"`
	OriginAddress string     `json:"origin_address"` // 发货详细地址
	DestLat       *float64   `json:"dest_lat"`
	DestLng       *float64   `json:"dest_lng"`
	DestAddress   string     `json:"dest_address"` // 收货详细地址
	DepartureTime *time.Time `json:"departure_time"`
	ETA           *time.Time `json:"eta"`

	// 发货人/收货人信息
	SenderName    string `json:"sender_name"`    // 发货人姓名
	SenderPhone   string `json:"sender_phone"`   // 发货人电话
	ReceiverName  string `json:"receiver_name"`  // 收货人姓名
	ReceiverPhone string `json:"receiver_phone"` // 收货人电话

	// 关键单证 (标准跨境型)
	BillOfLading string `json:"bill_of_lading"` // 提单号 MBL/HBL/AWB
	ContainerNo  string `json:"container_no"`   // 箱号/车牌号
	SealNo       string `json:"seal_no"`        // 封条号 (Phase 1新增)

	// 详细路由 - ETD/ATD
	ETD *time.Time `json:"etd"` // 预计出发时间
	ATD *time.Time `json:"atd"` // 实际出发时间

	// 船务信息 (Phase 1新增)
	VesselName string `json:"vessel_name"` // 船名
	VoyageNo   string `json:"voyage_no"`   // 航次
	Carrier    string `json:"carrier"`     // 船司/航司

	// 订单关联 (Phase 1新增)
	PONumbers     string `json:"po_numbers"`      // PO单号(可多个,逗号分隔)
	SKUIDs        string `json:"sku_ids"`         // SKU ID(可多个)
	FBAShipmentID string `json:"fba_shipment_id"` // FBA发货编号

	// 货物量纲
	Pieces *int     `json:"pieces"` // 件数
	Weight *float64 `json:"weight"` // 重量 kg
	Volume *float64 `json:"volume"` // 体积 m³

	// 费用信息 (Phase 1新增)
	FreightCost *float64 `json:"freight_cost"` // 运费 USD
	Surcharges  *float64 `json:"surcharges"`   // 附加费 USD
	CustomsFee  *float64 `json:"customs_fee"`  // 关税 USD
	OtherCost   *float64 `json:"other_cost"`   // 其他费用 USD

	// 路线规划信息 (用于自动生成Stages)
	OriginPortCode string `json:"origin_port_code"` // 起运港代码
	DestPortCode   string `json:"dest_port_code"`   // 目的港代码
}

type ShipmentUpdateRequest struct {
	DeviceID      *string  `json:"device_id"`
	OrgID         *string  `json:"org_id"`         // 所属组织ID
	TransportType *string  `json:"transport_type"` // 运输类型
	TransportMode *string  `json:"transport_mode"` // 运输模式: lcl/fcl
	ContainerType *string  `json:"container_type"` // 柜型: 20GP/40GP/40HQ
	CargoType     *string  `json:"cargo_type"`     // 货物类型
	CargoName     *string  `json:"cargo_name"`     // 货物名称
	Origin        *string  `json:"origin"`
	Destination   *string  `json:"destination"`
	OriginLat     *float64 `json:"origin_lat"`
	OriginLng     *float64 `json:"origin_lng"`
	OriginAddress *string  `json:"origin_address"`
	DestLat       *float64 `json:"dest_lat"`
	DestLng       *float64 `json:"dest_lng"`
	DestAddress   *string  `json:"dest_address"`
	Status        *string  `json:"status"`
	Progress      *int     `json:"progress"`

	// 发货人/收货人信息
	SenderName    *string `json:"sender_name"`    // 发货人姓名
	SenderPhone   *string `json:"sender_phone"`   // 发货人电话
	ReceiverName  *string `json:"receiver_name"`  // 收货人姓名
	ReceiverPhone *string `json:"receiver_phone"` // 收货人电话

	// 关键单证 (标准跨境型)
	BillOfLading *string `json:"bill_of_lading"` // 提单号 MBL/HBL/AWB
	ContainerNo  *string `json:"container_no"`   // 箱号/车牌号
	SealNo       *string `json:"seal_no"`        // 封条号 (Phase 1新增)

	// 详细路由 - ETD/ATD/ATA
	ETD *time.Time `json:"etd"` // 预计出发时间
	ATD *time.Time `json:"atd"` // 实际出发时间
	ETA *time.Time `json:"eta"` // 预计到达时间
	ATA *time.Time `json:"ata"` // 实际到达时间 (Phase 1新增)

	// 船务信息 (Phase 1新增)
	VesselName *string `json:"vessel_name"` // 船名
	VoyageNo   *string `json:"voyage_no"`   // 航次
	Carrier    *string `json:"carrier"`     // 船司/航司

	// 订单关联 (Phase 1新增)
	PONumbers     *string `json:"po_numbers"`      // PO单号(可多个,逗号分隔)
	SKUIDs        *string `json:"sku_ids"`         // SKU ID(可多个)
	FBAShipmentID *string `json:"fba_shipment_id"` // FBA发货编号

	// 货物量纲
	Pieces *int     `json:"pieces"` // 件数
	Weight *float64 `json:"weight"` // 重量 kg
	Volume *float64 `json:"volume"` // 体积 m³

	// 费用信息 (Phase 1新增)
	FreightCost *float64 `json:"freight_cost"` // 运费 USD
	Surcharges  *float64 `json:"surcharges"`   // 附加费 USD
	CustomsFee  *float64 `json:"customs_fee"`  // 关税 USD
	OtherCost   *float64 `json:"other_cost"`   // 其他费用 USD
	TotalCost   *float64 `json:"total_cost"`   // 总费用 USD
}
