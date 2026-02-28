package models

import "time"

// ==================== OCR 识别类型 ====================

// OCRDocumentType OCR识别的文档类型
type OCRDocumentType string

const (
	OCRDocPackingList OCRDocumentType = "packing_list" // 装箱单
	OCRDocInvoice     OCRDocumentType = "invoice"      // 商业发票
	OCRDocBOL         OCRDocumentType = "bol"          // 提单
	OCRDocCustomsDec  OCRDocumentType = "customs_dec"  // 报关单
	OCRDocGeneral     OCRDocumentType = "general"      // 通用文档
)

// ==================== 装箱单识别结果 ====================

// PackingListItem 装箱单明细项
type PackingListItem struct {
	ItemNo      string  `json:"item_no"`      // 序号
	Description string  `json:"description"`  // 品名描述
	Quantity    int     `json:"quantity"`     // 数量
	Unit        string  `json:"unit"`         // 单位
	GrossWeight float64 `json:"gross_weight"` // 毛重(kg)
	NetWeight   float64 `json:"net_weight"`   // 净重(kg)
	CBM         float64 `json:"cbm"`          // 体积(立方米)
	CartonNo    string  `json:"carton_no"`    // 箱号
	Remarks     string  `json:"remarks"`      // 备注
}

// PackingListData 装箱单识别数据
type PackingListData struct {
	// 基础信息
	PONumber     string `json:"po_number"`     // PO号
	InvoiceNo    string `json:"invoice_no"`    // 发票号
	ContainerNo  string `json:"container_no"`  // 柜号
	SealNo       string `json:"seal_no"`       // 封条号
	ShipmentDate string `json:"shipment_date"` // 发运日期
	Destination  string `json:"destination"`   // 目的地

	// 发货方信息
	ShipperName    string `json:"shipper_name"`
	ShipperAddress string `json:"shipper_address"`

	// 收货方信息
	ConsigneeName    string `json:"consignee_name"`
	ConsigneeAddress string `json:"consignee_address"`

	// 明细
	Items []PackingListItem `json:"items"`

	// 汇总
	TotalCartons     int     `json:"total_cartons"`      // 总箱数
	TotalQuantity    int     `json:"total_quantity"`     // 总数量
	TotalGrossWeight float64 `json:"total_gross_weight"` // 总毛重
	TotalNetWeight   float64 `json:"total_net_weight"`   // 总净重
	TotalCBM         float64 `json:"total_cbm"`          // 总体积

	// 识别元数据
	Confidence float64 `json:"confidence"` // 整体置信度 0-1
}

// ==================== 商业发票识别结果 ====================

// InvoiceItem 发票明细项
type InvoiceItem struct {
	ItemNo      string  `json:"item_no"`
	Description string  `json:"description"`
	HSCode      string  `json:"hs_code"` // 海关编码
	Quantity    int     `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"` // 单价
	Currency    string  `json:"currency"`   // 币种
	Amount      float64 `json:"amount"`     // 金额
	Origin      string  `json:"origin"`     // 原产地
}

// InvoiceData 发票识别数据
type InvoiceData struct {
	// 基础信息
	InvoiceNo   string `json:"invoice_no"`
	InvoiceDate string `json:"invoice_date"`
	PONumber    string `json:"po_number"`

	// 发票方信息
	SellerName    string `json:"seller_name"`
	SellerAddress string `json:"seller_address"`
	SellerTaxNo   string `json:"seller_tax_no"`

	// 购买方信息
	BuyerName    string `json:"buyer_name"`
	BuyerAddress string `json:"buyer_address"`

	// 明细
	Items []InvoiceItem `json:"items"`

	// 汇总
	Currency      string  `json:"currency"`
	SubTotal      float64 `json:"sub_total"`
	Tax           float64 `json:"tax"`
	TotalAmount   float64 `json:"total_amount"`
	AmountInWords string  `json:"amount_in_words"` // 金额大写

	// 贸易条款
	PaymentTerms    string `json:"payment_terms"`  // 付款条款
	DeliveryTerms   string `json:"delivery_terms"` // 交货条款 (FOB/CIF等)
	PortOfLoading   string `json:"port_of_loading"`
	PortOfDischarge string `json:"port_of_discharge"`

	// 识别元数据
	Confidence float64 `json:"confidence"`
}

