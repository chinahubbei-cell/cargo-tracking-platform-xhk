package handlers

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"trackcard-server/config"
	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

// ===== S-3: 登录速率限制器（防暴力破解） =====

type loginAttempt struct {
	count    int
	firstAt  time.Time
	lockedAt time.Time
}

var (
	loginAttempts = make(map[string]*loginAttempt)
	loginMu       sync.Mutex
)

const (
	maxLoginAttempts     = 5                // 最大尝试次数
	loginWindowDuration  = 1 * time.Minute  // 窗口时间
	loginLockoutDuration = 15 * time.Minute // 锁定时间
)

// checkLoginRateLimit 检查登录速率限制，返回 true 表示被限制
func checkLoginRateLimit(key string) bool {
	loginMu.Lock()
	defer loginMu.Unlock()

	now := time.Now()
	attempt, exists := loginAttempts[key]

	if !exists {
		loginAttempts[key] = &loginAttempt{count: 0, firstAt: now}
		return false
	}

	// 如果当前处于锁定状态
	if !attempt.lockedAt.IsZero() && now.Before(attempt.lockedAt.Add(loginLockoutDuration)) {
		return true
	}

	// 窗口过期，重置
	if now.After(attempt.firstAt.Add(loginWindowDuration)) {
		attempt.count = 0
		attempt.firstAt = now
		attempt.lockedAt = time.Time{}
	}

	return false
}

// recordLoginFailure 记录登录失败
func recordLoginFailure(key string) {
	loginMu.Lock()
	defer loginMu.Unlock()

	attempt, exists := loginAttempts[key]
	if !exists {
		loginAttempts[key] = &loginAttempt{count: 1, firstAt: time.Now()}
		return
	}
	attempt.count++
	if attempt.count >= maxLoginAttempts {
		attempt.lockedAt = time.Now()
		log.Printf("[Security] Account locked due to %d failed attempts: %s", attempt.count, key)
	}
}

// clearLoginAttempts 登录成功后清除记录
func clearLoginAttempts(key string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginAttempts, key)
}

// ===== M-9: 组织服务状态校验（消除重复） =====

func checkOrgServiceValid(org *models.Organization) (string, bool) {
	if org == nil {
		return "", true
	}
	if org.ServiceStatus != "active" && org.ServiceStatus != "trial" && org.ServiceStatus != "" {
		return "组织服务已被禁用或过期，请联系管理员", false
	}
	if org.ServiceEnd != nil && !org.ServiceEnd.IsZero() && time.Now().After(*org.ServiceEnd) {
		return "组织服务已到期，请联系管理员续费", false
	}
	return "", true
}

// ===== G-6: 登录响应构建（消除重复） =====

func buildLoginResponse(db *gorm.DB, user *models.User, loginTime time.Time) ([]map[string]interface{}, map[string]interface{}) {
	var userOrgs []models.UserOrganization
	db.Preload("Organization").Where("user_id = ?", user.ID).Find(&userOrgs)

	var orgs []map[string]interface{}
	var primaryOrgName string
	for _, uo := range userOrgs {
		if uo.Organization != nil {
			orgs = append(orgs, map[string]interface{}{
				"id":         uo.OrganizationID,
				"name":       uo.Organization.Name,
				"is_primary": uo.IsPrimary,
				"position":   uo.Position,
			})
			if uo.IsPrimary {
				primaryOrgName = uo.Organization.Name
			}
		}
	}

	userResp := user.ToResponse()
	userWithOrg := map[string]interface{}{
		"id":                 userResp.ID,
		"email":              userResp.Email,
		"phone_country_code": userResp.PhoneCountryCode,
		"phone_number":       userResp.PhoneNumber,
		"name":               userResp.Name,
		"role":               userResp.Role,
		"permissions":        userResp.Permissions,
		"status":             userResp.Status,
		"avatar":             userResp.Avatar,
		"last_login":         loginTime,
		"created_at":         userResp.CreatedAt,
		"updated_at":         userResp.UpdatedAt,
		"organizations":      orgs,
		"primary_org_name":   primaryOrgName,
	}

	return orgs, userWithOrg
}

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

type PhonePasswordLoginRequest struct {
	PhoneCountryCode string `json:"phone_country_code"`
	PhoneNumber      string `json:"phone_number" binding:"required"`
	Password         string `json:"password" binding:"required,min=6"`
}

