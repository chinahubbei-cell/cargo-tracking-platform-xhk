package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/utils"
)

type PartnerHandler struct {
	db *gorm.DB
}

func NewPartnerHandler(db *gorm.DB) *PartnerHandler {
	return &PartnerHandler{db: db}
}

// List 获取合作伙伴列表
func (h *PartnerHandler) List(c *gin.Context) {
	var partners []models.Partner
	query := h.db.Model(&models.Partner{})

	// 物流环节筛选
	if stage := c.Query("stage"); stage != "" {
		query = query.Where("stage = ?", stage)
	}

	// 类型筛选
	if partnerType := c.Query("type"); partnerType != "" {
		query = query.Where("type = ?", partnerType)
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 国家筛选
	if country := c.Query("country"); country != "" {
		query = query.Where("country = ?", country)
	}

	// 搜索
	if search := c.Query("search"); search != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ? OR contact_name ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 组织过滤 (数据隔离)
	if orgID := c.Query("org_id"); orgID != "" {
		query = query.Where("owner_org_id = ?", orgID)
	}

	query = query.Order("stage ASC, created_at DESC")

	if err := query.Find(&partners).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 转换响应
	var responses []models.PartnerResponse
	for _, p := range partners {
		responses = append(responses, p.ToResponse())
	}

	utils.SuccessResponse(c, responses)
}

// Get 获取单个合作伙伴详情
func (h *PartnerHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var partner models.Partner
	if err := h.db.First(&partner, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "合作伙伴不存在")
		return
	}

	utils.SuccessResponse(c, partner.ToResponse())
}

// Create 创建合作伙伴
func (h *PartnerHandler) Create(c *gin.Context) {
	var req models.PartnerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的合作伙伴信息")
		return
	}

	// 从上下文获取组织ID
	orgID := ""
	if id, exists := c.Get("org_id"); exists {
		orgID = id.(string)
	}

	partner := models.Partner{
		Name:                req.Name,
		Code:                req.Code,
		Type:                req.Type,
		Stage:               req.Stage,
		SubType:             req.SubType,
		ContactName:         req.ContactName,
		Phone:               req.Phone,
		Email:               req.Email,
		Address:             req.Address,
		Country:             req.Country,
		Region:              req.Region,
		ServicePorts:        req.ServicePorts,
		ServiceRoutes:       req.ServiceRoutes,
		Certifications:      req.Certifications,
		APIConfig:           req.APIConfig,
		ServiceCapabilities: req.ServiceCapabilities,
		ContractInfo:        req.ContractInfo,
		PaymentTerms:        req.PaymentTerms,
		InsuranceCoverage:   req.InsuranceCoverage,
		Status:              "active",
		OwnerOrgID:          orgID,
	}

	if err := h.db.Create(&partner).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.CreatedResponse(c, partner.ToResponse())
}

