package services

import (
	"fmt"
	"sync"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ShipmentIDGenerator 运单号生成器
type ShipmentIDGenerator struct {
	db    *gorm.DB
	mutex sync.Mutex
}

var (
	idGenerator     *ShipmentIDGenerator
	idGeneratorOnce sync.Once
)

// InitShipmentIDGenerator 初始化运单号生成器
func InitShipmentIDGenerator(db *gorm.DB) {
	idGeneratorOnce.Do(func() {
		idGenerator = &ShipmentIDGenerator{
			db: db,
		}
	})
}

// GetShipmentIDGenerator 获取运单号生成器实例
func GetShipmentIDGenerator() *ShipmentIDGenerator {
	return idGenerator
}

// GenerateID 生成新的运单号
// 格式: YYMMDDXXXXXX (12位全数字)
// 示例: 260116000001 (2026年1月16日第1单)
func (g *ShipmentIDGenerator) GenerateID() (string, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	now := time.Now()
	datePrefix := now.Format("060102") // YYMMDD格式

	var sequence models.ShipmentSequence

	// 使用事务确保原子性
	err := g.db.Transaction(func(tx *gorm.DB) error {
		// 尝试获取或创建当日序号记录（使用行锁）
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("date_prefix = ?", datePrefix).
			First(&sequence)

		if result.Error == gorm.ErrRecordNotFound {
			// 当日第一单，创建新记录
			sequence = models.ShipmentSequence{
				DatePrefix:   datePrefix,
				LastSequence: 1,
				UpdatedAt:    now,
			}
			if err := tx.Create(&sequence).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else {
			// 递增序号
			sequence.LastSequence++
			sequence.UpdatedAt = now
			if err := tx.Save(&sequence).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("生成运单号失败: %w", err)
	}

	// 格式化运单号: YYMMDD + 6位序号(左补零)
	shipmentID := fmt.Sprintf("%s%06d", datePrefix, sequence.LastSequence)
	return shipmentID, nil
}

// MigrateExistingShipments 迁移现有运单号为新格式
func (g *ShipmentIDGenerator) MigrateExistingShipments() error {
	var shipments []models.Shipment

	// 查找所有非新格式的运单（新格式为12位纯数字）
	if err := g.db.Where("LENGTH(id) != 12 OR id NOT REGEXP '^[0-9]+$'").Find(&shipments).Error; err != nil {
		// 如果REGEXP不支持，使用简单条件
		if err := g.db.Where("id LIKE 'SH-%'").Find(&shipments).Error; err != nil {
			return err
		}
	}

	for _, s := range shipments {
		newID, err := g.GenerateID()
		if err != nil {
			return fmt.Errorf("迁移运单 %s 失败: %w", s.ID, err)
		}

		// 更新运单ID
		if err := g.db.Model(&models.Shipment{}).Where("id = ?", s.ID).Update("id", newID).Error; err != nil {
			return fmt.Errorf("更新运单ID失败: %w", err)
		}
	}

	return nil
}
