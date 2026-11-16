package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupContextForTokenAlwaysEnablesAutoGroupRetry(t *testing.T) {
	for _, tc := range []struct {
		name    string
		group   string
		stored  bool
		enabled bool
	}{
		{name: "legacy auto token with retry disabled", group: "auto", stored: false, enabled: true},
		{name: "auto token with retry enabled", group: "auto", stored: true, enabled: true},
		{name: "fixed group stays within its group", group: "default", stored: true, enabled: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			token := &model.Token{Group: tc.group, CrossGroupRetry: tc.stored}

			require.NoError(t, SetupContextForToken(ctx, token))
			assert.Equal(t, tc.enabled, common.GetContextKeyBool(ctx, constant.ContextKeyTokenCrossGroupRetry))
			assert.Equal(t, tc.group, common.GetContextKeyString(ctx, constant.ContextKeyTokenGroup))
		})
	}
}
