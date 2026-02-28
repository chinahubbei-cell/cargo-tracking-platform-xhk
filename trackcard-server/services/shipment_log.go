package services

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"trackcard-server/models"
)

// ShipmentLogService 运单日志服务
type ShipmentLogService struct {
	db *gorm.DB
}

// NewShipmentLogService 创建运单日志服务
func NewShipmentLogService(db *gorm.DB) *ShipmentLogService {
	return &ShipmentLogService{db: db}
}

// Log 记录运单日志
func (s *ShipmentLogService) Log(shipmentID, action, field, oldValue, newValue, description, operatorID, operatorIP string) {
	logEntry := models.ShipmentLog{
		ShipmentID:  shipmentID,
		Action:      action,
		Field:       field,
		OldValue:    oldValue,
		NewValue:    newValue,
		Description: description,
		OperatorID:  operatorID,
		OperatorIP:  operatorIP,
	}
	if err := s.db.Create(&logEntry).Error; err != nil {
		log.Printf("[ShipmentLog] 记录日志失败: %v", err)
	}
}

// LogCreated 记录运单创建
func (s *ShipmentLogService) LogCreated(shipmentID, operatorID, operatorIP string) {
	s.Log(shipmentID, models.LogActionCreated, "", "", "", "运单创建", operatorID, operatorIP)
}

// LogStatusChanged 记录状态变更
func (s *ShipmentLogService) LogStatusChanged(shipmentID, oldStatus, newStatus, operatorID, operatorIP string) {
	statusLabels := map[string]string{
		"pending":    "待发货",
		"in_transit": "运输中",
		"delivered":  "已到达",
		"cancelled":  "已取消",
	}
	oldLabel := statusLabels[oldStatus]
	newLabel := statusLabels[newStatus]
	if oldLabel == "" {
		oldLabel = oldStatus
	}
	if newLabel == "" {
		newLabel = newStatus
	}
	desc := fmt.Sprintf("状态从【%s】变更为【%s】", oldLabel, newLabel)
	s.Log(shipmentID, models.LogActionStatusChanged, "status", oldStatus, newStatus, desc, operatorID, operatorIP)
}

// LogDelivered 记录签收确认
func (s *ShipmentLogService) LogDelivered(shipmentID, receiver, operatorID, operatorIP string) {
	desc := "运单已签收"
	if receiver != "" {
		desc = fmt.Sprintf("运单已签收，签收人：%s", receiver)
	}
	s.Log(shipmentID, "delivered", "receiver", "", receiver, desc, operatorID, operatorIP)
}

// getExternalDeviceID 获取设备的外部ID号
func (s *ShipmentLogService) getExternalDeviceID(deviceID string) string {
	var device models.Device
	if err := s.db.Select("external_device_id").Where("id = ?", deviceID).First(&device).Error; err == nil {
		if device.ExternalDeviceID != nil && *device.ExternalDeviceID != "" {
			return *device.ExternalDeviceID
		}
	}
	return deviceID // 如果查询失败，返回原始ID
}

// LogDeviceBound 记录设备绑定
func (s *ShipmentLogService) LogDeviceBound(shipmentID, deviceID, operatorID, operatorIP string) {
	externalID := s.getExternalDeviceID(deviceID)
	desc := fmt.Sprintf("绑定设备【%s】", externalID)
	s.Log(shipmentID, models.LogActionDeviceBound, "device_id", "", deviceID, desc, operatorID, operatorIP)
}

// LogDeviceUnbound 记录设备解绑
func (s *ShipmentLogService) LogDeviceUnbound(shipmentID, deviceID, reason, operatorID, operatorIP string) {
	externalID := s.getExternalDeviceID(deviceID)
	reasonLabels := map[string]string{
		"replaced":  "更换设备",
		"completed": "运单完成",
		"manual":    "手动解绑",
	}
	reasonLabel := reasonLabels[reason]
	if reasonLabel == "" {
		reasonLabel = reason
	}
	desc := fmt.Sprintf("解绑设备【%s】，原因：%s", externalID, reasonLabel)
	s.Log(shipmentID, models.LogActionDeviceUnbound, "device_id", deviceID, "", desc, operatorID, operatorIP)
}

