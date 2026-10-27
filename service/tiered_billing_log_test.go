package service

import (
	"encoding/base64"
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectTieredBillingMultiplierMatches(t *testing.T) {
	info := &relaycommon.RelayInfo{TieredBillingSnapshot: &billingexpr.BillingSnapshot{ExprString: `tier("base", p)`}}
	for _, test := range []struct {
		name   string
		result *billingexpr.TieredResult
	}{
		{"matched", &billingexpr.TieredResult{MatchedTier: "base", MatchedMultipliers: []int{0, 2}}},
		{"no matches", &billingexpr.TieredResult{MatchedTier: "base", MatchedMultipliers: []int{}}},
		{"failed settlement", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			other := map[string]interface{}{}
			InjectTieredBillingInfo(other, info, test.result)
			expression, err := base64.StdEncoding.DecodeString(other["expr_b64"].(string))
			require.NoError(t, err)
			assert.Equal(t, info.TieredBillingSnapshot.ExprString, string(expression))
			if test.result == nil {
				assert.NotContains(t, other, "matched_multipliers")
				return
			}
			assert.Equal(t, test.result.MatchedTier, other["matched_tier"])
			assert.Equal(t, test.result.MatchedMultipliers, other["matched_multipliers"])
		})
	}
}
