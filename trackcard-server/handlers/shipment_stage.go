package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

// ShipmentStageHandler 运输环节处理器
type ShipmentStageHandler struct {
	db *gorm.DB
}

// NewShipmentStageHandler 创建运输环节处理器
func NewShipmentStageHandler(db *gorm.DB) *ShipmentStageHandler {
	return &ShipmentStageHandler{db: db}
}

// GetStages 获取运单的所有环节
// GET /api/shipments/:id/stages
func (h *ShipmentStageHandler) GetStages(c *gin.Context) {
	shipmentID := c.Param("id")

	// 检查运单是否存在
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "运单不存在"})
		return
	}

	svc := services.GetShipmentStageService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	stages, err := svc.GetStagesSummary(shipmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取环节失败: " + err.Error()})
		return
	}

	// 如果运单没有环节数据，自动初始化默认环节
	if len(stages) == 0 {
		if err := svc.CreateStagesForShipment(shipmentID, "", ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化环节失败: " + err.Error()})
			return
		}
		// 重新获取环节数据
		stages, err = svc.GetStagesSummary(shipmentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取环节失败: " + err.Error()})
			return
		}
		// 更新运单的当前环节为第一个环节
		h.db.Model(&shipment).Update("current_stage", "first_mile")
		shipment.CurrentStage = "first_mile"
	}

	// 进度逻辑优化:
	// 不再强制用环节数量覆盖进度(避免覆盖由于GPS计算出的精确进度)
	// 仅当进度为0时(新运单或无轨迹)，才使用环节数量进行估算兜底
	// 仅当进度为0时(新运单或无轨迹)，才使用环节数量进行估算兜底
	calculatedProgress := shipment.Progress
	if shipment.Progress == 0 {
		completedCount := 0
		for _, stage := range stages {
			if stage.Status == "completed" {
				completedCount++
			}
		}
		// 简单估算兜底
		estimated := completedCount * 20
		if estimated > 0 {
			calculatedProgress = estimated
			h.db.Model(&shipment).Update("progress", calculatedProgress)
			shipment.Progress = calculatedProgress
		}
	}

	// 计算总费用
	totalCost, _ := svc.CalculateTotalCost(shipmentID)

	// 构造响应数据
	data := gin.H{
		"stages":        stages,
		"current_stage": shipment.CurrentStage,
		"total_cost":    totalCost,
		"progress":      calculatedProgress,
	}

	// 使用标准响应包装
	utils.SuccessResponse(c, data)
}

// RegenerateAll 批量重新生成所有活跃运单的环节
// POST /api/admin/stages/regenerate
func (h *ShipmentStageHandler) RegenerateAll(c *gin.Context) {
	svc := services.GetShipmentStageService()
	if svc == nil {
		utils.InternalError(c, "服务未初始化")
		return
	}

	// 只重新生成未完成的运单，避免破坏历史数据
	var shipments []models.Shipment
	if err := h.db.Where("status IN ?", []string{"pending", "in_transit"}).Find(&shipments).Error; err != nil {
		utils.InternalError(c, "查询运单失败: "+err.Error())
		return
	}

	successCount := 0
	failureCount := 0

	for _, s := range shipments {
		if err := svc.RegenerateStages(s.ID); err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	utils.SuccessResponse(c, gin.H{
		"message":       "批量重新生成完成",
		"total":         len(shipments),
		"success_count": successCount,
		"failure_count": failureCount,
	})
}

// GetStage 获取单个环节详情
// GET /api/shipments/:id/stages/:stage_code
func (h *ShipmentStageHandler) GetStage(c *gin.Context) {
	shipmentID := c.Param("id")
	stageCode := models.StageCode(c.Param("stage_code"))

	svc := services.GetShipmentStageService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	stage, err := svc.GetStageByCode(shipmentID, stageCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "环节不存在"})
		return
	}

	c.JSON(http.StatusOK, stage.ToResponse())
}

// UpdateStage 更新环节信息
// PUT /api/shipments/:id/stages/:stage_code
func (h *ShipmentStageHandler) UpdateStage(c *gin.Context) {
	shipmentID := c.Param("id")
	stageCode := models.StageCode(c.Param("stage_code"))

	var req models.ShipmentStageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	svc := services.GetShipmentStageService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	// 获取环节
	stage, err := svc.GetStageByCode(shipmentID, stageCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "环节不存在"})
		return
	}

	// 更新环节
	if err := svc.UpdateStage(stage.ID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		return
	}

	// 同步关键数据到运单主表
	if err := svc.SyncDataToShipment(shipmentID, stageCode); err != nil {
		// 同步失败只记录日志，不影响主流程
		c.Set("sync_warning", err.Error())
	}

	// 记录操作日志
	user, _ := c.Get("user")
	if u, ok := user.(*models.User); ok {
		services.ShipmentLog.Log(
			shipmentID,
			"stage_update",
			"stage",
			"",
			string(stageCode),
			"更新环节["+models.GetStageName(stageCode)+"]信息",
			u.ID,
			c.ClientIP(),
		)
	}

	// 返回更新后的环节
	updatedStage, _ := svc.GetStageByCode(shipmentID, stageCode)
	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"stage":   updatedStage.ToResponse(),
	})
}

