package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveLegacyConsumptionExportStorageDropsFilesAndTaskRows(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Exec("CREATE TABLE consumption_export_files (id INTEGER PRIMARY KEY)").Error)

	legacyActiveKey := "consumption_export:101"
	legacyTask := &SystemTask{
		TaskID:    "legacy_consumption_export",
		Type:      "consumption_export",
		Status:    SystemTaskStatusRunning,
		ActiveKey: &legacyActiveKey,
		LockedBy:  "legacy-runner",
	}
	require.NoError(t, DB.Create(legacyTask).Error)
	require.NoError(t, DB.Create(&SystemTaskLock{
		Type:        "consumption_export",
		TaskID:      legacyTask.TaskID,
		LockedBy:    legacyTask.LockedBy,
		LockedUntil: 100,
	}).Error)
	require.NoError(t, DB.Create(&SystemTask{
		TaskID: "retained_task",
		Type:   SystemTaskTypeLogCleanup,
		Status: SystemTaskStatusSucceeded,
	}).Error)

	require.NoError(t, removeLegacyConsumptionExportStorage())
	assert.False(t, DB.Migrator().HasTable("consumption_export_files"))

	var legacyTaskCount int64
	require.NoError(t, DB.Model(&SystemTask{}).Where("type = ?", "consumption_export").Count(&legacyTaskCount).Error)
	assert.Zero(t, legacyTaskCount)
	var legacyLockCount int64
	require.NoError(t, DB.Model(&SystemTaskLock{}).Where("type = ?", "consumption_export").Count(&legacyLockCount).Error)
	assert.Zero(t, legacyLockCount)
	var retainedTaskCount int64
	require.NoError(t, DB.Model(&SystemTask{}).Where("task_id = ?", "retained_task").Count(&retainedTaskCount).Error)
	assert.Equal(t, int64(1), retainedTaskCount)
}
