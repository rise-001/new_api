package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProcessChannelErrorUsesSnapshotWithoutLeakingChannelMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousErrorLogEnabled := constant.ErrorLogEnabled

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, database.AutoMigrate(&model.User{}, &model.Log{}))
	model.DB, model.LOG_DB = database, database
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	constant.ErrorLogEnabled = true
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		constant.ErrorLogEnabled = previousErrorLogEnabled
		require.NoError(t, sqlDB.Close())
	})

	require.NoError(t, database.Create(&model.User{Id: 7, Username: "log-owner", Group: "default"}).Error)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("id", 7)
	ctx.Set("username", "log-owner")
	ctx.Set("token_name", "test-token")
	ctx.Set("token_id", 11)
	ctx.Set("original_model", "gpt-test")
	ctx.Set("group", "default")
	ctx.Set("channel_id", 202)
	ctx.Set("channel_name", "mutable-context-channel")
	ctx.Set("channel_type", 9)
	ctx.Set("use_channel", []string{"101"})
	common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Now().Add(-time.Second))

	channelSnapshot := types.ChannelError{
		ChannelId:   101,
		ChannelType: 1,
		ChannelName: "snapshot-channel",
		AutoBan:     false,
	}
	apiErr := types.NewOpenAIError(errors.New("upstream failed"), types.ErrorCodeBadResponseStatusCode, http.StatusBadGateway)

	processChannelError(ctx, channelSnapshot, apiErr, nil)

	var stored model.Log
	require.NoError(t, database.First(&stored).Error)
	assert.Equal(t, channelSnapshot.ChannelId, stored.ChannelId)
	storedOther, err := common.StrToMap(stored.Other)
	require.NoError(t, err)
	assert.Equal(t, float64(http.StatusBadGateway), storedOther["status_code"])
	for _, key := range []string{"channel_id", "channel_name", "channel_type"} {
		assert.NotContains(t, storedOther, key)
	}
	adminInfo, ok := storedOther["admin_info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"101"}, adminInfo["use_channel"])

	logs, total, err := model.GetUserLogs(7, model.LogFilter{Type: model.LogTypeError}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, channelSnapshot.ChannelId, logs[0].ChannelId)
	assert.Empty(t, logs[0].ChannelName)
	userOther, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, userOther, "admin_info")
	for _, key := range []string{"channel_id", "channel_name", "channel_type"} {
		assert.NotContains(t, userOther, key)
	}
}

func TestClientErrorMasking(t *testing.T) {
	require.NoError(t, i18n.Init())
	db, dialect := openTaskDialectDatabase(t, &model.Option{}, &model.SystemTask{}, &model.Log{}, &model.User{})
	oldDB, oldLog := model.DB, model.LOG_DB
	oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
	oldOptions, oldErrorLog := common.OptionMap, constant.ErrorLogEnabled
	oldRedis := common.RedisEnabled
	common.RedisEnabled = false
	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(dialect, dialect)
	common.OptionMap = map[string]string{}
	constant.ErrorLogEnabled = true
	const key = operation_setting.ClientErrorMaskOptionKey
	t.Cleanup(func() {
		patterns, err := operation_setting.CompileClientErrorPatterns(oldOptions[key])
		require.NoError(t, err)
		patterns.Apply()
		model.DB, model.LOG_DB = oldDB, oldLog
		common.SetDatabaseTypes(oldMainType, oldLogType)
		common.OptionMap, constant.ErrorLogEnabled = oldOptions, oldErrorLog
		common.RedisEnabled = oldRedis
	})
	const rules = "(?i)no available channel\r\n\n无可用渠道"
	require.NoError(t, model.UpdateOption(key, rules))
	require.EqualError(t, model.UpdateOption(key, "discard\n["), "Invalid regular expression on line 2")
	var saved model.Option
	require.NoError(t, db.First(&saved, &model.Option{Key: key}).Error)
	assert.Equal(t, rules, saved.Value)

	require.NoError(t, db.Create(&model.User{Id: 7, Username: "mask-test", Group: "default"}).Error)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("id", 7)
	ctx.Request.Header.Set("Accept-Language", "zh-CN")
	ctx.Set(common.RequestIdKey, "mask-test-id")
	upstream := types.WithOpenAIError(types.OpenAIError{Type: "upstream_error", Code: "unavailable", Message: "No available channel", Metadata: []byte(`{"detail":"provider detail"}`)}, http.StatusServiceUnavailable)
	service.RecordRelayErrorLog(ctx, 0, upstream, nil)
	want := "如果多次出现，请联系客服 (request id: mask-test-id)"
	client := service.ClientRelayError(ctx, upstream, true)
	assert.Equal(t, http.StatusServiceUnavailable, client.StatusCode)
	assert.Equal(t, want, client.ToOpenAIError().Message)
	assert.Empty(t, client.ToOpenAIError().Metadata)
	assert.Equal(t, want, client.ToClaudeError().Message)
	var stored model.Log
	require.NoError(t, db.First(&stored, &model.Log{RequestId: "mask-test-id"}).Error)
	assert.Contains(t, stored.Content, "No available channel")
	assert.NotContains(t, stored.Content, want)

	claude := types.WithClaudeError(types.ClaudeError{Type: "api_error", Message: "无可用渠道"}, http.StatusServiceUnavailable)
	assert.Equal(t, want, service.ClientRelayError(ctx, claude, false).ToClaudeError().Message)
	assert.Equal(t, "无可用渠道", claude.ToClaudeError().Message)
	message, masked := service.ClientErrorMessage(ctx, "another error", false)
	assert.False(t, masked)
	assert.Equal(t, "another error", message)

	require.NoError(t, model.UpdateOption(key, "(?i)model"))
	engine := gin.New()
	engine.Use(middleware.RequestId())
	engine.POST("/v1/chat/completions", middleware.Distribute())
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept-Language", "en")
	reply := httptest.NewRecorder()
	engine.ServeHTTP(reply, request)
	require.Equal(t, http.StatusBadRequest, reply.Code)
	requestID := reply.Header().Get(common.RequestIdKey)
	require.NotEmpty(t, requestID)
	var body struct{ Error types.OpenAIError }
	require.NoError(t, common.Unmarshal(reply.Body.Bytes(), &body))
	assert.Equal(t, "If this happens repeatedly, please contact support (request id: "+requestID+")", body.Error.Message)
	require.NoError(t, model.UpdateOption(key, ""))
	assert.Equal(t, "无可用渠道", service.ClientRelayError(ctx, claude, false).ToClaudeError().Message)
}

