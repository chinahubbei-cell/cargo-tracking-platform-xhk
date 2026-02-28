package handlers

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

type DeviceHandler struct {
	db *gorm.DB
}

// DeviceWithBinding 设备响应（包含绑定运单信息）
type DeviceWithBinding struct {
	models.Device
	BindingStatus      string  `json:"binding_status"`       // 绑定状态: bound/unbound
	BoundTransportType string  `json:"bound_transport_type"` // 绑定运单的运输类型
	BoundCargoName     string  `json:"bound_cargo_name"`     // 绑定运单的货物名称
	BoundShipmentID    *string `json:"bound_shipment_id"`    // 最新绑定的运单ID
	OrgName            string  `json:"org_name"`             // 组织名称
}

func NewDeviceHandler(db *gorm.DB) *DeviceHandler {
	return &DeviceHandler{db: db}
}

func (h *DeviceHandler) List(c *gin.Context) {
	status := c.Query("status")
	deviceType := c.Query("type")
	search := c.Query("search")
	syncExternal := c.Query("syncExternal")
	orgID := c.Query("org_id") // 按组织筛选

	query := h.db.Model(&models.Device{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if deviceType != "" {
		query = query.Where("type = ?", deviceType)
	}
	if search != "" {
		query = query.Where("id LIKE ? OR name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 数据权限过滤
	userID, exists := c.Get("user_id")
	if exists {
		userIDStr, ok := userID.(string)
		if ok && userIDStr != "" {
			query = services.DataPermission.ApplyOrgFilter(query, userIDStr, orgID, "org_id")
		}
	}

	var devices []models.Device
	if err := query.Order("last_update DESC").Find(&devices).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 如果需要同步外部数据
	if syncExternal == "true" {
		devices = h.syncExternalDeviceData(devices)
	}

	// 定义完成状态（不再绑定的状态）
	completedStatuses := []string{"delivered", "cancelled"}

	// 收集所有设备ID和外部设备ID用于批量查询绑定运单
	deviceIDs := make([]string, 0, len(devices)*2)
	deviceIDToIndex := make(map[string]int) // 设备ID/外部ID -> 设备在数组中的索引
	for i, d := range devices {
		deviceIDs = append(deviceIDs, d.ID)
		deviceIDToIndex[d.ID] = i
		// 同时添加external_device_id（如果有）
		if d.ExternalDeviceID != nil && *d.ExternalDeviceID != "" {
			deviceIDs = append(deviceIDs, *d.ExternalDeviceID)
			deviceIDToIndex[*d.ExternalDeviceID] = i
		}
	}

	// 批量查询所有设备的最新绑定运单（非完成状态）
	// 修复：运单中的device_id可能是设备ID或外部设备ID
	var shipments []models.Shipment
	if len(deviceIDs) > 0 {
		h.db.Raw(`
			SELECT s.* FROM shipments s
			INNER JOIN (
				SELECT device_id, MAX(created_at) as max_created
				FROM shipments 
				WHERE device_id IN (?) AND status NOT IN (?) AND deleted_at IS NULL
				GROUP BY device_id
			) latest ON s.device_id = latest.device_id AND s.created_at = latest.max_created
			WHERE s.deleted_at IS NULL
		`, deviceIDs, completedStatuses).Scan(&shipments)
	}

	// 构建包含绑定信息的响应
	result := make([]DeviceWithBinding, len(devices))
	// 收集需要查询的组织ID
	orgIDs := make([]string, 0)
	orgIDSet := make(map[string]bool)
	for i, device := range devices {
		result[i] = DeviceWithBinding{
			Device:        device,
			BindingStatus: "unbound",
		}
		if device.OrgID != nil && *device.OrgID != "" && !orgIDSet[*device.OrgID] {
			orgIDs = append(orgIDs, *device.OrgID)
			orgIDSet[*device.OrgID] = true
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

	// 填充组织名称
	for i := range result {
		if result[i].OrgID != nil && *result[i].OrgID != "" {
			if name, ok := orgNameMap[*result[i].OrgID]; ok {
				result[i].OrgName = name
			}
		}
	}

	// 遍历找到的运单，匹配到对应设备
	for _, shipment := range shipments {
		if shipment.DeviceID == nil {
			continue
		}
		// 通过device_id查找对应设备索引
		if idx, exists := deviceIDToIndex[*shipment.DeviceID]; exists {
			result[idx].BindingStatus = "bound"
			result[idx].BoundTransportType = shipment.TransportType
			result[idx].BoundCargoName = shipment.CargoName
			result[idx].BoundShipmentID = &shipment.ID
		}
	}

	utils.SuccessResponse(c, result)
}

func (h *DeviceHandler) syncExternalDeviceData(devices []models.Device) []models.Device {
	var externalIDs []string
	deviceMap := make(map[string]*models.Device)

	for i := range devices {
		if devices[i].ExternalDeviceID != nil && *devices[i].ExternalDeviceID != "" {
			externalIDs = append(externalIDs, *devices[i].ExternalDeviceID)
			deviceMap[*devices[i].ExternalDeviceID] = &devices[i]
		}
	}

	if len(externalIDs) == 0 {
		return devices
	}

	extDataList, err := services.Kuaihuoyun.GetDeviceInfoList(externalIDs)
	if err != nil {
		log.Printf("[DeviceSync] Failed to fetch external data: %v", err)
		return devices
	}

	for _, ext := range extDataList {
		device, ok := deviceMap[ext.Device]
		if !ok {
			continue
		}

		status := "offline"
		if ext.Status == 1 {
			status = "online"
		}

		device.Status = status
		device.Battery = ext.PowerRate
		device.Latitude = &ext.Latitude
		device.Longitude = &ext.Longitude
		device.Speed = ext.Speed
		device.Direction = ext.Direction
		device.LocateType = &ext.LocateType
		device.Temperature = ext.Temperature
		device.Humidity = ext.Humidity

		locateTime := time.Unix(ext.LocateTime, 0)
		device.LastUpdate = locateTime

		// 持久化到数据库
		h.db.Model(device).Updates(map[string]interface{}{
			"status":      device.Status,
			"battery":     device.Battery,
			"latitude":    device.Latitude,
			"longitude":   device.Longitude,
			"speed":       device.Speed,
			"direction":   device.Direction,
			"locate_type": device.LocateType,
			"temperature": device.Temperature,
			"humidity":    device.Humidity,
			"last_update": device.LastUpdate,
		})
	}

	log.Printf("[DeviceSync] Synced %d device(s) from external API", len(extDataList))
	return devices
}

func (h *DeviceHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var device models.Device
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "设备不存在")
		return
	}

	// 如果有外部设备ID，获取实时数据
	if device.ExternalDeviceID != nil && *device.ExternalDeviceID != "" {
		extData, err := services.Kuaihuoyun.GetDeviceInfo(*device.ExternalDeviceID)
		if err == nil && extData != nil {
			status := "offline"
			if extData.Status == 1 {
				status = "online"
			}

			device.Status = status
			device.Battery = extData.PowerRate
			device.Latitude = &extData.Latitude
			device.Longitude = &extData.Longitude
			device.Speed = extData.Speed
			device.Direction = extData.Direction
			device.LocateType = &extData.LocateType
			device.Temperature = extData.Temperature
			device.Humidity = extData.Humidity

			locateTime := time.Unix(extData.LocateTime, 0)
			device.LastUpdate = locateTime

			h.db.Model(&device).Updates(map[string]interface{}{
				"status":      device.Status,
				"battery":     device.Battery,
				"latitude":    device.Latitude,
				"longitude":   device.Longitude,
				"speed":       device.Speed,
				"direction":   device.Direction,
				"locate_type": device.LocateType,
				"temperature": device.Temperature,
				"humidity":    device.Humidity,
				"last_update": device.LastUpdate,
			})
		}
	}

	utils.SuccessResponse(c, device)
}

func (h *DeviceHandler) Create(c *gin.Context) {
	var req models.DeviceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的设备信息")
		return
	}

	// 检查设备ID是否已存在
	if req.ID != "" {
		var count int64
		h.db.Model(&models.Device{}).Where("id = ?", req.ID).Count(&count)
		if count > 0 {
			utils.BadRequest(c, "设备号已存在")
			return
		}
	}

	// 检查外部设备ID是否已绑定（排除已删除的设备）
	if req.ExternalDeviceID != nil && *req.ExternalDeviceID != "" {
		var count int64
		h.db.Model(&models.Device{}).Where("external_device_id = ? AND deleted_at IS NULL", *req.ExternalDeviceID).Count(&count)
		if count > 0 {
			utils.BadRequest(c, "当前设备已被绑定")
			return
		}
	}

	// 获取当前用户的主组织ID
	var userOrgID *string
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok && userIDStr != "" {
			var userOrg models.UserOrganization
			// 优先使用主组织，否则使用第一个组织
			if err := h.db.Where("user_id = ? AND is_primary = ?", userIDStr, true).First(&userOrg).Error; err == nil {
				userOrgID = &userOrg.OrganizationID
			} else {
				if err := h.db.Where("user_id = ?", userIDStr).First(&userOrg).Error; err == nil {
					userOrgID = &userOrg.OrganizationID
				}
			}
		}
	}

	device := models.Device{
		ID:               req.ID,
		Name:             req.Name,
		Type:             req.Type,
		Provider:         req.Provider,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		ExternalDeviceID: req.ExternalDeviceID,
		OrgID:            userOrgID, // 自动关联用户的组织
		Status:           "online",
		Battery:          100,
	}

	// 设置默认值
	if device.Type == "" {
		device.Type = "container"
	}
	if device.Provider == "" {
		device.Provider = "kuaihuoyun"
	}
	// 如果没有设置名称，使用设备ID作为默认名称
	if device.Name == "" {
		if req.ExternalDeviceID != nil && *req.ExternalDeviceID != "" {
			device.Name = "设备-" + *req.ExternalDeviceID
		} else {
			device.Name = "未命名设备"
		}
	}

	if err := h.db.Create(&device).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 如果绑定了外部设备，立即获取实时数据
	if device.ExternalDeviceID != nil && *device.ExternalDeviceID != "" {
		extData, err := services.Kuaihuoyun.GetDeviceInfo(*device.ExternalDeviceID)
		if err == nil && extData != nil {
			status := "offline"
			if extData.Status == 1 {
				status = "online"
			}

			h.db.Model(&device).Updates(map[string]interface{}{
				"status":      status,
				"battery":     extData.PowerRate,
				"latitude":    extData.Latitude,
				"longitude":   extData.Longitude,
				"speed":       extData.Speed,
				"direction":   extData.Direction,
				"locate_type": extData.LocateType,
				"temperature": extData.Temperature,
				"humidity":    extData.Humidity,
				"last_update": time.Unix(extData.LocateTime, 0),
			})

			h.db.First(&device, "id = ?", device.ID)
		}
	}

	utils.CreatedResponse(c, device)
}

func (h *DeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var device models.Device
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "设备不存在")
		return
	}

	var req models.DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求数据")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Battery != nil {
		updates["battery"] = *req.Battery
	}
	if req.Latitude != nil {
		updates["latitude"] = *req.Latitude
	}
	if req.Longitude != nil {
		updates["longitude"] = *req.Longitude
	}
	if req.ExternalDeviceID != nil {
		updates["external_device_id"] = *req.ExternalDeviceID
	}
	if req.OrgID != nil {
		updates["org_id"] = *req.OrgID
	}
	updates["last_update"] = time.Now()

	if err := h.db.Model(&device).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.First(&device, "id = ?", id)
	utils.SuccessResponse(c, device)
}

