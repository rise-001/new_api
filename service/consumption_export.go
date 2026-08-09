package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	consumptionExportBatchSize       = 1000
	consumptionExportMaxRecords      = 50000
	consumptionExportMaxRangeSeconds = int64(366 * 24 * time.Hour / time.Second)
	consumptionExportFileTTL         = time.Hour
	consumptionExportCleanupInterval = time.Minute
)

type ConsumptionExportPayload struct {
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
	TokenID        int    `json:"token_id"`
	TokenName      string `json:"token_name"`
	GroupByToken   bool   `json:"group_by_token"`
	DailySummary   bool   `json:"daily_summary"`
	TimezoneOffset int    `json:"timezone_offset"`
}

func (payload ConsumptionExportPayload) Validate() error {
	if payload.StartTimestamp <= 0 || payload.EndTimestamp <= 0 {
		return errors.New("start time and end time are required")
	}
	if payload.StartTimestamp >= payload.EndTimestamp {
		return errors.New("end time must be later than start time")
	}
	if payload.EndTimestamp-payload.StartTimestamp > consumptionExportMaxRangeSeconds {
		return errors.New("export time range cannot exceed 366 days")
	}
	if payload.TokenID < 0 {
		return errors.New("token id is invalid")
	}
	if payload.TimezoneOffset < -840 || payload.TimezoneOffset > 720 {
		return errors.New("timezone offset is invalid")
	}
	return nil
}

type ConsumptionExportState struct {
	Total     int64  `json:"total"`
	Processed int64  `json:"processed"`
	Progress  int    `json:"progress"`
	Stage     string `json:"stage"`
}

type ConsumptionExportResult struct {
	RecordCount int64  `json:"record_count"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	ExpiresAt   int64  `json:"expires_at"`
	ModelCount  int    `json:"model_count"`
}

type ConsumptionExportTaskResponse struct {
	ID        int64                    `json:"id"`
	TaskID    string                   `json:"task_id"`
	Status    model.SystemTaskStatus   `json:"status"`
	Payload   ConsumptionExportPayload `json:"payload"`
	State     ConsumptionExportState   `json:"state"`
	Result    ConsumptionExportResult  `json:"result"`
	Error     string                   `json:"error"`
	Available bool                     `json:"available"`
	CreatedAt int64                    `json:"created_at"`
	UpdatedAt int64                    `json:"updated_at"`
}

type consumptionExportRecord struct {
	Sequence         int64
	UserID           int
	Username         string
	CreatedAt        string
	ModelName        string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	Quota            int64
	Amount           float64
	Duration         string
	TokenName        string
	LogType          string
	Group            string
	IP               string
	Detail           string
}

type consumptionExportStats struct {
	ModelName     string
	ConsumeCount  int64
	ConsumeAmount float64
	RefundCount   int64
	RefundAmount  float64
}

type consumptionDailyStats struct {
	Date             string
	ModelName        string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	ConsumeCount     int64
	ConsumeAmount    float64
	RefundCount      int64
	RefundAmount     float64
}

type consumptionLogOther struct {
	CacheTokens         int64 `json:"cache_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	CacheCreation5m     int64 `json:"cache_creation_tokens_5m"`
	CacheCreation1h     int64 `json:"cache_creation_tokens_1h"`
	InputTokensTotal    int64 `json:"input_tokens_total"`
	FirstResponseTimeMS int64 `json:"frt"`
}

type consumptionExportHandler struct{}

func (consumptionExportHandler) Type() string {
	return model.SystemTaskTypeConsumptionExport
}

func (consumptionExportHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	runConsumptionExportTask(ctx, task, runnerID)
}

func init() {
	RegisterSystemTaskHandler(consumptionExportHandler{})
}

func StartConsumptionExportTask(userID int, payload ConsumptionExportPayload) (*model.SystemTask, bool, error) {
	if err := payload.Validate(); err != nil {
		return nil, false, err
	}
	if payload.TokenID > 0 {
		token, err := model.GetTokenByIds(payload.TokenID, userID)
		if err != nil {
			return nil, false, errors.New("selected token does not exist")
		}
		payload.TokenName = token.Name
	}

	activeTask, err := model.GetActiveUserSystemTask(model.SystemTaskTypeConsumptionExport, userID)
	if err != nil {
		return nil, false, err
	}
	if activeTask != nil {
		return activeTask, false, nil
	}

	state := ConsumptionExportState{Stage: "pending"}
	task, err := model.CreateUserSystemTask(model.SystemTaskTypeConsumptionExport, userID, payload, state)
	if err != nil {
		activeTask, activeErr := model.GetActiveUserSystemTask(model.SystemTaskTypeConsumptionExport, userID)
		if activeErr == nil && activeTask != nil {
			return activeTask, false, nil
		}
		return nil, false, err
	}
	notifySystemTaskRunner()
	return task, true, nil
}

