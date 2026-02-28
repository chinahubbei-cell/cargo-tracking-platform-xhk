package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
)

type OrganizationHandler struct {
	db *gorm.DB
}

func NewOrganizationHandler(db *gorm.DB) *OrganizationHandler {
	return &OrganizationHandler{db: db}
}

// List 获取组织列表（支持树形结构）
func (h *OrganizationHandler) List(c *gin.Context) {
	tree := c.Query("tree") == "true"
	search := c.Query("search")
	parentID := c.Query("parent_id")

	// 获取当前用户ID
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	userRole, _ := c.Get("user_role")
	userRoleStr, _ := userRole.(string)

	query := h.db.Model(&models.Organization{}).
		Preload("Leader").
		Where("deleted_at IS NULL")

	// 非管理员只能看到自己有权限的组织
	if userRoleStr != "admin" && userIDStr != "" {
		visibleOrgIDs := h.getVisibleOrgIDs(userIDStr)
		if len(visibleOrgIDs) == 0 {
			// 用户没有任何组织归属，返回空
			if tree {
				c.JSON(http.StatusOK, []models.OrganizationTreeNode{})
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "data": []models.OrganizationResponse{}})
			return
		}
		query = query.Where("id IN ?", visibleOrgIDs)
	}

	if search != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if parentID != "" {
		if parentID == "root" {
			query = query.Where("parent_id IS NULL")
		} else {
			query = query.Where("parent_id = ?", parentID)
		}
	}

	var orgs []models.Organization
	if err := query.Order("sort ASC, created_at ASC").Find(&orgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取组织列表失败"})
		return
	}

	if tree {
		// 构建树形结构
		treeNodes := h.buildTree(orgs)
		c.JSON(http.StatusOK, treeNodes)
		return
	}

	// 计算用户和设备数量
	var responses []models.OrganizationResponse
	for _, org := range orgs {
		resp := org.ToResponse()
		// 统计用户数
		var userCount int64
		h.db.Model(&models.UserOrganization{}).
			Where("organization_id = ?", org.ID).
			Count(&userCount)
		resp.UserCount = int(userCount)
		// 统计设备数
		var deviceCount int64
		h.db.Model(&models.Device{}).
			Where("org_id = ?", org.ID).
			Count(&deviceCount)
		resp.DeviceCount = int(deviceCount)
		responses = append(responses, resp)
	}

	c.JSON(http.StatusOK, responses)
}

// getVisibleOrgIDs 获取用户可见的组织ID列表
func (h *OrganizationHandler) getVisibleOrgIDs(userID string) []string {
	// 获取用户所属的组织
	var userOrgs []models.UserOrganization
	h.db.Where("user_id = ?", userID).Find(&userOrgs)

	if len(userOrgs) == 0 {
		return []string{}
	}

	visibleIDs := make(map[string]bool)
	for _, uo := range userOrgs {
		// 添加用户直属组织
		visibleIDs[uo.OrganizationID] = true
		// 添加所有子组织（递归）
		h.addDescendantIDs(uo.OrganizationID, visibleIDs)
	}

	result := make([]string, 0, len(visibleIDs))
	for id := range visibleIDs {
		result = append(result, id)
	}
	return result
}

// addDescendantIDs 递归添加子组织ID
func (h *OrganizationHandler) addDescendantIDs(orgID string, visibleIDs map[string]bool) {
	var children []models.Organization
	h.db.Where("parent_id = ? AND deleted_at IS NULL", orgID).Find(&children)

	for _, child := range children {
		visibleIDs[child.ID] = true
		h.addDescendantIDs(child.ID, visibleIDs)
	}
}

// buildTree 构建组织树
func (h *OrganizationHandler) buildTree(orgs []models.Organization) []*models.OrganizationTreeNode {
	nodeMap := make(map[string]*models.OrganizationTreeNode)
	var roots []*models.OrganizationTreeNode

	// 创建节点映射
	for _, org := range orgs {
		leaderName := ""
		if org.Leader != nil {
			leaderName = org.Leader.Name
		}

		var userCount int64
		h.db.Model(&models.UserOrganization{}).
			Where("organization_id = ?", org.ID).
			Count(&userCount)

		var deviceCount int64
		h.db.Model(&models.Device{}).
			Where("org_id = ?", org.ID).
			Count(&deviceCount)

		node := &models.OrganizationTreeNode{
			ID:          org.ID,
			Name:        org.Name,
			Code:        org.Code,
			ParentID:    org.ParentID,
			Type:        org.Type,
			Level:       org.Level,
			Sort:        org.Sort,
			Status:      org.Status,
			LeaderName:  leaderName,
			UserCount:   int(userCount),
			DeviceCount: int(deviceCount),
			Children:    []*models.OrganizationTreeNode{},
		}
		nodeMap[org.ID] = node
	}

	// 构建树关系
	for _, org := range orgs {
		node := nodeMap[org.ID]
		if org.ParentID == nil || *org.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[*org.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// 父节点不存在，作为根节点
			roots = append(roots, node)
		}
	}

	// 排序
	sortTree(roots)
	return roots
}

