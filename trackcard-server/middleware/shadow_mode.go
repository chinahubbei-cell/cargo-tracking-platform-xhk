package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
)

// ShadowModeInfo 影子模式信息（存储在上下文中）
type ShadowModeInfo struct {
	Enabled       bool
	ShadowForType string // user/partner/customer
	ShadowForID   string
	ShadowForName string
	Reason        string
}

// ShadowModeMiddleware 影子模式中间件
// 检测请求头中的影子模式标记，记录代操作日志
func ShadowModeMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用影子模式
		shadowForType := c.GetHeader("X-Shadow-For-Type")
		shadowForID := c.GetHeader("X-Shadow-For-ID")
		shadowForName := c.GetHeader("X-Shadow-For-Name")
		shadowReason := c.GetHeader("X-Shadow-Reason")

		if shadowForType != "" && shadowForID != "" {
			// 验证当前用户是否有权限使用影子模式
			userRole, exists := c.Get("user_role")
			if !exists {
				c.AbortWithStatusJSON(403, gin.H{"error": "无法验证用户身份"})
				return
			}

			// 只有admin和operator可以使用影子模式
			role, ok := userRole.(string)
			if !ok || (role != "admin" && role != "operator") {
				c.AbortWithStatusJSON(403, gin.H{"error": "您没有权限使用影子模式"})
				return
			}

			// 设置影子模式信息
			shadowInfo := &ShadowModeInfo{
				Enabled:       true,
				ShadowForType: shadowForType,
				ShadowForID:   shadowForID,
				ShadowForName: shadowForName,
				Reason:        shadowReason,
			}
			c.Set("shadow_mode", shadowInfo)
			c.Set("is_shadow_mode", true)
			c.Set("shadow_target", shadowForID)

			// 读取请求体（用于日志）
			var requestBody string
			if c.Request.Body != nil {
				bodyBytes, _ := io.ReadAll(c.Request.Body)
				requestBody = string(bodyBytes)
				// 重置请求体
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// 记录影子操作开始
			operatorID, _ := c.Get("user_id")
			operatorName, _ := c.Get("user_name")
			operatorRole, _ := c.Get("user_role")

			// 安全的类型断言
			operatorIDStr := ""
			if id, ok := operatorID.(string); ok {
				operatorIDStr = id
			}

			log := &models.ShadowOperationLog{
				OperatorID:    operatorIDStr,
				OperatorName:  getString(operatorName),
				OperatorRole:  getString(operatorRole),
				ShadowForType: shadowForType,
				ShadowForID:   shadowForID,
				ShadowForName: shadowForName,
				Action:        c.Request.Method + " " + c.FullPath(),
				Resource:      c.Request.URL.Path,
				Method:        c.Request.Method,
				RequestBody:   requestBody,
				ClientIP:      c.ClientIP(),
				UserAgent:     c.Request.UserAgent(),
				Reason:        shadowReason,
				CreatedAt:     time.Now(),
			}

			// 提取关联资源
			if shipmentID := c.Param("id"); shipmentID != "" && strings.Contains(c.FullPath(), "shipments") {
				log.ShipmentID = &shipmentID
			}

			// 继续处理请求
			c.Next()

			// 请求完成后保存日志
			log.ResponseCode = c.Writer.Status()
			go db.Create(log)
			return
		}

		c.Next()
	}
}

// AuditMiddleware 通用审计日志中间件
// 记录所有写操作（POST/PUT/DELETE）
func AuditMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只记录写操作
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			c.Next()
			return
		}

		// 跳过某些路径
		path := c.Request.URL.Path
		skipPaths := []string{"/api/auth/login", "/api/health", "/api/m/"}
		for _, skip := range skipPaths {
			if strings.HasPrefix(path, skip) {
				c.Next()
				return
			}
		}

		// 获取用户信息
		userID, _ := c.Get("user_id")
		userName, _ := c.Get("user_name")
		userRole, _ := c.Get("user_role")

		if userID == nil {
			c.Next()
			return
		}

		// 安全的类型断言
		userIDStr, ok := userID.(string)
		if !ok {
			c.Next()
			return
		}

		// 检查是否影子模式
		shadowInfo, _ := c.Get("shadow_mode")
		isShadowMode := shadowInfo != nil

		// 构建审计日志
		auditLog := &models.AuditLog{
			UserID:       userIDStr,
			UserName:     getString(userName),
			UserRole:     getString(userRole),
			IsShadowMode: isShadowMode,
			Method:       method,
			Path:         path,
			ClientIP:     c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			CreatedAt:    time.Now(),
		}

		// 影子模式信息
		if isShadowMode {
			info := shadowInfo.(*ShadowModeInfo)
			auditLog.ShadowForID = &info.ShadowForID
			auditLog.ShadowForName = &info.ShadowForName
		}

		// 解析资源类型和ID
		auditLog.ResourceType = extractResourceType(path)
		auditLog.ResourceID = c.Param("id")
		auditLog.Action = methodToAction(method)
		auditLog.Description = generateDescription(method, auditLog.ResourceType, auditLog.ResourceID)

		// 继续处理请求
		c.Next()

		// 请求完成后保存日志
		auditLog.StatusCode = c.Writer.Status()
		go db.Create(auditLog)
	}
}

// ==================== 辅助函数 ====================

func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func extractResourceType(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		// /api/shipments/xxx -> shipments
		// /api/documents/xxx -> documents
		return parts[1]
	}
	return "unknown"
}

func methodToAction(method string) models.AuditActionType {
	switch method {
	case "POST":
		return models.AuditActionCreate
	case "PUT", "PATCH":
		return models.AuditActionUpdate
	case "DELETE":
		return models.AuditActionDelete
	default:
		return models.AuditActionView
	}
}

func generateDescription(method, resourceType, resourceID string) string {
	typeNames := map[string]string{
		"shipments":     "运单",
		"documents":     "文档",
		"users":         "用户",
		"devices":       "设备",
		"organizations": "组织",
		"partners":      "合作伙伴",
		"rates":         "费率",
		"alerts":        "预警",
		"milestones":    "节点",
		"magic-links":   "魔术链接",
	}

	typeName := typeNames[resourceType]
	if typeName == "" {
		typeName = resourceType
	}

	switch method {
	case "POST":
		return "创建" + typeName
	case "PUT", "PATCH":
		if resourceID != "" {
			return "更新" + typeName + " " + resourceID
		}
		return "更新" + typeName
	case "DELETE":
		if resourceID != "" {
			return "删除" + typeName + " " + resourceID
		}
		return "删除" + typeName
	default:
		return "操作" + typeName
	}
}

// GetShadowModeInfo 从上下文获取影子模式信息
func GetShadowModeInfo(c *gin.Context) *ShadowModeInfo {
	if info, exists := c.Get("shadow_mode"); exists {
		return info.(*ShadowModeInfo)
	}
	return nil
}

// IsShadowMode 检查是否处于影子模式
func IsShadowMode(c *gin.Context) bool {
	if isShadow, exists := c.Get("is_shadow_mode"); exists {
		return isShadow.(bool)
	}
	return false
}
