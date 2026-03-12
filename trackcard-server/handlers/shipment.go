package handlers

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

// 合法的状态转换路径
var validStatusTransitions = map[string][]string{
	"pending":    {"in_transit", "cancelled"},
	"in_transit": {"delivered", "cancelled"},
	"delivered":  {}, // 终态
	"cancelled":  {}, // 终态
}

// isValidStatusTransition 检查状态转换是否合法
func isValidStatusTransition(oldStatus, newStatus string) bool {
	if oldStatus == newStatus {
		return true // 状态未变
	}
	allowed, ok := validStatusTransitions[oldStatus]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

type ShipmentHandler struct {
	db *gorm.DB
}

func NewShipmentHandler(db *gorm.DB) *ShipmentHandler {
	return &ShipmentHandler{db: db}
}

// getOperatorInfo 获取操作人信息
func (h *ShipmentHandler) getOperatorInfo(c *gin.Context) (name, ip string) {
	ip = c.ClientIP()
	operatorID, exists := c.Get("user_id")
	if !exists {
		return "", ip
	}
	operatorIDStr, ok := operatorID.(string)
	if !ok || operatorIDStr == "" {
		return "", ip
	}
	var user models.User
	if err := h.db.Select("name").First(&user, "id = ?", operatorIDStr).Error; err == nil {
		return user.Name, ip
	}
	return "", ip
}

func (h *ShipmentHandler) List(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("search")
	orgID := c.Query("org_id") // 按组织筛选
	limitStr := c.Query("limit")

	query := h.db.Model(&models.Shipment{}).Preload("Device")

	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query = query.Limit(limit)
		}
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	// Optimization: Check for Precise ID Search FIRST
	// If the search term is a valid Shipment ID (exact match), we skip the heavy fuzzy search and Org filter.
	isPreciseSearch := false
	if search != "" && len(search) >= 10 {
		var count int64
		// Quick check on primary key (very fast)
		h.db.Model(&models.Shipment{}).Where("id = ?", search).Count(&count)
		if count > 0 {
			isPreciseSearch = true
		}
	}

	// Apply Search Logic
	if isPreciseSearch {
		// Fast Path: Exact ID match
		query = query.Where("id = ?", search)
	} else if search != "" {
		// Slow Path: Fuzzy search across multiple fields
		escapedSearch := strings.ReplaceAll(search, "%", "\\%")
		escapedSearch = strings.ReplaceAll(escapedSearch, "_", "\\_")
		query = query.Where("id LIKE ? OR origin LIKE ? OR destination LIKE ? OR bill_of_lading LIKE ? OR container_no LIKE ? OR sender_name LIKE ? OR receiver_name LIKE ? OR cargo_name LIKE ?",
			"%"+escapedSearch+"%", "%"+escapedSearch+"%", "%"+escapedSearch+"%", "%"+escapedSearch+"%", "%"+escapedSearch+"%", "%"+escapedSearch+"%", "%"+escapedSearch+"%", "%"+escapedSearch+"%")
	}

	// Data permission filter
	userID, exists := c.Get("user_id")
	if exists {
		userIDStr, ok := userID.(string)
		if ok && userIDStr != "" {
			// Skip Org Filter for precise searches (Public Tracking behavior)
			if !isPreciseSearch {
				query = services.DataPermission.ApplyOrgFilter(query, userIDStr, orgID, "org_id")
			}
		}
	}

	var shipments []models.Shipment
	if err := query.Order("created_at DESC").Find(&shipments).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 收集需要查询的组织ID
	orgIDs := make([]string, 0)
	orgIDSet := make(map[string]bool)
	for _, s := range shipments {
		if s.OrgID != nil && *s.OrgID != "" && !orgIDSet[*s.OrgID] {
			orgIDs = append(orgIDs, *s.OrgID)
			orgIDSet[*s.OrgID] = true
		}
	}

	// 批量查询组织名称
	orgNameMap := make(map[string]string)
	if len(orgIDs) > 0 {
		var orgs []models.Organization
		h.db.Where("id IN (?)", orgIDs).Find(&orgs)
		for _, org := range orgs {
			orgNameMap[org.ID] = org.Name
		}
	}

	// 构建包含组织名称和总耗时的响应
	type ShipmentWithOrgName struct {
		models.Shipment
		OrgName       string `json:"org_name"`
		TotalDuration string `json:"total_duration"`
	}
	result := make([]ShipmentWithOrgName, len(shipments))
	for i, s := range shipments {
		// 数据一致性保障：delivered状态的运单进度应为100%
		if s.Status == "delivered" && s.Progress != 100 {
			s.Progress = 100
			// 异步修复数据库中的不一致数据
			go h.db.Model(&models.Shipment{}).Where("id = ?", s.ID).Update("progress", 100)
		}
		result[i] = ShipmentWithOrgName{
			Shipment:      s,
			TotalDuration: s.GetTotalDuration(),
		}
		if s.OrgID != nil && *s.OrgID != "" {
			if name, ok := orgNameMap[*s.OrgID]; ok {
				result[i].OrgName = name
			}
		}
	}

	utils.SuccessResponse(c, result)
}

func (h *ShipmentHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var shipment models.Shipment
	if err := h.db.Preload("Device").First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	// 确保 Device 被正确加载，如果没有则尝试手动加载
	if shipment.DeviceID != nil && *shipment.DeviceID != "" && shipment.Device == nil {
		var device models.Device
		if err := h.db.First(&device, "id = ?", *shipment.DeviceID).Error; err == nil {
			shipment.Device = &device
		}
	}

	// 添加总耗时到响应
	response := gin.H{
		"shipment":       shipment,
		"total_duration": shipment.GetTotalDuration(),
	}
	utils.SuccessResponse(c, response)
}

