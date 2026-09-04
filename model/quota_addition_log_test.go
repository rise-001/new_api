package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetQuotaAdditionLogsReturnsTargetUsersAcrossAuditFormats(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
	})

	users := []*User{
		{Id: 1, Username: "admin", Password: "unused", Role: common.RoleAdminUser},
		{Id: 2, Username: "001", Password: "unused", Role: common.RoleCommonUser},
		{Id: 3, Username: "legacy-user", Password: "unused", Role: common.RoleCommonUser},
	}
	for _, user := range users {
		require.NoError(t, db.Create(user).Error)
	}

	logs := []*Log{
		{
			UserId: 1, Username: "admin", CreatedAt: 100, Type: LogTypeManage,
			Content: "Increased user quota by $1.000000 quota",
			Other: common.MapToJsonStr(map[string]interface{}{
				"op": buildOpField("user.quota_add", map[string]interface{}{
					"quota": "$1.000000 quota", "target_user_id": 2,
				}),
			}),
		},
		{
			UserId: 3, Username: "legacy-user", CreatedAt: 101, Type: LogTypeManage,
			Content: "管理员增加用户额度 $2.000000 quota",
		},
		{
			UserId: 1, Username: "admin", CreatedAt: 102, Type: LogTypeManage,
			Content: "Increased user quota by $3.000000 quota",
			Other: common.MapToJsonStr(map[string]interface{}{
				"op": buildOpField("user.quota_add", map[string]interface{}{
					"quota": "$3.000000 quota",
				}),
			}),
		},
		{
			UserId: 1, Username: "admin", CreatedAt: 103, Type: LogTypeManage,
			Content: "Increased user quota by $4.000000 quota",
			Other: common.MapToJsonStr(map[string]interface{}{
				"op": buildOpField("user.quota_add", map[string]interface{}{
					"quota": "$4.000000 quota", "target_user_id": 2,
				}),
			}),
		},
		{UserId: 1, Username: "admin", CreatedAt: 102, Type: LogTypeManage, Content: "Updated user 001"},
	}
	for _, log := range logs {
		require.NoError(t, db.Create(log).Error)
	}

	records, total, err := GetQuotaAdditionLogs(100, 103, "", 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, records, 3)
	assert.Equal(t, "admin", records[0].Username)
	assert.Equal(t, "$3.000000 quota", records[0].Amount)
	assert.Equal(t, "legacy-user", records[1].Username)
	assert.Equal(t, "$2.000000 quota", records[1].Amount)
	assert.Equal(t, 2, records[2].UserId)
	assert.Equal(t, "001", records[2].Username)
	assert.Equal(t, "$1.000000 quota", records[2].Amount)

	targetRecords, targetTotal, err := GetQuotaAdditionLogs(0, 0, "001", 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 2, targetTotal)
	require.Len(t, targetRecords, 2)
	assert.Equal(t, []int64{103, 100}, []int64{targetRecords[0].CreatedAt, targetRecords[1].CreatedAt})

	selfRecords, selfTotal, err := GetQuotaAdditionLogs(0, 0, "admin", 0, 20)
	require.NoError(t, err)
	assert.EqualValues(t, 1, selfTotal)
	require.Len(t, selfRecords, 1)
	assert.Equal(t, int64(102), selfRecords[0].CreatedAt)
}