type SendSMSCodeRequest struct {
	PhoneCountryCode string `json:"phone_country_code"`
	PhoneNumber      string `json:"phone_number" binding:"required"`
	Scene            string `json:"scene" binding:"required,oneof=login reset_password"`
}

type SMSLoginRequest struct {
	PhoneCountryCode string `json:"phone_country_code"`
	PhoneNumber      string `json:"phone_number" binding:"required"`
	Code             string `json:"code" binding:"required,len=6"`
}

type SelectOrgRequest struct {
	OrgID string `json:"org_id" binding:"required"`
}

type ResetPasswordBySMSRequest struct {
	PhoneCountryCode string `json:"phone_country_code"`
	PhoneNumber      string `json:"phone_number" binding:"required"`
	Code             string `json:"code" binding:"required,len=6"`
	NewPassword      string `json:"new_password" binding:"required,min=6"`
}

func normalizePhone(countryCode, phone string) (string, string) {
	cc := strings.TrimSpace(countryCode)
	if cc == "" {
		cc = "+86"
	}
	p := strings.TrimSpace(phone)
	p = strings.TrimPrefix(p, cc)
	p = strings.TrimPrefix(p, "+86")
	p = strings.TrimPrefix(p, "86")
	return cc, p
}

func codeHash(phoneCountryCode, phoneNumber, code, scene string) string {
	raw := fmt.Sprintf("%s|%s|%s|%s", phoneCountryCode, phoneNumber, code, scene)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:])
}

