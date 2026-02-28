package models

import (
	"time"

	"gorm.io/gorm"
)

// DocumentType 单据类型
type DocumentType string

const (
	DocTypeBL          DocumentType = "bl"           // 提单
	DocTypeInvoice     DocumentType = "invoice"      // 商业发票
	DocTypePackingList DocumentType = "packing_list" // 装箱单
	DocTypeCustomsDec  DocumentType = "customs_dec"  // 报关单
	DocTypeCO          DocumentType = "co"           // 产地证
	DocTypeInsurance   DocumentType = "insurance"    // 保险单
	DocTypeContract    DocumentType = "contract"     // 合同
	DocTypeOther       DocumentType = "other"        // 其他
)

// DocumentTypeNames 单据类型中文名
var DocumentTypeNames = map[DocumentType]string{
	DocTypeBL:          "提单",
	DocTypeInvoice:     "商业发票",
	DocTypePackingList: "装箱单",
	DocTypeCustomsDec:  "报关单",
	DocTypeCO:          "产地证",
	DocTypeInsurance:   "保险单",
	DocTypeContract:    "合同",
	DocTypeOther:       "其他",
}

// DocumentStatus 单据状态
type DocumentStatus string

const (
	DocStatusPending  DocumentStatus = "pending"  // 待审核
	DocStatusApproved DocumentStatus = "approved" // 已通过
	DocStatusRejected DocumentStatus = "rejected" // 已驳回
)

// ShipmentDocument 运单单据
type ShipmentDocument struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	ShipmentID string  `gorm:"type:varchar(50);index;not null" json:"shipment_id"`
	PartnerID  *string `gorm:"type:varchar(50);index" json:"partner_id"` // 上传者(空=内部用户)
	UploaderID string  `gorm:"type:varchar(50)" json:"uploader_id"`      // 上传用户ID

	// 文档信息
	DocType  DocumentType `gorm:"type:varchar(50)" json:"doc_type"`
	DocName  string       `gorm:"type:varchar(100)" json:"doc_name"` // 中文名称
	FileName string       `gorm:"type:varchar(200)" json:"file_name"`
	FileSize int64        `json:"file_size"`
	FilePath string       `gorm:"type:varchar(500)" json:"file_path"` // 存储路径
	MimeType string       `gorm:"type:varchar(100)" json:"mime_type"`

	// 审核状态
	Status     DocumentStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ReviewerID *string        `gorm:"type:varchar(50)" json:"reviewer_id"`
	ReviewedAt *time.Time     `json:"reviewed_at"`
	Remarks    string         `gorm:"type:varchar(500)" json:"remarks"`

	UploadedAt time.Time      `json:"uploaded_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Shipment *Shipment `gorm:"foreignKey:ShipmentID;references:ID" json:"shipment,omitempty"`
	Partner  *Partner  `gorm:"foreignKey:PartnerID;references:ID" json:"partner,omitempty"`
}

// TableName 设置表名
func (ShipmentDocument) TableName() string {
	return "shipment_documents"
}

// DocumentResponse 响应结构
type DocumentResponse struct {
	ID           uint           `json:"id"`
	ShipmentID   string         `json:"shipment_id"`
	PartnerID    *string        `json:"partner_id"`
	PartnerName  string         `json:"partner_name,omitempty"`
	UploaderID   string         `json:"uploader_id"`
	UploaderName string         `json:"uploader_name,omitempty"`
	DocType      DocumentType   `json:"doc_type"`
	DocTypeName  string         `json:"doc_type_name"`
	DocName      string         `json:"doc_name"`
	FileName     string         `json:"file_name"`
	FileSize     int64          `json:"file_size"`
	MimeType     string         `json:"mime_type"`
	Status       DocumentStatus `json:"status"`
	ReviewerID   *string        `json:"reviewer_id"`
	ReviewedAt   *time.Time     `json:"reviewed_at"`
	Remarks      string         `json:"remarks"`
	UploadedAt   time.Time      `json:"uploaded_at"`
	DownloadURL  string         `json:"download_url,omitempty"` // 前端显示用
}

func (d *ShipmentDocument) ToResponse() DocumentResponse {
	resp := DocumentResponse{
		ID:          d.ID,
		ShipmentID:  d.ShipmentID,
		PartnerID:   d.PartnerID,
		UploaderID:  d.UploaderID,
		DocType:     d.DocType,
		DocTypeName: DocumentTypeNames[d.DocType],
		DocName:     d.DocName,
		FileName:    d.FileName,
		FileSize:    d.FileSize,
		MimeType:    d.MimeType,
		Status:      d.Status,
		ReviewerID:  d.ReviewerID,
		ReviewedAt:  d.ReviewedAt,
		Remarks:     d.Remarks,
		UploadedAt:  d.UploadedAt,
	}
	if d.Partner != nil {
		resp.PartnerName = d.Partner.Name
	}
	return resp
}

// DocumentUploadRequest 上传请求 (表单方式)
type DocumentUploadRequest struct {
	DocType DocumentType `form:"doc_type" binding:"required"`
	DocName string       `form:"doc_name"`
	Remarks string       `form:"remarks"`
}

// DocumentReviewRequest 审核请求
type DocumentReviewRequest struct {
	Status  DocumentStatus `json:"status" binding:"required"` // approved/rejected
	Remarks string         `json:"remarks"`
}
