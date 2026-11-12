package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferRecordsAPICombinesModeUserAndPageSize(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.Create(&[]model.Log{
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

func TestTransferRecordsAPIRejectsUnknownType(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/transfer?mode=unknown", nil)
	GetQuotaAdditionLogs(c)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}
