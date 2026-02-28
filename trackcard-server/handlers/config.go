package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

type ConfigHandler struct {
	db *gorm.DB
}

func NewConfigHandler(db *gorm.DB) *ConfigHandler {
	return &ConfigHandler{db: db}
}

func (h *ConfigHandler) Get(c *gin.Context) {
	var configs []models.SystemConfig
	h.db.Find(&configs)

	result := make(map[string]string)
	for _, cfg := range configs {
		// 不返回敏感密钥的完整值
		if cfg.Key == "kuaihuoyun_secret_key" && len(cfg.Value) > 4 {
			result[cfg.Key] = cfg.Value[:4] + "****"
		} else {
			result[cfg.Key] = cfg.Value
		}
	}

	utils.SuccessResponse(c, result)
}

func (h *ConfigHandler) Update(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的配置数据")
		return
	}

	for key, value := range req {
		var config models.SystemConfig
		result := h.db.First(&config, "key = ?", key)

		if result.Error == gorm.ErrRecordNotFound {
			config = models.SystemConfig{
				Key:       key,
				Value:     value,
				UpdatedAt: time.Now(),
			}
			h.db.Create(&config)
		} else {
			h.db.Model(&config).Updates(map[string]interface{}{
				"value":      value,
				"updated_at": time.Now(),
			})
		}

		// 如果更新了快货运配置，重新初始化服务
		if key == "kuaihuoyun_cid" || key == "kuaihuoyun_secret_key" {
			services.InitKuaihuoyun()
		}
	}

	utils.SuccessResponse(c, gin.H{"message": "配置更新成功"})
}

func (h *ConfigHandler) GetByKey(c *gin.Context) {
	key := c.Param("key")

	var config models.SystemConfig
	if err := h.db.First(&config, "key = ?", key).Error; err != nil {
		utils.NotFound(c, "配置项不存在")
		return
	}

	// 不返回敏感密钥的完整值
	value := config.Value
	if key == "kuaihuoyun_secret_key" && len(value) > 4 {
		value = value[:4] + "****"
	}

	utils.SuccessResponse(c, gin.H{
		"key":        config.Key,
		"value":      value,
		"updated_at": config.UpdatedAt,
	})
}

func (h *ConfigHandler) TestKuaihuoyunAPI(c *gin.Context) {
	if !services.Kuaihuoyun.IsConfigured() {
		utils.BadRequest(c, "快货运API未配置")
		return
	}

	// 尝试获取一个测试设备的信息
	info, err := services.Kuaihuoyun.GetDeviceInfo("868120342395115")
	if err != nil {
		utils.InternalError(c, "API测试失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"success": true,
		"message": "API连接成功",
		"data":    info,
	})
}

// GetShipmentFieldConfig 获取运单字段配置
func (h *ConfigHandler) GetShipmentFieldConfig(c *gin.Context) {
	// 定义所有可配置的字段
	fieldKeys := []string{
		"shipment_field_enabled_bill_of_lading",
		"shipment_field_enabled_container_no",
		"shipment_field_enabled_seal_no",
		"shipment_field_enabled_vessel_name",
		"shipment_field_enabled_voyage_no",
		"shipment_field_enabled_carrier",
		"shipment_field_enabled_po_numbers",
		"shipment_field_enabled_sku_ids",
		"shipment_field_enabled_fba_shipment_id",
		"shipment_field_enabled_surcharges",
		"shipment_field_enabled_customs_fee",
		"shipment_field_enabled_other_cost",
	}

	var configs []models.SystemConfig
	h.db.Where("key IN ?", fieldKeys).Find(&configs)

	// 构建返回结果
	result := gin.H{
		"bill_of_lading":     false,
		"container_no":      false,
		"seal_no":            false,
		"vessel_name":        false,
		"voyage_no":          false,
		"carrier":            false,
		"po_numbers":         false,
		"sku_ids":            false,
		"fba_shipment_id":    false,
		"surcharges":          false,
		"customs_fee":        false,
		"other_cost":         false,
	}

	// 从数据库读取配置
	for _, cfg := range configs {
		key := cfg.Key
		value := cfg.Value == "true"

		// 映射到简化字段名
		switch key {
		case "shipment_field_enabled_bill_of_lading":
			result["bill_of_lading"] = value
		case "shipment_field_enabled_container_no":
			result["container_no"] = value
		case "shipment_field_enabled_seal_no":
			result["seal_no"] = value
		case "shipment_field_enabled_vessel_name":
			result["vessel_name"] = value
		case "shipment_field_enabled_voyage_no":
			result["voyage_no"] = value
		case "shipment_field_enabled_carrier":
			result["carrier"] = value
		case "shipment_field_enabled_po_numbers":
			result["po_numbers"] = value
		case "shipment_field_enabled_sku_ids":
			result["sku_ids"] = value
		case "shipment_field_enabled_fba_shipment_id":
			result["fba_shipment_id"] = value
		case "shipment_field_enabled_surcharges":
			result["surcharges"] = value
		case "shipment_field_enabled_customs_fee":
			result["customs_fee"] = value
		case "shipment_field_enabled_other_cost":
			result["other_cost"] = value
		}
	}

	utils.SuccessResponse(c, result)
}

// UpdateShipmentFieldConfig 更新运单字段配置
func (h *ConfigHandler) UpdateShipmentFieldConfig(c *gin.Context) {
	var req map[string]bool
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的配置数据")
		return
	}

	// 字段映射
	fieldMapping := map[string]string{
		"bill_of_lading":    "shipment_field_enabled_bill_of_lading",
		"container_no":      "shipment_field_enabled_container_no",
		"seal_no":            "shipment_field_enabled_seal_no",
		"vessel_name":        "shipment_field_enabled_vessel_name",
		"voyage_no":          "shipment_field_enabled_voyage_no",
		"carrier":            "shipment_field_enabled_carrier",
		"po_numbers":         "shipment_field_enabled_po_numbers",
		"sku_ids":            "shipment_field_enabled_sku_ids",
		"fba_shipment_id":    "shipment_field_enabled_fba_shipment_id",
		"surcharges":          "shipment_field_enabled_surcharges",
		"customs_fee":        "shipment_field_enabled_customs_fee",
		"other_cost":         "shipment_field_enabled_other_cost",
	}

	// 更新配置
	for fieldKey, enabled := range req {
		configKey, ok := fieldMapping[fieldKey]
		if !ok {
			continue
		}

		value := "false"
		if enabled {
			value = "true"
		}

		var config models.SystemConfig
		result := h.db.First(&config, "key = ?", configKey)

		if result.Error == gorm.ErrRecordNotFound {
			config = models.SystemConfig{
				Key:       configKey,
				Value:     value,
				UpdatedAt: time.Now(),
			}
			h.db.Create(&config)
		} else {
			h.db.Model(&config).Update("value", value)
		}
	}

	utils.SuccessResponse(c, gin.H{"message": "运单字段配置更新成功"})
}