func (h *ShipmentHandler) Create(c *gin.Context) {
	var req models.ShipmentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的运单信息")
		return
	}

	// 自动生成运单号
	idGen := services.GetShipmentIDGenerator()
	shipmentID, err := idGen.GenerateID()
	if err != nil {
		utils.InternalError(c, "生成运单号失败: "+err.Error())
		return
	}

	// 确定运单所属组织
	var shipmentOrgID *string
	userIDValue, _ := c.Get("user_id")
	userIDStr, _ := userIDValue.(string)

	// 优先使用请求中传入的组织ID
	if req.OrgID != nil && *req.OrgID != "" {
		// 验证用户是否有权限访问该组织
		visibleOrgIDs := services.DataPermission.GetVisibleOrgIDs(userIDStr)
		hasAccess := false
		for _, id := range visibleOrgIDs {
			if id == *req.OrgID {
				hasAccess = true
				break
			}
		}
		// 放宽权限检查：只要用户有关联组织且有权限访问该组织，就允许
		if hasAccess || len(visibleOrgIDs) > 0 {
			// 检查用户是否属于根组织
			isRootUser := services.DataPermission.CanAccessAllData(userIDStr)
			// 如果是根组织或者有权限访问该组织，都允许创建
			if isRootUser || hasAccess {
				shipmentOrgID = req.OrgID
			} else {
				utils.Forbidden(c, "无权创建该组织的运单")
				return
			}
		} else {
			// 没有指定组织ID，使用用户的主组织
			shipmentOrgID = req.OrgID
		}
	} else {
		// 使用用户的主组织
		if userIDStr != "" {
			var userOrg models.UserOrganization
			if err := h.db.Where("user_id = ? AND is_primary = ?", userIDStr, true).First(&userOrg).Error; err == nil {
				shipmentOrgID = &userOrg.OrganizationID
			} else if err := h.db.Where("user_id = ?", userIDStr).First(&userOrg).Error; err == nil {
				shipmentOrgID = &userOrg.OrganizationID
			}
		}
	}

	// 处理空字符串设备ID，防止外键约束错误
	if req.DeviceID != nil && *req.DeviceID == "" {
		req.DeviceID = nil
	}

	shipment := models.Shipment{
		ID:                shipmentID, // 使用自动生成的运单号
		DeviceID:          req.DeviceID,
		OrgID:             shipmentOrgID, // 使用确定的组织ID
		TransportType:     req.TransportType,
		TransportMode:     req.TransportMode,
		ContainerType:     req.ContainerType,
		CargoType:         req.CargoType,
		CargoName:         req.CargoName,
		Origin:            req.Origin,
		Destination:       req.Destination,
		OriginLat:         req.OriginLat,
		OriginLng:         req.OriginLng,
		OriginAddress:     req.OriginAddress,
		DestLat:           req.DestLat,
		DestLng:           req.DestLng,
		DestAddress:       req.DestAddress,
		DepartureTime:     req.DepartureTime,
		ETA:               req.ETA,
		Status:            "pending",
		Progress:          0,
		AutoStatusEnabled: true, // 显式设置，确保围栏检测功能默认启用
		// 发货人/收货人信息
		SenderName:    req.SenderName,
		SenderPhone:   req.SenderPhone,
		ReceiverName:  req.ReceiverName,
		ReceiverPhone: req.ReceiverPhone,
		// 关键单证 (标准跨境型)
		BillOfLading: req.BillOfLading,
		ContainerNo:  req.ContainerNo,
		SealNo:       req.SealNo, // Phase 1新增
		// 详细路由 - ETD/ATD
		ETD: req.ETD,
		ATD: req.ATD,
		// 船务信息 (Phase 1新增)
		VesselName: req.VesselName,
		VoyageNo:   req.VoyageNo,
		Carrier:    req.Carrier,
		// 订单关联 (Phase 1新增)
		PONumbers:     req.PONumbers,
		SKUIDs:        req.SKUIDs,
		FBAShipmentID: req.FBAShipmentID,
		// 货物量纲
		Pieces: req.Pieces,
		Weight: req.Weight,
		Volume: req.Volume,
		// 费用信息 (Phase 1新增)
		FreightCost: req.FreightCost,
		Surcharges:  req.Surcharges,
		CustomsFee:  req.CustomsFee,
		OtherCost:   req.OtherCost,
	}

	// 自动计算总费用
	var totalCost float64
	if req.FreightCost != nil {
		totalCost += *req.FreightCost
	}
	if req.Surcharges != nil {
		totalCost += *req.Surcharges
	}
	if req.CustomsFee != nil {
		totalCost += *req.CustomsFee
	}
	if req.OtherCost != nil {
		totalCost += *req.OtherCost
	}
	if totalCost > 0 {
		shipment.TotalCost = &totalCost
	}

	if req.DeviceID != nil && *req.DeviceID != "" {
		now := time.Now()
		shipment.DeviceBoundAt = &now
	}

	// 从系统配置读取默认围栏半径
	var originRadiusConfig models.SystemConfig
	if err := h.db.First(&originRadiusConfig, "key = ?", "default_origin_radius").Error; err == nil {
		if radius, parseErr := strconv.Atoi(originRadiusConfig.Value); parseErr == nil && radius > 0 {
			shipment.OriginRadius = radius
		}
	}
	var destRadiusConfig models.SystemConfig
	if err := h.db.First(&destRadiusConfig, "key = ?", "default_dest_radius").Error; err == nil {
		if radius, parseErr := strconv.Atoi(destRadiusConfig.Value); parseErr == nil && radius > 0 {
			shipment.DestRadius = radius
		}
	}

	if err := h.db.Create(&shipment).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 🔍 详细日志：检查接收到的字段
	log.Printf("=== 运单创建请求详情 ====")
	log.Printf("请求组织ID: %v", req.OrgID)
	log.Printf("设备ID: %v", req.DeviceID)
	log.Printf("运输类型: %v", req.TransportType)
	log.Printf("运输模式: %v", req.TransportMode)
	log.Printf("柜型: %v", req.ContainerType)
	log.Printf("货物类型: %v", req.CargoType)
	log.Printf("货物名称: %v", req.CargoName)
	log.Printf("提单号: %v", req.BillOfLading)
	log.Printf("箱号: %v", req.ContainerNo)
	log.Printf("封条号: %v", req.SealNo)
	log.Printf("船名: %v", req.VesselName)
	log.Printf("航次: %v", req.VoyageNo)
	log.Printf("承运人: %v", req.Carrier)
	log.Printf("PO单号: %v", req.PONumbers)
	log.Printf("SKU IDs: %v", req.SKUIDs)
	log.Printf("FBA ID: %v", req.FBAShipmentID)
	log.Printf("发货人: %v", req.SenderName)
	log.Printf("发货人电话: %v", req.SenderPhone)
	log.Printf("收货人: %v", req.ReceiverName)
	log.Printf("收货人电话: %v", req.ReceiverPhone)
	log.Printf("件数: %v", req.Pieces)
	log.Printf("重量: %v", req.Weight)
	log.Printf("体积: %v", req.Volume)
	log.Printf("运费: %v", req.FreightCost)
	log.Printf("附加费: %v", req.Surcharges)
	log.Printf("关税: %v", req.CustomsFee)
	log.Printf("其他费用: %v", req.OtherCost)
	log.Printf("=== 运单创建完毕 ====")
	log.Printf("创建的运单ID: %s", shipment.ID)
	log.Printf("数据库中的设备ID: %v", shipment.DeviceID)
	log.Printf("数据库中的组织ID: %v", shipment.OrgID)

	// 获取操作人信息
	operatorName, operatorIP := h.getOperatorInfo(c)

	// 记录创建日志
	services.ShipmentLog.LogCreated(shipment.ID, operatorName, operatorIP)

	// 记录设备绑定历史
	if req.DeviceID != nil && *req.DeviceID != "" {
		services.DeviceBinding.BindDevice(shipment.ID, *req.DeviceID)
		services.ShipmentLog.LogDeviceBound(shipment.ID, *req.DeviceID, operatorName, operatorIP)

		// 立即同步设备轨迹，实现绑定后即时获取位置
		if services.Scheduler != nil {
			go services.Scheduler.SyncDeviceImmediate(*req.DeviceID)
		}
	}

	// Phase 6: 自动创建5个运输环节 (改为异步执行，避免阻塞前端响应)
	if stageSvc := services.GetShipmentStageService(); stageSvc != nil {
		// 捕获变量并在协程中使用
		shipmentID := shipment.ID
		originPortCode := req.OriginPortCode
		destPortCode := req.DestPortCode
		go func() {
			if err := stageSvc.CreateStagesForShipment(shipmentID, originPortCode, destPortCode); err != nil {
				// 环节创建失败只记录日志，不影响运单创建
				fmt.Printf("Warning: Failed to create stages for shipment %s: %v\n", shipmentID, err)
			}
		}()
	}

	// 自动保存客户信息 (发货人/收货人)
	if shipment.OrgID != nil && *shipment.OrgID != "" {
		custHandler := NewCustomerHandler(h.db)
		// 保存发货人
		if shipment.SenderPhone != "" {
			go func() {
				if err := custHandler.SaveOrUpdateFromShipment(*shipment.OrgID, models.CustomerTypeSender, shipment.SenderName, shipment.SenderPhone, shipment.OriginAddress); err != nil {
					fmt.Printf("Warning: Failed to auto-save sender: %v\n", err)
				}
			}()
		}
		// 保存收货人
		if shipment.ReceiverPhone != "" {
			go func() {
				if err := custHandler.SaveOrUpdateFromShipment(*shipment.OrgID, models.CustomerTypeReceiver, shipment.ReceiverName, shipment.ReceiverPhone, shipment.DestAddress); err != nil {
					fmt.Printf("Warning: Failed to auto-save receiver: %v\n", err)
				}
			}()
		}
	}

	h.db.Preload("Device").First(&shipment, "id = ?", shipment.ID)
	utils.CreatedResponse(c, shipment)
}

