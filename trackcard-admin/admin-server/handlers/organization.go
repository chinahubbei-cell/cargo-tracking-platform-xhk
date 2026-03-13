package handlers

import (
	"net/http"
	"strconv"
	"time"

	"trackcard-admin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrganizationHandler struct {
	db *gorm.DB
}

func NewOrganizationHandler(db *gorm.DB) *OrganizationHandler {
	return &OrganizationHandler{db: db}
}

// PaginationParams 通用分页参数
type PaginationParams struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (p *PaginationParams) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 1000 {
		p.PageSize = 1000
	}
}

// List 获取组织列表（带分页）
func (h *OrganizationHandler) List(c *gin.Context) {
	var page PaginationParams
	c.ShouldBindQuery(&page)
	page.Normalize()

	var total int64
	var orgs []models.Organization

	query := h.db.Model(&models.Organization{})

	// 筛选条件
	if status := c.Query("status"); status != "" {
		query = query.Where("service_status = ?", status)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR contact_name LIKE ? OR contact_phone LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 仅展示一级和二级机构
	query = query.Where("level <= ?", 2)

	// 获取总数
	query.Count(&total)

	// 分页查询
	offset := (page.Page - 1) * page.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(page.PageSize).Find(&orgs)

	// 批量获取设备统计（避免N+1）及父级名称
	if len(orgs) > 0 {
		orgIDs := make([]string, len(orgs))
		var parentIDs []string
		for i, org := range orgs {
			orgIDs[i] = org.ID
			if org.ParentID != "" {
				parentIDs = append(parentIDs, org.ParentID)
			}
		}

		// 获取上级组织
		if len(parentIDs) > 0 {
			var parents []models.Organization
			h.db.Where("id IN ?", parentIDs).Find(&parents)
			parentMap := make(map[string]string)
			for _, p := range parents {
				parentMap[p.ID] = p.Name
			}
			for i := range orgs {
				if orgs[i].ParentID != "" {
					orgs[i].ParentName = parentMap[orgs[i].ParentID]
				}
			}
		}

		type DeviceCount struct {
			OrgID string `gorm:"column:org_id"`
			Count int    `gorm:"column:count"`
		}
		var counts []DeviceCount
		h.db.Table("hardware_devices").
			Select("org_id, COUNT(*) as count").
			Where("org_id IN ?", orgIDs).
			Group("org_id").
			Find(&counts)

		countMap := make(map[string]int)
		for _, c := range counts {
			countMap[c.OrgID] = c.Count
		}
		for i := range orgs {
			orgs[i].DeviceCount = countMap[orgs[i].ID]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orgs,
		"pagination": gin.H{
			"page":        page.Page,
			"page_size":   page.PageSize,
			"total":       total,
			"total_pages": (total + int64(page.PageSize) - 1) / int64(page.PageSize),
		},
	})
}

type CreateOrgRequest struct {
	Name         string `json:"name" binding:"required"`
	ParentID     string `json:"parent_id"`
	ShortName    string `json:"short_name"`
	CompanyName  string `json:"company_name"`
	CreditCode   string `json:"credit_code"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email"`
	Address      string `json:"address"`
	Remark       string `json:"remark"`
	// 服务配置（仅主账号使用）
	ServiceStatus string `json:"service_status"`
	ServiceStart  string `json:"service_start"`
	ServiceEnd    string `json:"service_end"`
	MaxDevices    int    `json:"max_devices"`
	MaxUsers      int    `json:"max_users"`
	MaxShipments  int    `json:"max_shipments"`
}

// Create 创建组织
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "组织名称不能为空"})
		return
	}

	// 检查重名
	var existing models.Organization
	if h.db.Where("name = ?", req.Name).First(&existing).Error == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "DUPLICATE_NAME", "message": "组织名称已存在"})
		return
	}

	// 确定层级
	level := 1
	if req.ParentID != "" {
		var parentOrg models.Organization
		if h.db.First(&parentOrg, "id = ?", req.ParentID).Error == nil {
			level = parentOrg.Level + 1
		}
	}

	org := models.Organization{
		Name:         req.Name,
		ParentID:     req.ParentID,
		Level:        level,
		ShortName:    req.ShortName,
		CompanyName:  req.CompanyName,
		CreditCode:   req.CreditCode,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		Address:      req.Address,
		Remark:       req.Remark,
	}

	// 主账号设置服务配置，二级账号继承主账号
	if req.ParentID == "" {
		org.ServiceStatus = "trial"
		if req.ServiceStatus != "" {
			org.ServiceStatus = req.ServiceStatus
		}
		org.MaxDevices = 10
		if req.MaxDevices > 0 {
			org.MaxDevices = req.MaxDevices
		}
		org.MaxUsers = 5
		if req.MaxUsers > 0 {
			org.MaxUsers = req.MaxUsers
		}
		org.MaxShipments = 100
		if req.MaxShipments > 0 {
			org.MaxShipments = req.MaxShipments
		}
		if req.ServiceStart != "" {
			if t, err := time.Parse(time.RFC3339, req.ServiceStart); err == nil {
				org.ServiceStart = &t
			} else if t, err := time.Parse(time.RFC3339Nano, req.ServiceStart); err == nil {
				org.ServiceStart = &t
			}
		}
		if req.ServiceEnd != "" {
			if t, err := time.Parse(time.RFC3339, req.ServiceEnd); err == nil {
				org.ServiceEnd = &t
			} else if t, err := time.Parse(time.RFC3339Nano, req.ServiceEnd); err == nil {
				org.ServiceEnd = &t
			}
		}
	}

	if err := h.db.Create(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "CREATE_FAILED", "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": org})
}