func ListConsumptionExportTasks(userID int, status model.SystemTaskStatus, offset int, limit int) ([]ConsumptionExportTaskResponse, int64, error) {
	tasks, total, err := model.ListUserSystemTasks(userID, model.SystemTaskTypeConsumptionExport, status, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	now := common.GetTimestamp()
	responses := make([]ConsumptionExportTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		response, err := buildConsumptionExportTaskResponse(task, now)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, response)
	}
	return responses, total, nil
}

func CancelConsumptionExportTask(taskID string, userID int) (bool, error) {
	canceled, err := model.CancelUserSystemTask(taskID, userID, model.SystemTaskTypeConsumptionExport)
	if canceled {
		notifySystemTaskRunner()
	}
	return canceled, err
}

func DeleteConsumptionExportTask(taskID string, userID int) (bool, error) {
	return model.DeleteConsumptionExportTask(taskID, userID)
}

func GetConsumptionExportFile(taskID string, userID int) (*model.ConsumptionExportFile, error) {
	return model.GetConsumptionExportFile(taskID, userID, common.GetTimestamp())
}

func buildConsumptionExportTaskResponse(task *model.SystemTask, now int64) (ConsumptionExportTaskResponse, error) {
	response := ConsumptionExportTaskResponse{
		ID:        task.ID,
		TaskID:    task.TaskID,
		Status:    task.Status,
		Error:     task.Error,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}
	if err := task.DecodePayload(&response.Payload); err != nil {
		return response, err
	}
	if err := task.DecodeState(&response.State); err != nil {
		return response, err
	}
	if task.Result != "" {
		if err := common.UnmarshalJsonStr(task.Result, &response.Result); err != nil {
			return response, err
		}
	}
	response.Available = task.Status == model.SystemTaskStatusSucceeded && response.Result.ExpiresAt > now
	return response, nil
}

func runConsumptionExportTask(ctx context.Context, task *model.SystemTask, runnerID string) {
	payload := ConsumptionExportPayload{}
	if err := task.DecodePayload(&payload); err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	if err := payload.Validate(); err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		failSystemTask(task, runnerID, errors.New("quota per unit must be greater than zero"))
		return
	}

	query := model.ConsumptionExportLogQuery{
		UserID:         task.UserID,
		StartTimestamp: payload.StartTimestamp,
		EndTimestamp:   payload.EndTimestamp,
		TokenID:        payload.TokenID,
	}
	total, err := model.CountConsumptionExportLogs(ctx, query)
	if err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	if total > consumptionExportMaxRecords {
		failSystemTask(task, runnerID, fmt.Errorf("export contains %d records; narrow the time range to at most %d records", total, consumptionExportMaxRecords))
		return
	}

	state := ConsumptionExportState{Total: total, Stage: "querying"}
	if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
		logSystemTaskLockError(ctx, task, err)
		return
	}

	location := time.FixedZone("export", -payload.TimezoneOffset*60)
	records := make([]consumptionExportRecord, 0, total)
	modelStats := make(map[string]*consumptionExportStats)
	dailyStats := make(map[string]*consumptionDailyStats)
	for offset := 0; int64(offset) < total; offset += consumptionExportBatchSize {
		select {
		case <-ctx.Done():
			return
		default:
		}

		logs, err := model.GetConsumptionExportLogs(ctx, query, offset, consumptionExportBatchSize)
		if err != nil {
			failSystemTask(task, runnerID, err)
			return
		}
		for _, log := range logs {
			record := consumptionRecordFromLog(log, int64(len(records)+1), location)
			records = append(records, record)
			stats := modelStats[record.ModelName]
			if stats == nil {
				stats = &consumptionExportStats{ModelName: record.ModelName}
				modelStats[record.ModelName] = stats
			}
			if log.Type == model.LogTypeRefund {
				stats.RefundCount++
				stats.RefundAmount += -record.Amount
			} else {
				stats.ConsumeCount++
				stats.ConsumeAmount += record.Amount
			}

			date := time.Unix(log.CreatedAt, 0).In(location).Format("2006-01-02")
			dailyKey := date + "\x00" + record.ModelName
			daily := dailyStats[dailyKey]
			if daily == nil {
				daily = &consumptionDailyStats{Date: date, ModelName: record.ModelName}
				dailyStats[dailyKey] = daily
			}
			daily.PromptTokens += record.PromptTokens
			daily.CompletionTokens += record.CompletionTokens
			daily.TotalTokens += record.TotalTokens
			if log.Type == model.LogTypeRefund {
				daily.RefundCount++
				daily.RefundAmount += -record.Amount
			} else {
				daily.ConsumeCount++
				daily.ConsumeAmount += record.Amount
			}
		}

		state.Processed += int64(len(logs))
		if total > 0 {
			state.Progress = int(state.Processed * 85 / total)
		}
		if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
			logSystemTaskLockError(ctx, task, err)
			return
		}
	}

	state.Stage = "building"
	state.Progress = 90
	if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
		logSystemTaskLockError(ctx, task, err)
		return
	}
	workbook, err := buildConsumptionExportWorkbook(payload, records, modelStats, dailyStats)
	if err != nil {
		failSystemTask(task, runnerID, err)
		return
	}
	select {
	case <-ctx.Done():
		return
	default:
	}

	state.Stage = "saving"
	state.Progress = 95
	if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
		logSystemTaskLockError(ctx, task, err)
		return
	}
	completedAt := common.GetTimestamp()
	fileName := fmt.Sprintf("consumption-%s.xlsx", time.Unix(completedAt, 0).In(location).Format("20060102-150405"))
	result := ConsumptionExportResult{
		RecordCount: total,
		FileName:    fileName,
		FileSize:    int64(len(workbook)),
		ExpiresAt:   completedAt + int64(consumptionExportFileTTL.Seconds()),
		ModelCount:  len(modelStats),
	}
	file := &model.ConsumptionExportFile{
		TaskID:    task.TaskID,
		UserID:    task.UserID,
		FileName:  fileName,
		Content:   workbook,
		ExpiresAt: result.ExpiresAt,
	}
	if err := model.SaveConsumptionExportFile(file); err != nil {
		failSystemTask(task, runnerID, err)
		return
	}

	state.Stage = "completed"
	state.Progress = 100
	if err := model.UpdateSystemTaskState(task.TaskID, runnerID, state); err != nil {
		_ = model.DeleteConsumptionExportFile(task.TaskID, task.UserID)
		logSystemTaskLockError(ctx, task, err)
		return
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, model.SystemTaskStatusSucceeded, result, ""); err != nil {
		_ = model.DeleteConsumptionExportFile(task.TaskID, task.UserID)
		logSystemTaskLockError(ctx, task, err)
	}
}

