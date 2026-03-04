package handlers

import (
	"log"
	"net/http"
	"sort"
	"time"

	"trackcard-admin/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeviceHandler struct {
	db *gorm.DB
}

func NewDeviceHandler(db *gorm.DB) *DeviceHandler {
	return &DeviceHandler{db: db}
}

// syncHardwareDevicesFromBusiness 同步业务侧设备归属到管理侧硬件表
// 目标：修复历史脏数据导致的组织归属显示错误（如 devices 已修正但 hardware_devices 仍是旧值）
func (h *DeviceHandler) syncHardwareDevicesFromBusiness() {
	type businessDeviceRow struct {
		ID               string     `gorm:"column:id"`
		Type             string     `gorm:"column:type"`
		Name             string     `gorm:"column:name"`
		ExternalDeviceID *string    `gorm:"column:external_device_id"`
		OrgID            *string    `gorm:"column:org_id"`
		OrgName          *string    `gorm:"column:org_name"`
		CreatedAt        time.Time  `gorm:"column:created_at"`
		LastUpdate       *time.Time `gorm:"column:last_update"`
	}

	var businessDevices []businessDeviceRow
	if err := h.db.Table("devices AS d").
		Select("d.id, d.type, d.name, d.external_device_id, d.org_id, o.name AS org_name, d.created_at, d.last_update").
		Joins("LEFT JOIN organizations o ON o.id = d.org_id").
		Where("d.deleted_at IS NULL").
		Find(&businessDevices).Error; err != nil {
		log.Printf("[AdminDeviceSync] query business devices failed: %v", err)
		return
	}

	if len(businessDevices) == 0 {
		return
	}

	var hardwareDevices []models.HardwareDevice
	if err := h.db.Find(&hardwareDevices).Error; err != nil {
		log.Printf("[AdminDeviceSync] query hardware devices failed: %v", err)
		return
	}

	hardwareByID := make(map[string]*models.HardwareDevice, len(hardwareDevices))
	hardwareByIMEI := make(map[string]*models.HardwareDevice, len(hardwareDevices))
	for i := range hardwareDevices {
		hd := &hardwareDevices[i]
		hardwareByID[hd.ID] = hd
		if hd.IMEI != "" {
			hardwareByIMEI[hd.IMEI] = hd
		}
	}

	// 可能存在同一 IMEI 多条业务设备记录（历史脏数据），按“最后更新时间/创建时间”降序，确保最新归属生效
	sort.SliceStable(businessDevices, func(i, j int) bool {
		ti := businessDevices[i].CreatedAt
		if businessDevices[i].LastUpdate != nil && !businessDevices[i].LastUpdate.IsZero() {
			ti = *businessDevices[i].LastUpdate
		}
		tj := businessDevices[j].CreatedAt
		if businessDevices[j].LastUpdate != nil && !businessDevices[j].LastUpdate.IsZero() {
			tj = *businessDevices[j].LastUpdate
		}
		return ti.After(tj)
	})

	processedHardware := make(map[string]bool)
	now := time.Now()
	for _, bd := range businessDevices {
		externalID := bd.ID
		if bd.ExternalDeviceID != nil && *bd.ExternalDeviceID != "" {
			externalID = *bd.ExternalDeviceID
		}

		target := hardwareByID[bd.ID]
		if target == nil && externalID != "" {
			target = hardwareByIMEI[externalID]
		}

		orgID := ""
		if bd.OrgID != nil {
			orgID = *bd.OrgID
		}
		orgName := ""
		if bd.OrgName != nil {
			orgName = *bd.OrgName
		}

		if target != nil {
			// 已处理过该硬件设备，跳过旧业务记录，避免被历史脏数据覆盖
			if processedHardware[target.ID] {
				continue
			}
			processedHardware[target.ID] = true

			updates := make(map[string]interface{})
			if target.OrgID != orgID {
				updates["org_id"] = orgID
			}
			if target.OrgName != orgName {
				updates["org_name"] = orgName
			}
			if len(updates) > 0 {
				updates["updated_at"] = now
				if err := h.db.Model(target).Updates(updates).Error; err != nil {
					log.Printf("[AdminDeviceSync] update hardware device %s failed: %v", target.ID, err)
				} else {
					target.OrgID = orgID
					target.OrgName = orgName
				}
			}
			continue
		}

		deviceType := bd.Type
		if deviceType == "" {
			deviceType = "container"
		}
		createdAt := bd.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		updatedAt := createdAt
		if bd.LastUpdate != nil && !bd.LastUpdate.IsZero() {
			updatedAt = *bd.LastUpdate
		}

		newDevice := models.HardwareDevice{
			ID:          bd.ID,
			DeviceType:  deviceType,
			DeviceModel: bd.Name,
			IMEI:        externalID,
			SN:          bd.ID,
			Status:      "allocated",
			OrgID:       orgID,
			OrgName:     orgName,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}
		if err := h.db.Create(&newDevice).Error; err != nil {
			log.Printf("[AdminDeviceSync] insert hardware device %s failed: %v", bd.ID, err)
			continue
		}
		hardwareByID[newDevice.ID] = &newDevice
		if newDevice.IMEI != "" {
			hardwareByIMEI[newDevice.IMEI] = &newDevice
		}
		processedHardware[newDevice.ID] = true
	}
}

