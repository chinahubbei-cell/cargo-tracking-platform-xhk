package handlers

import (
	"net/http"

	"trackcard-server/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CustomerHandler struct {
	db *gorm.DB
}

func NewCustomerHandler(db *gorm.DB) *CustomerHandler {
	return &CustomerHandler{db: db}
}

// List 获取客户列表
func (h *CustomerHandler) List(c *gin.Context) {
	// 获取当前用户的组织ID
	orgID := c.GetString("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "组织ID未设置"})
		return
	}

	// 获取筛选条件
	customerType := c.Query("type") // sender / receiver

	var customers []models.Customer
	query := h.db.Where("org_id = ?", orgID)

	if customerType != "" {
		query = query.Where("type = ?", customerType)
	}

	if err := query.Order("created_at DESC").Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取客户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": customers})
}

// Get 获取客户详情
func (h *CustomerHandler) Get(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")

	var customer models.Customer
	if err := h.db.Where("id = ? AND org_id = ?", id, orgID).First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "客户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取客户详情失败"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// Create 创建客户
func (h *CustomerHandler) Create(c *gin.Context) {
	var req models.CustomerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 使用请求中的组织ID或用户的当前组织ID
	orgID := req.OrgID
	if orgID == "" {
		orgID = c.GetString("org_id")
	}
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "组织ID未设置"})
		return
	}

	// 检查是否已存在相同手机号的客户（同组织、同类型）
	var existing models.Customer
	if err := h.db.Where("org_id = ? AND type = ? AND phone = ?", orgID, req.Type, req.Phone).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该手机号已存在同类型客户记录"})
		return
	}

	customer := models.Customer{
		OrgID:   orgID,
		Type:    req.Type,
		Name:    req.Name,
		Phone:   req.Phone,
		Company: req.Company,
		Address: req.Address,
		City:    req.City,
		Country: req.Country,
	}

	if err := h.db.Create(&customer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建客户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

// Update 更新客户
func (h *CustomerHandler) Update(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")

	var customer models.Customer
	if err := h.db.Where("id = ? AND org_id = ?", id, orgID).First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "客户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取客户失败"})
		return
	}

	var req models.CustomerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 如果更新手机号，需要检查唯一性
	if req.Phone != nil && *req.Phone != customer.Phone {
		var existing models.Customer
		if err := h.db.Where("org_id = ? AND type = ? AND phone = ? AND id != ?", orgID, customer.Type, *req.Phone, id).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "该手机号已存在同类型客户记录"})
			return
		}
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Company != nil {
		updates["company"] = *req.Company
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.Country != nil {
		updates["country"] = *req.Country
	}

	if len(updates) > 0 {
		if err := h.db.Model(&customer).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新客户失败"})
			return
		}
	}

	// 重新获取更新后的数据
	h.db.First(&customer, "id = ?", id)
	c.JSON(http.StatusOK, customer)
}

// Delete 删除客户
func (h *CustomerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("org_id")

	result := h.db.Where("id = ? AND org_id = ?", id, orgID).Delete(&models.Customer{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除客户失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "客户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// Search 按手机号模糊搜索客户
func (h *CustomerHandler) Search(c *gin.Context) {
	orgID := c.GetString("org_id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "组织ID未设置"})
		return
	}

	phone := c.Query("phone")
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号"})
		return
	}

	customerType := c.Query("type")

	var customers []models.Customer
	// 使用 LIKE 进行模糊匹配
	query := h.db.Where("org_id = ? AND phone LIKE ?", orgID, phone+"%")
	if customerType != "" {
		query = query.Where("type = ?", customerType)
	}

	if err := query.Limit(10).Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索失败"})
		return
	}

	// 返回结果列表（即使为空也返回空数组）
	c.JSON(http.StatusOK, gin.H{"success": true, "data": customers})
}

// SaveOrUpdateFromShipment 从运单保存或更新客户（内部调用）
func (h *CustomerHandler) SaveOrUpdateFromShipment(orgID string, customerType models.CustomerType, name, phone, address string) error {
	if phone == "" || name == "" {
		return nil // 没有手机号或姓名则跳过
	}

	var existing models.Customer
	err := h.db.Where("org_id = ? AND type = ? AND phone = ?", orgID, customerType, phone).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 新建
		customer := models.Customer{
			OrgID:   orgID,
			Type:    customerType,
			Name:    name,
			Phone:   phone,
			Address: address,
		}
		return h.db.Create(&customer).Error
	} else if err != nil {
		return err
	}

	// 已存在，更新信息（如果有变化）
	updates := make(map[string]interface{})
	if name != "" && name != existing.Name {
		updates["name"] = name
	}
	if address != "" && address != existing.Address {
		updates["address"] = address
	}
	if len(updates) > 0 {
		return h.db.Model(&existing).Updates(updates).Error
	}
	return nil
}