func (h *DeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&models.Device{}, "id = ?", id).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"success": true})
}

func (h *DeviceHandler) GetTrack(c *gin.Context) {
	id := c.Param("id")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	var device models.Device
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "设备不存在")
		return
	}

	// 默认查询最近24小时
	now := time.Now()
	if endTime == "" {
		endTime = now.Format("2006-01-02 15:04:05")
	}
	if startTime == "" {
		startTime = now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	}

	// 先从本地数据库查询
	var localTrack []models.DeviceTrack
	h.db.Where("device_id = ? AND locate_time BETWEEN ? AND ?", device.ID, startTime, endTime).
		Order("locate_time ASC").
		Find(&localTrack)

	// 如果有外部设备ID，从外部API获取并持久化
	if device.ExternalDeviceID != nil && *device.ExternalDeviceID != "" {
		extTrack, err := services.Kuaihuoyun.GetTrack(*device.ExternalDeviceID, startTime, endTime)
		if err == nil && len(extTrack) > 0 {
			for _, point := range extTrack {
				locateTime := time.Unix(point.LocateTime, 0)

				// 检查是否已存在
				var count int64
				h.db.Model(&models.DeviceTrack{}).
					Where("device_id = ? AND locate_time = ?", device.ID, locateTime).
					Count(&count)

				if count == 0 {
					track := models.DeviceTrack{
						DeviceID:    device.ID,
						Latitude:    point.Latitude,
						Longitude:   point.Longitude,
						Speed:       point.Speed,
						Direction:   point.Direction,
						Temperature: point.Temperature,
						Humidity:    point.Humidity,
						LocateType:  point.LocateType,
						LocateTime:  locateTime,
						SyncedAt:    time.Now(),
					}
					h.db.Create(&track)
				}
			}

			// 重新查询
			h.db.Where("device_id = ? AND locate_time BETWEEN ? AND ?", device.ID, startTime, endTime).
				Order("locate_time ASC").
				Find(&localTrack)
		}
	}

	// 转换为API格式
	result := make([]models.TrackPoint, len(localTrack))
	for i, h := range localTrack {
		result[i] = models.TrackPoint{
			Device:      device.ID,
			Speed:       h.Speed,
			Direction:   h.Direction,
			LocateTime:  h.LocateTime.Unix(),
			Longitude:   h.Longitude,
			Latitude:    h.Latitude,
			LocateType:  h.LocateType,
			RunStatus:   2,
			Temperature: h.Temperature,
			Humidity:    h.Humidity,
		}
	}

	utils.SuccessResponse(c, result)
}

