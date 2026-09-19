package service

import (
	"archive/zip"
	"bytes"
	"context"
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

// renderConsumptionWorksheet writes a sheet out and asserts the result is
// well-formed XML, which every worksheet the exporter produces must be for
// Excel to open the workbook at all.
func renderConsumptionWorksheet(t *testing.T, sheet xlsxSheet) string {
	t.Helper()
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
	return worksheet.String()
}

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

func TestDeleteExpiredConsumptionExportTempFilesRemovesStaleWorkbooksAndRowFiles(t *testing.T) {
	directory := t.TempDir()
	staleWorkbook, err := os.CreateTemp(directory, consumptionExportFilePattern)
	require.NoError(t, err)
	require.NoError(t, staleWorkbook.Close())
	staleRows, err := os.CreateTemp(directory, consumptionExportRowsPattern)
	require.NoError(t, err)
	require.NoError(t, staleRows.Close())
	recent, err := os.CreateTemp(directory, consumptionExportFilePattern)
	require.NoError(t, err)
	require.NoError(t, recent.Close())
	unrelated, err := os.CreateTemp(directory, "other-*.xlsx")
	require.NoError(t, err)
	require.NoError(t, unrelated.Close())

	now := time.Now()
	for _, stale := range []string{staleWorkbook.Name(), staleRows.Name()} {
		require.NoError(t, os.Chtimes(stale, now.Add(-2*time.Hour), now.Add(-2*time.Hour)))
	}
	deleted, err := deleteExpiredConsumptionExportTempFiles(directory, now.Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)
	_, err = os.Stat(staleWorkbook.Name())
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(staleRows.Name())
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(recent.Name())
	require.NoError(t, err)
	_, err = os.Stat(unrelated.Name())
	require.NoError(t, err)
}

// Records reach the exporter in log order, so rows for different tokens are
// interleaved. Each token worksheet must still receive only its own rows, and
// the 序号 column must stay continuous across worksheets in sheet order.
func TestConsumptionExportDetailNumbersTokenWorksheetsContinuously(t *testing.T) {
	detail, err := newConsumptionExportDetail(
		ConsumptionExportPayload{GroupByToken: true},
		map[string]int64{"alpha": 2, "beta": 1},
		3,
	)
	require.NoError(t, err)
	defer detail.close(context.Background())

	require.NoError(t, detail.write(consumptionExportRecord{TokenName: "alpha", ModelName: "gpt-5", Quota: 10, Amount: 0.5}))
	require.NoError(t, detail.write(consumptionExportRecord{TokenName: "beta", ModelName: "gpt-5", Quota: 20, Amount: 0.125}))
	require.NoError(t, detail.write(consumptionExportRecord{TokenName: "alpha", ModelName: "gpt-5", Quota: 30, Amount: 0.25}))

	sheets, err := detail.finish()
	require.NoError(t, err)
	require.Len(t, sheets, 2)
	assert.Equal(t, "alpha", sheets[0].Name)
	assert.Equal(t, "beta", sheets[1].Name)
	assert.Equal(t, 3, sheets[0].RowCount)
	assert.Equal(t, 2, sheets[1].RowCount)

	alpha := renderConsumptionWorksheet(t, sheets[0])
	assert.Contains(t, alpha, `dimension ref="A1:R4"`)
	assert.Contains(t, alpha, `<c r="A2" s="2"><v>1</v></c>`)
	assert.Contains(t, alpha, `<c r="A3" s="2"><v>2</v></c>`)
	assert.Contains(t, alpha, `<c r="K4" s="4"><v>40</v></c>`)
	assert.Contains(t, alpha, `<c r="L4" s="5"><v>0.75</v></c>`)

	beta := renderConsumptionWorksheet(t, sheets[1])
	assert.Contains(t, beta, `dimension ref="A1:R3"`)
	assert.Contains(t, beta, `<c r="A2" s="2"><v>3</v></c>`)
	assert.Contains(t, beta, `<c r="K3" s="4"><v>20</v></c>`)
}

func TestConsumptionExportDetailRejectsRecordsAddedAfterPlanning(t *testing.T) {
	detail, err := newConsumptionExportDetail(
		ConsumptionExportPayload{GroupByToken: true},
		map[string]int64{"alpha": 1},
		1,
	)
	require.NoError(t, err)
	defer detail.close(context.Background())

	require.NoError(t, detail.write(consumptionExportRecord{TokenName: "alpha"}))
	require.ErrorContains(t,
		detail.write(consumptionExportRecord{TokenName: "alpha"}),
		"expected 1 records, found at least 2",
	)
	require.ErrorContains(t,
		detail.write(consumptionExportRecord{TokenName: "beta"}),
		`API token "beta" appeared after the worksheets were planned`,
	)
}

func TestConsumptionExportDetailFinishRejectsMissingRecords(t *testing.T) {
	detail, err := newConsumptionExportDetail(ConsumptionExportPayload{}, nil, 2)
	require.NoError(t, err)
	defer detail.close(context.Background())

	require.NoError(t, detail.write(consumptionExportRecord{TokenName: "alpha"}))
	_, err = detail.finish()
	require.ErrorContains(t, err, `worksheet "消费清单" expected 2 records, found 1`)
}

func TestConsumptionExportDetailLayoutsPerPayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     ConsumptionExportPayload
		tokenCounts map[string]int64
		records     []consumptionExportRecord
		expected    []string
	}{
		{
			name:     "single detail sheet",
			records:  []consumptionExportRecord{{TokenName: "alpha"}},
			expected: []string{"消费清单"},
		},
		{
			name:        "one sheet per token",
			payload:     ConsumptionExportPayload{GroupByToken: true},
			tokenCounts: map[string]int64{"beta": 1, "alpha": 1},
			records:     []consumptionExportRecord{{TokenName: "beta"}, {TokenName: "alpha"}},
			expected:    []string{"alpha", "beta"},
		},
		{
			name:     "grouped export without tokens",
			payload:  ConsumptionExportPayload{GroupByToken: true},
			expected: []string{"消费清单"},
		},
		{
			// The single pass still visits every log row to build the
			// summaries, so detail records must be dropped rather than fail.
			name:     "daily summary keeps no detail sheets",
			payload:  ConsumptionExportPayload{DailySummary: true, GroupByToken: true},
			records:  []consumptionExportRecord{{TokenName: "alpha"}},
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			detail, err := newConsumptionExportDetail(test.payload, test.tokenCounts, int64(len(test.records)))
			require.NoError(t, err)
			defer detail.close(context.Background())

			for _, record := range test.records {
				require.NoError(t, detail.write(record))
			}
			sheets, err := detail.finish()
			require.NoError(t, err)

			names := make([]string, 0, len(sheets))
			for _, sheet := range sheets {
				names = append(names, sheet.Name)
			}
			assert.Equal(t, test.expected, names)
		})
	}
}

