package model

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumptionExportFileHonorsOwnerAndExpiration(t *testing.T) {
	taskID, err := GenerateSystemTaskID()
	require.NoError(t, err)
	now := common.GetTimestamp()
	file := &ConsumptionExportFile{
		TaskID: taskID, UserID: 101, FileName: "consumption.xlsx", Content: []byte("workbook"), ExpiresAt: now + 60,
	}
	require.NoError(t, SaveConsumptionExportFile(file))
	t.Cleanup(func() {
		_ = DB.Where("task_id = ?", taskID).Delete(&ConsumptionExportFile{}).Error
	})

	owned, err := GetConsumptionExportFile(taskID, 101, now)
	require.NoError(t, err)
	require.NotNil(t, owned)
	assert.Equal(t, []byte("workbook"), owned.Content)

	notOwned, err := GetConsumptionExportFile(taskID, 202, now)
	require.NoError(t, err)
	assert.Nil(t, notOwned)

	expired, err := GetConsumptionExportFile(taskID, 101, now+60)
	require.NoError(t, err)
	assert.Nil(t, expired)

	deleted, err := DeleteExpiredConsumptionExportFiles(context.Background(), now+60)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, int64(1))
}
