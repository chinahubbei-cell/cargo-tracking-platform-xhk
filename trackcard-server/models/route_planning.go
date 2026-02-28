package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Port 港口/口岸模型
type Port struct {
	ID                string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Code              string         `json:"code" gorm:"type:varchar(20);uniqueIndex"`       // UN/LOCODE 代码 (如 CNSZX)
	Name              string         `json:"name" gorm:"type:varchar(200);not null"`         // 港口名称
	NameEn            string         `json:"name_en" gorm:"type:varchar(200)"`               // 英文名称
	Country           string         `json:"country" gorm:"type:varchar(10)"`                // 国家代码
	Region            string         `json:"region" gorm:"type:varchar(100)"`                // 区域 (East Asia, Europe等)
	Type              string         `json:"type" gorm:"type:varchar(20);default:'seaport'"` // 类型: seaport/airport/rail/inland
	Tier              int            `json:"tier" gorm:"default:2"`                          // 港口等级: 1=枢纽港 2=干线港 3=支线港
	Latitude          float64        `json:"latitude"`                                       // 纬度
	Longitude         float64        `json:"longitude"`                                      // 经度
	Timezone          string         `json:"timezone" gorm:"type:varchar(50)"`               // 时区 (如 Asia/Shanghai)
	GeofenceKM        float64        `json:"geofence_km" gorm:"default:15"`                  // 电子围栏半径(KM)
	IsTransitHub      bool           `json:"is_transit_hub" gorm:"default:false"`            // 是否中转枢纽
	CustomsEfficiency int            `json:"customs_efficiency" gorm:"default:3"`            // 清关效率评分 (1-5)
	CongestionLevel   int            `json:"congestion_level" gorm:"default:1"`              // 拥堵等级 (1-5)
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate 创建前自动生成UUID
func (p *Port) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// ShippingLine 航线/服务模型
type ShippingLine struct {
	ID            string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Carrier       string         `json:"carrier" gorm:"type:varchar(100)"`                   // 承运商 (COSCO, MAERSK等)
	TransportMode string         `json:"transport_mode" gorm:"type:varchar(20)"`             // 运输方式: ocean/air/rail
	POLPortID     string         `json:"pol_port_id" gorm:"type:varchar(36)"`                // 起运港ID
	PODPortID     string         `json:"pod_port_id" gorm:"type:varchar(36)"`                // 目的港ID
	TransitPorts  string         `json:"transit_ports" gorm:"type:text"`                     // 中转港列表 (JSON)
	TransitDays   int            `json:"transit_days"`                                       // 航程天数
	Frequency     string         `json:"frequency" gorm:"type:varchar(50);default:'weekly'"` // 班期
	BaseCost      float64        `json:"base_cost"`                                          // 基础运价 (USD/TEU)
	Active        bool           `json:"active" gorm:"default:true"`                         // 是否激活
	POLPort       *Port          `json:"pol_port,omitempty" gorm:"foreignKey:POLPortID"`
	PODPort       *Port          `json:"pod_port,omitempty" gorm:"foreignKey:PODPortID"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate 创建前自动生成UUID
func (s *ShippingLine) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// LogisticsZone 物流区域围栏模型
type LogisticsZone struct {
	ID             string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string         `json:"name" gorm:"type:varchar(200);not null"` // 区域名称
	Country        string         `json:"country" gorm:"type:varchar(10)"`        // 国家代码
	Polygon        string         `json:"polygon" gorm:"type:text"`               // GeoJSON 围栏坐标
	CenterLat      float64        `json:"center_lat"`                             // 中心点纬度
	CenterLng      float64        `json:"center_lng"`                             // 中心点经度
	DefaultPortIDs string         `json:"default_port_ids" gorm:"type:text"`      // 默认推荐港口ID列表 (JSON)
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate 创建前自动生成UUID
func (z *LogisticsZone) BeforeCreate(tx *gorm.DB) error {
	if z.ID == "" {
		z.ID = uuid.New().String()
	}
	return nil
}

// CostTimeModel 成本时效模型
type CostTimeModel struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SegmentType string         `json:"segment_type" gorm:"type:varchar(50)"` // 环节: first_mile/line_haul/last_mile
	FromZoneID  string         `json:"from_zone_id" gorm:"type:varchar(36)"` // 起始区域ID
	ToPortID    string         `json:"to_port_id" gorm:"type:varchar(36)"`   // 目标港口ID
	AvgCost     float64        `json:"avg_cost"`                             // 平均成本
	AvgDays     float64        `json:"avg_days"`                             // 平均天数
	DistanceKM  float64        `json:"distance_km"`                          // 距离(公里)
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate 创建前自动生成UUID
func (c *CostTimeModel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// RoutePlan 路径规划结果模型
type RoutePlan struct {
	ID              string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ShipmentID      *string        `json:"shipment_id" gorm:"type:varchar(36)"`      // 关联运单
	OriginAddress   string         `json:"origin_address" gorm:"type:varchar(500)"`  // 发货地址
	DestAddress     string         `json:"dest_address" gorm:"type:varchar(500)"`    // 收货地址
	CargoType       string         `json:"cargo_type" gorm:"type:varchar(50)"`       // 货物类型
	RecommendedType string         `json:"recommended_type" gorm:"type:varchar(20)"` // 推荐类型: fastest/cheapest/safest
	PlanData        string         `json:"plan_data" gorm:"type:text"`               // 完整规划数据 (JSON)
	TotalDays       int            `json:"total_days"`                               // 总天数
	TotalCost       float64        `json:"total_cost"`                               // 总成本
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate 创建前自动生成UUID
func (r *RoutePlan) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// RouteSegment 路径段 (用于JSON)
type RouteSegment struct {
	Type         string   `json:"type"`          // first_mile/line_haul/last_mile
	From         string   `json:"from"`          // 起点
	To           string   `json:"to"`            // 终点
	FromLat      float64  `json:"from_lat"`      // 起点纬度
	FromLng      float64  `json:"from_lng"`      // 起点经度
	ToLat        float64  `json:"to_lat"`        // 终点纬度
	ToLng        float64  `json:"to_lng"`        // 终点经度
	Mode         string   `json:"mode"`          // 运输方式
	Carrier      string   `json:"carrier"`       // 承运商
	Days         int      `json:"days"`          // 天数
	Cost         float64  `json:"cost"`          // 成本
	DistanceKM   float64  `json:"distance_km"`   // 距离
	TransitPorts []string `json:"transit_ports"` // 中转港
}

// RouteRecommendation 路径推荐结果
type RouteRecommendation struct {
	Type           string         `json:"type"`            // fastest/cheapest/safest
	Label          string         `json:"label"`           // 显示标签
	TotalDays      int            `json:"total_days"`      // 总天数
	TotalCost      float64        `json:"total_cost"`      // 总成本
	DeviceCoverage string         `json:"device_coverage"` // 设备覆盖率
	Segments       []RouteSegment `json:"segments"`        // 路径段
}
