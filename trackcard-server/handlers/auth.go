package handlers

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"trackcard-server/config"
	"trackcard-server/models"
	"trackcard-server/utils"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	Token string              `json:"token"`
	User  models.UserResponse `json:"user"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的登录凭证")
		return
	}

	// 支持 "admin" 简写登录
	email := strings.ToLower(req.Email)
	if email == "admin" {
		email = "admin@trackcard.com"
	}

	var user models.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		log.Printf("[Login Debug] User not found: %s", email)
		utils.Unauthorized(c, "用户名或密码错误")
		return
	}

	if user.Status != "active" {
		log.Printf("[Login Debug] User inactive: %s", email)
		utils.Forbidden(c, "账户已被禁用")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Printf("[Login Debug] Password mismatch for %s. Hash: %s... Input: %s", email, user.Password[:10], req.Password)
		utils.Unauthorized(c, "邮箱或密码错误")
		return
	}

	// 获取主组织ID
	var primaryOrgID string
	var userOrg models.UserOrganization
	if err := h.db.Where("user_id = ? AND is_primary = ?", user.ID, true).First(&userOrg).Error; err == nil {
		primaryOrgID = userOrg.OrganizationID
	}

	token, err := utils.GenerateToken(user.ID, user.Email, user.Role, primaryOrgID)
	if err != nil {
		utils.InternalError(c, "生成令牌失败")
		return
	}

	// 更新最后登录时间
	now := time.Now()
	h.db.Model(&user).Update("last_login", now)

	// 获取用户所属组织
	var userOrgs []models.UserOrganization
	h.db.Preload("Organization").Where("user_id = ?", user.ID).Find(&userOrgs)

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

	// 构建包含组织信息的用户响应
	userResp := user.ToResponse()
	userWithOrg := map[string]interface{}{
		"id":               userResp.ID,
		"email":            userResp.Email,
		"name":             userResp.Name,
		"role":             userResp.Role,
		"permissions":      userResp.Permissions,
		"status":           userResp.Status,
		"avatar":           userResp.Avatar,
		"last_login":       now,
		"created_at":       userResp.CreatedAt,
		"updated_at":       userResp.UpdatedAt,
		"organizations":    orgs,
		"primary_org_name": primaryOrgName,
	}

	utils.SuccessResponse(c, gin.H{
		"token": token,
		"user":  userWithOrg,
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	// 构建包含组织信息的响应
	userResp := user.ToResponse()
	result := map[string]interface{}{
		"id":          userResp.ID,
		"email":       userResp.Email,
		"name":        userResp.Name,
		"role":        userResp.Role,
		"permissions": userResp.Permissions,
		"status":      userResp.Status,
		"avatar":      userResp.Avatar,
		"last_login":  userResp.LastLogin,
		"created_at":  userResp.CreatedAt,
		"updated_at":  userResp.UpdatedAt,
	}

	// 获取用户所属组织
	var userOrgs []models.UserOrganization
	h.db.Preload("Organization").Where("user_id = ?", user.ID).Find(&userOrgs)

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
	result["organizations"] = orgs
	result["primary_org_name"] = primaryOrgName

	utils.SuccessResponse(c, result)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的密码")
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		utils.BadRequest(c, "原密码错误")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}

	h.db.Model(&user).Update("password", string(hashedPassword))
	utils.SuccessResponse(c, gin.H{"message": "密码修改成功"})
}

// SeedAdmin 创建默认管理员用户
func SeedAdmin(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Where("email = ?", "admin@trackcard.com").Count(&count)
	if count > 0 {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(config.AppConfig.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := models.User{
		ID:       "user-admin",
		Email:    "admin@trackcard.com",
		Password: string(hashedPassword),
		Name:     "张伟",
		Role:     "admin",
		Status:   "active",
	}

	return db.Create(&admin).Error
}

// SeedConfig 创建默认系统配置
func SeedConfig(db *gorm.DB) error {
	configs := []models.SystemConfig{
		{Key: "geofence_radius", Value: "1000"},
		{Key: "port_geofence_radius", Value: "5000"},
		{Key: "kuaihuoyun_cid", Value: config.AppConfig.KuaihuoyunCID},
		{Key: "kuaihuoyun_secret_key", Value: config.AppConfig.KuaihuoyunSecretKey},
	}

	for _, cfg := range configs {
		var count int64
		db.Model(&models.SystemConfig{}).Where("key = ?", cfg.Key).Count(&count)
		if count == 0 {
			db.Create(&cfg)
		}
	}

	return nil
}