func (h *ShipmentHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	// 保存原始值用于日志比较
	originalShipment := shipment

	var req models.ShipmentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求数据")
		return
	}

	updates := make(map[string]interface{})
	if req.DeviceID != nil {
		if *req.DeviceID == "" {
			updates["device_id"] = nil
		} else {
			updates["device_id"] = *req.DeviceID
		}

		// 只有当设置为非空且发生变化时才更新绑定时间
		if *req.DeviceID != "" && (shipment.DeviceID == nil || *shipment.DeviceID != *req.DeviceID) {
			now := time.Now()
			updates["device_bound_at"] = now
		}
	}
	if req.TransportType != nil {
		updates["transport_type"] = *req.TransportType
	}
	if req.TransportMode != nil {
		updates["transport_mode"] = *req.TransportMode
	}
	if req.ContainerType != nil {
		updates["container_type"] = *req.ContainerType
	}
	if req.CargoType != nil {
		updates["cargo_type"] = *req.CargoType
	}
	if req.OrgID != nil {
		updates["org_id"] = *req.OrgID
	}
	if req.CargoName != nil {
		updates["cargo_name"] = *req.CargoName
	}
	if req.Origin != nil {
		updates["origin"] = *req.Origin
	}
	if req.Destination != nil {
		updates["destination"] = *req.Destination
	}
	if req.OriginLat != nil {
		updates["origin_lat"] = *req.OriginLat
	}
	if req.OriginLng != nil {
		updates["origin_lng"] = *req.OriginLng
	}
	if req.OriginAddress != nil {
		updates["origin_address"] = *req.OriginAddress
	}
	if req.DestLat != nil {
		updates["dest_lat"] = *req.DestLat
	}
	if req.DestLng != nil {
		updates["dest_lng"] = *req.DestLng
	}
	if req.DestAddress != nil {
		updates["dest_address"] = *req.DestAddress
	}
	if req.Status != nil {
		// 验证状态转换合法性
		if !isValidStatusTransition(shipment.Status, *req.Status) {
			utils.BadRequest(c, fmt.Sprintf("无效的状态转换: %s -> %s", shipment.Status, *req.Status))
			return
		}
		updates["status"] = *req.Status
		now := time.Now()
		updates["status_updated_at"] = now

		if *req.Status == "in_transit" && shipment.LeftOriginAt == nil {
			updates["left_origin_at"] = now
			// 自动记录ATD（如果未手动设置）
			if shipment.ATD == nil {
				updates["atd"] = now
			}
		}
		// 运单签收后保留设备绑定信息（用于历史轨迹查询）
		// 设备列表通过查询非完成运单来判断"可用状态"
		if *req.Status == "delivered" {
			updates["arrived_dest_at"] = now
			updates["track_end_at"] = now // 修复: 手动改为已送达时，必须截断轨迹
			updates["progress"] = 100
			// 自动记录ATA（如果未手动设置）
			if shipment.ATA == nil {
				updates["ata"] = now
			}
		}
	}
	if req.Progress != nil {
		updates["progress"] = *req.Progress
	}

	// 关键单证 (标准跨境型)
	if req.BillOfLading != nil {
		updates["bill_of_lading"] = *req.BillOfLading
	}
	if req.ContainerNo != nil {
		updates["container_no"] = *req.ContainerNo
	}
	if req.SealNo != nil {
		updates["seal_no"] = *req.SealNo
	}

	// 详细路由 - ETD/ATD/ATA
	if req.ETD != nil {
		updates["etd"] = *req.ETD
	}
	if req.ATD != nil {
		updates["atd"] = *req.ATD
	}
	if req.ETA != nil {
		updates["eta"] = *req.ETA
	}
	if req.ATA != nil {
		updates["ata"] = *req.ATA
	}

	// 船务信息 (Phase 1新增)
	if req.VesselName != nil {
		updates["vessel_name"] = *req.VesselName
	}
	if req.VoyageNo != nil {
		updates["voyage_no"] = *req.VoyageNo
	}
	if req.Carrier != nil {
		updates["carrier"] = *req.Carrier
	}

	// 订单关联 (Phase 1新增)
	if req.PONumbers != nil {
		updates["po_numbers"] = *req.PONumbers
	}
	if req.SKUIDs != nil {
		updates["sku_ids"] = *req.SKUIDs
	}
	if req.FBAShipmentID != nil {
		updates["fba_shipment_id"] = *req.FBAShipmentID
	}

	// 货物量纲
	if req.Pieces != nil {
		updates["pieces"] = *req.Pieces
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Volume != nil {
		updates["volume"] = *req.Volume
	}

	// 费用信息 (Phase 1新增)
	costChanged := false
	if req.FreightCost != nil {
		updates["freight_cost"] = *req.FreightCost
		costChanged = true
	}
	if req.Surcharges != nil {
		updates["surcharges"] = *req.Surcharges
		costChanged = true
	}
	if req.CustomsFee != nil {
		updates["customs_fee"] = *req.CustomsFee
		costChanged = true
	}
	if req.OtherCost != nil {
		updates["other_cost"] = *req.OtherCost
		costChanged = true
	}

	// 如果任何费用字段变化，自动重新计算总费用
	if costChanged {
		var totalCost float64
		// 使用新值或原值
		if req.FreightCost != nil {
			totalCost += *req.FreightCost
		} else if originalShipment.FreightCost != nil {
			totalCost += *originalShipment.FreightCost
		}
		if req.Surcharges != nil {
			totalCost += *req.Surcharges
		} else if originalShipment.Surcharges != nil {
			totalCost += *originalShipment.Surcharges
		}
		if req.CustomsFee != nil {
			totalCost += *req.CustomsFee
		} else if originalShipment.CustomsFee != nil {
			totalCost += *originalShipment.CustomsFee
		}
		if req.OtherCost != nil {
			totalCost += *req.OtherCost
		} else if originalShipment.OtherCost != nil {
			totalCost += *originalShipment.OtherCost
		}
		updates["total_cost"] = totalCost
	} else if req.TotalCost != nil {
		// 如果直接传了total_cost，使用传入的值
		updates["total_cost"] = *req.TotalCost
	}

	if err := h.db.Model(&shipment).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 获取操作人信息
	operatorName, operatorIP := h.getOperatorInfo(c)

	// ===== 记录所有字段变更日志 =====
	logFieldChange := func(field string, oldVal, newVal interface{}) {
		oldStr := fmt.Sprintf("%v", oldVal)
		newStr := fmt.Sprintf("%v", newVal)
		// 处理nil值和空指针
		if oldVal == nil {
			oldStr = ""
		}
		if newVal == nil {
			newStr = ""
		}
		// 处理time.Time类型
		if t, ok := oldVal.(time.Time); ok {
			if t.IsZero() {
				oldStr = ""
			} else {
				oldStr = t.Format("2006-01-02 15:04:05")
			}
		}
		if t, ok := oldVal.(*time.Time); ok && t != nil {
			oldStr = t.Format("2006-01-02 15:04:05")
		} else if t, ok := oldVal.(*time.Time); ok && t == nil {
			oldStr = ""
		}
		if t, ok := newVal.(time.Time); ok {
			if t.IsZero() {
				newStr = ""
			} else {
				newStr = t.Format("2006-01-02 15:04:05")
			}
		}
		// 处理float数字，格式化为2位小数
		if f, ok := oldVal.(float64); ok {
			oldStr = strconv.FormatFloat(f, 'f', 2, 64)
		}
		if f, ok := oldVal.(*float64); ok && f != nil {
			oldStr = strconv.FormatFloat(*f, 'f', 2, 64)
		}
		if f, ok := newVal.(float64); ok {
			newStr = strconv.FormatFloat(f, 'f', 2, 64)
		}

		if oldStr != newStr {
			services.ShipmentLog.LogFieldUpdated(id, field, oldStr, newStr, operatorName, operatorIP)
		}
	}

	// 比较并记录各字段变化（排除特殊处理的字段：status和device_id）
	if req.TransportType != nil && *req.TransportType != originalShipment.TransportType {
		logFieldChange("transport_type", originalShipment.TransportType, *req.TransportType)
	}
	if req.CargoName != nil && *req.CargoName != originalShipment.CargoName {
		logFieldChange("cargo_name", originalShipment.CargoName, *req.CargoName)
	}
	if req.Origin != nil && *req.Origin != originalShipment.Origin {
		logFieldChange("origin", originalShipment.Origin, *req.Origin)
	}
	if req.Destination != nil && *req.Destination != originalShipment.Destination {
		logFieldChange("destination", originalShipment.Destination, *req.Destination)
	}
	// 地址类字段 (string类型)
	if req.OriginAddress != nil && *req.OriginAddress != originalShipment.OriginAddress {
		logFieldChange("origin_address", originalShipment.OriginAddress, *req.OriginAddress)
	}
	if req.DestAddress != nil && *req.DestAddress != originalShipment.DestAddress {
		logFieldChange("dest_address", originalShipment.DestAddress, *req.DestAddress)
	}
	// 单证信息 (string类型)
	if req.BillOfLading != nil && *req.BillOfLading != originalShipment.BillOfLading {
		logFieldChange("bill_of_lading", originalShipment.BillOfLading, *req.BillOfLading)
	}
	if req.ContainerNo != nil && *req.ContainerNo != originalShipment.ContainerNo {
		logFieldChange("container_no", originalShipment.ContainerNo, *req.ContainerNo)
	}
	if req.SealNo != nil && *req.SealNo != originalShipment.SealNo {
		logFieldChange("seal_no", originalShipment.SealNo, *req.SealNo)
	}
	// 船务信息 (string类型)
	if req.VesselName != nil && *req.VesselName != originalShipment.VesselName {
		logFieldChange("vessel_name", originalShipment.VesselName, *req.VesselName)
	}
	if req.VoyageNo != nil && *req.VoyageNo != originalShipment.VoyageNo {
		logFieldChange("voyage_no", originalShipment.VoyageNo, *req.VoyageNo)
	}
	if req.Carrier != nil && *req.Carrier != originalShipment.Carrier {
		logFieldChange("carrier", originalShipment.Carrier, *req.Carrier)
	}
	// 订单关联 (string类型)
	if req.PONumbers != nil && *req.PONumbers != originalShipment.PONumbers {
		logFieldChange("po_numbers", originalShipment.PONumbers, *req.PONumbers)
	}
	if req.SKUIDs != nil && *req.SKUIDs != originalShipment.SKUIDs {
		logFieldChange("sku_ids", originalShipment.SKUIDs, *req.SKUIDs)
	}
	if req.FBAShipmentID != nil && *req.FBAShipmentID != originalShipment.FBAShipmentID {
		logFieldChange("fba_shipment_id", originalShipment.FBAShipmentID, *req.FBAShipmentID)
	}
	// 货物量纲
	if req.Pieces != nil {
		oldVal := 0
		if originalShipment.Pieces != nil {
			oldVal = *originalShipment.Pieces
		}
		if *req.Pieces != oldVal {
			logFieldChange("pieces", oldVal, *req.Pieces)
		}
	}
	if req.Weight != nil {
		oldVal := 0.0
		if originalShipment.Weight != nil {
			oldVal = *originalShipment.Weight
		}
		if *req.Weight != oldVal {
			logFieldChange("weight", oldVal, *req.Weight)
		}
	}
	if req.Volume != nil {
		oldVal := 0.0
		if originalShipment.Volume != nil {
			oldVal = *originalShipment.Volume
		}
		if *req.Volume != oldVal {
			logFieldChange("volume", oldVal, *req.Volume)
		}
	}
	// 费用信息
	if req.FreightCost != nil {
		oldVal := 0.0
		if originalShipment.FreightCost != nil {
			oldVal = *originalShipment.FreightCost
		}
		if *req.FreightCost != oldVal {
			logFieldChange("freight_cost", oldVal, *req.FreightCost)
		}
	}
	if req.Surcharges != nil {
		oldVal := 0.0
		if originalShipment.Surcharges != nil {
			oldVal = *originalShipment.Surcharges
		}
		if *req.Surcharges != oldVal {
			logFieldChange("surcharges", oldVal, *req.Surcharges)
		}
	}
	if req.CustomsFee != nil {
		oldVal := 0.0
		if originalShipment.CustomsFee != nil {
			oldVal = *originalShipment.CustomsFee
		}
		if *req.CustomsFee != oldVal {
			logFieldChange("customs_fee", oldVal, *req.CustomsFee)
		}
	}
	if req.OtherCost != nil {
		oldVal := 0.0
		if originalShipment.OtherCost != nil {
			oldVal = *originalShipment.OtherCost
		}
		if *req.OtherCost != oldVal {
			logFieldChange("other_cost", oldVal, *req.OtherCost)
		}
	}
	if req.TotalCost != nil {
		oldVal := 0.0
		if originalShipment.TotalCost != nil {
			oldVal = *originalShipment.TotalCost
		}
		if *req.TotalCost != oldVal {
			logFieldChange("total_cost", oldVal, *req.TotalCost)
		}
	}
	// 时间信息
	if req.ETD != nil {
		var oldTime *time.Time = originalShipment.ETD
		oldStr := ""
		newStr := req.ETD.Format("2006-01-02 15:04:05")
		if oldTime != nil {
			oldStr = oldTime.Format("2006-01-02 15:04:05")
		}
		if oldStr != newStr {
			services.ShipmentLog.LogFieldUpdated(id, "etd", oldStr, newStr, operatorName, operatorIP)
		}
	}
	if req.ETA != nil {
		var oldTime *time.Time = originalShipment.ETA
		oldStr := ""
		newStr := req.ETA.Format("2006-01-02 15:04:05")
		if oldTime != nil {
			oldStr = oldTime.Format("2006-01-02 15:04:05")
		}
		if oldStr != newStr {
			services.ShipmentLog.LogFieldUpdated(id, "eta", oldStr, newStr, operatorName, operatorIP)
		}
	}
	if req.ATD != nil {
		var oldTime *time.Time = originalShipment.ATD
		oldStr := ""
		newStr := req.ATD.Format("2006-01-02 15:04:05")
		if oldTime != nil {
			oldStr = oldTime.Format("2006-01-02 15:04:05")
		}
		if oldStr != newStr {
			services.ShipmentLog.LogFieldUpdated(id, "atd", oldStr, newStr, operatorName, operatorIP)
		}
	}
	if req.ATA != nil {
		var oldTime *time.Time = originalShipment.ATA
		oldStr := ""
		newStr := req.ATA.Format("2006-01-02 15:04:05")
		if oldTime != nil {
			oldStr = oldTime.Format("2006-01-02 15:04:05")
		}
		if oldStr != newStr {
			services.ShipmentLog.LogFieldUpdated(id, "ata", oldStr, newStr, operatorName, operatorIP)
		}
	}

	// 记录状态变更日志（原有逻辑）
	if req.Status != nil && *req.Status != originalShipment.Status {
		services.ShipmentLog.LogStatusChanged(id, originalShipment.Status, *req.Status, operatorName, operatorIP)

		// 运单完成时，解绑设备并记录
		if *req.Status == "delivered" || *req.Status == "cancelled" {
			services.DeviceBinding.UnbindDevice(id, "completed")
			if originalShipment.DeviceID != nil && *originalShipment.DeviceID != "" {
				services.ShipmentLog.LogDeviceUnbound(id, *originalShipment.DeviceID, "completed", operatorName, operatorIP)
			}
		}
	}

	// 记录设备变更（原有逻辑）
	if req.DeviceID != nil {
		oldDeviceID := ""
		if originalShipment.DeviceID != nil {
			oldDeviceID = *originalShipment.DeviceID
		}
		newDeviceID := *req.DeviceID

		if oldDeviceID != newDeviceID {
			if oldDeviceID != "" && newDeviceID != "" {
				// 更换设备
				services.DeviceBinding.BindDevice(id, newDeviceID)
				services.ShipmentLog.LogDeviceReplaced(id, oldDeviceID, newDeviceID, operatorName, operatorIP)
			} else if newDeviceID != "" {
				// 新绑定设备
				services.DeviceBinding.BindDevice(id, newDeviceID)
				services.ShipmentLog.LogDeviceBound(id, newDeviceID, operatorName, operatorIP)
			} else if oldDeviceID != "" {
				// 手动解绑
				services.DeviceBinding.UnbindDevice(id, "manual")
				services.ShipmentLog.LogDeviceUnbound(id, oldDeviceID, "manual", operatorName, operatorIP)
			}

			// 绑定新设备后立即同步轨迹
			if newDeviceID != "" && services.Scheduler != nil {
				go services.Scheduler.SyncDeviceImmediate(newDeviceID)
			}
		}
	}

	// Phase 6 Check: 如果关键字段(运输方式/起运地/目的地)发生变化，自动重新规划路线
	shouldRegenerateStages := false
	// 检查运输方式变更
	if req.TransportType != nil && *req.TransportType != originalShipment.TransportType {
		shouldRegenerateStages = true
	}
	// 检查起运地变更 (城市或详细地址)
	if (req.Origin != nil && *req.Origin != originalShipment.Origin) ||
		(req.OriginAddress != nil && *req.OriginAddress != originalShipment.OriginAddress) {
		shouldRegenerateStages = true
	}
	// 检查目的地变更 (城市或详细地址)
	if (req.Destination != nil && *req.Destination != originalShipment.Destination) ||
		(req.DestAddress != nil && *req.DestAddress != originalShipment.DestAddress) {
		shouldRegenerateStages = true
	}

	if shouldRegenerateStages {
		if stageSvc := services.GetShipmentStageService(); stageSvc != nil {
			shipmentID := shipment.ID
			// 只有非完成/取消状态的运单才重新规划，防止破坏历史记录
			if shipment.Status != "delivered" && shipment.Status != "cancelled" {
				go func() {
					// 重新生成只需ID，它会重新读取最新的运单信息
					if err := stageSvc.RegenerateStages(shipmentID); err != nil {
						fmt.Printf("Warning: Failed to auto-regenerate stages for shipment %s: %v\n", shipmentID, err)
					} else {
						fmt.Printf("Info: Auto-regenerated stages for shipment %s due to update\n", shipmentID)
					}
				}()
			}
		}
	}

	h.db.Preload("Device").First(&shipment, "id = ?", id)
	utils.SuccessResponse(c, shipment)
}