// Get 获取组织详情
func (h *OrganizationHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_ID", "message": "ID不能为空"})
		return
	}

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "组织不存在"})
		return
	}

	// 统计设备数
	var deviceCount int64
	h.db.Model(&models.HardwareDevice{}).Where("org_id = ?", id).Count(&deviceCount)
	org.DeviceCount = int(deviceCount)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": org})
}

type UpdateOrgRequest struct {
	Name          *string `json:"name"`
	ParentID      *string `json:"parent_id"`
	ShortName     *string `json:"short_name"`
	CompanyName   *string `json:"company_name"`
	CreditCode    *string `json:"credit_code"`
	ContactName   *string `json:"contact_name"`
	ContactPhone  *string `json:"contact_phone"`
	ContactEmail  *string `json:"contact_email"`
	Address       *string `json:"address"`
	Remark        *string `json:"remark"`
	ServiceStatus *string `json:"service_status"`
	ServiceStart  *string `json:"service_start"`
	ServiceEnd    *string `json:"service_end"`
	MaxDevices    *int    `json:"max_devices"`
	MaxUsers      *int    `json:"max_users"`
	MaxShipments  *int    `json:"max_shipments"`
}

// Update 更新组织（白名单限制）
func (h *OrganizationHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "组织不存在"})
		return
	}

	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "请求参数错误"})
		return
	}

	// 只更新允许的字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		// 检查重名
		var existing models.Organization
		if h.db.Where("name = ? AND id != ?", *req.Name, id).First(&existing).Error == nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "DUPLICATE_NAME", "message": "组织名称已存在"})
			return
		}
		updates["name"] = *req.Name
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
		if *req.ParentID != "" {
			var parentOrg models.Organization
			if h.db.First(&parentOrg, "id = ?", *req.ParentID).Error == nil {
				updates["level"] = parentOrg.Level + 1
			}
		} else {
			updates["level"] = 1
		}
	}
	if req.ShortName != nil {
		updates["short_name"] = *req.ShortName
	}
	if req.CompanyName != nil {
		updates["company_name"] = *req.CompanyName
	}
	if req.CreditCode != nil {
		updates["credit_code"] = *req.CreditCode
	}
	if req.ContactName != nil {
		updates["contact_name"] = *req.ContactName
	}
	if req.ContactPhone != nil {
		updates["contact_phone"] = *req.ContactPhone
	}
	if req.ContactEmail != nil {
		updates["contact_email"] = *req.ContactEmail
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if req.ServiceStatus != nil {
		updates["service_status"] = *req.ServiceStatus
	}
	if req.ServiceStart != nil {
		if *req.ServiceStart == "" {
			updates["service_start"] = nil
		} else {
			if t, err := time.Parse(time.RFC3339Nano, *req.ServiceStart); err == nil {
				updates["service_start"] = t
			} else if t, err := time.Parse(time.RFC3339, *req.ServiceStart); err == nil {
				updates["service_start"] = t
			}
		}
	}
	if req.ServiceEnd != nil {
		if *req.ServiceEnd == "" {
			updates["service_end"] = nil
		} else {
			if t, err := time.Parse(time.RFC3339Nano, *req.ServiceEnd); err == nil {
				updates["service_end"] = t
			} else if t, err := time.Parse(time.RFC3339, *req.ServiceEnd); err == nil {
				updates["service_end"] = t
			}
		}
	}
	if req.MaxDevices != nil {
		updates["max_devices"] = *req.MaxDevices
	}
	if req.MaxUsers != nil {
		updates["max_users"] = *req.MaxUsers
	}
	if req.MaxShipments != nil {
		updates["max_shipments"] = *req.MaxShipments
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		h.db.Model(&org).Updates(updates)
	}

	// 重新加载
	h.db.First(&org, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": org})
}

// Delete 删除组织
func (h *OrganizationHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "组织不存在"})
		return
	}

	// 检查是否有关联设备
	var deviceCount int64
	h.db.Model(&models.HardwareDevice{}).Where("org_id = ?", id).Count(&deviceCount)
	if deviceCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "HAS_DEVICES", "message": "请先解绑该组织下的设备"})
		return
	}

	h.db.Delete(&org)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