// Update 更新合作伙伴
func (h *PartnerHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var partner models.Partner
	if err := h.db.First(&partner, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "合作伙伴不存在")
		return
	}

	var req models.PartnerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.SubType != nil {
		updates["sub_type"] = *req.SubType
	}
	if req.ContactName != nil {
		updates["contact_name"] = *req.ContactName
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Country != nil {
		updates["country"] = *req.Country
	}
	if req.Region != nil {
		updates["region"] = *req.Region
	}
	if req.ServicePorts != nil {
		updates["service_ports"] = *req.ServicePorts
	}
	if req.ServiceRoutes != nil {
		updates["service_routes"] = *req.ServiceRoutes
	}
	if req.Rating != nil {
		updates["rating"] = *req.Rating
	}
	if req.Certifications != nil {
		updates["certifications"] = *req.Certifications
	}
	if req.APIConfig != nil {
		updates["api_config"] = *req.APIConfig
	}
	if req.ServiceCapabilities != nil {
		updates["service_capabilities"] = *req.ServiceCapabilities
	}
	if req.ContractInfo != nil {
		updates["contract_info"] = *req.ContractInfo
	}
	if req.PaymentTerms != nil {
		updates["payment_terms"] = *req.PaymentTerms
	}
	if req.InsuranceCoverage != nil {
		updates["insurance_coverage"] = *req.InsuranceCoverage
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := h.db.Model(&partner).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.First(&partner, "id = ?", id)
	utils.SuccessResponse(c, partner.ToResponse())
}

// Delete 删除合作伙伴
func (h *PartnerHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	var partner models.Partner
	if err := h.db.First(&partner, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "合作伙伴不存在")
		return
	}

	// 检查是否有关联的协作记录
	var count int64
	h.db.Model(&models.ShipmentCollaboration{}).Where("partner_id = ?", id).Count(&count)
	if count > 0 {
		utils.BadRequest(c, "该合作伙伴有关联的协作记录，无法删除")
		return
	}

	if err := h.db.Delete(&partner).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// GetStats 获取合作伙伴统计
func (h *PartnerHandler) GetStats(c *gin.Context) {
	id := c.Param("id")

	var partner models.Partner
	if err := h.db.First(&partner, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "合作伙伴不存在")
		return
	}

	// 统计协作数据
	var totalCollabs int64
	var completedCollabs int64
	var avgDuration float64

	h.db.Model(&models.ShipmentCollaboration{}).Where("partner_id = ?", id).Count(&totalCollabs)
	h.db.Model(&models.ShipmentCollaboration{}).Where("partner_id = ? AND status = 'completed'", id).Count(&completedCollabs)

	// 计算平均完成时间
	var result struct {
		AvgDuration float64
	}
	h.db.Model(&models.ShipmentCollaboration{}).
		Select("AVG(EXTRACT(EPOCH FROM (completed_at - assigned_at))/3600) as avg_duration").
		Where("partner_id = ? AND status = 'completed' AND completed_at IS NOT NULL", id).
		Scan(&result)
	avgDuration = result.AvgDuration

	utils.SuccessResponse(c, gin.H{
		"partner_id":        id,
		"total_collabs":     totalCollabs,
		"completed_collabs": completedCollabs,
		"completion_rate":   float64(completedCollabs) / float64(max(totalCollabs, 1)) * 100,
		"avg_duration_hrs":  avgDuration,
	})
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// AssignToShipment 分配合作伙伴到运单
func (h *PartnerHandler) AssignToShipment(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	// 验证运单存在
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	var req models.CollaborationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的分配信息")
		return
	}

	// 验证合作伙伴存在
	var partner models.Partner
	if err := h.db.First(&partner, "id = ?", req.PartnerID).Error; err != nil {
		utils.NotFound(c, "合作伙伴不存在")
		return
	}

	// 获取分配人信息
	userID := ""
	if id, exists := c.Get("user_id"); exists {
		userID = id.(string)
	}

	collab := models.ShipmentCollaboration{
		ShipmentID: shipmentID,
		PartnerID:  req.PartnerID,
		Role:       req.Role,
		Status:     models.CollabStatusInvited,
		TaskDesc:   req.TaskDesc,
		AssignedAt: time.Now(),
		AssignedBy: userID,
	}

	if err := h.db.Create(&collab).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 更新合作伙伴统计
	h.db.Model(&partner).Update("total_shipments", gorm.Expr("total_shipments + 1"))

	// 重新加载关联
	h.db.Preload("Partner").First(&collab, collab.ID)

	utils.CreatedResponse(c, collab.ToResponse())
}

// GetShipmentCollaborations 获取运单的协作记录
func (h *PartnerHandler) GetShipmentCollaborations(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	var collabs []models.ShipmentCollaboration
	if err := h.db.Preload("Partner").
		Where("shipment_id = ?", shipmentID).
		Order("assigned_at DESC").
		Find(&collabs).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	var responses []models.CollaborationResponse
	for _, c := range collabs {
		responses = append(responses, c.ToResponse())
	}

	utils.SuccessResponse(c, responses)
}

// UpdateCollaboration 更新协作状态
func (h *PartnerHandler) UpdateCollaboration(c *gin.Context) {
	id := c.Param("id")

	var collab models.ShipmentCollaboration
	if err := h.db.First(&collab, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "协作记录不存在")
		return
	}

	var req models.CollaborationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Status != nil {
		updates["status"] = *req.Status
		// 设置时间戳
		now := time.Now()
		switch *req.Status {
		case models.CollabStatusAccepted:
			updates["accepted_at"] = now
		case models.CollabStatusCompleted:
			updates["completed_at"] = now
		}
	}
	if req.Remarks != nil {
		updates["remarks"] = *req.Remarks
	}

	if err := h.db.Model(&collab).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.Preload("Partner").First(&collab, id)
	utils.SuccessResponse(c, collab.ToResponse())
}
