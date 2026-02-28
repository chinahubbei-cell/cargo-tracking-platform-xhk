package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerType 客户类型
type CustomerType string

const (
	CustomerTypeSender   CustomerType = "sender"   // 发货人
	CustomerTypeReceiver CustomerType = "receiver" // 收货人
)

// Customer 客户信息
type Customer struct {
	ID        string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	OrgID     string         `gorm:"type:varchar(50);not null;index" json:"org_id"` // 所属组织ID
	Type      CustomerType   `gorm:"type:varchar(20);not null;index" json:"type"`   // 类型：sender/receiver
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`        // 姓名
	Phone     string         `gorm:"type:varchar(50);not null;index" json:"phone"`  // 手机号
	Company   string         `gorm:"type:varchar(200)" json:"company"`              // 公司名称
	Address   string         `gorm:"type:varchar(500)" json:"address"`              // 详细地址
	City      string         `gorm:"type:varchar(100)" json:"city"`                 // 城市
	Country   string         `gorm:"type:varchar(100)" json:"country"`              // 国家
	Latitude  float64        `gorm:"type:decimal(10,7)" json:"latitude"`            // 纬度
	Longitude float64        `gorm:"type:decimal(10,7)" json:"longitude"`           // 经度
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate 创建前生成UUID
func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// TableName 表名
func (Customer) TableName() string {
	return "customers"
}

// CustomerCreateRequest 创建客户请求
type CustomerCreateRequest struct {
	OrgID     string       `json:"org_id"`                   // 所属组织ID（可选，不传则使用用户主组织）
	Type      CustomerType `json:"type" binding:"required"`  // 类型
	Name      string       `json:"name" binding:"required"`  // 姓名
	Phone     string       `json:"phone" binding:"required"` // 手机号
	Company   string       `json:"company"`                  // 公司名称
	Address   string       `json:"address"`                  // 详细地址
	City      string       `json:"city"`                     // 城市
	Country   string       `json:"country"`                  // 国家
	Latitude  float64      `json:"latitude"`                 // 纬度
	Longitude float64      `json:"longitude"`                // 经度
}

// CustomerUpdateRequest 更新客户请求
type CustomerUpdateRequest struct {
	Name      *string  `json:"name"`      // 姓名
	Phone     *string  `json:"phone"`     // 手机号
	Company   *string  `json:"company"`   // 公司名称
	Address   *string  `json:"address"`   // 详细地址
	City      *string  `json:"city"`      // 城市
	Country   *string  `json:"country"`   // 国家
	Latitude  *float64 `json:"latitude"`  // 纬度
	Longitude *float64 `json:"longitude"` // 经度
}

// CustomerSearchRequest 搜索客户请求
type CustomerSearchRequest struct {
	Phone string       `form:"phone" binding:"required"` // 手机号
	Type  CustomerType `form:"type"`                     // 类型（可选）
}
