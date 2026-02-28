package models

import (
	"time"

	"gorm.io/gorm"
)

// CollaborationStatus 协作状态
type CollaborationStatus string

const (
	CollabStatusInvited    CollaborationStatus = "invited"     // 已邀请
	CollabStatusAccepted   CollaborationStatus = "accepted"    // 已接受
	CollabStatusInProgress CollaborationStatus = "in_progress" // 进行中
	CollabStatusCompleted  CollaborationStatus = "completed"   // 已完成
	CollabStatusCancelled  CollaborationStatus = "cancelled"   // 已取消
)

// ShipmentCollaboration 运单协作记录
type ShipmentCollaboration struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ShipmentID string `gorm:"type:varchar(50);index;not null" json:"shipment_id"`
	PartnerID  string `gorm:"type:varchar(50);index;not null" json:"partner_id"`

	// 协作信息
	Role   PartnerType         `gorm:"type:varchar(20)" json:"role"` // forwarder/broker/trucker
	Status CollaborationStatus `gorm:"type:varchar(20);default:'invited'" json:"status"`

	// 时间节点
	AssignedAt  time.Time  `json:"assigned_at"`
	AcceptedAt  *time.Time `json:"accepted_at"`
	CompletedAt *time.Time `json:"completed_at"`

	// 任务描述
	TaskDesc string `gorm:"type:varchar(500)" json:"task_desc"`
	Remarks  string `gorm:"type:text" json:"remarks"`

	// 分配人
	AssignedBy     string `gorm:"type:varchar(50)" json:"assigned_by"`
	AssignedByName string `gorm:"-" json:"assigned_by_name,omitempty"` // 非数据库字段

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Shipment *Shipment `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
	Partner  *Partner  `gorm:"foreignKey:PartnerID;references:ID" json:"partner,omitempty"`
}

// TableName 设置表名
func (ShipmentCollaboration) TableName() string {
	return "shipment_collaborations"
}

// CollaborationResponse 响应结构
type CollaborationResponse struct {
	ID             uint                `json:"id"`
	ShipmentID     string              `json:"shipment_id"`
	PartnerID      string              `json:"partner_id"`
	PartnerName    string              `json:"partner_name"`
	PartnerType    PartnerType         `json:"partner_type"`
	Role           PartnerType         `json:"role"`
	Status         CollaborationStatus `json:"status"`
	AssignedAt     time.Time           `json:"assigned_at"`
	AcceptedAt     *time.Time          `json:"accepted_at"`
	CompletedAt    *time.Time          `json:"completed_at"`
	TaskDesc       string              `json:"task_desc"`
	Remarks        string              `json:"remarks"`
	AssignedBy     string              `json:"assigned_by"`
	AssignedByName string              `json:"assigned_by_name"`
}

func (c *ShipmentCollaboration) ToResponse() CollaborationResponse {
	resp := CollaborationResponse{
		ID:             c.ID,
		ShipmentID:     c.ShipmentID,
		PartnerID:      c.PartnerID,
		Role:           c.Role,
		Status:         c.Status,
		AssignedAt:     c.AssignedAt,
		AcceptedAt:     c.AcceptedAt,
		CompletedAt:    c.CompletedAt,
		TaskDesc:       c.TaskDesc,
		Remarks:        c.Remarks,
		AssignedBy:     c.AssignedBy,
		AssignedByName: c.AssignedByName,
	}
	if c.Partner != nil {
		resp.PartnerName = c.Partner.Name
		resp.PartnerType = c.Partner.Type
	}
	return resp
}

// CollaborationCreateRequest 创建请求
type CollaborationCreateRequest struct {
	PartnerID string      `json:"partner_id" binding:"required"`
	Role      PartnerType `json:"role" binding:"required"`
	TaskDesc  string      `json:"task_desc"`
}

// CollaborationUpdateRequest 更新请求
type CollaborationUpdateRequest struct {
	Status  *CollaborationStatus `json:"status"`
	Remarks *string              `json:"remarks"`
}
