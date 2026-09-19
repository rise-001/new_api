package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
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
	consumptionExportMaxRecords      = 300000
	consumptionExportMaxTokenSheets  = 200
	consumptionExportMaxRangeSeconds = int64(31 * 24 * time.Hour / time.Second)
	consumptionExportCleanupInterval = time.Minute
	consumptionExportTempFileTTL     = 24 * time.Hour

	// Both patterns double as os.CreateTemp patterns and filepath.Glob
	// patterns, so the startup cleanup always covers what the exporter writes.
	consumptionExportFilePattern = "new-api-consumption-*.xlsx"
	consumptionExportRowsPattern = "new-api-consumption-rows-*.xml"
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
		return errors.New("export time range cannot exceed 31 days")
	}
	if payload.TokenID < 0 {
		return errors.New("token id is invalid")
	}
	if payload.TimezoneOffset < -840 || payload.TimezoneOffset > 720 {
		return errors.New("timezone offset is invalid")
	}
	return nil
}

type ConsumptionExportDownload struct {
	File     *os.File
	FileName string
	userID   int
	closeErr error
	close    sync.Once
}

func (download *ConsumptionExportDownload) Close() error {
	download.close.Do(func() {
		defer activeConsumptionExports.Delete(download.userID)
		closeErr := download.File.Close()
		removeErr := os.Remove(download.File.Name())
		download.closeErr = errors.Join(closeErr, removeErr)
	})
	return download.closeErr
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

type consumptionTokenStats struct {
	TokenName        string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ConsumeCount     int64
	ConsumeAmount    float64
	RefundCount      int64
	RefundAmount     float64
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

type consumptionDetailTotals struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	Quota            int64
	Amount           float64
}

type consumptionLogOther struct {
	CacheTokens         int64 `json:"cache_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	CacheCreation5m     int64 `json:"cache_creation_tokens_5m"`
	CacheCreation1h     int64 `json:"cache_creation_tokens_1h"`
	InputTokensTotal    int64 `json:"input_tokens_total"`
	FirstResponseTimeMS int64 `json:"frt"`
}

var activeConsumptionExports sync.Map

func GenerateConsumptionExport(ctx context.Context, userID int, payload ConsumptionExportPayload) (*ConsumptionExportDownload, error) {
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	if payload.TokenID > 0 {
		token, err := model.GetTokenByIds(payload.TokenID, userID)
		if err != nil {
			return nil, errors.New("selected token does not exist")
		}
		payload.TokenName = token.Name
	}
	if common.QuotaPerUnit <= 0 || math.IsNaN(common.QuotaPerUnit) || math.IsInf(common.QuotaPerUnit, 0) {
		return nil, errors.New("quota per unit must be greater than zero")
	}
	if _, loaded := activeConsumptionExports.LoadOrStore(userID, struct{}{}); loaded {
		return nil, errors.New("An export is already being generated")
	}

	temporaryFile, err := os.CreateTemp("", consumptionExportFilePattern)
	if err != nil {
		activeConsumptionExports.Delete(userID)
		return nil, err
	}
	download := &ConsumptionExportDownload{File: temporaryFile, userID: userID}
	succeeded := false
	defer func() {
		if !succeeded {
			_ = download.Close()
		}
	}()

	query := model.ConsumptionExportLogQuery{
		UserID:         userID,
		StartTimestamp: payload.StartTimestamp,
		EndTimestamp:   payload.EndTimestamp,
		TokenID:        payload.TokenID,
	}

	// The detail worksheets are laid out before any row is read, so the pass
	// below can write every row as it arrives while keeping row numbers
	// continuous across per-token worksheets. Counting is an aggregate query;
	// it does not read the rows themselves.
	total := int64(0)
	tokenCounts := map[string]int64{}
	if payload.GroupByToken && !payload.DailySummary {
		tokenCounts, err = model.CountConsumptionExportLogsByToken(ctx, query)
		if err != nil {
			return nil, err
		}
		if len(tokenCounts) > consumptionExportMaxTokenSheets {
			return nil, fmt.Errorf("export spans %d API tokens but at most %d separate worksheets are supported; turn off the per-token worksheet option", len(tokenCounts), consumptionExportMaxTokenSheets)
		}
		for _, count := range tokenCounts {
			total += count
		}
	} else {
		total, err = model.CountConsumptionExportLogs(ctx, query)
		if err != nil {
			return nil, err
		}
	}
	if total > consumptionExportMaxRecords {
		return nil, fmt.Errorf("export contains %d records; narrow the time range to at most %d records", total, consumptionExportMaxRecords)
	}

	detail, detailErr := newConsumptionExportDetail(payload, tokenCounts, total)
	defer detail.close(ctx)
	if detailErr != nil {
		return nil, detailErr
	}

	location := time.FixedZone("export", -payload.TimezoneOffset*60)
	modelStats := make(map[string]*consumptionExportStats)
	tokenStats := make(map[string]*consumptionTokenStats)
	dailyStats := make(map[string]*consumptionDailyStats)
	recordCount := int64(0)
	err = forEachConsumptionExportLog(ctx, query, func(log *model.Log) error {
		recordCount++
		if recordCount > consumptionExportMaxRecords {
			return fmt.Errorf("export contains more than %d records; narrow the time range", consumptionExportMaxRecords)
		}
		record := consumptionRecordFromLog(log, location)
		token := tokenStats[record.TokenName]
		if token == nil {
			token = &consumptionTokenStats{TokenName: record.TokenName}
			tokenStats[record.TokenName] = token
		}
		token.PromptTokens += record.PromptTokens
		token.CompletionTokens += record.CompletionTokens
		token.TotalTokens += record.TotalTokens
		token.CacheReadTokens += record.CacheReadTokens
		token.CacheWriteTokens += record.CacheWriteTokens
		if log.Type == model.LogTypeRefund {
			token.RefundCount++
			token.RefundAmount += -record.Amount
		} else {
			token.ConsumeCount++
			token.ConsumeAmount += record.Amount
		}

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
		return detail.write(record)
	})
	if err != nil {
		return nil, err
	}

	sheets := []xlsxSheet{buildTokenSummarySheet(tokenStats)}
	if payload.DailySummary {
		sheets = append(sheets, buildDailySummarySheet(dailyStats))
	} else {
		detailSheets, err := detail.finish()
		if err != nil {
			return nil, err
		}
		sheets = append(sheets, detailSheets...)
	}
	sheets = append(sheets, buildModelSummarySheet(modelStats))

	if err := writeXLSX(temporaryFile, sheets); err != nil {
		return nil, err
	}
	if _, err := temporaryFile.Seek(0, 0); err != nil {
		return nil, err
	}
	download.FileName = fmt.Sprintf("consumption-%s.xlsx", time.Now().In(location).Format("20060102-150405"))
	succeeded = true
	return download, nil
}

func forEachConsumptionExportLog(ctx context.Context, query model.ConsumptionExportLogQuery, visit func(*model.Log) error) error {
	cursor := model.ConsumptionExportLogCursor{}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		logs, err := model.GetConsumptionExportLogsAfter(ctx, query, cursor, consumptionExportBatchSize)
		if err != nil {
			return err
		}
		for _, log := range logs {
			if err := visit(log); err != nil {
				return err
			}
		}
		if len(logs) < consumptionExportBatchSize {
			return nil
		}
		last := logs[len(logs)-1]
		cursor.CreatedAt = last.CreatedAt
		cursor.ID = last.Id
		cursor.Offset += len(logs)
	}
}

// consumptionExportDetailSheet accumulates one worksheet of consumption detail
// rows in a temporary file while the logs are read. start is the row number
// that precedes this worksheet, so row numbers stay continuous when the export
// is split into one worksheet per API token.
type consumptionExportDetailSheet struct {
	name     string
	file     *os.File
	rows     *xlsxRowWriter
	totals   consumptionDetailTotals
	start    int64
	expected int64
	written  int64
}

// consumptionExportDetail routes every consumption record to its worksheet
// during the single pass over the logs. Daily-summary exports carry no detail
// worksheets, in which case it accepts and drops every record.
type consumptionExportDetail struct {
	sheets  []*consumptionExportDetailSheet
	byToken map[string]*consumptionExportDetailSheet
}

func newConsumptionExportDetailSheet(name string, start int64, expected int64) (*consumptionExportDetailSheet, error) {
	file, err := os.CreateTemp("", consumptionExportRowsPattern)
	if err != nil {
		return nil, err
	}
	return &consumptionExportDetailSheet{
		name:     name,
		file:     file,
		rows:     newXLSXRowWriter(file, len(consumptionDetailColumns())),
		start:    start,
		expected: expected,
	}, nil
}

func newConsumptionExportDetail(payload ConsumptionExportPayload, tokenCounts map[string]int64, total int64) (*consumptionExportDetail, error) {
	detail := &consumptionExportDetail{}
	if payload.DailySummary {
		return detail, nil
	}

	if !payload.GroupByToken || len(tokenCounts) == 0 {
		sheet, err := newConsumptionExportDetailSheet("消费清单", 0, total)
		if err != nil {
			return detail, err
		}
		detail.sheets = append(detail.sheets, sheet)
		return detail, nil
	}

	tokenNames := make([]string, 0, len(tokenCounts))
	for tokenName := range tokenCounts {
		tokenNames = append(tokenNames, tokenName)
	}
	sort.Strings(tokenNames)
	usedSheetNames := map[string]int{"令牌汇总": 1, "模型统计": 1}
	detail.byToken = make(map[string]*consumptionExportDetailSheet, len(tokenNames))
	start := int64(0)
	for _, tokenName := range tokenNames {
		displayName := tokenName
		if displayName == "" {
			displayName = "未命名令牌"
		}
		sheet, err := newConsumptionExportDetailSheet(uniqueSheetName(displayName, usedSheetNames), start, tokenCounts[tokenName])
		if err != nil {
			return detail, err
		}
		detail.sheets = append(detail.sheets, sheet)
		detail.byToken[tokenName] = sheet
		start += tokenCounts[tokenName]
	}
	return detail, nil
}

func (detail *consumptionExportDetail) write(record consumptionExportRecord) error {
	if len(detail.sheets) == 0 {
		return nil
	}
	sheet := detail.sheets[0]
	if detail.byToken != nil {
		sheet = detail.byToken[record.TokenName]
		if sheet == nil {
			return fmt.Errorf("consumption export changed while being generated: API token %q appeared after the worksheets were planned; retry", record.TokenName)
		}
	}
	if sheet.written >= sheet.expected {
		return fmt.Errorf("consumption export changed while being generated: worksheet %q expected %d records, found at least %d; retry", sheet.name, sheet.expected, sheet.written+1)
	}
	sheet.written++
	record.Sequence = sheet.start + sheet.written
	sheet.totals.Add(record)
	return sheet.rows.Write(consumptionDetailRow(record))
}

// finish appends the totals row to every worksheet and rewinds its temporary
// file so the workbook writer can copy the rendered rows straight in.
func (detail *consumptionExportDetail) finish() ([]xlsxSheet, error) {
	sheets := make([]xlsxSheet, 0, len(detail.sheets))
	for _, sheet := range detail.sheets {
		if sheet.written != sheet.expected {
			return nil, fmt.Errorf("consumption export changed while being generated: worksheet %q expected %d records, found %d; retry", sheet.name, sheet.expected, sheet.written)
		}
		if err := sheet.rows.Write(sheet.totals.Row()); err != nil {
			return nil, err
		}
		if err := sheet.rows.Flush(); err != nil {
			return nil, err
		}
		if _, err := sheet.file.Seek(0, 0); err != nil {
			return nil, err
		}
		sheets = append(sheets, xlsxSheet{
			Name:     sheet.name,
			Columns:  consumptionDetailColumns(),
			RawRows:  sheet.file,
			RowCount: sheet.rows.Count(),
		})
	}
	return sheets, nil
}

func (detail *consumptionExportDetail) close(ctx context.Context) {
	for _, sheet := range detail.sheets {
		path := sheet.file.Name()
		closeErr := sheet.file.Close()
		removeErr := os.Remove(path)
		if err := errors.Join(closeErr, removeErr); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("failed to remove temporary consumption export rows: %v", err))
		}
	}
}

func consumptionRecordFromLog(log *model.Log, location *time.Location) consumptionExportRecord {
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

func buildTokenSummarySheet(statsByToken map[string]*consumptionTokenStats) xlsxSheet {
	columns := []xlsxColumn{
		{Header: "令牌名称", Width: 24}, {Header: "消费笔数", Width: 12}, {Header: "退款笔数", Width: 12},
		{Header: "输入Token", Width: 13}, {Header: "输出Token", Width: 13}, {Header: "总Token", Width: 13},
		{Header: "缓存读Token", Width: 14}, {Header: "缓存写Token", Width: 14}, {Header: "消费金额($)", Width: 15},
		{Header: "退款金额($)", Width: 15}, {Header: "净消费金额($)", Width: 15},
	}
	tokenNames := make([]string, 0, len(statsByToken))
	for name := range statsByToken {
		tokenNames = append(tokenNames, name)
	}
	sort.Strings(tokenNames)
	rows := make([][]xlsxCell, 0, len(tokenNames)+1)
	var promptTokens, completionTokens, totalTokens, cacheReadTokens, cacheWriteTokens int64
	var consumeCount, refundCount int64
	var consumeAmount, refundAmount float64
	for _, name := range tokenNames {
		stats := statsByToken[name]
		displayName := stats.TokenName
		if displayName == "" {
			displayName = "未命名令牌"
		}
		rows = append(rows, []xlsxCell{
			textCell(displayName), integerCell(stats.ConsumeCount), integerCell(stats.RefundCount),
			integerCell(stats.PromptTokens), integerCell(stats.CompletionTokens), integerCell(stats.TotalTokens),
			integerCell(stats.CacheReadTokens), integerCell(stats.CacheWriteTokens), amountCell(stats.ConsumeAmount),
			amountCell(stats.RefundAmount), amountCell(stats.ConsumeAmount - stats.RefundAmount),
		})
		promptTokens += stats.PromptTokens
		completionTokens += stats.CompletionTokens
		totalTokens += stats.TotalTokens
		cacheReadTokens += stats.CacheReadTokens
		cacheWriteTokens += stats.CacheWriteTokens
		consumeCount += stats.ConsumeCount
		refundCount += stats.RefundCount
		consumeAmount += stats.ConsumeAmount
		refundAmount += stats.RefundAmount
	}
	rows = append(rows, []xlsxCell{
		totalTextCell("合计"), totalIntegerCell(consumeCount), totalIntegerCell(refundCount),
		totalIntegerCell(promptTokens), totalIntegerCell(completionTokens), totalIntegerCell(totalTokens),
		totalIntegerCell(cacheReadTokens), totalIntegerCell(cacheWriteTokens), totalAmountCell(consumeAmount),
		totalAmountCell(refundAmount), totalAmountCell(consumeAmount - refundAmount),
	})
	return xlsxSheet{Name: "令牌汇总", Columns: columns, Rows: rows}
}

func consumptionDetailColumns() []xlsxColumn {
	return []xlsxColumn{
		{Header: "序号", Width: 8}, {Header: "用户ID", Width: 10}, {Header: "用户名", Width: 16},
		{Header: "消费时间", Width: 20}, {Header: "模型名称", Width: 26}, {Header: "输入Token", Width: 13},
		{Header: "输出Token", Width: 13}, {Header: "总Token", Width: 13}, {Header: "缓存读Token", Width: 14},
		{Header: "缓存写Token", Width: 14}, {Header: "消费额度", Width: 13}, {Header: "消费金额($)", Width: 15},
		{Header: "用时/首字", Width: 14}, {Header: "令牌名称", Width: 20}, {Header: "日志类型", Width: 12},
		{Header: "分组", Width: 14}, {Header: "IP", Width: 16}, {Header: "日志详情", Width: 48},
	}
}

func consumptionDetailRow(record consumptionExportRecord) []xlsxCell {
	return []xlsxCell{
		integerCell(record.Sequence), integerCell(int64(record.UserID)), textCell(record.Username), textCell(record.CreatedAt),
		textCell(record.ModelName), integerCell(record.PromptTokens), integerCell(record.CompletionTokens), integerCell(record.TotalTokens),
		integerCell(record.CacheReadTokens), integerCell(record.CacheWriteTokens), integerCell(record.Quota), amountCell(record.Amount),
		textCell(record.Duration), textCell(record.TokenName), textCell(record.LogType), textCell(record.Group), textCell(record.IP), textCell(record.Detail),
	}
}

func (totals *consumptionDetailTotals) Add(record consumptionExportRecord) {
	totals.PromptTokens += record.PromptTokens
	totals.CompletionTokens += record.CompletionTokens
	totals.TotalTokens += record.TotalTokens
	totals.CacheReadTokens += record.CacheReadTokens
	totals.CacheWriteTokens += record.CacheWriteTokens
	totals.Quota += record.Quota
	totals.Amount += record.Amount
}

func (totals consumptionDetailTotals) Row() []xlsxCell {
	return []xlsxCell{
		totalTextCell("合计"), totalTextCell(""), totalTextCell(""), totalTextCell(""), totalTextCell(""),
		totalIntegerCell(totals.PromptTokens), totalIntegerCell(totals.CompletionTokens), totalIntegerCell(totals.TotalTokens),
		totalIntegerCell(totals.CacheReadTokens), totalIntegerCell(totals.CacheWriteTokens), totalIntegerCell(totals.Quota),
		totalAmountCell(totals.Amount), totalTextCell(""), totalTextCell(""), totalTextCell(""), totalTextCell(""),
		totalTextCell(""), totalTextCell(""),
	}
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

var consumptionExportTempCleanupOnce sync.Once

func deleteExpiredConsumptionExportTempFiles(directory string, cutoff time.Time) (int, error) {
	paths := make([]string, 0)
	for _, pattern := range []string{consumptionExportFilePattern, consumptionExportRowsPattern} {
		matches, err := filepath.Glob(filepath.Join(directory, pattern))
		if err != nil {
			return 0, err
		}
		paths = append(paths, matches...)
	}
	deleted := 0
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return deleted, err
		}
		if info.IsDir() || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func StartConsumptionExportTempCleanup() {
	consumptionExportTempCleanupOnce.Do(func() {
		gopool.Go(func() {
			cleanup := func() {
				if _, err := deleteExpiredConsumptionExportTempFiles(os.TempDir(), time.Now().Add(-consumptionExportTempFileTTL)); err != nil {
					logger.LogWarn(context.Background(), fmt.Sprintf("temporary consumption export cleanup failed: %v", err))
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
