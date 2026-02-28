package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileService 文件存储服务接口
type FileService interface {
	// Upload 上传文件，返回存储路径
	Upload(file *multipart.FileHeader, category string) (string, error)
	// UploadReader 从Reader上传
	UploadReader(reader io.Reader, filename, category string) (string, error)
	// Download 下载文件，返回文件路径
	GetFilePath(storedPath string) string
	// Delete 删除文件
	Delete(storedPath string) error
	// GetURL 获取文件访问URL (用于前端下载)
	GetURL(storedPath string) string
}

// LocalFileStore 本地文件存储实现
type LocalFileStore struct {
	basePath string // 基础存储路径
	baseURL  string // 访问URL前缀
}

// NewLocalFileStore 创建本地文件存储服务
func NewLocalFileStore(basePath, baseURL string) *LocalFileStore {
	// 确保目录存在
	os.MkdirAll(basePath, 0755)
	absBasePath, _ := filepath.Abs(basePath)
	return &LocalFileStore{
		basePath: absBasePath,
		baseURL:  baseURL,
	}
}

// Upload 上传文件
func (s *LocalFileStore) Upload(file *multipart.FileHeader, category string) (string, error) {
	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// 生成存储路径
	storedPath := s.generatePath(file.Filename, category)
	fullPath := filepath.Join(s.basePath, storedPath)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建目标文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return storedPath, nil
}

// UploadReader 从Reader上传
func (s *LocalFileStore) UploadReader(reader io.Reader, filename, category string) (string, error) {
	// 生成存储路径
	storedPath := s.generatePath(filename, category)
	fullPath := filepath.Join(s.basePath, storedPath)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建目标文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return storedPath, nil
}

// GetFilePath 获取文件完整路径 (带路径遍历保护)
func (s *LocalFileStore) GetFilePath(storedPath string) string {
	fullPath := filepath.Join(s.basePath, storedPath)
	cleanPath := filepath.Clean(fullPath)
	// 安全检查: 确保路径在basePath内
	if !strings.HasPrefix(cleanPath, s.basePath) {
		return "" // 路径遍历攻击，返回空
	}
	return cleanPath
}

// Delete 删除文件 (带路径遍历保护)
func (s *LocalFileStore) Delete(storedPath string) error {
	fullPath := s.GetFilePath(storedPath)
	if fullPath == "" {
		return fmt.Errorf("invalid path")
	}
	return os.Remove(fullPath)
}

// GetURL 获取文件访问URL
func (s *LocalFileStore) GetURL(storedPath string) string {
	if s.baseURL == "" {
		return "/api/files/" + storedPath
	}
	return s.baseURL + "/" + storedPath
}

// generatePath 生成存储路径: category/2026/01/uuid_filename
func (s *LocalFileStore) generatePath(filename, category string) string {
	now := time.Now()
	ext := filepath.Ext(filename)
	uniqueID := uuid.New().String()[:8]

	return fmt.Sprintf("%s/%d/%02d/%s%s",
		category,
		now.Year(),
		now.Month(),
		uniqueID,
		ext,
	)
}

// FileCategory 文件分类常量
const (
	FileCategoryDocuments = "documents" // 运单单据
	FileCategoryGenerated = "generated" // 生成的文档
	FileCategoryTemplates = "templates" // 模板文件
	FileCategoryOCR       = "ocr"       // OCR临时文件
)

// 全局文件服务实例
var fileService FileService

// InitFileService 初始化文件服务
func InitFileService(basePath, baseURL string) {
	fileService = NewLocalFileStore(basePath, baseURL)
}

// GetFileService 获取文件服务实例
func GetFileService() FileService {
	return fileService
}