func (h *DeviceHandler) GetHistory(c *gin.Context) {
	id := c.Param("id")
	limit := c.DefaultQuery("limit", "100")

	var history []models.LocationHistory
	h.db.Where("device_id = ?", id).
		Order("timestamp DESC").
		Limit(100).
		Find(&history)

	_ = limit
	utils.SuccessResponse(c, history)
}

// SeedDevices 创建示例设备
func SeedDevices(db *gorm.DB) error {
	var count int64
	db.Model(&models.Device{}).Count(&count)
	if count > 0 {
		return nil
	}

	lat1, lng1 := -6.210275, 106.828993
	lat2, lng2 := 27.834288, 109.261438
	extID1, extID2 := "868120342395115", "868120342395412"
	temp1, humidity1 := 29.8, 73.7
	temp2, humidity2 := 20.1, 51.1

	devices := []models.Device{
		{
			ID:               "XHK-001",
			Name:             "小黑卡测试设备",
			Type:             "container",
			Status:           "online",
			Battery:          87,
			Latitude:         &lat1,
			Longitude:        &lng1,
			ExternalDeviceID: &extID1,
			Temperature:      &temp1,
			Humidity:         &humidity1,
		},
		{
			ID:               "XHK-002",
			Name:             "小黑卡测试设备2",
			Type:             "container",
			Status:           "online",
			Battery:          56,
			Latitude:         &lat2,
			Longitude:        &lng2,
			ExternalDeviceID: &extID2,
			Temperature:      &temp2,
			Humidity:         &humidity2,
		},
	}

	for _, device := range devices {
		db.Create(&device)
	}

	log.Println("📱 Inserted sample devices")
	return nil
}
