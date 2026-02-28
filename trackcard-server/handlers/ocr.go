package handlers

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
)

// OCRHandler OCR处理器
type OCRHandler struct {
	db *gorm.DB
}

// NewOCRHandler 创建OCR处理器
func NewOCRHandler(db *gorm.DB) *OCRHandler {
	return &OCRHandler{db: db}
}

// Recognize 识别上传的文档
// POST /api/ocr/recognize
func (h *OCRHandler) Recognize(c *gin.Context) {
	// 获取文档类型
	docTypeStr := c.DefaultPostForm("doc_type", string(models.DocTypeOther))
	docType := models.DocumentType(docTypeStr)

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}
	defer file.Close()

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	// 获取OCR服务
	ocrService := services.GetOCRService()
	if ocrService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OCR服务未初始化"})
		return
	}

	// 根据文档类型调用不同的识别方法
	var fields []services.OCRField
	switch docType {
	case models.DocTypeBL:
		// 识别提单
		provider := ocrService.GetProvider()
		if provider != nil {
			fields, err = provider.RecognizeBL(fileData)
		}
	case models.DocTypeCustomsDec:
		// 识别报关单
		provider := ocrService.GetProvider()
		if provider != nil {
			fields, err = provider.RecognizeCustomsDeclaration(fileData)
		}
	default:
		// 通用识别
		provider := ocrService.GetProvider()
		if provider != nil {
			fields, err = provider.RecognizeGeneral(fileData)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "识别失败: " + err.Error()})
		return
	}

	// 计算平均置信度
	var avgConfidence float64
	if len(fields) > 0 {
		var total float64
		for _, f := range fields {
			total += f.Confidence
		}
		avgConfidence = total / float64(len(fields))
	}

	// 转换为响应格式
	results := make([]gin.H, 0, len(fields))
	for _, f := range fields {
		results = append(results, gin.H{
			"field_name":  f.FieldName,
			"field_label": models.OCRFieldNames[f.FieldName],
			"field_value": f.FieldValue,
			"confidence":  f.Confidence,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"file_name":  header.Filename,
		"doc_type":   docType,
		"confidence": avgConfidence,
		"fields":     results,
	})
}

// RecognizeAndSave 识别并保存到文档
// POST /api/shipments/:id/ocr
func (h *OCRHandler) RecognizeAndSave(c *gin.Context) {
	shipmentID := c.Param("id")

	// 获取文档类型
	docTypeStr := c.DefaultPostForm("doc_type", string(models.DocTypeOther))
	docType := models.DocumentType(docTypeStr)

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}
	defer file.Close()

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	// 保存文件
	fileService := services.GetFileService()
	if fileService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "文件服务未初始化"})
		return
	}

	// 使用 UploadReader 方法保存文件
	filePath, err := fileService.UploadReader(
		bytes.NewReader(fileData),
		header.Filename,
		services.FileCategoryDocuments,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	// 创建文档记录
	userID, _ := c.Get("user_id")
	doc := &models.ShipmentDocument{
		ShipmentID: shipmentID,
		DocType:    docType,
		DocName:    models.DocumentTypeNames[docType],
		FileName:   header.Filename,
		FileSize:   header.Size,
		FilePath:   filePath,
		MimeType:   header.Header.Get("Content-Type"),
		UploaderID: userID.(string),
	}

	if err := h.db.Create(doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文档记录失败"})
		return
	}

	// 调用OCR识别
	ocrService := services.GetOCRService()
	if ocrService != nil {
		go func() {
			// 异步识别
			_ = ocrService.RecognizeDocument(doc.ID, docType)
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"document_id": doc.ID,
		"file_name":   header.Filename,
		"message":     "文档已上传，正在进行OCR识别",
	})
}

// GetOCRResults 获取文档的OCR识别结果
// GET /api/documents/:id/ocr
func (h *OCRHandler) GetOCRResults(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
		return
	}

	var results []models.OCRResult
	if err := h.db.Where("document_id = ?", docID).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取识别结果失败"})
		return
	}

	// 转换为响应格式
	responses := make([]models.OCRResultResponse, 0, len(results))
	for _, r := range results {
		responses = append(responses, r.ToResponse())
	}

	c.JSON(http.StatusOK, responses)
}

