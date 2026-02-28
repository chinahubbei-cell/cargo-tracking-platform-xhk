package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Route 线路模型
type Route struct {
	ID             string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name           string         `json:"name" gorm:"type:varchar(200);not null"`          // 线路名称
	Description    string         `json:"description" gorm:"type:text"`                    // 描述
	Type           string         `json:"type" gorm:"type:varchar(50);default:'road'"`     // 类型: ocean/air/rail/road/multimodal
	Status         string         `json:"status" gorm:"type:varchar(20);default:'active'"` // 状态: active/inactive
	Nodes          string         `json:"nodes" gorm:"type:text"`                          // 节点列表 (JSON字符串)
	TotalDistance  float64        `json:"total_distance" gorm:"default:0"`                 // 总距离(公里)
	EstimatedHours float64        `json:"estimated_hours" gorm:"default:0"`                // 预计时间(小时)
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// RouteNode 线路节点结构 (用于JSON序列化)
type RouteNode struct {
	Order     int     `json:"order"`     // 节点顺序
	Name      string  `json:"name"`      // 节点名称
	Type      string  `json:"type"`      // 类型: origin/waypoint/destination
	Latitude  float64 `json:"latitude"`  // 纬度
	Longitude float64 `json:"longitude"` // 经度
	Transport string  `json:"transport"` // 下一段运输方式: ocean/air/rail/road
}

// BeforeCreate 创建前自动生成UUID
func (r *Route) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// CreateRouteRequest 创建线路请求
type CreateRouteRequest struct {
	Name           string      `json:"name" binding:"required"`
	Description    string      `json:"description"`
	Type           string      `json:"type"`
	Nodes          []RouteNode `json:"nodes"`
	TotalDistance  float64     `json:"total_distance"`
	EstimatedHours float64     `json:"estimated_hours"`
}

// UpdateRouteRequest 更新线路请求
type UpdateRouteRequest struct {
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Type           string      `json:"type"`
	Status         string      `json:"status"`
	Nodes          []RouteNode `json:"nodes"`
	TotalDistance  float64     `json:"total_distance"`
	EstimatedHours float64     `json:"estimated_hours"`
}
