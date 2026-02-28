package handlers

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
	"trackcard-server/utils"
)

type DocumentHandler struct {
	db          *gorm.DB
	fileService services.FileService
}

func NewDocumentHandler(db *gorm.DB, fileService services.FileService) *DocumentHandler {
	return &DocumentHandler{
		db:          db,
		fileService: fileService,
	}
}

// UploadDocument 上传运单单据
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	// 验证运单存在
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	// 获取上传文件
	file, err := c.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "请选择要上传的文件")
		return
	}

	// 安全检查: 文件大小限制 (50MB)
	const maxFileSize = 50 * 1024 * 1024
	if file.Size > maxFileSize {
		utils.BadRequest(c, "文件大小超过50MB限制")
		return
	}

	// 安全检查: MIME类型白名单
	allowedMimeTypes := map[string]bool{
		"application/pdf":    true,
		"image/jpeg":         true,
		"image/png":          true,
		"image/gif":          true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	}
	mimeType := file.Header.Get("Content-Type")
	if !allowedMimeTypes[mimeType] {
		utils.BadRequest(c, "不支持的文件类型，仅支持PDF/图片/Office文档")
		return
	}

	// 获取表单参数
	docType := c.PostForm("doc_type")
	docName := c.PostForm("doc_name")
	remarks := c.PostForm("remarks")

	if docType == "" {
		utils.BadRequest(c, "请指定单据类型")
		return
	}

	// 上传文件
	storedPath, err := h.fileService.Upload(file, services.FileCategoryDocuments)
	if err != nil {
		utils.InternalError(c, "文件上传失败: "+err.Error())
		return
	}

	// 获取用户信息
	userID := ""
	if id, exists := c.Get("user_id"); exists {
		userID = id.(string)
	}

	// 获取合作伙伴ID (如果是外部用户)
	var partnerID *string
	if pid, exists := c.Get("partner_id"); exists {
		pidStr := pid.(string)
		partnerID = &pidStr
	}

	// 自动设置单据名称
	if docName == "" {
		docName = models.DocumentTypeNames[models.DocumentType(docType)]
	}

	// 创建文档记录
	doc := models.ShipmentDocument{
		ShipmentID: shipmentID,
		PartnerID:  partnerID,
		UploaderID: userID,
		DocType:    models.DocumentType(docType),
		DocName:    docName,
		FileName:   file.Filename,
		FileSize:   file.Size,
		FilePath:   storedPath,
		MimeType:   file.Header.Get("Content-Type"),
		Status:     models.DocStatusPending,
		Remarks:    remarks,
		UploadedAt: time.Now(),
	}

	if err := h.db.Create(&doc).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.CreatedResponse(c, doc.ToResponse())
}

// GetShipmentDocuments 获取运单的所有单据
func (h *DocumentHandler) GetShipmentDocuments(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	var docs []models.ShipmentDocument
	if err := h.db.Preload("Partner").
		Where("shipment_id = ?", shipmentID).
		Order("uploaded_at DESC").
		Find(&docs).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	var responses []models.DocumentResponse
	for _, d := range docs {
		resp := d.ToResponse()
		resp.DownloadURL = h.fileService.GetURL(d.FilePath)
		responses = append(responses, resp)
	}

	utils.SuccessResponse(c, responses)
}

// DownloadDocument 下载单据
func (h *DocumentHandler) DownloadDocument(c *gin.Context) {
	id := c.Param("id")

	var doc models.ShipmentDocument
	if err := h.db.First(&doc, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "单据不存在")
		return
	}

	filePath := h.fileService.GetFilePath(doc.FilePath)
	c.FileAttachment(filePath, doc.FileName)
}

// ReviewDocument 审核单据
func (h *DocumentHandler) ReviewDocument(c *gin.Context) {
	id := c.Param("id")

	var doc models.ShipmentDocument
	if err := h.db.First(&doc, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "单据不存在")
		return
	}

	var req models.DocumentReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 获取审核人ID
	reviewerID := ""
	if id, exists := c.Get("user_id"); exists {
		reviewerID = id.(string)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      req.Status,
		"remarks":     req.Remarks,
		"reviewer_id": reviewerID,
		"reviewed_at": now,
	}

	if err := h.db.Model(&doc).Updates(updates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	h.db.First(&doc, id)
	utils.SuccessResponse(c, doc.ToResponse())
}

// DeleteDocument 删除单据
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	id := c.Param("id")

	var doc models.ShipmentDocument
	if err := h.db.First(&doc, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "单据不存在")
		return
	}

	// 删除文件
	if err := h.fileService.Delete(doc.FilePath); err != nil {
		// 记录日志但不影响删除记录
		fmt.Printf("Failed to delete file: %v\n", err)
	}

	if err := h.db.Delete(&doc).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// GetGeneratedDocuments 获取生成的文档列表
func (h *DocumentHandler) GetGeneratedDocuments(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	var docs []models.GeneratedDocument
	if err := h.db.Preload("Template").
		Where("shipment_id = ?", shipmentID).
		Order("generated_at DESC").
		Find(&docs).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 添加下载URL
	type GeneratedDocResponse struct {
		models.GeneratedDocument
		DownloadURL string `json:"download_url"`
	}

	var responses []GeneratedDocResponse
	for _, d := range docs {
		responses = append(responses, GeneratedDocResponse{
			GeneratedDocument: d,
			DownloadURL:       h.fileService.GetURL(d.FilePath),
		})
	}

	utils.SuccessResponse(c, responses)
}