// ConfirmOCRResults 确认OCR识别结果
// POST /api/documents/:id/ocr/confirm
func (h *OCRHandler) ConfirmOCRResults(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文档ID"})
		return
	}

	var req struct {
		Results []struct {
			FieldName  string `json:"field_name"`
			FieldValue string `json:"field_value"`
		} `json:"results"`
		ImportToShipment bool `json:"import_to_shipment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	userID, _ := c.Get("user_id")

	// 更新OCR结果
	for _, r := range req.Results {
		h.db.Model(&models.OCRResult{}).
			Where("document_id = ? AND field_name = ?", docID, r.FieldName).
			Updates(map[string]interface{}{
				"field_value":  r.FieldValue,
				"is_confirmed": true,
				"confirmed_by": userID,
			})
	}

	// 如果需要导入到运单
	if req.ImportToShipment {
		// 获取文档关联的运单
		var doc models.ShipmentDocument
		if err := h.db.First(&doc, docID).Error; err == nil {
			h.applyOCRToShipment(doc.ShipmentID, req.Results)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "识别结果已确认",
	})
}

// applyOCRToShipment 将OCR结果应用到运单
func (h *OCRHandler) applyOCRToShipment(shipmentID string, results []struct {
	FieldName  string `json:"field_name"`
	FieldValue string `json:"field_value"`
}) {
	updates := make(map[string]interface{})

	for _, r := range results {
		switch r.FieldName {
		case models.OCRFieldVesselName:
			updates["vessel_name"] = r.FieldValue
		case models.OCRFieldVoyageNo:
			updates["voyage"] = r.FieldValue
		case models.OCRFieldContainerNo:
			updates["container_no"] = r.FieldValue
		case models.OCRFieldBillOfLading:
			updates["bl_no"] = r.FieldValue
		case models.OCRFieldSealNo:
			updates["seal_no"] = r.FieldValue
		case models.OCRFieldWeight:
			if weight, err := strconv.ParseFloat(r.FieldValue, 64); err == nil {
				updates["weight"] = weight
			}
		case models.OCRFieldVolume:
			if volume, err := strconv.ParseFloat(r.FieldValue, 64); err == nil {
				updates["volume"] = volume
			}
		case models.OCRFieldPieces:
			if pieces, err := strconv.Atoi(r.FieldValue); err == nil {
				updates["package_count"] = pieces
			}
		}
	}

	if len(updates) > 0 {
		h.db.Model(&models.Shipment{}).Where("id = ?", shipmentID).Updates(updates)
	}
}

// GetSupportedDocTypes 获取支持的文档类型
// GET /api/ocr/doc-types
func (h *OCRHandler) GetSupportedDocTypes(c *gin.Context) {
	docTypes := []gin.H{
		{
			"code":        models.DocTypeBL,
			"name":        models.DocumentTypeNames[models.DocTypeBL],
			"description": "提单OCR识别，支持船名、航次、柜号、提单号等字段",
		},
		{
			"code":        models.DocTypeCustomsDec,
			"name":        models.DocumentTypeNames[models.DocTypeCustomsDec],
			"description": "报关单OCR识别，支持HS编码、申报金额、件数、毛重等字段",
		},
		{
			"code":        models.DocTypeInvoice,
			"name":        models.DocumentTypeNames[models.DocTypeInvoice],
			"description": "商业发票OCR识别，支持发票号、金额、买卖双方等字段",
		},
		{
			"code":        models.DocTypePackingList,
			"name":        models.DocumentTypeNames[models.DocTypePackingList],
			"description": "装箱单OCR识别，支持品名、数量、重量、体积等字段",
		},
		{
			"code":        models.DocTypeOther,
			"name":        "通用文档",
			"description": "通用文字识别，自动提取文档中的文字内容",
		},
	}
	c.JSON(http.StatusOK, docTypes)
}

// GetOCRProviderInfo 获取OCR服务提供商信息
// GET /api/ocr/provider
func (h *OCRHandler) GetOCRProviderInfo(c *gin.Context) {
	ocrService := services.GetOCRService()
	if ocrService == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled":  false,
			"provider": nil,
		})
		return
	}

	provider := ocrService.GetProvider()
	if provider == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled":  false,
			"provider": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled":  true,
		"provider": provider.GetProviderName(),
	})
}

// fileExtension 获取文件扩展名
func fileExtension(filename string) string {
	return filepath.Ext(filename)
}
