package services

import (
	"fmt"
	"time"
	"trackcard-server/models"

	"gorm.io/gorm"
)

type AlertCheckService struct {
	db *gorm.DB
}

var AlertChecker *AlertCheckService

func InitAlertChecker(db *gorm.DB) {
	AlertChecker = &AlertCheckService{db: db}
}

// CheckAllAlerts 检查所有预警
func (s *AlertCheckService) CheckAllAlerts(shipment *models.Shipment) error {
	if shipment.Status == "completed" || shipment.Status == "cancelled" {
		return nil
	}

	// 1. 物理环境类预警 (基于IoT设备)
	if shipment.Device != nil {
		if err := s.checkPhysicalAlerts(shipment); err != nil {
			fmt.Printf("Error checking physical alerts for shipment %s: %v\n", shipment.ID, err)
		}
	}

	// 2. 节点异动类预警 (基于轨迹和时间)
	if err := s.checkNodeAlerts(shipment); err != nil {
		fmt.Printf("Error checking node alerts for shipment %s: %v\n", shipment.ID, err)
	}

	// 3. 末端作业与关务类预警
	if err := s.checkOperationAlerts(shipment); err != nil {
		fmt.Printf("Error checking operation alerts for shipment %s: %v\n", shipment.ID, err)
	}

	// 4. 船务类预警 (基于船司数据) - Phase 2新增
	if shipment.TransportType == "sea" && shipment.BillOfLading != "" {
		if err := s.checkCarrierAlerts(shipment); err != nil {
			fmt.Printf("Error checking carrier alerts for shipment %s: %v\n", shipment.ID, err)
		}
	}

	return nil
}

// 检查物理环境预警
func (s *AlertCheckService) checkPhysicalAlerts(shipment *models.Shipment) error {
	device := shipment.Device

	// 温度预警
	if shipment.MaxTemperature != nil && device.Temperature != nil && *device.Temperature > *shipment.MaxTemperature {
		s.createOrUpdateAlert(shipment, "temp_high", "warning", "高温预警",
			fmt.Sprintf("当前温度 %.1f°C 超过阈值 %.1f°C", *device.Temperature, *shipment.MaxTemperature), "physical")
	}
	if shipment.MinTemperature != nil && device.Temperature != nil && *device.Temperature < *shipment.MinTemperature {
		s.createOrUpdateAlert(shipment, "temp_low", "warning", "低温预警",
			fmt.Sprintf("当前温度 %.1f°C 低于阈值 %.1f°C", *device.Temperature, *shipment.MinTemperature), "physical")
	}

	// 湿度预警
	if shipment.MaxHumidity != nil && device.Humidity != nil && *device.Humidity > *shipment.MaxHumidity {
		s.createOrUpdateAlert(shipment, "humidity_high", "warning", "高湿预警",
			fmt.Sprintf("当前湿度 %.1f%% 超过阈值 %.1f%%", *device.Humidity, *shipment.MaxHumidity), "physical")
	}

	// 震动/倾斜预警 (假设设备数据已同步)
	if shipment.MaxShock != nil && device.Shock != nil && *device.Shock > *shipment.MaxShock {
		s.createOrUpdateAlert(shipment, "shock_detected", "critical", "剧烈震动",
			fmt.Sprintf("检测到震动 %.1fg 超过阈值 %.1fg", *device.Shock, *shipment.MaxShock), "physical")
	}
	if shipment.MaxTilt != nil && device.Tilt != nil && *device.Tilt > *shipment.MaxTilt {
		s.createOrUpdateAlert(shipment, "tilt_detected", "warning", "货物倾斜",
			fmt.Sprintf("检测到倾斜 %.1f° 超过阈值 %.1f°", *device.Tilt, *shipment.MaxTilt), "physical")
	}

	// 光感/开箱预警
	if device.Light != nil && *device.Light > 10.0 { // 假设10lux以上为开箱
		s.createOrUpdateAlert(shipment, "door_open", "critical", "非法开箱/光感告警",
			fmt.Sprintf("检测到光线 %.1flux，疑似非法开箱", *device.Light), "physical")
	}

	return nil
}

// 检查节点异动预警
func (s *AlertCheckService) checkNodeAlerts(shipment *models.Shipment) error {
	// ETD 未发出预警：预计发出时间已过但状态仍为pending
	if shipment.ETD != nil && shipment.Status == "pending" {
		if time.Now().After(*shipment.ETD) {
			delayHours := time.Since(*shipment.ETD).Hours()
			s.createOrUpdateAlert(shipment, "etd_not_departed", "warning", "货物未按时发出",
				fmt.Sprintf("预计发出时间 %s 已过，货物仍未离开发货地（延误 %.1f 小时）",
					shipment.ETD.Format("2006-01-02 15:04"), delayHours), "node")
		}
	}

	// ETA 严重偏离预警
	if shipment.ETA != nil {
		threshold := 24 // 默认24小时
		if shipment.MaxETADelayHours != nil {
			threshold = *shipment.MaxETADelayHours
		}

		// 如果当前时间超过ETA + 阈值且未到达
		if time.Now().After(shipment.ETA.Add(time.Duration(threshold)*time.Hour)) && shipment.Status != "delivered" {
			s.createOrUpdateAlert(shipment, "eta_delay", "warning", "ETA严重延误",
				fmt.Sprintf("原预计到达 %s，已超期%d小时", shipment.ETA.Format("2006-01-02"), threshold), "node")
		}
	}
	return nil
}

