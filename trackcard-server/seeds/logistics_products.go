package seeds

import (
	"log"

	"trackcard-server/models"

	"gorm.io/gorm"
)

// SeedLogisticsProducts 初始化物流产品和节点模板
func SeedLogisticsProducts(db *gorm.DB) error {
	// 检查是否已有数据
	var count int64
	db.Model(&models.LogisticsProduct{}).Count(&count)
	if count > 0 {
		log.Println("📦 Logistics products already seeded, skipping...")
		return nil
	}

	log.Println("🌱 Seeding logistics products and milestone templates...")

	// ========== 1. 海运整箱 FCL ==========
	seaFCL := &models.LogisticsProduct{
		Name:        "海运整箱 (FCL)",
		Code:        "sea_fcl",
		Description: "整箱海运服务，适用于大宗货物运输",
		IsActive:    true,
		SortOrder:   1,
	}
	if err := db.Create(seaFCL).Error; err != nil {
		return err
	}

	seaFCLTemplate := &models.MilestoneTemplate{
		ProductID:   seaFCL.ID,
		Name:        "标准海运整箱流程",
		Description: "包含前程运输、起运港、干线、目的港、末端配送5个核心环节，支持中转港",
		Version:     1,
		IsActive:    true,
		IsDefault:   true,
	}
	db.Create(seaFCLTemplate)

	seaFCLNodes := []models.MilestoneNode{
		// 前程运输
		{TemplateID: seaFCLTemplate.ID, NodeCode: "pickup", NodeName: "工厂提货", NodeNameEn: "Factory Pickup", NodeOrder: 1, GroupCode: strPtr("first_mile"), GroupName: strPtr("前程运输"), IsMandatory: true, IsVisible: true, Icon: "truck", TriggerType: "manual", ResponsibleRole: "trucker"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "container_loading", NodeName: "装柜完成", NodeNameEn: "Container Loaded", NodeOrder: 2, GroupCode: strPtr("first_mile"), GroupName: strPtr("前程运输"), IsMandatory: true, IsVisible: true, Icon: "container", TriggerType: "iot", ResponsibleRole: "shipper"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "yard_arrival", NodeName: "到达堆场", NodeNameEn: "Yard Arrival", NodeOrder: 3, GroupCode: strPtr("first_mile"), GroupName: strPtr("前程运输"), IsMandatory: true, IsVisible: true, Icon: "location", TriggerType: "geofence", ResponsibleRole: "trucker"},
		// 起运港
		{TemplateID: seaFCLTemplate.ID, NodeCode: "customs_export", NodeName: "出口报关", NodeNameEn: "Export Customs", NodeOrder: 4, GroupCode: strPtr("origin_port"), GroupName: strPtr("起运港"), IsMandatory: true, IsVisible: true, Icon: "file", TriggerType: "event", ResponsibleRole: "customs_broker"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "customs_cleared", NodeName: "报关放行", NodeNameEn: "Customs Cleared", NodeOrder: 5, GroupCode: strPtr("origin_port"), GroupName: strPtr("起运港"), IsMandatory: true, IsVisible: true, Icon: "check", TriggerType: "api", ResponsibleRole: "customs_broker"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "vessel_loading", NodeName: "装船完成", NodeNameEn: "Vessel Loaded", NodeOrder: 6, GroupCode: strPtr("origin_port"), GroupName: strPtr("起运港"), IsMandatory: true, IsVisible: true, Icon: "ship", TriggerType: "api", ResponsibleRole: "carrier"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "vessel_departed", NodeName: "船舶离港", NodeNameEn: "Vessel Departed", NodeOrder: 7, GroupCode: strPtr("origin_port"), GroupName: strPtr("起运港"), IsMandatory: true, IsVisible: true, Icon: "departure", TriggerType: "api", ResponsibleRole: "carrier"},
		// 干线（可选中转港）
		{TemplateID: seaFCLTemplate.ID, NodeCode: "transit_port_1", NodeName: "中转港1到达", NodeNameEn: "Transit Port 1 Arrival", NodeOrder: 8, GroupCode: strPtr("main_carriage"), GroupName: strPtr("干线运输"), IsMandatory: false, IsVisible: true, Icon: "anchor", TriggerType: "api", ResponsibleRole: "carrier"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "transit_port_1_departed", NodeName: "中转港1离港", NodeNameEn: "Transit Port 1 Departed", NodeOrder: 9, GroupCode: strPtr("main_carriage"), GroupName: strPtr("干线运输"), IsMandatory: false, IsVisible: true, Icon: "departure", TriggerType: "api", ResponsibleRole: "carrier"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "transit_port_2", NodeName: "中转港2到达", NodeNameEn: "Transit Port 2 Arrival", NodeOrder: 10, GroupCode: strPtr("main_carriage"), GroupName: strPtr("干线运输"), IsMandatory: false, IsVisible: true, Icon: "anchor", TriggerType: "api", ResponsibleRole: "carrier"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "transit_port_2_departed", NodeName: "中转港2离港", NodeNameEn: "Transit Port 2 Departed", NodeOrder: 11, GroupCode: strPtr("main_carriage"), GroupName: strPtr("干线运输"), IsMandatory: false, IsVisible: true, Icon: "departure", TriggerType: "api", ResponsibleRole: "carrier"},
		// 目的港
		{TemplateID: seaFCLTemplate.ID, NodeCode: "vessel_arrived", NodeName: "船舶抵港", NodeNameEn: "Vessel Arrived", NodeOrder: 12, GroupCode: strPtr("dest_port"), GroupName: strPtr("目的港"), IsMandatory: true, IsVisible: true, Icon: "arrival", TriggerType: "api", ResponsibleRole: "carrier"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "discharge", NodeName: "卸船完成", NodeNameEn: "Discharged", NodeOrder: 13, GroupCode: strPtr("dest_port"), GroupName: strPtr("目的港"), IsMandatory: true, IsVisible: true, Icon: "unload", TriggerType: "api", ResponsibleRole: "carrier"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "customs_import", NodeName: "进口报关", NodeNameEn: "Import Customs", NodeOrder: 14, GroupCode: strPtr("dest_port"), GroupName: strPtr("目的港"), IsMandatory: true, IsVisible: true, Icon: "file", TriggerType: "event", ResponsibleRole: "customs_broker", TimeoutHours: 72, TimeoutAction: "alert"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "import_cleared", NodeName: "清关放行", NodeNameEn: "Import Cleared", NodeOrder: 15, GroupCode: strPtr("dest_port"), GroupName: strPtr("目的港"), IsMandatory: true, IsVisible: true, Icon: "check", TriggerType: "api", ResponsibleRole: "customs_broker"},
		// 末端配送
		{TemplateID: seaFCLTemplate.ID, NodeCode: "pickup_dest", NodeName: "目的港提柜", NodeNameEn: "Container Pickup", NodeOrder: 16, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "truck", TriggerType: "manual", ResponsibleRole: "trucker"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "delivery", NodeName: "送达收货人", NodeNameEn: "Delivered", NodeOrder: 17, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "home", TriggerType: "geofence", ResponsibleRole: "trucker"},
		{TemplateID: seaFCLTemplate.ID, NodeCode: "pod_confirmed", NodeName: "签收确认", NodeNameEn: "POD Confirmed", NodeOrder: 18, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "signature", TriggerType: "manual", ResponsibleRole: "consignee"},
	}
	db.Create(&seaFCLNodes)

	// ========== 2. 国际空运 ==========
	airFreight := &models.LogisticsProduct{
		Name:        "国际空运",
		Code:        "air_freight",
		Description: "国际航空货运服务",
		IsActive:    true,
		SortOrder:   2,
	}
	db.Create(airFreight)

	airTemplate := &models.MilestoneTemplate{
		ProductID:   airFreight.ID,
		Name:        "标准空运流程",
		Description: "12个节点的空运标准流程",
		Version:     1,
		IsActive:    true,
		IsDefault:   true,
	}
	db.Create(airTemplate)

	airNodes := []models.MilestoneNode{
		{TemplateID: airTemplate.ID, NodeCode: "pickup", NodeName: "提货", NodeNameEn: "Pickup", NodeOrder: 1, GroupCode: strPtr("first_mile"), GroupName: strPtr("前程运输"), IsMandatory: true, IsVisible: true, Icon: "truck"},
		{TemplateID: airTemplate.ID, NodeCode: "warehouse_in", NodeName: "入仓", NodeNameEn: "Warehouse In", NodeOrder: 2, GroupCode: strPtr("first_mile"), GroupName: strPtr("前程运输"), IsMandatory: true, IsVisible: true, Icon: "warehouse"},
		{TemplateID: airTemplate.ID, NodeCode: "customs_export", NodeName: "出口报关", NodeNameEn: "Export Customs", NodeOrder: 3, GroupCode: strPtr("origin_airport"), GroupName: strPtr("起运机场"), IsMandatory: true, IsVisible: true, Icon: "file"},
		{TemplateID: airTemplate.ID, NodeCode: "flight_departed", NodeName: "航班起飞", NodeNameEn: "Flight Departed", NodeOrder: 4, GroupCode: strPtr("origin_airport"), GroupName: strPtr("起运机场"), IsMandatory: true, IsVisible: true, Icon: "plane-departure", TriggerType: "api"},
		{TemplateID: airTemplate.ID, NodeCode: "transit_airport", NodeName: "中转机场", NodeNameEn: "Transit Airport", NodeOrder: 5, GroupCode: strPtr("main_carriage"), GroupName: strPtr("干线运输"), IsMandatory: false, IsVisible: true, Icon: "plane"},
		{TemplateID: airTemplate.ID, NodeCode: "flight_arrived", NodeName: "航班抵达", NodeNameEn: "Flight Arrived", NodeOrder: 6, GroupCode: strPtr("dest_airport"), GroupName: strPtr("目的机场"), IsMandatory: true, IsVisible: true, Icon: "plane-arrival", TriggerType: "api"},
		{TemplateID: airTemplate.ID, NodeCode: "customs_import", NodeName: "进口报关", NodeNameEn: "Import Customs", NodeOrder: 7, GroupCode: strPtr("dest_airport"), GroupName: strPtr("目的机场"), IsMandatory: true, IsVisible: true, Icon: "file", TimeoutHours: 48, TimeoutAction: "alert"},
		{TemplateID: airTemplate.ID, NodeCode: "import_cleared", NodeName: "清关放行", NodeNameEn: "Import Cleared", NodeOrder: 8, GroupCode: strPtr("dest_airport"), GroupName: strPtr("目的机场"), IsMandatory: true, IsVisible: true, Icon: "check"},
		{TemplateID: airTemplate.ID, NodeCode: "warehouse_out", NodeName: "出仓", NodeNameEn: "Warehouse Out", NodeOrder: 9, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "warehouse"},
		{TemplateID: airTemplate.ID, NodeCode: "out_delivery", NodeName: "派送中", NodeNameEn: "Out for Delivery", NodeOrder: 10, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "truck"},
		{TemplateID: airTemplate.ID, NodeCode: "delivered", NodeName: "已签收", NodeNameEn: "Delivered", NodeOrder: 11, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "check-circle"},
		{TemplateID: airTemplate.ID, NodeCode: "pod_confirmed", NodeName: "POD确认", NodeNameEn: "POD Confirmed", NodeOrder: 12, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "signature"},
	}
	db.Create(&airNodes)

	// ========== 3. 陆地运输 ==========
	landTransport := &models.LogisticsProduct{
		Name:        "陆地运输",
		Code:        "land_transport",
		Description: "国内/跨境公路运输或铁路运输",
		IsActive:    true,
		SortOrder:   3,
	}
	db.Create(landTransport)

	landTemplate := &models.MilestoneTemplate{
		ProductID:   landTransport.ID,
		Name:        "标准陆运流程",
		Description: "公路/铁路运输标准流程",
		Version:     1,
		IsActive:    true,
		IsDefault:   true,
	}
	db.Create(landTemplate)

	landNodes := []models.MilestoneNode{
		{TemplateID: landTemplate.ID, NodeCode: "pickup", NodeName: "提货", NodeNameEn: "Pickup", NodeOrder: 1, GroupCode: strPtr("origin"), GroupName: strPtr("起运地"), IsMandatory: true, IsVisible: true, Icon: "truck", TriggerType: "manual"},
		{TemplateID: landTemplate.ID, NodeCode: "loading", NodeName: "装车完成", NodeNameEn: "Loaded", NodeOrder: 2, GroupCode: strPtr("origin"), GroupName: strPtr("起运地"), IsMandatory: true, IsVisible: true, Icon: "box", TriggerType: "iot"},
		{TemplateID: landTemplate.ID, NodeCode: "departed", NodeName: "发车", NodeNameEn: "Departed", NodeOrder: 3, GroupCode: strPtr("origin"), GroupName: strPtr("起运地"), IsMandatory: true, IsVisible: true, Icon: "departure", TriggerType: "geofence"},
		{TemplateID: landTemplate.ID, NodeCode: "in_transit", NodeName: "运输中", NodeNameEn: "In Transit", NodeOrder: 4, GroupCode: strPtr("transit"), GroupName: strPtr("运输途中"), IsMandatory: true, IsVisible: true, Icon: "road", TriggerType: "iot"},
		{TemplateID: landTemplate.ID, NodeCode: "border_crossing", NodeName: "边境口岸", NodeNameEn: "Border Crossing", NodeOrder: 5, GroupCode: strPtr("transit"), GroupName: strPtr("运输途中"), IsMandatory: false, IsVisible: true, Icon: "flag", TriggerType: "geofence"},
		{TemplateID: landTemplate.ID, NodeCode: "customs", NodeName: "口岸清关", NodeNameEn: "Customs Clearance", NodeOrder: 6, GroupCode: strPtr("transit"), GroupName: strPtr("运输途中"), IsMandatory: false, IsVisible: true, Icon: "file", TriggerType: "manual"},
		{TemplateID: landTemplate.ID, NodeCode: "arrived", NodeName: "到达目的地", NodeNameEn: "Arrived", NodeOrder: 7, GroupCode: strPtr("destination"), GroupName: strPtr("目的地"), IsMandatory: true, IsVisible: true, Icon: "arrival", TriggerType: "geofence"},
		{TemplateID: landTemplate.ID, NodeCode: "unloading", NodeName: "卸货中", NodeNameEn: "Unloading", NodeOrder: 8, GroupCode: strPtr("destination"), GroupName: strPtr("目的地"), IsMandatory: true, IsVisible: true, Icon: "unload"},
		{TemplateID: landTemplate.ID, NodeCode: "delivered", NodeName: "签收确认", NodeNameEn: "Delivered", NodeOrder: 9, GroupCode: strPtr("destination"), GroupName: strPtr("目的地"), IsMandatory: true, IsVisible: true, Icon: "signature", TriggerType: "manual"},
	}
	db.Create(&landNodes)

	// ========== 4. 多式联运 ==========
	multimodal := &models.LogisticsProduct{
		Name:        "多式联运",
		Code:        "multimodal",
		Description: "海运+空运+陆运组合运输方案",
		IsActive:    true,
		SortOrder:   4,
	}
	db.Create(multimodal)

	multimodalTemplate := &models.MilestoneTemplate{
		ProductID:   multimodal.ID,
		Name:        "海陆联运流程",
		Description: "海运+陆运组合，适用于内陆目的地",
		Version:     1,
		IsActive:    true,
		IsDefault:   true,
	}
	db.Create(multimodalTemplate)

	multimodalNodes := []models.MilestoneNode{
		// 首段陆运
		{TemplateID: multimodalTemplate.ID, NodeCode: "origin_pickup", NodeName: "工厂提货", NodeNameEn: "Factory Pickup", NodeOrder: 1, GroupCode: strPtr("first_mile"), GroupName: strPtr("首段陆运"), IsMandatory: true, IsVisible: true, Icon: "truck"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "origin_port_arrival", NodeName: "到达起运港", NodeNameEn: "Origin Port Arrival", NodeOrder: 2, GroupCode: strPtr("first_mile"), GroupName: strPtr("首段陆运"), IsMandatory: true, IsVisible: true, Icon: "anchor"},
		// 海运段
		{TemplateID: multimodalTemplate.ID, NodeCode: "export_customs", NodeName: "出口报关", NodeNameEn: "Export Customs", NodeOrder: 3, GroupCode: strPtr("sea_leg"), GroupName: strPtr("海运段"), IsMandatory: true, IsVisible: true, Icon: "file"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "vessel_loading", NodeName: "装船", NodeNameEn: "Vessel Loading", NodeOrder: 4, GroupCode: strPtr("sea_leg"), GroupName: strPtr("海运段"), IsMandatory: true, IsVisible: true, Icon: "ship"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "vessel_departed", NodeName: "船舶离港", NodeNameEn: "Vessel Departed", NodeOrder: 5, GroupCode: strPtr("sea_leg"), GroupName: strPtr("海运段"), IsMandatory: true, IsVisible: true, Icon: "departure", TriggerType: "api"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "vessel_arrived", NodeName: "船舶抵港", NodeNameEn: "Vessel Arrived", NodeOrder: 6, GroupCode: strPtr("sea_leg"), GroupName: strPtr("海运段"), IsMandatory: true, IsVisible: true, Icon: "arrival", TriggerType: "api"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "discharge", NodeName: "卸船完成", NodeNameEn: "Discharged", NodeOrder: 7, GroupCode: strPtr("sea_leg"), GroupName: strPtr("海运段"), IsMandatory: true, IsVisible: true, Icon: "unload"},
		// 换装陆运
		{TemplateID: multimodalTemplate.ID, NodeCode: "import_customs", NodeName: "进口清关", NodeNameEn: "Import Customs", NodeOrder: 8, GroupCode: strPtr("transload"), GroupName: strPtr("换装中转"), IsMandatory: true, IsVisible: true, Icon: "file"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "rail_loading", NodeName: "铁路/公路装载", NodeNameEn: "Rail/Truck Loading", NodeOrder: 9, GroupCode: strPtr("transload"), GroupName: strPtr("换装中转"), IsMandatory: true, IsVisible: true, Icon: "train"},
		// 内陆运输
		{TemplateID: multimodalTemplate.ID, NodeCode: "inland_transit", NodeName: "内陆运输中", NodeNameEn: "Inland Transit", NodeOrder: 10, GroupCode: strPtr("inland_leg"), GroupName: strPtr("内陆段"), IsMandatory: true, IsVisible: true, Icon: "road", TriggerType: "iot"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "inland_arrival", NodeName: "到达内陆场站", NodeNameEn: "Inland Terminal Arrival", NodeOrder: 11, GroupCode: strPtr("inland_leg"), GroupName: strPtr("内陆段"), IsMandatory: true, IsVisible: true, Icon: "location"},
		// 末端配送
		{TemplateID: multimodalTemplate.ID, NodeCode: "delivery", NodeName: "送货上门", NodeNameEn: "Door Delivery", NodeOrder: 12, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "home"},
		{TemplateID: multimodalTemplate.ID, NodeCode: "pod_confirmed", NodeName: "签收确认", NodeNameEn: "POD Confirmed", NodeOrder: 13, GroupCode: strPtr("last_mile"), GroupName: strPtr("末端配送"), IsMandatory: true, IsVisible: true, Icon: "signature"},
	}
	db.Create(&multimodalNodes)

	// ========== 5. 海运拼箱 LCL ==========
	seaLCL := &models.LogisticsProduct{
		Name:        "海运拼箱 (LCL)",
		Code:        "sea_lcl",
		Description: "拼箱海运服务，适用于小批量货物",
		IsActive:    true,
		SortOrder:   5,
	}
	db.Create(seaLCL)

	// LCL可复用FCL模板或创建简化版

	log.Printf("✅ Seeded 5 logistics products with milestone templates")
	return nil
}

func strPtr(s string) *string {
	return &s
}
