package middleware

import (
	"strings"
	"time"

	"trackcard-server/models"
	"trackcard-server/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// authDB 用于认证中间件中的数据库查询（由 InitAuthDB 注入）
var authDB *gorm.DB

// InitAuthDB 注入数据库实例供认证中间件使用
func InitAuthDB(db *gorm.DB) {
	authDB = db
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Unauthorized(c, "认证令牌格式错误")
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			utils.Unauthorized(c, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		// org_id 优先使用请求参数中的，如果没有才使用 JWT claims 中的
		requestOrgID := c.Query("org_id")
		if requestOrgID == "" {
			requestOrgID = c.PostForm("org_id")
		}
		if requestOrgID == "" {
			requestOrgID = claims.OrgID
		}

		// 安全校验：如果请求的 org_id 与 JWT 中不同，必须验证用户是否属于该组织
		if requestOrgID != "" && requestOrgID != claims.OrgID && authDB != nil {
			var count int64
			authDB.Model(&models.UserOrganization{}).
				Where("user_id = ? AND organization_id = ?", claims.UserID, requestOrgID).
				Count(&count)
			if count == 0 {
				utils.Forbidden(c, "无权访问该组织")
				c.Abort()
				return
			}
		}

		c.Set("org_id", requestOrgID)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			utils.Unauthorized(c, "未认证")
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			utils.InternalError(c, "用户角色类型错误")
			c.Abort()
			return
		}
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		utils.Forbidden(c, "权限不足")
		c.Abort()
	}
}

// CheckOrgService 检查组织服务状态和期限
func CheckOrgService(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, exists := c.Get("org_id")
		if !exists {
			c.Next()
			return
		}

		orgIDStr, ok := orgID.(string)
		if !ok || orgIDStr == "" {
			c.Next()
			return
		}

		// 检查缓存或数据库
		var org models.Organization
		if err := db.Select("service_status, service_end").First(&org, "id = ?", orgIDStr).Error; err == nil {
			if org.ServiceStatus != "active" && org.ServiceStatus != "trial" && org.ServiceStatus != "" {
				utils.Forbidden(c, "组织服务已被禁用或过期，请联系管理员")
				c.Abort()
				return
			}
			if org.ServiceEnd != nil && !org.ServiceEnd.IsZero() && time.Now().After(*org.ServiceEnd) {
				utils.Forbidden(c, "组织服务已到期，请联系管理员续费")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
