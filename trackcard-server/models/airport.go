package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Airport 机场模型
// 用于存储全球机场信息，支持空运线路规划和电子围栏触发
type Airport struct {
	ID                string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	IATACode          string         `json:"iata_code" gorm:"type:varchar(3);uniqueIndex"` // IATA代码 (PVG, LAX)
	ICAOCode          string         `json:"icao_code" gorm:"type:varchar(4)"`             // ICAO代码 (ZSPD)
	Name              string         `json:"name" gorm:"type:varchar(200);not null"`       // 机场名称
	NameEn            string         `json:"name_en" gorm:"type:varchar(200)"`             // 英文名称
	City              string         `json:"city" gorm:"type:varchar(100)"`                // 所在城市
	Country           string         `json:"country" gorm:"type:varchar(10)"`              // 国家代码
	Region            string         `json:"region" gorm:"type:varchar(100)"`              // 区域 (East Asia, Europe等)
	Type              string         `json:"type" gorm:"type:varchar(20);default:'cargo'"` // 类型: cargo/passenger/mixed
	Tier              int            `json:"tier" gorm:"default:2"`                        // 等级: 1=枢纽 2=区域 3=支线
	Latitude          float64        `json:"latitude"`                                     // 纬度
	Longitude         float64        `json:"longitude"`                                    // 经度
	Timezone          string         `json:"timezone" gorm:"type:varchar(50)"`             // 时区
	GeofenceKM        float64        `json:"geofence_km" gorm:"default:10"`                // 围栏半径(KM)
	IsCargoHub        bool           `json:"is_cargo_hub" gorm:"default:false"`            // 是否货运枢纽
	CustomsEfficiency int            `json:"customs_efficiency" gorm:"default:3"`          // 清关效率评分 (1-5)
	CongestionLevel   int            `json:"congestion_level" gorm:"default:1"`            // 拥堵等级 (1-5)
	RunwayCount       int            `json:"runway_count" gorm:"default:1"`                // 跑道数量
	CargoTerminals    int            `json:"cargo_terminals" gorm:"default:1"`             // 货运航站楼数
	AnnualCargoTons   float64        `json:"annual_cargo_tons"`                            // 年货运量(万吨)
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate 创建前自动生成UUID
func (a *Airport) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// AirportGeofence 机场电子围栏
// 用于检测设备进入/离开机场区域
type AirportGeofence struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	AirportCode string    `json:"airport_code" gorm:"type:varchar(4);uniqueIndex"` // IATA代码
	AirportName string    `json:"airport_name" gorm:"type:varchar(200)"`
	City        string    `json:"city" gorm:"type:varchar(100)"`
	Country     string    `json:"country" gorm:"type:varchar(10)"`
	Longitude   float64   `json:"longitude"`
	Latitude    float64   `json:"latitude"`
	RadiusKM    float64   `json:"radius_km" gorm:"default:10"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate 创建前自动生成UUID
func (ag *AirportGeofence) BeforeCreate(tx *gorm.DB) error {
	if ag.ID == "" {
		ag.ID = uuid.New().String()
	}
	return nil
}
