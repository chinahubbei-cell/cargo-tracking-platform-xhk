package models

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 任务状态和类型 ====================

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"     // 待处理
	TaskStatusAssigned   TaskStatus = "assigned"    // 已分配
	TaskStatusInProgress TaskStatus = "in_progress" // 进行中
	TaskStatusCompleted  TaskStatus = "completed"   // 已完成
	TaskStatusCancelled  TaskStatus = "cancelled"   // 已取消
	TaskStatusExpired    TaskStatus = "expired"     // 已过期
	TaskStatusFailed     TaskStatus = "failed"      // 失败
)

// TaskPriority 任务优先级
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityNormal TaskPriority = "normal"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityUrgent TaskPriority = "urgent"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeConfirmPickup   TaskType = "confirm_pickup"   // 确认提货
	TaskTypeConfirmDelivery TaskType = "confirm_delivery" // 确认送达
	TaskTypeUploadDocument  TaskType = "upload_document"  // 上传单据
	TaskTypeConfirmCleared  TaskType = "confirm_cleared"  // 确认清关
	TaskTypeReportLocation  TaskType = "report_location"  // 上报位置
	TaskTypeApproveDocument TaskType = "approve_document" // 审批单据
	TaskTypeHandleException TaskType = "handle_exception" // 处理异常
	TaskTypeCustom          TaskType = "custom"           // 自定义任务
)

// ==================== 任务模型 ====================

// Task 任务
type Task struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	TaskNo string `gorm:"type:varchar(50);uniqueIndex" json:"task_no"` // 任务编号

	// 关联信息
	ShipmentID  *string `gorm:"type:varchar(50);index" json:"shipment_id"`
	MilestoneID *uint   `gorm:"index" json:"milestone_id"` // 关联节点

	// 任务内容
	TaskType    TaskType     `gorm:"type:varchar(50);index;not null" json:"task_type"`
	Title       string       `gorm:"type:varchar(200);not null" json:"title"`
	Description string       `gorm:"type:text" json:"description"`
	Priority    TaskPriority `gorm:"type:varchar(20);default:'normal'" json:"priority"`

	// 分配信息
	AssigneeType string     `gorm:"type:varchar(50)" json:"assignee_type"` // user/partner/role
	AssigneeID   *string    `gorm:"type:varchar(50);index" json:"assignee_id"`
	AssigneeName string     `gorm:"type:varchar(100)" json:"assignee_name"`
	AssignedBy   *string    `gorm:"type:varchar(50)" json:"assigned_by"`
	AssignedAt   *time.Time `json:"assigned_at"`

	// 状态
	Status TaskStatus `gorm:"type:varchar(20);default:'pending';index" json:"status"`

	// 时间
	DueAt       *time.Time `json:"due_at"`       // 截止时间
	StartedAt   *time.Time `json:"started_at"`   // 开始处理时间
	CompletedAt *time.Time `json:"completed_at"` // 完成时间

	// Magic Link
	MagicLinkID *uint `gorm:"index" json:"magic_link_id"`

	// 结果
	Result     string `gorm:"type:text" json:"result,omitempty"` // 处理结果JSON
	FailReason string `gorm:"type:varchar(500)" json:"fail_reason,omitempty"`

	// 提醒
	ReminderSent   bool       `gorm:"default:false" json:"reminder_sent"`
	ReminderSentAt *time.Time `json:"reminder_sent_at,omitempty"`

	// 元数据
	Metadata string `gorm:"type:text" json:"metadata,omitempty"` // 额外数据JSON

	CreatedBy string         `gorm:"type:varchar(50)" json:"created_by"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Shipment  *Shipment          `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
	Milestone *ShipmentMilestone `gorm:"foreignKey:MilestoneID" json:"milestone,omitempty"`
	MagicLink *MagicLink         `gorm:"foreignKey:MagicLinkID" json:"magic_link,omitempty"`
}

func (Task) TableName() string {
	return "tasks"
}

// ==================== 任务规则模型 ====================