func consumptionRecordFromLog(log *model.Log, sequence int64, location *time.Location) consumptionExportRecord {
	other := consumptionLogOther{}
	if log.Other != "" {
		_ = common.UnmarshalJsonStr(log.Other, &other)
	}
	promptTokens := int64(log.PromptTokens)
	if other.InputTokensTotal > 0 {
		promptTokens = other.InputTokensTotal
	}
	cacheWriteTokens := other.CacheCreationTokens
	if other.CacheCreation5m+other.CacheCreation1h > 0 {
		cacheWriteTokens = other.CacheCreation5m + other.CacheCreation1h
	}
	sign := int64(1)
	logType := "消费"
	if log.Type == model.LogTypeRefund {
		sign = -1
		logType = "退款"
	}
	amount := float64(log.Quota) / common.QuotaPerUnit * float64(sign)
	duration := fmt.Sprintf("%.1fs", float64(log.UseTime))
	if other.FirstResponseTimeMS > 0 {
		duration += fmt.Sprintf(" / %.1fs", float64(other.FirstResponseTimeMS)/1000)
	}
	return consumptionExportRecord{
		Sequence:         sequence,
		UserID:           log.UserId,
		Username:         log.Username,
		CreatedAt:        time.Unix(log.CreatedAt, 0).In(location).Format("2006-01-02 15:04:05"),
		ModelName:        log.ModelName,
		PromptTokens:     promptTokens,
		CompletionTokens: int64(log.CompletionTokens),
		TotalTokens:      promptTokens + int64(log.CompletionTokens),
		CacheReadTokens:  other.CacheTokens,
		CacheWriteTokens: cacheWriteTokens,
		Quota:            int64(log.Quota) * sign,
		Amount:           amount,
		Duration:         duration,
		TokenName:        log.TokenName,
		LogType:          logType,
		Group:            log.Group,
		IP:               log.Ip,
		Detail:           log.Content,
	}
}

