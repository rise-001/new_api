package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{Id: 1, Username: "admin", Password: "unused", AffCode: "admin", Role: common.RoleAdminUser},
		{Id: 2, Username: "001", Password: "unused", AffCode: "001", Role: common.RoleCommonUser},
		{Id: 3, Username: "legacy-user", Password: "unused", AffCode: "legacy", Role: common.RoleCommonUser},
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

	records, total, err := GetQuotaAdditionLogs(100, 103, "", 0, 20, "")
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

	targetRecords, targetTotal, err := GetQuotaAdditionLogs(0, 0, "001", 0, 20, "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, targetTotal)
	require.Len(t, targetRecords, 2)
	assert.Equal(t, []int64{103, 100}, []int64{targetRecords[0].CreatedAt, targetRecords[1].CreatedAt})

	selfRecords, selfTotal, err := GetQuotaAdditionLogs(0, 0, "admin", 0, 20, "")
	require.NoError(t, err)
	assert.EqualValues(t, 1, selfTotal)
	require.Len(t, selfRecords, 1)
	assert.Equal(t, int64(102), selfRecords[0].CreatedAt)
}

func TestTransferRecordsIncludeTopupsAndRedemptionsByDefault(t *testing.T) {
	truncateTables(t)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{Id: 1, UserId: 2, Username: "alice", Type: LogTypeTopup, CreatedAt: 100,
			Content: "使用在线充值成功，充值金额: $1.000000 quota，支付金额：1.000000"},
		{Id: 2, UserId: 2, Username: "alice", Type: LogTypeTopup, CreatedAt: 200,
			Content: "通过兑换码充值 $2.000000 quota，兑换码ID 42"},
	}).Error)

	records, total, err := GetQuotaAdditionLogs(0, 0, "", 0, 20, "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, records, 2)
	assert.Equal(t, "redemption", records[0].Mode)
	assert.Equal(t, "$2.000000 quota", records[0].Amount)
	assert.Equal(t, "topup", records[1].Mode)
	assert.Equal(t, "$1.000000 quota", records[1].Amount)
}

