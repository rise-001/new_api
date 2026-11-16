package controller

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTokenAlwaysEnablesAutoGroupRetry(t *testing.T) {
	for _, tc := range []struct {
		name  string
		retry any
	}{
		{name: "omitted"},
		{name: "explicitly disabled", retry: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTokenControllerTestDB(t)
			body := gin.H{"name": "auto-key", "group": "auto", "unlimited_quota": true, "expired_time": -1}
			if tc.retry != nil {
				body["cross_group_retry"] = tc.retry
			}
			ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/", body, 1)

			AddToken(ctx)

			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)
			var token model.Token
			require.NoError(t, db.Where("user_id = ?", 1).First(&token).Error)
			assert.True(t, token.CrossGroupRetry)
		})
	}
}

func TestUpdateTokenCannotDisableAutoGroupRetry(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "auto-key", "auto-retry-test-key")
	body := gin.H{
		"id": token.Id, "name": token.Name, "group": "auto",
		"unlimited_quota": true, "expired_time": -1, "cross_group_retry": false,
	}
	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/token/", body, 1)

	UpdateToken(ctx)

	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var stored model.Token
	require.NoError(t, db.First(&stored, token.Id).Error)
	assert.True(t, stored.CrossGroupRetry)
	var detail model.Token
	require.NoError(t, common.Unmarshal(response.Data, &detail))
	assert.True(t, detail.CrossGroupRetry)
}

func TestGetLegacyAutoTokenReportsRetryEnabled(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "legacy-auto-key", "legacy-auto-retry-test-key")
	require.NoError(t, db.Model(token).Update("group", "auto").Error)
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/token/"+strconv.Itoa(token.Id), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}

	GetToken(ctx)

	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var detail model.Token
	require.NoError(t, common.Unmarshal(response.Data, &detail))
	assert.True(t, detail.CrossGroupRetry)
	assert.Equal(t, token.GetMaskedKey(), detail.Key)
}