func buildConsumptionExportWorkbook(payload ConsumptionExportPayload, records []consumptionExportRecord, modelStats map[string]*consumptionExportStats, dailyStats map[string]*consumptionDailyStats) ([]byte, error) {
	sheets := make([]xlsxSheet, 0)
	if payload.DailySummary {
		sheets = append(sheets, buildDailySummarySheet(dailyStats))
	} else if payload.GroupByToken {
		recordsByToken := make(map[string][]consumptionExportRecord)
		for _, record := range records {
			name := record.TokenName
			if name == "" {
				name = "未命名令牌"
			}
			recordsByToken[name] = append(recordsByToken[name], record)
		}
		tokenNames := make([]string, 0, len(recordsByToken))
		for name := range recordsByToken {
			tokenNames = append(tokenNames, name)
		}
		sort.Strings(tokenNames)
		usedSheetNames := make(map[string]int)
		for _, tokenName := range tokenNames {
			sheetName := uniqueSheetName(tokenName, usedSheetNames)
			sheets = append(sheets, buildConsumptionDetailSheet(sheetName, recordsByToken[tokenName]))
		}
		if len(tokenNames) == 0 {
			sheets = append(sheets, buildConsumptionDetailSheet("消费清单", nil))
		}
	} else {
		sheets = append(sheets, buildConsumptionDetailSheet("消费清单", records))
	}
	sheets = append(sheets, buildModelSummarySheet(modelStats))
	return buildXLSX(sheets)
}

func buildConsumptionDetailSheet(name string, records []consumptionExportRecord) xlsxSheet {
	columns := []xlsxColumn{
		{Header: "序号", Width: 8}, {Header: "用户ID", Width: 10}, {Header: "用户名", Width: 16},
		{Header: "消费时间", Width: 20}, {Header: "模型名称", Width: 26}, {Header: "输入Token", Width: 13},
		{Header: "输出Token", Width: 13}, {Header: "总Token", Width: 13}, {Header: "缓存读Token", Width: 14},
		{Header: "缓存写Token", Width: 14}, {Header: "消费额度", Width: 13}, {Header: "消费金额($)", Width: 15},
		{Header: "用时/首字", Width: 14}, {Header: "令牌名称", Width: 20}, {Header: "日志类型", Width: 12},
		{Header: "分组", Width: 14}, {Header: "IP", Width: 16}, {Header: "日志详情", Width: 48},
	}
	rows := make([][]xlsxCell, 0, len(records)+1)
	var totalPrompt, totalCompletion, totalTokens, totalCacheRead, totalCacheWrite, totalQuota int64
	var totalAmount float64
	for _, record := range records {
		rows = append(rows, []xlsxCell{
			integerCell(record.Sequence), integerCell(int64(record.UserID)), textCell(record.Username), textCell(record.CreatedAt),
			textCell(record.ModelName), integerCell(record.PromptTokens), integerCell(record.CompletionTokens), integerCell(record.TotalTokens),
			integerCell(record.CacheReadTokens), integerCell(record.CacheWriteTokens), integerCell(record.Quota), amountCell(record.Amount),
			textCell(record.Duration), textCell(record.TokenName), textCell(record.LogType), textCell(record.Group), textCell(record.IP), textCell(record.Detail),
		})
		totalPrompt += record.PromptTokens
		totalCompletion += record.CompletionTokens
		totalTokens += record.TotalTokens
		totalCacheRead += record.CacheReadTokens
		totalCacheWrite += record.CacheWriteTokens
		totalQuota += record.Quota
		totalAmount += record.Amount
	}
	rows = append(rows, []xlsxCell{
		totalTextCell("合计"), totalTextCell(""), totalTextCell(""), totalTextCell(""), totalTextCell(""),
		totalIntegerCell(totalPrompt), totalIntegerCell(totalCompletion), totalIntegerCell(totalTokens), totalIntegerCell(totalCacheRead),
		totalIntegerCell(totalCacheWrite), totalIntegerCell(totalQuota), totalAmountCell(totalAmount), totalTextCell(""), totalTextCell(""),
		totalTextCell(""), totalTextCell(""), totalTextCell(""), totalTextCell(""),
	})
	return xlsxSheet{Name: name, Columns: columns, Rows: rows}
}