// List 获取设备列表（带分页）
func (h *DeviceHandler) List(c *gin.Context) {
	var page PaginationParams
	c.ShouldBindQuery(&page)
	page.Normalize()

	var total int64
	var devices []models.HardwareDevice

	query := h.db.Model(&models.HardwareDevice{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if orgID := c.Query("org_id"); orgID != "" {
		query = query.Where("org_id = ?", orgID)
	}
	if deviceType := c.Query("type"); deviceType != "" {
		query = query.Where("device_type = ?", deviceType)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("imei LIKE ? OR sn LIKE ? OR org_name LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	h.syncHardwareDevicesFromBusiness()

	// 总数
	query.Count(&total)

	// 分页
	offset := (page.Page - 1) * page.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(page.PageSize).Find(&devices)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    devices,
		"pagination": gin.H{
			"page":        page.Page,
			"page_size":   page.PageSize,
			"total":       total,
			"total_pages": (total + int64(page.PageSize) - 1) / int64(page.PageSize),
		},
	})
}

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	DeviceType  string `json:"device_type" binding:"required"`
	IMEI        string `json:"imei"`
	SN          string `json:"sn"`
	DeviceModel string `json:"device_model"`
	BatchNo     string `json:"batch_no"`
	Remark      string `json:"remark"`
}

// Create 入库设备
func (h *DeviceHandler) Create(c *gin.Context) {
	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "设备类型不能为空"})
		return
	}

	if req.IMEI == "" && req.SN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "MISSING_ID", "message": "IMEI或SN至少填一个"})
		return
	}

	// 检查重复
	if req.IMEI != "" {
		var existing models.HardwareDevice
		if h.db.Where("imei = ?", req.IMEI).First(&existing).Error == nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "DUPLICATE_IMEI", "message": "IMEI已存在"})
			return
		}
	}
	if req.SN != "" {
		var existing models.HardwareDevice
		if h.db.Where("sn = ?", req.SN).First(&existing).Error == nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "DUPLICATE_SN", "message": "SN已存在"})
			return
		}
	}

	device := models.HardwareDevice{
		DeviceType:  req.DeviceType,
		IMEI:        req.IMEI,
		SN:          req.SN,
		DeviceModel: req.DeviceModel,
		BatchNo:     req.BatchNo,
		Remark:      req.Remark,
		Status:      "in_stock",
	}

	if err := h.db.Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "CREATE_FAILED", "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": device})
}

type BatchImportRequest struct {
	Devices []CreateDeviceRequest `json:"devices" binding:"required,min=1,max=500"`
}

