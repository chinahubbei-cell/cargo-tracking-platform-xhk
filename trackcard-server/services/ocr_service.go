package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"trackcard-server/models"
)

// OCRProvider OCR服务提供商接口
type OCRProvider interface {
	// RecognizeBL 识别提单
	RecognizeBL(imageData []byte) ([]OCRField, error)
	// RecognizeCustomsDeclaration 识别报关单
	RecognizeCustomsDeclaration(imageData []byte) ([]OCRField, error)
	// RecognizeGeneral 通用文字识别
	RecognizeGeneral(imageData []byte) ([]OCRField, error)
	// GetProviderName 获取提供商名称
	GetProviderName() string
}

// OCRField OCR识别字段
type OCRField struct {
	FieldName   string  `json:"field_name"`
	FieldValue  string  `json:"field_value"`
	Confidence  float64 `json:"confidence"`
	BoundingBox string  `json:"bounding_box,omitempty"`
}

// OCRService OCR服务
type OCRService struct {
	db          *gorm.DB
	provider    OCRProvider
	fileService FileService
}

// NewOCRService 创建OCR服务
func NewOCRService(db *gorm.DB, provider OCRProvider, fileService FileService) *OCRService {
	return &OCRService{
		db:          db,
		provider:    provider,
		fileService: fileService,
	}
}

// GetProvider 获取OCR提供商
func (s *OCRService) GetProvider() OCRProvider {
	return s.provider
}

// RecognizeDocument 识别文档
func (s *OCRService) RecognizeDocument(documentID uint, docType models.DocumentType) error {
	// 获取文档
	var doc models.ShipmentDocument
	if err := s.db.First(&doc, documentID).Error; err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	// 读取文件
	filePath := s.fileService.GetFilePath(doc.FilePath)
	imageData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 根据文档类型选择识别方法
	var fields []OCRField
	switch docType {
	case models.DocTypeBL:
		fields, err = s.provider.RecognizeBL(imageData)
	case models.DocTypeCustomsDec:
		fields, err = s.provider.RecognizeCustomsDeclaration(imageData)
	default:
		fields, err = s.provider.RecognizeGeneral(imageData)
	}

	if err != nil {
		return fmt.Errorf("OCR recognition failed: %w", err)
	}

	// 保存识别结果
	for _, field := range fields {
		result := models.OCRResult{
			DocumentID:  documentID,
			FieldName:   field.FieldName,
			FieldValue:  field.FieldValue,
			Confidence:  field.Confidence,
			BoundingBox: field.BoundingBox,
		}
		s.db.Create(&result)
	}

	return nil
}

// ===== 腾讯云 OCR 实现 =====

// TencentOCR 腾讯云OCR提供商
type TencentOCR struct {
	SecretID  string
	SecretKey string
	Region    string
}

// NewTencentOCR 创建腾讯云OCR
func NewTencentOCR(secretID, secretKey, region string) *TencentOCR {
	return &TencentOCR{
		SecretID:  secretID,
		SecretKey: secretKey,
		Region:    region,
	}
}

func (t *TencentOCR) GetProviderName() string {
	return "tencent"
}

// RecognizeBL 识别提单 (使用通用印刷体识别)
func (t *TencentOCR) RecognizeBL(imageData []byte) ([]OCRField, error) {
	// 调用腾讯云通用印刷体识别API
	result, err := t.callTencentOCR("GeneralBasicOCR", imageData)
	if err != nil {
		return nil, err
	}

	// 解析提单关键字段
	return t.parseBLFields(result), nil
}

// RecognizeCustomsDeclaration 识别报关单
func (t *TencentOCR) RecognizeCustomsDeclaration(imageData []byte) ([]OCRField, error) {
	result, err := t.callTencentOCR("GeneralBasicOCR", imageData)
	if err != nil {
		return nil, err
	}

	return t.parseCustomsFields(result), nil
}

// RecognizeGeneral 通用识别
func (t *TencentOCR) RecognizeGeneral(imageData []byte) ([]OCRField, error) {
	result, err := t.callTencentOCR("GeneralBasicOCR", imageData)
	if err != nil {
		return nil, err
	}

	return t.parseGeneralFields(result), nil
}