// TransitionStage 手动推进到下一环节
// POST /api/shipments/:id/stages/transition
func (h *ShipmentStageHandler) TransitionStage(c *gin.Context) {
	shipmentID := c.Param("id")

	var req struct {
		Note string `json:"note"` // 推进备注
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	if len(req.Note) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备注长度不能超过500字符"})
		return
	}

	// 检查运单是否存在
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "运单不存在"})
		return
	}

	svc := services.GetShipmentStageService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	// 执行环节推进
	if err := svc.TransitionToNextStage(shipmentID, models.TriggerManual, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "推进失败: " + err.Error()})
		return
	}

	// 记录操作日志（即使没有用户信息也记录）
	operatorID := "系统"
	user, exists := c.Get("user")
	if exists {
		if u, ok := user.(*models.User); ok {
			operatorID = u.ID
		}
	}

	// 获取更新后的运单信息以获取新的当前环节
	var updatedShipment models.Shipment
	h.db.First(&updatedShipment, "id = ?", shipmentID)

	services.ShipmentLog.LogStageTransition(
		shipmentID,
		shipment.CurrentStage,
		updatedShipment.CurrentStage,
		req.Note,
		operatorID,
		c.ClientIP(),
	)

	// 获取更新后的环节信息
	stages, _ := svc.GetStagesSummary(shipmentID)
	h.db.First(&shipment, "id = ?", shipmentID) // 刷新运单数据

	c.JSON(http.StatusOK, gin.H{
		"message":       "推进成功",
		"current_stage": shipment.CurrentStage,
		"stages":        stages,
	})
}

// CompleteStage 完成当前环节（不自动推进到下一环节）
// POST /api/shipments/:id/stages/:stage_code/complete
func (h *ShipmentStageHandler) CompleteStage(c *gin.Context) {
	shipmentID := c.Param("id")
	stageCode := models.StageCode(c.Param("stage_code"))

	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	if len(req.Note) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备注长度不能超过500字符"})
		return
	}

	svc := services.GetShipmentStageService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	// 获取环节
	stage, err := svc.GetStageByCode(shipmentID, stageCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "环节不存在"})
		return
	}

	// 状态校验：只能完成"进行中"的环节
	if stage.Status != models.StageStatusInProgress {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能完成进行中的环节，当前状态: " + string(stage.Status)})
		return
	}

	// 标记为完成，并设置实际结束时间
	now := time.Now()
	status := models.StageStatusCompleted
	trigger := models.TriggerManual
	updateReq := &models.ShipmentStageUpdateRequest{
		Status:      &status,
		ActualEnd:   &now,
		TriggerType: &trigger,
		TriggerNote: &req.Note,
	}

	if err := svc.UpdateStage(stage.ID, updateReq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "完成失败: " + err.Error()})
		return
	}

	// 记录日志
	user, _ := c.Get("user")
	if u, ok := user.(*models.User); ok {
		services.ShipmentLog.Log(
			shipmentID,
			"stage_complete",
			"stage",
			"",
			string(stageCode),
			"完成环节["+models.GetStageName(stageCode)+"]",
			u.ID,
			c.ClientIP(),
		)
	}

	c.JSON(http.StatusOK, gin.H{"message": "环节已完成"})
}

// StartStage 开始某个环节
// POST /api/shipments/:id/stages/:stage_code/start
func (h *ShipmentStageHandler) StartStage(c *gin.Context) {
	shipmentID := c.Param("id")
	stageCode := models.StageCode(c.Param("stage_code"))

	var req struct {
		PartnerID   string `json:"partner_id"`
		PartnerName string `json:"partner_name"`
		Note        string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	if len(req.Note) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备注长度不能超过500字符"})
		return
	}

	svc := services.GetShipmentStageService()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务未初始化"})
		return
	}

	// 获取环节
	stage, err := svc.GetStageByCode(shipmentID, stageCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "环节不存在"})
		return
	}

	// 状态校验：只能开始"待开始"的环节
	if stage.Status != models.StageStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能开始待开始的环节，当前状态: " + string(stage.Status)})
		return
	}

	// 顺序校验：检查前一个环节是否已完成（允许跳过检查第一个环节）
	stageOrder := models.GetStageOrder(stageCode)
	if stageOrder > 1 {
		allCodes := models.AllStageCodes()
		prevCode := allCodes[stageOrder-2] // 索引 = order - 2
		prevStage, _ := svc.GetStageByCode(shipmentID, prevCode)
		if prevStage != nil && prevStage.Status != models.StageStatusCompleted {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先完成上一环节: " + models.GetStageName(prevCode)})
			return
		}
	}

	// 更新为进行中，并设置实际开始时间
	now := time.Now()
	status := models.StageStatusInProgress
	trigger := models.TriggerManual
	updateReq := &models.ShipmentStageUpdateRequest{
		Status:      &status,
		ActualStart: &now,
		TriggerType: &trigger,
		TriggerNote: &req.Note,
	}
	if req.PartnerID != "" {
		updateReq.PartnerID = &req.PartnerID
	}
	if req.PartnerName != "" {
		updateReq.PartnerName = &req.PartnerName
	}

	if err := svc.UpdateStage(stage.ID, updateReq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "开始失败: " + err.Error()})
		return
	}

	// 更新运单当前环节
	h.db.Model(&models.Shipment{}).Where("id = ?", shipmentID).Update("current_stage", string(stageCode))

	// 记录日志
	user, _ := c.Get("user")
	if u, ok := user.(*models.User); ok {
		services.ShipmentLog.Log(
			shipmentID,
			"stage_start",
			"stage",
			"",
			string(stageCode),
			"开始环节["+models.GetStageName(stageCode)+"]",
			u.ID,
			c.ClientIP(),
		)
	}

	c.JSON(http.StatusOK, gin.H{"message": "环节已开始"})
}
