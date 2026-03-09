package handlers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/utils"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) List(c *gin.Context) {
	status := c.Query("status")
	role := c.Query("role")
	search := c.Query("search")

	query := h.db.Model(&models.User{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if search != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var users []models.User
	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 构建包含组织信息的响应
	var result []map[string]interface{}
	for _, u := range users {
		userResp := u.ToResponse()
		userData := map[string]interface{}{
			"id":                 userResp.ID,
			"email":              userResp.Email,
			"phone_country_code": userResp.PhoneCountryCode,
			"phone_number":       userResp.PhoneNumber,
			"name":               userResp.Name,
			"role":               userResp.Role,
			"permissions":        userResp.Permissions,
			"status":             userResp.Status,
			"avatar":             userResp.Avatar,
			"last_login":         userResp.LastLogin,
			"created_at":         userResp.CreatedAt,
			"updated_at":         userResp.UpdatedAt,
		}

		// 获取用户所属组织
		var userOrgs []models.UserOrganization
		h.db.Preload("Organization").Where("user_id = ?", u.ID).Find(&userOrgs)

		var orgs []map[string]interface{}
		var primaryOrgName string
		for _, uo := range userOrgs {
			if uo.Organization != nil {
				orgInfo := map[string]interface{}{
					"id":         uo.OrganizationID,
					"name":       uo.Organization.Name,
					"is_primary": uo.IsPrimary,
					"position":   uo.Position,
				}
				orgs = append(orgs, orgInfo)
				if uo.IsPrimary {
					primaryOrgName = uo.Organization.Name
				}
			}
		}
		userData["organizations"] = orgs
		userData["primary_org_name"] = primaryOrgName

		result = append(result, userData)
	}

	utils.SuccessResponse(c, result)
}

func (h *UserHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	utils.SuccessResponse(c, user.ToResponse())
}

type CreateUserRequest struct {
	Email            string  `json:"email" binding:"required,email"`
	Password         string  `json:"password" binding:"required,min=6"`
	Name             string  `json:"name" binding:"required"`
	Role             string  `json:"role"`
	Avatar           *string `json:"avatar"`
	PhoneCountryCode string  `json:"phone_country_code"`
	PhoneNumber      *string `json:"phone_number"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的用户信息")
		return
	}

	// 检查邮箱是否已存在
	var count int64
	h.db.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		utils.BadRequest(c, "该邮箱已被注册")
		return
	}

	if req.PhoneNumber != nil && *req.PhoneNumber != "" {
		cc, phone := normalizePhone(req.PhoneCountryCode, *req.PhoneNumber)
		req.PhoneCountryCode = cc
		req.PhoneNumber = &phone

		var phoneCount int64
		h.db.Model(&models.User{}).Where("phone_country_code = ? AND phone_number = ?", cc, phone).Count(&phoneCount)
		if phoneCount > 0 {
			utils.BadRequest(c, "该手机号已被注册")
			return
		}
	} else {
		if req.PhoneCountryCode == "" {
			req.PhoneCountryCode = "+86"
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}

	user := models.User{
		Email:            req.Email,
		Password:         string(hashedPassword),
		Name:             req.Name,
		Role:             req.Role,
		Avatar:           req.Avatar,
		PhoneCountryCode: req.PhoneCountryCode,
		PhoneNumber:      req.PhoneNumber,
		Status:           "active",
	}

	if user.Role == "" {
		user.Role = "viewer"
	}

	if err := h.db.Create(&user).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.CreatedResponse(c, user.ToResponse())
}

type UpdateUserRequest struct {
	Name             *string `json:"name"`
	Role             *string `json:"role"`
	Status           *string `json:"status"`
	Avatar           *string `json:"avatar"`
	Permissions      *string `json:"permissions"`
	PhoneCountryCode *string `json:"phone_country_code"`
	PhoneNumber      *string `json:"phone_number"`
}

func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.BadRequest(c, "无效的请求数据")
		return
	}

	updates := make(map[string]interface{})
	if v, ok := payload["name"]; ok {
		if name, ok2 := v.(string); ok2 {
			updates["name"] = name
		}
	}
	if v, ok := payload["role"]; ok {
		if role, ok2 := v.(string); ok2 {
			updates["role"] = role
		}
	}
	if v, ok := payload["status"]; ok {
		if status, ok2 := v.(string); ok2 {
			updates["status"] = status
		}
	}
	if v, ok := payload["avatar"]; ok {
		if v == nil {
			updates["avatar"] = nil
		} else if avatar, ok2 := v.(string); ok2 {
			updates["avatar"] = avatar
		}
	}
	if v, ok := payload["permissions"]; ok {
		if perms, ok2 := v.(string); ok2 {
			updates["permissions"] = perms
		}
	}
	hasPhoneUpdate := false
	newPhone := user.PhoneNumber
	newCc := user.PhoneCountryCode

	if v, ok := payload["phone_country_code"]; ok {
		if cc, ok2 := v.(string); ok2 {
			newCc = cc
			hasPhoneUpdate = true
		}
	}
	if v, ok := payload["phone_number"]; ok {
		if v == nil {
			updates["phone_number"] = nil
			newPhone = nil
			hasPhoneUpdate = false // no need to check conflict if removing phone
		} else if phone, ok2 := v.(string); ok2 {
			newPhone = &phone
			hasPhoneUpdate = true
		}
	}

	if hasPhoneUpdate && newPhone != nil && *newPhone != "" {
		normCc, normPhone := normalizePhone(newCc, *newPhone)
		updates["phone_country_code"] = normCc
		updates["phone_number"] = normPhone

		var phoneCount int64
		h.db.Model(&models.User{}).Where("phone_country_code = ? AND phone_number = ? AND id != ?", normCc, normPhone, id).Count(&phoneCount)
		if phoneCount > 0 {
			utils.BadRequest(c, "该手机号已被其他账号注册")
			return
		}
	} else if hasPhoneUpdate && (newPhone == nil || *newPhone == "") {
		updates["phone_country_code"] = newCc
	}

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.First(&user, "id = ?", id)
	utils.SuccessResponse(c, user.ToResponse())
}

func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// 不允许删除自己
	currentUserID, _ := c.Get("user_id")
	if currentUserID == id {
		utils.BadRequest(c, "不能删除自己的账户")
		return
	}

	if err := h.db.Delete(&models.User{}, "id = ?", id).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"success": true})
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效密码")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}

	h.db.Model(&user).Update("password", string(hashedPassword))
	utils.SuccessResponse(c, gin.H{"message": "密码重置成功"})
}
