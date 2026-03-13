package main

import (
	"log"
	"os"
	"path/filepath"

	"trackcard-admin/handlers"
	"trackcard-admin/middleware"
	"trackcard-admin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// 1. 初始化数据库（共享trackcard.db）
	db := initDatabase()

	// 2. 自动迁移
	autoMigrate(db)

	// 3. 创建默认管理员
	createDefaultAdmin(db)

	// 4. 初始化Gin
	r := gin.Default()

	// 5. CORS中间件
	r.Use(middleware.CORS())

	// 6. 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "trackcard-admin"})
	})

	// 7. API路由
	api := r.Group("/api/admin")
	{
		// 认证（无需token）
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.NewAuthHandler(db).Login)
		}

		// 需要认证的路由
		protected := api.Group("")
		protected.Use(middleware.JWTAuth())
		{
			// 管理员信息
			protected.GET("/me", handlers.NewAuthHandler(db).GetCurrentUser)
			protected.PUT("/me/password", handlers.NewAuthHandler(db).ChangePassword)

			// 组织管理
			orgHandler := handlers.NewOrganizationHandler(db)
			protected.GET("/orgs", orgHandler.List)
			protected.POST("/orgs", orgHandler.Create)
			protected.GET("/orgs/:id", orgHandler.Get)
			protected.PUT("/orgs/:id", orgHandler.Update)
			protected.DELETE("/orgs/:id", orgHandler.Delete)
			protected.PUT("/orgs/:id/service", orgHandler.SetService)
			protected.POST("/orgs/:id/renew", orgHandler.Renew)
			protected.GET("/orgs/expiring", orgHandler.GetExpiring)
			protected.GET("/orgs/:id/renewals", orgHandler.GetRenewals)
			protected.GET("/orgs/:id/devices", orgHandler.GetDevices)
			protected.GET("/orgs/:id/stats", orgHandler.GetStats)

			// 设备管理
			deviceHandler := handlers.NewDeviceHandler(db)
			protected.GET("/devices", deviceHandler.List)
			protected.POST("/devices", deviceHandler.Create)
			protected.POST("/devices/batch-import", deviceHandler.BatchImport)
			protected.GET("/devices/:id", deviceHandler.Get)
			protected.PUT("/devices/:id", deviceHandler.Update)
			protected.PUT("/devices/:id/allocate", deviceHandler.Allocate)
			protected.PUT("/devices/:id/return", deviceHandler.Return)
			protected.GET("/devices/stats", deviceHandler.Stats)
			protected.GET("/devices/logs", deviceHandler.GetLogs)

			// 仪表盘
			protected.GET("/dashboard", handlers.NewDashboardHandler(db).GetStats)
		}
	}

	// 8. 启动服务
	port := os.Getenv("ADMIN_PORT")
	if port == "" {
		port = "8001"
	}
	log.Printf("[Admin] TrackCard管理后台启动于 http://localhost:%s", port)
	r.Run(":" + port)
}

func initDatabase() *gorm.DB {
	// 使用与主系统相同的数据库
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		// 优先使用主系统数据库：../../trackcard-server/trackcard.db
		candidates := []string{
			filepath.Join("..", "..", "trackcard-server", "trackcard.db"),
			filepath.Join("..", "..", "trackcard.db"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				dbPath = candidate
				break
			}
		}
		if dbPath == "" {
			dbPath = candidates[0]
		}
	}
	if abs, err := filepath.Abs(dbPath); err == nil {
		dbPath = abs
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// SQLite优化
	sqlDB, _ := db.DB()
	sqlDB.Exec("PRAGMA journal_mode=WAL")
	sqlDB.Exec("PRAGMA busy_timeout=5000")

	log.Printf("[Admin] 数据库连接成功: %s", dbPath)
	return db
}

func autoMigrate(db *gorm.DB) {
	db.AutoMigrate(
		&models.AdminUser{},
		&models.Organization{},
		&models.ServiceRenewal{},
		&models.HardwareDevice{},
		&models.DeviceAllocationLog{},
	)
	log.Println("[Admin] 数据库迁移完成")
}

func createDefaultAdmin(db *gorm.DB) {
	var count int64
	db.Model(&models.AdminUser{}).Count(&count)
	if count == 0 {
		admin := &models.AdminUser{
			Username: "admin",
			Name:     "系统管理员",
			Email:    "admin@trackcard.com",
			Role:     "super_admin",
			Status:   "active",
		}
		admin.SetPassword("admin123")
		db.Create(admin)
		log.Println("[Admin] 默认管理员账号已创建: admin / admin123")
	}
}