// LogDeviceReplaced 记录设备更换
func (s *ShipmentLogService) LogDeviceReplaced(shipmentID, oldDeviceID, newDeviceID, operatorID, operatorIP string) {
	oldExternalID := s.getExternalDeviceID(oldDeviceID)
	newExternalID := s.getExternalDeviceID(newDeviceID)
	desc := fmt.Sprintf("设备从【%s】更换为【%s】", oldExternalID, newExternalID)
	s.Log(shipmentID, models.LogActionDeviceReplaced, "device_id", oldDeviceID, newDeviceID, desc, operatorID, operatorIP)
}

// LogFieldUpdated 记录字段更新
func (s *ShipmentLogService) LogFieldUpdated(shipmentID, field, oldValue, newValue, operatorID, operatorIP string) {
	fieldLabels := map[string]string{
		// 基础信息
		"origin":         "发货地",
		"destination":    "目的地",
		"cargo_name":     "货物名称",
		"transport_type": "运输类型",
		"cargo_type":     "货物类型",
		"transport_mode": "运输模式",
		"container_type": "柜型",
		"org_id":         "组织机构",
		// 单证信息
		"bill_of_lading": "提单号",
		"container_no":   "箱号/车牌",
		"seal_no":        "封条号",
		// 船务信息
		"vessel_name": "船名",
		"voyage_no":   "航次",
		"carrier":     "船司/航司",
		// 订单关联
		"po_numbers":      "PO单号",
		"sku_ids":         "SKU ID",
		"fba_shipment_id": "FBA编号",
		// 货物量纲
		"pieces": "件数",
		"weight": "重量(kg)",
		"volume": "体积(m³)",
		// 费用信息
		"freight_cost": "运费",
		"surcharges":   "附加费",
		"customs_fee":  "关税",
		"other_cost":   "其他费用",
		"total_cost":   "总费用",
		// 时间信息
		"etd": "预计出发时间(ETD)",
		"atd": "实际出发时间(ATD)",
		"eta": "预计到达时间(ETA)",
		"ata": "实际到达时间(ATA)",
		// 路由信息
		"sender_name":    "发货人",
		"sender_phone":   "发货电话",
		"origin_address": "发货地址",
		"receiver_name":  "收货人",
		"receiver_phone": "收货电话",
		"dest_address":   "收货地址",
		// 坐标
		"origin_lat": "发货地纬度",
		"origin_lng": "发货地经度",
		"dest_lat":   "目的地纬度",
		"dest_lng":   "目的地经度",
	}
	fieldLabel := fieldLabels[field]
	if fieldLabel == "" {
		fieldLabel = field
	}
	desc := fmt.Sprintf("【%s】从【%s】变更为【%s】", fieldLabel, oldValue, newValue)
	s.Log(shipmentID, models.LogActionUpdated, field, oldValue, newValue, desc, operatorID, operatorIP)
}

// LogStageTransition 记录环节推进
func (s *ShipmentLogService) LogStageTransition(shipmentID, fromStage, toStage, note, operatorID, operatorIP string) {
	stageLabels := map[string]string{
		"first_mile":    "前程运输",
		"origin_port":   "起运港",
		"main_carriage": "干线运输",
		"dest_port":     "目的港",
		"last_mile":     "末端配送",
	}
	fromLabel := stageLabels[fromStage]
	toLabel := stageLabels[toStage]
	if fromLabel == "" {
		fromLabel = fromStage
	}
	if toLabel == "" {
		toLabel = toStage
	}
	desc := fmt.Sprintf("环节推进：【%s】→【%s】", fromLabel, toLabel)
	if note != "" {
		desc = desc + "，备注：" + note
	}
	s.Log(shipmentID, "stage_transition", "current_stage", fromStage, toStage, desc, operatorID, operatorIP)
}

// LogDeleted 记录运单删除
func (s *ShipmentLogService) LogDeleted(shipmentID, operatorID, operatorIP string) {
	s.Log(shipmentID, models.LogActionDeleted, "", "", "", "运单删除", operatorID, operatorIP)
}

// GetLogs 获取运单日志列表
func (s *ShipmentLogService) GetLogs(shipmentID string) []models.ShipmentLog {
	var logs []models.ShipmentLog
	s.db.Where("shipment_id = ?", shipmentID).Order("created_at DESC").Find(&logs)
	return logs
}

// ShipmentLog 全局运单日志服务实例
var ShipmentLog *ShipmentLogService

// InitShipmentLog 初始化运单日志服务
func InitShipmentLog(db *gorm.DB) {
	ShipmentLog = NewShipmentLogService(db)
	log.Println("[ShipmentLog] 运单日志服务初始化完成")
}