// BatchImport 批量导入（带事务）
func (h *DeviceHandler) BatchImport(c *gin.Context) {
	var req BatchImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "设备列表无效（1-500条）"})
		return
	}

	var successCount, failCount int
	var failedItems []string

	// 使用事务
	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Devices {
			if item.IMEI == "" && item.SN == "" {
				failCount++
				failedItems = append(failedItems, "缺少IMEI/SN")
				continue
			}

			// 检查IMEI重复
			if item.IMEI != "" {
				var existing models.HardwareDevice
				if tx.Where("imei = ?", item.IMEI).First(&existing).Error == nil {
					failCount++
					failedItems = append(failedItems, "IMEI重复: "+item.IMEI)
					continue
				}
			}

			device := models.HardwareDevice{
				DeviceType:  item.DeviceType,
				IMEI:        item.IMEI,
				SN:          item.SN,
				DeviceModel: item.DeviceModel,
				BatchNo:     item.BatchNo,
				Remark:      item.Remark,
				Status:      "in_stock",
			}

			if err := tx.Create(&device).Error; err != nil {
				failCount++
				failedItems = append(failedItems, "创建失败: "+item.IMEI)
			} else {
				successCount++
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "TX_FAILED", "message": "批量导入失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量导入完成",
		"data": gin.H{
			"success_count": successCount,
			"fail_count":    failCount,
			"failed_items":  failedItems,
		},
	})
}

// Get 获取设备详情
func (h *DeviceHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_ID", "message": "ID不能为空"})
		return
	}

	var device models.HardwareDevice
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "设备不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": device})
}

// UpdateDeviceRequest 更新设备请求（白名单）
type UpdateDeviceRequest struct {
	DeviceModel *string `json:"device_model"`
	BatchNo     *string `json:"batch_no"`
	Remark      *string `json:"remark"`
}

// Update 更新设备（白名单限制）
func (h *DeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var device models.HardwareDevice
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "设备不存在"})
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "请求参数错误"})
		return
	}

	updates := make(map[string]interface{})
	if req.DeviceModel != nil {
		updates["device_model"] = *req.DeviceModel
	}
	if req.BatchNo != nil {
		updates["batch_no"] = *req.BatchNo
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		h.db.Model(&device).Updates(updates)
	}

	h.db.First(&device, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": device})
}

type AllocateRequest struct {
	OrgID          string `json:"org_id" binding:"required"`
	OrgName        string `json:"org_name"`
	SubAccountID   string `json:"sub_account_id"`
	SubAccountName string `json:"sub_account_name"`
	OrderID        string `json:"order_id"`
	Remark         string `json:"remark"`
}

// Allocate 分配设备
func (h *DeviceHandler) Allocate(c *gin.Context) {
	id := c.Param("id")
	adminID := c.GetString("user_id")

	var device models.HardwareDevice
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "设备不存在"})
		return
	}

	if !device.IsAvailable() {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "NOT_AVAILABLE", "message": "设备不可分配（当前状态: " + device.Status + "）"})
		return
	}

	var req AllocateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "INVALID_PARAMS", "message": "组织ID不能为空"})
		return
	}

	// 验证组织存在
	var org models.Organization
	if err := h.db.First(&org, "id = ?", req.OrgID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "ORG_NOT_FOUND", "message": "组织不存在"})
		return
	}

	// 检查配额
	var currentCount int64
	h.db.Model(&models.HardwareDevice{}).Where("org_id = ?", req.OrgID).Count(&currentCount)
	if int(currentCount) >= org.MaxDevices {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "QUOTA_EXCEEDED", "message": "超出设备配额限制"})
		return
	}

	now := time.Now()
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&device).Updates(map[string]interface{}{
			"status":           "allocated",
			"org_id":           req.OrgID,
			"org_name":         org.Name,
			"sub_account_id":   req.SubAccountID,
			"sub_account_name": req.SubAccountName,
			"order_id":         req.OrderID,
			"allocated_at":     now,
			"allocated_by":     adminID,
			"updated_at":       now,
		}).Error; err != nil {
			return err
		}

		// 同步到业务表 devices
		deviceName := device.DeviceModel
		if deviceName == "" {
			deviceName = "设备-" + device.IMEI
		}
		if err := tx.Exec(`
			INSERT INTO devices (id, name, type, status, external_device_id, org_id, sub_account_id, service_status, provider, created_at, last_update)
			VALUES (?, ?, ?, 'unactivated', ?, ?, ?, 'active', 'kuaihuoyun', ?, ?)
			ON CONFLICT(id) DO UPDATE SET 
			org_id=excluded.org_id, 
			sub_account_id=excluded.sub_account_id,
			service_status='active',
			deleted_at=NULL
		`, device.ID, deviceName, "container", device.IMEI, req.OrgID, req.SubAccountID, now, now).Error; err != nil {
			return err
		}

		// 记录分配日志
		log := &models.DeviceAllocationLog{
			DeviceID:  device.ID,
			IMEI:      device.IMEI,
			Action:    "allocate",
			ToOrgID:   req.OrgID,
			OrgName:   org.Name,
			OrderID:   req.OrderID,
			Remark:    req.Remark,
			CreatedBy: adminID,
		}
		return tx.Create(log).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "TX_FAILED", "message": "分配同步失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "分配成功"})
}

