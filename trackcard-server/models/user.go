package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID               string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Email            string         `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"`
	Password         string         `gorm:"type:varchar(255);not null" json:"-"`
	PhoneCountryCode string         `gorm:"type:varchar(8);default:'+86';uniqueIndex:idx_user_phone,priority:1" json:"phone_country_code"`
	PhoneNumber      *string        `gorm:"type:varchar(20);uniqueIndex:idx_user_phone,priority:2" json:"phone_number"`
	PhoneVerifiedAt  *time.Time     `json:"phone_verified_at"`
	LastOrgID        *string        `gorm:"type:varchar(50)" json:"last_org_id"`
	Name             string         `gorm:"type:varchar(100);not null" json:"name"`
	Role             string         `gorm:"type:varchar(20);default:'viewer'" json:"role"`
	Permissions      string         `gorm:"type:text;default:'{}'" json:"permissions"`
	Status           string         `gorm:"type:varchar(20);default:'active'" json:"status"`
	WechatOpenID     *string        `gorm:"type:varchar(100);index" json:"wechat_openid"`
	WechatUnionID    *string        `gorm:"type:varchar(100);index" json:"wechat_unionid"`
	Avatar           *string        `gorm:"type:varchar(500)" json:"avatar"`
	LastLogin        *time.Time     `json:"last_login"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = "user-" + uuid.New().String()[:8]
	}
	return nil
}

type UserResponse struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	PhoneCountryCode string     `json:"phone_country_code"`
	PhoneNumber      *string    `json:"phone_number"`
	Name             string     `json:"name"`
	Role             string     `json:"role"`
	Permissions      string     `json:"permissions"`
	Status           string     `json:"status"`
	Avatar           *string    `json:"avatar"`
	LastLogin        *time.Time `json:"last_login"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:               u.ID,
		Email:            u.Email,
		PhoneCountryCode: u.PhoneCountryCode,
		PhoneNumber:      u.PhoneNumber,
		Name:             u.Name,
		Role:             u.Role,
		Permissions:      u.Permissions,
		Status:           u.Status,
		Avatar:           u.Avatar,
		LastLogin:        u.LastLogin,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}