// 检查末端作业与关务预警
func (s *AlertCheckService) checkOperationAlerts(shipment *models.Shipment) error {
	// 海关查验预警
	if shipment.CustomsStatus == "examination" {
		holdDuration := 0.0
		if shipment.CustomsHoldSince != nil {
			holdDuration = time.Since(*shipment.CustomsHoldSince).Hours()
		}

		threshold := 48 // 默认48小时
		if shipment.MaxCustomsHoldHours != nil {
			threshold = *shipment.MaxCustomsHoldHours
		}

		if holdDuration > float64(threshold) {
			s.createOrUpdateAlert(shipment, "customs_hold", "warning", "海关查验滞留",
				fmt.Sprintf("货物处于查验状态已超过 %.0f 小时（阈值：%d小时）", holdDuration, threshold), "operation")
		}
	}

	// 免租期倒计时预警
	if shipment.FreeTimeExpiration != nil {
		timeLeft := time.Until(*shipment.FreeTimeExpiration)

		threshold := 24 // 默认24小时
		if shipment.FreeTimeWarningHours != nil {
			threshold = *shipment.FreeTimeWarningHours
		}

		if timeLeft > 0 && timeLeft < time.Duration(threshold)*time.Hour {
			s.createOrUpdateAlert(shipment, "free_time_expiring", "critical", "免租期即将到期",
				fmt.Sprintf("免租期将在 %.1f 小时后结束，请尽快安排提货/还柜避免滞期费", timeLeft.Hours()), "operation")
		} else if timeLeft < 0 {
			s.createOrUpdateAlert(shipment, "free_time_expired", "critical", "免租期已逾期",
				fmt.Sprintf("免租期已过期 %.1f 小时，正在产生滞期费", -timeLeft.Hours()), "operation")
		}
	}

	return nil
}

// 检查船务类预警 (Phase 2新增)
func (s *AlertCheckService) checkCarrierAlerts(shipment *models.Shipment) error {
	// (1) 船期延误预警 - 基于船司最新ETA与原预计的对比
	if shipment.ETA != nil && shipment.ATA == nil {
		// 检查是否已超过ETA
		if time.Now().After(*shipment.ETA) {
			delayHours := time.Since(*shipment.ETA).Hours()
			if delayHours > 24 {
				s.createOrUpdateAlert(shipment, "vessel_delay", "warning", "船期延误",
					fmt.Sprintf("已超过预计到达时间 %.0f 小时", delayHours), "carrier")
			}
		}
	}

	// (2) 获取最近的船司事件检查异常
	var latestTrack models.CarrierTrack
	err := s.db.Where("shipment_id = ?", shipment.ID).
		Order("event_time DESC").
		First(&latestTrack).Error

	if err == nil {
		// 检查是否长时间无更新 (超过7天)
		if time.Since(latestTrack.EventTime) > 7*24*time.Hour {
			s.createOrUpdateAlert(shipment, "carrier_stale", "info", "船务状态未更新",
				fmt.Sprintf("最后更新: %s, 已超过7天未获取新状态",
					latestTrack.EventTime.Format("2006-01-02")), "carrier")
		}

		// 检查是否有ETA变更
		if latestTrack.ETAUpdate != nil && shipment.ETA != nil {
			diff := latestTrack.ETAUpdate.Sub(*shipment.ETA)
			if diff > 48*time.Hour {
				s.createOrUpdateAlert(shipment, "eta_change", "warning", "ETA大幅变更",
					fmt.Sprintf("船司最新ETA较原计划延后 %.0f 小时", diff.Hours()), "carrier")
			}
		}
	}

	return nil
}

func (s *AlertCheckService) createOrUpdateAlert(shipment *models.Shipment, alertType, severity, title, message, category string) {
	// 简单的去重逻辑：检查是否已有同类型未解决的告警
	var count int64
	s.db.Model(&models.Alert{}).Where("shipment_id = ? AND type = ? AND status = 'pending'", shipment.ID, alertType).Count(&count)
	if count > 0 {
		return // 已存在未解决的同类型告警，跳过
	}

	alert := models.Alert{
		ShipmentID: &shipment.ID,
		DeviceID:   shipment.DeviceID,
		Type:       alertType,
		Severity:   severity,
		Title:      title,
		Message:    &message,
		Status:     "pending",
		Category:   category,
		CreatedAt:  time.Now(),
	}
	s.db.Create(&alert)
}