func sortTree(nodes []*models.OrganizationTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Sort != nodes[j].Sort {
			return nodes[i].Sort < nodes[j].Sort
		}
		return nodes[i].Name < nodes[j].Name
	})
	for _, node := range nodes {
		if len(node.Children) > 0 {
			sortTree(node.Children)
		}
	}
}

// Get 获取单个组织详情
func (h *OrganizationHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var org models.Organization
	if err := h.db.Preload("Leader").
		Preload("Parent").
		First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "组织不存在"})
		return
	}

	resp := org.ToResponse()

	// 统计用户数
	var userCount int64
	h.db.Model(&models.UserOrganization{}).
		Where("organization_id = ?", org.ID).
		Count(&userCount)
	resp.UserCount = int(userCount)

	// 统计设备数
	var deviceCount int64
	h.db.Model(&models.Device{}).
		Where("org_id = ?", org.ID).
		Count(&deviceCount)
	resp.DeviceCount = int(deviceCount)

	// 获取子组织
	var children []models.Organization
	h.db.Where("parent_id = ? AND deleted_at IS NULL", id).
		Order("sort ASC").
		Find(&children)

	for _, child := range children {
		childResp := child.ToResponse()
		resp.Children = append(resp.Children, childResp)
	}

	c.JSON(http.StatusOK, resp)
}

// CreateOrganizationRequest 创建组织请求
type CreateOrganizationRequest struct {
	Name        string                  `json:"name" binding:"required"`
	Code        string                  `json:"code" binding:"required"`
	ParentID    *string                 `json:"parent_id"`
	Type        models.OrganizationType `json:"type"`
	Sort        int                     `json:"sort"`
	LeaderID    *string                 `json:"leader_id"`
	Description *string                 `json:"description"`
}

// Create 创建组织
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 检查编码是否重复
	var existing models.Organization
	if err := h.db.Where("code = ? AND deleted_at IS NULL", req.Code).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "组织编码已存在"})
		return
	}

	// 计算层级和路径
	level := 1
	path := ""
	if req.ParentID != nil && *req.ParentID != "" {
		var parent models.Organization
		if err := h.db.First(&parent, "id = ?", *req.ParentID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "父组织不存在"})
			return
		}
		level = parent.Level + 1
		if level > 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "最多支持3级组织架构"})
			return
		}
		path = parent.Path + "/" + req.Code
	} else {
		path = req.Code
	}

	// 默认类型
	if req.Type == "" {
		switch level {
		case 1:
			req.Type = models.OrgTypeCompany
		case 2:
			req.Type = models.OrgTypeBranch
		case 3:
			req.Type = models.OrgTypeDept
		}
	}

	org := models.Organization{
		Name:        req.Name,
		Code:        req.Code,
		ParentID:    req.ParentID,
		Type:        req.Type,
		Level:       level,
		Path:        path,
		Sort:        req.Sort,
		Status:      "active",
		LeaderID:    req.LeaderID,
		Description: req.Description,
	}

	if err := h.db.Create(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建组织失败"})
		return
	}

	c.JSON(http.StatusCreated, org.ToResponse())
}

// UpdateOrganizationRequest 更新组织请求
type UpdateOrganizationRequest struct {
	Name        *string                  `json:"name"`
	Code        *string                  `json:"code"`
	Type        *models.OrganizationType `json:"type"`
	Sort        *int                     `json:"sort"`
	Status      *string                  `json:"status"`
	LeaderID    *string                  `json:"leader_id"`
	Description *string                  `json:"description"`
}