func (h *AuthHandler) checkCode(phoneCountryCode, phoneNumber, code, scene string) error {
	var rec models.AuthVerificationCode
	err := h.db.Where("scene = ? AND phone_country_code = ? AND phone_number = ? AND used_at IS NULL", scene, phoneCountryCode, phoneNumber).Order("created_at DESC").First(&rec).Error
	if err != nil {
		return fmt.Errorf("验证码无效")
	}
	if time.Now().After(rec.ExpiresAt) {
		return fmt.Errorf("验证码已过期")
	}
	if rec.AttemptCount >= 5 {
		return fmt.Errorf("验证码尝试次数过多")
	}
	if rec.CodeHash != codeHash(phoneCountryCode, phoneNumber, code, scene) {
		h.db.Model(&rec).Update("attempt_count", rec.AttemptCount+1)
		return fmt.Errorf("验证码错误")
	}
	now := time.Now()
	h.db.Model(&rec).Updates(map[string]interface{}{"used_at": now, "attempt_count": rec.AttemptCount + 1})
	return nil
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

	// S-3: 暴力破解防护 - 基于 IP+账号 双维度限制
	rateLimitKey := fmt.Sprintf("login:%s:%s", c.ClientIP(), email)
	if checkLoginRateLimit(rateLimitKey) {
		log.Printf("[Security] Login rate limited: IP=%s, email=%s", c.ClientIP(), email)
		utils.TooManyRequests(c, 15)
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		recordLoginFailure(rateLimitKey)
		utils.Unauthorized(c, "用户名或密码错误")
		return
	}

	if user.Status != "active" {
		utils.Forbidden(c, "账户已被禁用")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		recordLoginFailure(rateLimitKey)
		log.Printf("[Login] Password mismatch for user %s from IP %s", email, c.ClientIP())
		utils.Unauthorized(c, "邮箱或密码错误")
		return
	}

	// 登录成功，清除限制记录
	clearLoginAttempts(rateLimitKey)

	// 获取主组织ID（使用 M-9 抽取的 checkOrgServiceValid）
	var primaryOrgID string
	var userOrg models.UserOrganization
	if err := h.db.Preload("Organization").Where("user_id = ? AND is_primary = ?", user.ID, true).First(&userOrg).Error; err == nil {
		if msg, ok := checkOrgServiceValid(userOrg.Organization); !ok {
			utils.Forbidden(c, msg)
			return
		}
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

	// G-6: 使用公共函数构建响应
	_, userWithOrg := buildLoginResponse(h.db, &user, now)

	utils.SuccessResponse(c, gin.H{
		"token": token,
		"user":  userWithOrg,
	})
}

func (h *AuthHandler) PhonePasswordLogin(c *gin.Context) {
	var req PhonePasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的登录凭证")
		return
	}

	cc, phone := normalizePhone(req.PhoneCountryCode, req.PhoneNumber)

	// S-3: 暴力破解防护
	rateLimitKey := fmt.Sprintf("phone_login:%s:%s", c.ClientIP(), phone)
	if checkLoginRateLimit(rateLimitKey) {
		log.Printf("[Security] Phone login rate limited: IP=%s, phone=%s", c.ClientIP(), phone)
		utils.TooManyRequests(c, 15)
		return
	}

	var user models.User
	if err := h.db.Where("phone_country_code = ? AND phone_number = ?", cc, phone).First(&user).Error; err != nil {
		recordLoginFailure(rateLimitKey)
		utils.Unauthorized(c, "手机号或密码错误")
		return
	}

	if user.Status != "active" {
		utils.Forbidden(c, "账户已被禁用")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		recordLoginFailure(rateLimitKey)
		log.Printf("[PhoneLogin] Password mismatch for %s from IP %s", phone, c.ClientIP())
		utils.Unauthorized(c, "手机号或密码错误")
		return
	}

	// 登录成功，清除限制记录
	clearLoginAttempts(rateLimitKey)

	// M-9: 使用抽取的组织校验函数
	var primaryOrgID string
	var userOrg models.UserOrganization
	if err := h.db.Preload("Organization").Where("user_id = ? AND is_primary = ?", user.ID, true).First(&userOrg).Error; err == nil {
		if msg, ok := checkOrgServiceValid(userOrg.Organization); !ok {
			utils.Forbidden(c, msg)
			return
		}
		primaryOrgID = userOrg.OrganizationID
	}

	token, err := utils.GenerateToken(user.ID, strings.TrimSpace(cc+" "+phone), user.Role, primaryOrgID)
	if err != nil {
		utils.InternalError(c, "生成令牌失败")
		return
	}

	now := time.Now()
	h.db.Model(&user).Update("last_login", now)

	// G-6: 使用公共函数构建响应
	_, userWithOrg := buildLoginResponse(h.db, &user, now)

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

func (h *AuthHandler) SendSMSCode(c *gin.Context) {
	var req SendSMSCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的手机号和验证码场景")
		return
	}
	cc, phone := normalizePhone(req.PhoneCountryCode, req.PhoneNumber)
	if len(phone) < 11 {
		utils.BadRequest(c, "手机号格式不正确")
		return
	}
	var recent int64
	h.db.Model(&models.AuthVerificationCode{}).Where("phone_country_code = ? AND phone_number = ? AND scene = ? AND created_at > ?", cc, phone, req.Scene, time.Now().Add(-1*time.Minute)).Count(&recent)
	if recent > 0 {
		utils.BadRequest(c, "请求过于频繁，请稍后再试")
		return
	}
	code := services.Generate6DigitCode()
	provider := services.NewSMSProvider()
	bizID, sendErr := provider.SendCode(cc, phone, code, req.Scene)
	status := "sent"
	errCode := ""
	errMsg := ""
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
	}
	now := time.Now()
	h.db.Create(&models.SMSSendLog{Provider: provider.Name(), PhoneCountryCode: cc, PhoneNumber: phone, TemplateCode: req.Scene, BizID: bizID, Status: status, ErrorCode: errCode, ErrorMessage: errMsg, SentAt: &now})
	if sendErr != nil {
		utils.InternalError(c, "短信发送失败，请检查短信通道配置")
		return
	}
	rec := models.AuthVerificationCode{Scene: req.Scene, PhoneCountryCode: cc, PhoneNumber: phone, CodeHash: codeHash(cc, phone, code, req.Scene), ExpiresAt: time.Now().Add(5 * time.Minute), RequestIP: c.ClientIP()}
	h.db.Create(&rec)
	resp := gin.H{"cooldown_seconds": 60}
	if config.AppConfig != nil && config.AppConfig.Mode == "debug" {
		resp["debug_code"] = code
	}
	utils.SuccessResponse(c, resp)
}

