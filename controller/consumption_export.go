package controller

import (
	"net/http"
	"net/url"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type createConsumptionExportRequest struct {
	StartTimestamp int64 `json:"start_timestamp" binding:"required"`
	EndTimestamp   int64 `json:"end_timestamp" binding:"required"`
	TokenID        int   `json:"token_id"`
	GroupByToken   bool  `json:"group_by_token"`
	DailySummary   bool  `json:"daily_summary"`
	TimezoneOffset int   `json:"timezone_offset"`
}

func CreateConsumptionExport(c *gin.Context) {
	request := createConsumptionExportRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	payload := service.ConsumptionExportPayload{
		StartTimestamp: request.StartTimestamp,
		EndTimestamp:   request.EndTimestamp,
		TokenID:        request.TokenID,
		GroupByToken:   request.GroupByToken,
		DailySummary:   request.DailySummary,
		TimezoneOffset: request.TimezoneOffset,
	}
	task, created, err := service.StartConsumptionExportTask(c.GetInt("id"), payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"task_id": task.TaskID,
			"status":  task.Status,
			"created": created,
		},
	})
}

func ListConsumptionExports(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := model.SystemTaskStatus(c.Query("status"))
	switch status {
	case "", model.SystemTaskStatusPending, model.SystemTaskStatusRunning, model.SystemTaskStatusSucceeded, model.SystemTaskStatusFailed, model.SystemTaskStatusCanceled:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid task status"})
		return
	}
	tasks, total, err := service.ListConsumptionExportTasks(c.GetInt("id"), status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasks)
	common.ApiSuccess(c, pageInfo)
}

func CancelConsumptionExport(c *gin.Context) {
	canceled, err := service.CancelConsumptionExportTask(c.Param("task_id"), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !canceled {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "export task is not active"})
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteConsumptionExport(c *gin.Context) {
	deleted, err := service.DeleteConsumptionExportTask(c.Param("task_id"), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !deleted {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "active export tasks cannot be deleted"})
		return
	}
	common.ApiSuccess(c, nil)
}

func DownloadConsumptionExport(c *gin.Context) {
	file, err := service.GetConsumptionExportFile(c.Param("task_id"), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if file == nil {
		c.JSON(http.StatusGone, gin.H{"success": false, "message": "export file has expired or does not exist"})
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(file.FileName))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", file.Content)
}