// Update 更新组织
func (h *OrganizationHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "组织不存在"})
		return
	}

	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 检查编码是否重复
	if req.Code != nil && *req.Code != org.Code {
		var existing models.Organization
		if err := h.db.Where("code = ? AND id != ? AND deleted_at IS NULL", *req.Code, id).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "组织编码已存在"})
			return
		}
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Code != nil {
		updates["code"] = *req.Code
		// 更新路径
		if org.ParentID != nil && *org.ParentID != "" {
			var parent models.Organization
			h.db.First(&parent, "id = ?", *org.ParentID)
			updates["path"] = parent.Path + "/" + *req.Code
		} else {
			updates["path"] = *req.Code
		}
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.LeaderID != nil {
		updates["leader_id"] = *req.LeaderID
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if err := h.db.Model(&org).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新组织失败"})
		return
	}

	h.db.First(&org, "id = ?", id)
	c.JSON(http.StatusOK, org.ToResponse())
}

// Delete 删除组织
func (h *OrganizationHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "组织不存在"})
		return
	}

	// 检查是否有子组织
	var childCount int64
	h.db.Model(&models.Organization{}).Where("parent_id = ? AND deleted_at IS NULL", id).Count(&childCount)
	if childCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先删除子组织"})
		return
	}

	// 检查是否有用户
	var userCount int64
	h.db.Model(&models.UserOrganization{}).Where("organization_id = ?", id).Count(&userCount)
	if userCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先移除该组织下的用户"})
		return
	}

	// 软删除
	if err := h.db.Delete(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// MoveRequest 移动组织请求
type MoveRequest struct {
	ParentID *string `json:"parent_id"` // 新的父组织ID，null表示移到根级
	Sort     *int    `json:"sort"`      // 新的排序号
}

// Move 移动组织（调整层级或排序）
func (h *OrganizationHandler) Move(c *gin.Context) {
	id := c.Param("id")

	var org models.Organization
	if err := h.db.First(&org, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "组织不存在"})
		return
	}

	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 检查是否循环引用（不能将自己移到自己的子节点下）
	if req.ParentID != nil && *req.ParentID != "" {
		if *req.ParentID == id {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能将组织移动到自己下面"})
			return
		}
		// 检查新父节点是否是当前节点的子节点
		if h.isDescendant(id, *req.ParentID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能将组织移动到自己的子组织下"})
			return
		}
	}

	// 计算新层级
	newLevel := 1
	newPath := org.Code
	if req.ParentID != nil && *req.ParentID != "" {
		var parent models.Organization
		if err := h.db.First(&parent, "id = ?", *req.ParentID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "父组织不存在"})
			return
		}
		newLevel = parent.Level + 1
		if newLevel > 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "最多支持3级组织架构"})
			return
		}
		newPath = parent.Path + "/" + org.Code
	}

	// 检查子组织层级是否超限
	maxDescendantLevel := h.getMaxDescendantLevel(id)
	descendantDepth := maxDescendantLevel - org.Level
	if newLevel+descendantDepth > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "移动后子组织层级将超过3级限制"})
		return
	}

	updates := map[string]interface{}{
		"parent_id": req.ParentID,
		"level":     newLevel,
		"path":      newPath,
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}

	if err := h.db.Model(&org).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移动组织失败"})
		return
	}

	// 更新子组织的层级和路径
	h.updateDescendantsPath(id, newPath, newLevel)

	h.db.First(&org, "id = ?", id)
	c.JSON(http.StatusOK, org.ToResponse())
}

// isDescendant 检查targetID是否是orgID的子节点
func (h *OrganizationHandler) isDescendant(orgID, targetID string) bool {
	var target models.Organization
	if err := h.db.First(&target, "id = ?", targetID).Error; err != nil {
		return false
	}
	if target.ParentID == nil {
		return false
	}
	if *target.ParentID == orgID {
		return true
	}
	return h.isDescendant(orgID, *target.ParentID)
}

// getMaxDescendantLevel 获取最大子节点层级
func (h *OrganizationHandler) getMaxDescendantLevel(orgID string) int {
	var maxLevel int
	h.db.Model(&models.Organization{}).
		Where("path LIKE ?", "%"+orgID+"%").
		Select("MAX(level)").
		Row().Scan(&maxLevel)
	return maxLevel
}

// updateDescendantsPath 更新子组织的路径和层级
func (h *OrganizationHandler) updateDescendantsPath(parentID, parentPath string, parentLevel int) {
	var children []models.Organization
	h.db.Where("parent_id = ?", parentID).Find(&children)

	for _, child := range children {
		newPath := parentPath + "/" + child.Code
		newLevel := parentLevel + 1
		h.db.Model(&child).Updates(map[string]interface{}{
			"path":  newPath,
			"level": newLevel,
		})
		h.updateDescendantsPath(child.ID, newPath, newLevel)
	}
}