func TestTransferRecordsFilterTypesAndTargetUsersBeforePagination(t *testing.T) {
	truncateTables(t)
	// The production log database may be separate from the user database.
	previousLogDB := LOG_DB
	logDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, logDB.AutoMigrate(&Log{}))
	LOG_DB = logDB
	t.Cleanup(func() {
		LOG_DB = previousLogDB
		sqlDB, err := logDB.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, DB.Create(&[]User{
		{Id: 1, Username: "admin", Password: "unused", AffCode: "admin"},
		{Id: 2, Username: "alice", Password: "unused", AffCode: "alice"},
		{Id: 20, Username: "bob", Password: "unused", AffCode: "bob"},
	}).Error)
	require.NoError(t, logDB.Create(&[]Log{
		{Id: 1, UserId: 2, Username: "alice", CreatedAt: 10, Type: LogTypeTopup,
			Content: "Waffo充值成功，充值额度: $1.000000，支付金额: 1.00"},
		{Id: 2, UserId: 1, Username: "admin", CreatedAt: 20, Type: LogTypeManage,
			Content: "Translated audit message", Other: `{"op":{"action":"user.quota_add","params":{"quota":"$2","target_user_id":2}}}`},
		{Id: 3, UserId: 1, Username: "admin", CreatedAt: 30, Type: LogTypeManage,
			Other: "{\n\t\"op\": {\"action\": \"user.quota_subtract\", \"params\": {\"target_user_id\": 2, \"quota\": \"$3\"}}}"},
		{Id: 4, UserId: 1, Username: "admin", CreatedAt: 40, Type: LogTypeManage,
			Other: `{"op":{"action":"user.quota_override","params":{"from":"$10","to":"$0","target_user_id":2}}}`},
		{Id: 5, UserId: 2, Username: "alice", CreatedAt: 50, Type: LogTypeTopup,
			Content: "通过兑换码充值 $5.000000，兑换码ID 123"},
		{Id: 6, UserId: 1, Username: "admin", CreatedAt: 60, Type: LogTypeManage,
			Other: `{"op":{"action":"user.quota_add","params":{"quota":"$6","target_user_id":20}}}`},
		{Id: 7, UserId: 1, Username: "admin", CreatedAt: 70, Type: LogTypeTopup,
			Content: "管理员补单成功，充值金额: $7.000000，支付金额：7.00"},
		{Id: 8, UserId: 99, Username: "archived", CreatedAt: 80, Type: LogTypeTopup,
			Content: "使用在线充值成功，充值金额: $8.000000，支付金额：8.00"},
		{Id: 9, UserId: 2, Username: "alice", CreatedAt: 90, Type: LogTypeConsume, Content: "API usage"},
		{Id: 10, UserId: 2, Username: "alice", CreatedAt: 90, Type: LogTypeManage, Content: "Updated user alice"},
		{Id: 11, UserId: 2, Username: "alice", CreatedAt: 90, Type: LogTypeTopup, Content: "订阅购买成功，套餐: Pro，支付金额: 10.00"},
		{Id: 12, UserId: 2, Username: "alice", CreatedAt: 90, Type: LogTypeTopup, Content: "使用余额购买订阅成功，套餐: Pro，扣除额度: 1000"},
		{Id: 13, UserId: 2, Username: "alice", CreatedAt: 35, Type: LogTypeManage, Content: "管理员增加用户额度 $13", Other: "invalid-json"},
		{Id: 14, UserId: 2, Username: "alice", CreatedAt: 40, Type: LogTypeManage, Content: "Overrode user quota from $10 to $0"},
	}).Error)

	for _, tt := range []struct {
		name, username, mode string
		start, end           int64
		offset, limit        int
		total                int64
		ids                  []int
	}{
		{name: "all history", limit: 20, total: 10, ids: []int{8, 7, 6, 5, 14, 4, 13, 3, 2, 1}},
		{name: "topups", mode: "topup", limit: 20, total: 3, ids: []int{8, 7, 1}},
		{name: "additions", mode: "add", limit: 20, total: 3, ids: []int{6, 13, 2}},
		{name: "reductions", mode: "subtract", limit: 20, total: 1, ids: []int{3}},
		{name: "overrides", mode: "override", limit: 20, total: 2, ids: []int{14, 4}},
		{name: "redemptions", mode: "redemption", limit: 20, total: 1, ids: []int{5}},
		{name: "target user across all modes", username: "alice", limit: 20, total: 7, ids: []int{5, 14, 4, 13, 3, 2, 1}},
		{name: "operator excludes changes to others", username: "admin", limit: 20, total: 1, ids: []int{7}},
		{name: "target ID does not match numeric prefix", username: "bob", limit: 20, total: 1, ids: []int{6}},
		{name: "deleted user history", username: "archived", limit: 20, total: 1, ids: []int{8}},
		{name: "unknown user", username: "missing", limit: 20, ids: []int{}},
		{name: "combined user and mode", username: "alice", mode: "subtract", limit: 20, total: 1, ids: []int{3}},
		{name: "filtered second page", username: "alice", start: 20, end: 50, offset: 2, limit: 2, total: 5, ids: []int{13, 3}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			records, total, err := GetQuotaAdditionLogs(tt.start, tt.end, tt.username, tt.offset, tt.limit, tt.mode)
			require.NoError(t, err)
			assert.Equal(t, tt.total, total)
			ids := make([]int, 0, len(records))
			for _, record := range records {
				ids = append(ids, record.Id)
				if tt.username != "" {
					assert.Equal(t, tt.username, record.Username)
				}
				if record.Mode == "override" {
					assert.Equal(t, "$0", record.Amount, "override shows the new quota, including zero")
				}
			}
			assert.Equal(t, tt.ids, ids)
		})
	}
}
