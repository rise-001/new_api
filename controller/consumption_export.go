package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
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
	download, err := service.GenerateConsumptionExport(c.Request.Context(), c.GetInt("id"), payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer func() {
		if err := download.Close(); err != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("failed to remove temporary consumption export: %v", err))
		}
	}()
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(download.FileName))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(
		c.Writer,
		c.Request,
		download.FileName,
		time.Time{},
		download.File,
	)
}
