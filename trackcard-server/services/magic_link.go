package services

import (
	"encoding/json"
	"fmt"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// MagicLinkService 魔术链接服务
type MagicLinkService struct {
	db      *gorm.DB
	baseURL string // 例如 https://tms.trackcard.com
}

var magicLinkService *MagicLinkService

// InitMagicLinkService 初始化魔术链接服务
func InitMagicLinkService(db *gorm.DB, baseURL string) {
	magicLinkService = &MagicLinkService{
		db:      db,
		baseURL: baseURL,
	}
}

// GetMagicLinkService 获取魔术链接服务实例
func GetMagicLinkService() *MagicLinkService {
	return magicLinkService
}

// CreateLink 创建魔术链接
func (s *MagicLinkService) CreateLink(req *models.CreateMagicLinkRequest) (*models.MagicLinkResponse, error) {
	// 默认有效期24小时
	expiresIn := 24
	if req.ExpiresIn > 0 {
		expiresIn = req.ExpiresIn
	}

	// 获取动作信息
	actionInfo, exists := models.ActionTypeInfo[req.ActionType]
	if !exists {
		return nil, fmt.Errorf("无效的动作类型: %s", req.ActionType)
	}

	link := &models.MagicLink{
		ShipmentID:  req.ShipmentID,
		TaskID:      req.TaskID,
		MilestoneID: req.MilestoneID,
		TargetRole:  req.TargetRole,
		TargetName:  req.TargetName,
		TargetPhone: req.TargetPhone,
		TargetEmail: req.TargetEmail,
		ActionType:  req.ActionType,
		ActionTitle: actionInfo.Title,
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn) * time.Hour),
	}

	if err := s.db.Create(link).Error; err != nil {
		return nil, err
	}

	return &models.MagicLinkResponse{
		ID:          link.ID,
		ShortURL:    fmt.Sprintf("%s/m/%s", s.baseURL, link.Token[:8]),
		FullURL:     fmt.Sprintf("%s/m/%s", s.baseURL, link.Token),
		Token:       link.Token,
		ActionType:  link.ActionType,
		ActionTitle: link.ActionTitle,
		ExpiresAt:   link.ExpiresAt,
		TargetPhone: link.TargetPhone,
		TargetEmail: link.TargetEmail,
	}, nil
}

// ValidateToken 验证Token并获取链接信息
func (s *MagicLinkService) ValidateToken(token string) (*models.MagicLink, error) {
	var link models.MagicLink

	// 支持短Token和完整Token
	if err := s.db.Where("token = ? OR token LIKE ?", token, token+"%").
		First(&link).Error; err != nil {
		return nil, fmt.Errorf("链接不存在或无效")
	}

	if link.IsUsed() {
		return nil, fmt.Errorf("链接已被使用")
	}

	if link.IsExpired() {
		return nil, fmt.Errorf("链接已过期")
	}

	return &link, nil
}

// GetActionPage 获取操作页面数据
func (s *MagicLinkService) GetActionPage(token string) (*models.MagicLinkActionPage, error) {
	link, err := s.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	actionInfo := models.ActionTypeInfo[link.ActionType]

	return &models.MagicLinkActionPage{
		ShipmentID:  link.ShipmentID,
		ActionType:  link.ActionType,
		ActionTitle: link.ActionTitle,
		Description: actionInfo.Description,
		NeedPhoto:   actionInfo.NeedPhoto,
		NeedGPS:     actionInfo.NeedGPS,
		TargetName:  link.TargetName,
	}, nil
}