type SetServiceRequest struct {
	ServiceStatus string     `json:"service_status"`
	ServiceStart  *time.Time `json:"service_start"`
	ServiceEnd    *time.Time `json:"service_end"`
	AutoRenew     bool       `json:"auto_renew"`
	MaxDevices    int        `json:"max_devices"`
	MaxUsers      int        `json:"max_users"`
	MaxShipments  int        `json:"max_shipments"`
}

// SetService 设置服务期限（手工）
func (h *OrganizationHandler) SetService(c *gin.Context) {
	id := c.Param("id")
	adminID := c.GetString("user_id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "组织不存在"})
		return
	}

	var req SetServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "请求参数错误"})
		return
	}

	// 验证服务状态
	validStatuses := map[string]bool{"trial": true, "active": true, "suspended": true, "expired": true}
	if req.ServiceStatus != "" && !validStatuses[req.ServiceStatus] {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_STATUS", "message": "无效的服务状态"})
		return
	}

	oldEndDate := org.ServiceEnd

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.ServiceStatus != "" {
		updates["service_status"] = req.ServiceStatus
	}
	if req.ServiceStart != nil {
		updates["service_start"] = req.ServiceStart
	}
	if req.ServiceEnd != nil {
		updates["service_end"] = req.ServiceEnd
	}
	updates["auto_renew"] = req.AutoRenew
	if req.MaxDevices > 0 {
		updates["max_devices"] = req.MaxDevices
	}
	if req.MaxUsers > 0 {
		updates["max_users"] = req.MaxUsers
	}
	if req.MaxShipments > 0 {
		updates["max_shipments"] = req.MaxShipments
	}

	h.db.Model(&org).Updates(updates)

	// 记录续费日志
	if req.ServiceEnd != nil && (oldEndDate == nil || !oldEndDate.Equal(*req.ServiceEnd)) {
		renewal := &models.ServiceRenewal{
			OrgID:       id,
			RenewalType: "manual",
			OldEndDate:  oldEndDate,
			NewEndDate:  req.ServiceEnd,
			CreatedBy:   adminID,
		}
		h.db.Create(renewal)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "服务设置成功"})
}