func buildModelSummarySheet(statsByModel map[string]*consumptionExportStats) xlsxSheet {
	columns := []xlsxColumn{
		{Header: "模型名称", Width: 30}, {Header: "成功记录数", Width: 14}, {Header: "消费金额($)", Width: 15},
		{Header: "退款记录数", Width: 14}, {Header: "退款金额($)", Width: 15}, {Header: "净消费金额($)", Width: 15},
	}
	modelNames := make([]string, 0, len(statsByModel))
	for name := range statsByModel {
		modelNames = append(modelNames, name)
	}
	sort.Strings(modelNames)
	rows := make([][]xlsxCell, 0, len(modelNames)+1)
	var consumeCount, refundCount int64
	var consumeAmount, refundAmount float64
	for _, name := range modelNames {
		stats := statsByModel[name]
		rows = append(rows, []xlsxCell{
			textCell(stats.ModelName), integerCell(stats.ConsumeCount), amountCell(stats.ConsumeAmount),
			integerCell(stats.RefundCount), amountCell(stats.RefundAmount), amountCell(stats.ConsumeAmount - stats.RefundAmount),
		})
		consumeCount += stats.ConsumeCount
		refundCount += stats.RefundCount
		consumeAmount += stats.ConsumeAmount
		refundAmount += stats.RefundAmount
	}
	rows = append(rows, []xlsxCell{
		totalTextCell("合计"), totalIntegerCell(consumeCount), totalAmountCell(consumeAmount), totalIntegerCell(refundCount),
		totalAmountCell(refundAmount), totalAmountCell(consumeAmount - refundAmount),
	})
	return xlsxSheet{Name: "模型统计", Columns: columns, Rows: rows}
}

func buildDailySummarySheet(statsByDay map[string]*consumptionDailyStats) xlsxSheet {
	columns := []xlsxColumn{
		{Header: "日期", Width: 14}, {Header: "模型名称", Width: 30}, {Header: "输入Token", Width: 13},
		{Header: "输出Token", Width: 13}, {Header: "总Token", Width: 13}, {Header: "消费记录数", Width: 14},
		{Header: "消费金额($)", Width: 15}, {Header: "退款记录数", Width: 14}, {Header: "退款金额($)", Width: 15},
		{Header: "净消费金额($)", Width: 15},
	}
	keys := make([]string, 0, len(statsByDay))
	for key := range statsByDay {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([][]xlsxCell, 0, len(keys)+1)
	var totalPrompt, totalCompletion, totalTokens, consumeCount, refundCount int64
	var consumeAmount, refundAmount float64
	for _, key := range keys {
		stats := statsByDay[key]
		rows = append(rows, []xlsxCell{
			textCell(stats.Date), textCell(stats.ModelName), integerCell(stats.PromptTokens), integerCell(stats.CompletionTokens),
			integerCell(stats.TotalTokens), integerCell(stats.ConsumeCount), amountCell(stats.ConsumeAmount), integerCell(stats.RefundCount),
			amountCell(stats.RefundAmount), amountCell(stats.ConsumeAmount - stats.RefundAmount),
		})
		totalPrompt += stats.PromptTokens
		totalCompletion += stats.CompletionTokens
		totalTokens += stats.TotalTokens
		consumeCount += stats.ConsumeCount
		consumeAmount += stats.ConsumeAmount
		refundCount += stats.RefundCount
		refundAmount += stats.RefundAmount
	}
	rows = append(rows, []xlsxCell{
		totalTextCell("合计"), totalTextCell(""), totalIntegerCell(totalPrompt), totalIntegerCell(totalCompletion), totalIntegerCell(totalTokens),
		totalIntegerCell(consumeCount), totalAmountCell(consumeAmount), totalIntegerCell(refundCount), totalAmountCell(refundAmount),
		totalAmountCell(consumeAmount - refundAmount),
	})
	return xlsxSheet{Name: "每日汇总", Columns: columns, Rows: rows}
}

func uniqueSheetName(name string, used map[string]int) string {
	name = sanitizeXLSXText(name)
	name = strings.Map(func(char rune) rune {
		if strings.ContainsRune(`[]:*?/\`, char) {
			return '-'
		}
		return char
	}, strings.TrimSpace(name))
	name = strings.Trim(name, "'")
	if name == "" {
		name = "未命名令牌"
	}
	for utf8.RuneCountInString(name) > 31 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	base := name
	count := used[base]
	for count > 0 {
		count++
		suffix := fmt.Sprintf("-%d", count)
		name = base
		for utf8.RuneCountInString(name)+utf8.RuneCountInString(suffix) > 31 {
			_, size := utf8.DecodeLastRuneInString(name)
			name = name[:len(name)-size]
		}
		name += suffix
		if used[name] == 0 {
			break
		}
	}
	used[base] = used[base] + 1
	used[name] = 1
	return name
}

var consumptionExportCleanupOnce sync.Once

func StartConsumptionExportCleanup() {
	consumptionExportCleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			cleanup := func() {
				if _, err := model.DeleteExpiredConsumptionExportFiles(context.Background(), common.GetTimestamp()); err != nil {
					logger.LogWarn(context.Background(), fmt.Sprintf("consumption export cleanup failed: %v", err))
				}
			}
			cleanup()
			ticker := time.NewTicker(consumptionExportCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				cleanup()
			}
		})
	})
}
