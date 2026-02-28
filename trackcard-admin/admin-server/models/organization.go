package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminUser 管理员用户（独立于客户账号体系）
type AdminUser struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"uniqueIndex;size:50" json:"username"`
	Password  string     `gorm:"size:255" json:"-"`
	Name      string     `gorm:"size:100" json:"name"`
	Email     string     `gorm:"size:100" json:"email"`
	Phone     string     `gorm:"size:20" json:"phone"`
	Role      string     `gorm:"size:20;default:admin" json:"role"`    // super_admin, admin, operator
	Status    string     `gorm:"size:20;default:active" json:"status"` // active, disabled
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (u *AdminUser) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// SetPassword 设置密码（加密存储）
func (u *AdminUser) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword 验证密码
func (u *AdminUser) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// Organization 组织/租户
type Organization struct {
	ID           string `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"size:200;not null" json:"name"`
	ShortName    string `gorm:"size:50" json:"short_name"`
	ContactName  string `gorm:"size:100" json:"contact_name"`
	ContactPhone string `gorm:"size:20" json:"contact_phone"`
	ContactEmail string `gorm:"size:100" json:"contact_email"`
	Address      string `gorm:"size:500" json:"address"`

	// 服务配置
	ServiceStatus string     `gorm:"size:20;default:trial" json:"service_status"` // trial, active, suspended, expired
	ServiceStart  *time.Time `json:"service_start"`
	ServiceEnd    *time.Time `json:"service_end"`
	AutoRenew     bool       `gorm:"default:false" json:"auto_renew"`

	// 配额
	MaxDevices   int `gorm:"default:10" json:"max_devices"`
	MaxUsers     int `gorm:"default:5" json:"max_users"`
	MaxShipments int `gorm:"default:100" json:"max_shipments"` // 每月

	// 统计
	DeviceCount   int `gorm:"-" json:"device_count"`
	UserCount     int `gorm:"-" json:"user_count"`
	ShipmentCount int `gorm:"-" json:"shipment_count"`

	Remark    string    `gorm:"size:500" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

// IsExpired 检查服务是否已过期
func (o *Organization) IsExpired() bool {
	if o.ServiceEnd == nil {
		return false
	}
	return time.Now().After(*o.ServiceEnd)
}

// DaysUntilExpiry 距离过期天数
func (o *Organization) DaysUntilExpiry() int {
	if o.ServiceEnd == nil {
		return 9999
	}
	days := int(time.Until(*o.ServiceEnd).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
