package middleware

import (
	"strings"
	"time"

	"trackcard-server/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware 处理跨域请求，从环境变量 CORS_ORIGINS 读取允许的来源
func CORSMiddleware() gin.HandlerFunc {
	// 从配置读取允许的来源，支持逗号分隔的多个域名
	origins := []string{
		"http://localhost:5173", "http://localhost:5174", "http://localhost:5175",
		"http://localhost:5180", "http://localhost:5181",
		"http://127.0.0.1:5173", "http://127.0.0.1:5174", "http://127.0.0.1:5175",
		"http://127.0.0.1:5180", "http://127.0.0.1:5181",
	}
	if config.AppConfig != nil && config.AppConfig.CORSOrigins != "" {
		customOrigins := strings.Split(config.AppConfig.CORSOrigins, ",")
		for _, o := range customOrigins {
			o = strings.TrimSpace(o)
			if o != "" {
				origins = append(origins, o)
			}
		}
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