// SubmitAction 处理用户提交的操作
func (s *MagicLinkService) SubmitAction(token string, req *models.SubmitMagicLinkRequest, clientIP string) error {
	link, err := s.ValidateToken(token)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 标记链接已使用
		now := time.Now()
		submittedJSON, _ := json.Marshal(req)

		if err := tx.Model(link).Updates(map[string]interface{}{
			"used_at":        now,
			"used_ip":        clientIP,
			"submitted_data": string(submittedJSON),
		}).Error; err != nil {
			return err
		}

		// 2. 根据动作类型更新对应的业务数据
		switch link.ActionType {
		case models.ActionConfirmPickup, models.ActionConfirmDelivery:
			// 更新运单节点状态
			if link.MilestoneID != nil {
				if err := tx.Model(&models.ShipmentMilestone{}).
					Where("id = ?", *link.MilestoneID).
					Updates(map[string]interface{}{
						"status":       models.StageStatusCompleted,
						"actual_end":   now,
						"trigger_type": models.TriggerAPI,
						"trigger_note": fmt.Sprintf("通过Magic Link完成 (IP: %s)", clientIP),
						"latitude":     req.Latitude,
						"longitude":    req.Longitude,
						"remarks":      req.Remarks,
					}).Error; err != nil {
					return err
				}
			}

		case models.ActionUploadPOD, models.ActionUploadDocument, models.ActionUploadCustomsDoc:
			// 创建文档记录
			if len(req.PhotoURLs) > 0 || len(req.Documents) > 0 {
				docURLs := append(req.PhotoURLs, req.Documents...)
				for _, url := range docURLs {
					// 根据动作类型决定文档类型
					var docType models.DocumentType
					switch link.ActionType {
					case models.ActionUploadPOD:
						docType = models.DocTypeOther // POD
					case models.ActionUploadCustomsDoc:
						docType = models.DocTypeCustomsDec
					default:
						docType = models.DocTypeOther
					}

					doc := &models.ShipmentDocument{
						ShipmentID: link.ShipmentID,
						DocType:    docType,
						DocName:    fmt.Sprintf("%s", link.ActionTitle),
						FileName:   fmt.Sprintf("%s_%s", link.ActionType, time.Now().Format("20060102150405")),
						FilePath:   url,
						UploaderID: fmt.Sprintf("magic_link:%s", link.TargetName),
						UploadedAt: time.Now(),
					}
					if err := tx.Create(doc).Error; err != nil {
						return err
					}
				}
			}

		case models.ActionConfirmCleared, models.ActionConfirmWarehouseIn, models.ActionConfirmWarehouseOut:
			// 更新节点状态为已完成
			if link.MilestoneID != nil {
				if err := tx.Model(&models.ShipmentMilestone{}).
					Where("id = ?", *link.MilestoneID).
					Updates(map[string]interface{}{
						"status":       models.StageStatusCompleted,
						"actual_end":   now,
						"trigger_type": models.TriggerAPI,
						"trigger_note": fmt.Sprintf("通过Magic Link确认 (%s)", link.TargetName),
						"remarks":      req.Remarks,
					}).Error; err != nil {
					return err
				}
			}

		case models.ActionReportLocation:
			// 记录位置上报
			if req.Latitude != 0 && req.Longitude != 0 {
				// 可以存入LocationHistory或其他位置表
				// 这里简化处理，记录到节点备注
				if link.MilestoneID != nil {
					tx.Model(&models.ShipmentMilestone{}).
						Where("id = ?", *link.MilestoneID).
						Updates(map[string]interface{}{
							"latitude":  req.Latitude,
							"longitude": req.Longitude,
							"remarks":   fmt.Sprintf("位置上报: %f, %f @ %s", req.Latitude, req.Longitude, now.Format("2006-01-02 15:04")),
						})
				}
			}
		}

		// 3. 如果关联了任务，标记任务完成（TODO: 待Task模型创建后启用）
		// if link.TaskID != nil {
		//     // 更新任务状态
		// }

		return nil
	})
}

// GetLinksByShipment 获取运单的所有魔术链接
func (s *MagicLinkService) GetLinksByShipment(shipmentID string) ([]models.MagicLink, error) {
	var links []models.MagicLink
	if err := s.db.Where("shipment_id = ?", shipmentID).
		Order("created_at DESC").
		Find(&links).Error; err != nil {
		return nil, err
	}
	return links, nil
}

// CleanupExpiredLinks 清理过期链接（定时任务调用）
func (s *MagicLinkService) CleanupExpiredLinks() (int64, error) {
	result := s.db.Where("expires_at < ? AND used_at IS NULL", time.Now().Add(-24*time.Hour)).
		Delete(&models.MagicLink{})
	return result.RowsAffected, result.Error
}
