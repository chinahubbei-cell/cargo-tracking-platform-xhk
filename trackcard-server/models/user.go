package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            string         `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Email         string         `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"`
	Password      string         `gorm:"type:varchar(255);not null" json:"-"`
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`
	Role          string         `gorm:"type:varchar(20);default:'viewer'" json:"role"`
	Permissions   string         `gorm:"type:text;default:'{}'" json:"permissions"`
	Status        string         `gorm:"type:varchar(20);default:'active'" json:"status"`
	WechatOpenID  *string        `gorm:"type:varchar(100);index" json:"wechat_openid"`
	WechatUnionID *string        `gorm:"type:varchar(100);index" json:"wechat_unionid"`
	Avatar        *string        `gorm:"type:varchar(500)" json:"avatar"`
	LastLogin     *time.Time     `json:"last_login"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = "user-" + uuid.New().String()[:8]
	}
	return nil
}

type UserResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Role        string     `json:"role"`
	Permissions string     `json:"permissions"`
	Status      string     `json:"status"`
	Avatar      *string    `json:"avatar"`
	LastLogin   *time.Time `json:"last_login"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Role:        u.Role,
		Permissions: u.Permissions,
		Status:      u.Status,
		Avatar:      u.Avatar,
		LastLogin:   u.LastLogin,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