func (h *ShipmentHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	// 检查运单是否存在
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	// 检查运单状态 - 运输中的运单不允许删除
	if shipment.Status == "in_transit" {
		utils.BadRequest(c, "运输中的运单无法删除，请先取消运单")
		return
	}

	if err := h.db.Delete(&models.Shipment{}, "id = ?", id).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 记录删除日志
	operatorName, operatorIP := h.getOperatorInfo(c)
	services.ShipmentLog.LogDeleted(id, operatorName, operatorIP)

	utils.SuccessResponse(c, gin.H{"success": true})
}

// TransitionStatus 快捷切换运单状态
// POST /api/shipments/:id/transition
// Body: { "action": "depart" | "deliver" | "cancel" }
func (h *ShipmentHandler) TransitionStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Action   string `json:"action" binding:"required"` // depart, deliver, cancel
		Receiver string `json:"receiver"`                  // 签收人姓名（签收时使用）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请指定操作类型: depart(发车), deliver(签收), cancel(取消)")
		return
	}

	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	now := time.Now()
	updates := make(map[string]interface{})
	var newStatus string
	var logAction string
	oldStatus := shipment.Status // 保存原始状态用于日志记录

	switch req.Action {
	case "depart":
		// 发车：pending -> in_transit
		if shipment.Status != "pending" {
			utils.BadRequest(c, fmt.Sprintf("当前状态为 %s，无法执行发车操作", shipment.Status))
			return
		}
		newStatus = "in_transit"
		updates["status"] = newStatus
		updates["progress"] = 50 // 发车后进度设为50%
		updates["left_origin_at"] = now
		updates["status_updated_at"] = now
		updates["current_milestone"] = "departed"
		logAction = "发车"

	case "deliver":
		// 签收：in_transit -> delivered
		if shipment.Status != "in_transit" {
			utils.BadRequest(c, fmt.Sprintf("当前状态为 %s，无法执行签收操作", shipment.Status))
			return
		}
		newStatus = "delivered"
		updates["status"] = newStatus
		updates["progress"] = 100
		updates["arrived_dest_at"] = now
		updates["status_updated_at"] = now
		updates["current_milestone"] = "delivered"
		logAction = "签收"

	case "cancel":
		// 取消：pending/in_transit -> cancelled
		if shipment.Status == "delivered" || shipment.Status == "cancelled" {
			utils.BadRequest(c, fmt.Sprintf("当前状态为 %s，无法取消", shipment.Status))
			return
		}
		newStatus = "cancelled"
		updates["status"] = newStatus
		updates["status_updated_at"] = now
		updates["current_milestone"] = "cancelled"
		logAction = "取消"

	default:
		utils.BadRequest(c, "无效的操作类型，可选: depart, deliver, cancel")
		return
	}

	if err := h.db.Model(&shipment).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 记录状态变更日志
	operatorName, operatorIP := h.getOperatorInfo(c)
	services.ShipmentLog.LogStatusChanged(id, oldStatus, newStatus, operatorName, operatorIP)

	// 特殊处理：签收或取消时进行收尾工作
	if req.Action == "deliver" || req.Action == "cancel" {
		// 1. 设置轨迹结束时间
		h.db.Model(&shipment).Update("track_end_at", now)

		// 2. 签收且有接收人时记录
		if req.Action == "deliver" && req.Receiver != "" {
			services.ShipmentLog.LogDelivered(id, req.Receiver, operatorName, operatorIP)
		}

		// 3. 自动解绑设备（如果是签收或取消，且运单绑定了设备）
		if shipment.DeviceID != nil && *shipment.DeviceID != "" {
			// 解绑设备绑定记录
			if services.DeviceBinding != nil {
				services.DeviceBinding.UnbindDevice(id, "manual_"+req.Action)
			}
			// 更新运单表的设备绑定信息
			h.db.Model(&shipment).Updates(map[string]interface{}{
				"unbound_device_id": *shipment.DeviceID,
				"device_id":         nil,
				"device_unbound_at": now,
			})
			// 记录解绑日志
			services.ShipmentLog.LogDeviceUnbound(id, *shipment.DeviceID, "manual_"+req.Action, operatorName, operatorIP)
		}

		// 4. 重置/关闭相关预警
		if services.AlertChecker != nil {
			services.AlertChecker.CheckAllAlerts(&shipment) // 重新检查以触发关闭逻辑
		}
	}

	// 返回更新后的运单
	h.db.Preload("Device").First(&shipment, "id = ?", id)

	utils.SuccessResponse(c, gin.H{
		"shipment": shipment,
		"message":  fmt.Sprintf("运单已%s", logAction),
	})
}

