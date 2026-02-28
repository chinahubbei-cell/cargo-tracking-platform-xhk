package services

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"

	"trackcard-server/models"
)

// PDFGenerator PDF文档生成器
type PDFGenerator struct {
	db          *gorm.DB
	fileService FileService
}

// NewPDFGenerator 创建PDF生成器
func NewPDFGenerator(db *gorm.DB, fileService FileService) *PDFGenerator {
	return &PDFGenerator{
		db:          db,
		fileService: fileService,
	}
}

// GeneratePackingList 生成装箱单PDF
func (g *PDFGenerator) GeneratePackingList(shipmentID, userID string) (*models.GeneratedDocument, error) {
	// 获取运单信息
	var shipment models.Shipment
	if err := g.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return nil, fmt.Errorf("shipment not found: %w", err)
	}

	// 创建PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// 设置字体 (使用内置字体，中文需要额外处理)
	pdf.SetFont("Helvetica", "B", 18)

	// 标题
	pdf.CellFormat(180, 12, "PACKING LIST", "0", 1, "C", false, 0, "")
	pdf.Ln(8)

	// 基本信息区
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(90, 7, "Shipper:", "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 7, "Consignee:", "1", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(90, 15, shipment.Origin, "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 15, shipment.Destination, "1", 1, "L", false, 0, "")

	pdf.Ln(5)

	// 运单信息
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "B/L No:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, shipment.BillOfLading, "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "Container No:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, shipment.ContainerNo, "1", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "Vessel:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, shipment.VesselName, "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "Voyage:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, shipment.VoyageNo, "1", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "Seal No:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, shipment.SealNo, "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "ETD:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	etdStr := ""
	if shipment.ETD != nil {
		etdStr = shipment.ETD.Format("2006-01-02")
	}
	pdf.CellFormat(45, 7, etdStr, "1", 1, "L", false, 0, "")

	pdf.Ln(8)

	// 货物信息表头
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(80, 8, "Description of Goods", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Quantity", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Weight (KG)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Volume (CBM)", "1", 1, "C", true, 0, "")

	// 货物信息
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(80, 10, shipment.CargoName, "1", 0, "L", false, 0, "")

	pieces := ""
	if shipment.Pieces != nil {
		pieces = fmt.Sprintf("%d", *shipment.Pieces)
	}
	pdf.CellFormat(35, 10, pieces, "1", 0, "C", false, 0, "")

	weight := ""
	if shipment.Weight != nil {
		weight = fmt.Sprintf("%.2f", *shipment.Weight)
	}
	pdf.CellFormat(35, 10, weight, "1", 0, "C", false, 0, "")

	volume := ""
	if shipment.Volume != nil {
		volume = fmt.Sprintf("%.2f", *shipment.Volume)
	}
	pdf.CellFormat(30, 10, volume, "1", 1, "C", false, 0, "")

	// PO信息
	if shipment.PONumbers != "" {
		pdf.Ln(5)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(45, 7, "PO Number(s):", "0", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(135, 7, shipment.PONumbers, "0", 1, "L", false, 0, "")
	}

	// 底部签名区
	pdf.Ln(20)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(90, 7, "Prepared by: ______________________", "0", 0, "L", false, 0, "")
	pdf.CellFormat(90, 7, fmt.Sprintf("Date: %s", time.Now().Format("2006-01-02")), "0", 1, "L", false, 0, "")

	pdf.Ln(10)
	pdf.CellFormat(90, 7, "Signature: ______________________", "0", 0, "L", false, 0, "")

	// 输出到buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	// 保存文件
	filename := fmt.Sprintf("PackingList_%s_%s.pdf", shipment.ID, time.Now().Format("20060102"))
	storedPath, err := g.fileService.UploadReader(&buf, filename, FileCategoryGenerated)
	if err != nil {
		return nil, fmt.Errorf("failed to save PDF: %w", err)
	}

	// 创建记录
	genDoc := &models.GeneratedDocument{
		ShipmentID:  shipmentID,
		DocType:     models.TemplatePackingList,
		FileName:    filename,
		FilePath:    storedPath,
		FileSize:    int64(buf.Len()),
		MimeType:    "application/pdf",
		GeneratedBy: userID,
		GeneratedAt: time.Now(),
	}

	if err := g.db.Create(genDoc).Error; err != nil {
		return nil, fmt.Errorf("failed to save record: %w", err)
	}

	return genDoc, nil
}