// ==================== 提单识别结果 ====================

// BOLData 提单识别数据
type BOLData struct {
	// 基础信息
	BLNumber   string `json:"bl_number"`
	BookingNo  string `json:"booking_no"`
	VesselName string `json:"vessel_name"`
	Voyage     string `json:"voyage"`

	// 港口信息
	PortOfLoading   string `json:"port_of_loading"`
	PortOfDischarge string `json:"port_of_discharge"`
	PlaceOfDelivery string `json:"place_of_delivery"`
	PlaceOfReceipt  string `json:"place_of_receipt"`

	// 日期
	OnBoardDate string `json:"on_board_date"`
	IssueDate   string `json:"issue_date"`
	IssuePlace  string `json:"issue_place"`

	// 当事方
	ShipperName    string `json:"shipper_name"`
	ShipperAddress string `json:"shipper_address"`
	ConsigneeName  string `json:"consignee_name"`
	NotifyParty    string `json:"notify_party"`

	// 货物信息
	Description  string  `json:"description"`
	PackageCount int     `json:"package_count"`
	PackageType  string  `json:"package_type"`
	GrossWeight  float64 `json:"gross_weight"`
	Measurement  float64 `json:"measurement"` // CBM
	ContainerNo  string  `json:"container_no"`
	SealNo       string  `json:"seal_no"`

	// 运费
	FreightTerms  string  `json:"freight_terms"` // PREPAID/COLLECT
	FreightAmount float64 `json:"freight_amount"`

	// 识别元数据
	Confidence float64 `json:"confidence"`
}

// ==================== OCR识别记录 ====================

// OCRRecognitionRecord OCR识别记录
type OCRRecognitionRecord struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	ShipmentID   *string         `gorm:"type:varchar(50);index" json:"shipment_id"`
	DocumentType OCRDocumentType `gorm:"type:varchar(30)" json:"document_type"`
	FileName     string          `gorm:"type:varchar(200)" json:"file_name"`
	FileSize     int64           `json:"file_size"`
	FilePath     string          `gorm:"type:varchar(500)" json:"file_path"`

	// 识别结果
	RawResult    string  `gorm:"type:text" json:"-"`             // 原始OCR返回
	ParsedResult string  `gorm:"type:text" json:"parsed_result"` // 解析后的结构化JSON
	Confidence   float64 `json:"confidence"`

	// 状态
	Status      string     `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending/processing/completed/failed
	ErrorMsg    string     `gorm:"type:varchar(500)" json:"error_msg,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	// 确认状态
	IsConfirmed   bool       `gorm:"default:false" json:"is_confirmed"`
	ConfirmedBy   *string    `gorm:"type:varchar(50)" json:"confirmed_by,omitempty"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
	ConfirmedData string     `gorm:"type:text" json:"confirmed_data,omitempty"` // 用户修正后的数据

	// 调用方
	CreatedBy string    `gorm:"type:varchar(50)" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (OCRRecognitionRecord) TableName() string {
	return "ocr_recognition_records"
}

// ==================== API 请求/响应结构 ====================

// OCRRecognizeRequest OCR识别请求
type OCRRecognizeRequest struct {
	DocumentType OCRDocumentType `json:"document_type" binding:"required"`
	ShipmentID   *string         `json:"shipment_id"`
	// 文件通过 multipart/form-data 上传
}

// OCRRecognizeResponse OCR识别响应
type OCRRecognizeResponse struct {
	RecordID     uint            `json:"record_id"`
	DocumentType OCRDocumentType `json:"document_type"`
	Confidence   float64         `json:"confidence"`
	Status       string          `json:"status"`

	// 根据类型返回不同的数据
	PackingList *PackingListData `json:"packing_list,omitempty"`
	Invoice     *InvoiceData     `json:"invoice,omitempty"`
	BOL         *BOLData         `json:"bol,omitempty"`
}

// OCRConfirmRequest 确认OCR结果请求
type OCRConfirmRequest struct {
	// 用户可以修正识别结果
	CorrectedData interface{} `json:"corrected_data"`
	// 是否直接导入到运单
	ImportToShipment bool `json:"import_to_shipment"`
}