// Helper structs for API response
type LocationPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type RouteResponse struct {
	Origin          *LocationPoint  `json:"origin"`
	Destination     *LocationPoint  `json:"destination"`
	CurrentLocation *LocationPoint  `json:"current_location"`
	Points          []LocationPoint `json:"points"`
}

func (h *ShipmentHandler) GetRoute(c *gin.Context) {
	id := c.Param("id")

	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	resp := RouteResponse{
		Points: []LocationPoint{},
	}

	// 添加起点
	if shipment.OriginLat != nil && shipment.OriginLng != nil {
		resp.Points = append([]LocationPoint{
			{
				Latitude:  *shipment.OriginLat,
				Longitude: *shipment.OriginLng,
			},
		}, resp.Points...)
		resp.Origin = &LocationPoint{
			Latitude:  *shipment.OriginLat,
			Longitude: *shipment.OriginLng,
		}
	}

	// 添加当前位置（如果设备在线）
	if shipment.Device != nil && shipment.Device.Status == "online" {
		if shipment.Device.Latitude != nil && shipment.Device.Longitude != nil {
			resp.Points = append(resp.Points, LocationPoint{
				Latitude:  *shipment.Device.Latitude,
				Longitude: *shipment.Device.Longitude,
			})
			resp.CurrentLocation = &LocationPoint{
				Latitude:  *shipment.Device.Latitude,
				Longitude: *shipment.Device.Longitude,
			}
		}
	}

	// 添加目的地
	if shipment.DestLat != nil && shipment.DestLng != nil {
		resp.Points = append(resp.Points, LocationPoint{
			Latitude:  *shipment.DestLat,
			Longitude: *shipment.DestLng,
		})
		resp.Destination = &LocationPoint{
			Latitude:  *shipment.DestLat,
			Longitude: *shipment.DestLng,
		}
	}

	utils.SuccessResponse(c, resp)
}
func (h *ShipmentHandler) CheckStatus(c *gin.Context) {
	id := c.Param("id")

	var shipment models.Shipment
	if err := h.db.Preload("Device").First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	if shipment.DeviceID == nil || *shipment.DeviceID == "" {
		utils.SuccessResponse(c, gin.H{
			"status":  shipment.Status,
			"changed": false,
			"message": "无法判断状态：设备未绑定",
		})
		return
	}

	// 记录原始状态
	oldStatus := shipment.Status

	// 委托给 GeofenceService 进行检测和状态更新
	// 这确保了所有副作用（如设置 track_end_at、自动解绑、日志记录）都能正确触发
	if services.Geofence != nil && shipment.Device != nil && shipment.Device.Latitude != nil && shipment.Device.Longitude != nil {
		services.Geofence.CheckAndUpdateStatus(*shipment.DeviceID, *shipment.Device.Latitude, *shipment.Device.Longitude)
	}

	// 重新加载运单以获取最新状态
	h.db.First(&shipment, "id = ?", id)

	changed := oldStatus != shipment.Status

	utils.SuccessResponse(c, gin.H{
		"status":     shipment.Status,
		"changed":    changed,
		"old_status": oldStatus,
	})
}