type ReturnRequest struct {
	Reason string `json:"reason"`
}

// Return 设备退回
func (h *DeviceHandler) Return(c *gin.Context) {
	id := c.Param("id")
	adminID := c.GetString("user_id")

	var device models.HardwareDevice
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "code": "NOT_FOUND", "message": "设备不存在"})
		return
	}

	if device.Status == "in_stock" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "ALREADY_IN_STOCK", "message": "设备已在库存中"})
		return
	}

	var req ReturnRequest
	c.ShouldBindJSON(&req)

	oldOrgID := device.OrgID
	oldOrgName := device.OrgName

	now := time.Now()
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&device).Updates(map[string]interface{}{
			"status":           "in_stock",
			"org_id":           "",
			"org_name":         "",
			"sub_account_id":   "",
			"sub_account_name": "",
			"returned_at":      now,
			"return_reason":    req.Reason,
			"updated_at":       now,
		}).Error; err != nil {
			return err
		}

		// 在业务表中软删除/释放该设备
		if err := tx.Exec("UPDATE devices SET deleted_at=?, org_id = NULL, sub_account_id = NULL WHERE id=?", now, device.ID).Error; err != nil {
			return err
		}

		// 记录退回日志
		log := &models.DeviceAllocationLog{
			DeviceID:  device.ID,
			IMEI:      device.IMEI,
			Action:    "return",
			FromOrgID: oldOrgID,
			OrgName:   oldOrgName,
			Remark:    req.Reason,
			CreatedBy: adminID,
		}
		return tx.Create(log).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "code": "TX_FAILED", "message": "退回同步失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "退回成功"})
}

// Stats 设备统计（优化：单次GROUP BY查询）
func (h *DeviceHandler) Stats(c *gin.Context) {
	type StatusCount struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}

	var counts []StatusCount
	h.db.Model(&models.HardwareDevice{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&counts)

	stats := models.DeviceStats{}
	for _, sc := range counts {
		switch sc.Status {
		case "in_stock":
			stats.InStock = int(sc.Count)
		case "allocated":
			stats.Allocated = int(sc.Count)
		case "activated":
			stats.Activated = int(sc.Count)
		case "returned":
			stats.Returned = int(sc.Count)
		case "damaged":
			stats.Damaged = int(sc.Count)
		}
		stats.Total += int(sc.Count)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// GetLogs 获取分配日志（带分页）
func (h *DeviceHandler) GetLogs(c *gin.Context) {
	var page PaginationParams
	c.ShouldBindQuery(&page)
	page.Normalize()

	var total int64
	var logs []models.DeviceAllocationLog

	query := h.db.Model(&models.DeviceAllocationLog{})

	if deviceID := c.Query("device_id"); deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if orgID := c.Query("org_id"); orgID != "" {
		query = query.Where("to_org_id = ? OR from_org_id = ?", orgID, orgID)
	}

	query.Count(&total)

	offset := (page.Page - 1) * page.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(page.PageSize).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
		"pagination": gin.H{
			"page":        page.Page,
			"page_size":   page.PageSize,
			"total":       total,
			"total_pages": (total + int64(page.PageSize) - 1) / int64(page.PageSize),
		},
	})
}
