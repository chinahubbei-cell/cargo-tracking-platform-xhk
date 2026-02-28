package handlers

import (
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/utils"
)

type RateHandler struct {
	db *gorm.DB
}

func NewRateHandler(db *gorm.DB) *RateHandler {
	return &RateHandler{db: db}
}

// List 获取运价列表
func (h *RateHandler) List(c *gin.Context) {
	var rates []models.FreightRate
	query := h.db.Model(&models.FreightRate{}).Preload("Partner")

	// 起运港筛选
	if origin := c.Query("origin"); origin != "" {
		query = query.Where("origin = ?", origin)
	}

	// 目的港筛选
	if destination := c.Query("destination"); destination != "" {
		query = query.Where("destination = ?", destination)
	}

	// 货代筛选
	if partnerID := c.Query("partner_id"); partnerID != "" {
		query = query.Where("partner_id = ?", partnerID)
	}

	// 柜型筛选
	if containerType := c.Query("container_type"); containerType != "" {
		query = query.Where("container_type = ?", containerType)
	}

	// 只显示有效的
	if active := c.Query("active"); active == "true" {
		now := time.Now()
		query = query.Where("is_active = ? AND valid_from <= ? AND valid_to >= ?", true, now, now)
	}

	// 组织筛选
	if orgID := c.Query("org_id"); orgID != "" {
		query = query.Where("owner_org_id = ?", orgID)
	}

	query = query.Order("total_fee ASC")

	if err := query.Find(&rates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	var responses []models.RateResponse
	for _, r := range rates {
		responses = append(responses, r.ToResponse())
	}

	utils.SuccessResponse(c, responses)
}

// Get 获取单个运价详情
func (h *RateHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var rate models.FreightRate
	if err := h.db.Preload("Partner").First(&rate, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运价不存在")
		return
	}

	utils.SuccessResponse(c, rate.ToResponse())
}

// Create 创建运价
func (h *RateHandler) Create(c *gin.Context) {
	var req models.RateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供有效的运价信息")
		return
	}

	// 验证货代存在
	var partner models.Partner
	if err := h.db.First(&partner, "id = ?", req.PartnerID).Error; err != nil {
		utils.BadRequest(c, "货代不存在")
		return
	}

	// 获取组织ID
	orgID := ""
	if id, exists := c.Get("org_id"); exists {
		orgID = id.(string)
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	rate := models.FreightRate{
		PartnerID:       req.PartnerID,
		Origin:          req.Origin,
		OriginName:      req.OriginName,
		Destination:     req.Destination,
		DestinationName: req.DestinationName,
		TransitDays:     req.TransitDays,
		Carrier:         req.Carrier,
		ContainerType:   req.ContainerType,
		Currency:        currency,
		OceanFreight:    req.OceanFreight,
		BAF:             req.BAF,
		CAF:             req.CAF,
		PSS:             req.PSS,
		GRI:             req.GRI,
		THC:             req.THC,
		DocFee:          req.DocFee,
		SealFee:         req.SealFee,
		OtherFee:        req.OtherFee,
		ValidFrom:       req.ValidFrom,
		ValidTo:         req.ValidTo,
		IsActive:        true,
		Remarks:         req.Remarks,
		OwnerOrgID:      orgID,
	}

	if err := h.db.Create(&rate).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.Preload("Partner").First(&rate, rate.ID)
	utils.CreatedResponse(c, rate.ToResponse())
}

// Update 更新运价 (白名单字段)
func (h *RateHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var rate models.FreightRate
	if err := h.db.First(&rate, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运价不存在")
		return
	}

	// 使用白名单结构体防止任意字段更新
	type RateUpdateRequest struct {
		OriginName      *string    `json:"origin_name"`
		DestinationName *string    `json:"destination_name"`
		TransitDays     *int       `json:"transit_days"`
		Carrier         *string    `json:"carrier"`
		OceanFreight    *float64   `json:"ocean_freight"`
		BAF             *float64   `json:"baf"`
		CAF             *float64   `json:"caf"`
		PSS             *float64   `json:"pss"`
		GRI             *float64   `json:"gri"`
		THC             *float64   `json:"thc"`
		DocFee          *float64   `json:"doc_fee"`
		SealFee         *float64   `json:"seal_fee"`
		OtherFee        *float64   `json:"other_fee"`
		ValidFrom       *time.Time `json:"valid_from"`
		ValidTo         *time.Time `json:"valid_to"`
		IsActive        *bool      `json:"is_active"`
		Remarks         *string    `json:"remarks"`
	}

	var req RateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.OriginName != nil {
		updates["origin_name"] = *req.OriginName
	}
	if req.DestinationName != nil {
		updates["destination_name"] = *req.DestinationName
	}
	if req.TransitDays != nil {
		updates["transit_days"] = *req.TransitDays
	}
	if req.Carrier != nil {
		updates["carrier"] = *req.Carrier
	}
	if req.OceanFreight != nil {
		updates["ocean_freight"] = *req.OceanFreight
	}
	if req.BAF != nil {
		updates["baf"] = *req.BAF
	}
	if req.CAF != nil {
		updates["caf"] = *req.CAF
	}
	if req.PSS != nil {
		updates["pss"] = *req.PSS
	}
	if req.GRI != nil {
		updates["gri"] = *req.GRI
	}
	if req.THC != nil {
		updates["thc"] = *req.THC
	}
	if req.DocFee != nil {
		updates["doc_fee"] = *req.DocFee
	}
	if req.SealFee != nil {
		updates["seal_fee"] = *req.SealFee
	}
	if req.OtherFee != nil {
		updates["other_fee"] = *req.OtherFee
	}
	if req.ValidFrom != nil {
		updates["valid_from"] = *req.ValidFrom
	}
	if req.ValidTo != nil {
		updates["valid_to"] = *req.ValidTo
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Remarks != nil {
		updates["remarks"] = *req.Remarks
	}

	if len(updates) == 0 {
		utils.BadRequest(c, "没有提供要更新的字段")
		return
	}

	if err := h.db.Model(&rate).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.Preload("Partner").First(&rate, id)
	utils.SuccessResponse(c, rate.ToResponse())
}

// Delete 删除运价
func (h *RateHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	var rate models.FreightRate
	if err := h.db.First(&rate, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "运价不存在")
		return
	}

	if err := h.db.Delete(&rate).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// Compare 智能比价 - 根据航线和柜型返回排序后的运价列表
func (h *RateHandler) Compare(c *gin.Context) {
	var req models.RateQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供起运港和目的港")
		return
	}

	now := time.Now()
	if req.ShipDate != nil {
		now = *req.ShipDate
	}

	var rates []models.FreightRate
	query := h.db.Model(&models.FreightRate{}).
		Preload("Partner").
		Where("origin = ? AND destination = ?", req.Origin, req.Destination).
		Where("is_active = ? AND valid_from <= ? AND valid_to >= ?", true, now, now)

	if req.ContainerType != "" {
		query = query.Where("container_type = ?", req.ContainerType)
	}

	if err := query.Find(&rates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 按总价排序
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].TotalFee < rates[j].TotalFee
	})

	// 添加排名和价差信息
	type RateWithRank struct {
		models.RateResponse
		Rank           int     `json:"rank"`
		PriceDiff      float64 `json:"price_diff"`     // 与最低价差
		PriceDiffPct   float64 `json:"price_diff_pct"` // 差价百分比
		Recommendation string  `json:"recommendation"` // 推荐说明
	}

	var results []RateWithRank
	lowestPrice := float64(0)
	if len(rates) > 0 {
		lowestPrice = rates[0].TotalFee
	}

	for i, r := range rates {
		resp := r.ToResponse()
		rank := RateWithRank{
			RateResponse: resp,
			Rank:         i + 1,
			PriceDiff:    resp.TotalFee - lowestPrice,
			PriceDiffPct: 0,
		}

		if lowestPrice > 0 {
			rank.PriceDiffPct = (resp.TotalFee - lowestPrice) / lowestPrice * 100
		}

		// 生成推荐说明
		if i == 0 {
			rank.Recommendation = "💰 最低价"
		} else if rank.PriceDiffPct <= 5 {
			rank.Recommendation = "✅ 性价比高"
		} else if r.TransitDays > 0 && len(rates) > 0 && r.TransitDays < rates[0].TransitDays {
			rank.Recommendation = "⚡ 时效快"
		}

		results = append(results, rank)
	}

	utils.SuccessResponse(c, gin.H{
		"origin":         req.Origin,
		"destination":    req.Destination,
		"container_type": req.ContainerType,
		"total_options":  len(results),
		"lowest_price":   lowestPrice,
		"rates":          results,
	})
}