type RenewRequest struct {
	PeriodMonths int     `json:"period_months" binding:"required,min=1,max=36"`
	Amount       float64 `json:"amount" binding:"min=0"`
	Remark       string  `json:"remark"`
}

// Renew 手动续费
func (h *OrganizationHandler) Renew(c *gin.Context) {
	id := c.Param("id")
	adminID := c.GetString("user_id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "组织不存在"})
		return
	}

	var req RenewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "续费月数必须在1-36之间"})
		return
	}

	// 计算新到期时间
	var baseDate time.Time
	if org.ServiceEnd != nil && org.ServiceEnd.After(time.Now()) {
		baseDate = *org.ServiceEnd
	} else {
		baseDate = time.Now()
	}
	newEndDate := baseDate.AddDate(0, req.PeriodMonths, 0)
	oldEndDate := org.ServiceEnd

	// 更新组织
	h.db.Model(&org).Updates(map[string]interface{}{
		"service_status": "active",
		"service_end":    newEndDate,
		"updated_at":     time.Now(),
	})

	// 记录续费
	renewal := &models.ServiceRenewal{
		OrgID:        id,
		RenewalType:  "manual",
		PeriodMonths: req.PeriodMonths,
		Amount:       req.Amount,
		OldEndDate:   oldEndDate,
		NewEndDate:   &newEndDate,
		CreatedBy:    adminID,
	}
	h.db.Create(renewal)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "续费成功",
		"data": gin.H{
			"new_end_date": newEndDate,
		},
	})
}

// GetExpiring 获取即将到期的组织
func (h *OrganizationHandler) GetExpiring(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	expireDate := time.Now().AddDate(0, 0, days)

	var orgs []models.Organization
	h.db.Where("service_end IS NOT NULL AND service_end <= ? AND service_status = ?",
		expireDate, "active").Order("service_end ASC").Limit(50).Find(&orgs)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": orgs})
}

// GetRenewals 获取续费记录
func (h *OrganizationHandler) GetRenewals(c *gin.Context) {
	id := c.Param("id")

	var renewals []models.ServiceRenewal
	h.db.Where("org_id = ?", id).Order("created_at DESC").Find(&renewals)

	// 填充操作人名称
	type RenewalWithAdmin struct {
		models.ServiceRenewal
		CreatedByName string `json:"created_by_name"`
	}
	var result []RenewalWithAdmin
	for _, r := range renewals {
		item := RenewalWithAdmin{ServiceRenewal: r}
		if r.CreatedBy != "" {
			var admin models.AdminUser
			if h.db.First(&admin, "id = ?", r.CreatedBy).Error == nil {
				item.CreatedByName = admin.Name
			}
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// GetDevices 获取客户关联的设备
func (h *OrganizationHandler) GetDevices(c *gin.Context) {
	id := c.Param("id")

	var devices []models.HardwareDevice
	h.db.Where("org_id = ?", id).Order("created_at DESC").Find(&devices)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": devices})
}

// GetStats 获取客户配额使用统计
func (h *OrganizationHandler) GetStats(c *gin.Context) {
	id := c.Param("id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "客户不存在"})
		return
	}

	// 设备数
	var deviceCount int64
	h.db.Model(&models.HardwareDevice{}).Where("org_id = ?", id).Count(&deviceCount)

	// 用户数 - 从主系统的 user_organizations 表统计
	var userCount int64
	h.db.Table("user_organizations").Where("organization_id = ?", id).Count(&userCount)

	// 本月运单数 - 从主系统的 shipments 表统计
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var shipmentCount int64
	h.db.Table("shipments").Where("org_id = ? AND created_at >= ? AND deleted_at IS NULL", id, monthStart).Count(&shipmentCount)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"device_count":   deviceCount,
			"max_devices":    org.MaxDevices,
			"user_count":     userCount,
			"max_users":      org.MaxUsers,
			"shipment_count": shipmentCount,
			"max_shipments":  org.MaxShipments,
		},
	})
}