// GetUsers 获取组织下的用户列表
func (h *OrganizationHandler) GetUsers(c *gin.Context) {
	orgID := c.Param("id")
	includeSub := c.Query("include_sub") == "true" // 是否包含子组织

	// 获取组织ID列表
	orgIDs := []string{orgID}
	if includeSub {
		orgIDs = h.getDescendantIDs(orgID)
		orgIDs = append(orgIDs, orgID)
	}

	var userOrgs []models.UserOrganization
	if err := h.db.Preload("User").Preload("Organization").
		Where("organization_id IN ?", orgIDs).
		Find(&userOrgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	var responses []map[string]interface{}
	for _, uo := range userOrgs {
		if uo.User != nil {
			resp := map[string]interface{}{
				"id":                uo.ID,
				"user_id":           uo.UserID,
				"organization_id":   uo.OrganizationID,
				"organization_name": "",
				"is_primary":        uo.IsPrimary,
				"position":          uo.Position,
				"joined_at":         uo.JoinedAt,
				"user": map[string]interface{}{
					"id":     uo.User.ID,
					"name":   uo.User.Name,
					"email":  uo.User.Email,
					"role":   uo.User.Role,
					"status": uo.User.Status,
					"avatar": uo.User.Avatar,
				},
			}
			if uo.Organization != nil {
				resp["organization_name"] = uo.Organization.Name
			}
			responses = append(responses, resp)
		}
	}

	c.JSON(http.StatusOK, responses)
}

// getDescendantIDs 获取所有子组织ID
func (h *OrganizationHandler) getDescendantIDs(orgID string) []string {
	var ids []string
	var children []models.Organization
	h.db.Where("parent_id = ? AND deleted_at IS NULL", orgID).Find(&children)

	for _, child := range children {
		ids = append(ids, child.ID)
		ids = append(ids, h.getDescendantIDs(child.ID)...)
	}
	return ids
}

// AddUserRequest 添加用户到组织请求
type AddUserRequest struct {
	UserID    string `json:"user_id" binding:"required"`
	IsPrimary bool   `json:"is_primary"`
	Position  string `json:"position"`
}

// AddUser 添加用户到组织
func (h *OrganizationHandler) AddUser(c *gin.Context) {
	orgID := c.Param("id")

	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 检查组织是否存在
	var org models.Organization
	if err := h.db.First(&org, "id = ?", orgID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "组织不存在"})
		return
	}

	// 检查用户是否存在
	var user models.User
	if err := h.db.First(&user, "id = ?", req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 检查是否已加入该组织
	var existing models.UserOrganization
	if err := h.db.Where("user_id = ? AND organization_id = ?", req.UserID, orgID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户已在该组织中"})
		return
	}

	// 如果设置为主部门，需要取消其他主部门
	if req.IsPrimary {
		h.db.Model(&models.UserOrganization{}).
			Where("user_id = ? AND is_primary = ?", req.UserID, true).
			Update("is_primary", false)
	}

	userOrg := models.UserOrganization{
		UserID:         req.UserID,
		OrganizationID: orgID,
		IsPrimary:      req.IsPrimary,
		Position:       req.Position,
		JoinedAt:       time.Now(),
	}

	if err := h.db.Create(&userOrg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加用户失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "添加成功"})
}

