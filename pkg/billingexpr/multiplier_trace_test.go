package billingexpr_test

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionalMultiplierTrace(t *testing.T) {
	for _, test := range []struct {
		name    string
		expr    string
		request billingexpr.RequestInput
		cost    float64
		matched []int
	}{
		{
			name: "multiple matches with an unmatched rule between them",
			expr: `(tier("峰谷价", p * 2 + c * 8)) * (hour("UTC") >= 0 ? 0.5 : 1) * (hour("UTC") < 0 ? 0.2 : 1) * (hour("UTC") < 24 ? 0.5 : 1)`,
			cost: 7, matched: []int{0, 2},
		},
		{
			name:    "request conditions use settlement input",
			expr:    `(tier("峰谷价", p * 2 + c * 8)) * (header("x-plan") == "fast" ? 2 : 1) * (param("discount") == true ? 0.5 : 1)`,
			request: billingexpr.RequestInput{Headers: map[string]string{"X-Plan": "fast"}, Body: []byte(`{"discount":true}`)},
			cost:    28, matched: []int{0, 1},
		},
		{
			name: "zero multiplier is still matched",
			expr: `(tier("峰谷价", p * 2 + c * 8)) * (hour("UTC") >= 0 ? 0 : 1)`,
			cost: 0, matched: []int{0},
		},
		{
			name: "unit multiplier is still matched",
			expr: `(tier("峰谷价", p * 2 + c * 8)) * (hour("UTC") >= 0 ? 1 : 1)`,
			cost: 28, matched: []int{0},
		},
		{
			name: "no conditions match",
			expr: `(tier("峰谷价", p * 2 + c * 8)) * (hour("UTC") < 0 ? 0.5 : 1)`,
			cost: 28, matched: []int{},
		},
		{
			name:    "parentheses and multiplication inside a quoted value",
			expr:    `v1:(tier("峰谷价", p * 2 + c * 8)) * (header("x-label") == "价) * (格" ? 0.5 : 1)`,
			request: billingexpr.RequestInput{Headers: map[string]string{"x-label": "价) * (格"}},
			cost:    14, matched: []int{0},
		},
		{
			name: "tier conditions are not multipliers",
			expr: `(p < 100 ? tier("峰谷价", p * 2 + c * 8) : tier("其他", p * 3)) * (hour("UTC") >= 0 ? 0.5 : 1)`,
			cost: 14, matched: []int{0},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cost, trace, err := billingexpr.RunExprWithRequest(test.expr, billingexpr.TokenParams{P: 10, C: 1}, test.request)
			require.NoError(t, err)
			assert.Equal(t, test.cost, cost)
			assert.Equal(t, "峰谷价", trace.MatchedTier)
			assert.Equal(t, test.matched, trace.MatchedMultipliers)
		})
	}
}

func TestConditionalMultiplierTraceIsPerSettlement(t *testing.T) {
	expression := `(tier("base", p * 2)) * (header("x-plan") == "fast" ? 0.5 : 1)`
	snapshot := &billingexpr.BillingSnapshot{
		ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
		QuotaPerUnit: 1_000_000, GroupRatio: 2, EstimatedTier: "base",
	}
	for _, test := range []struct {
		header  string
		quota   int
		matched []int
	}{
		{"fast", 20, []int{0}},
		{"standard", 40, []int{}},
	} {
		t.Run(test.header, func(t *testing.T) {
			result, err := billingexpr.ComputeTieredQuotaWithRequest(snapshot, billingexpr.TokenParams{P: 10}, billingexpr.RequestInput{
				Headers: map[string]string{"x-plan": test.header},
			})
			require.NoError(t, err)
			assert.Equal(t, test.quota, result.ActualQuotaAfterGroup)
			assert.Equal(t, test.matched, result.MatchedMultipliers)
		})
	}
}