// GetPerformance 获取货代航线绩效
func (h *RateHandler) GetPerformance(c *gin.Context) {
	partnerID := c.Query("partner_id")
	routeLane := c.Query("route_lane")

	query := h.db.Model(&models.PartnerPerformance{}).Preload("Partner")

	if partnerID != "" {
		query = query.Where("partner_id = ?", partnerID)
	}
	if routeLane != "" {
		query = query.Where("route_lane = ?", routeLane)
	}

	var performances []models.PartnerPerformance
	if err := query.Find(&performances).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, performances)
}

// GetRoutes 获取可用航线列表 (起运港-目的港组合)
func (h *RateHandler) GetRoutes(c *gin.Context) {
	type RouteInfo struct {
		Origin          string  `json:"origin"`
		OriginName      string  `json:"origin_name"`
		Destination     string  `json:"destination"`
		DestinationName string  `json:"destination_name"`
		OptionsCount    int     `json:"options_count"`
		LowestPrice     float64 `json:"lowest_price"`
	}

	var routes []RouteInfo
	now := time.Now()

	h.db.Model(&models.FreightRate{}).
		Select("origin, origin_name, destination, destination_name, COUNT(*) as options_count, MIN(total_fee) as lowest_price").
		Where("is_active = ? AND valid_from <= ? AND valid_to >= ?", true, now, now).
		Group("origin, origin_name, destination, destination_name").
		Order("origin, destination").
		Scan(&routes)

	utils.SuccessResponse(c, routes)
}
