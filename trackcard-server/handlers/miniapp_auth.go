package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"trackcard-server/config"
	"trackcard-server/models"
	"trackcard-server/utils"
)

type MiniAppAuthHandler struct {
	db *gorm.DB
}

func NewMiniAppAuthHandler(db *gorm.DB) *MiniAppAuthHandler {
	return &MiniAppAuthHandler{db: db}
}

type WechatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

// REMOVED binding:required for DEBUGGING
type WechatBindRequest struct {
	Code     string `json:"code"`
	Email    string `json:"username"` // Match frontend "username" key, map to Email
	Password string `json:"password"`
}

type WechatSessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// Login handles WeChat Mini-program login
func (h *MiniAppAuthHandler) Login(c *gin.Context) {
	var req WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, fmt.Sprintf("Invalid request parameters: %v", err))
		return
	}

	// 1. Get OpenID from WeChat API
	openID, _, err := h.getWechatSession(req.Code)
	if err != nil {
		utils.InternalError(c, "Failed to authenticate with WeChat: "+err.Error())
		return
	}

	// 2. Find user by OpenID
	var user models.User
	result := h.db.Where("wechat_open_id = ?", openID).First(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Return NOT_FOUND to trigger binding flow on frontend
			c.JSON(http.StatusOK, gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User not bound",
				"data": gin.H{
					"openid": openID,
				},
			})
			return
		}
		utils.InternalError(c, "Database error")
		return
	}

	if user.Status != "active" {
		utils.Forbidden(c, "Account is disabled")
		return
	}

	// 3. Login successful, generate token
	h.generateTokenAndRespond(c, &user)
}

// Bind handles WeChat Mini-program account binding
func (h *MiniAppAuthHandler) Bind(c *gin.Context) {
	var req WechatBindRequest
	// Remove validation check for DEBUGGING
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf(" [DEBUG] Bind JSON Error (Ignored type mismatch): %v\n", err)
	}
	fmt.Printf(" [DEBUG] Bind Request Received: Code='%s', Email='%s', PassLen=%d\n", req.Code, req.Email, len(req.Password))

	// Validate manually
	if req.Code == "" {
		fmt.Printf(" [DEBUG] Code is EMPTY! Request Payload might be wrong.\n")
		utils.BadRequest(c, "Code is required")
		return
	}
	if req.Email == "" {
		utils.BadRequest(c, "Email is required")
		return
	}

	// Exchange Code for OpenID
	openID, _, err := h.getWechatSession(req.Code)
	if err != nil {
		utils.InternalError(c, "Failed to get OpenID from WeChat: "+err.Error())
		return
	}

	// 1. Verify credentials
	var user models.User
	if err := h.db.Where("email = ? OR name = ?", req.Email, req.Email).First(&user).Error; err != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	if user.Status != "active" {
		utils.Forbidden(c, "Account is disabled")
		return
	}

	// 2. Update User with OpenID
	updates := map[string]interface{}{
		"wechat_open_id": openID,
		"last_login":     time.Now(),
	}
	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		utils.InternalError(c, "Failed to bind account")
		return
	}

	// Reload user to get latest state
	h.db.First(&user, "id = ?", user.ID)

	// 3. Generate token
	h.generateTokenAndRespond(c, &user)
}

func (h *MiniAppAuthHandler) generateTokenAndRespond(c *gin.Context, user *models.User) {
	// Get Primary Org
	var primaryOrgID string
	var userOrg models.UserOrganization
	if err := h.db.Where("user_id = ? AND is_primary = ?", user.ID, true).First(&userOrg).Error; err == nil {
		primaryOrgID = userOrg.OrganizationID
	}

	token, err := utils.GenerateToken(user.ID, user.Email, user.Role, primaryOrgID)
	if err != nil {
		utils.InternalError(c, "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "Login successful",
		"data": gin.H{
			"token": token,
			"user":  user,
		},
	})
}

// getWechatSession exchanges code for session info
func (h *MiniAppAuthHandler) getWechatSession(code string) (string, string, error) {
	appID := config.AppConfig.WechatAppID
	appSecret := config.AppConfig.WechatAppSecret

	if appSecret == "" {
		// MOCK MODE: Return mock OpenID
		return "mock_openid_" + code, "mock_session_key", nil
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		appID, appSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var sess WechatSessionResponse
	if err := json.Unmarshal(body, &sess); err != nil {
		return "", "", err
	}

	if sess.ErrCode != 0 {
		return "", "", fmt.Errorf("WeChat API Error: %d %s", sess.ErrCode, sess.ErrMsg)
	}

	return sess.OpenID, sess.SessionKey, nil
}
