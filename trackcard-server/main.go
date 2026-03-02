package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"trackcard-server/config"
	"trackcard-server/handlers"
	"trackcard-server/middleware"
	"trackcard-server/models"
	"trackcard-server/seeds"
	"trackcard-server/services"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库
	db, err := config.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移数据库结构
	if err := db.AutoMigrate(
		&models.User{},
		&models.Device{},
		&models.Shipment{},
		&models.LocationHistory{},
		&models.Alert{},
		&models.SystemConfig{},
		&models.DeviceTrack{},
		&models.DeviceSyncStatus{},
		&models.ShipmentSequence{},      // 运单序号表
		&models.Route{},                 // 线路规划表
		&models.Port{},                  // 港口表
		&models.ShippingLine{},          // 航线表
		&models.LogisticsZone{},         // 物流区域表
		&models.CostTimeModel{},         // 成本时效模型
		&models.RoutePlan{},             // 规划结果表
		&models.ShipmentDeviceBinding{}, // 运单设备绑定历史表
		&models.ShipmentLog{},           // 运单操作日志表
		&models.Organization{},          // 组织机构表
		&models.UserOrganization{},      // 用户组织关联表
		&models.PortGeofence{},          // 港口围栏表
		&models.CarrierTrack{},          // 船司追踪表 (Phase 2新增)
		&models.CarrierMilestone{},      // 船司里程碑表 (Phase 2新增)
		&models.Partner{},               // 合作伙伴表 (Phase 3新增)
		&models.ShipmentCollaboration{}, // 运单协作表 (Phase 3新增)
		&models.ShipmentDocument{},      // 运单单据表 (Phase 3新增)
		&models.FreightRate{},           // 运价表 (Phase 4新增)
		&models.PartnerPerformance{},    // 货代绩效表 (Phase 4新增)
		&models.DocumentTemplate{},      // 文档模板表 (Phase 5新增)
		&models.OCRResult{},             // OCR识别结果表 (Phase 5新增)
		&models.GeneratedDocument{},     // 生成文档表 (Phase 5新增)
		&models.ShipmentStage{},         // 运输环节表 (Phase 6新增)
		// Phase 7: 节点配置引擎
		&models.LogisticsProduct{},  // 物流产品表
		&models.MilestoneTemplate{}, // 节点模板表
		&models.MilestoneNode{},     // 节点定义表
		&models.ShipmentMilestone{}, // 运单节点实例表 (Phase 7节点配置引擎)
		// Phase 8: 交互网关层
		&models.MagicLink{},          // 魔术链接表
		&models.AuditLog{},           // 审计日志表
		&models.ShadowOperationLog{}, // 影子操作日志表
		&models.Task{},               // 任务表 (Phase 8新增)
		&models.TaskDispatchRule{},   // 任务分发规则表 (Phase 8新增)
		&models.Customer{},           // 客户表
		&models.Airport{},            // 机场表 (Phase 9新增)
		&models.AirportGeofence{},    // 机场围栏表 (Phase 9新增)
		&models.DeviceStopRecord{},   // 设备停留记录表
		&models.TransitCityRecord{},  // 运单途经城市记录表
		&models.AuthVerificationCode{},
		&models.SMSSendLog{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("📦 Database migrated successfully")

	// 初始化快货运服务
	services.InitKuaihuoyun()
	log.Println("📡 Kuaihuoyun service initialized")

	// 初始化地理围栏服务
	services.InitGeofence(db)
	log.Println("📍 Geofence service initialized")

	// 初始化运单号生成器
	services.InitShipmentIDGenerator(db)
	log.Println("🔢 Shipment ID generator initialized")

	// 初始化文件存储服务 (Phase 5)
	services.InitFileService("./uploads", "/api/files")
	log.Println("📁 File service initialized")

	// 初始化PDF生成器 (Phase 5)
	services.InitPDFGenerator(db, services.GetFileService())
	log.Println("📄 PDF generator initialized")

	// 初始化并启动同步调度器
	services.InitScheduler(db)
	services.Scheduler.Start()
	log.Println("⏰ Sync scheduler started")

	// 初始化设备绑定服务
	services.InitDeviceBinding(db)

	// 初始化运单日志服务
	services.InitShipmentLog(db)

	// 围栏服务依赖日志服务，确保日志服务先初始化
	// 启动后自动补录缺失的围栏触发日志（异步执行，不阻塞启动）
	go func() {
		if services.Geofence != nil {
			count, err := services.Geofence.BackfillMissingLogs()
			if err != nil {
				log.Printf("⚠️ [Geofence] 自动补录失败: %v", err)
			} else if count > 0 {
				log.Printf("✅ [Geofence] 自动补录完成，共补录 %d 条围栏触发日志", count)
			}
		}
	}()

	// 启动电子围栏后台检查任务（每60秒检查一次）
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		log.Println("🔄 [Geofence] 后台围栏检查任务已启动 (间隔: 60秒)")

		for range ticker.C {
			if services.Geofence == nil {
				continue
			}

			// 查询所有有设备且启用自动状态的运单
			var shipments []models.Shipment
			if err := db.Where("device_id IS NOT NULL AND device_id != '' AND auto_status_enabled = ? AND status IN ?",
				true, []string{"pending", "in_transit"}).Find(&shipments).Error; err != nil {
				log.Printf("⚠️ [Geofence] 查询运单失败: %v", err)
				continue
			}

			if len(shipments) == 0 {
				continue
			}

			checkedCount := 0
			for _, shipment := range shipments {
				if shipment.DeviceID == nil || *shipment.DeviceID == "" {
					continue
				}

				// 获取设备最新轨迹
				var track models.DeviceTrack
				if err := db.Where("device_id = ?", *shipment.DeviceID).
					Order("locate_time DESC").First(&track).Error; err != nil {
					continue
				}

				// 调用围栏检测
				services.Geofence.CheckAndUpdateStatus(*shipment.DeviceID, track.Latitude, track.Longitude)
				checkedCount++
			}

			if checkedCount > 0 {
				log.Printf("🔍 [Geofence] 后台检查完成，共检查 %d 个运单", checkedCount)
			}
		}
	}()

	// 初始化数据权限服务
	services.InitDataPermission(db)
	log.Println("🔐 Data permission service initialized")

	// 初始化运输环节服务 (Phase 6新增)
	services.InitShipmentStageService(db)
	log.Println("📦 Shipment stage service initialized")

	// 初始化腾讯地图服务 (SecretKey从环境变量读取，用于签名验证)
	tencentMapSK := os.Getenv("TENCENT_MAP_SK")
	if tencentMapSK == "" {
		tencentMapSK = "1oJ0xgyshoOMEoTzQvGxfNl80V85oKWS" // 默认SK
	}
	services.InitTencentMap("C42BZ-YNQKV-VV5PV-5A2IY-TRSWQ-7XFR5", tencentMapSK)
	services.InitTencentMap("C42BZ-YNQKV-VV5PV-5A2IY-TRSWQ-7XFR5", tencentMapSK)
	log.Println("🗺️ Tencent Map service initialized")

	// 初始化OpenStreetMap Nominatim服务 (用于海外地址解析)
	// 请务必提供真实的联系邮箱，遵守Nominatim使用政策
	services.InitNominatimService("admin@trackcard.com")
	log.Println("🌍 Nominatim service initialized")

	// 初始化预警检测服务
	services.InitAlertChecker(db)
	log.Println("🚨 Alert checker service initialized")

	// 初始化船司追踪服务 (Phase 2新增)
	mockProvider := services.NewMockCarrierProvider(true) // 开发阶段使用Mock
	services.InitCarrierTracking(db, mockProvider)
	log.Println("🚢 Carrier tracking service initialized")

	// 初始化节点配置引擎 (Phase 7新增)
	services.InitMilestoneEngine(db)
	log.Println("🎯 Milestone engine initialized")

	// 初始化魔术链接服务 (Phase 8新增)
	magicLinkBaseURL := os.Getenv("MAGIC_LINK_BASE_URL")
	if magicLinkBaseURL == "" {
		magicLinkBaseURL = "http://localhost:5173" // 默认使用前端地址
	}
	services.InitMagicLinkService(db, magicLinkBaseURL)
	log.Println("🔗 Magic link service initialized")

	// 填充默认数据
	if err := handlers.SeedAdmin(db); err != nil {
		log.Printf("Warning: Failed to seed admin: %v", err)
	}
	if err := handlers.SeedConfig(db); err != nil {
		log.Printf("Warning: Failed to seed config: %v", err)
	}
	// if err := handlers.SeedDevices(db); err != nil {
	// 	log.Printf("Warning: Failed to seed devices: %v", err)
	// }
	// 初始化全球核心港口数据
	if err := seeds.SeedCorePorts(db); err != nil {
		log.Printf("Warning: Failed to seed core ports: %v", err)
	}
	// 初始化港口围栏数据
	if err := handlers.SeedPortGeofences(db); err != nil {
		log.Printf("Warning: Failed to seed port geofences: %v", err)
	}
	// 初始化全球机场数据 (Phase 9新增)
	if err := seeds.SeedAirports(db); err != nil {
		log.Printf("Warning: Failed to seed airports: %v", err)
	}
	// 初始化机场围栏数据 (Phase 9新增)
	if err := handlers.SeedAirportGeofences(db); err != nil {
		log.Printf("Warning: Failed to seed airport geofences: %v", err)
	}
	// 初始化示例运单数据
	if err := handlers.SeedShipments(db); err != nil {
		log.Printf("Warning: Failed to seed shipments: %v", err)
	}
	// 初始化示例组织机构数据
	if err := handlers.SeedOrganizations(db); err != nil {
		log.Printf("Warning: Failed to seed organizations: %v", err)
	}
	// 初始化物流产品和节点模板 (Phase 7新增)
	if err := seeds.SeedLogisticsProducts(db); err != nil {
		log.Printf("Warning: Failed to seed logistics products: %v", err)
	}

	// 初始化Gin引擎
	gin.SetMode(cfg.Mode)
	r := gin.Default()

	// 注册CORS中间件 (允许跨域请求)
	r.Use(middleware.CORSMiddleware())

	// 配置CORS - 从环境变量读取允许的来源
	allowedOrigins := config.AppConfig.CORSOrigins
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173,http://localhost:3000" // 开发环境默认值
	}
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// 检查来源是否在允许列表中
		allowedOriginFound := false
		for _, allowed := range strings.Split(allowedOrigins, ",") {
			if strings.TrimSpace(allowed) == origin || strings.TrimSpace(allowed) == "*" {
				c.Header("Access-Control-Allow-Origin", origin)
				allowedOriginFound = true
				break
			}
		}
		// 如果没有找到匹配的来源，默认允许 localhost 开发环境
		if !allowedOriginFound && origin != "" && strings.Contains(origin, "localhost") {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Shadow-Target, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(db)
	deviceHandler := handlers.NewDeviceHandler(db)
	shipmentHandler := handlers.NewShipmentHandler(db)
	alertHandler := handlers.NewAlertHandler(db)
	userHandler := handlers.NewUserHandler(db)
	dashboardHandler := handlers.NewDashboardHandler(db)
	configHandler := handlers.NewConfigHandler(db)
	routeHandler := handlers.NewRouteHandler(db)
	routePlanningHandler := handlers.NewRoutePlanningHandler(db)
	portHandler := handlers.NewPortHandler(db)
	organizationHandler := handlers.NewOrganizationHandler(db)
	searchHandler := handlers.NewSearchHandler(db)
	partnerHandler := handlers.NewPartnerHandler(db)                              // Phase 3新增
	rateHandler := handlers.NewRateHandler(db)                                    // Phase 4新增
	documentHandler := handlers.NewDocumentHandler(db, services.GetFileService()) // Phase 5新增
	stageHandler := handlers.NewShipmentStageHandler(db)                          // Phase 6新增
	geocodeHandler := handlers.NewGeocodeHandler()                                // 腾讯地图地理编码
	milestoneHandler := handlers.NewMilestoneHandler(db)                          // Phase 7新增
	magicLinkHandler := handlers.NewMagicLinkHandler(db)                          // Phase 8新增
	ocrHandler := handlers.NewOCRHandler(db)                                      // Phase 8新增: OCR服务
	auditHandler := handlers.NewAuditHandler(db)                                  // Phase 8新增: 审计日志

	// 初始化任务分发器 (Phase 8新增)
	services.InitTaskDispatcher(db)
	taskHandler := handlers.NewTaskHandler(db)
	geofenceHandler := handlers.NewGeofenceHandler(db)       // 围栏管理处理器
	customerHandler := handlers.NewCustomerHandler(db)       // 客户管理处理器
	airportHandler := handlers.NewAirportHandler(db)         // 机场管理处理器 (Phase 9新增)
	miniAppAuthHandler := handlers.NewMiniAppAuthHandler(db) // 小程序认证处理器
	deviceStopHandler := handlers.NewDeviceStopHandler(db)   // 设备停留记录处理器

	// API路由
	api := r.Group("/api")
	{
		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "ok",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})

		// Public Routes
		// 临时用于通过脚本触发全量重新生成 (已移除)

		// 认证路由（无需认证）
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/sms/send-code", authHandler.SendSMSCode)
			auth.POST("/sms/login", authHandler.SMSLogin)
			auth.POST("/password/reset-by-sms", authHandler.ResetPasswordBySMS)
		}

		// Magic Link 公开路由（无需认证）- Phase 8新增
		magic := api.Group("/m")
		{
			magic.GET("/:token", magicLinkHandler.HandleMagicLink)         // 获取操作页面数据
			magic.POST("/:token/submit", magicLinkHandler.SubmitMagicLink) // 提交操作
		}

		// 小程序 API
		miniapp := api.Group("/miniapp")
		{
			auth := miniapp.Group("/auth")
			{
				auth.POST("/login", miniAppAuthHandler.Login)
				auth.POST("/bind", miniAppAuthHandler.Bind)
			}
		}

		// 需要认证的路由
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		protected.Use(middleware.ShadowModeMiddleware(db)) // Phase 8: 影子模式中间件
		protected.Use(middleware.AuditMiddleware(db))      // Phase 8: 审计日志中间件
		{
			// 认证相关
			protected.GET("/auth/me", authHandler.GetCurrentUser)
			protected.POST("/auth/change-password", authHandler.ChangePassword)
			protected.POST("/auth/select-org", authHandler.SelectOrg)
			protected.GET("/auth/orgs", authHandler.ListUserOrgs)
			protected.POST("/auth/switch-org", authHandler.SwitchOrg)

			// 地理编码 (腾讯地图)
			protected.GET("/geocode", geocodeHandler.Geocode)
			protected.GET("/geocode/suggestion", geocodeHandler.Suggestion)
			protected.GET("/geocode/reverse", geocodeHandler.ReverseGeocode)

			// 搜索建议
			protected.GET("/search/suggestions", searchHandler.Suggestions)

			// 设备管理
			devices := protected.Group("/devices")
			{
				devices.GET("", deviceHandler.List)
				devices.GET("/:id", deviceHandler.Get)
				devices.POST("", middleware.RequireRole("admin", "operator"), deviceHandler.Create)
				devices.PUT("/:id", middleware.RequireRole("admin", "operator"), deviceHandler.Update)
				devices.DELETE("/:id", middleware.RequireRole("admin"), deviceHandler.Delete)
				devices.GET("/:id/track", deviceHandler.GetTrack)
				devices.GET("/:id/history", deviceHandler.GetHistory)
			}

			// 设备停留记录管理
			deviceStops := protected.Group("/device-stops")
			{
				deviceStops.GET("", deviceStopHandler.GetStopRecords)
				deviceStops.GET("/stats/:device_external_id", deviceStopHandler.GetDeviceStopStats)
				deviceStops.GET("/current/:device_external_id", deviceStopHandler.GetCurrentStop)
				deviceStops.GET("/record/:id", deviceStopHandler.GetStopByID)
				deviceStops.DELETE("/record/:id", middleware.RequireRole("admin", "operator"), deviceStopHandler.DeleteStop)
				deviceStops.DELETE("/batch", middleware.RequireRole("admin", "operator"), deviceStopHandler.BatchDeleteStops)
				deviceStops.POST("/update-active", deviceStopHandler.UpdateActiveStops)
				deviceStops.POST("/check-alerts", deviceStopHandler.CheckAlerts)
				deviceStops.POST("/analyze", middleware.RequireRole("admin", "operator"), deviceStopHandler.AnalyzeDeviceStops)
			}

			// 运单管理
			shipments := protected.Group("/shipments")
			{
				shipments.GET("", shipmentHandler.List)
				shipments.GET("/:id", shipmentHandler.Get)
				shipments.POST("", middleware.RequireRole("admin", "operator"), shipmentHandler.Create)
				shipments.PUT("/:id", middleware.RequireRole("admin", "operator"), shipmentHandler.Update)
				shipments.DELETE("/:id", middleware.RequireRole("admin"), shipmentHandler.Delete)
				shipments.GET("/:id/route", shipmentHandler.GetRoute)
				shipments.POST("/:id/check-status", shipmentHandler.CheckStatus)
				shipments.GET("/:id/logs", shipmentHandler.GetLogs)                                                              // 获取运单操作日志
				shipments.GET("/:id/bindings", shipmentHandler.GetBindingHistory)                                                // 获取设备绑定历史
				shipments.POST("/:id/transition", middleware.RequireRole("admin", "operator"), shipmentHandler.TransitionStatus) // 快捷状态切换
				// Phase 2: 船司追踪
				shipments.GET("/:id/carrier-tracks", shipmentHandler.GetCarrierTracks)                                             // 获取船司追踪事件
				shipments.GET("/:id/milestones", shipmentHandler.GetMilestones)                                                    // 获取统一里程碑
				shipments.POST("/:id/sync-carrier", middleware.RequireRole("admin", "operator"), shipmentHandler.SyncCarrierTrack) // 手动同步船司数据
				// Phase 3: 协作
				shipments.GET("/:id/collaborations", partnerHandler.GetShipmentCollaborations)                                      // 获取运单协作记录
				shipments.POST("/:id/collaborations", middleware.RequireRole("admin", "operator"), partnerHandler.AssignToShipment) // 分配协作方
				// Phase 5: 运单单据
				shipments.GET("/:id/documents", documentHandler.GetShipmentDocuments)                                         // 获取运单单据
				shipments.POST("/:id/documents", middleware.RequireRole("admin", "operator"), documentHandler.UploadDocument) // 上传单据
				shipments.GET("/:id/generated-documents", documentHandler.GetGeneratedDocuments)                              // 获取生成的文档
				// Phase 5: PDF生成
				shipments.POST("/stages/regenerate", middleware.RequireRole("admin"), stageHandler.RegenerateAll)                              // 批量重新生成环节 (Phase 6 maintenance)
				shipments.POST("/:id/generate/packing-list", middleware.RequireRole("admin", "operator"), documentHandler.GeneratePackingList) // 生成装箱单
				shipments.POST("/:id/generate/invoice", middleware.RequireRole("admin", "operator"), documentHandler.GenerateInvoice)          // 生成发票
				// Phase 6: 运输环节管理
				shipments.GET("/:id/stages", stageHandler.GetStages)                                                                        // 获取运单环节列表
				shipments.GET("/:id/stages/:stage_code", stageHandler.GetStage)                                                             // 获取单个环节详情
				shipments.PUT("/:id/stages/:stage_code", middleware.RequireRole("admin", "operator"), stageHandler.UpdateStage)             // 更新环节信息
				shipments.POST("/:id/stages/transition", middleware.RequireRole("admin", "operator"), stageHandler.TransitionStage)         // 手动推进环节
				shipments.POST("/:id/stages/:stage_code/start", middleware.RequireRole("admin", "operator"), stageHandler.StartStage)       // 开始环节
				shipments.POST("/:id/stages/:stage_code/complete", middleware.RequireRole("admin", "operator"), stageHandler.CompleteStage) // 完成环节
				// 设备轨迹数据
				shipments.GET("/:id/tracks", shipmentHandler.GetShipmentTracks) // 获取设备轨迹
				// 设备停留记录
				shipments.GET("/:id/stops", deviceStopHandler.GetShipmentStopRecords)            // 获取运单停留记录
				shipments.GET("/:id/transit-cities", deviceStopHandler.GetShipmentTransitCities) // 获取运单途经国家/城市
			}

			params := protected.Group("/partners")
			{
				params.GET("", partnerHandler.List)
				params.GET("/:id", partnerHandler.Get)
				params.GET("/:id/stats", partnerHandler.GetStats)
				params.POST("", middleware.RequireRole("admin"), partnerHandler.Create)
				params.PUT("/:id", middleware.RequireRole("admin"), partnerHandler.Update)
				params.DELETE("/:id", middleware.RequireRole("admin"), partnerHandler.Delete)
			}

			// 客户管理
			customers := protected.Group("/customers")
			{
				customers.GET("", customerHandler.List)
				customers.GET("/search", customerHandler.Search) // 搜索接口
				customers.GET("/:id", customerHandler.Get)
				customers.POST("", middleware.RequireRole("admin", "operator"), customerHandler.Create)
				customers.PUT("/:id", middleware.RequireRole("admin", "operator"), customerHandler.Update)
				customers.DELETE("/:id", middleware.RequireRole("admin"), customerHandler.Delete)
			}

			// Phase 3: 协作记录管理
			collaborations := protected.Group("/collaborations")
			{
				collaborations.PUT("/:id", partnerHandler.UpdateCollaboration) // 更新协作状态
			}

			// Phase 4: 运价管理
			rates := protected.Group("/rates")
			{
				rates.GET("", rateHandler.List)
				rates.GET("/routes", rateHandler.GetRoutes)           // 获取可用航线
				rates.GET("/performance", rateHandler.GetPerformance) // 货代绩效
				rates.POST("/compare", rateHandler.Compare)           // 智能比价
				rates.GET("/:id", rateHandler.Get)
				rates.POST("", middleware.RequireRole("admin", "operator"), rateHandler.Create)
				rates.PUT("/:id", middleware.RequireRole("admin", "operator"), rateHandler.Update)
				rates.DELETE("/:id", middleware.RequireRole("admin"), rateHandler.Delete)
			}

			// Phase 5: 文档管理
			documents := protected.Group("/documents")
			{
				documents.GET("/templates", documentHandler.ListTemplates)
				documents.GET("/:id/ocr-results", documentHandler.GetOCRResults)
				documents.POST("/:id/ocr", middleware.RequireRole("admin", "operator"), documentHandler.TriggerOCR) // 触发OCR识别
				documents.POST("/:id/apply-ocr", middleware.RequireRole("admin", "operator"), documentHandler.ApplyOCRResult)
				documents.GET("/:id/download", documentHandler.DownloadDocument)
				documents.PUT("/:id/review", middleware.RequireRole("admin", "operator"), documentHandler.ReviewDocument)
				documents.DELETE("/:id", middleware.RequireRole("admin"), documentHandler.DeleteDocument)
			}

			// 运单单据 (嵌套在shipments下)
			// 已在 /shipments/:id/documents 路由中

			// 预警管理
			alerts := protected.Group("/alerts")
			{
				alerts.GET("", alertHandler.List)
				alerts.GET("/stats", alertHandler.Stats)
				alerts.GET("/:id", alertHandler.Get)
				alerts.POST("", middleware.RequireRole("admin", "operator"), alertHandler.Create)
				alerts.PUT("/:id", middleware.RequireRole("admin", "operator"), alertHandler.Update)
				alerts.DELETE("/:id", middleware.RequireRole("admin"), alertHandler.Delete)
				alerts.POST("/:id/resolve", middleware.RequireRole("admin", "operator"), alertHandler.Resolve)
			}

			// 用户管理
			users := protected.Group("/users")
			users.Use(middleware.RequireRole("admin"))
			{
				users.GET("", userHandler.List)
				users.GET("/:id", userHandler.Get)
				users.POST("", userHandler.Create)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
				users.POST("/:id/reset-password", userHandler.ResetPassword)
				// 获取用户所属组织列表
				users.GET("/:id/organizations", organizationHandler.GetUserOrganizations)
			}

			// 仪表盘
			dashboard := protected.Group("/dashboard")
			{
				dashboard.GET("/stats", dashboardHandler.GetStats)
				dashboard.GET("/locations", dashboardHandler.GetLocations)
				dashboard.GET("/recent-alerts", dashboardHandler.GetRecentAlerts)
				dashboard.GET("/recent-shipments", dashboardHandler.GetRecentShipments)
				dashboard.GET("/sync-stats", dashboardHandler.GetSyncStats)
			}

			// 系统配置
			configRoutes := protected.Group("/config")
			configRoutes.Use(middleware.RequireRole("admin"))
			{
				configRoutes.GET("", configHandler.Get)
				configRoutes.POST("", configHandler.Update)
				configRoutes.GET("/:key", configHandler.GetByKey)
				configRoutes.POST("/test-kuaihuoyun", configHandler.TestKuaihuoyunAPI)
				// 运单字段配置
				configRoutes.GET("/shipment-fields", configHandler.GetShipmentFieldConfig)
				configRoutes.PUT("/shipment-fields", configHandler.UpdateShipmentFieldConfig)
			}

			// 线路规划
			routes := protected.Group("/routes")
			{
				routes.GET("", routeHandler.List)
				routes.GET("/:id", routeHandler.Get)
				routes.POST("", middleware.RequireRole("admin", "operator"), routeHandler.Create)
				routes.PUT("/:id", middleware.RequireRole("admin", "operator"), routeHandler.Update)
				routes.DELETE("/:id", middleware.RequireRole("admin"), routeHandler.Delete)
			}

			// 线路自动化规划
			routePlanning := protected.Group("/route-planning")
			{
				// 路径计算
				routePlanning.POST("/calculate", routePlanningHandler.CalculateRoutes)
				routePlanning.GET("/plans", routePlanningHandler.ListRoutePlans)
				routePlanning.POST("/plans", routePlanningHandler.SaveRoutePlan)

				// 港口管理
				routePlanning.GET("/ports", routePlanningHandler.ListPorts)
				routePlanning.GET("/ports/:id", routePlanningHandler.GetPort)
				routePlanning.POST("/ports", middleware.RequireRole("admin"), routePlanningHandler.CreatePort)
				routePlanning.PUT("/ports/:id", middleware.RequireRole("admin"), routePlanningHandler.UpdatePort)
				routePlanning.DELETE("/ports/:id", middleware.RequireRole("admin"), routePlanningHandler.DeletePort)

				// 航线管理
				routePlanning.GET("/shipping-lines", routePlanningHandler.ListShippingLines)
				routePlanning.POST("/shipping-lines", middleware.RequireRole("admin"), routePlanningHandler.CreateShippingLine)
			}

			// 全球港口管理 (Global Ports)
			ports := protected.Group("/ports")
			{
				ports.GET("", portHandler.GetPorts)
				ports.GET("/regions", portHandler.GetPortRegions)
				ports.GET("/countries", portHandler.GetPortCountries)
				ports.GET("/distance", portHandler.CalculateDistance)
				ports.GET("/:code", portHandler.GetPort)
				ports.POST("", middleware.RequireRole("admin"), portHandler.CreatePort)
				ports.PUT("/:code", middleware.RequireRole("admin"), portHandler.UpdatePort)
				ports.DELETE("/:code", middleware.RequireRole("admin"), portHandler.DeletePort)
			}

			// 港口围栏数据 (Port Geofences)
			portGeofences := protected.Group("/port-geofences")
			{
				portGeofences.GET("", portHandler.GetPortGeofences)
				portGeofences.GET("/:code", portHandler.GetPortGeofence)
			}

			// 全球机场管理 (Global Airports) - Phase 9新增
			airports := protected.Group("/airports")
			{
				airports.GET("", airportHandler.GetAirports)
				airports.GET("/regions", airportHandler.GetAirportRegions)
				airports.GET("/countries", airportHandler.GetAirportCountries)
				airports.GET("/distance", airportHandler.CalculateDistance)
				airports.GET("/:code", airportHandler.GetAirport)
				airports.POST("", middleware.RequireRole("admin"), airportHandler.CreateAirport)
				airports.PUT("/:code", middleware.RequireRole("admin"), airportHandler.UpdateAirport)
				airports.DELETE("/:code", middleware.RequireRole("admin"), airportHandler.DeleteAirport)
			}

			// 机场围栏数据 (Airport Geofences) - Phase 9新增
			airportGeofences := protected.Group("/airport-geofences")
			{
				airportGeofences.GET("", airportHandler.GetAirportGeofences)
				airportGeofences.GET("/:code", airportHandler.GetAirportGeofence)
			}

			// 组织机构管理
			organizations := protected.Group("/organizations")
			{
				organizations.GET("", organizationHandler.List)
				organizations.GET("/:id", organizationHandler.Get)
				organizations.POST("", middleware.RequireRole("admin"), organizationHandler.Create)
				organizations.PUT("/:id", middleware.RequireRole("admin"), organizationHandler.Update)
				organizations.DELETE("/:id", middleware.RequireRole("admin"), organizationHandler.Delete)
				organizations.PUT("/:id/move", middleware.RequireRole("admin"), organizationHandler.Move)

				// 组织用户管理
				organizations.GET("/:id/users", organizationHandler.GetUsers)
				organizations.POST("/:id/users", middleware.RequireRole("admin"), organizationHandler.AddUser)
				organizations.PUT("/:id/users/:user_id", middleware.RequireRole("admin"), organizationHandler.UpdateUserOrg)
				organizations.DELETE("/:id/users/:user_id", middleware.RequireRole("admin"), organizationHandler.RemoveUser)

				// 组织设备列表
				organizations.GET("/:id/devices", organizationHandler.GetDevices)
			}

			// Phase 7: 节点配置引擎
			milestones := protected.Group("/milestones")
			{
				// 物流产品管理
				milestones.GET("/products", milestoneHandler.ListProducts)
				milestones.GET("/products/:id", milestoneHandler.GetProduct)
				milestones.POST("/products", middleware.RequireRole("admin"), milestoneHandler.CreateProduct)

				// 模板管理
				milestones.GET("/templates", milestoneHandler.ListTemplates)
				milestones.GET("/templates/:id", milestoneHandler.GetTemplate)
				milestones.POST("/templates", middleware.RequireRole("admin"), milestoneHandler.CreateTemplate)
				milestones.GET("/templates/:id/nodes", milestoneHandler.ListNodes)

				// 节点管理
				milestones.POST("/nodes", middleware.RequireRole("admin"), milestoneHandler.CreateNode)
				milestones.PUT("/nodes/:id", middleware.RequireRole("admin"), milestoneHandler.UpdateNode)
				milestones.DELETE("/nodes/:id", middleware.RequireRole("admin"), milestoneHandler.DeleteNode)
			}

			// Phase 8: Magic Link 管理
			magicLinks := protected.Group("/magic-links")
			{
				magicLinks.POST("", middleware.RequireRole("admin", "operator"), magicLinkHandler.CreateLink)
				magicLinks.GET("/action-types", magicLinkHandler.GetActionTypes)
			}
			// 运单关联的魔术链接
			protected.GET("/shipments/:id/magic-links", magicLinkHandler.GetLinksByShipment)

			// Phase 8: OCR 识别服务
			ocr := protected.Group("/ocr")
			{
				ocr.POST("/recognize", ocrHandler.Recognize) // 即时识别
				ocr.GET("/doc-types", ocrHandler.GetSupportedDocTypes)
				ocr.GET("/provider", ocrHandler.GetOCRProviderInfo)
			}
			// 运单文档OCR
			protected.POST("/shipments/:id/ocr", middleware.RequireRole("admin", "operator"), ocrHandler.RecognizeAndSave)
			// 文档OCR结果
			protected.GET("/documents/:id/ocr", ocrHandler.GetOCRResults)
			protected.POST("/documents/:id/ocr/confirm", middleware.RequireRole("admin", "operator"), ocrHandler.ConfirmOCRResults)

			// Phase 8: 审计日志和影子模式
			audit := protected.Group("/audit")
			{
				audit.GET("/logs", middleware.RequireRole("admin"), auditHandler.ListAuditLogs)
				audit.GET("/shadow-logs", middleware.RequireRole("admin"), auditHandler.ListShadowLogs)
				audit.GET("/stats", middleware.RequireRole("admin"), auditHandler.GetAuditStats)
				audit.GET("/shadow-mode/status", auditHandler.GetShadowModeStatus)
				audit.GET("/shadow-mode/targets", middleware.RequireRole("admin", "operator"), auditHandler.GetShadowTargets)
			}

			// Phase 8: 任务管理
			tasks := protected.Group("/tasks")
			{
				tasks.GET("", taskHandler.ListTasks)
				tasks.POST("", middleware.RequireRole("admin", "operator"), taskHandler.CreateTask)
				tasks.GET("/stats", taskHandler.GetStats)
				tasks.GET("/dispatch-rules", taskHandler.ListDispatchRules)
				tasks.POST("/dispatch-rules", middleware.RequireRole("admin"), taskHandler.CreateDispatchRule)

				tasks.GET("/:id", taskHandler.GetTask)
				tasks.PUT("/:id/status", taskHandler.UpdateTaskStatus)
			}
			// 运单审计日志
			protected.GET("/shipments/:id/audit", auditHandler.GetResourceAuditLogs)

			// 围栏管理API (Admin)
			geofence := protected.Group("/admin/geofence")
			geofence.Use(middleware.RequireRole("admin"))
			{
				geofence.POST("/diagnose", geofenceHandler.DiagnoseShipment)         // 诊断运单围栏状态
				geofence.POST("/backfill", geofenceHandler.BackfillLogs)             // 补录缺失日志
				geofence.POST("/trigger/:shipment_id", geofenceHandler.TriggerCheck) // 手动触发围栏检测
			}
		}
	}

	// 启动服务
	log.Printf("🚀 TrackCard API Server running at http://localhost:%s", cfg.Port)
	log.Printf("📊 API Endpoints: http://localhost:%s/api", cfg.Port)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