// callTencentOCR 调用腾讯云OCR API
func (t *TencentOCR) callTencentOCR(action string, imageData []byte) (map[string]interface{}, error) {
	host := "ocr.tencentcloudapi.com"
	algorithm := "TC3-HMAC-SHA256"
	service := "ocr"
	version := "2018-11-19"
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	// 构建请求体
	payload := map[string]string{
		"ImageBase64": base64.StdEncoding.EncodeToString(imageData),
	}
	payloadBytes, _ := json.Marshal(payload)

	// 构建规范请求串
	httpRequestMethod := "POST"
	canonicalUri := "/"
	canonicalQueryString := ""
	contentType := "application/json; charset=utf-8"
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		contentType, host, action)
	signedHeaders := "content-type;host;x-tc-action"

	hashedPayload := sha256Hex(payloadBytes)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod, canonicalUri, canonicalQueryString,
		canonicalHeaders, signedHeaders, hashedPayload)

	// 构建待签名字符串
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256Hex([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm, timestamp, credentialScope, hashedCanonicalRequest)

	// 计算签名
	secretDate := hmacSHA256([]byte("TC3"+t.SecretKey), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	// 构建Authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, t.SecretID, credentialScope, signedHeaders, signature)

	// 发送请求
	req, _ := http.NewRequest("POST", "https://"+host, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Region", t.Region)
	req.Header.Set("Authorization", authorization)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	return result, nil
}

// parseBLFields 解析提单字段
func (t *TencentOCR) parseBLFields(result map[string]interface{}) []OCRField {
	var fields []OCRField

	// 从OCR结果中提取文本
	response, ok := result["Response"].(map[string]interface{})
	if !ok {
		return fields
	}

	textDetections, ok := response["TextDetections"].([]interface{})
	if !ok {
		return fields
	}

	// 关键词匹配
	keywords := map[string]string{
		"B/L":       models.OCRFieldBillOfLading,
		"BILL OF":   models.OCRFieldBillOfLading,
		"VESSEL":    models.OCRFieldVesselName,
		"VOYAGE":    models.OCRFieldVoyageNo,
		"CONTAINER": models.OCRFieldContainerNo,
		"SEAL":      models.OCRFieldSealNo,
		"WEIGHT":    models.OCRFieldWeight,
		"CONSIGNEE": models.OCRFieldConsignee,
		"SHIPPER":   models.OCRFieldShipper,
	}

	for i, detection := range textDetections {
		det, ok := detection.(map[string]interface{})
		if !ok {
			continue
		}

		text, _ := det["DetectedText"].(string)
		confidence, _ := det["Confidence"].(float64)

		// 检查是否包含关键词
		for keyword, fieldName := range keywords {
			if containsIgnoreCase(text, keyword) && i+1 < len(textDetections) {
				// 获取下一行作为值
				nextDet, ok := textDetections[i+1].(map[string]interface{})
				if ok {
					value, _ := nextDet["DetectedText"].(string)
					fields = append(fields, OCRField{
						FieldName:  fieldName,
						FieldValue: value,
						Confidence: confidence,
					})
				}
			}
		}
	}

	return fields
}

// parseCustomsFields 解析报关单字段
func (t *TencentOCR) parseCustomsFields(result map[string]interface{}) []OCRField {
	var fields []OCRField

	response, ok := result["Response"].(map[string]interface{})
	if !ok {
		return fields
	}

	textDetections, ok := response["TextDetections"].([]interface{})
	if !ok {
		return fields
	}

	keywords := map[string]string{
		"HS": models.OCRFieldHSCode,
		"申报": models.OCRFieldDeclareValue,
		"货物": "goods_name",
		"数量": models.OCRFieldPieces,
		"毛重": models.OCRFieldWeight,
	}

	for i, detection := range textDetections {
		det, ok := detection.(map[string]interface{})
		if !ok {
			continue
		}

		text, _ := det["DetectedText"].(string)
		confidence, _ := det["Confidence"].(float64)

		for keyword, fieldName := range keywords {
			if containsIgnoreCase(text, keyword) && i+1 < len(textDetections) {
				nextDet, ok := textDetections[i+1].(map[string]interface{})
				if ok {
					value, _ := nextDet["DetectedText"].(string)
					fields = append(fields, OCRField{
						FieldName:  fieldName,
						FieldValue: value,
						Confidence: confidence,
					})
				}
			}
		}
	}

	return fields
}

// parseGeneralFields 通用解析
func (t *TencentOCR) parseGeneralFields(result map[string]interface{}) []OCRField {
	var fields []OCRField

	response, ok := result["Response"].(map[string]interface{})
	if !ok {
		return fields
	}

	textDetections, ok := response["TextDetections"].([]interface{})
	if !ok {
		return fields
	}

	for i, detection := range textDetections {
		det, ok := detection.(map[string]interface{})
		if !ok {
			continue
		}

		text, _ := det["DetectedText"].(string)
		confidence, _ := det["Confidence"].(float64)

		fields = append(fields, OCRField{
			FieldName:  fmt.Sprintf("line_%d", i),
			FieldValue: text,
			Confidence: confidence,
		})
	}

	return fields
}

// ===== 阿里云 OCR 实现 =====

// AliyunOCR 阿里云OCR提供商
type AliyunOCR struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
}

// NewAliyunOCR 创建阿里云OCR
func NewAliyunOCR(accessKeyID, accessKeySecret string) *AliyunOCR {
	return &AliyunOCR{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		Endpoint:        "https://ocr-api.cn-hangzhou.aliyuncs.com",
	}
}

func (a *AliyunOCR) GetProviderName() string {
	return "aliyun"
}

// RecognizeBL 识别提单
func (a *AliyunOCR) RecognizeBL(imageData []byte) ([]OCRField, error) {
	// 调用阿里云通用文字识别
	result, err := a.callAliyunOCR("/api/v1/ocr/general", imageData)
	if err != nil {
		return nil, err
	}

	return a.parseBLFields(result), nil
}

// RecognizeCustomsDeclaration 识别报关单
func (a *AliyunOCR) RecognizeCustomsDeclaration(imageData []byte) ([]OCRField, error) {
	result, err := a.callAliyunOCR("/api/v1/ocr/general", imageData)
	if err != nil {
		return nil, err
	}

	return a.parseCustomsFields(result), nil
}

// RecognizeGeneral 通用识别
func (a *AliyunOCR) RecognizeGeneral(imageData []byte) ([]OCRField, error) {
	result, err := a.callAliyunOCR("/api/v1/ocr/general", imageData)
	if err != nil {
		return nil, err
	}

	return a.parseGeneralFields(result), nil
}

// callAliyunOCR 调用阿里云OCR API
func (a *AliyunOCR) callAliyunOCR(path string, imageData []byte) (map[string]interface{}, error) {
	payload := map[string]string{
		"image": base64.StdEncoding.EncodeToString(imageData),
	}
	payloadBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", a.Endpoint+path, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "APPCODE "+a.AccessKeySecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	return result, nil
}

func (a *AliyunOCR) parseBLFields(result map[string]interface{}) []OCRField {
	// 类似腾讯云的解析逻辑
	var fields []OCRField
	// ... 解析逻辑
	return fields
}

func (a *AliyunOCR) parseCustomsFields(result map[string]interface{}) []OCRField {
	var fields []OCRField
	return fields
}

func (a *AliyunOCR) parseGeneralFields(result map[string]interface{}) []OCRField {
	var fields []OCRField
	return fields
}

// ===== 辅助函数 =====

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToUpper(s), strings.ToUpper(substr))
}

// ===== 全局OCR服务 =====

var ocrService *OCRService

// InitOCRService 初始化OCR服务
func InitOCRService(db *gorm.DB, fileService FileService, providerType string) {
	var provider OCRProvider

	switch providerType {
	case "tencent":
		provider = NewTencentOCR(
			os.Getenv("TENCENT_SECRET_ID"),
			os.Getenv("TENCENT_SECRET_KEY"),
			os.Getenv("TENCENT_REGION"),
		)
	case "aliyun":
		provider = NewAliyunOCR(
			os.Getenv("ALIYUN_ACCESS_KEY_ID"),
			os.Getenv("ALIYUN_ACCESS_KEY_SECRET"),
		)
	default:
		// 默认使用Mock或不初始化
		return
	}

	ocrService = NewOCRService(db, provider, fileService)
}

// GetOCRService 获取OCR服务
func GetOCRService() *OCRService {
	return ocrService
}