func (h *AuthHandler) SMSLogin(c *gin.Context) {
	var req SMSLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的手机号和验证码")
		return
	}
	cc, phone := normalizePhone(req.PhoneCountryCode, req.PhoneNumber)
	if err := h.checkCode(cc, phone, req.Code, "login"); err != nil {
		utils.Unauthorized(c, err.Error())
		return
	}
	var user models.User
	if err := h.db.Where("phone_country_code = ? AND phone_number = ?", cc, phone).First(&user).Error; err != nil {
		utils.NotFound(c, "该手机号未绑定账号")
		return
	}
	var userOrgs []models.UserOrganization
	h.db.Preload("Organization").Where("user_id = ?", user.ID).Find(&userOrgs)
	if len(userOrgs) == 0 {
		utils.Forbidden(c, "该账号未分配机构")
		return
	}
	orgs := make([]map[string]interface{}, 0, len(userOrgs))
	for _, uo := range userOrgs {
		if uo.Organization != nil {
			orgs = append(orgs, map[string]interface{}{"id": uo.OrganizationID, "name": uo.Organization.Name, "is_primary": uo.IsPrimary, "position": uo.Position})
		}
	}
	if user.LastOrgID == nil || *user.LastOrgID == "" {
		tokenTemp, _ := utils.GenerateToken(user.ID, user.Email, user.Role, "")
		utils.SuccessResponse(c, gin.H{"need_select_org": true, "token_temp": tokenTemp, "orgs": orgs})
		return
	}

	var lastOrg models.Organization
	if err := h.db.Select("service_status, service_end").First(&lastOrg, "id = ?", *user.LastOrgID).Error; err == nil {
		if msg, ok := checkOrgServiceValid(&lastOrg); !ok {
			utils.Forbidden(c, msg)
			return
		}
	}

	token, err := utils.GenerateToken(user.ID, user.Email, user.Role, *user.LastOrgID)
	if err != nil {
		utils.InternalError(c, "生成令牌失败")
		return
	}
	now := time.Now()
	h.db.Model(&user).Updates(map[string]interface{}{"last_login": now, "phone_verified_at": now})
	utils.SuccessResponse(c, gin.H{"need_select_org": false, "token": token, "user": user.ToResponse(), "orgs": orgs})
}

func (h *AuthHandler) SelectOrg(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req SelectOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供机构ID")
		return
	}
	var uo models.UserOrganization
	if err := h.db.Preload("Organization").Where("user_id = ? AND organization_id = ?", userID, req.OrgID).First(&uo).Error; err != nil {
		utils.Forbidden(c, "无权限访问该机构")
		return
	}
	if msg, ok := checkOrgServiceValid(uo.Organization); !ok {
		utils.Forbidden(c, msg)
		return
	}
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	token, err := utils.GenerateToken(user.ID, user.Email, user.Role, req.OrgID)
	if err != nil {
		utils.InternalError(c, "生成令牌失败")
		return
	}
	now := time.Now()
	h.db.Model(&user).Updates(map[string]interface{}{"last_org_id": req.OrgID, "last_login": now})
	utils.SuccessResponse(c, gin.H{"token": token, "current_org": gin.H{"id": req.OrgID}})
}

func (h *AuthHandler) ListUserOrgs(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var userOrgs []models.UserOrganization
	h.db.Preload("Organization").Where("user_id = ?", userID).Find(&userOrgs)
	orgs := make([]map[string]interface{}, 0, len(userOrgs))
	for _, uo := range userOrgs {
		if uo.Organization != nil {
			orgs = append(orgs, map[string]interface{}{"id": uo.OrganizationID, "name": uo.Organization.Name, "is_primary": uo.IsPrimary, "position": uo.Position})
		}
	}
	utils.SuccessResponse(c, orgs)
}

func (h *AuthHandler) SwitchOrg(c *gin.Context) {
	h.SelectOrg(c)
}

func (h *AuthHandler) ResetPasswordBySMS(c *gin.Context) {
	var req ResetPasswordBySMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效参数")
		return
	}
	cc, phone := normalizePhone(req.PhoneCountryCode, req.PhoneNumber)
	if err := h.checkCode(cc, phone, req.Code, "reset_password"); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	var user models.User
	if err := h.db.Where("phone_country_code = ? AND phone_number = ?", cc, phone).First(&user).Error; err != nil {
		utils.NotFound(c, "用户不存在")
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "密码加密失败")
		return
	}
	now := time.Now()
	h.db.Model(&user).Updates(map[string]interface{}{"password": string(hashedPassword), "phone_verified_at": now})
	utils.SuccessResponse(c, gin.H{"message": "密码重置成功"})
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