// GenerateInvoice 生成商业发票PDF
func (g *PDFGenerator) GenerateInvoice(shipmentID, userID string) (*models.GeneratedDocument, error) {
	// 获取运单信息
	var shipment models.Shipment
	if err := g.db.First(&shipment, "id = ?", shipmentID).Error; err != nil {
		return nil, fmt.Errorf("shipment not found: %w", err)
	}

	// 创建PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// 标题
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(180, 12, "COMMERCIAL INVOICE", "0", 1, "C", false, 0, "")
	pdf.Ln(8)

	// 发票编号和日期
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "Invoice No:", "0", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	invoiceNo := fmt.Sprintf("INV-%s-%s", shipment.ID, time.Now().Format("20060102"))
	pdf.CellFormat(45, 7, invoiceNo, "0", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "Date:", "0", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, time.Now().Format("2006-01-02"), "0", 1, "L", false, 0, "")

	pdf.Ln(5)

	// 发货人/收货人
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(90, 7, "Seller/Exporter:", "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 7, "Buyer/Importer:", "1", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(90, 20, shipment.Origin, "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 20, shipment.Destination, "1", 1, "L", false, 0, "")

	pdf.Ln(5)

	// 运输信息
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "B/L No:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, shipment.BillOfLading, "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(45, 7, "Container No:", "1", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(45, 7, shipment.ContainerNo, "1", 1, "L", false, 0, "")

	pdf.Ln(5)

	// 货物明细表
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(60, 8, "Description", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Quantity", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Unit Price", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Amount (USD)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "HS Code", "1", 1, "C", true, 0, "")

	// 货物行
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(60, 10, shipment.CargoName, "1", 0, "L", false, 0, "")

	pieces := ""
	if shipment.Pieces != nil {
		pieces = fmt.Sprintf("%d", *shipment.Pieces)
	}
	pdf.CellFormat(25, 10, pieces, "1", 0, "C", false, 0, "")

	// 单价 (假设总费用/件数)
	unitPrice := ""
	amount := ""
	if shipment.TotalCost != nil {
		amount = fmt.Sprintf("%.2f", *shipment.TotalCost)
		if shipment.Pieces != nil && *shipment.Pieces > 0 {
			unitPrice = fmt.Sprintf("%.2f", *shipment.TotalCost/float64(*shipment.Pieces))
		}
	}
	pdf.CellFormat(30, 10, unitPrice, "1", 0, "C", false, 0, "")
	pdf.CellFormat(35, 10, amount, "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 10, "", "1", 1, "C", false, 0, "") // HS Code

	// 合计
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(115, 8, "TOTAL", "1", 0, "R", false, 0, "")
	pdf.CellFormat(35, 8, amount, "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 8, "USD", "1", 1, "C", false, 0, "")

	// 条款
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(180, 6, "Terms of Sale: FOB", "0", 1, "L", false, 0, "")
	pdf.CellFormat(180, 6, "Country of Origin: China", "0", 1, "L", false, 0, "")

	// 签名
	pdf.Ln(15)
	pdf.CellFormat(90, 7, "Authorized Signature: ______________________", "0", 0, "L", false, 0, "")
	pdf.CellFormat(90, 7, fmt.Sprintf("Date: %s", time.Now().Format("2006-01-02")), "0", 1, "L", false, 0, "")

	// 输出到buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	// 保存文件
	filename := fmt.Sprintf("Invoice_%s_%s.pdf", shipment.ID, time.Now().Format("20060102"))
	storedPath, err := g.fileService.UploadReader(&buf, filename, FileCategoryGenerated)
	if err != nil {
		return nil, fmt.Errorf("failed to save PDF: %w", err)
	}

	// 创建记录
	genDoc := &models.GeneratedDocument{
		ShipmentID:  shipmentID,
		DocType:     models.TemplateInvoice,
		FileName:    filename,
		FilePath:    storedPath,
		FileSize:    int64(buf.Len()),
		MimeType:    "application/pdf",
		GeneratedBy: userID,
		GeneratedAt: time.Now(),
	}

	if err := g.db.Create(genDoc).Error; err != nil {
		return nil, fmt.Errorf("failed to save record: %w", err)
	}

	return genDoc, nil
}

// 全局PDF生成器实例
var pdfGenerator *PDFGenerator

// InitPDFGenerator 初始化PDF生成器
func InitPDFGenerator(db *gorm.DB, fileService FileService) {
	pdfGenerator = NewPDFGenerator(db, fileService)
}

// GetPDFGenerator 获取PDF生成器
func GetPDFGenerator() *PDFGenerator {
	return pdfGenerator
}