// ListTemplates 获取文档模板列表
func (h *DocumentHandler) ListTemplates(c *gin.Context) {
	var templates []models.DocumentTemplate
	if err := h.db.Where("is_active = ?", true).Find(&templates).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.SuccessResponse(c, templates)
}

// GetOCRResults 获取单据的OCR识别结果
func (h *DocumentHandler) GetOCRResults(c *gin.Context) {
	docID := c.Param("id")

	var results []models.OCRResult
	if err := h.db.Where("document_id = ?", docID).
		Order("confidence DESC").
		Find(&results).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	var responses []models.OCRResultResponse
	for _, r := range results {
		responses = append(responses, r.ToResponse())
	}

	utils.SuccessResponse(c, responses)
}

// ApplyOCRResult 将OCR结果应用到运单
func (h *DocumentHandler) ApplyOCRResult(c *gin.Context) {
	docID := c.Param("id")

	// 获取单据
	var doc models.ShipmentDocument
	if err := h.db.First(&doc, "id = ?", docID).Error; err != nil {
		utils.NotFound(c, "单据不存在")
		return
	}

	// 获取运单
	var shipment models.Shipment
	if err := h.db.First(&shipment, "id = ?", doc.ShipmentID).Error; err != nil {
		utils.NotFound(c, "运单不存在")
		return
	}

	// 获取所有未应用的OCR结果
	var results []models.OCRResult
	if err := h.db.Where("document_id = ? AND applied = ?", docID, false).Find(&results).Error; err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	// 应用到运单
	updates := make(map[string]interface{})
	appliedCount := 0

	for _, r := range results {
		switch r.FieldName {
		case models.OCRFieldVesselName:
			updates["vessel_name"] = r.FieldValue
		case models.OCRFieldVoyageNo:
			updates["voyage_no"] = r.FieldValue
		case models.OCRFieldContainerNo:
			updates["container_no"] = r.FieldValue
		case models.OCRFieldBillOfLading:
			updates["bill_of_lading"] = r.FieldValue
		case models.OCRFieldSealNo:
			updates["seal_no"] = r.FieldValue
		}

		// 标记为已应用
		now := time.Now()
		h.db.Model(&r).Updates(map[string]interface{}{
			"applied":    true,
			"applied_at": now,
		})
		appliedCount++
	}

	if len(updates) > 0 {
		if err := h.db.Model(&shipment).Updates(updates).Error; err != nil {
			utils.InternalError(c, err.Error())
			return
		}
	}

	utils.SuccessResponse(c, gin.H{
		"message":        "OCR结果应用成功",
		"applied_count":  appliedCount,
		"updated_fields": updates,
	})
}

// GetMimeType 根据文件扩展名获取MIME类型
func GetMimeType(filename string) string {
	ext := filepath.Ext(filename)
	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// GeneratePackingList 生成装箱单PDF
func (h *DocumentHandler) GeneratePackingList(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	// 获取用户ID
	userID := ""
	if id, exists := c.Get("user_id"); exists {
		userID = id.(string)
	}

	// 生成PDF
	pdfGen := services.GetPDFGenerator()
	if pdfGen == nil {
		utils.InternalError(c, "PDF生成服务未初始化")
		return
	}

	doc, err := pdfGen.GeneratePackingList(shipmentID, userID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message":      "装箱单生成成功",
		"document":     doc,
		"download_url": h.fileService.GetURL(doc.FilePath),
	})
}

// GenerateInvoice 生成商业发票PDF
func (h *DocumentHandler) GenerateInvoice(c *gin.Context) {
	shipmentID := c.Param("shipment_id")

	// 获取用户ID
	userID := ""
	if id, exists := c.Get("user_id"); exists {
		userID = id.(string)
	}

	// 生成PDF
	pdfGen := services.GetPDFGenerator()
	if pdfGen == nil {
		utils.InternalError(c, "PDF生成服务未初始化")
		return
	}

	doc, err := pdfGen.GenerateInvoice(shipmentID, userID)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.CreatedResponse(c, gin.H{
		"message":      "商业发票生成成功",
		"document":     doc,
		"download_url": h.fileService.GetURL(doc.FilePath),
	})
}

// DownloadGeneratedDocument 下载生成的文档
func (h *DocumentHandler) DownloadGeneratedDocument(c *gin.Context) {
	id := c.Param("id")

	var doc models.GeneratedDocument
	if err := h.db.First(&doc, "id = ?", id).Error; err != nil {
		utils.NotFound(c, "文档不存在")
		return
	}

	filePath := h.fileService.GetFilePath(doc.FilePath)
	c.FileAttachment(filePath, doc.FileName)
}

// TriggerOCR 触发OCR识别
func (h *DocumentHandler) TriggerOCR(c *gin.Context) {
	docID := c.Param("id")

	var doc models.ShipmentDocument
	if err := h.db.First(&doc, "id = ?", docID).Error; err != nil {
		utils.NotFound(c, "单据不存在")
		return
	}

	// 获取OCR服务
	ocrSvc := services.GetOCRService()
	if ocrSvc == nil {
		utils.BadRequest(c, "OCR服务未配置,请在环境变量中配置腾讯云或阿里云OCR密钥")
		return
	}

	// 触发识别
	if err := ocrSvc.RecognizeDocument(doc.ID, doc.DocType); err != nil {
		utils.InternalError(c, "OCR识别失败: "+err.Error())
		return
	}

	// 获取识别结果
	var results []models.OCRResult
	h.db.Where("document_id = ?", docID).Find(&results)

	var responses []models.OCRResultResponse
	for _, r := range results {
		responses = append(responses, r.ToResponse())
	}

	utils.SuccessResponse(c, gin.H{
		"message": "OCR识别完成",
		"results": responses,
	})
}
