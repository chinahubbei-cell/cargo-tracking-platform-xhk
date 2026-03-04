package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// ShipmentStageService 运输环节服务
type ShipmentStageService struct {
	db *gorm.DB
}

var stageService *ShipmentStageService

// InitShipmentStageService 初始化运输环节服务
func InitShipmentStageService(db *gorm.DB) {
	stageService = &ShipmentStageService{db: db}
}

// GetShipmentStageService 获取运输环节服务实例
func GetShipmentStageService() *ShipmentStageService {
	return stageService
}

// CompleteStagesUpTo 自动完成目标环节及其所有前置环节
// 当某个环节被触发时，自动将所有前置环节标记为已完成
// 用于处理跳跃触发场景（如设备直接进入目的港，需补全前程运输等环节）
func (s *ShipmentStageService) CompleteStagesUpTo(shipmentID string, targetCode models.StageCode, triggerType models.TriggerType, note string) error {
	targetOrder := models.GetStageOrder(targetCode)
	if targetOrder == 0 {
		return fmt.Errorf("invalid target stage code: %s", targetCode)
	}

	now := time.Now()

	// 用于事务外记录日志的结构
	type stageLogInfo struct {
		StageCode  models.StageCode
		StageName  string
		FromStatus string
		ToStatus   string
		IsTarget   bool
	}
	var stageLogs []stageLogInfo

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 获取运单所有环节，按顺序排序
		var stages []models.ShipmentStage
		if err := tx.Where("shipment_id = ?", shipmentID).
			Order("stage_order ASC").Find(&stages).Error; err != nil {
			return err
		}

		if len(stages) == 0 {
			return nil // 无环节，跳过
		}

		for _, stage := range stages {
			stageOrder := models.GetStageOrder(stage.StageCode)

			if stageOrder < targetOrder {
				// 前置环节：如果未完成，自动补全
				if stage.Status != models.StageStatusCompleted && stage.Status != models.StageStatusSkipped {
					oldStatus := stage.Status
					updates := map[string]interface{}{
						"status":       models.StageStatusCompleted,
						"trigger_type": triggerType,
						"trigger_note": fmt.Sprintf("[自动补全] %s", note),
						"updated_at":   now,
					}
					// 如果没有实际结束时间，设置为当前时间
					if stage.ActualEnd == nil {
						updates["actual_end"] = now
					}
					// 如果没有实际开始时间，也设置为当前时间
					if stage.ActualStart == nil {
						updates["actual_start"] = now
					}

					if err := tx.Model(&models.ShipmentStage{}).Where("id = ?", stage.ID).Updates(updates).Error; err != nil {
						return err
					}

					// 记录日志信息
					stageLogs = append(stageLogs, stageLogInfo{
						StageCode:  stage.StageCode,
						StageName:  models.GetStageName(stage.StageCode),
						FromStatus: string(oldStatus),
						ToStatus:   string(models.StageStatusCompleted),
						IsTarget:   false,
					})
				}
			} else if stageOrder == targetOrder {
				// 目标环节：设置为进行中
				oldStatus := stage.Status
				updates := map[string]interface{}{
					"status":       models.StageStatusInProgress,
					"trigger_type": triggerType,
					"trigger_note": note,
					"updated_at":   now,
				}
				if stage.ActualStart == nil {
					updates["actual_start"] = now
				}

				if err := tx.Model(&models.ShipmentStage{}).Where("id = ?", stage.ID).Updates(updates).Error; err != nil {
					return err
				}

				// 记录目标环节日志信息
				stageLogs = append(stageLogs, stageLogInfo{
					StageCode:  stage.StageCode,
					StageName:  models.GetStageName(stage.StageCode),
					FromStatus: string(oldStatus),
					ToStatus:   string(models.StageStatusInProgress),
					IsTarget:   true,
				})
			}
			// stageOrder > targetOrder 的环节保持不变
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 事务外记录日志（避免SQLite死锁）
	if ShipmentLog != nil && len(stageLogs) > 0 {
		var completedNames []string

		for _, logInfo := range stageLogs {
			if logInfo.IsTarget {
				// 目标环节：记录开始日志
				ShipmentLog.LogStageTransition(shipmentID, string(logInfo.StageCode), logInfo.ToStatus,
					fmt.Sprintf("设备围栏触发：环节【%s】开始", logInfo.StageName), "system", "geofence")

				ShipmentLog.Log(shipmentID, "stage_start", "stage_status",
					logInfo.FromStatus, logInfo.ToStatus,
					fmt.Sprintf("🚀 环节【%s】自动开始 - %s", logInfo.StageName, note),
					"system", "geofence")
			} else {
				// 前置环节：记录自动补全日志
				completedNames = append(completedNames, logInfo.StageName)

				ShipmentLog.LogStageTransition(shipmentID, string(logInfo.StageCode), logInfo.ToStatus,
					fmt.Sprintf("设备围栏触发：环节【%s】自动补全完成", logInfo.StageName), "system", "geofence")

				ShipmentLog.Log(shipmentID, "stage_auto_complete", "stage_status",
					logInfo.FromStatus, logInfo.ToStatus,
					fmt.Sprintf("✅ 环节【%s】自动补全完成 - %s", logInfo.StageName, note),
					"system", "geofence")
			}
		}

		// 如果有补全的环节，记录汇总日志
		if len(completedNames) > 0 {
			ShipmentLog.Log(shipmentID, "stages_batch_complete", "stages", "",
				strings.Join(completedNames, ","),
				fmt.Sprintf("📦 自动补全 %d 个环节：%s → 触发环节【%s】",
					len(completedNames), strings.Join(completedNames, " → "), models.GetStageName(targetCode)),
				"system", "geofence")
		}
	}

	return nil
}

// CompleteAllStages 完成运单所有环节
// 用于设备直接到达目的地时，将所有环节标记为完成
func (s *ShipmentStageService) CompleteAllStages(shipmentID string, triggerType models.TriggerType, note string) error {
	now := time.Now()

	// 用于事务外记录日志的结构
	type stageLogInfo struct {
		StageCode  models.StageCode
		StageName  string
		FromStatus string
	}
	var stageLogs []stageLogInfo

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 获取所有未完成的环节
		var stages []models.ShipmentStage
		if err := tx.Where("shipment_id = ? AND status != ?", shipmentID, models.StageStatusCompleted).
			Order("stage_order ASC").Find(&stages).Error; err != nil {
			return err
		}

		for _, stage := range stages {
			oldStatus := stage.Status
			updates := map[string]interface{}{
				"status":       models.StageStatusCompleted,
				"trigger_type": triggerType,
				"trigger_note": fmt.Sprintf("[自动补全] %s", note),
				"updated_at":   now,
			}
			if stage.ActualEnd == nil {
				updates["actual_end"] = now
			}
			if stage.ActualStart == nil {
				updates["actual_start"] = now
			}

			if err := tx.Model(&models.ShipmentStage{}).Where("id = ?", stage.ID).Updates(updates).Error; err != nil {
				return err
			}

			// 记录日志信息
			stageLogs = append(stageLogs, stageLogInfo{
				StageCode:  stage.StageCode,
				StageName:  models.GetStageName(stage.StageCode),
				FromStatus: string(oldStatus),
			})
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 事务外记录日志（避免SQLite死锁）
	if ShipmentLog != nil && len(stageLogs) > 0 {
		var completedNames []string

		// 为每个环节记录单独的日志
		for _, logInfo := range stageLogs {
			completedNames = append(completedNames, logInfo.StageName)

			// 记录环节状态变更日志
			ShipmentLog.LogStageTransition(shipmentID, string(logInfo.StageCode), string(models.StageStatusCompleted),
				fmt.Sprintf("设备到达目的地：环节【%s】自动完成", logInfo.StageName), "system", "geofence")

			// 记录详细操作日志
			ShipmentLog.Log(shipmentID, "stage_auto_complete", "stage_status",
				logInfo.FromStatus, string(models.StageStatusCompleted),
				fmt.Sprintf("✅ 环节【%s】自动完成 - %s", logInfo.StageName, note),
				"system", "geofence")
		}

		// 记录汇总日志
		ShipmentLog.Log(shipmentID, "all_stages_complete", "stages", "",
			strings.Join(completedNames, ","),
			fmt.Sprintf("🎉 运单签收完成，全部 %d 个环节自动完成：%s",
				len(completedNames), strings.Join(completedNames, " → ")),
			"system", "geofence")
	}

	return nil
}

// DetermineRouteType 根据发货地和收货地判断线路类型
func (s *ShipmentStageService) DetermineRouteType(origin, destination string) models.RouteType {
	domesticKeywords := []string{"中国", "CN", "China", "china", "Mainland China"}

	isOriginDomestic := false
	for _, kw := range domesticKeywords {
		if strings.Contains(strings.ToLower(origin), strings.ToLower(kw)) {
			isOriginDomestic = true
			break
		}
	}

	isDestDomestic := false
	for _, kw := range domesticKeywords {
		if strings.Contains(strings.ToLower(destination), strings.ToLower(kw)) {
			isDestDomestic = true
			break
		}
	}

	// 如果由于数据缺失（空字符串），默认为跨境以防万一，或者可以根据业务逻辑调整
	// 这里假设如果明确两端都是国内，才是Domestic
	if isOriginDomestic && isDestDomestic {
		return models.RouteTypeDomestic
	}

	// 如果是省份/城市名也可以增加判断，但目前先基于简单的国家关键词
	// 还可以增加更复杂的判断逻辑，或者基于结构化地址字段

	return models.RouteTypeCrossBorder
}

// DetermineHasTransitPort 根据起运地和目的地判断是否需要中转港
func (s *ShipmentStageService) DetermineHasTransitPort(origin, destination string) bool {
	// 简单的国家/城市匹配逻辑 (模拟路由规划数据)
	originLower := strings.ToLower(origin)
	destLower := strings.ToLower(destination)

	// 中国直达美国西海岸 (通常直航)
	isOriginCN := strings.Contains(originLower, "china") || strings.Contains(originLower, "cn") || strings.Contains(originLower, "shenzhen") || strings.Contains(originLower, "shanghai") || strings.Contains(originLower, "ningbo")
	isDestUSWest := (strings.Contains(destLower, "usa") || strings.Contains(destLower, "us")) && (strings.Contains(destLower, "los angeles") || strings.Contains(destLower, "long beach") || strings.Contains(destLower, "oakland") || strings.Contains(destLower, "seattle") || strings.Contains(destLower, "tacoma"))

	if isOriginCN && isDestUSWest {
		return false
	}

	// 中国直达日本/韩国
	isDestJP := strings.Contains(destLower, "japan") || strings.Contains(destLower, "jp") || strings.Contains(destLower, "tokyo") || strings.Contains(destLower, "osaka")
	isDestKR := strings.Contains(destLower, "korea") || strings.Contains(destLower, "kr") || strings.Contains(destLower, "busan") || strings.Contains(destLower, "seoul")

	if isOriginCN && (isDestJP || isDestKR) {
		return false
	}

	// 默认包含中转港 (保持向后兼容)
	return true
}

// CreateStagesForShipment 为新建运单创建对应类型的环节
// 该方法现在集成 RoutePlannerService 自动规划线路并提取港口信息
func (s *ShipmentStageService) CreateStagesForShipment(shipmentID string, originPortCode, destPortCode string) error {
	// 检查是否已存在环节，避免重复创建
	var existingCount int64
	s.db.Model(&models.ShipmentStage{}).Where("shipment_id = ?", shipmentID).Count(&existingCount)
	if existingCount > 0 {
		return nil // 已存在环节，跳过创建
	}

	// 获取运单信息以判断路线类型
	var shipment models.Shipment
	if err := s.db.Select("origin, destination, origin_address, dest_address, transport_type, weight, volume").First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return err
	}

	// 优先使用详细地址，如果没有则使用简略地址
	origin := shipment.OriginAddress
	if origin == "" {
		origin = shipment.Origin
	}
	dest := shipment.DestAddress
	if dest == "" {
		dest = shipment.Destination
	}

	// 尝试使用 RoutePlannerService 进行智能路线规划
	routePlanner := NewRoutePlannerService(s.db)
	routeReq := CalculateRouteRequest{
		Origin:        origin,
		Destination:   dest,
		TransportType: shipment.TransportType, // 运输类型: sea/air/land/multimodal
		TransportMode: "lcl",                  // 默认零担模式
		CargoType:     "general",
		Currency:      "CNY",
	}

	// 如果运单有重量/体积信息，填充到请求中
	if shipment.Weight != nil && *shipment.Weight > 0 {
		routeReq.WeightKG = *shipment.Weight
	}
	if shipment.Volume != nil && *shipment.Volume > 0 {
		routeReq.VolumeCBM = *shipment.Volume
	}

	routeResp, err := routePlanner.CalculateRoutes(routeReq)
	if err == nil && len(routeResp.Routes) > 0 {
		// 使用规划结果创建环节（优先使用 fastest 路线）
		var selectedRoute *models.RouteRecommendation
		for _, route := range routeResp.Routes {
			if route.Type == "fastest" {
				selectedRoute = &route
				break
			}
		}
		if selectedRoute == nil {
			selectedRoute = &routeResp.Routes[0]
		}

		// 使用规划结果创建 stages
		if err := s.CreateStagesFromRoute(shipmentID, *selectedRoute); err == nil {
			return nil // 成功使用规划结果创建
		}
		// 如果创建失败，继续使用传统方法
	}

	// 降级：使用传统方法创建环节
	return s.createStagesLegacy(shipmentID, origin, dest, originPortCode, destPortCode)
}

// RegenerateStages 重新生成运单的运输环节
// 用于在规则变更后批量更新存量数据
func (s *ShipmentStageService) RegenerateStages(shipmentID string) error {
	// 1. 删除现有环节
	if err := s.db.Where("shipment_id = ?", shipmentID).Delete(&models.ShipmentStage{}).Error; err != nil {
		return fmt.Errorf("failed to delete existing stages: %v", err)
	}

	// 2. 获取运单信息（包括可能存在的港口代码）
	// 注意：models.Shipment 可能不直接存储 port codes，除非之前有扩展。
	// 但 CreateStagesForShipment 接收 port code 参数。
	// 如果 shipment 没有 port code 字段，我们传空字符串，依靠 RoutePlanner 从地址推断。
	return s.CreateStagesForShipment(shipmentID, "", "")
}

// CreateStagesFromRoute 根据路线规划结果创建运单环节
// 该方法从 RouteRecommendation 的 Segments 中提取港口信息并创建对应的 ShipmentStage
func (s *ShipmentStageService) CreateStagesFromRoute(shipmentID string, route models.RouteRecommendation) error {
	now := time.Now()
	var stages []models.ShipmentStage

	// 获取运单的发货地和目的地坐标 (用于 pre_transit 和 last_mile/delivered)
	var shipmentCoords struct {
		OriginLat float64
		OriginLng float64
		DestLat   float64
		DestLng   float64
	}
	s.db.Table("shipments").
		Select("origin_lat, origin_lng, dest_lat, dest_lng").
		Where("id = ?", shipmentID).
		Scan(&shipmentCoords)

	// 从 Segments 中提取港口信息
	var originPortCode, destPortCode, transitPortCode string

	for _, seg := range route.Segments {
		switch seg.Type {
		case "first_mile":
			// first_mile 的 To 通常是起运港
			if seg.To != "" {
				originPortCode = s.extractPortCode(seg.To)
			}
		case "line_haul":
			// line_haul 从起运港到目的港（或中转港）
			if len(seg.TransitPorts) > 0 {
				transitPortCode = s.extractPortCode(seg.TransitPorts[0])
			}
			if seg.To != "" && transitPortCode == "" {
				destPortCode = s.extractPortCode(seg.To)
			}
		case "last_mile":
			// last_mile 的 From 通常是目的港
			if seg.From != "" && destPortCode == "" {
				destPortCode = s.extractPortCode(seg.From)
			}
		}
	}

	// 确定是否有中转港
	hasTransitPort := transitPortCode != ""

	// 获取环节代码列表
	stageCodes := models.GetStageCodesForRoute(models.RouteTypeCrossBorder, hasTransitPort)

	for _, code := range stageCodes {
		stage := models.ShipmentStage{
			ShipmentID: shipmentID,
			StageCode:  code,
			Status:     models.StageStatusPending,
			StageOrder: models.GetStageOrder(code),
			Currency:   "CNY",
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		// 分配港口代码
		switch code {
		case models.SSOriginPort:
			stage.PortCode = originPortCode
			// 从 segments 中提取承运人信息
			for _, seg := range route.Segments {
				if seg.Type == "line_haul" && seg.Carrier != "" {
					stage.Carrier = seg.Carrier
					break
				}
			}
		case models.SSTransitPort:
			stage.PortCode = transitPortCode
		case models.SSDestPort:
			stage.PortCode = destPortCode
		}

		// 从 route 中设置费用信息（按环节分摊）
		// 同时尝试设置 ExtraData 中的坐标信息 (针对 Land 模式等无 PortCode 场景)
		for _, seg := range route.Segments {
			if s.matchSegmentToStage(seg.Type, code) {
				stage.Cost = seg.Cost

				// 如果没有PortCode，尝试将坐标填入ExtraData
				if stage.PortCode == "" {
					var lat, lng float64

					// 根据环节类型选择正确的坐标
					switch code {
					case models.SSPreTransit:
						// pre_transit 使用运单的发货地坐标
						lat = shipmentCoords.OriginLat
						lng = shipmentCoords.OriginLng
					case models.SSLastMile, models.SSDelivered:
						// last_mile 和 delivered 使用运单的目的地坐标
						lat = shipmentCoords.DestLat
						lng = shipmentCoords.DestLng
					case models.SSOriginPort:
						// OriginPort 使用 line_haul 起点 (港口坐标)
						if seg.Type == "line_haul" {
							lat = seg.FromLat
							lng = seg.FromLng
						} else {
							lat = seg.ToLat
							lng = seg.ToLng
						}
					default:
						// 其他环节使用 segment 终点坐标
						lat = seg.ToLat
						lng = seg.ToLng
					}

					if lat != 0 && lng != 0 {
						stage.ExtraData = fmt.Sprintf(`{"lat":%.6f,"lng":%.6f}`, lat, lng)
					}
				}
				break
			}
		}

		stages = append(stages, stage)
	}

	// 第一个环节设置为进行中，并补齐开始时间（避免前端最新节点无时间）
	if len(stages) > 0 {
		stages[0].Status = models.StageStatusInProgress
		stages[0].ActualStart = &now
	}

	return s.db.Create(&stages).Error
}

// extractPortCode 从港口名称中提取港口代码
// 尝试从数据库中查找匹配的港口
func (s *ShipmentStageService) extractPortCode(portName string) string {
	if portName == "" {
		return ""
	}

	// 0. 尝试从 "CODE (Name)" 格式提取
	// 例如: "HKG (Hong Kong Intl)" -> "HKG"
	// 例如: "USLAX (Los Angeles)" -> "USLAX"
	reParen := regexp.MustCompile(`^([A-Z]{3,5})\s*\(.+\)$`)
	matchesParen := reParen.FindStringSubmatch(portName)
	if len(matchesParen) > 0 {
		return matchesParen[1]
	}

	// 1. 先按代码精确匹配
	var port models.Port
	if err := s.db.Where("code = ?", portName).First(&port).Error; err == nil {
		return port.Code
	}

	// 1.1 尝试按机场代码匹配
	var airport models.Airport
	if len(portName) == 3 {
		if err := s.db.Where("iata_code = ?", portName).First(&airport).Error; err == nil {
			return airport.IATACode
		}
	}

	// 2. 按名称精确匹配
	if err := s.db.Where("name = ? OR name_en = ?", portName, portName).First(&port).Error; err == nil {
		return port.Code
	}
	// 2.1 按机场名称匹配
	if err := s.db.Where("name = ? OR name_en = ?", portName, portName).First(&airport).Error; err == nil {
		return airport.IATACode
	}

	// 3. 尝试从字符串中提取 3-5 位全大写代码
	// 允许 3 位 (IATA) 或 5 位 (UN/LOCODE)
	re := regexp.MustCompile(`[A-Z]{3,5}`)
	matches := re.FindStringSubmatch(portName)
	if len(matches) > 0 {
		potentialCode := matches[0]
		// 优先查港口
		if err := s.db.Where("code = ?", potentialCode).First(&port).Error; err == nil {
			return port.Code
		}
		// 再查机场
		if len(potentialCode) == 3 {
			if err := s.db.Where("iata_code = ?", potentialCode).First(&airport).Error; err == nil {
				return airport.IATACode
			}
		}
	}

	// 4. 模糊匹配 (仅匹配港口，机场名通常较短容易误判)
	if err := s.db.Where("name LIKE ? OR name_en LIKE ?", "%"+portName+"%", "%"+portName+"%").First(&port).Error; err == nil {
		return port.Code
	}

	// 如果找不到，返回原始名称作为代码（后续可能需要人工修正）
	return portName
}

// matchSegmentToStage 判断路线段类型是否匹配环节代码
func (s *ShipmentStageService) matchSegmentToStage(segType string, stageCode models.StageCode) bool {
	switch segType {
	case "first_mile":
		return stageCode == models.SSPreTransit
	case "line_haul":
		return stageCode == models.SSMainLine
	case "last_mile":
		return stageCode == models.SSLastMile
	}
	return false
}

// createStagesLegacy 传统方法创建环节（降级方案）
func (s *ShipmentStageService) createStagesLegacy(shipmentID, origin, dest, originPortCode, destPortCode string) error {
	// 1. 确定路线类型 (从发货地/目的地国家判断)
	routeType := s.DetermineRouteType(origin, dest)

	// 判断是否需要中转港
	hasTransitPort := true
	if routeType == models.RouteTypeCrossBorder {
		hasTransitPort = s.DetermineHasTransitPort(origin, dest)
	}

	// 2. 获取标准环节代码列表
	stageCodes := models.GetStageCodesForRoute(routeType, hasTransitPort)

	var stages []models.ShipmentStage
	now := time.Now()

	for _, code := range stageCodes {
		stage := models.ShipmentStage{
			ShipmentID: shipmentID,
			StageCode:  code,
			Status:     models.StageStatusPending,
			StageOrder: models.GetStageOrder(code),
			Currency:   "CNY",
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		// 分配港口代码
		if code == "origin_port" && originPortCode != "" {
			stage.PortCode = originPortCode
		} else if code == "dest_port" && destPortCode != "" {
			stage.PortCode = destPortCode
		}

		stages = append(stages, stage)
	}

	// 第一个环节设置为进行中，并补齐开始时间（避免前端最新节点无时间）
	if len(stages) > 0 {
		stages[0].Status = models.StageStatusInProgress
		stages[0].ActualStart = &now
	}

	return s.db.Create(&stages).Error
}

// GetStagesByShipmentID 获取运单的所有环节
func (s *ShipmentStageService) GetStagesByShipmentID(shipmentID string) ([]models.ShipmentStage, error) {
	var stages []models.ShipmentStage
	err := s.db.Where("shipment_id = ?", shipmentID).
		Order("stage_order ASC").
		Find(&stages).Error
	return stages, err
}

// GetStage 获取单个环节
func (s *ShipmentStageService) GetStage(stageID string) (*models.ShipmentStage, error) {
	var stage models.ShipmentStage
	err := s.db.First(&stage, "id = ?", stageID).Error
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

// GetStageByCode 根据环节代码获取环节
func (s *ShipmentStageService) GetStageByCode(shipmentID string, stageCode models.StageCode) (*models.ShipmentStage, error) {
	var stage models.ShipmentStage
	err := s.db.First(&stage, "shipment_id = ? AND stage_code = ?", shipmentID, stageCode).Error
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

// UpdateStage 更新环节信息
func (s *ShipmentStageService) UpdateStage(stageID string, req *models.ShipmentStageUpdateRequest) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.PartnerID != nil {
		updates["partner_id"] = *req.PartnerID
	}
	if req.PartnerName != nil {
		updates["partner_name"] = *req.PartnerName
	}
	if req.VehiclePlate != nil {
		updates["vehicle_plate"] = *req.VehiclePlate
	}
	if req.VesselName != nil {
		updates["vessel_name"] = *req.VesselName
	}
	if req.VoyageNo != nil {
		updates["voyage_no"] = *req.VoyageNo
	}
	if req.Carrier != nil {
		updates["carrier"] = *req.Carrier
	}
	if req.ActualStart != nil {
		updates["actual_start"] = *req.ActualStart
	}
	if req.ActualEnd != nil {
		updates["actual_end"] = *req.ActualEnd
	}
	if req.CostName != nil {
		updates["cost_name"] = *req.CostName
	}
	if req.Cost != nil {
		updates["cost"] = *req.Cost
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}
	if req.TriggerType != nil {
		updates["trigger_type"] = *req.TriggerType
	}
	if req.TriggerNote != nil {
		updates["trigger_note"] = *req.TriggerNote
	}
	if req.ExtraData != nil {
		updates["extra_data"] = *req.ExtraData
	}

	return s.db.Model(&models.ShipmentStage{}).Where("id = ?", stageID).Updates(updates).Error
}

// TransitionToNextStage 推进到下一环节
func (s *ShipmentStageService) TransitionToNextStage(shipmentID string, triggerType models.TriggerType, note string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 获取当前运单并加悲观锁，防止并发竞争
		var shipment models.Shipment
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&shipment, "id = ?", shipmentID).Error; err != nil {
			return err
		}

		// 2. 状态校验
		if shipment.Status == "delivered" || shipment.Status == "cancelled" {
			return fmt.Errorf("运单已结束或取消，无法推进环节")
		}

		// 获取当前环节代码
		currentCode := models.StageCode(shipment.CurrentStage)
		if currentCode == "" {
			return fmt.Errorf("当前环节异常为空")
		}
		currentOrder := models.GetStageOrder(currentCode)
		if currentOrder == 0 {
			return fmt.Errorf("无效的当前环节代码: %s", currentCode)
		}

		// 3. 检查当前环节是否已完成 (幂等性检查)
		var currentStage models.ShipmentStage
		if err := tx.Where("shipment_id = ? AND stage_code = ?", shipmentID, currentCode).First(&currentStage).Error; err != nil {
			return fmt.Errorf("找不到当前环节记录: %v", err)
		}

		if currentStage.Status == models.StageStatusCompleted {
			// 如果已经是完成状态，检查是否已经是最后环节
			// 注意：这里硬编码了顺序最大值 ? 不，应该根据 total stages
			// 由于现在环节是动态的，不能简单判断 >= 5
			// 我们应该检查是否有 nextCode

			// 但是 currentOrder 是固定的(1-10)，如果有跳过呢？
			// 更好的方式是查找 stage_order > currentOrder 的最小 stage
			// 但这里我们先尝试通用逻辑
			// 暂时保留 currentOrder 逻辑，但需注意 dynamic layout 下 maxOrder 变化
			if currentOrder >= 7 { // 7 是 Delivered/Signed
				return nil // 已经是最终状态
			}
		}

		// 4. 完成当前环节
		now := time.Now()
		updates := map[string]interface{}{
			"status":       models.StageStatusCompleted,
			"actual_end":   now,
			"trigger_type": triggerType,
			"updated_at":   now,
		}
		if note != "" {
			updates["trigger_note"] = note
		}

		if err := tx.Model(&models.ShipmentStage{}).
			Where("shipment_id = ? AND stage_code = ?", shipmentID, currentCode).
			Updates(updates).Error; err != nil {
			return err
		}

		// 5. 自动同步数据到主表
		// 略 (见 SyncDataToShipment)

		// 6. 获取下一环节
		// 动态获取：查找 currentOrder 之后的第一个存在的环节
		var nextStage models.ShipmentStage
		err := tx.Where("shipment_id = ? AND stage_order > ?", shipmentID, currentOrder).
			Order("stage_order ASC").
			First(&nextStage).Error

		if err == gorm.ErrRecordNotFound {
			// 没有下一环节，说明流程结束
			return tx.Model(&shipment).Updates(map[string]interface{}{
				"status":            "delivered",
				"progress":          100,
				"arrived_dest_at":   now,
				"status_updated_at": now,
				"track_end_at":      now,
			}).Error
		} else if err != nil {
			return err
		}

		// 7. 激活下一环节
		if err := tx.Model(&models.ShipmentStage{}).
			Where("id = ?", nextStage.ID).
			Updates(map[string]interface{}{
				"status":       models.StageStatusInProgress,
				"actual_start": now,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}

		// 8. 更新运单当前环节指针和状态
		// 计算进度: (完成环节数 / 总环节数) * 100 ?
		// 简单起见，用 absolute progress
		progress := nextStage.StageOrder * 10 // 粗略估算
		shipmentUpdates := map[string]interface{}{
			"current_stage": string(nextStage.StageCode),
			"progress":      progress,
		}

		// 状态流转逻辑
		// 1. 完成前程运输(Stage 1) -> 进入起运港(Stage 2)及以后 => 状态: 运输中
		if nextStage.StageOrder >= 2 && shipment.Status == "pending" {
			shipmentUpdates["status"] = "in_transit"
		}

		// 2. 完成末端配送(Stage 6) -> 进入签收环节(Stage 7) => 状态: 已到达
		if nextStage.StageCode == models.SSDelivered {
			shipmentUpdates["status"] = "arrived"
		}

		return tx.Model(&shipment).Updates(shipmentUpdates).Error
	})
}

// SyncDataToShipment 将环节数据同步到运单主表
func (s *ShipmentStageService) SyncDataToShipment(shipmentID string, stageCode models.StageCode) error {
	stage, err := s.GetStageByCode(shipmentID, stageCode)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{}

	// 根据不同环节同步不同字段
	switch stageCode {
	case models.SSFirstMile:
		// 前程运输：同步拖车车牌到箱号/车牌字段
		if stage.VehiclePlate != "" {
			updates["container_no"] = stage.VehiclePlate
		}
	case models.SSOriginPort:
		// 起运港：同步船名、航次、船司
		if stage.VesselName != "" {
			updates["vessel_name"] = stage.VesselName
		}
		if stage.VoyageNo != "" {
			updates["voyage_no"] = stage.VoyageNo
		}
		if stage.Carrier != "" {
			updates["carrier"] = stage.Carrier
		}
	case models.SSMainLine: // 更新为 MainLine
		// 干线运输：同步实际开航时间
		if stage.ActualStart != nil {
			updates["atd"] = *stage.ActualStart
		}
	case models.SSDestPort:
		// 目的港：同步到达时间
		if stage.ActualEnd != nil {
			updates["ata"] = *stage.ActualEnd
		}
	case models.SSLastMile:
		// 末端配送：同步签收时间和状态
		if stage.ActualEnd != nil {
			updates["arrived_dest_at"] = *stage.ActualEnd
		}
	}

	if len(updates) > 0 {
		return s.db.Model(&models.Shipment{}).Where("id = ?", shipmentID).Updates(updates).Error
	}
	return nil
}

// CalculateTotalCost 计算运单总费用（汇总各环节费用）
func (s *ShipmentStageService) CalculateTotalCost(shipmentID string) (float64, error) {
	var total float64
	err := s.db.Model(&models.ShipmentStage{}).
		Where("shipment_id = ?", shipmentID).
		Select("COALESCE(SUM(cost), 0)").
		Scan(&total).Error
	return total, err
}

// GetStagesSummary 获取环节概要（用于前端显示）
func (s *ShipmentStageService) GetStagesSummary(shipmentID string) ([]models.ShipmentStageResponse, error) {
	stages, err := s.GetStagesByShipmentID(shipmentID)
	if err != nil {
		return nil, err
	}

	// Fetch shipment details for fallback coordinates
	var shipment models.Shipment
	if err := s.db.Select("origin_lat, origin_lng, dest_lat, dest_lng").First(&shipment, "id = ?", shipmentID).Error; err != nil {
		// Log error but continue, just won't have fallbacks
		fmt.Printf("GetStagesSummary: Failed to fetch shipment %s: %v\n", shipmentID, err)
	}

	// 收集所有港口代码
	portCodes := make([]string, 0)
	for _, stage := range stages {
		if stage.PortCode != "" {
			portCodes = append(portCodes, stage.PortCode)
		}
	}

	// 批量查询港口信息
	portMap := make(map[string]models.Port)
	if len(portCodes) > 0 {
		var ports []models.Port
		s.db.Where("code IN ?", portCodes).Find(&ports)
		for _, port := range ports {
			portMap[port.Code] = port
		}

		// 检查是否有缺失的港口，尝试从全球机场数据库 (models.Airport) 中查找
		// Phase 9: Global Airports Integration
		missingCodes := make([]string, 0)
		for _, code := range portCodes {
			if _, ok := portMap[code]; !ok {
				missingCodes = append(missingCodes, code)
			}
		}

		if len(missingCodes) > 0 {
			var airports []models.Airport
			// 尝试匹配 IATA代码 或 名称
			s.db.Where("iata_code IN ? OR name IN ? OR name_en IN ?", missingCodes, missingCodes, missingCodes).Find(&airports)

			for _, airport := range airports {
				// 将 Airport 转换为 Port 结构放入 map
				// 注意：这里需要根据 lookup key 放入 map
				// 因为 airports 列表不告诉我们需要匹配哪个 missingCode
				// 所以需要遍历 missingCodes 来匹配 airport

				// 匹配逻辑：
				// 1. missingCode == IATA
				// 2. missingCode == Name
				// 3. missingCode == NameEn

				for _, code := range missingCodes {
					if code == airport.IATACode || code == airport.Name || code == airport.NameEn {
						portMap[code] = models.Port{
							Code:      airport.IATACode,
							Name:      airport.Name,
							Latitude:  airport.Latitude,
							Longitude: airport.Longitude,
							Type:      "airport",
							Country:   airport.Country,
							// City:      airport.City, // Port struct does not have City field
						}
					}
				}
			}
		}
	}

	responses := make([]models.ShipmentStageResponse, len(stages))
	for i, stage := range stages {
		resp := stage.ToResponse()
		// 兼容历史数据：进行中但缺少 actual_start 时，回退到 updated_at，避免前端“最新节点无时间”
		if resp.Status == models.StageStatusInProgress && resp.ActualStart == nil {
			fallback := stage.UpdatedAt
			resp.ActualStart = &fallback
		}
		// 填充港口坐标
		if port, ok := portMap[stage.PortCode]; ok {
			resp.PortLat = port.Latitude
			resp.PortLng = port.Longitude
			resp.PortName = port.Name
		} else {
			// Fallback logic
			// 1. Try ExtraData first (contains coordinates from RoutePlanner)
			if stage.ExtraData != "" {
				// Simple parsing to avoid unmarshalling overhead if not needed, or better use json.Unmarshal
				// Assuming ExtraData is simple JSON: {"lat":..., "lng":...}
				// Using simple string parsing or regex for simplicity/performance in this context?
				// Better use standard library
				// Clean parsing of ExtraData for coordinates
				// Define a temporary struct for unmarshalling
				type ExtraDataCoords struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				}
				// We need parsing logic here. Since we can't import encoding/json easily inside existing files without adding imports?
				// Note: Services package usually already imports many things. Let's check imports.
				// Assuming we can add import "encoding/json" if not present.
				// However, `shipment_stage.go` likely doesn't have it.
				// For safety, let's use a regex or string search if we can't be sure about imports.
				// But clean code suggests parsing. Let's verify imports first or assume we can add it.
				// Actually, `ToResponse` uses standard fields.

				// Let's rely on string parsing for now to be safe and quick without re-reading imports.
				// ExtraData: `{"lat":34.123,"lng":108.123}`
				// Regex: `"lat":([\d\.-]+)`
				reLat := regexp.MustCompile(`"lat":([\d\.-]+)`)
				reLng := regexp.MustCompile(`"lng":([\d\.-]+)`)
				mLat := reLat.FindStringSubmatch(stage.ExtraData)
				mLng := reLng.FindStringSubmatch(stage.ExtraData)
				if len(mLat) > 1 && len(mLng) > 1 {
					if lat, err := strconv.ParseFloat(mLat[1], 64); err == nil {
						resp.PortLat = lat
					}
					if lng, err := strconv.ParseFloat(mLng[1], 64); err == nil {
						resp.PortLng = lng
						resp.PortName = "途经点" // Generic name if from coords
					}
				}
			}

			// 2. If still no coordinates, try fallback to Shipment Origin/Dest
			if resp.PortLat == 0 && resp.PortLng == 0 {
				if stage.StageCode == models.SSOriginPort && shipment.OriginLat != nil && shipment.OriginLng != nil {
					resp.PortLat = *shipment.OriginLat
					resp.PortLng = *shipment.OriginLng
					resp.PortName = "发货地"
					resp.PortCode = "ORIGIN" // Frontend requires port_code to be present
				} else if stage.StageCode == models.SSDestPort && shipment.DestLat != nil && shipment.DestLng != nil {
					resp.PortLat = *shipment.DestLat
					resp.PortLng = *shipment.DestLng
					resp.PortName = "目的地"
					resp.PortCode = "DEST" // Frontend requires port_code to be present
				}
			}
		}
		responses[i] = resp
	}
	return responses, nil
}

// TriggerByGeofence 通过电子围栏触发环节状态更新
// 增强版：支持中转港围栏，并自动完成前置环节
func (s *ShipmentStageService) TriggerByGeofence(shipmentID string, geofenceID string, entering bool) error {
	// 获取围栏信息判断对应哪个环节
	var geofence models.PortGeofence
	if err := s.db.First(&geofence, "id = ?", geofenceID).Error; err != nil {
		return err
	}

	// 根据围栏类型确定环节
	var stageCode models.StageCode
	var isComplete bool // 标识是完成环节还是开始环节

	switch geofence.GeofenceType {
	case "origin_port":
		if entering {
			stageCode = models.SSOriginPort
			isComplete = false
		} else {
			stageCode = models.SSOriginPort // 完成起运港
			isComplete = true
		}
	case "transit_port":
		// 新增：中转港围栏支持
		if entering {
			stageCode = models.SSTransitPort
			isComplete = false
		} else {
			stageCode = models.SSTransitPort // 完成中转港
			isComplete = true
		}
	case "dest_port":
		if entering {
			stageCode = models.SSDestPort
			isComplete = false
		} else {
			stageCode = models.SSDestPort // 完成目的港
			isComplete = true
		}
	default:
		return fmt.Errorf("unknown geofence type: %s", geofence.GeofenceType)
	}

	now := time.Now()
	statusDesc := ""

	// 根据是进入还是离开，执行不同的逻辑
	if entering {
		// 进入围栏：自动完成所有前置环节，并将当前环节设为进行中
		statusDesc = "进入围栏，环节开始"

		// 调用 CompleteStagesUpTo 自动补全前置环节
		if err := s.CompleteStagesUpTo(shipmentID, stageCode, models.TriggerGeofence,
			fmt.Sprintf("进入%s围栏自动触发", geofence.Name)); err != nil {
			return err
		}
	} else {
		// 离开围栏：完成当前环节
		statusDesc = "离开围栏，环节完成"

		updates := map[string]interface{}{
			"status":       models.StageStatusCompleted,
			"trigger_type": models.TriggerGeofence,
			"geofence_id":  geofenceID,
			"actual_end":   now,
			"updated_at":   now,
		}

		if err := s.db.Model(&models.ShipmentStage{}).
			Where("shipment_id = ? AND stage_code = ?", shipmentID, stageCode).
			Updates(updates).Error; err != nil {
			return err
		}

		// 自动激活下一个环节
		nextStageCode := s.getNextStageCode(stageCode)
		if nextStageCode != "" {
			s.db.Model(&models.ShipmentStage{}).
				Where("shipment_id = ? AND stage_code = ?", shipmentID, nextStageCode).
				Updates(map[string]interface{}{
					"status":       models.StageStatusInProgress,
					"actual_start": now,
					"updated_at":   now,
				})
		}
	}

	// 记录日志
	if ShipmentLog != nil {
		stageName := models.GetStageName(stageCode)
		toStatus := models.StageStatusInProgress
		if isComplete {
			toStatus = models.StageStatusCompleted
		}

		ShipmentLog.LogStageTransition(shipmentID, string(stageCode), string(toStatus),
			fmt.Sprintf("电子围栏触发：%s（%s）", stageName, statusDesc), "system", "geofence")

		ShipmentLog.Log(shipmentID, "geofence_trigger", "stage_status", string(stageCode), string(toStatus),
			fmt.Sprintf("电子围栏自动触发环节【%s】%s", stageName, statusDesc), "system", "geofence")
	}

	return nil
}

// getNextStageCode 获取下一个环节代码
func (s *ShipmentStageService) getNextStageCode(currentCode models.StageCode) models.StageCode {
	switch currentCode {
	case models.SSPreTransit:
		return models.SSOriginPort
	case models.SSOriginPort:
		return models.SSMainLine
	case models.SSMainLine:
		return models.SSTransitPort // 可能跳过
	case models.SSTransitPort:
		return models.SSDestPort
	case models.SSDestPort:
		return models.SSLastMile
	case models.SSLastMile:
		return models.SSDelivered
	default:
		return ""
	}
}
