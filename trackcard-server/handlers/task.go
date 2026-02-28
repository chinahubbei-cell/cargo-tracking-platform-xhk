package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trackcard-server/models"
	"trackcard-server/services"
)

// TaskHandler 任务处理器
type TaskHandler struct {
	db         *gorm.DB
	dispatcher *services.TaskDispatcher
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(db *gorm.DB) *TaskHandler {
	return &TaskHandler{
		db:         db,
		dispatcher: services.GetTaskDispatcher(),
	}
}

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	// 使用分发器创建任务
	task, err := h.dispatcher.CreateTask(&req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, task.ToResponse())
}

// ListTasks 获取任务列表
func (h *TaskHandler) ListTasks(c *gin.Context) {
	var query models.TaskListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tasks, total, err := h.dispatcher.ListTasks(&query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}

	responses := make([]models.TaskResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = t.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"items": responses,
		"total": total,
		"page":  query.Page,
		"size":  query.PageSize,
	})
}

// GetTask 获取任务详情
func (h *TaskHandler) GetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.dispatcher.GetTask(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		}
		return
	}

	c.JSON(http.StatusOK, task.ToResponse())
}

// UpdateTaskStatus 更新任务状态
func (h *TaskHandler) UpdateTaskStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req models.UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.dispatcher.UpdateTaskStatus(uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreateDispatchRule 创建分发规则
func (h *TaskHandler) CreateDispatchRule(c *gin.Context) {
	var req models.CreateDispatchRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.dispatcher.CreateDispatchRule(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// ListDispatchRules 获取分发规则列表
func (h *TaskHandler) ListDispatchRules(c *gin.Context) {
	activeOnly := c.Query("active_only") == "true"

	rules, err := h.dispatcher.ListDispatchRules(activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list rules"})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// GetStats 获取概览统计
func (h *TaskHandler) GetStats(c *gin.Context) {
	shipmentID := c.Query("shipment_id")
	var shipmentIDPtr *string
	if shipmentID != "" {
		shipmentIDPtr = &shipmentID
	}

	stats, err := h.dispatcher.GetTaskStats(shipmentIDPtr)
	if err != nil {
		log.Printf("Failed to get task stats: %v", err) // Log error but return empty stats
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	c.JSON(http.StatusOK, stats)
}