func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// SeedShipments 创建示例运单数据
func SeedShipments(db *gorm.DB) error {
	var count int64
	db.Model(&models.Shipment{}).Count(&count)
	if count > 0 {
		return nil
	}

	// 示例运单数据
	originLat1, originLng1 := 31.2304, 121.4737
	destLat1, destLng1 := 40.7128, -74.006
	originLat2, originLng2 := 22.5431, 114.0579
	destLat2, destLng2 := 51.5074, -0.1278
	originLat3, originLng3 := 39.9042, 116.4074
	destLat3, destLng3 := 35.6762, 139.6503

	now := time.Now()
	eta1 := now.Add(15 * 24 * time.Hour)
	eta2 := now.Add(20 * 24 * time.Hour)
	eta3 := now.Add(5 * 24 * time.Hour)

	shipments := []models.Shipment{
		{
			ID:            "SHN-SH-0001",
			TransportType: "sea",
			CargoName:     "电子产品",
			Origin:        "上海, 中国",
			Destination:   "纽约, 美国",
			OriginLat:     &originLat1,
			OriginLng:     &originLng1,
			DestLat:       &destLat1,
			DestLng:       &destLng1,
			Status:        "in_transit",
			Progress:      45,
			ETA:           &eta1,
			BillOfLading:  "MAEU123456789",
			ContainerNo:   "MSKU1234567",
		},
		{
			ID:            "SHN-SZ-0002",
			TransportType: "sea",
			CargoName:     "服装纺织品",
			Origin:        "深圳, 中国",
			Destination:   "伦敦, 英国",
			OriginLat:     &originLat2,
			OriginLng:     &originLng2,
			DestLat:       &destLat2,
			DestLng:       &destLng2,
			Status:        "pending",
			Progress:      0,
			ETA:           &eta2,
			BillOfLading:  "COSCO987654321",
			ContainerNo:   "COSU7654321",
		},
		{
			ID:            "SHN-BJ-0003",
			TransportType: "air",
			CargoName:     "精密仪器",
			Origin:        "北京, 中国",
			Destination:   "东京, 日本",
			OriginLat:     &originLat3,
			OriginLng:     &originLng3,
			DestLat:       &destLat3,
			DestLng:       &destLng3,
			Status:        "in_transit",
			Progress:      80,
			ETA:           &eta3,
			BillOfLading:  "AIR2024001234",
			ContainerNo:   "",
		},
	}

	for _, shipment := range shipments {
		db.Create(&shipment)
	}

	return nil
}

