package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
				processChannelError(c, types.ChannelError{ChannelId: 167 + i}, upstreamError)
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