// TaskDispatchRule 任务分发规则
type TaskDispatchRule struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:varchar(500)" json:"description"`

	// 触发条件
	TriggerEvent     string `gorm:"type:varchar(50);index" json:"trigger_event"` // milestone_started, milestone_completed, etc.
	TriggerMilestone string `gorm:"type:varchar(50)" json:"trigger_milestone"`   // 节点代码
	TriggerCondition string `gorm:"type:text" json:"trigger_condition"`          // JSON条件表达式

	// 生成任务
	TaskType        TaskType     `gorm:"type:varchar(50)" json:"task_type"`
	TaskTitle       string       `gorm:"type:varchar(200)" json:"task_title"`
	TaskDescription string       `gorm:"type:text" json:"task_description"`
	TaskPriority    TaskPriority `gorm:"type:varchar(20);default:'normal'" json:"task_priority"`

	// 分配目标
	AssignToType   string `gorm:"type:varchar(50)" json:"assign_to_type"` // shipment_partner, role, specific_user
	AssignToRole   string `gorm:"type:varchar(50)" json:"assign_to_role"` // trucker, customs_broker等
	AssignToUserID string `gorm:"type:varchar(50)" json:"assign_to_user_id"`

	// 时限
	DueHours         int  `json:"due_hours"`                              // 截止时间(小时)
	AutoGenerateLink bool `gorm:"default:true" json:"auto_generate_link"` // 自动生成Magic Link

	// 状态
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TaskDispatchRule) TableName() string {
	return "task_dispatch_rules"
}

// ==================== API请求/响应结构 ====================

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	ShipmentID   *string      `json:"shipment_id"`
	MilestoneID  *uint        `json:"milestone_id"`
	TaskType     TaskType     `json:"task_type" binding:"required"`
	Title        string       `json:"title" binding:"required"`
	Description  string       `json:"description"`
	Priority     TaskPriority `json:"priority"`
	AssigneeType string       `json:"assignee_type"` // user/partner
	AssigneeID   string       `json:"assignee_id"`
	DueHours     int          `json:"due_hours"`
	GenerateLink bool         `json:"generate_link"` // 是否生成Magic Link
}

// TaskResponse 任务响应
type TaskResponse struct {
	ID           uint         `json:"id"`
	TaskNo       string       `json:"task_no"`
	ShipmentID   *string      `json:"shipment_id"`
	TaskType     TaskType     `json:"task_type"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Priority     TaskPriority `json:"priority"`
	AssigneeName string       `json:"assignee_name"`
	Status       TaskStatus   `json:"status"`
	DueAt        *time.Time   `json:"due_at"`
	MagicLinkURL string       `json:"magic_link_url,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

func (t *Task) ToResponse() TaskResponse {
	resp := TaskResponse{
		ID:           t.ID,
		TaskNo:       t.TaskNo,
		ShipmentID:   t.ShipmentID,
		TaskType:     t.TaskType,
		Title:        t.Title,
		Description:  t.Description,
		Priority:     t.Priority,
		AssigneeName: t.AssigneeName,
		Status:       t.Status,
		DueAt:        t.DueAt,
		CreatedAt:    t.CreatedAt,
	}
	if t.MagicLink != nil {
		resp.MagicLinkURL = "/m/" + t.MagicLink.Token
	}
	return resp
}

// TaskListQuery 任务列表查询
type TaskListQuery struct {
	ShipmentID string `form:"shipment_id"`
	Status     string `form:"status"`
	TaskType   string `form:"task_type"`
	AssigneeID string `form:"assignee_id"`
	Priority   string `form:"priority"`
	Overdue    *bool  `form:"overdue"` // 是否已逾期
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// UpdateTaskStatusRequest 更新任务状态请求
type UpdateTaskStatusRequest struct {
	Status     TaskStatus `json:"status" binding:"required"`
	Result     string     `json:"result,omitempty"`
	FailReason string     `json:"fail_reason,omitempty"`
}

// CreateDispatchRuleRequest 创建分发规则请求
type CreateDispatchRuleRequest struct {
	Name             string       `json:"name" binding:"required"`
	Description      string       `json:"description"`
	TriggerEvent     string       `json:"trigger_event" binding:"required"`
	TriggerMilestone string       `json:"trigger_milestone"`
	TaskType         TaskType     `json:"task_type" binding:"required"`
	TaskTitle        string       `json:"task_title" binding:"required"`
	TaskDescription  string       `json:"task_description"`
	TaskPriority     TaskPriority `json:"task_priority"`
	AssignToType     string       `json:"assign_to_type" binding:"required"`
	AssignToRole     string       `json:"assign_to_role"`
	DueHours         int          `json:"due_hours"`
	AutoGenerateLink bool         `json:"auto_generate_link"`
}
