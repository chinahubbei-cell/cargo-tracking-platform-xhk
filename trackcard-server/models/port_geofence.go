package models

import "time"

// PortGeofence 港口围栏数据
type PortGeofence struct {
	ID            string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Code          string    `gorm:"type:varchar(20);uniqueIndex" json:"code"`               // 港口代码 CNSHA
	Name          string    `gorm:"type:varchar(100)" json:"name"`                          // 港口英文名
	NameCN        string    `gorm:"type:varchar(100)" json:"name_cn"`                       // 港口中文名
	Country       string    `gorm:"type:varchar(50)" json:"country"`                        // 国家
	CountryCN     string    `gorm:"type:varchar(50)" json:"country_cn"`                     // 国家中文
	GeofenceType  string    `gorm:"type:varchar(20);default:'circle'" json:"geofence_type"` // circle/polygon
	CenterLat     float64   `gorm:"type:decimal(10,6)" json:"center_lat"`                   // 中心纬度
	CenterLng     float64   `gorm:"type:decimal(10,6)" json:"center_lng"`                   // 中心经度
	Radius        int       `gorm:"default:5000" json:"radius"`                             // 圆形围栏半径(米)
	PolygonPoints string    `gorm:"type:text" json:"polygon_points"`                        // 多边形顶点JSON [[lat,lng],...]
	Color         string    `gorm:"type:varchar(20);default:'#1890ff'" json:"color"`        // 围栏颜色
	IsActive      bool      `gorm:"default:true" json:"is_active"`                          // 是否启用
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (PortGeofence) TableName() string {
	return "port_geofences"
}
