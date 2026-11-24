package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferRecordsAPICombinesModeUserAndPageSize(t *testing.T) {
	setupManageUserTestDB(t)
	require.NoError(t, model.LOG_DB.Create(&[]model.Log{
		{Id: 1, UserId: 2, Username: "alice", CreatedAt: 10, Type: model.LogTypeTopup, Content: "通过兑换码充值 $1，兑换码ID 1"},
		{Id: 2, UserId: 2, Username: "alice", CreatedAt: 20, Type: model.LogTypeTopup, Content: "通过兑换码充值 $2，兑换码ID 2"},
		{Id: 3, UserId: 3, Username: "bob", CreatedAt: 30, Type: model.LogTypeTopup, Content: "通过兑换码充值 $3，兑换码ID 3"},
		{Id: 4, UserId: 2, Username: "alice", CreatedAt: 40, Type: model.LogTypeTopup, Content: "使用在线充值成功，充值金额: $4，支付金额：4.00"},
	}).Error)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/transfer?username=alice&mode=redemption&page_size=1&p=2", nil)
	GetQuotaAdditionLogs(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Items    []model.QuotaAdditionLog `json:"items"`
			Total    int                      `json:"total"`
			PageSize int                      `json:"page_size"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, 2, response.Data.Total)
	assert.Equal(t, 1, response.Data.PageSize)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, 1, response.Data.Items[0].Id)
	assert.Equal(t, "redemption", response.Data.Items[0].Mode)
}

func TestTransferRecordsFromCurrentLogWriters(t *testing.T) {
	db := setupManageUserTestDB(t)
	u := model.User{Id: 2, Username: "alice", AffCode: "alice"}
	require.NoError(t, db.Create(&u).Error)
	model.RecordTopupLog(u.Id, common.NewMessage("Online top-up succeeded, amount added: {{quota}}, payment amount: {{amount}}", map[string]any{"quota": "$3.50", "amount": "2.00"}), "", "", "")
	model.RecordLogWithAdminInfo(u.Id, model.LogTypeTopup, "Increased user quota by 500000", nil, &model.AuditOperation{Action: "user.quota_add", Params: model.AuditFields{"quota": 500000, "target_user_id": u.Id}})
	model.RecordLogWithAdminInfo(u.Id, model.LogTypeTopup, "Decreased user quota by 250000", nil, &model.AuditOperation{Action: "user.quota_subtract", Params: model.AuditFields{"quota": 250000}})
	model.RecordLogWithAdminInfo(u.Id, model.LogTypeTopup, "Overrode user quota from 250000 to 0", nil, &model.AuditOperation{Action: "user.quota_override", Params: model.AuditFields{"from": 250000, "to": 0}})
	model.RecordLog(u.Id, model.LogTypeTopup, common.NewMessage("Topped up {{quota}} with a redemption code, code ID {{id}}", map[string]any{"quota": "$2", "id": 19}))
	model.RecordLog(u.Id, model.LogTypeTopup, common.NewMessage("{{provider}} top-up succeeded, amount added: {{quota}}, payment amount: {{amount}}", map[string]any{"provider": "Creem", "quota": 125000, "amount": "8.00"}))
	model.RecordLog(u.Id, model.LogTypeTopup, common.NewMessage("Subscription purchased, plan: {{plan}}, payment amount: {{amount}}, payment method: {{method}}", map[string]any{"plan": "Pro", "amount": "9.00", "method": "card"}))
	model.RecordLog(u.Id, model.LogTypeTopup, common.NewMessage("Subscription purchased with balance, plan: Pro, payment amount: 1, quota deducted: 500000"))

	recharge := "topup:$3.50"
	creem := "topup:" + logger.FormatQuota(125000)
	add := "add:" + logger.FormatQuota(500000)
	subtract := "subtract:" + logger.FormatQuota(250000)
	override := "override:" + logger.FormatQuota(0)
	redemption := "redemption:$2"
	for _, tc := range []struct {
		mode string
		want []string
	}{
		{"", []string{recharge, creem, add, subtract, override, redemption}},
		{"topup", []string{recharge, creem}},
		{"add", []string{add}},
		{"subtract", []string{subtract}},
		{"override", []string{override}},
		{"redemption", []string{redemption}},
	} {
		records, total, err := model.GetQuotaAdditionLogs(0, 0, u.Username, 0, 20, tc.mode)
		require.NoError(t, err)
		assert.EqualValues(t, len(tc.want), total, "filter %q", tc.mode)
		items := make([]string, 0, len(records))
		for _, record := range records {
			items = append(items, record.Mode+":"+record.Amount)
		}
		assert.ElementsMatch(t, tc.want, items, "filter %q", tc.mode)
	}
}

func TestTransferRecordsAPIRejectsUnknownType(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/transfer?mode=unknown", nil)
	GetQuotaAdditionLogs(c)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
