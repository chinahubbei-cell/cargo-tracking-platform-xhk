package models

import (
	"time"

	"gorm.io/gorm"
)

// DocumentTemplate 文档模板
type DocumentTemplate struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Code         string `gorm:"type:varchar(50);uniqueIndex" json:"code"` // packing_list, invoice
	Name         string `gorm:"type:varchar(100)" json:"name"`
	TemplateType string `gorm:"type:varchar(20)" json:"template_type"` // html, json
	Content      string `gorm:"type:text" json:"content"`              // 模板内容
	Description  string `gorm:"type:varchar(500)" json:"description"`
	IsActive     bool   `gorm:"default:true" json:"is_active"`

	// 组织隔离
	OwnerOrgID string `gorm:"type:varchar(50);index" json:"owner_org_id"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 设置表名
func (DocumentTemplate) TableName() string {
	return "document_templates"
}

// 预定义模板代码
const (
	TemplatePackingList  = "packing_list"
	TemplateInvoice      = "invoice"
	TemplateShippingMark = "shipping_mark"
)

// OCRResult OCR识别结果
type OCRResult struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	DocumentID uint `gorm:"index" json:"document_id"` // 关联 ShipmentDocument

	// 识别字段
	FieldName  string  `gorm:"type:varchar(50)" json:"field_name"`   // vessel_name, voyage_no, container_no...
	FieldValue string  `gorm:"type:varchar(500)" json:"field_value"` // 识别值
	Confidence float64 `gorm:"type:decimal(5,2)" json:"confidence"`  // 置信度 0-100

	// 位置信息
	BoundingBox string `gorm:"type:varchar(200)" json:"bounding_box"` // JSON: {"x":0,"y":0,"w":100,"h":20}

	// 是否已应用到运单
	Applied   bool       `gorm:"default:false" json:"applied"`
	AppliedAt *time.Time `json:"applied_at"`

	CreatedAt time.Time `json:"created_at"`

	// 关联
	Document *ShipmentDocument `gorm:"foreignKey:DocumentID;references:ID" json:"document,omitempty"`
}

// TableName 设置表名
func (OCRResult) TableName() string {
	return "ocr_results"
}

// OCR字段名称常量
const (
	OCRFieldVesselName   = "vessel_name"
	OCRFieldVoyageNo     = "voyage_no"
	OCRFieldContainerNo  = "container_no"
	OCRFieldBillOfLading = "bill_of_lading"
	OCRFieldSealNo       = "seal_no"
	OCRFieldWeight       = "weight"
	OCRFieldVolume       = "volume"
	OCRFieldPieces       = "pieces"
	OCRFieldHSCode       = "hs_code"
	OCRFieldDeclareValue = "declare_value"
	OCRFieldConsignee    = "consignee"
	OCRFieldShipper      = "shipper"
)

// OCRFieldNames 字段中文名
var OCRFieldNames = map[string]string{
	OCRFieldVesselName:   "船名",
	OCRFieldVoyageNo:     "航次",
	OCRFieldContainerNo:  "箱号",
	OCRFieldBillOfLading: "提单号",
	OCRFieldSealNo:       "封条号",
	OCRFieldWeight:       "重量",
	OCRFieldVolume:       "体积",
	OCRFieldPieces:       "件数",
	OCRFieldHSCode:       "HS编码",
	OCRFieldDeclareValue: "申报金额",
	OCRFieldConsignee:    "收货人",
	OCRFieldShipper:      "发货人",
}

// OCRResultResponse 响应结构
type OCRResultResponse struct {
	ID          uint    `json:"id"`
	DocumentID  uint    `json:"document_id"`
	FieldName   string  `json:"field_name"`
	FieldLabel  string  `json:"field_label"` // 中文标签
	FieldValue  string  `json:"field_value"`
	Confidence  float64 `json:"confidence"`
	Applied     bool    `json:"applied"`
	BoundingBox string  `json:"bounding_box"`
}

func (r *OCRResult) ToResponse() OCRResultResponse {
	return OCRResultResponse{
		ID:          r.ID,
		DocumentID:  r.DocumentID,
		FieldName:   r.FieldName,
		FieldLabel:  OCRFieldNames[r.FieldName],
		FieldValue:  r.FieldValue,
		Confidence:  r.Confidence,
		Applied:     r.Applied,
		BoundingBox: r.BoundingBox,
	}
}

// GeneratedDocument 生成的文档记录
type GeneratedDocument struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ShipmentID string `gorm:"type:varchar(50);index" json:"shipment_id"`
	TemplateID uint   `gorm:"index" json:"template_id"`

	// 文档信息
	DocType  string `gorm:"type:varchar(50)" json:"doc_type"` // packing_list, invoice
	FileName string `gorm:"type:varchar(200)" json:"file_name"`
	FilePath string `gorm:"type:varchar(500)" json:"file_path"`
	FileSize int64  `json:"file_size"`
	MimeType string `gorm:"type:varchar(100)" json:"mime_type"`
	Version  int    `gorm:"default:1" json:"version"` // 版本号

	// 生成信息
	GeneratedBy string    `gorm:"type:varchar(50)" json:"generated_by"`
	GeneratedAt time.Time `json:"generated_at"`

	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Shipment *Shipment         `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
	Template *DocumentTemplate `gorm:"foreignKey:TemplateID;references:ID" json:"template,omitempty"`
}

// TableName 设置表名
func (GeneratedDocument) TableName() string {
	return "generated_documents"
}
