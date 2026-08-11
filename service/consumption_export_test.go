package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumptionExportPayloadValidateRejectsInvalidRanges(t *testing.T) {
	tests := []struct {
		name    string
		payload ConsumptionExportPayload
	}{
		{name: "missing start", payload: ConsumptionExportPayload{EndTimestamp: 20}},
		{name: "reversed range", payload: ConsumptionExportPayload{StartTimestamp: 20, EndTimestamp: 10}},
		{name: "invalid token", payload: ConsumptionExportPayload{StartTimestamp: 1, EndTimestamp: 10, TokenID: -1}},
		{name: "invalid timezone", payload: ConsumptionExportPayload{StartTimestamp: 1, EndTimestamp: 10, TimezoneOffset: 721}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, test.payload.Validate())
		})
	}
}

func TestConsumptionExportPayloadValidateEnforces31DayRange(t *testing.T) {
	const maxRangeSeconds = int64(31 * 24 * 60 * 60)

	withinLimit := ConsumptionExportPayload{
		StartTimestamp: 1,
		EndTimestamp:   1 + maxRangeSeconds,
	}
	require.NoError(t, withinLimit.Validate())

	exceedsLimit := ConsumptionExportPayload{
		StartTimestamp: 1,
		EndTimestamp:   1 + maxRangeSeconds + 1,
	}
	require.EqualError(t, exceedsLimit.Validate(), "export time range cannot exceed 31 days")
}

func TestConsumptionExportDownloadCloseRemovesTemporaryFile(t *testing.T) {
	temporaryFile, err := os.CreateTemp(t.TempDir(), "consumption-*.xlsx")
	require.NoError(t, err)
	userID := 42
	activeConsumptionExports.Store(userID, struct{}{})
	download := &ConsumptionExportDownload{File: temporaryFile, userID: userID}

	require.NoError(t, download.Close())
	_, err = os.Stat(temporaryFile.Name())
	require.ErrorIs(t, err, os.ErrNotExist)
	_, active := activeConsumptionExports.Load(userID)
	assert.False(t, active)
	require.NoError(t, download.Close())
}

func TestDeleteExpiredConsumptionExportTempFilesOnlyRemovesStaleExports(t *testing.T) {
	directory := t.TempDir()
	stale, err := os.CreateTemp(directory, "new-api-consumption-*.xlsx")
	require.NoError(t, err)
	require.NoError(t, stale.Close())
	recent, err := os.CreateTemp(directory, "new-api-consumption-*.xlsx")
	require.NoError(t, err)
	require.NoError(t, recent.Close())
	unrelated, err := os.CreateTemp(directory, "other-*.xlsx")
	require.NoError(t, err)
	require.NoError(t, unrelated.Close())

	now := time.Now()
	require.NoError(t, os.Chtimes(stale.Name(), now.Add(-2*time.Hour), now.Add(-2*time.Hour)))
	deleted, err := deleteExpiredConsumptionExportTempFiles(directory, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	_, err = os.Stat(stale.Name())
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(recent.Name())
	require.NoError(t, err)
	_, err = os.Stat(unrelated.Name())
	require.NoError(t, err)
}

func TestBuildStreamingConsumptionDetailSheetWritesRowsAndTotals(t *testing.T) {
	records := []consumptionExportRecord{
		{Sequence: 1, UserID: 7, ModelName: "gpt-5", PromptTokens: 100, CompletionTokens: 25, TotalTokens: 125, Quota: 500, Amount: 0.001},
		{Sequence: 2, UserID: 7, ModelName: "gpt-5", PromptTokens: 20, CompletionTokens: 5, TotalTokens: 25, Quota: -100, Amount: -0.0002},
	}
	sheet := buildStreamingConsumptionDetailSheet("消费清单", int64(len(records)), func(write func(consumptionExportRecord) error) error {
		for _, record := range records {
			if err := write(record); err != nil {
				return err
			}
		}
		return nil
	})

	var worksheet bytes.Buffer
	require.NoError(t, writeWorksheetXML(&worksheet, sheet))
	decoder := xml.NewDecoder(bytes.NewReader(worksheet.Bytes()))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}
	content := worksheet.String()
	assert.Contains(t, content, `dimension ref="A1:R4"`)
	assert.Contains(t, content, "gpt-5")
	assert.Contains(t, content, "合计")
	assert.Contains(t, content, "0.0008")
}