func TestBuildTokenSummarySheetAggregatesTokensCountsAndAmounts(t *testing.T) {
	sheet := buildTokenSummarySheet(map[string]*consumptionTokenStats{
		"": {
			PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12, CacheReadTokens: 3,
			ConsumeCount: 1, ConsumeAmount: 0.5,
		},
		"api-default": {
			TokenName: "api-default", PromptTokens: 100, CompletionTokens: 25, TotalTokens: 125,
			CacheReadTokens: 20, CacheWriteTokens: 5, ConsumeCount: 2, ConsumeAmount: 5.5,
			RefundCount: 1, RefundAmount: 0.25,
		},
	})

	content := renderConsumptionWorksheet(t, sheet)
	assert.Contains(t, content, `dimension ref="A1:K4"`)
	assert.Contains(t, content, "未命名令牌")
	assert.Contains(t, content, "api-default")
	assert.Contains(t, content, `<c r="B4" s="4"><v>3</v></c>`)
	assert.Contains(t, content, `<c r="C4" s="4"><v>1</v></c>`)
	assert.Contains(t, content, `<c r="F4" s="4"><v>137</v></c>`)
	assert.Contains(t, content, `<c r="K3" s="3"><v>5.25</v></c>`)
	assert.Contains(t, content, `<c r="K4" s="5"><v>5.75</v></c>`)
}

