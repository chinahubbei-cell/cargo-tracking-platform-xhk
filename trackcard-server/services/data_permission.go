package services

import (
	"gorm.io/gorm"

	"trackcard-server/models"
)

// DataPermissionService 数据权限服务
type DataPermissionService struct {
	db *gorm.DB
}

var DataPermission *DataPermissionService

// InitDataPermission 初始化数据权限服务
func InitDataPermission(db *gorm.DB) {
	DataPermission = &DataPermissionService{db: db}
}

// GetUserOrgIDs 获取用户所属的所有组织ID
func (s *DataPermissionService) GetUserOrgIDs(userID string) []string {
	var userOrgs []models.UserOrganization
	s.db.Where("user_id = ?", userID).Find(&userOrgs)

	orgIDs := make([]string, len(userOrgs))
	for i, uo := range userOrgs {
		orgIDs[i] = uo.OrganizationID
	}
	return orgIDs
}

// GetUserPrimaryOrgID 获取用户的主组织ID
func (s *DataPermissionService) GetUserPrimaryOrgID(userID string) string {
	var userOrg models.UserOrganization
	if err := s.db.Where("user_id = ? AND is_primary = ?", userID, true).First(&userOrg).Error; err != nil {
		// 如果没有主组织，返回第一个组织
		if err := s.db.Where("user_id = ?", userID).First(&userOrg).Error; err != nil {
			return ""
		}
	}
	return userOrg.OrganizationID
}

// GetOrgAndDescendantIDs 获取组织及其所有子组织的ID
func (s *DataPermissionService) GetOrgAndDescendantIDs(orgID string) []string {
	ids := []string{orgID}
	ids = append(ids, s.getDescendantIDs(orgID)...)
	return ids
}

// getDescendantIDs 获取所有子组织ID（递归）
func (s *DataPermissionService) getDescendantIDs(orgID string) []string {
	var ids []string
	var children []models.Organization
	s.db.Where("parent_id = ? AND deleted_at IS NULL", orgID).Find(&children)

	for _, child := range children {
		ids = append(ids, child.ID)
		ids = append(ids, s.getDescendantIDs(child.ID)...)
	}
	return ids
}

// GetVisibleOrgIDs 获取用户可见的所有组织ID
// 规则：用户所属组织 + 这些组织的所有子组织
func (s *DataPermissionService) GetVisibleOrgIDs(userID string) []string {
	userOrgIDs := s.GetUserOrgIDs(userID)
	if len(userOrgIDs) == 0 {
		return []string{}
	}

	visibleIDs := make(map[string]bool)
	for _, orgID := range userOrgIDs {
		// 添加用户直属组织
		visibleIDs[orgID] = true
		// 添加所有子组织
		for _, descendantID := range s.getDescendantIDs(orgID) {
			visibleIDs[descendantID] = true
		}
	}

	result := make([]string, 0, len(visibleIDs))
	for id := range visibleIDs {
		result = append(result, id)
	}
	return result
}

// IsRootOrg 检查组织是否是根组织（总部）
func (s *DataPermissionService) IsRootOrg(orgID string) bool {
	var org models.Organization
	if err := s.db.First(&org, "id = ?", orgID).Error; err != nil {
		return false
	}
	return org.ParentID == nil || *org.ParentID == ""
}

// CanAccessAllData 检查用户是否可以访问所有数据
// 规则：如果用户属于根组织（总部），则可以看到所有数据
func (s *DataPermissionService) CanAccessAllData(userID string) bool {
	userOrgIDs := s.GetUserOrgIDs(userID)
	for _, orgID := range userOrgIDs {
		if s.IsRootOrg(orgID) {
			return true
		}
	}
	return false
}

// FilterQueryByOrg 根据用户权限过滤查询
// 返回添加了组织条件的查询，以及是否有权限访问（无组织关联时返回false）
func (s *DataPermissionService) FilterQueryByOrg(query *gorm.DB, userID string, orgField string) (*gorm.DB, bool) {
	// 检查用户是否可以访问所有数据
	if s.CanAccessAllData(userID) {
		return query, true
	}

	// 获取用户可见的组织ID列表
	visibleOrgIDs := s.GetVisibleOrgIDs(userID)
	if len(visibleOrgIDs) == 0 {
		// 用户没有组织关联，无法查看任何数据（除非数据没有组织归属）
		return query.Where(orgField+" IS NULL OR "+orgField+" = ?", ""), true
	}

	// 添加组织过滤条件：属于可见组织或没有组织归属
	return query.Where(orgField+" IN ? OR "+orgField+" IS NULL OR "+orgField+" = ?", visibleOrgIDs, ""), true
}

// ApplyOrgFilter 应用组织筛选（用于API查询参数）
func (s *DataPermissionService) ApplyOrgFilter(query *gorm.DB, userID string, requestedOrgID string, orgField string) *gorm.DB {
	if requestedOrgID != "" {
		// 用户请求特定组织的数据
		// 检查用户是否有权限访问该组织
		visibleOrgIDs := s.GetVisibleOrgIDs(userID)
		hasAccess := false
		for _, id := range visibleOrgIDs {
			if id == requestedOrgID {
				hasAccess = true
				break
			}
		}

		if hasAccess || s.CanAccessAllData(userID) {
			// 获取请求组织及其子组织
			orgIDs := s.GetOrgAndDescendantIDs(requestedOrgID)
			return query.Where(orgField+" IN ?", orgIDs)
		} else {
			// 无权限，返回空结果
			return query.Where("1 = 0")
		}
	}

	// 没有指定组织，按用户权限过滤
	filteredQuery, _ := s.FilterQueryByOrg(query, userID, orgField)
	return filteredQuery
}
