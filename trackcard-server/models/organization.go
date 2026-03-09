package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationType 组织类型
type OrganizationType string

const (
	OrgTypeGroup   OrganizationType = "group"   // 集团
	OrgTypeCompany OrganizationType = "company" // 公司
	OrgTypeBranch  OrganizationType = "branch"  // 分公司/分支
	OrgTypeDept    OrganizationType = "dept"    // 部门
	OrgTypeTeam    OrganizationType = "team"    // 团队/小组
)

// Organization 组织机构模型
type Organization struct {
	ID          string           `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name        string           `gorm:"type:varchar(100);not null" json:"name"`
	Code        string           `gorm:"uniqueIndex;type:varchar(50);not null" json:"code"`
	ParentID    *string          `gorm:"type:varchar(50);index" json:"parent_id"`
	Type        OrganizationType `gorm:"type:varchar(20);default:'dept'" json:"type"`
	Level       int              `gorm:"type:int;default:1" json:"level"` // 层级: 1-3
	Path        string           `gorm:"type:varchar(500)" json:"path"`   // 路径，如：org1/org2/org3
	Sort        int              `gorm:"type:int;default:0" json:"sort"`  // 排序号
	Status      string           `gorm:"type:varchar(20);default:'active'" json:"status"`
	LeaderID    *string          `gorm:"type:varchar(50)" json:"leader_id"` // 负责人ID
	Description *string          `gorm:"type:varchar(500)" json:"description"`

	ServiceStatus string     `gorm:"type:varchar(20);default:'active'" json:"service_status"`
	ServiceStart  *time.Time `json:"service_start"`
	ServiceEnd    *time.Time `json:"service_end"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Parent   *Organization  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Organization `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Leader   *User          `gorm:"foreignKey:LeaderID" json:"leader,omitempty"`
}

func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = "org-" + uuid.New().String()[:8]
	}
	return nil
}

// OrganizationResponse 组织机构响应结构
type OrganizationResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Code        string                 `json:"code"`
	ParentID    *string                `json:"parent_id"`
	Type        OrganizationType       `json:"type"`
	Level       int                    `json:"level"`
	Path        string                 `json:"path"`
	Sort        int                    `json:"sort"`
	Status      string                 `json:"status"`
	LeaderID    *string                `json:"leader_id"`
	LeaderName    string                 `json:"leader_name,omitempty"`
	Description   *string                `json:"description"`
	ServiceStatus string                 `json:"service_status"`
	ServiceStart  *time.Time             `json:"service_start"`
	ServiceEnd    *time.Time             `json:"service_end"`
	UserCount     int                    `json:"user_count"`   // 用户数量
	DeviceCount   int                    `json:"device_count"` // 设备数量
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Children    []OrganizationResponse `json:"children,omitempty"`
}

func (o *Organization) ToResponse() OrganizationResponse {
	resp := OrganizationResponse{
		ID:          o.ID,
		Name:        o.Name,
		Code:        o.Code,
		ParentID:    o.ParentID,
		Type:        o.Type,
		Level:       o.Level,
		Path:        o.Path,
		Sort:        o.Sort,
		Status:        o.Status,
		LeaderID:      o.LeaderID,
		Description:   o.Description,
		ServiceStatus: o.ServiceStatus,
		ServiceStart:  o.ServiceStart,
		ServiceEnd:    o.ServiceEnd,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}
	if o.Leader != nil {
		resp.LeaderName = o.Leader.Name
	}
	return resp
}

// UserOrganization 用户-组织关联表（多对多，支持主部门和兼职）
type UserOrganization struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         string    `gorm:"type:varchar(50);not null;index" json:"user_id"`
	OrganizationID string    `gorm:"type:varchar(50);not null;index" json:"organization_id"`
	IsPrimary      bool      `gorm:"default:false" json:"is_primary"`   // 是否主部门
	Position       string    `gorm:"type:varchar(100)" json:"position"` // 职位
	JoinedAt       time.Time `json:"joined_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 关联
	User         *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

// TableName 设置表名
func (UserOrganization) TableName() string {
	return "user_organizations"
}

// UserOrganizationResponse 用户组织关联响应
type UserOrganizationResponse struct {
	ID               uint      `json:"id"`
	UserID           string    `json:"user_id"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	OrganizationCode string    `json:"organization_code"`
	IsPrimary        bool      `json:"is_primary"`
	Position         string    `json:"position"`
	JoinedAt         time.Time `json:"joined_at"`
}

func (uo *UserOrganization) ToResponse() UserOrganizationResponse {
	resp := UserOrganizationResponse{
		ID:             uo.ID,
		UserID:         uo.UserID,
		OrganizationID: uo.OrganizationID,
		IsPrimary:      uo.IsPrimary,
		Position:       uo.Position,
		JoinedAt:       uo.JoinedAt,
	}
	if uo.Organization != nil {
		resp.OrganizationName = uo.Organization.Name
		resp.OrganizationCode = uo.Organization.Code
	}
	return resp
}

// OrganizationTreeNode 树形结构节点
type OrganizationTreeNode struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Code        string                  `json:"code"`
	ParentID    *string                 `json:"parent_id"`
	Type        OrganizationType        `json:"type"`
	Level       int                     `json:"level"`
	Sort          int                     `json:"sort"`
	Status        string                  `json:"status"`
	ServiceStatus string                  `json:"service_status"`
	ServiceStart  *time.Time              `json:"service_start"`
	ServiceEnd    *time.Time              `json:"service_end"`
	LeaderName    string                  `json:"leader_name,omitempty"`
	UserCount     int                     `json:"user_count"`
	DeviceCount   int                     `json:"device_count"`
	Children    []*OrganizationTreeNode `json:"children,omitempty"`
}