func TestWriteWorksheetXMLRejectsUnexpectedStreamingRowCount(t *testing.T) {
	sheet := xlsxSheet{
		Name:     "消费清单",
		Columns:  []xlsxColumn{{Header: "序号", Width: 8}},
		RowCount: 2,
		WriteRows: func(write func([]xlsxCell) error) error {
			return write([]xlsxCell{integerCell(1)})
		},
	}

	err := writeWorksheetXML(io.Discard, sheet)
	require.EqualError(t, err, `worksheet "消费清单" wrote 1 rows, expected 2`)
}

func TestStreamingConsumptionExportWorkbookIncludesDetailsTotalsAndModelSummary(t *testing.T) {
	records := []consumptionExportRecord{
		{
			Sequence: 1, UserID: 7, Username: "demo", CreatedAt: "2026-08-09 12:00:00", ModelName: "gpt-5",
			PromptTokens: 100, CompletionTokens: 25, TotalTokens: 125, CacheReadTokens: 10, CacheWriteTokens: 5,
			Quota: 500, Amount: 0.001, Duration: "1.0s / 0.2s", TokenName: "default", LogType: "消费", Group: "default",
		},
		{
			Sequence: 2, UserID: 7, Username: "demo", CreatedAt: "2026-08-09 12:01:00", ModelName: "gpt-5",
			Quota: -100, Amount: -0.0002, TokenName: "default", LogType: "退款", Group: "default",
		},
	}
	modelStats := map[string]*consumptionExportStats{
		"gpt-5": {ModelName: "gpt-5", ConsumeCount: 1, ConsumeAmount: 0.001, RefundCount: 1, RefundAmount: 0.0002},
	}

	detailSheet := buildStreamingConsumptionDetailSheet("消费清单", int64(len(records)), func(write func(consumptionExportRecord) error) error {
		for _, record := range records {
			if err := write(record); err != nil {
				return err
			}
		}
		return nil
	})
	workbook, err := buildXLSX([]xlsxSheet{detailSheet, buildModelSummarySheet(modelStats)})
	require.NoError(t, err)
	require.NotEmpty(t, workbook)

	archive, err := zip.NewReader(bytes.NewReader(workbook), int64(len(workbook)))
	require.NoError(t, err)
	files := make(map[string]string, len(archive.File))
	for _, file := range archive.File {
		reader, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		files[file.Name] = string(content)
		if strings.HasSuffix(file.Name, ".xml") {
			decoder := xml.NewDecoder(bytes.NewReader(content))
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				require.NoError(t, err, file.Name)
			}
		}
	}

	assert.Contains(t, files["xl/workbook.xml"], `name="消费清单"`)
	assert.Contains(t, files["xl/workbook.xml"], `name="模型统计"`)
	assert.Contains(t, files["xl/worksheets/sheet1.xml"], "输入Token")
	assert.Contains(t, files["xl/worksheets/sheet1.xml"], "gpt-5")
	assert.Contains(t, files["xl/worksheets/sheet1.xml"], "合计")
	assert.Contains(t, files["xl/worksheets/sheet2.xml"], "退款记录数")
	assert.Contains(t, files["xl/worksheets/sheet2.xml"], "0.0008")
}

func TestConsumptionRecordFromRefundLogUsesNegativeNetAmountsAndDetailedTokens(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })

	log := &model.Log{
		UserId: 7, Type: model.LogTypeRefund, Quota: 500, PromptTokens: 90, CompletionTokens: 25,
		Other: `{"input_tokens_total":100,"cache_tokens":10,"cache_creation_tokens_5m":3,"cache_creation_tokens_1h":2}`,
	}
	record := consumptionRecordFromLog(log, 1, time.UTC)

	assert.Equal(t, int64(100), record.PromptTokens)
	assert.Equal(t, int64(125), record.TotalTokens)
	assert.Equal(t, int64(10), record.CacheReadTokens)
	assert.Equal(t, int64(5), record.CacheWriteTokens)
	assert.Equal(t, int64(-500), record.Quota)
	assert.InDelta(t, -0.001, record.Amount, 0.0000001)
	assert.Equal(t, "退款", record.LogType)
}

func TestUniqueSheetNameSanitizesExcelRestrictionsAndDuplicates(t *testing.T) {
	used := map[string]int{}

	assert.Equal(t, "daily-key", uniqueSheetName("'daily/key'", used))
	assert.Equal(t, "daily-key-2", uniqueSheetName("daily:key", used))
	assert.Equal(t, "未命名令牌", uniqueSheetName("''", used))
}