// GetLogs 获取运单操作日志
func (h *ShipmentHandler) GetLogs(c *gin.Context) {
	id := c.Param("id")

	// 检查运单是否存在
	var count int64
	h.db.Model(&models.Shipment{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		utils.NotFound(c, "运单不存在")
		return
	}

	logs := services.ShipmentLog.GetLogs(id)
	utils.SuccessResponse(c, logs)
}

// GetBindingHistory 获取运单设备绑定历史
func (h *ShipmentHandler) GetBindingHistory(c *gin.Context) {
	id := c.Param("id")

	// 检查运单是否存在
	var count int64
	h.db.Model(&models.Shipment{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		utils.NotFound(c, "运单不存在")
		return
	}

	bindings := services.DeviceBinding.GetBindingHistory(id)
	utils.SuccessResponse(c, bindings)
}

// GetCarrierTracks 获取运单的船司追踪事件 (Phase 2)
func (h *ShipmentHandler) GetCarrierTracks(c *gin.Context) {
	id := c.Param("id")

	// 检查运单是否存在
	var count int64
	h.db.Model(&models.Shipment{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		utils.NotFound(c, "运单不存在")
		return
	}

	if services.CarrierTracking == nil {
		utils.SuccessResponse(c, []models.CarrierTrack{})
		return
	}

	tracks, err := services.CarrierTracking.GetShipmentTracks(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, tracks)
}

// GetMilestones 获取运单的统一里程碑 (Phase 2)
func (h *ShipmentHandler) GetMilestones(c *gin.Context) {
	id := c.Param("id")

	// 检查运单是否存在
	var count int64
	h.db.Model(&models.Shipment{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		utils.NotFound(c, "运单不存在")
		return
	}

	if services.CarrierTracking == nil {
		utils.SuccessResponse(c, []models.ShipmentMilestone{})
		return
	}

	milestones, err := services.CarrierTracking.GetShipmentMilestones(id)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, milestones)
}

// SyncCarrierTrack 手动触发运单船司数据同步 (Phase 2)
func (h *ShipmentHandler) SyncCarrierTrack(c *gin.Context) {
	id := c.Param("id")

	// 获取运单
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	if shipment.TransportType != "sea" || shipment.BillOfLading == "" {
		utils.BadRequest(c, "该运单不是海运或缺少提单号")
		return
	}

	if services.CarrierTracking == nil || !services.CarrierTracking.IsConfigured() {
		utils.BadRequest(c, "船司追踪服务未配置")
		return
	}

	tracks, err := services.CarrierTracking.TrackShipment(&shipment)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"synced_events": len(tracks),
		"message":       fmt.Sprintf("成功同步 %d 个船务事件", len(tracks)),
	})
}

// GetShipmentTracks 获取运单关联设备的轨迹数据
// 支持多设备轨迹合并（处理中途换绑设备的情况）
func (h *ShipmentHandler) GetShipmentTracks(c *gin.Context) {
	id := c.Param("id")

	// 解析时间范围参数
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var shipment models.Shipment
	if err := h.db.Preload("Device").First(&shipment, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	// 1. 获取设备绑定历史
	if services.DeviceBinding == nil {
		utils.InternalError(c, "设备绑定服务未初始化")
		return
	}
	bindings := services.DeviceBinding.GetBindingHistory(id)

	// 如果没有绑定历史（可能是旧数据），尝试构建默认绑定
	if len(bindings) == 0 {
		if shipment.DeviceID != nil && *shipment.DeviceID != "" {
			boundAt := shipment.CreatedAt
			if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() {
				boundAt = *shipment.DeviceBoundAt
			}

			// 检查设备是否被其他运单使用，避免显示无关轨迹
			// 查找同一设备的其他运单，排除当前运单
			var otherShipments []models.Shipment
			h.db.Where("device_id = ? AND id != ? AND deleted_at IS NULL", *shipment.DeviceID, shipment.ID).
				Find(&otherShipments)

			// 如果设备被其他运单使用，找出最后一个已完成运单的track_end_at作为起始时间
			earliestStart := boundAt
			for _, other := range otherShipments {
				if other.TrackEndAt != nil && !other.TrackEndAt.IsZero() {
					// 如果其他运单有track_end_at，且在boundAt之后，使用较晚的时间
					if other.TrackEndAt.After(earliestStart) {
						earliestStart = *other.TrackEndAt
					}
				}
			}

			bindings = append(bindings, models.ShipmentDeviceBinding{
				DeviceID: *shipment.DeviceID,
				BoundAt:  earliestStart, // 使用修正后的起始时间
				// UnboundAt is null (active)
			})
		} else if shipment.UnboundDeviceID != nil && *shipment.UnboundDeviceID != "" {
			// 对于已结束的旧运单，如果只有unbound信息
			bindings = append(bindings, models.ShipmentDeviceBinding{
				DeviceID:  *shipment.UnboundDeviceID,
				BoundAt:   shipment.CreatedAt,       // 估算
				UnboundAt: shipment.DeviceUnboundAt, // 实际解绑时间
			})
		}
	}

	if len(bindings) == 0 {
		utils.BadRequest(c, "该运单未绑定或未曾绑定设备")
		return
	}

	// 检查设备是否被其他运单使用，避免显示无关轨迹
	// 策略：使用运单自己的绑定窗口作为主时间范围，避免 left_origin_at 过晚导致前段轨迹被截断
	// 1. 默认从运单创建时间开始
	// 2. 如存在 device_bound_at，则使用更晚的绑定时间
	// 3. 仅在没有绑定时间时才退化到 left_origin_at
	// 4. 如有 track_end_at，后续会在分段内限制截止时间
	var queryStartTime time.Time
	queryStartTime = shipment.CreatedAt
	if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() && shipment.DeviceBoundAt.After(queryStartTime) {
		queryStartTime = *shipment.DeviceBoundAt
	} else if (shipment.DeviceBoundAt == nil || shipment.DeviceBoundAt.IsZero()) &&
		shipment.LeftOriginAt != nil && !shipment.LeftOriginAt.IsZero() &&
		shipment.LeftOriginAt.After(queryStartTime) {
		queryStartTime = *shipment.LeftOriginAt
	}

	// 注意：不再检查其他运单的track_end_at
	// 原因：设备绑定管理混乱，多个运单可能同时使用同一设备
	// 使用left_origin_at可以确保显示当前运单自己的轨迹
	// 如果需要更精确的控制，应该改进设备绑定管理，确保同一时刻只有一个运单使用设备

	fallbackStartTime := shipment.CreatedAt
	if shipment.DeviceBoundAt != nil && !shipment.DeviceBoundAt.IsZero() && shipment.DeviceBoundAt.After(fallbackStartTime) {
		fallbackStartTime = *shipment.DeviceBoundAt
	}

	// 2. 遍历绑定历史，查询分段轨迹
	var allTracks []models.DeviceTrack

	// 解析请求的时间范围
	var reqStartTime, reqEndTime time.Time
	if startTimeStr != "" {
		reqStartTime, _ = time.Parse("2006-01-02 15:04:05", startTimeStr)
	}
	if endTimeStr != "" {
		reqEndTime, _ = time.Parse("2006-01-02 15:04:05", endTimeStr)
	}

	for _, binding := range bindings {
		// 确定当前分段的时间窗口
		segmentStart := queryStartTime // 使用计算出的起始时间
		bindingStart := resolveBindingStartTime(&shipment, binding)
		if bindingStart.After(segmentStart) {
			segmentStart = bindingStart
		}
		segmentEnd := time.Now()
		if binding.UnboundAt != nil {
			segmentEnd = *binding.UnboundAt
		}

		// 与请求时间范围取交集
		queryStart := segmentStart
		if !reqStartTime.IsZero() && reqStartTime.After(queryStart) {
			queryStart = reqStartTime
		}

		queryEnd := segmentEnd
		if !reqEndTime.IsZero() && reqEndTime.Before(queryEnd) {
			queryEnd = reqEndTime
		}
		// 运单结束时间限制
		if shipment.TrackEndAt != nil && shipment.TrackEndAt.Before(queryEnd) {
			shouldApplyTrackEnd := true
			if endTimeStr == "" &&
				(shipment.Status == "delivered" || shipment.Status == "cancelled") &&
				(binding.UnboundAt == nil || binding.UnboundAt.IsZero()) {
				// 已签收但设备仍保持绑定时，允许继续展示硬件回传轨迹，避免 track_end_at 过早导致整段为空
				shouldApplyTrackEnd = false
			}
			if shouldApplyTrackEnd {
				queryEnd = *shipment.TrackEndAt
			}
		}

		// 如果时间窗口无效，跳过
		if queryStart.After(queryEnd) {
			continue
		}

		// 查询该设备的轨迹
		var segmentTracks []models.DeviceTrack
		// 获取对应的internal ID
		var device models.Device
		if err := h.db.Select("id").First(&device, "id = ?", binding.DeviceID).Error; err != nil {
			continue
		}

		loadTracks := func(start, end time.Time) []models.DeviceTrack {
			// 在读取前做一次增量补拉，确保轨迹点可持续增长（适用于所有运单）
			if services.Scheduler != nil {
				if _, err := services.Scheduler.SyncDeviceTrack(device.ID, start, end); err != nil {
					// 同步失败不阻塞主流程，继续返回本地已有轨迹
					log.Printf("[ShipmentTracks] 增量同步失败 shipment=%s device=%s: %v", id, device.ID, err)
				}
			}

			var tracks []models.DeviceTrack
			h.db.Where("device_id = ? AND locate_time >= ? AND locate_time <= ?", device.ID, start, end).
				Order("locate_time ASC").
				Find(&tracks)
			return tracks
		}

		// 在读取前做一次增量补拉，确保轨迹点可持续增长（适用于所有运单）
		segmentTracks = loadTracks(queryStart, queryEnd)

		if len(segmentTracks) == 0 &&
			startTimeStr == "" &&
			shipment.LeftOriginAt != nil &&
			!shipment.LeftOriginAt.IsZero() &&
			fallbackStartTime.Before(queryStart) {
			fallbackQueryStart := fallbackStartTime
			if !reqStartTime.IsZero() && reqStartTime.After(fallbackQueryStart) {
				fallbackQueryStart = reqStartTime
			}
			if !fallbackQueryStart.After(queryEnd) {
				segmentTracks = loadTracks(fallbackQueryStart, queryEnd)
			}
		}

		allTracks = append(allTracks, segmentTracks...)
	}

	// 数据库查询已有序，且绑定时间一般不重叠，直接append通常有序。
	// 如果确实发现了乱序情况再加sort，目前保持性能优先。

	// G-4: 截断时通知前端
	totalAvailable := len(allTracks)
	truncated := false
	if totalAvailable > 10000 {
		allTracks = allTracks[:10000]
		truncated = true
	}

	// 4. 构建current_position（使用当前绑定设备的最新位置，如果没有则为空）
	var currentPosition interface{} = nil
	currentBinding := services.DeviceBinding.GetCurrentBinding(id)
	if currentBinding != nil {
		var currentDevice models.Device
		if err := h.db.First(&currentDevice, "id = ?", currentBinding.DeviceID).Error; err == nil {
			if currentDevice.Latitude != nil && currentDevice.Longitude != nil && *currentDevice.Latitude != 0 && *currentDevice.Longitude != 0 {
				currentPosition = gin.H{
					"lat":       *currentDevice.Latitude,
					"lng":       *currentDevice.Longitude,
					"speed":     currentDevice.Speed,
					"timestamp": currentDevice.LastUpdate,
				}
			}
		}
	}

	// 转换轨迹数据格式以匹配前端期望（lat/lng而不是latitude/longitude）
	formattedTracks := make([]gin.H, 0, len(allTracks))
	for _, t := range allTracks {
		formattedTracks = append(formattedTracks, gin.H{
			"id":          t.ID,
			"device_id":   t.DeviceID,
			"lat":         t.Latitude,
			"lng":         t.Longitude,
			"speed":       t.Speed,
			"direction":   t.Direction,
			"temperature": t.Temperature,
			"humidity":    t.Humidity,
			"locate_time": t.LocateTime,
		})
	}

	// 构建设备ID列表（用于前端参考）
	deviceIDs := make([]string, 0, len(bindings))
	for _, b := range bindings {
		deviceIDs = append(deviceIDs, b.DeviceID)
	}

	utils.SuccessResponse(c, gin.H{
		"shipment_id":      id,
		"device_ids":       deviceIDs, // 返回所有关联过的设备ID
		"tracks":           formattedTracks,
		"current_position": currentPosition,
		"truncated":        truncated,
		"total_available":  totalAvailable,
	})
}
