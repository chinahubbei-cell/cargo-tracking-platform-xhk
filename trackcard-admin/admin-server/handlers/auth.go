package handlers

import (
	"net/http"
	"time"

	"trackcard-admin/middleware"
	"trackcard-admin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 管理员登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	var user models.AdminUser
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "用户名或密码错误"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "账号已禁用"})
		return
	}

	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "用户名或密码错误"})
		return
	}

	// 生成token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "生成Token失败"})
		return
	}

	// 更新最后登录时间
	now := time.Now()
	h.db.Model(&user).Update("last_login", now)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"name":     user.Name,
				"email":    user.Email,
				"role":     user.Role,
			},
		},
	})
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("user_id")

	var user models.AdminUser
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"name":       user.Name,
			"email":      user.Email,
			"phone":      user.Phone,
			"role":       user.Role,
			"last_login": user.LastLogin,
		},
	})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	var user models.AdminUser
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	if !user.CheckPassword(req.OldPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "原密码错误"})
		return
	}

	user.SetPassword(req.NewPassword)
	h.db.Model(&user).Update("password", user.Password)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "密码修改成功"})
}
