package services

import (
	"encoding/json"
	"fmt"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// TaskDispatcher 任务分发器
type TaskDispatcher struct {
	db *gorm.DB
}

var taskDispatcher *TaskDispatcher

// InitTaskDispatcher 初始化任务分发器
func InitTaskDispatcher(db *gorm.DB) {
	taskDispatcher = &TaskDispatcher{db: db}
}

// GetTaskDispatcher 获取任务分发器实例
func GetTaskDispatcher() *TaskDispatcher {
	return taskDispatcher
}

// ==================== 任务管理 ====================

// CreateTask 创建任务
func (d *TaskDispatcher) CreateTask(req *models.CreateTaskRequest, createdBy string) (*models.Task, error) {
	// 生成任务编号
	taskNo := d.generateTaskNo()

	task := &models.Task{
		TaskNo:       taskNo,
		ShipmentID:   req.ShipmentID,
		MilestoneID:  req.MilestoneID,
		TaskType:     req.TaskType,
		Title:        req.Title,
		Description:  req.Description,
		Priority:     req.Priority,
		AssigneeType: req.AssigneeType,
		Status:       models.TaskStatusPending,
		CreatedBy:    createdBy,
	}

	// 设置优先级默认值
	if task.Priority == "" {
		task.Priority = models.TaskPriorityNormal
	}

	// 设置截止时间
	if req.DueHours > 0 {
		dueAt := time.Now().Add(time.Duration(req.DueHours) * time.Hour)
		task.DueAt = &dueAt
	}

	// 分配任务
	if req.AssigneeID != "" {
		task.AssigneeID = &req.AssigneeID
		task.AssigneeName = d.getAssigneeName(req.AssigneeType, req.AssigneeID)
		now := time.Now()
		task.AssignedAt = &now
		task.Status = models.TaskStatusAssigned
	}

	// 保存任务
	if err := d.db.Create(task).Error; err != nil {
		return nil, err
	}

	// 是否生成Magic Link
	if req.GenerateLink && req.AssigneeID != "" {
		if err := d.generateMagicLink(task); err != nil {
			// 不影响任务创建，只记录日志
			fmt.Printf("Warning: Failed to generate magic link for task %s: %v\n", task.TaskNo, err)
		}
	}

	return task, nil
}

// generateTaskNo 生成任务编号
func (d *TaskDispatcher) generateTaskNo() string {
	return fmt.Sprintf("TSK%s%04d", time.Now().Format("20060102150405"), time.Now().Nanosecond()%10000)
}

// getAssigneeName 获取被分配人名称
func (d *TaskDispatcher) getAssigneeName(assigneeType, assigneeID string) string {
	switch assigneeType {
	case "user":
		var user models.User
		if err := d.db.Select("name").Where("id = ?", assigneeID).First(&user).Error; err == nil {
			return user.Name
		}
	case "partner":
		var partner models.Partner
		if err := d.db.Select("name").Where("id = ?", assigneeID).First(&partner).Error; err == nil {
			return partner.Name
		}
	}
	return ""
}

// generateMagicLink 为任务生成Magic Link
func (d *TaskDispatcher) generateMagicLink(task *models.Task) error {
	magicLinkService := GetMagicLinkService()
	if magicLinkService == nil {
		return fmt.Errorf("magic link service not initialized")
	}

	// 映射任务类型到Magic Link动作类型
	actionType := d.mapTaskTypeToAction(task.TaskType)
	if actionType == "" {
		return nil // 不需要生成链接的任务类型
	}

	// 获取被分配人信息
	var targetPhone, targetEmail string
	if task.AssigneeType == "partner" && task.AssigneeID != nil {
		var partner models.Partner
		if err := d.db.Select("phone, email").
			Where("id = ?", *task.AssigneeID).First(&partner).Error; err == nil {
			targetPhone = partner.Phone
			targetEmail = partner.Email
		}
	}

	// 转换ID类型 (uint -> string)
	taskIDStr := fmt.Sprintf("%d", task.ID)
	var milestoneIDStr *string
	if task.MilestoneID != nil {
		s := fmt.Sprintf("%d", *task.MilestoneID)
		milestoneIDStr = &s
	}

	req := &models.CreateMagicLinkRequest{
		ShipmentID:  *task.ShipmentID,
		TaskID:      &taskIDStr,
		MilestoneID: milestoneIDStr,
		TargetRole:  task.AssigneeType,
		TargetName:  task.AssigneeName,
		TargetPhone: targetPhone,
		TargetEmail: targetEmail,
		ActionType:  actionType,
		ExpiresIn:   24, // 默认24小时
	}

	// 根据截止时间调整有效期
	if task.DueAt != nil {
		hoursUntilDue := int(time.Until(*task.DueAt).Hours())
		if hoursUntilDue > 0 && hoursUntilDue < 24 {
			req.ExpiresIn = hoursUntilDue + 2 // 额外2小时缓冲
		}
	}

	resp, err := magicLinkService.CreateLink(req)
	if err != nil {
		return err
	}

	// 更新任务关联的Magic Link
	// MagicLink创建成功后，可以通过token查询
	_ = resp // Magic link created successfully, link available via /m/:token
	return nil
}

// mapTaskTypeToAction 任务类型映射到Magic Link动作
func (d *TaskDispatcher) mapTaskTypeToAction(taskType models.TaskType) string {
	mapping := map[models.TaskType]string{
		models.TaskTypeConfirmPickup:   models.ActionConfirmPickup,
		models.TaskTypeConfirmDelivery: models.ActionConfirmDelivery,
		models.TaskTypeUploadDocument:  models.ActionUploadDocument,
		models.TaskTypeConfirmCleared:  models.ActionConfirmCleared,
		models.TaskTypeReportLocation:  models.ActionReportLocation,
	}
	return mapping[taskType]
}

// ==================== 任务查询 ====================

// GetTask 获取任务详情
func (d *TaskDispatcher) GetTask(taskID uint) (*models.Task, error) {
	var task models.Task
	if err := d.db.Preload("MagicLink").First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks 查询任务列表
func (d *TaskDispatcher) ListTasks(query *models.TaskListQuery) ([]models.Task, int64, error) {
	tx := d.db.Model(&models.Task{})

	if query.ShipmentID != "" {
		tx = tx.Where("shipment_id = ?", query.ShipmentID)
	}
	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}
	if query.TaskType != "" {
		tx = tx.Where("task_type = ?", query.TaskType)
	}
	if query.AssigneeID != "" {
		tx = tx.Where("assignee_id = ?", query.AssigneeID)
	}
	if query.Priority != "" {
		tx = tx.Where("priority = ?", query.Priority)
	}
	if query.Overdue != nil && *query.Overdue {
		tx = tx.Where("due_at < ? AND status NOT IN ?", time.Now(),
			[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusCancelled})
	}

	var total int64
	tx.Count(&total)

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	var tasks []models.Task
	offset := (query.Page - 1) * query.PageSize
	if err := tx.Preload("MagicLink").
		Order("created_at DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// ==================== 任务状态管理 ====================

// UpdateTaskStatus 更新任务状态
func (d *TaskDispatcher) UpdateTaskStatus(taskID uint, req *models.UpdateTaskStatusRequest) error {
	var task models.Task
	if err := d.db.First(&task, taskID).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"status": req.Status,
	}

	switch req.Status {
	case models.TaskStatusInProgress:
		now := time.Now()
		updates["started_at"] = now
	case models.TaskStatusCompleted:
		now := time.Now()
		updates["completed_at"] = now
		if req.Result != "" {
			updates["result"] = req.Result
		}
	case models.TaskStatusFailed:
		updates["fail_reason"] = req.FailReason
	}

	return d.db.Model(&task).Updates(updates).Error
}

// CompleteTask 完成任务
func (d *TaskDispatcher) CompleteTask(taskID uint, result interface{}) error {
	resultJSON := ""
	if result != nil {
		bytes, _ := json.Marshal(result)
		resultJSON = string(bytes)
	}

	return d.UpdateTaskStatus(taskID, &models.UpdateTaskStatusRequest{
		Status: models.TaskStatusCompleted,
		Result: resultJSON,
	})
}

// ==================== 规则管理 ====================

// CreateDispatchRule 创建分发规则
func (d *TaskDispatcher) CreateDispatchRule(req *models.CreateDispatchRuleRequest) (*models.TaskDispatchRule, error) {
	rule := &models.TaskDispatchRule{
		Name:             req.Name,
		Description:      req.Description,
		TriggerEvent:     req.TriggerEvent,
		TriggerMilestone: req.TriggerMilestone,
		TaskType:         req.TaskType,
		TaskTitle:        req.TaskTitle,
		TaskDescription:  req.TaskDescription,
		TaskPriority:     req.TaskPriority,
		AssignToType:     req.AssignToType,
		AssignToRole:     req.AssignToRole,
		DueHours:         req.DueHours,
		AutoGenerateLink: req.AutoGenerateLink,
		IsActive:         true,
	}

	if rule.TaskPriority == "" {
		rule.TaskPriority = models.TaskPriorityNormal
	}

	if err := d.db.Create(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

// ListDispatchRules 获取分发规则列表
func (d *TaskDispatcher) ListDispatchRules(activeOnly bool) ([]models.TaskDispatchRule, error) {
	var rules []models.TaskDispatchRule
	tx := d.db.Model(&models.TaskDispatchRule{})
	if activeOnly {
		tx = tx.Where("is_active = ?", true)
	}
	if err := tx.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ==================== 事件触发 ====================

// OnMilestoneEvent 节点事件触发
// 当节点状态变化时调用，自动创建相关任务
func (d *TaskDispatcher) OnMilestoneEvent(event string, milestone *models.ShipmentMilestone) error {
	// 查找匹配的分发规则
	var rules []models.TaskDispatchRule
	if err := d.db.Where("is_active = ? AND trigger_event = ?", true, event).
		Find(&rules).Error; err != nil {
		return err
	}

	for _, rule := range rules {
		// 检查节点匹配
		if rule.TriggerMilestone != "" && rule.TriggerMilestone != milestone.NodeCode {
			continue
		}

		// 确定分配对象
		assigneeID, assigneeName := d.resolveAssignee(&rule, milestone)

		// 创建任务
		// 注意: ShipmentMilestone.ID 是 string, 但 Task.MilestoneID 是 *uint
		// 由于类型不匹配，暂时不关联 MilestoneID
		dueAt := time.Now().Add(time.Duration(rule.DueHours) * time.Hour)
		task := &models.Task{
			TaskNo:       d.generateTaskNo(),
			ShipmentID:   &milestone.ShipmentID,
			MilestoneID:  nil, // 类型不匹配，留空
			TaskType:     rule.TaskType,
			Title:        rule.TaskTitle,
			Description:  rule.TaskDescription,
			Priority:     rule.TaskPriority,
			AssigneeType: rule.AssignToType,
			AssigneeID:   &assigneeID,
			AssigneeName: assigneeName,
			Status:       models.TaskStatusAssigned,
			DueAt:        &dueAt,
			CreatedBy:    "system",
		}

		if assigneeID != "" {
			now := time.Now()
			task.AssignedAt = &now
		}

		if err := d.db.Create(task).Error; err != nil {
			continue // 单个任务失败不影响其他
		}

		// 生成Magic Link
		if rule.AutoGenerateLink && assigneeID != "" {
			_ = d.generateMagicLink(task)
		}
	}

	return nil
}

// resolveAssignee 解析分配对象
func (d *TaskDispatcher) resolveAssignee(rule *models.TaskDispatchRule, milestone *models.ShipmentMilestone) (string, string) {
	switch rule.AssignToType {
	case "specific_user":
		if rule.AssignToUserID != "" {
			return rule.AssignToUserID, d.getAssigneeName("user", rule.AssignToUserID)
		}
	case "shipment_partner":
		// 从运单关联的合作伙伴中查找指定角色
		if rule.AssignToRole != "" {
			var collab models.ShipmentCollaboration
			if err := d.db.Where("shipment_id = ? AND role = ?",
				milestone.ShipmentID, rule.AssignToRole).
				Preload("Partner").
				First(&collab).Error; err == nil {
				return collab.PartnerID, collab.Partner.Name
			}
		}
	}
	return "", ""
}

// ==================== 定时任务 ====================

// CheckOverdueTasks 检查逾期任务
func (d *TaskDispatcher) CheckOverdueTasks() ([]models.Task, error) {
	var tasks []models.Task
	if err := d.db.Where("due_at < ? AND status NOT IN ?", time.Now(),
		[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusCancelled, models.TaskStatusExpired}).
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	// 标记为逾期（但不改变状态，只是添加标记）
	for _, task := range tasks {
		// 这里可以发送提醒通知
		if !task.ReminderSent {
			now := time.Now()
			d.db.Model(&task).Updates(map[string]interface{}{
				"reminder_sent":    true,
				"reminder_sent_at": now,
			})
		}
	}

	return tasks, nil
}

// GetTaskStats 获取任务统计
func (d *TaskDispatcher) GetTaskStats(shipmentID *string) (map[string]interface{}, error) {
	tx := d.db.Model(&models.Task{})
	if shipmentID != nil {
		tx = tx.Where("shipment_id = ?", *shipmentID)
	}

	type StatusCount struct {
		Status models.TaskStatus `json:"status"`
		Count  int64             `json:"count"`
	}

	var statusCounts []StatusCount
	if err := tx.Select("status, count(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, err
	}

	// 逾期任务数
	var overdueCount int64
	d.db.Model(&models.Task{}).
		Where("due_at < ? AND status NOT IN ?", time.Now(),
			[]models.TaskStatus{models.TaskStatusCompleted, models.TaskStatusCancelled}).
		Count(&overdueCount)

	return map[string]interface{}{
		"by_status": statusCounts,
		"overdue":   overdueCount,
	}, nil
}
