package config

import (
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	Port       string
	Mode       string // dev, test, release

	// 快货运 API 配置
	KuaihuoyunCID       string
	KuaihuoyunSecretKey string
	KuaihuoyunBaseURL   string

	// 安全配置
	CORSOrigins   string // 允许的CORS来源,逗号分隔
	AdminPassword string // 默认管理员密码

	// 微信小程序配置
	WechatAppID     string
	WechatAppSecret string
}

var AppConfig *Config

func Load() *Config {
	AppConfig = &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5001"),
		DBUser:     getEnv("DB_USER", "trackcard"),
		DBPassword: getEnv("DB_PASSWORD", "trackcard123"),
		DBName:     getEnv("DB_NAME", "trackcard"),
		JWTSecret:  getEnv("JWT_SECRET", "trackcard-jwt-secret-2026"),
		Port:       getEnv("PORT", "5051"),
		Mode:       getEnv("GIN_MODE", "debug"),

		KuaihuoyunCID:       getEnv("KUAIHUOYUN_CID", "1067"),
		KuaihuoyunSecretKey: getEnv("KUAIHUOYUN_SECRET_KEY", "O0DcRZTWrCwZqL44"),
		KuaihuoyunBaseURL:   getEnv("KUAIHUOYUN_BASE_URL", "http://bapi.kuaihuoyun.com/api/v1"),

		CORSOrigins:   getEnv("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),

		WechatAppID:     getEnv("WECHAT_APP_ID", "wx2548d0b3a6d16415"),
		WechatAppSecret: getEnv("WECHAT_APP_SECRET", ""),
	}
	return AppConfig
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