// RemoveUser 从组织移除用户
func (h *OrganizationHandler) RemoveUser(c *gin.Context) {
	orgID := c.Param("id")
	userID := c.Param("user_id")

	result := h.db.Where("organization_id = ? AND user_id = ?", orgID, userID).
		Delete(&models.UserOrganization{})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不在该组织中"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}

// UpdateUserOrgRequest 更新用户组织关系请求
type UpdateUserOrgRequest struct {
	IsPrimary *bool   `json:"is_primary"`
	Position  *string `json:"position"`
}

// UpdateUserOrg 更新用户在组织中的信息
func (h *OrganizationHandler) UpdateUserOrg(c *gin.Context) {
	orgID := c.Param("id")
	userID := c.Param("user_id")

	var userOrg models.UserOrganization
	if err := h.db.Where("organization_id = ? AND user_id = ?", orgID, userID).First(&userOrg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不在该组织中"})
		return
	}

	var req UpdateUserOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	updates := make(map[string]interface{})
	if req.IsPrimary != nil {
		updates["is_primary"] = *req.IsPrimary
		// 如果设置为主部门，取消其他主部门
		if *req.IsPrimary {
			h.db.Model(&models.UserOrganization{}).
				Where("user_id = ? AND organization_id != ? AND is_primary = ?", userID, orgID, true).
				Update("is_primary", false)
		}
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}

	if err := h.db.Model(&userOrg).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// GetDevices 获取组织下的设备列表
func (h *OrganizationHandler) GetDevices(c *gin.Context) {
	orgID := c.Param("id")
	includeSub := c.Query("include_sub") == "true"

	// 获取组织ID列表
	orgIDs := []string{orgID}
	if includeSub {
		orgIDs = h.getDescendantIDs(orgID)
		orgIDs = append(orgIDs, orgID)
	}

	var devices []models.Device
	if err := h.db.Where("org_id IN ?", orgIDs).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设备列表失败"})
		return
	}

	c.JSON(http.StatusOK, devices)
}

// GetUserOrganizations 获取用户所属的所有组织
func (h *OrganizationHandler) GetUserOrganizations(c *gin.Context) {
	userID := c.Param("id") // 从users/:id/organizations路由获取

	var userOrgs []models.UserOrganization
	if err := h.db.Preload("Organization").
		Where("user_id = ?", userID).
		Order("is_primary DESC, joined_at ASC").
		Find(&userOrgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取组织列表失败"})
		return
	}

	var responses []models.UserOrganizationResponse
	for _, uo := range userOrgs {
		responses = append(responses, uo.ToResponse())
	}

	c.JSON(http.StatusOK, responses)
}

// 为设备添加org_id字段的helper函数
func (h *OrganizationHandler) addOrgFieldToDevice() {
	// 检查字段是否已存在
	if !h.db.Migrator().HasColumn(&models.Device{}, "org_id") {
		h.db.Migrator().AddColumn(&models.Device{}, "org_id")
	}
}

// 初始化示例组织数据
func SeedOrganizations(db *gorm.DB) error {
	var count int64
	db.Model(&models.Organization{}).Count(&count)
	if count > 0 {
		return nil
	}

	// 创建根组织
	hq := models.Organization{
		ID:     "org-hq",
		Name:   "货运跟踪集团总部",
		Code:   "HQ",
		Type:   models.OrgTypeGroup,
		Level:  1,
		Path:   "HQ",
		Sort:   1,
		Status: "active",
	}
	db.Create(&hq)

	// 华东分公司
	eastBranch := models.Organization{
		ID:       "org-east",
		Name:     "华东分公司",
		Code:     "EAST",
		ParentID: &hq.ID,
		Type:     models.OrgTypeBranch,
		Level:    2,
		Path:     "HQ/EAST",
		Sort:     1,
		Status:   "active",
	}
	db.Create(&eastBranch)

	// 华南分公司
	southBranch := models.Organization{
		ID:       "org-south",
		Name:     "华南分公司",
		Code:     "SOUTH",
		ParentID: &hq.ID,
		Type:     models.OrgTypeBranch,
		Level:    2,
		Path:     "HQ/SOUTH",
		Sort:     2,
		Status:   "active",
	}
	db.Create(&southBranch)

	// 华东运营部
	eastOps := models.Organization{
		ID:       "org-east-ops",
		Name:     "运营部",
		Code:     "EAST-OPS",
		ParentID: &eastBranch.ID,
		Type:     models.OrgTypeDept,
		Level:    3,
		Path:     "HQ/EAST/EAST-OPS",
		Sort:     1,
		Status:   "active",
	}
	db.Create(&eastOps)

	// 华东技术部
	eastTech := models.Organization{
		ID:       "org-east-tech",
		Name:     "技术部",
		Code:     "EAST-TECH",
		ParentID: &eastBranch.ID,
		Type:     models.OrgTypeDept,
		Level:    3,
		Path:     "HQ/EAST/EAST-TECH",
		Sort:     2,
		Status:   "active",
	}
	db.Create(&eastTech)

	// 确保管理员已加入总部
	var adminOrgCount int64
	db.Model(&models.UserOrganization{}).Where("user_id = ? AND organization_id = ?", "user-admin", "org-hq").Count(&adminOrgCount)
	if adminOrgCount == 0 {
		db.Create(&models.UserOrganization{
			UserID:         "user-admin",
			OrganizationID: "org-hq",
			IsPrimary:      true,
			Position:       "超级管理员",
			JoinedAt:       time.Now(),
		})
		fmt.Println("✅ Admin added to HQ organization")
	}

	fmt.Println("✅ Organization seed data created")
	return nil
}
