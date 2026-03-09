package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret     []byte
	jwtSecretOnce sync.Once
)

// getJWTSecret 从环境变量获取JWT密钥
func getJWTSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "trackcard-jwt-secret-2026"
			log.Println("[Warning] JWT_SECRET not set, using default trackcard-server secret")
		}
		jwtSecret = []byte(secret)
	})
	return jwtSecret
}

// Claims JWT声明
type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	OrgID    string `json:"org_id"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT Token
func GenerateToken(userID, username, role string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Email:    username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "trackcard-admin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// JWTAuth JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "code": "AUTH_REQUIRED", "message": "未提供认证信息"})
			c.Abort()
			return
		}

		// 提取token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "code": "AUTH_FORMAT_ERROR", "message": "认证格式错误"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "code": "TOKEN_INVALID", "message": "Token无效或已过期"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "code": "TOKEN_PARSE_ERROR", "message": "Token解析失败"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		if claims.Username != "" {
			c.Set("username", claims.Username)
		} else {
			c.Set("username", claims.Email)
		}
		c.Set("role", claims.Role)

		c.Next()
	}
}

// CORS 跨域中间件（支持配置）
func CORS() gin.HandlerFunc {
	allowedOrigins := []string{"http://localhost:5175", "http://127.0.0.1:5175"}
	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false

		// 开发环境允许所有本地地址
		if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
			allowed = true
		} else {
			for _, ao := range allowedOrigins {
				if origin == strings.TrimSpace(ao) {
					allowed = true
					break
				}
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequireRole 角色权限检查
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("role")
		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"success": false, "code": "FORBIDDEN", "message": "权限不足"})
		c.Abort()
	}
}
