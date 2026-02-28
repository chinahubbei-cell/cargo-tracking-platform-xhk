package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error:   message,
	})
}

func BadRequest(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

func InternalError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}

func TooManyRequests(c *gin.Context, retryAfterMinutes int) {
	c.JSON(http.StatusTooManyRequests, Response{
		Success: false,
		Error:   "登录尝试次数过多，请稍后再试",
		Message: "请等待" + string(rune(retryAfterMinutes+'0')) + "分钟后重试",
	})
}