func TestConsumptionExportWorkbookIncludesTokenDetailsAndModelSummaries(t *testing.T) {
	modelStats := map[string]*consumptionExportStats{
		"gpt-5": {ModelName: "gpt-5", ConsumeCount: 1, ConsumeAmount: 0.001, RefundCount: 1, RefundAmount: 0.0002},
	}
	tokenStats := map[string]*consumptionTokenStats{
		"default": {
			TokenName: "default", PromptTokens: 100, CompletionTokens: 25, TotalTokens: 125,
			CacheReadTokens: 10, CacheWriteTokens: 5, ConsumeCount: 1, ConsumeAmount: 0.001,
			RefundCount: 1, RefundAmount: 0.0002,
		},
	}

	detail, err := newConsumptionExportDetail(ConsumptionExportPayload{}, nil, 2)
	require.NoError(t, err)
	defer detail.close(context.Background())
	require.NoError(t, detail.write(consumptionExportRecord{
		UserID: 7, Username: "demo", CreatedAt: "2026-08-09 12:00:00", ModelName: "gpt-5",
		PromptTokens: 100, CompletionTokens: 25, TotalTokens: 125, CacheReadTokens: 10, CacheWriteTokens: 5,
		Quota: 500, Amount: 0.001, Duration: "1.0s / 0.2s", TokenName: "default", LogType: "消费", Group: "default",
	}))
	require.NoError(t, detail.write(consumptionExportRecord{
		UserID: 7, Username: "demo", CreatedAt: "2026-08-09 12:01:00", ModelName: "gpt-5",
		Quota: -100, Amount: -0.0002, TokenName: "default", LogType: "退款", Group: "default",
	}))
	detailSheets, err := detail.finish()
	require.NoError(t, err)

	sheets := append([]xlsxSheet{buildTokenSummarySheet(tokenStats)}, detailSheets...)
	sheets = append(sheets, buildModelSummarySheet(modelStats))
	workbook, err := buildXLSX(sheets)
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

	assert.Contains(t, files["xl/workbook.xml"], `name="令牌汇总"`)
	assert.Contains(t, files["xl/workbook.xml"], `name="消费清单"`)
	assert.Contains(t, files["xl/workbook.xml"], `name="模型统计"`)
	assert.Contains(t, files["xl/worksheets/sheet1.xml"], "令牌名称")
	assert.Contains(t, files["xl/worksheets/sheet1.xml"], "default")
	assert.Contains(t, files["xl/worksheets/sheet1.xml"], "0.0008")
	assert.Contains(t, files["xl/worksheets/sheet2.xml"], "输入Token")
	assert.Contains(t, files["xl/worksheets/sheet2.xml"], "gpt-5")
	assert.Contains(t, files["xl/worksheets/sheet2.xml"], "合计")
	assert.Contains(t, files["xl/worksheets/sheet3.xml"], "退款记录数")
	assert.Contains(t, files["xl/worksheets/sheet3.xml"], "0.0008")
}

func TestConsumptionRecordFromRefundLogUsesNegativeNetAmountsAndDetailedTokens(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })

	log := &model.Log{
		UserId: 7, Type: model.LogTypeRefund, Quota: 500, PromptTokens: 90, CompletionTokens: 25,
		Other: `{"input_tokens_total":100,"cache_tokens":10,"cache_creation_tokens_5m":3,"cache_creation_tokens_1h":2}`,
	}
	record := consumptionRecordFromLog(log, time.UTC)

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
