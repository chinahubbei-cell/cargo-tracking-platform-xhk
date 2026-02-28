package services

import (
	"encoding/json"
	"fmt"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// MilestoneEngine 节点配置引擎
type MilestoneEngine struct {
	db             *gorm.DB
	taskDispatcher *TaskDispatcher
}

var milestoneEngine *MilestoneEngine

// InitMilestoneEngine 初始化节点配置引擎
func InitMilestoneEngine(db *gorm.DB) {
	milestoneEngine = &MilestoneEngine{
		db:             db,
		taskDispatcher: GetTaskDispatcher(),
	}
}

// GetMilestoneEngine 获取节点配置引擎实例
func GetMilestoneEngine() *MilestoneEngine {
	return milestoneEngine
}

// ==================== 物流产品管理 ====================

// CreateProduct 创建物流产品
func (e *MilestoneEngine) CreateProduct(req *models.CreateProductRequest) (*models.LogisticsProduct, error) {
	product := &models.LogisticsProduct{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		IsActive:    true,
	}
	if err := e.db.Create(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

// GetProducts 获取所有物流产品
func (e *MilestoneEngine) GetProducts() ([]models.ProductResponse, error) {
	var products []models.LogisticsProduct
	if err := e.db.Preload("Templates").Find(&products).Error; err != nil {
		return nil, err
	}

	responses := make([]models.ProductResponse, len(products))
	for i, p := range products {
		responses[i] = models.ProductResponse{
			ID:            p.ID,
			Name:          p.Name,
			Code:          p.Code,
			Description:   p.Description,
			IsActive:      p.IsActive,
			TemplateCount: len(p.Templates),
		}
	}
	return responses, nil
}

// GetProduct 获取单个物流产品
func (e *MilestoneEngine) GetProduct(id string) (*models.ProductResponse, error) {
	var product models.LogisticsProduct
	if err := e.db.Preload("Templates.Nodes").First(&product, "id = ?", id).Error; err != nil {
		return nil, err
	}

	templates := make([]models.TemplateResponse, len(product.Templates))
	for i, t := range product.Templates {
		templates[i] = e.templateToResponse(&t)
	}

	return &models.ProductResponse{
		ID:            product.ID,
		Name:          product.Name,
		Code:          product.Code,
		Description:   product.Description,
		IsActive:      product.IsActive,
		TemplateCount: len(product.Templates),
		Templates:     templates,
	}, nil
}

// GetProductByCode 根据代码获取物流产品
func (e *MilestoneEngine) GetProductByCode(code string) (*models.LogisticsProduct, error) {
	var product models.LogisticsProduct
	if err := e.db.First(&product, "code = ?", code).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// ==================== 模板管理 ====================

// CreateTemplate 创建节点模板
func (e *MilestoneEngine) CreateTemplate(req *models.CreateTemplateRequest) (*models.MilestoneTemplate, error) {
	template := &models.MilestoneTemplate{
		ProductID:   req.ProductID,
		Name:        req.Name,
		Description: req.Description,
		Version:     1,
		IsActive:    true,
		IsDefault:   req.IsDefault,
	}

	// 如果设为默认，取消其他默认模板
	if req.IsDefault {
		e.db.Model(&models.MilestoneTemplate{}).
			Where("product_id = ? AND is_default = ?", req.ProductID, true).
			Update("is_default", false)
	}

	if err := e.db.Create(template).Error; err != nil {
		return nil, err
	}
	return template, nil
}

// GetTemplates 获取模板列表
func (e *MilestoneEngine) GetTemplates(productID string) ([]models.TemplateResponse, error) {
	var templates []models.MilestoneTemplate
	query := e.db.Preload("Nodes").Preload("Product")
	if productID != "" {
		query = query.Where("product_id = ?", productID)
	}
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}

	responses := make([]models.TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = e.templateToResponse(&t)
	}
	return responses, nil
}

// GetTemplate 获取单个模板
func (e *MilestoneEngine) GetTemplate(id string) (*models.TemplateResponse, error) {
	var template models.MilestoneTemplate
	if err := e.db.Preload("Nodes", func(db *gorm.DB) *gorm.DB {
		return db.Order("node_order ASC")
	}).Preload("Product").First(&template, "id = ?", id).Error; err != nil {
		return nil, err
	}

	resp := e.templateToResponse(&template)
	return &resp, nil
}

// GetDefaultTemplate 获取产品的默认模板
func (e *MilestoneEngine) GetDefaultTemplate(productID string) (*models.MilestoneTemplate, error) {
	var template models.MilestoneTemplate
	if err := e.db.Preload("Nodes", func(db *gorm.DB) *gorm.DB {
		return db.Order("node_order ASC")
	}).First(&template, "product_id = ? AND is_default = ?", productID, true).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

func (e *MilestoneEngine) templateToResponse(t *models.MilestoneTemplate) models.TemplateResponse {
	nodes := make([]models.NodeResponse, len(t.Nodes))
	for i, n := range t.Nodes {
		nodes[i] = e.nodeToResponse(&n)
	}

	productName := ""
	if t.Product.ID != "" {
		productName = t.Product.Name
	}

	return models.TemplateResponse{
		ID:          t.ID,
		ProductID:   t.ProductID,
		ProductName: productName,
		Name:        t.Name,
		Description: t.Description,
		Version:     t.Version,
		IsActive:    t.IsActive,
		IsDefault:   t.IsDefault,
		NodeCount:   len(t.Nodes),
		Nodes:       nodes,
	}
}

// ==================== 节点管理 ====================

// CreateNode 创建节点
func (e *MilestoneEngine) CreateNode(req *models.CreateNodeRequest) (*models.MilestoneNode, error) {
	var triggerConditions []byte
	if req.TriggerConditions != nil {
		triggerConditions, _ = json.Marshal(req.TriggerConditions)
	}

	nodeType := req.NodeType
	if nodeType == "" {
		nodeType = models.NodeTypeStandard
	}

	node := &models.MilestoneNode{
		TemplateID:        req.TemplateID,
		NodeCode:          req.NodeCode,
		NodeName:          req.NodeName,
		NodeNameEn:        req.NodeNameEn,
		NodeType:          nodeType,
		NodeOrder:         req.NodeOrder,
		ParentNodeID:      req.ParentNodeID,
		GroupCode:         req.GroupCode,
		GroupName:         req.GroupName,
		IsMandatory:       req.IsMandatory,
		IsVisible:         req.IsVisible,
		Icon:              req.Icon,
		TriggerType:       req.TriggerType,
		TriggerConditions: triggerConditions,
		TimeoutHours:      req.TimeoutHours,
		TimeoutAction:     req.TimeoutAction,
		ResponsibleRole:   req.ResponsibleRole,
	}

	if err := e.db.Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// GetNodes 获取模板的所有节点
func (e *MilestoneEngine) GetNodes(templateID string) ([]models.NodeResponse, error) {
	var nodes []models.MilestoneNode
	if err := e.db.Where("template_id = ?", templateID).
		Order("node_order ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	responses := make([]models.NodeResponse, len(nodes))
	for i, n := range nodes {
		responses[i] = e.nodeToResponse(&n)
	}
	return responses, nil
}

// UpdateNode 更新节点
func (e *MilestoneEngine) UpdateNode(nodeID string, req *models.CreateNodeRequest) error {
	updates := map[string]interface{}{
		"node_name":        req.NodeName,
		"node_name_en":     req.NodeNameEn,
		"node_order":       req.NodeOrder,
		"parent_node_id":   req.ParentNodeID,
		"group_code":       req.GroupCode,
		"group_name":       req.GroupName,
		"is_mandatory":     req.IsMandatory,
		"is_visible":       req.IsVisible,
		"icon":             req.Icon,
		"trigger_type":     req.TriggerType,
		"timeout_hours":    req.TimeoutHours,
		"timeout_action":   req.TimeoutAction,
		"responsible_role": req.ResponsibleRole,
		"updated_at":       time.Now(),
	}

	if req.TriggerConditions != nil {
		triggerConditions, _ := json.Marshal(req.TriggerConditions)
		updates["trigger_conditions"] = triggerConditions
	}

	return e.db.Model(&models.MilestoneNode{}).Where("id = ?", nodeID).Updates(updates).Error
}

// DeleteNode 删除节点
func (e *MilestoneEngine) DeleteNode(nodeID string) error {
	return e.db.Delete(&models.MilestoneNode{}, "id = ?", nodeID).Error
}

func (e *MilestoneEngine) nodeToResponse(n *models.MilestoneNode) models.NodeResponse {
	return models.NodeResponse{
		ID:              n.ID,
		NodeCode:        n.NodeCode,
		NodeName:        n.NodeName,
		NodeNameEn:      n.NodeNameEn,
		NodeType:        n.NodeType,
		NodeOrder:       n.NodeOrder,
		ParentNodeID:    n.ParentNodeID,
		GroupCode:       n.GroupCode,
		GroupName:       n.GroupName,
		IsMandatory:     n.IsMandatory,
		IsVisible:       n.IsVisible,
		Icon:            n.Icon,
		TriggerType:     n.TriggerType,
		TimeoutHours:    n.TimeoutHours,
		TimeoutAction:   n.TimeoutAction,
		ResponsibleRole: n.ResponsibleRole,
	}
}

// ==================== 运单节点实例管理 ====================

// CreateMilestonesForShipment 为运单创建节点实例
func (e *MilestoneEngine) CreateMilestonesForShipment(shipmentID string, templateID string) error {
	// 获取模板节点
	var nodes []models.MilestoneNode
	if err := e.db.Where("template_id = ?", templateID).
		Order("node_order ASC").
		Find(&nodes).Error; err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("模板无节点定义")
	}

	// 创建运单节点实例
	milestones := make([]models.ShipmentMilestone, len(nodes))
	for i, node := range nodes {
		milestones[i] = models.ShipmentMilestone{
			ShipmentID: shipmentID,
			NodeID:     node.ID,
			NodeCode:   node.NodeCode,
			NodeName:   node.NodeName,
			NodeOrder:  node.NodeOrder,
			GroupCode:  node.GroupCode,
			GroupName:  node.GroupName,
			Status:     models.StageStatusPending,
		}
	}

	// 第一个节点设为进行中
	if len(milestones) > 0 {
		milestones[0].Status = models.StageStatusInProgress
		milestones[0].ActualStart = timePtr(time.Now())
	}

	if err := e.db.Create(&milestones).Error; err != nil {
		return err
	}

	// 触发第一个节点的开始事件
	if len(milestones) > 0 {
		go func() {
			_ = e.taskDispatcher.OnMilestoneEvent("milestone_started", &milestones[0])
		}()
	}

	return nil
}

// GetShipmentMilestones 获取运单的所有节点实例
func (e *MilestoneEngine) GetShipmentMilestones(shipmentID string) ([]models.ShipmentMilestone, error) {
	var milestones []models.ShipmentMilestone
	if err := e.db.Preload("Node").
		Where("shipment_id = ?", shipmentID).
		Order("node_order ASC").
		Find(&milestones).Error; err != nil {
		return nil, err
	}
	return milestones, nil
}

// UpdateMilestoneStatus 更新节点实例状态
func (e *MilestoneEngine) UpdateMilestoneStatus(milestoneID string, status models.StageStatus, triggerType models.TriggerType, note string) error {
	updates := map[string]interface{}{
		"status":       status,
		"trigger_type": triggerType,
		"trigger_note": note,
		"updated_at":   time.Now(),
	}

	if status == models.StageStatusInProgress {
		updates["actual_start"] = time.Now()
	} else if status == models.StageStatusCompleted {
		updates["actual_end"] = time.Now()
	}

	return e.db.Model(&models.ShipmentMilestone{}).Where("id = ?", milestoneID).Updates(updates).Error
}

// TransitionToNextMilestone 推进到下一节点
func (e *MilestoneEngine) TransitionToNextMilestone(shipmentID string, triggerType models.TriggerType, note string) error {
	err := e.db.Transaction(func(tx *gorm.DB) error {
		// 获取当前进行中的节点
		var currentMilestone models.ShipmentMilestone
		if err := tx.Where("shipment_id = ? AND status = ?", shipmentID, models.StageStatusInProgress).
			First(&currentMilestone).Error; err != nil {
			return fmt.Errorf("找不到进行中的节点: %v", err)
		}

		// 获取运单当前状态
		var shipment models.Shipment
		if err := tx.First(&shipment, "id = ?", shipmentID).Error; err != nil {
			return fmt.Errorf("找不到运单: %v", err)
		}

		// 完成当前节点
		now := time.Now()
		if err := tx.Model(&currentMilestone).Updates(map[string]interface{}{
			"status":       models.StageStatusCompleted,
			"actual_end":   now,
			"trigger_type": triggerType,
			"trigger_note": note,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}

		// 查找下一个节点
		var nextMilestone models.ShipmentMilestone
		if err := tx.Where("shipment_id = ? AND node_order > ?", shipmentID, currentMilestone.NodeOrder).
			Order("node_order ASC").
			First(&nextMilestone).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 没有下一个节点，运单完成
				updates := map[string]interface{}{
					"status":            "delivered",
					"progress":          100,
					"status_updated_at": now,
					"current_milestone": "arrived",
				}
				// 自动设置ATA（如果未设置）
				if shipment.ATA == nil {
					updates["ata"] = now
					updates["arrived_dest_at"] = now
				}
				// 修复: 流程结束自动送达时，必须截断轨迹
				updates["track_end_at"] = now
				return tx.Model(&models.Shipment{}).
					Where("id = ?", shipmentID).
					Updates(updates).Error
			}
			return err
		}

		// 激活下一个节点
		if err := tx.Model(&nextMilestone).Updates(map[string]interface{}{
			"status":       models.StageStatusInProgress,
			"actual_start": now,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}

		// 更新运单进度
		var totalCount, completedCount int64
		if err := tx.Model(&models.ShipmentMilestone{}).Where("shipment_id = ?", shipmentID).Count(&totalCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.ShipmentMilestone{}).Where("shipment_id = ? AND status = ?", shipmentID, models.StageStatusCompleted).Count(&completedCount).Error; err != nil {
			return err
		}

		progress := 0
		if totalCount > 0 {
			progress = int((completedCount * 100) / totalCount)
		}

		// 构建更新字段
		shipmentUpdates := map[string]interface{}{
			"current_stage": nextMilestone.NodeCode,
			"progress":      progress,
		}

		// 如果运单状态是pending且第一个环节完成，自动设置为in_transit
		if shipment.Status == "pending" && completedCount >= 1 {
			shipmentUpdates["status"] = "in_transit"
			shipmentUpdates["status_updated_at"] = now
			shipmentUpdates["current_milestone"] = "departed"
			// 自动设置ATD（如果未设置）
			if shipment.ATD == nil {
				shipmentUpdates["atd"] = now
				shipmentUpdates["left_origin_at"] = now
			}
		}

		return tx.Model(&models.Shipment{}).
			Where("id = ?", shipmentID).
			Updates(shipmentUpdates).Error
	})

	// 事务成功后触发任务分发和记录日志
	if err == nil {
		// 异步触发，避免阻塞
		go func() {
			// 获取当前进行中的节点信息
			var currentMilestone models.ShipmentMilestone
			if err := e.db.Where("shipment_id = ? AND status = ?", shipmentID, models.StageStatusInProgress).
				First(&currentMilestone).Error; err == nil {
				_ = e.taskDispatcher.OnMilestoneEvent("milestone_started", &currentMilestone)
			}

			// 获取上一个已完成的节点信息用于记录日志
			var prevMilestone models.ShipmentMilestone
			if err := e.db.Where("shipment_id = ? AND status = ?", shipmentID, models.StageStatusCompleted).
				Order("actual_end DESC").
				First(&prevMilestone).Error; err == nil {
				// 记录环节推进日志
				ShipmentLog.LogStageTransition(
					shipmentID,
					prevMilestone.NodeCode,
					currentMilestone.NodeCode,
					string(triggerType)+": "+note,
					"system",
					"",
				)
			}
		}()
	}

	return err
}

func timePtr(t time.Time) *time.Time {
	return &t
}
