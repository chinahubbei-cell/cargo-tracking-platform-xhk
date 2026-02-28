package middleware

import (
	"strings"

	"trackcard-server/utils"

	"github.com/gin-gonic/gin"
)

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
		c.Set("org_id", claims.OrgID)

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