func TestProcessChannelErrorRecordsSelectedGroup(t *testing.T) {
	previousErrorLogEnabled := constant.ErrorLogEnabled
	previousAutomaticDisable := common.AutomaticDisableChannelEnabled
	constant.ErrorLogEnabled = true
	common.AutomaticDisableChannelEnabled = false
	t.Cleanup(func() {
		constant.ErrorLogEnabled = previousErrorLogEnabled
		common.AutomaticDisableChannelEnabled = previousAutomaticDisable
	})

	tests := []struct {
		name           string
		usingGroup     string
		selectedGroups []string
		wantGroups     []string
	}{
		{
			name:       "automatic selection records the actual group",
			usingGroup: "auto", selectedGroups: []string{"standard"}, wantGroups: []string{"standard"},
		},
		{
			name:       "cross-group retries record the group for each failed attempt",
			usingGroup: "auto", selectedGroups: []string{"standard", "fallback"}, wantGroups: []string{"standard", "fallback"},
		},
		{
			name:       "explicit group is preserved",
			usingGroup: "premium", selectedGroups: []string{""}, wantGroups: []string{"premium"},
		},
		{
			name:       "automatic selection without a resolved group preserves auto",
			usingGroup: "auto", selectedGroups: []string{""}, wantGroups: []string{"auto"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupManageUserTestDB(t)
			user := model.User{Username: "relay-error-user", Group: "account-group"}
			require.NoError(t, db.Create(&user).Error)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			common.SetContextKey(c, constant.ContextKeyUserId, user.Id)
			common.SetContextKey(c, constant.ContextKeyUserName, user.Username)
			common.SetContextKey(c, constant.ContextKeyUserGroup, user.Group)
			common.SetContextKey(c, constant.ContextKeyUsingGroup, tt.usingGroup)
			common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-4o-mini")
			upstreamError := types.NewErrorWithStatusCode(errors.New("upstream unavailable"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)

			for i, group := range tt.selectedGroups {
				common.SetContextKey(c, constant.ContextKeyAutoGroup, group)
				common.SetContextKey(c, constant.ContextKeyChannelId, 167+i)
				processChannelError(c, types.ChannelError{ChannelId: 167 + i}, upstreamError, nil)
			}

			var logs []model.Log
			require.NoError(t, db.Where("user_id = ? AND type = ?", user.Id, model.LogTypeError).Order("id").Find(&logs).Error)
			require.Len(t, logs, len(tt.wantGroups))
			for i, log := range logs {
				assert.Equal(t, tt.wantGroups[i], log.Group)
				assert.Equal(t, 167+i, log.ChannelId)
			}
		})
	}
}
